# Thmanyah Assessment — Write-up

## 1. How I read the problem

The brief is a media catalog with two very different workloads, so I treated them as two
sides of one system rather than one CRUD app:

- **CMS** — internal, write-heavy, low traffic, correctness- and permission-sensitive.
  Editors create/edit/publish shows & episodes and import from external sources.
- **Discovery** — public, read-only, **the part that must absorb 10M users/hour**, with
  search over published content.

Most of my decisions follow from that split: optimise the two paths independently, and keep
the write side simple while making the read side cheap to scale horizontally.

## 2. Architecture at a glance

A **modular monolith** shipped as **two binaries** (`cmd/cms`, `cmd/discovery`) over a shared
`internal/`. One codebase, but each side deploys and scales on its own.

```
Editors → CMS  ─┐                         ┌─→ Postgres (source of truth + FTS)
                ├── same data, two paths ──┤
Public  → Discovery (cached) ─┘           └─→ Redis (read-path cache)
```

- **Package-by-feature + ports & adapters.** Each package owns a capability and exposes an
  interface; infrastructure is an adapter behind it (`store`: Postgres + in-memory; `cache`:
  Redis + memory + noop; `ingestion`: RSS/CSV/YouTube behind one `SourceImporter`). Siblings
  don’t import each other — they meet at interfaces. This is where **SOLID / low coupling /
  clear boundaries** actually live: handlers depend on ports, not on Bun or Redis.
- **`main` is only an orchestrator** — it loads config and calls package constructors
  (`store.New`, `cache.New`, `ingestion.Importers`); it contains no build logic.
- The domain (`catalog`) is pure Go with JSON tags only — no ORM tags, no infra imports.

## 3. Key decisions

**Read/write split (CQRS-lite).** Two services over one database. *Why:* the read path can be
replicated, cached, and rate-limited without touching the write path, and a Discovery outage
never blocks editors. *Alternative:* a single API — simpler, but couples scaling and blast
radius. *Trade-off:* a little duplication (two routers, two mains) for independent scaling.

**Postgres as the single source of truth.** *Why:* relational integrity for shows/episodes,
transactions for imports, and it doubles as the search engine (below). *Alternative:* a
document store — rejected, the data is relational and I didn’t want eventual-consistency
headaches on the write side.

**Search: Postgres FTS, not a dedicated engine.** A generated `tsvector` column +
GIN index, queried with `websearch_to_tsquery`. *Why:* one fewer system to run, index stays
in sync transactionally, and it comfortably covers catalog-sized search. *Alternative:*
Elasticsearch/Meilisearch — more powerful (relevance, typo-tolerance) but operational
overhead and a sync pipeline I didn’t think the scope justified. **Honest limit:** Postgres
has no Arabic stemmer, so I use the `simple` config (normalisation, no stemming) — documented
as a known limitation, and the `Searcher` port means swapping in a real engine later is a
contained change.

**Cache-aside + singleflight (Discovery).** Reads go through `Remember[T]`: check Redis →
on miss hit Postgres → backfill, with **singleflight** collapsing concurrent misses into one
DB query (stampede protection). *Why:* this is the core 10M/hr lever. *Trade-off:* TTL-based
invalidation (60s) means brief staleness; acceptable for a public catalog, and explicit
invalidation-on-publish is a documented next step.

**Pagination: offset for CMS, keyset for Discovery.** CMS uses offset + total count (editors
want “page 7 of 20”). Discovery uses **keyset/cursor** (`(published_at, id)`), which stays
O(1) regardless of depth. *Why:* deep `OFFSET` degrades exactly where public traffic is
highest; keyset doesn’t.

**Auth: JWT + a code-defined RBAC matrix.** Roles (`admin/editor/viewer`) live as a column on
`cms_users`; the role→permission matrix is **code constants**, not tables. *Why:* permissions
are behaviour, not data — keeping them in code makes them testable and removes a join on every
request. *Alternative:* `roles`/`permissions` tables (from my initial ERD) — more flexible at
runtime, but over-engineered for three fixed roles.

**Ingestion: one seam, idempotent upserts.** Every source implements `SourceImporter`; a
registry maps `source → importer` (config-driven on/off). Idempotency comes from
`external_refs UNIQUE(source, owner_type, external_id)` — re-running an import updates instead
of duplicating. *Why:* adding a source is a new adapter + one line (open/closed), and imports
are safe to retry.

**ORM confined to the adapter.** Bun lives only in the Postgres adapter via `row` structs with
`bun:` tags and a `toDomain()` mapping. *Why:* the domain stays persistence-ignorant, and the
same **contract test suite runs against both the in-memory and Postgres** adapters — proving
they’re interchangeable.

**Actor-model in-memory store.** The memory adapter is a goroutine that owns its map and takes
commands over a channel (no mutexes), sharded by UUID. *Why:* it makes the fast test backend
genuinely concurrency-safe and mirrors a real ownership model rather than lock-juggling.

## 4. Scaling to 10M users/hour

I split this into **what the app does** and **what the infrastructure does**, and built the
former so it slots into the latter:

| Built in the app | Described as infra (see `docs/scaling.md`) |
|---|---|
| Stateless services (scale horizontally) | CDN in front of Discovery |
| Read/write split (scale reads alone) | Postgres read replicas |
| Redis cache-aside + singleflight | PgBouncer connection pooling |
| FTS + GIN, status/published indexes | Autoscaling / partitioning |
| Keyset pagination, gzip, per-IP rate limit | Async outbox indexing |

The request funnel is **CDN → cache → replicas → primary**, so the primary sees a tiny
fraction of the 10M. I didn’t stand up that infra in a take-home, but the application is
stateless, cache-first, and index-backed precisely so it can.

## 5. Difficulties I hit (and fixes)

- **Arabic full-text search.** Postgres ships no Arabic dictionary; stemming isn’t available.
  I chose the `simple` config (matches on normalised tokens) and documented it rather than
  fake it. The `Searcher` port keeps a future swap cheap.
- **A real bug the RSS test caught.** Importing the فنجان feed created the show as `draft`
  (the default), and Discovery only serves published content — so the show “vanished.” The
  fix was to publish imported shows. Good reminder that the read/write status boundary is
  real, and that testing with real data beats synthetic fixtures.
- **Go interface collision.** `ShowRepository` and `EpisodeRepository` both need `Create`, so
  one struct can’t satisfy both — I split them into separate repositories. A small Go-ism, but
  it shaped the store’s layout.
- **Idempotent imports.** Designing `external_refs` as the dedupe key (not slugs, which can
  collide) is what makes re-imports safe.
- **Keeping the domain clean.** Resisting the urge to put `bun:` tags on domain structs; the
  `row`/`toDomain` split costs a few lines but keeps the dependency arrow pointing inward.

## 6. Testing

- **Contract tests** run one suite against **both** store adapters (memory + Postgres) — the
  strongest guarantee that the abstraction holds.
- **Integration tests** use testcontainers for real Postgres/Redis (behind a build tag).
- `-race` on the concurrent in-memory store; full **end-to-end verification on Docker**
  (auth → import 364 real episodes → search → cache hit → metrics scraped).

## 7. What I’d do next

Async import jobs (`import_jobs` + worker + `202`/status endpoint) so large feeds don’t block
the request; explicit cache invalidation on publish; OTLP tracing alongside the Prometheus
metrics; denormalising people/topics into `search_text`; and read replicas + PgBouncer when
real traffic warrants. Each is a contained change because the boundaries are already in place —
which was the whole point of the design.
