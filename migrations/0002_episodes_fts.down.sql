DROP INDEX IF EXISTS ix_episodes_search_tsv;
ALTER TABLE episodes DROP COLUMN IF EXISTS search_tsv;
