-- Wrapper marked IMMUTABLE so the result can drive a GENERATED column.
-- summary is what you search by, then the session ref, then the
-- accomplished/deferred bullets. Mirrors solutions_search_vector.
CREATE OR REPLACE FUNCTION journal_search_vector(
  summary      TEXT,
  session_ref  TEXT,
  accomplished TEXT[],
  deferred     TEXT[]
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(summary, '')),     'A') ||
    setweight(to_tsvector('english', coalesce(session_ref, '')), 'B') ||
    setweight(to_tsvector('english', array_to_string(coalesce(accomplished, '{}'), ' ')), 'C') ||
    setweight(to_tsvector('english', array_to_string(coalesce(deferred, '{}'), ' ')),     'C')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

ALTER TABLE journal_entries
  ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    journal_search_vector(summary, session_ref, accomplished, deferred)
  ) STORED;

CREATE INDEX journal_entries_search_vector_idx ON journal_entries USING GIN (search_vector);
