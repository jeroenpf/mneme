# Mneme

Local AI dev knowledge service. Go + PostgreSQL + Vue3, deployed to a Raspberry Pi 4 (4GB) at `mneme.local`. Exposes an MCP server so Claude Code can push/query documents.

**Source of truth:** [`.architecture/plans/2026-05-22_mneme-implementation_1.html`](.architecture/plans/2026-05-22_mneme-implementation_1.html) — full stack, schema, block types, phase plan. Read it before making non-trivial changes.

## Constraints worth remembering

- **Pi 4 / 4GB** — Postgres tuned to `shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`. Keep server memory bounded; it shares the Pi with Pi-hole.
- **Go stdlib + minimal deps** — `chi` for routing, `pgx/v5` direct (no ORM), `os.LookupEnv` for config (no config library), `golang-migrate` for SQL migrations.
- **Vue3 standard runtime** — body is structured JSON dispatched via `<component :is>`, no runtime template compilation.
