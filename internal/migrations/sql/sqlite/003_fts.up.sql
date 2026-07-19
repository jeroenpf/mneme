-- FTS5 full-text search (plan-sqlite-backend p2-t4). One content-owning FTS5
-- virtual table per searchable type, its columns mirroring the inputs of the
-- corresponding Postgres <type>_search_vector function in the same order, so a
-- weighted bm25() call at query time reproduces the setweight A/B/C tiers
-- (Postgres weights {A,B,C,D} = {1.0, 0.4, 0.2, 0.1}). The FTS rowid is kept in
-- lockstep with the base row's rowid by INSERT/UPDATE/DELETE triggers, so
-- search can join back to the base table for title/project/updated_at while
-- snippet() highlights over the stored FTS copy. JSON array columns
-- (tags/accomplished/deferred) are flattened to space-joined text; the JSON
-- body is indexed as-is (mirrors Postgres body::text, weighted lowest).

-- documents: title(A), ticket(B), tags(B), body(C)
CREATE VIRTUAL TABLE documents_fts USING fts5(title, ticket, tags, body);

CREATE TRIGGER documents_fts_ai AFTER INSERT ON documents BEGIN
  INSERT INTO documents_fts (rowid, title, ticket, tags, body)
  VALUES (
    NEW.rowid, NEW.title, COALESCE(NEW.ticket, ''),
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.tags)),
    NEW.body
  );
END;
CREATE TRIGGER documents_fts_ad AFTER DELETE ON documents BEGIN
  DELETE FROM documents_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER documents_fts_au AFTER UPDATE ON documents BEGIN
  DELETE FROM documents_fts WHERE rowid = OLD.rowid;
  INSERT INTO documents_fts (rowid, title, ticket, tags, body)
  VALUES (
    NEW.rowid, NEW.title, COALESCE(NEW.ticket, ''),
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.tags)),
    NEW.body
  );
END;

-- decisions: title(A), rationale(B), decision(B), alternatives(C), consequences(C)
CREATE VIRTUAL TABLE decisions_fts USING fts5(title, rationale, decision, alternatives, consequences);

CREATE TRIGGER decisions_fts_ai AFTER INSERT ON decisions BEGIN
  INSERT INTO decisions_fts (rowid, title, rationale, decision, alternatives, consequences)
  VALUES (NEW.rowid, NEW.title, NEW.rationale, NEW.decision, NEW.alternatives, NEW.consequences);
END;
CREATE TRIGGER decisions_fts_ad AFTER DELETE ON decisions BEGIN
  DELETE FROM decisions_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER decisions_fts_au AFTER UPDATE ON decisions BEGIN
  DELETE FROM decisions_fts WHERE rowid = OLD.rowid;
  INSERT INTO decisions_fts (rowid, title, rationale, decision, alternatives, consequences)
  VALUES (NEW.rowid, NEW.title, NEW.rationale, NEW.decision, NEW.alternatives, NEW.consequences);
END;

-- snippets: title(A), description(B), content(C)
CREATE VIRTUAL TABLE snippets_fts USING fts5(title, description, content);

CREATE TRIGGER snippets_fts_ai AFTER INSERT ON snippets BEGIN
  INSERT INTO snippets_fts (rowid, title, description, content)
  VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content);
END;
CREATE TRIGGER snippets_fts_ad AFTER DELETE ON snippets BEGIN
  DELETE FROM snippets_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER snippets_fts_au AFTER UPDATE ON snippets BEGIN
  DELETE FROM snippets_fts WHERE rowid = OLD.rowid;
  INSERT INTO snippets_fts (rowid, title, description, content)
  VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content);
END;

-- solutions: error_description(A), solution(B)
CREATE VIRTUAL TABLE solutions_fts USING fts5(error_description, solution);

CREATE TRIGGER solutions_fts_ai AFTER INSERT ON solutions BEGIN
  INSERT INTO solutions_fts (rowid, error_description, solution)
  VALUES (NEW.rowid, NEW.error_description, NEW.solution);
END;
CREATE TRIGGER solutions_fts_ad AFTER DELETE ON solutions BEGIN
  DELETE FROM solutions_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER solutions_fts_au AFTER UPDATE ON solutions BEGIN
  DELETE FROM solutions_fts WHERE rowid = OLD.rowid;
  INSERT INTO solutions_fts (rowid, error_description, solution)
  VALUES (NEW.rowid, NEW.error_description, NEW.solution);
END;

-- journal: summary(A), session_ref(B), accomplished(C), deferred(C)
CREATE VIRTUAL TABLE journal_fts USING fts5(summary, session_ref, accomplished, deferred);

CREATE TRIGGER journal_fts_ai AFTER INSERT ON journal_entries BEGIN
  INSERT INTO journal_fts (rowid, summary, session_ref, accomplished, deferred)
  VALUES (
    NEW.rowid, NEW.summary, NEW.session_ref,
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.accomplished)),
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.deferred))
  );
END;
CREATE TRIGGER journal_fts_ad AFTER DELETE ON journal_entries BEGIN
  DELETE FROM journal_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER journal_fts_au AFTER UPDATE ON journal_entries BEGIN
  DELETE FROM journal_fts WHERE rowid = OLD.rowid;
  INSERT INTO journal_fts (rowid, summary, session_ref, accomplished, deferred)
  VALUES (
    NEW.rowid, NEW.summary, NEW.session_ref,
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.accomplished)),
    (SELECT COALESCE(group_concat(value, ' '), '') FROM json_each(NEW.deferred))
  );
END;

-- memories: key(A), value(B), area(C)
CREATE VIRTUAL TABLE memories_fts USING fts5(key, value, area);

CREATE TRIGGER memories_fts_ai AFTER INSERT ON memories BEGIN
  INSERT INTO memories_fts (rowid, key, value, area)
  VALUES (NEW.rowid, NEW.key, NEW.value, COALESCE(NEW.area, ''));
END;
CREATE TRIGGER memories_fts_ad AFTER DELETE ON memories BEGIN
  DELETE FROM memories_fts WHERE rowid = OLD.rowid;
END;
CREATE TRIGGER memories_fts_au AFTER UPDATE ON memories BEGIN
  DELETE FROM memories_fts WHERE rowid = OLD.rowid;
  INSERT INTO memories_fts (rowid, key, value, area)
  VALUES (NEW.rowid, NEW.key, NEW.value, COALESCE(NEW.area, ''));
END;
