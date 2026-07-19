-- Optimistic-concurrency + history support (roadmap P6). Every document
-- carries a monotonic revision, bumped on each update; existing rows start at 1.
ALTER TABLE documents ADD COLUMN revision INT NOT NULL DEFAULT 1;
