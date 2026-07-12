CREATE TABLE memories (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scope      TEXT NOT NULL CHECK (scope IN ('global','project','area')),
  project    TEXT REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  area       TEXT,
  key        TEXT NOT NULL,
  value      TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT memories_scope_shape CHECK (
       (scope = 'global'  AND project IS NULL     AND area IS NULL)
    OR (scope = 'project' AND project IS NOT NULL AND area IS NULL)
    OR (scope = 'area'    AND project IS NOT NULL AND area IS NOT NULL)
  ),
  CONSTRAINT memories_identity UNIQUE NULLS NOT DISTINCT (scope, project, area, key)
);
