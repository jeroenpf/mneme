-- updated_at triggers (plan-sqlite-backend p2-t3). Postgres uses a BEFORE
-- UPDATE trigger that rewrites NEW.updated_at; SQLite AFTER triggers cannot
-- mutate NEW, so each fires a follow-up UPDATE stamping updated_at. The
-- `WHEN NEW.updated_at = OLD.updated_at` guard makes the trigger self-limiting:
-- the inner UPDATE changes updated_at, so even with recursive_triggers on it
-- will not re-fire. Mirrors the five tables that carry a set_updated_at trigger
-- in Postgres (documents, decisions, snippets, journal_entries, solutions);
-- memories/env stamp updated_at explicitly in their upserts.

CREATE TRIGGER documents_set_updated_at
AFTER UPDATE ON documents FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE documents SET updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER decisions_set_updated_at
AFTER UPDATE ON decisions FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE decisions SET updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER snippets_set_updated_at
AFTER UPDATE ON snippets FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE snippets SET updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER journal_entries_set_updated_at
AFTER UPDATE ON journal_entries FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE journal_entries SET updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER solutions_set_updated_at
AFTER UPDATE ON solutions FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE solutions SET updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = NEW.id;
END;
