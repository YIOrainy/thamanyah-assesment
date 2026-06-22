# Thmanyah Assesment

A media catalog backend for thamanyah which splitted be 2 cmds:

- **CMS** (`cmd/cms`) — internal, authenticated. Editors manage shows/episodes and run multi-source imports.
- **Discovery** (`cmd/discovery`) — public, read-only. Full-text search and browse over **published** content.

Both share one Postgres (source of truth). Discovery is cache-backed and built for read scale.

## Quick start

**1. Create your config** from the template and change the values:

```bash
cp config.example.yaml config.yaml
```

**2. Run everything:**

```bash
docker compose up -d --build
# CMS :8080  ·  Discovery :8081  ·  Grafana :3000  ·  Prometheus :9090  ·  Pyroscope :4040
```

Migrations run automatically; an admin user is bootstrapped from config. Override host ports if they clash:

Try it:

```bash
TOKEN=$(curl -s -XPOST localhost:8080/api/v1/auth/login \
  -d '{"email":"admin@thmanyah.local","password":"admin12345"}' | jq -r .access_token)

# import a real podcast feed
curl -XPOST localhost:8080/api/v1/imports -H "Authorization: Bearer $TOKEN" \
  -d '{"source":"rss","query":"https://feeds.npr.org/510289/podcast.xml"}'

# search
curl 'localhost:8081/api/v1/search?q=finjan'
```

### Import Thmanyah's فنجان feed

The `$TOKEN` from the login above already has `imports:run`, so just call the import API with the real RSS feed:

```bash
curl -XPOST localhost:8080/api/v1/imports -H "Authorization: Bearer $TOKEN" \
  -d '{"source":"rss","query":"https://files.hosting.thmanyah.com/podcasts/89/1713955813943-768/rss-feed.rss"}'
# → {"shows_created":1,"episodes_created":364,...}   (re-run is idempotent → *_updated)

# then read it back via Discovery (public)
curl 'localhost:8081/api/v1/search?q=فنجان'
```

## API docs

Each service serves its OpenAPI spec and live Swagger UI:

- `GET /openapi.yaml` — the spec (`api/openapi.yaml`)
- `GET /docs` — Swagger UI

## Database schema

```mermaid
erDiagram
    SHOWS {
        uuid        id PK
        text        title
        text        slug UK
        text        format "podcast | documentary | sports"
        text        language
        text        status "draft | published | archived"
        uuid        created_by
        timestamptz created_at
    }
    EPISODES {
        uuid        id PK
        uuid        show_id FK
        text        title
        text        slug
        int         episode_number
        text        content_type "audio | video"
        text        status
        timestamptz published_at
        tsvector    search_tsv "generated + GIN index (FTS)"
    }
    CMS_USERS {
        uuid        id PK
        text        email UK
        text        password_hash
        text        role "admin | editor | viewer"
    }
    EXTERNAL_REFS {
        uuid        id PK
        text        source "rss | csv | youtube"
        text        owner_type "show | episode"
        uuid        owner_id
        text        external_id
        timestamptz imported_at
    }

    SHOWS    ||--o{ EPISODES      : has
    SHOWS    ||--o{ EXTERNAL_REFS : "provenance (owner_type=show)"
    EPISODES ||--o{ EXTERNAL_REFS : "provenance (owner_type=episode)"
```

Key constraints (in migrations, not visible above):
- `episodes`: `UNIQUE(show_id, slug)` and `UNIQUE(show_id, episode_number)`.
- `external_refs`: `UNIQUE(source, owner_type, external_id)` — the **idempotency key** for imports. The `owner` is polymorphic, so the links to `shows`/`episodes` are logical (no DB foreign key).
- Search is Postgres FTS via the generated `episodes.search_tsv` (GIN), Arabic-normalized.

### Roles & permissions (RBAC)

Roles aren't separate tables — `cms_users.role` holds the role, and the role→permission
matrix is **code-defined** (`internal/auth/permissions.go`). The JWT carries the role (and
optional narrowing scopes); each CMS route requires a permission.

| permission | admin | editor | viewer |
|---|:---:|:---:|:---:|
| `users:manage`     | ✓ |   |   |
| `shows:read`       | ✓ | ✓ | ✓ |
| `shows:write`      | ✓ | ✓ |   |
| `shows:publish`    | ✓ | ✓ |   |
| `episodes:read`    | ✓ | ✓ | ✓ |
| `episodes:write`   | ✓ | ✓ |   |
| `episodes:publish` | ✓ | ✓ |   |
| `imports:run`      | ✓ | ✓ |   |

Discovery is public (no auth).
