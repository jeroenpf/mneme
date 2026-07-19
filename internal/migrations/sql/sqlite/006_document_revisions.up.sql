-- Append-only revision snapshots (roadmap P6). Mirrors postgres 019; JSON
-- columns are TEXT (tags/meta/body follow the same convention as documents).
CREATE TABLE document_revisions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id TEXT NOT NULL,
  revision    INTEGER NOT NULL,
  op          TEXT NOT NULL,
  actor       TEXT NOT NULL DEFAULT '',
  target_ids  TEXT NOT NULL DEFAULT '[]',
  title       TEXT NOT NULL,
  status      TEXT NOT NULL,
  meta        TEXT NOT NULL DEFAULT '{}',
  body        TEXT NOT NULL DEFAULT '{}',
  created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  UNIQUE (document_id, revision)
);

CREATE INDEX document_revisions_doc_idx ON document_revisions (document_id, revision DESC);
