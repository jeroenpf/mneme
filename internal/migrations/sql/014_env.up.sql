CREATE TABLE env_entries (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project     TEXT NOT NULL REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  description TEXT,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT env_entries_identity UNIQUE (project, key)
);
