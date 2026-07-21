# Phase 2.7 — Env registry (design)

**Status:** accepted · **Date:** 2026-07-13 · **Parent plan:** `#sp-2-7`

## Problem

A coding agent working in a project keeps re-discovering non-secret local
facts: "what port does the API run on?", "what's the Docker service name for
postgres?", "what's the local URL?". These live nowhere durable, so the agent
asks or greps for them every session. Phase 2.7 gives each project a small,
non-secret **env registry** — key / value / description — that Claude Code reads
automatically at session start (via `get_context_bundle`) and can query or write
over MCP. It is deliberately **not** a secrets store.

## Shape: a flatter memory system

The env registry mirrors the Phase 2.1 memory system end to end, minus the
`global → project → area` hierarchy. Every env entry is project-scoped; there is
no scope enum and no merge step. Where memory has `(scope, project, area, key)`,
env has `(project, key)`. The store/REST/MCP/Vue layering is otherwise identical,
so this design references the memory implementation as the template throughout.

## Non-goals

- **Not searched, not embedded.** Env stays out of `store.SearchTypes` (memory
  is too). No embedding enqueue on write.
- **Not for secrets.** No encryption, no redaction, no access control — it is a
  local single-user dev tool serving non-secret config. The UI carries a
  prominent, persistent warning; the docs (Deliverable B) repeat it.

## Data model & migration (`014_env`)

`internal/migrations/sql/014_env.{up,down}.sql`:

```sql
CREATE TABLE env_entries (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project     TEXT NOT NULL REFERENCES projects(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  description TEXT,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT env_entries_identity UNIQUE (project, key)
);
```

- The FK auto-names its constraint `env_entries_project_fkey`; the store keys on
  that name to translate a violation into `ErrInvalidProject` (exactly as
  `SetMemory` keys on `memories_project_fkey`).
- No `updated_at` trigger. `SetEnv` stamps `updated_at = now()` inside the upsert,
  mirroring `SetMemory`. The down migration is `DROP TABLE IF EXISTS env_entries;`.

`models.EnvEntry` (in `internal/models/models.go`):

```go
type EnvEntry struct {
    ID          string    `json:"id"`
    Project     string    `json:"project"`
    Key         string    `json:"key"`
    Value       string    `json:"value"`
    Description *string   `json:"description,omitempty"` // nullable, like Project.Description
    UpdatedAt   time.Time `json:"updated_at"`
}
```

`Project` is a non-pointer string: unlike memory (where global entries have no
project), every env entry has one. `Description` is a pointer so "absent" round-
trips as JSON `null`/omitted and SQL `NULL`.

## Store contract

Add to the `Store` interface and implement on `*PostgresStore`:

```go
// SetEnv upserts an env entry by (project, key), filling e.ID and
// e.UpdatedAt from the DB. Returns ErrInvalidProject when project is
// an unknown slug.
SetEnv(ctx context.Context, e *models.EnvEntry) error

// ListEnv returns a project's env entries ordered by key.
ListEnv(ctx context.Context, project string) ([]*models.EnvEntry, error)

// DeleteEnv removes one entry by (project, key). Returns ErrNotFound
// when no row matched.
DeleteEnv(ctx context.Context, project, key string) error
```

- `SetEnv`: `INSERT ... ON CONFLICT (project, key) DO UPDATE SET value =
  EXCLUDED.value, description = EXCLUDED.description, updated_at = now()
  RETURNING id, updated_at`. PUT semantics: the upsert replaces both `value` and
  `description`, so omitting a description clears it. FK violation on
  `env_entries_project_fkey` → `ErrInvalidProject`.
- `ListEnv`: `SELECT ... WHERE project = $1 ORDER BY key`.
- `DeleteEnv`: `DELETE ... WHERE project = $1 AND key = $2`; `RowsAffected() == 0`
  → `ErrNotFound`.

**Testcontainer test** (`postgres_test.go`, env section): upsert-then-list;
upsert-again updates value + description (and bumps `updated_at`); description
NULL round-trips; delete removes; delete-missing → `ErrNotFound`; set with an
unknown project → `ErrInvalidProject`.

## REST API

`internal/api/env.go` (`EnvHandler{Store}`), wired in `routes.go`. All three
routes require `project` as a query param (env has no un-scoped listing — this is
the one deliberate divergence from memory, whose list is un-filtered):

| Method + path | Query | Body | Success | Errors |
|---|---|---|---|---|
| `GET /api/v1/env` | `project` (required) | — | `200 {items: EnvEntry[]}` | 400 if `project` missing |
| `PUT /api/v1/env/{key}` | `project` (required) | `{value, description?}` | `200 EnvEntry` | 400 missing project/key/value; 400 unknown project |
| `DELETE /api/v1/env/{key}` | `project` (required) | — | `204` | 404 unknown key |

Reuses `writeJSON` / `writeError` / `decodeJSON` / `writeStoreError` (which maps
`ErrInvalidProject`→400 "unknown project", `ErrNotFound`→404). `value` empty →
400 "value is required", matching `MemoryHandler.Upsert`. Table-driven handler
test in `env_test.go` using the existing `setup_test.go` harness.

## MCP tools

`internal/mcp/tools_env.go`, three tools registered in `server.go`:

- **`get_env{project}`** → flat `{key: value}` object (`map[string]string`). The
  "just knows the port" contract: compact, paste-free, for the agent to read.
- **`set_env{project, key, value, description?}`** → the stored `EnvEntry`.
- **`list_env{project}`** → full `EnvEntry` records incl. description.

Input validation mirrors `tools_memory.go`: trim inputs, `project`/`key`/`value`
required where applicable, `translateStoreErr` for store errors. Per-tool test in
`tools_env_test.go` against the shared MCP test harness. `get_env`'s flat-map
projection (drop everything but key→value) is a small pure helper, unit-tested
directly like `mergeMemory`.

## Context bundle fold-in

`get_env` is folded into `get_context_bundle` so a session-start bundle carries
env automatically — **full records, with descriptions in the digest** (decided:
the "what is X" that makes "what port does X run on" answerable belongs at session
start, and matches how decisions/snippets/journal ride the bundle as full records).

- `internal/bundle/bundle.go`: add `Env []*models.EnvEntry` to `Bundle`; add an
  `envEntries(ctx, project)` step calling `store.ListEnv(project)` (a real project
  always exists here — `Assemble` already verified it). Ordered by key.
- `internal/bundle/render.go`: a `## Env` section after `## Memory`, rendering
  `- KEY = value` and appending ` — description` when present; `_none_` when empty,
  matching the other sections' stable shape.
- The MCP `get_context_bundle` handler and `/api/v1/bundle` need no changes — they
  serialize whatever `Bundle` carries. Extend `bundle_test.go` (assembler +
  render) and the MCP/API bundle tests' expectations.

## Vue UI

Per-project env manager, reached from the Topbar `env →` link (added alongside
`memory →`), route `/env`:

- `web/src/api/env.ts` — `EnvEntry` type; `listEnv(project)`, `setEnv({project,
  key, value, description?})`, `deleteEnv({project, key})` over the REST layer.
- `web/src/composables/useEnv.ts` — fetches a project's entries; refetches on
  project change; sorted by key.
- `web/src/pages/EnvView.vue` — a project `<select>` (populated from
  `useProjects`, defaulting to the first, deep-linkable via `?project=`), a
  **prominent persistent banner: "⚠ Never store secrets here — this is non-secret
  local config (ports, service names, URLs). Secrets do not belong in Mneme."**,
  then a key / value / description table with inline edit (blur-to-save, like
  MemoryView), a delete affordance per row, and an add row (key + value +
  description). Writes go through the REST layer — the sanctioned-write exception
  in the read-mostly UI, exactly as MemoryView does.
- Router entry + Topbar link + their test updates. Component tests
  (`EnvView.test.ts`, `useEnv.test.ts`, `env.test.ts` for the api client) mirror
  the memory suite.

## Bookkeeping (parent plan HTML)

In the Mneme implementation plan:

- Tick all four `#sp-2-7` task checkboxes → header count `4 / 4`.
- Flip `sub-phase-num todo">2.7` and `sub-pip todo">2.7` to `done` (match a
  completed sibling like 2.6), making the hero's "Phase 2 · complete" true.
- In the MCP tool-surface table, add a `2.7` group with `get_env` / `set_env` /
  `list_env` rows (the table currently lists none for 2.7).
- Do **not** touch 1.9 (intentional backlog) or the hero copy beyond what the
  done-flip implies.

## Deliverable B — Run + usage documentation (repo-tracked)

Durable, present-tense docs about the artifact, written to git (not Mneme):

- **`README.md`** — what Mneme is; how to run it, derived from the `Makefile` +
  `CLAUDE.md` (`make setup-host` once, `make dev`, the `https://mneme.dev:8443`
  URL and why `.dev` not `.local`, mkcert/TLS, `make test`/`build`/`verify-host`).
  Every documented command is run and verified before it is written down.
- **`docs/using-mneme.md`** — the concrete steps + copy-pasteable block a *new*
  project adds so its coding agent uses Mneme:
  - Registering the MCP server (Claude Code: `claude mcp add --transport http
    mneme https://mneme.dev:8443/mcp`), including the verified way the client
    trusts the mkcert cert. **Proven to connect before being written down.**
  - A lean **AGENTS.md / CLAUDE.md snippet**: Mneme is the source of truth for
    plans/specs/decisions/snippets/journal/env; call `get_context_bundle(project)`
    at session start, `search` before assuming something's missing, record
    decisions/snippets/journal/solutions/env as a session side-effect. It
    complements — does not duplicate — the `instructions` string the server
    already pushes on connect (`internal/mcp/server.go`).
  - A note that the env registry (this phase) is where per-project
    ports/URLs/service names go, and that secrets never do.

Docs are not test-driven but are verified live (commands run, MCP connection
proven) before the effort is called complete.

## Testing & conventions

- TDD throughout the Go/Vue work: failing test first, run it, confirm it fails
  for the right reason, minimal impl, re-run. Exceptions (verified by consumers'
  tests + a live check): pure `main.go`/route wiring and Vue route/config.
- `make test` stays hermetic (Docker/testcontainers, no network, no keys).
  `go test ./...` + vitest green; `go vet ./...` clean.
- One commit per plan task; final `docs(plan): phase 2.7 complete — env registry`.
```
