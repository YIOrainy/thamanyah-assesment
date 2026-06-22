CREATE TABLE cms_users (
    id            uuid PRIMARY KEY,
    name          text        NOT NULL,
    email         text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    role          text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_cms_users_role CHECK (role IN ('admin', 'editor', 'viewer'))
);
