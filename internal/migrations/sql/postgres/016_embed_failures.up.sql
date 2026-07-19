-- Terminal embedding failures per source, so the pipeline can surface a
-- "failed" count and offer manual retry. One row per source (keyed like the
-- reconciliation buckets); attempts accrues across retries.
CREATE TABLE embed_failures (
  source_type     TEXT NOT NULL,
  source_id       TEXT NOT NULL,
  error           TEXT NOT NULL,
  attempts        INT NOT NULL DEFAULT 1,
  first_failed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_failed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (source_type, source_id)
);

CREATE INDEX embed_failures_type_idx ON embed_failures (source_type);
