# Mneme references — compatibility, migration, backup & rollback

**Date:** 2026-07-19
**Status:** accepted

## Context

Mneme gives every addressable entity a short, stable, server-generated public id
(`doc_…`, `dec_…`, `blk_…`, `task_…`, …) and a self-describing `mneme://` reference
grammar so a pasted reference resolves with one `resolve_reference` call. This document
records the compatibility guarantees, the migration path for existing data, how to back
up and roll back, and the security boundary the reference system deliberately does **not**
cross. It is the durable companion to `plan-generated-ids-and-references` (the evolving
plan lives in Mneme).

## Identity model (three layers, kept distinct)

- **Public id** — the one portable identity: `<prefix>_<12 Crockford base32 chars>`. Stable
  across edits, renames, and moves; safe to paste into a reference. Minted by `internal/ids`.
- **Internal id** — the storage key. Documents key on their **slug** (`documents.id`);
  decisions/snippets/journal/solutions key on a UUID. Never an addressing surface on its own.
- **Slug / title** — mutable human labels. Display and search affordances, never identity.

A block or task also carries a **document-local id**. It may be a generated `blk_`/`task_`
id or a **legacy semantic id** (`overview`, `s6-t1`) preserved from before generated ids. In
a reference the *owner* is always a strict `doc_` public id; the *child* id is lenient — any
non-empty, slash-free token. The relation segment (`block`/`task`), not the id's prefix,
marks the kind.

## Compatibility guarantees

1. **Dual addressing for documents.** `GET /api/v1/documents/{id}`, the MCP `get_document`,
   and every surgical MCP tool accept **either** the slug **or** the `doc_` public id
   (`loadDoc` falls back to the by-public-id lookup). Existing slug-addressed links and
   bookmarks keep working; a resolved reference's public id works directly.
2. **Legacy semantic ids resolve.** `resolve_reference` resolves
   `mneme://document/doc_…/block/<semantic>` and `.../task/<semantic>` by walking the
   document body, so references to pre-migration content resolve without rewriting stored
   documents. The copy-reference UI is forward-compatible: it emits a reference using the
   block/task's actual id, semantic or generated.
3. **No silent id churn.** Existing ids — generated or semantic — are never rewritten. The
   migration only *adds* ids to nodes that lack one.
4. **Deprecation window.** Slug-based document addressing is retained indefinitely for now;
   there is no removal date. Any future deprecation will be announced here first.

## Migration — `mneme migrate ids`

Some documents written before strict id validation may have blocks/tasks without ids
(unaddressable) or, rarely, duplicate ids (ambiguous). `mneme migrate ids` reconciles them:

- **Reports before changing.** With no flag it scans every document and prints what *would*
  change — how many ids would be minted per document — and persists nothing.
- **`--apply`** mints ids for nodes missing one and persists. Existing ids are preserved.
- **Duplicates are never auto-resolved.** A document with duplicate ids is reported as a
  problem for manual repair — choosing which node keeps the id is a human decision.

The operation is idempotent: a document whose tree is already complete and unique is left
untouched. Run the report, back up (below), then `--apply`.

## Backup & rollback

- **Backup.** Postgres: `pg_dump` (or `make psql`-adjacent tooling) of the `mneme` database.
  SQLite (portable binary): copy the single `*.db` file while the server is stopped. A
  first-class `mneme export` / `import` round-trip is planned (roadmap P6-t7) and currently
  stubbed.
- **Rollback.** The id migration is **additive** (it only adds missing ids), so the safe
  rollback is to restore the pre-migration backup. Because ids are stable, references copied
  after a migration keep resolving after a restore as long as the referenced nodes still
  exist.

## Security & future workspace-authorization boundary

A public id is an **identifier, never an access-control token** (`internal/ids` doctrine).
Its opacity is not a secret and knowing one grants no authority. `resolve_reference` runs
the **same** access checks as a normal read — today that is "local single user, full
access." Should Mneme ever grow multi-workspace access control, reference resolution must
gate on the caller's workspace exactly as normal reads do; resolving a reference must never
become a side channel around authorization. This plan builds the local identity and
reference foundation such features could reuse — it does **not** add workspaces, RBAC,
sharing, or hosted links.
