-- Embeddings + terminal-failure tracking (plan-sqlite-backend p2-t5). The
-- pgvector column becomes a BLOB of packed little-endian float32 (Go handles
-- pack/unpack); brute-force cosine in Go replaces the ivfflat index — exact and
-- sub-millisecond at personal scale. Indexes on (source_type, source_id) and
-- project match the Postgres access paths used by the reconciler and search.

CREATE TABLE embeddings (
  id           TEXT PRIMARY KEY,
  source_type  TEXT NOT NULL,
  source_id    TEXT NOT NULL,
  chunk_id     TEXT NOT NULL,
  chunk_text   TEXT NOT NULL,
  embedding    BLOB,                            -- packed float32, little-endian
  project      TEXT,
  source_title TEXT NOT NULL,
  model        TEXT DEFAULT 'voyage-4-large',
  created_at   TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  UNIQUE (source_type, source_id, chunk_id)
);
CREATE INDEX embeddings_source_idx  ON embeddings (source_type, source_id);
CREATE INDEX embeddings_project_idx ON embeddings (project);

CREATE TABLE embed_failures (
  source_type     TEXT NOT NULL,
  source_id       TEXT NOT NULL,
  error           TEXT NOT NULL,
  attempts        INTEGER NOT NULL DEFAULT 1,
  first_failed_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  last_failed_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
  PRIMARY KEY (source_type, source_id)
);
CREATE INDEX embed_failures_type_idx ON embed_failures (source_type);
