-- Wrapper marked IMMUTABLE so the result can drive a GENERATED column.
-- Mirrors documents_search_vector: title weighted highest, then the "why"
-- (rationale/decision), then the supporting prose.
CREATE OR REPLACE FUNCTION decisions_search_vector(
  title        TEXT,
  decision     TEXT,
  rationale    TEXT,
  alternatives TEXT,
  consequences TEXT
) RETURNS tsvector AS $$
  SELECT
    setweight(to_tsvector('english', coalesce(title, '')),        'A') ||
    setweight(to_tsvector('english', coalesce(rationale, '')),    'B') ||
    setweight(to_tsvector('english', coalesce(decision, '')),     'B') ||
    setweight(to_tsvector('english', coalesce(alternatives, '')), 'C') ||
    setweight(to_tsvector('english', coalesce(consequences, '')), 'C')
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE TABLE decisions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title         TEXT NOT NULL,
  project       TEXT REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  decision      TEXT NOT NULL,
  rationale     TEXT NOT NULL DEFAULT '',
  alternatives  TEXT NOT NULL DEFAULT '',
  consequences  TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL DEFAULT 'accepted'
                CHECK (status IN ('proposed','accepted','deprecated')),
  search_vector TSVECTOR GENERATED ALWAYS AS (
    decisions_search_vector(title, decision, rationale, alternatives, consequences)
  ) STORED,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX decisions_search_vector_idx ON decisions USING GIN (search_vector);
CREATE INDEX decisions_project_idx       ON decisions (project);
CREATE INDEX decisions_status_idx        ON decisions (status);

CREATE TRIGGER decisions_set_updated_at
  BEFORE UPDATE ON decisions
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
