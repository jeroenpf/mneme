CREATE TABLE embeddings (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_type  TEXT NOT NULL,
  source_id    TEXT NOT NULL,
  chunk_id     TEXT NOT NULL,
  chunk_text   TEXT NOT NULL,
  embedding    vector(1536),
  project      TEXT,
  source_title TEXT NOT NULL,
  model        TEXT DEFAULT 'voyage-code-2',
  created_at   TIMESTAMPTZ DEFAULT now(),
  UNIQUE (source_type, source_id, chunk_id)
);

CREATE INDEX embeddings_vec_idx
  ON embeddings USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);
CREATE INDEX embeddings_project_idx     ON embeddings (project);
CREATE INDEX embeddings_source_type_idx ON embeddings (source_type);
