-- Wrapper marked IMMUTABLE so the result can drive a GENERATED column.
-- Mirrors decisions_search_vector: title weighted highest, then the
-- human-facing description, then the code content itself.
CREATE OR REPLACE FUNCTION snippets_search_vector(
  title       TEXT,
  description TEXT,
  content     TEXT
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(title, '')),       'A') ||
    setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(content, '')),     'C')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE TABLE snippets (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title         TEXT NOT NULL,
  project       TEXT REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  language      TEXT NOT NULL DEFAULT '',
  content       TEXT NOT NULL,
  tags          TEXT[] NOT NULL DEFAULT '{}',
  description   TEXT NOT NULL DEFAULT '',
  search_vector TSVECTOR GENERATED ALWAYS AS (
    snippets_search_vector(title, description, content)
  ) STORED,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX snippets_search_vector_idx ON snippets USING GIN (search_vector);
CREATE INDEX snippets_project_idx       ON snippets (project);
CREATE INDEX snippets_language_idx      ON snippets (language);
CREATE INDEX snippets_tags_idx          ON snippets USING GIN (tags);

CREATE TRIGGER snippets_set_updated_at
  BEFORE UPDATE ON snippets
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
