# Mneme

Local AI dev knowledge service. Go + PostgreSQL + Vue3, deployed to a Raspberry Pi 4 (4GB) at `mneme.local`. Exposes an MCP server so Claude Code can push/query documents.

**Source of truth:** [`.architecture/plans/2026-05-22_mneme-implementation_1.html`](.architecture/plans/2026-05-22_mneme-implementation_1.html) — full stack, schema, block types, phase plan. Read it before making non-trivial changes.

## Constraints worth remembering

- **Pi 4 / 4GB** — Postgres tuned to `shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`. Keep server memory bounded; it shares the Pi with Pi-hole.
- **Pragmatic Dependencies (Go)** — Prefer the standard library, but use high-quality dependencies (e.g., config managers, routers) if they significantly simplify the code. Avoid heavy "magic" frameworks like ORMs; stick to raw SQL (`pgx/v5`).
- **Vue3 standard runtime** — body is structured JSON dispatched via `<component :is>`, no runtime template compilation.

## Go Guidelines & Idioms

- **Structure**: Keep `main.go` clean (wiring & lifecycle only). Business and routing logic belongs in `internal/`.
- **Interfaces for Data**: Define interfaces (e.g., `Store`) to abstract database implementations and make unit testing easier.
- **Error Handling**: Always wrap errors (`fmt.Errorf("action: %w", err)`) and define typed domain errors (e.g., `ErrNotFound`) for API translation.
- **Strong Typing**: Use custom types for enums instead of raw strings to leverage compile-time safety.
- **Context**: Pass `context.Context` down through the entire call stack to handle timeouts gracefully.
