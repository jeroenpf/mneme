DROP INDEX IF EXISTS journal_entries_search_vector_idx;
ALTER TABLE journal_entries DROP COLUMN IF EXISTS search_vector;
DROP FUNCTION IF EXISTS journal_search_vector(TEXT, TEXT, TEXT[], TEXT[]);
