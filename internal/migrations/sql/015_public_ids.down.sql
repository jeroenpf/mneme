DROP INDEX IF EXISTS projects_public_id_idx;
DROP INDEX IF EXISTS documents_public_id_idx;
DROP INDEX IF EXISTS decisions_public_id_idx;
DROP INDEX IF EXISTS snippets_public_id_idx;
DROP INDEX IF EXISTS journal_entries_public_id_idx;
DROP INDEX IF EXISTS solutions_public_id_idx;

ALTER TABLE projects        DROP COLUMN IF EXISTS public_id;
ALTER TABLE documents       DROP COLUMN IF EXISTS public_id;
ALTER TABLE decisions       DROP COLUMN IF EXISTS public_id;
ALTER TABLE snippets        DROP COLUMN IF EXISTS public_id;
ALTER TABLE journal_entries DROP COLUMN IF EXISTS public_id;
ALTER TABLE solutions       DROP COLUMN IF EXISTS public_id;

DROP FUNCTION IF EXISTS gen_public_id(TEXT);
-- pgcrypto is left installed: harmless, and other objects may rely on it.
