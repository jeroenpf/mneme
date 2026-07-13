-- voyage-4-large is 1024-dim (voyage-code-2 was 1536). The table has no
-- rows yet, so drop the index, recreate the column at the new dimension,
-- flip the model default, and rebuild the ivfflat index. No backfill.
DROP INDEX IF EXISTS embeddings_vec_idx;
ALTER TABLE embeddings DROP COLUMN embedding;
ALTER TABLE embeddings ADD COLUMN embedding vector(1024);
ALTER TABLE embeddings ALTER COLUMN model SET DEFAULT 'voyage-4-large';
CREATE INDEX embeddings_vec_idx
  ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
