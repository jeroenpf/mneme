-- Add full-text search to memories so they join unified retrieval (road-p4-t5).
-- env is deliberately left out — env is looked up exactly by key, never
-- fuzzily searched. Mirrors journal_search_vector: the key (identifier)
-- weighted highest, then the value, then the area context. Params are prefixed
-- to avoid colliding with the non-reserved column names key/value.
CREATE OR REPLACE FUNCTION memories_search_vector(
  mem_key   TEXT,
  mem_value TEXT,
  mem_area  TEXT
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(mem_key, '')),   'A') ||
    setweight(to_tsvector('english', coalesce(mem_value, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(mem_area, '')),  'C')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

ALTER TABLE memories
  ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    memories_search_vector(key, value, area)
  ) STORED;

CREATE INDEX memories_search_vector_idx ON memories USING GIN (search_vector);
