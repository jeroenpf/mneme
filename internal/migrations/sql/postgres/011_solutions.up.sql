-- Wrapper marked IMMUTABLE so the result can drive a GENERATED column.
-- The error text is weighted highest (you search by the symptom you hit),
-- then the fix. Mirrors decisions_search_vector / snippets_search_vector.
CREATE OR REPLACE FUNCTION solutions_search_vector(
  error_description TEXT,
  solution          TEXT
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(error_description, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(solution, '')),          'B')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE TABLE solutions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project           TEXT REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  error_description TEXT NOT NULL,
  solution          TEXT NOT NULL,
  tags              TEXT[] NOT NULL DEFAULT '{}',
  source_url        TEXT NOT NULL DEFAULT '',
  search_vector     TSVECTOR GENERATED ALWAYS AS (
    solutions_search_vector(error_description, solution)
  ) STORED,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX solutions_search_vector_idx ON solutions USING GIN (search_vector);
CREATE INDEX solutions_project_idx       ON solutions (project);
CREATE INDEX solutions_tags_idx          ON solutions USING GIN (tags);

CREATE TRIGGER solutions_set_updated_at
  BEFORE UPDATE ON solutions
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
