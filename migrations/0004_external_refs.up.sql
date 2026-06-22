CREATE TABLE external_refs (
    id           uuid PRIMARY KEY,
    source       text        NOT NULL,
    owner_type   text        NOT NULL,
    owner_id     uuid        NOT NULL,
    external_id  text        NOT NULL,
    external_url text        NOT NULL DEFAULT '',
    imported_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_external_refs       UNIQUE (source, owner_type, external_id),
    CONSTRAINT ck_external_refs_owner CHECK (owner_type IN ('show', 'episode'))
);

CREATE INDEX ix_external_refs_owner ON external_refs (owner_type, owner_id);
