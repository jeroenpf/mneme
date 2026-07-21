# Mneme — Architecture

**Status:** accepted · **Updated:** 2026-07-20

Mneme is a local, single-user AI development knowledge service. It gives Claude Code
(and a human, through a read-mostly web UI) a durable place to push and query the *work*
around a codebase — plans, decisions, journals, snippets, memory, and context bundles —
over an MCP server, with cross-project recall that plain markdown files can't provide.

This file is the durable architecture overview: enough to understand and build the
project from a fresh clone. Live, evolving work docs (plans, journals) live *in Mneme
itself*, not the repo — see [Repo vs. Mneme delineation](2026-07-11-repo-vs-mneme-delineation.md).

## Shape

A single Go binary. It:

- serves an **MCP server** (over HTTP, or HTTPS in `mneme.dev` mode) — the primary consumer is Claude Code,
- serves a **REST API** that backs the web UI,
- serves the **Vue 3 SPA** viewer, embedded into the binary via `//go:embed`,
- stores everything in **SQLite by default** (pure-Go `modernc.org/sqlite`, no CGO) or **PostgreSQL** when configured,
- terminates its own **TLS** (no reverse proxy).

`make build-portable` produces one self-contained, CGO-free file with the SPA baked in —
no external runtime dependencies, no Docker, no database server (SQLite is in-process).

```
Claude Code ──MCP over HTTP(S)──▶ ┌──────────────────────────┐
Browser     ──HTTP(S)──────────▶  │ mneme (single Go binary) │
                                  │  MCP · REST · SPA · TLS  │
                                  └───────────┬──────────────┘
                                              ▼
                                  SQLite (default)  or  PostgreSQL
```

## Run modes

`mneme init` runs an interactive wizard that writes a config; `mneme server` runs it.
Storage and network mode are chosen independently:

| Network mode | URL | Setup |
|---|---|---|
| `localhost` (default) | `http://localhost:8765` | Zero-setup, plain HTTP |
| `mneme.dev` | `https://mneme.dev:8443` | Trusted HTTPS — runs `mkcert` and adds an `/etc/hosts` entry (`mneme.dev → 127.0.0.1`) |

The host is `mneme.dev`, **not** `mneme.local`: macOS routes `*.local` through mDNS,
which stalls resolution ~5s before it reads `/etc/hosts`.

**Storage:** SQLite (default, under `~/.mneme/`) or a PostgreSQL DSN.
**Search:** lexical by default (SQLite FTS / Postgres `tsvector`), or semantic when
embeddings are enabled (Voyage AI); hybrid ranking merges the two.

## CLI

| Command | Purpose |
|---|---|
| `mneme init` | Interactive wizard → writes config (backend, search, network mode) |
| `mneme server` | Run the server (MCP + REST + SPA) |
| `mneme doctor` | Environment scorecard — cert, `/etc/hosts` entry, `/health`, backups |
| `mneme migrate` | Apply schema migrations (SQLite + Postgres SQL under `internal/migrations/`) |
| `mneme export` / `mneme import` | Back up / restore all local knowledge to a single file |

## Storage & data model

A **document** is metadata plus a body. The body is a JSON tree of **typed blocks**; the
Vue viewer maps each block's `type` to a registered component via `<component :is>` — no
runtime template compilation.

- **meta:** `type` (`plan | spec | adr | report | brainstorm | journal`), `title`,
  `project`, `status`, `tags`, optional `phases[]`, optional `custom_fields{}`.
- **body.sections:** an ordered tree of blocks.

| Block type | Purpose |
|---|---|
| `section` | Groups blocks under a heading (`children[]`) |
| `subphase` | Plan phase with `tasks[]` |
| `text` | Prose (inline markdown) |
| `tasklist` | Standalone task list |
| `callout` | `info \| warn \| success \| danger \| note` |
| `code` | Code block with language + copy button |
| `table` | Columns + rows |
| `diagram` | Mermaid |
| `keyvalue` | Two-column key/value grid |

Every block has a stable, document-unique `id`. The surgical MCP tools (`tick_task`,
`update_section`, `advance_phase`, …) address blocks by that `id` — server-validated and
~100× cheaper than re-emitting a whole document with `push_document`.

Beyond documents, Mneme stores cross-project **memory**, a **decision log**, a **snippet
library**, a **dev journal**, an **error/solution DB**, and an **env registry**, and
assembles a **context bundle** (one call returns a session-ready markdown digest). All are
reachable through a single ranked `search(q, types?)`.

## Repo layout

| Path | Contents |
|---|---|
| `cmd/mneme/` | Entry point — thin (signal wiring + process exit only) |
| `internal/` | `cli`, `api`, `mcp`, `store` (SQLite + Postgres), `embed`, `migrations`, `web` (embed), `config`, … |
| `web/` | Vue 3 + Vite SPA (built and embedded into the binary) |
| `docs/` | User guides; `docs/specs/` holds accepted specs/ADRs (this file included) |

## Interfaces & boundaries

- **MCP server** is the primary write path. Mutations go through MCP; the Vue UI is a
  **read-mostly viewer** (browse and review, no editing).
- **REST API** backs the SPA.
- **Repo owns the code, Mneme owns the work.** Git holds durable, present-tense docs about
  the artifact (this file, README, accepted specs/ADRs). Mneme holds evolving work docs
  (plans, journals, brainstorms). Pointers cross the line, never copies; Mneme never
  ingests repo files. Full decision: [delineation spec](2026-07-11-repo-vs-mneme-delineation.md).

## Constraints

- **Personal-scale.** Single-user tool. When on Postgres it's tuned small
  (`shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`); no reason to grow it.
- **Local-only by design.** Reachable only from the machine it runs on.
- **Pragmatic dependencies (Go).** Prefer the standard library; use high-quality libraries
  (`chi`, `pgx/v5`, `cobra`) where they clearly simplify the code. Raw SQL, no ORM.
- **Vue 3 standard runtime.** The body is structured JSON dispatched via `<component :is>`
  — no runtime template compilation.
