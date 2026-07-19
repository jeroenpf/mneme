DROP INDEX IF EXISTS embeddings_vec_idx;
ALTER TABLE embeddings DROP COLUMN embedding;
ALTER TABLE embeddings ADD COLUMN embedding vector(1536);
ALTER TABLE embeddings ALTER COLUMN model SET DEFAULT 'voyage-code-2';
CREATE INDEX embeddings_vec_idx
  ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
