-- Wrapper marked IMMUTABLE so the result can drive a GENERATED column.
-- Safe because we pin the regconfig to 'english' literally; built-in
-- to_tsvector(regconfig, text) is itself immutable when the config is fixed.
CREATE OR REPLACE FUNCTION documents_search_vector(
  title  TEXT,
  ticket TEXT,
  tags   TEXT[],
  body   JSONB
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(ticket, '')), 'B') ||
    setweight(to_tsvector('english', array_to_string(coalesce(tags, '{}'), ' ')), 'B') ||
    setweight(to_tsvector('english', coalesce(body::text, '')), 'C')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE TABLE documents (
  id            TEXT PRIMARY KEY,
  title         TEXT NOT NULL,
  project       TEXT,
  category      TEXT,
  type          TEXT NOT NULL
                CHECK (type IN ('plan','report','spec','adr','brainstorm','journal')),
  status        TEXT NOT NULL DEFAULT 'todo'
                CHECK (status IN ('todo','in-progress','complete','blocked','archived')),
  ticket        TEXT,
  repo          TEXT,
  tags          TEXT[] NOT NULL DEFAULT '{}',
  phase_current INT,
  phase_total   INT,
  meta          JSONB NOT NULL DEFAULT '{}'::jsonb,
  body          JSONB NOT NULL DEFAULT '{}'::jsonb,
  search_vector TSVECTOR GENERATED ALWAYS AS (
    documents_search_vector(title, ticket, tags, body)
  ) STORED,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX documents_search_vector_idx ON documents USING GIN (search_vector);
CREATE INDEX documents_tags_idx          ON documents USING GIN (tags);
CREATE INDEX documents_project_idx       ON documents (project);
CREATE INDEX documents_type_status_idx   ON documents (type, status);
