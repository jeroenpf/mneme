-- Optimistic-concurrency + history support (roadmap P6). Mirrors postgres 018.
ALTER TABLE documents ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;
