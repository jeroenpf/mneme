# Mneme

Local AI dev knowledge service. Go + PostgreSQL + Vue3, runs locally on the laptop under Docker Compose, reached at `https://mneme.dev:8443` (`mneme.dev` → `127.0.0.1` via `/etc/hosts`; the Go server terminates TLS with an mkcert cert, no reverse proxy). Exposes an MCP server so Claude Code can push/query documents. The host is `mneme.dev`, **not** `mneme.local` — macOS routes `*.local` through mDNS, which stalls resolution ~5s before reading `/etc/hosts`.

**Source of truth:** [`.architecture/plans/2026-05-22_mneme-implementation_1.html`](.architecture/plans/2026-05-22_mneme-implementation_1.html) — full stack, schema, block types, phase plan. Read it before making non-trivial changes. Deployment design: [`.architecture/specs/2026-05-26-local-deployment.md`](.architecture/specs/2026-05-26-local-deployment.md).

## Constraints worth remembering

- **Personal-scale dataset** — Postgres tuned to `shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`. Right-sized for a single-user dev tool; no reason to consume more.
- **Local-only by design** — reachable only from this Mac. Phone/iPad/other devices can't hit `mneme.local`. Acceptable: the consumer is Claude Code on the laptop.
- **Repo owns the code, Mneme owns the work** — git holds durable, present-tense docs about the artifact (README, accepted specs/ADRs); Mneme holds evolving work docs (plans, journals, notes, brainstorms). Docs are born in Mneme and graduate to the repo as md when they harden. Pointers only across the line, never copies; Mneme never ingests repo files. The Vue UI is a read-mostly viewer — mutations go through MCP. Full decision: [`.architecture/specs/2026-07-11-repo-vs-mneme-delineation.md`](.architecture/specs/2026-07-11-repo-vs-mneme-delineation.md).
- **Pragmatic Dependencies (Go)** — Prefer the standard library, but use high-quality dependencies (e.g., config managers, routers) if they significantly simplify the code. Avoid heavy "magic" frameworks like ORMs; stick to raw SQL (`pgx/v5`).
- **Vue3 standard runtime** — body is structured JSON dispatched via `<component :is>`, no runtime template compilation.

## Go Guidelines & Idioms

- **Structure**: Keep `main.go` clean (wiring & lifecycle only). Business and routing logic belongs in `internal/`.
- **Interfaces for Data**: Define interfaces (e.g., `Store`) to abstract database implementations and make unit testing easier.
- **Error Handling**: Always wrap errors (`fmt.Errorf("action: %w", err)`) and define typed domain errors (e.g., `ErrNotFound`) for API translation.
- **Strong Typing**: Use custom types for enums instead of raw strings to leverage compile-time safety.
- **Context**: Pass `context.Context` down through the entire call stack to handle timeouts gracefully.

## Dev workflow

A top-level `Makefile` wraps the dev loop. Common targets:

- `make setup-host` — one-time per machine (needs `sudo`): installs the mkcert CA + leaf cert into `.certs/` and writes the `mneme.dev → 127.0.0.1` `/etc/hosts` entry. `make verify-host` is a non-destructive scorecard (cert, hosts entry, `https://mneme.dev:8443/health`).
- `make dev` — full stack: postgres + Go (live-reload via air, TLS on `127.0.0.1:8443`) + Vite on `:5273`. Vite proxies `/api` and `/health` to `https://mneme.dev:8443`; it exports `NODE_EXTRA_CA_CERTS` so Node trusts the mkcert CA.
- `make test` — `go test ./...` + Vue/vitest.
- `make build` — builds the SPA into `web/dist/`, copies into `internal/web/dist/` for `//go:embed`, then builds the `mneme` binary at the repo root (`go build -o mneme ./cmd/mneme`).
- `make logs` / `make psql` — backend log tail / psql shell inside the container.
- `make seed` — load dev sample data (`scripts/dev-seed.sql`) into postgres; idempotent.
- `make down` / `make reset` — stop containers / drop the postgres volume (the latter requires `RESET=yes`).

Run `make help` for the full list.
