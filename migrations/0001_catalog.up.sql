CREATE TABLE shows (
    id          uuid PRIMARY KEY,
    title       text        NOT NULL,
    slug        text        NOT NULL UNIQUE,
    description text        NOT NULL DEFAULT '',
    format      text        NOT NULL,
    language    text        NOT NULL,
    status      text        NOT NULL DEFAULT 'draft',
    created_by  uuid        NOT NULL,
    updated_by  uuid        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_shows_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT ck_shows_format CHECK (format IN ('podcast', 'documentary', 'sports'))
);

CREATE TABLE episodes (
    id               uuid PRIMARY KEY,
    show_id          uuid        NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    title            text        NOT NULL,
    slug             text        NOT NULL,
    description      text        NOT NULL DEFAULT '',
    episode_number   int         NOT NULL,
    content_type     text        NOT NULL,
    language         text        NOT NULL,
    duration_seconds int         NOT NULL DEFAULT 0,
    status           text        NOT NULL DEFAULT 'draft',
    published_at     timestamptz,
    search_text      text        NOT NULL DEFAULT '',
    created_by       uuid        NOT NULL,
    updated_by       uuid        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_episodes_show_slug   UNIQUE (show_id, slug),
    CONSTRAINT uq_episodes_show_number UNIQUE (show_id, episode_number),
    CONSTRAINT ck_episodes_status       CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT ck_episodes_content_type CHECK (content_type IN ('audio', 'video'))
);

CREATE INDEX ix_episodes_show_id          ON episodes (show_id);
CREATE INDEX ix_episodes_status_published ON episodes (status, published_at DESC);
