package store

import (
	"context"

	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

// SearchFilter narrows a search. Zero-value fields are ignored.
type SearchFilter struct {
	Language string
	Limit    int
	Cursor   string // keyset pagination
}

// Searcher is the read-side full-text search port over published episodes.
type Searcher interface {
	SearchEpisodes(ctx context.Context, query string, f SearchFilter) ([]*catalog.Episode, error)
}
