-- Append-only revision snapshots: audit trail + history source for diff/restore
-- (roadmap P6). One row per committed document write, capturing post-write
-- state plus who/what/when. Never updated or deleted in normal operation.
CREATE TABLE document_revisions (
  id          BIGSERIAL PRIMARY KEY,
  document_id TEXT NOT NULL,
  revision    INT  NOT NULL,
  op          TEXT NOT NULL,
  actor       TEXT NOT NULL DEFAULT '',
  target_ids  TEXT[] NOT NULL DEFAULT '{}',
  title       TEXT NOT NULL,
  status      TEXT NOT NULL,
  meta        JSONB NOT NULL DEFAULT '{}'::jsonb,
  body        JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, revision)
);

CREATE INDEX document_revisions_doc_idx ON document_revisions (document_id, revision DESC);
