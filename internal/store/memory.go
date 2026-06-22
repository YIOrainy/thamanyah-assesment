package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

// actor serializes access to shared state by running every operation as a
// closure on a single goroutine — no mutex ("share memory by communicating").
type actor struct {
	cmds chan func()
	done chan struct{}
}

func newActor() *actor {
	a := &actor{cmds: make(chan func()), done: make(chan struct{})}
	go a.run()
	return a
}

func (a *actor) run() {
	for {
		select {
		case cmd := <-a.cmds:
			cmd()
		case <-a.done:
			return
		}
	}
}

func (a *actor) Close() { close(a.done) }

// submit runs fn on the actor's goroutine and returns its result, respecting ctx.
func submit[T any](ctx context.Context, a *actor, fn func() (T, error)) (T, error) {
	var zero T
	type result struct {
		val T
		err error
	}
	reply := make(chan result, 1)
	select {
	case a.cmds <- func() { v, err := fn(); reply <- result{v, err} }:
	case <-ctx.Done():
		return zero, ctx.Err()
	}
	select {
	case res := <-reply:
		return res.val, res.err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// shardIndex maps an id to one of n shards via FNV-1a (samd-style hash sharding).
func shardIndex(id uuid.UUID, n int) int {
	if n <= 1 {
		return 0
	}
	var h uint32 = 2166136261
	for _, b := range id[:] {
		h ^= uint32(b)
		h *= 16777619
	}
	return int(h % uint32(n))
}

type showShard struct {
	*actor
	items map[uuid.UUID]catalog.Show
}

// MemoryShowRepository shards shows across N actors. Operations keyed by ID hit
// one shard; slug/list/uniqueness fan out to all shards.
type MemoryShowRepository struct {
	shards []*showShard
}

func NewMemoryShowRepository(actors int) *MemoryShowRepository {
	if actors < 1 {
		actors = 1
	}
	r := &MemoryShowRepository{shards: make([]*showShard, actors)}
	for i := range r.shards {
		r.shards[i] = &showShard{actor: newActor(), items: make(map[uuid.UUID]catalog.Show)}
	}
	return r
}

func (r *MemoryShowRepository) Close() {
	for _, s := range r.shards {
		s.Close()
	}
}

// Actors reports how many shard goroutines back this repository.
func (r *MemoryShowRepository) Actors() int { return len(r.shards) }

var _ ShowRepository = (*MemoryShowRepository)(nil)

func (r *MemoryShowRepository) shardFor(id uuid.UUID) *showShard {
	return r.shards[shardIndex(id, len(r.shards))]
}

func (r *MemoryShowRepository) Create(ctx context.Context, show *catalog.Show) error {
	// global slug uniqueness — fan out before inserting into the target shard.
	if _, err := r.GetBySlug(ctx, show.Slug); err == nil {
		return ErrConflict
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	sh := r.shardFor(show.ID)
	_, err := submit(ctx, sh.actor, func() (struct{}, error) {
		if _, ok := sh.items[show.ID]; ok {
			return struct{}{}, ErrConflict
		}
		sh.items[show.ID] = *show
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryShowRepository) Update(ctx context.Context, show *catalog.Show) error {
	sh := r.shardFor(show.ID)
	_, err := submit(ctx, sh.actor, func() (struct{}, error) {
		if _, ok := sh.items[show.ID]; !ok {
			return struct{}{}, ErrNotFound
		}
		sh.items[show.ID] = *show
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryShowRepository) GetByID(ctx context.Context, id uuid.UUID) (*catalog.Show, error) {
	sh := r.shardFor(id)
	return submit(ctx, sh.actor, func() (*catalog.Show, error) {
		s, ok := sh.items[id]
		if !ok {
			return nil, ErrNotFound
		}
		return &s, nil
	})
}

func (r *MemoryShowRepository) GetBySlug(ctx context.Context, slug string) (*catalog.Show, error) {
	for _, sh := range r.shards {
		got, err := submit(ctx, sh.actor, func() (*catalog.Show, error) {
			for _, s := range sh.items {
				if s.Slug == slug {
					cp := s
					return &cp, nil
				}
			}
			return nil, ErrNotFound
		})
		if err == nil {
			return got, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}

func showMatches(s catalog.Show, f ShowFilter) bool {
	if f.Format != "" && s.Format != f.Format {
		return false
	}
	if f.Status != "" && s.Status != f.Status {
		return false
	}
	if f.Language != "" && s.Language != f.Language {
		return false
	}
	return true
}

func (r *MemoryShowRepository) List(ctx context.Context, f ShowFilter) ([]*catalog.Show, error) {
	out := make([]*catalog.Show, 0)
	for _, sh := range r.shards {
		part, err := submit(ctx, sh.actor, func() ([]*catalog.Show, error) {
			var xs []*catalog.Show
			for _, s := range sh.items {
				if showMatches(s, f) {
					cp := s
					xs = append(xs, &cp)
				}
			}
			return xs, nil
		})
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID.String() > out[j].ID.String()
	})

	// keyset (Discovery) takes precedence over offset (CMS)
	if f.Cursor != "" {
		ts, id, err := DecodeCursor(f.Cursor)
		if err != nil {
			return nil, err
		}
		kept := make([]*catalog.Show, 0, len(out))
		for _, s := range out {
			if s.CreatedAt.Before(ts) || (s.CreatedAt.Equal(ts) && s.ID.String() < id.String()) {
				kept = append(kept, s)
			}
		}
		out = kept
	} else if f.Offset > 0 {
		if f.Offset >= len(out) {
			out = []*catalog.Show{}
		} else {
			out = out[f.Offset:]
		}
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (r *MemoryShowRepository) Count(ctx context.Context, f ShowFilter) (int, error) {
	total := 0
	for _, sh := range r.shards {
		n, err := submit(ctx, sh.actor, func() (int, error) {
			c := 0
			for _, s := range sh.items {
				if showMatches(s, f) {
					c++
				}
			}
			return c, nil
		})
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

type episodeShard struct {
	*actor
	items map[uuid.UUID]catalog.Episode
}

type MemoryEpisodeRepository struct {
	shards []*episodeShard
}

func NewMemoryEpisodeRepository(actors int) *MemoryEpisodeRepository {
	if actors < 1 {
		actors = 1
	}
	r := &MemoryEpisodeRepository{shards: make([]*episodeShard, actors)}
	for i := range r.shards {
		r.shards[i] = &episodeShard{actor: newActor(), items: make(map[uuid.UUID]catalog.Episode)}
	}
	return r
}

func (r *MemoryEpisodeRepository) Close() {
	for _, s := range r.shards {
		s.Close()
	}
}

// Actors reports how many shard goroutines back this repository.
func (r *MemoryEpisodeRepository) Actors() int { return len(r.shards) }

var (
	_ EpisodeRepository = (*MemoryEpisodeRepository)(nil)
	_ Searcher          = (*MemoryEpisodeRepository)(nil)
)

// SearchEpisodes is a naive substring search over published episodes (the
// in-memory approximation of the Postgres full-text adapter).
func (r *MemoryEpisodeRepository) SearchEpisodes(ctx context.Context, query string, f SearchFilter) ([]*catalog.Episode, error) {
	ql := strings.ToLower(query)
	out := make([]*catalog.Episode, 0)
	for _, sh := range r.shards {
		part, err := submit(ctx, sh.actor, func() ([]*catalog.Episode, error) {
			var xs []*catalog.Episode
			for _, e := range sh.items {
				if e.Status != catalog.StatusPublished {
					continue
				}
				if f.Language != "" && e.Language != f.Language {
					continue
				}
				if ql != "" && !strings.Contains(strings.ToLower(e.Title+" "+e.Description), ql) {
					continue
				}
				cp := e
				xs = append(xs, &cp)
			}
			return xs, nil
		})
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := episodePublishedAt(out[i]), episodePublishedAt(out[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].ID.String() > out[j].ID.String()
	})
	if f.Cursor != "" {
		ts, id, err := DecodeCursor(f.Cursor)
		if err != nil {
			return nil, err
		}
		kept := make([]*catalog.Episode, 0, len(out))
		for _, e := range out {
			pa := episodePublishedAt(e)
			if pa.Before(ts) || (pa.Equal(ts) && e.ID.String() < id.String()) {
				kept = append(kept, e)
			}
		}
		out = kept
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func episodePublishedAt(e *catalog.Episode) time.Time {
	if e.PublishedAt != nil {
		return *e.PublishedAt
	}
	return time.Time{}
}

func (r *MemoryEpisodeRepository) shardFor(id uuid.UUID) *episodeShard {
	return r.shards[shardIndex(id, len(r.shards))]
}

func (r *MemoryEpisodeRepository) Create(ctx context.Context, ep *catalog.Episode) error {
	// per-show (slug, episode_number) uniqueness — fan out before inserting.
	conflict, err := r.hasShowConflict(ctx, ep)
	if err != nil {
		return err
	}
	if conflict {
		return ErrConflict
	}
	sh := r.shardFor(ep.ID)
	_, err = submit(ctx, sh.actor, func() (struct{}, error) {
		if _, ok := sh.items[ep.ID]; ok {
			return struct{}{}, ErrConflict
		}
		sh.items[ep.ID] = *ep
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryEpisodeRepository) hasShowConflict(ctx context.Context, ep *catalog.Episode) (bool, error) {
	for _, sh := range r.shards {
		found, err := submit(ctx, sh.actor, func() (bool, error) {
			for _, e := range sh.items {
				if e.ShowID == ep.ShowID && (e.Slug == ep.Slug || e.EpisodeNumber == ep.EpisodeNumber) {
					return true, nil
				}
			}
			return false, nil
		})
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryEpisodeRepository) Update(ctx context.Context, ep *catalog.Episode) error {
	sh := r.shardFor(ep.ID)
	_, err := submit(ctx, sh.actor, func() (struct{}, error) {
		if _, ok := sh.items[ep.ID]; !ok {
			return struct{}{}, ErrNotFound
		}
		sh.items[ep.ID] = *ep
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryEpisodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*catalog.Episode, error) {
	sh := r.shardFor(id)
	return submit(ctx, sh.actor, func() (*catalog.Episode, error) {
		e, ok := sh.items[id]
		if !ok {
			return nil, ErrNotFound
		}
		return &e, nil
	})
}

func (r *MemoryEpisodeRepository) GetBySlug(ctx context.Context, slug string) (*catalog.Episode, error) {
	for _, sh := range r.shards {
		got, err := submit(ctx, sh.actor, func() (*catalog.Episode, error) {
			for _, e := range sh.items {
				if e.Slug == slug {
					cp := e
					return &cp, nil
				}
			}
			return nil, ErrNotFound
		})
		if err == nil {
			return got, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}

func episodeMatches(e catalog.Episode, f EpisodeFilter) bool {
	if f.ShowID != uuid.Nil && e.ShowID != f.ShowID {
		return false
	}
	if f.Status != "" && e.Status != f.Status {
		return false
	}
	return true
}

func (r *MemoryEpisodeRepository) List(ctx context.Context, f EpisodeFilter) ([]*catalog.Episode, error) {
	out := make([]*catalog.Episode, 0)
	for _, sh := range r.shards {
		part, err := submit(ctx, sh.actor, func() ([]*catalog.Episode, error) {
			var xs []*catalog.Episode
			for _, e := range sh.items {
				if episodeMatches(e, f) {
					cp := e
					xs = append(xs, &cp)
				}
			}
			return xs, nil
		})
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EpisodeNumber < out[j].EpisodeNumber
	})
	if f.Offset > 0 {
		if f.Offset >= len(out) {
			out = []*catalog.Episode{}
		} else {
			out = out[f.Offset:]
		}
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (r *MemoryEpisodeRepository) Count(ctx context.Context, f EpisodeFilter) (int, error) {
	total := 0
	for _, sh := range r.shards {
		n, err := submit(ctx, sh.actor, func() (int, error) {
			c := 0
			for _, e := range sh.items {
				if episodeMatches(e, f) {
					c++
				}
			}
			return c, nil
		})
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// MemoryUserRepository is an in-memory UserRepository (single actor — users are
// low-volume, so no sharding needed).
type MemoryUserRepository struct {
	*actor
	byID    map[uuid.UUID]auth.User
	byEmail map[string]uuid.UUID
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		actor:   newActor(),
		byID:    make(map[uuid.UUID]auth.User),
		byEmail: make(map[string]uuid.UUID),
	}
}

var _ UserRepository = (*MemoryUserRepository)(nil)

func (r *MemoryUserRepository) Create(ctx context.Context, u *auth.User) error {
	_, err := submit(ctx, r.actor, func() (struct{}, error) {
		if _, ok := r.byEmail[u.Email]; ok {
			return struct{}{}, ErrConflict
		}
		if _, ok := r.byID[u.ID]; ok {
			return struct{}{}, ErrConflict
		}
		r.byID[u.ID] = *u
		r.byEmail[u.Email] = u.ID
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryUserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	return submit(ctx, r.actor, func() (*auth.User, error) {
		id, ok := r.byEmail[email]
		if !ok {
			return nil, ErrNotFound
		}
		u := r.byID[id]
		return &u, nil
	})
}

func (r *MemoryUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	return submit(ctx, r.actor, func() (*auth.User, error) {
		u, ok := r.byID[id]
		if !ok {
			return nil, ErrNotFound
		}
		return &u, nil
	})
}
