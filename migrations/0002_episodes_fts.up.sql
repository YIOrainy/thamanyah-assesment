ALTER TABLE episodes
    ADD COLUMN search_tsv tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;

CREATE INDEX ix_episodes_search_tsv ON episodes USING GIN (search_tsv);
