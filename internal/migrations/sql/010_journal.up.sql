CREATE TABLE journal_entries (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project       TEXT REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  session_ref   TEXT NOT NULL DEFAULT '',
  summary       TEXT NOT NULL,
  accomplished  TEXT[] NOT NULL DEFAULT '{}',
  deferred      TEXT[] NOT NULL DEFAULT '{}',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX journal_entries_project_idx    ON journal_entries (project);
CREATE INDEX journal_entries_created_at_idx ON journal_entries (created_at DESC);

CREATE TRIGGER journal_entries_set_updated_at
  BEFORE UPDATE ON journal_entries
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
