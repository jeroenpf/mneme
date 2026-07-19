-- Every top-level entity gains a stable, opaque public_id: the portable
-- identity that agents paste and reference, distinct from the internal
-- UUID/slug primary key. See internal/ids for the matching contract — body
-- block/task ids are minted in Go by the same rules; top-level ids are minted
-- here so every insert path (Go, seed SQL, manual) is covered.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- gen_public_id mints "<prefix>_<12 lower-case Crockford-base32 chars>"
-- (alphabet excludes i/l/o/u; 60 bits of entropy), mirroring internal/ids.
-- VOLATILE so a column DEFAULT re-evaluates it per row: that both backfills
-- every existing row distinctly during ADD COLUMN and stamps future inserts.
CREATE OR REPLACE FUNCTION gen_public_id(prefix TEXT) RETURNS TEXT AS $$
DECLARE
  alphabet CONSTANT TEXT := '0123456789abcdefghjkmnpqrstvwxyz';
  raw      BYTEA := gen_random_bytes(12);
  body     TEXT  := '';
  i        INT;
BEGIN
  FOR i IN 0..11 LOOP
    -- low 5 bits of each byte index into the 32-char alphabet (1-based substr)
    body := body || substr(alphabet, (get_byte(raw, i) & 31) + 1, 1);
  END LOOP;
  RETURN prefix || '_' || body;
END;
$$ LANGUAGE plpgsql VOLATILE;

ALTER TABLE projects        ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('prj');
ALTER TABLE documents       ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('doc');
ALTER TABLE decisions       ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('dec');
ALTER TABLE snippets        ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('snip');
ALTER TABLE journal_entries ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('jrnl');
ALTER TABLE solutions       ADD COLUMN public_id TEXT NOT NULL DEFAULT gen_public_id('sol');

-- Unique indexes enforce per-entity global uniqueness and would fail the
-- migration loudly if the backfill ever produced a collision.
CREATE UNIQUE INDEX projects_public_id_idx        ON projects (public_id);
CREATE UNIQUE INDEX documents_public_id_idx       ON documents (public_id);
CREATE UNIQUE INDEX decisions_public_id_idx       ON decisions (public_id);
CREATE UNIQUE INDEX snippets_public_id_idx        ON snippets (public_id);
CREATE UNIQUE INDEX journal_entries_public_id_idx ON journal_entries (public_id);
CREATE UNIQUE INDEX solutions_public_id_idx       ON solutions (public_id);
