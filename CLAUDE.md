# Mneme

Local AI dev knowledge service. Go + PostgreSQL + Vue3, runs locally on the laptop under Docker Compose, reached at `https://mneme.local` (via a `127.0.0.2` loopback alias so it never collides with other Docker projects). Exposes an MCP server so Claude Code can push/query documents.

**Source of truth:** [`.architecture/plans/2026-05-22_mneme-implementation_1.html`](.architecture/plans/2026-05-22_mneme-implementation_1.html) — full stack, schema, block types, phase plan. Read it before making non-trivial changes. Deployment design: [`.architecture/specs/2026-05-26-local-deployment.md`](.architecture/specs/2026-05-26-local-deployment.md).

## Constraints worth remembering

- **Personal-scale dataset** — Postgres tuned to `shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`. Right-sized for a single-user dev tool; no reason to consume more.
- **Local-only by design** — reachable only from this Mac. Phone/iPad/other devices can't hit `mneme.local`. Acceptable: the consumer is Claude Code on the laptop.
- **Pragmatic Dependencies (Go)** — Prefer the standard library, but use high-quality dependencies (e.g., config managers, routers) if they significantly simplify the code. Avoid heavy "magic" frameworks like ORMs; stick to raw SQL (`pgx/v5`).
- **Vue3 standard runtime** — body is structured JSON dispatched via `<component :is>`, no runtime template compilation.

## Go Guidelines & Idioms

- **Structure**: Keep `main.go` clean (wiring & lifecycle only). Business and routing logic belongs in `internal/`.
- **Interfaces for Data**: Define interfaces (e.g., `Store`) to abstract database implementations and make unit testing easier.
- **Error Handling**: Always wrap errors (`fmt.Errorf("action: %w", err)`) and define typed domain errors (e.g., `ErrNotFound`) for API translation.
- **Strong Typing**: Use custom types for enums instead of raw strings to leverage compile-time safety.
- **Context**: Pass `context.Context` down through the entire call stack to handle timeouts gracefully.
