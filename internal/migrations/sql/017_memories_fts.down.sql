DROP INDEX IF EXISTS memories_search_vector_idx;
ALTER TABLE memories DROP COLUMN IF EXISTS search_vector;
DROP FUNCTION IF EXISTS memories_search_vector(TEXT, TEXT, TEXT);
