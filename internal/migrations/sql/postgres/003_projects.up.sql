CREATE TABLE projects (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE documents
  ADD CONSTRAINT documents_project_fkey
  FOREIGN KEY (project) REFERENCES projects(slug)
  ON UPDATE CASCADE ON DELETE SET NULL
  DEFERRABLE INITIALLY DEFERRED;
