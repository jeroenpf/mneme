-- SQLite core tables (plan-sqlite-backend p2-t2). Mirrors the Postgres schema
-- with SQLite-native types: TEXT PKs supplied app-side (no gen_random_uuid),
-- JSON stored as TEXT (meta/body, and the tags/accomplished/deferred arrays),
-- millisecond-precision timestamps via strftime, and public_id minted in Go
-- (no plpgsql gen_public_id). Full-text search is a separate concern: one FTS5
-- virtual table per searchable type is created in a later migration, not a
-- generated search_vector column.

CREATE TABLE projects (
  id          TEXT PRIMARY KEY,
  public_id   TEXT NOT NULL,
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX projects_public_id_idx ON projects (public_id);

CREATE TABLE documents (
  id            TEXT PRIMARY KEY,
  public_id     TEXT NOT NULL,
  title         TEXT NOT NULL,
  project       TEXT REFERENCES projects (slug)
                  ON UPDATE CASCADE ON DELETE SET NULL
                  DEFERRABLE INITIALLY DEFERRED,
  category      TEXT,
  type          TEXT NOT NULL
                  CHECK (type IN ('plan', 'report', 'spec', 'adr', 'brainstorm', 'journal')),
  status        TEXT NOT NULL DEFAULT 'todo'
                  CHECK (status IN ('todo', 'in-progress', 'complete', 'blocked', 'archived')),
  ticket        TEXT,
  repo          TEXT,
  tags          TEXT NOT NULL DEFAULT '[]',   -- JSON array
  phase_current INTEGER,
  phase_total   INTEGER,
  meta          TEXT NOT NULL DEFAULT '{}',   -- JSON object
  body          TEXT NOT NULL DEFAULT '{}',   -- JSON object
  created_at    TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  updated_at    TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX documents_public_id_idx ON documents (public_id);
CREATE INDEX documents_project_idx     ON documents (project);
CREATE INDEX documents_type_status_idx ON documents (type, status);

CREATE TABLE decisions (
  id           TEXT PRIMARY KEY,
  public_id    TEXT NOT NULL,
  title        TEXT NOT NULL,
  project      TEXT REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  decision     TEXT NOT NULL,
  rationale    TEXT NOT NULL DEFAULT '',
  alternatives TEXT NOT NULL DEFAULT '',
  consequences TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'accepted'
                 CHECK (status IN ('proposed', 'accepted', 'deprecated')),
  created_at   TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  updated_at   TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX decisions_public_id_idx ON decisions (public_id);
CREATE INDEX decisions_project_idx ON decisions (project);
CREATE INDEX decisions_status_idx  ON decisions (status);

CREATE TABLE snippets (
  id          TEXT PRIMARY KEY,
  public_id   TEXT NOT NULL,
  title       TEXT NOT NULL,
  project     TEXT REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  language    TEXT NOT NULL DEFAULT '',
  content     TEXT NOT NULL,
  tags        TEXT NOT NULL DEFAULT '[]',   -- JSON array
  description TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX snippets_public_id_idx ON snippets (public_id);
CREATE INDEX snippets_project_idx  ON snippets (project);
CREATE INDEX snippets_language_idx ON snippets (language);

CREATE TABLE journal_entries (
  id           TEXT PRIMARY KEY,
  public_id    TEXT NOT NULL,
  project      TEXT REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  session_ref  TEXT NOT NULL DEFAULT '',
  summary      TEXT NOT NULL,
  accomplished TEXT NOT NULL DEFAULT '[]',   -- JSON array
  deferred     TEXT NOT NULL DEFAULT '[]',   -- JSON array
  created_at   TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  updated_at   TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX journal_entries_public_id_idx ON journal_entries (public_id);
CREATE INDEX journal_entries_project_idx    ON journal_entries (project);
CREATE INDEX journal_entries_created_at_idx ON journal_entries (created_at DESC);

CREATE TABLE solutions (
  id                TEXT PRIMARY KEY,
  public_id         TEXT NOT NULL,
  project           TEXT REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  error_description TEXT NOT NULL,
  solution          TEXT NOT NULL,
  tags              TEXT NOT NULL DEFAULT '[]',   -- JSON array
  source_url        TEXT NOT NULL DEFAULT '',
  created_at        TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  updated_at        TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);
CREATE UNIQUE INDEX solutions_public_id_idx ON solutions (public_id);
CREATE INDEX solutions_project_idx ON solutions (project);

CREATE TABLE memories (
  id         TEXT PRIMARY KEY,
  scope      TEXT NOT NULL CHECK (scope IN ('global', 'project', 'area')),
  project    TEXT REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  area       TEXT,
  key        TEXT NOT NULL,
  value      TEXT NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  CONSTRAINT memories_scope_shape CHECK (
         (scope = 'global'  AND project IS NULL     AND area IS NULL)
      OR (scope = 'project' AND project IS NOT NULL AND area IS NULL)
      OR (scope = 'area'    AND project IS NOT NULL AND area IS NOT NULL)
  )
);
-- Postgres uses UNIQUE NULLS NOT DISTINCT; SQLite treats NULLs as distinct in a
-- UNIQUE index, so COALESCE the nullable scope columns to '' to reproduce the
-- "one row per (scope, project, area, key) identity" upsert key.
CREATE UNIQUE INDEX memories_identity
  ON memories (scope, COALESCE(project, ''), COALESCE(area, ''), key);

CREATE TABLE env_entries (
  id          TEXT PRIMARY KEY,
  project     TEXT NOT NULL REFERENCES projects (slug) ON UPDATE CASCADE ON DELETE CASCADE,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  description TEXT,
  updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  CONSTRAINT env_entries_identity UNIQUE (project, key)
);
