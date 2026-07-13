# Mneme

A local, single-user AI-dev **knowledge service**. Go + PostgreSQL + Vue 3, running
on your laptop under Docker Compose. Mneme exposes an **MCP server** so a coding agent
(Claude Code) can push and query a project's evolving work knowledge — plans, specs,
decisions, code snippets, a dev journal, solved errors, and a non-secret env registry —
plus a read-mostly web UI to browse it all.

The split is deliberate: **the repo owns the code, Mneme owns the work.** Git holds
durable, present-tense docs about the artifact (README, accepted specs/ADRs); Mneme holds
the evolving work docs (plans, journals, notes). Mneme is **local-only and is not a
substitute for version control** — anything durable or shareable belongs in git. Work is
born in Mneme and graduates to the repo as markdown when it hardens; only pointers cross the
line, never copies. Full rationale:
[`.architecture/specs/2026-07-11-repo-vs-mneme-delineation.md`](.architecture/specs/2026-07-11-repo-vs-mneme-delineation.md).

To wire Mneme into another project so its coding agent uses it, see
[`docs/using-mneme.md`](docs/using-mneme.md).

## Requirements

- **Docker + Docker Compose** — Postgres and the Go app run in containers.
- **[mkcert](https://github.com/FiloSottile/mkcert)** — issues the locally-trusted TLS cert.
- **make**, and (for local binary builds / tests) **Go 1.25+** and **Node 22+**.
- macOS. Mneme is reachable only from this machine by design — other devices can't hit it.

## First-time setup (once per machine)

```bash
make setup-host   # needs sudo
```

This installs the mkcert CA + a leaf cert into `.certs/`, and maps `mneme.dev → 127.0.0.1`
in `/etc/hosts` (plus the supporting loopback/LaunchDaemon plumbing). Verify it anytime
with the non-destructive scorecard:

```bash
make verify-host
```

You want all-green: mkcert on PATH, root CA present, cert valid and covering `mneme.dev`,
the `/etc/hosts` entry, and a `200` from `https://mneme.dev:8443/health`.

> **Why `mneme.dev`, not `mneme.local`?** macOS routes `*.local` names through mDNS, which
> stalls resolution for ~5 s before it ever reads `/etc/hosts`. `.dev` resolves instantly.

## Run it

```bash
make dev
```

Brings up the full stack: Postgres + the Go server (live-reloaded by `air`, terminating
**TLS directly** on `127.0.0.1:8443` with the mkcert leaf cert — no reverse proxy) + the
Vite dev server on `:5273`. Vite proxies `/api` and `/health` to `https://mneme.dev:8443`
and trusts the mkcert CA via `NODE_EXTRA_CA_CERTS`. `Ctrl-C` stops Vite; the backend
containers keep running (`make down` to stop them).

Open the app at:

```
https://mneme.dev:8443
```

> **TLS + curl:** the Go process serves HTTPS with the mkcert cert. Use **`/usr/bin/curl`**
> — the system curl trusts the mkcert CA in the macOS keychain; a Homebrew curl won't.
> ```bash
> /usr/bin/curl -sS https://mneme.dev:8443/health   # {"db":true,"ok":true}
> ```

## Common tasks

| Command | What it does |
|---|---|
| `make dev` | Full dev stack (postgres + Go via air + Vite). |
| `make up` / `make down` | Start / stop the backend containers (volumes preserved). |
| `make test` | Go tests (`go test ./...`) + Vue tests (vitest). Needs Docker — the Go suite uses testcontainers. |
| `make build` | Build the SPA, copy it into `internal/web/dist/` for `//go:embed`, then build the Go binary at `cmd/server/server`. |
| `make logs` | Tail the backend logs (app + postgres). |
| `make psql` | Open a `psql` shell inside the postgres container. |
| `make seed` | Load dev sample data (`scripts/dev-seed.sql`); idempotent. |
| `make tidy` | Refresh Go + Node dependency manifests. |
| `make reset` | **Destructive** — drop containers and the postgres volume. Requires `RESET=yes`. |
| `make help` | The full target list. |

> **After `make build`, restart the app container:** `air` live-reloads Go edits off the
> bind mount, but it does **not** re-embed the `//go:embed` SPA. If `:8443` serves a stale
> page after a UI change, run `docker compose restart app` so it re-embeds
> `internal/web/dist/`.

## Layout

- `cmd/server/` — entrypoint (wiring & lifecycle only).
- `internal/` — the app: `store` (raw SQL over pgx/v5, no ORM), `api` (go-chi REST),
  `mcp` (the MCP tool surface), `bundle` (the session-start context bundle), `migrations`,
  `models`, `embed` (Voyage embeddings for hybrid search).
- `web/` — the Vue 3 SPA (embedded into the Go binary at build time).
- `.architecture/` — plans and specs (the durable, graduated ones; live work lives in Mneme).
