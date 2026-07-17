# Getting started with Mneme

Mneme is a **local, single-user knowledge service** for AI-assisted development. It runs
entirely on your Mac under Docker Compose and is reached at **`https://mneme.dev:8443`**.
Nothing leaves your machine.

This guide gets you from a fresh clone to a running instance with **one command** —
`make start` — which you re-run whenever updates arrive. It is not a dev setup; it builds
the app and serves the real, embedded UI.

---

## Prerequisites

Install these once on your Mac:

- **Docker Desktop** — running. (Postgres and the Go app run in containers.)
- **Xcode Command Line Tools** (gives you `git` and `make`):
  ```bash
  xcode-select --install
  ```
- **Homebrew**, then:
  ```bash
  brew install mkcert node
  ```
  - Firefox user? Also `brew install nss` so mkcert can trust the cert there.

You do **not** need Go installed — the Go server compiles inside the Docker container.

---

## 1. Get the code

Clone the repository (it's private — you'll need access granted to your GitHub account),
then enter it:

```bash
git clone https://github.com/jeroenpf/mneme.git
cd mneme
```

---

## 2. One-time host setup

```bash
sudo make setup-host
```

This is idempotent and safe to re-run. It:

- installs a **mkcert** root CA into your macOS keychain (so the local TLS cert is trusted),
- issues a leaf cert into `.certs/` for `mneme.dev`,
- adds `127.0.0.1  mneme.dev` to `/etc/hosts`.

> **Why `mneme.dev` and not `mneme.local`?** macOS routes `*.local` names through mDNS,
> which stalls resolution ~5 s before it ever reads `/etc/hosts`. `.dev` resolves instantly.

Check it anytime with the non-destructive scorecard:

```bash
make verify-host
```

The `https://mneme.dev:8443/health` line will be **red until the stack is running** — that's
expected at this point. Everything else should be green.

---

## 3. Create your `.env` (Voyage AI — for semantic search)

Out of the box, Mneme's search is **full-text only**. To get semantic / hybrid search
(finding things by meaning, not just keywords), give it a **Voyage AI** embedding key.

1. **Get a key.** Sign up at <https://www.voyageai.com/>, open the dashboard, and create an
   API key (it looks like `pa-...`).

2. **Create a `.env` file in the repo root** with:

   ```dotenv
   MNEME_VOYAGE_API_KEY=pa-your-key-here
   MNEME_VOYAGE_RPM=3
   ```

Notes:

- `.env` is **git-ignored** — never commit it. Docker Compose reads it automatically to fill
  in the container's environment.
- **`MNEME_VOYAGE_RPM`** proactively throttles requests to stay under your account's rate
  limit. Voyage's **free / no-payment-method tier is limited (~3 requests/min)**, so set
  `3` there. On a paid tier you can raise it or drop the line entirely (default `0` = no
  proactive throttle, just back off on rate-limit errors).
- The embedding model defaults to `voyage-4-large`. Override with `MNEME_VOYAGE_MODEL=...`
  only if you have a reason to.
- **Backfill is automatic.** Once the key is set and the stack (re)starts, a background
  worker embeds your existing content — no manual step. So you can also **add `.env` later**
  and just re-run `make start` to switch semantic search on.

**Don't want semantic search?** Skip this whole step. Mneme runs fine with full-text search
and no `.env`.

> `.env` is only ever for **secrets** like this API key. Non-secret local config (ports,
> URLs, service names) lives in Mneme's env registry, not here.

---

## 4. Build and run — the one command

```bash
make start
```

This builds the SPA, embeds it into the Go binary, brings up Postgres + the Go server
(terminating TLS on `127.0.0.1:8443` — no reverse proxy), and applies any database
migrations. When it finishes, open:

```
https://mneme.dev:8443
```

The mkcert cert is trusted, so there are no browser warnings.

Optionally load some sample data to look at:

```bash
make seed
```

Confirm the backend is healthy (use **system** curl — Homebrew's curl won't trust the
keychain CA):

```bash
/usr/bin/curl -sS https://mneme.dev:8443/health   # {"db":true,"ok":true}
```

---

## Updating later

When a new version arrives, pull it and run the same one command:

```bash
git pull        # or however you receive updates
make start
```

`make start` rebuilds the SPA, re-embeds it, applies any new database migrations, and
restarts the app. Your data (the Postgres volume) is preserved.

---

## Everyday commands

| Command | What it does |
|---|---|
| `make start` | Build + run at `https://mneme.dev:8443`. Re-run to update. |
| `make down` | Stop the containers (your data volume is kept). |
| `make logs` | Tail the backend logs (app + postgres). |
| `make psql` | Open a `psql` shell inside the postgres container. |
| `make seed` | Load dev sample data (idempotent). |
| `make verify-host` | Re-check the host setup (cert, hosts entry, health). |
| `make help` | The full target list. |

To wipe everything and start fresh (**destroys your data**):

```bash
RESET=yes make reset
```

---

## Wiring Claude Code into your projects (optional)

To let Claude Code push and query knowledge over MCP, register the server from any project
directory:

```bash
claude mcp add --transport http mneme https://mneme.dev:8443/mcp
```

Full details — registering a project, telling the agent to use it — are in
[`docs/using-mneme.md`](docs/using-mneme.md).

---

## Troubleshooting

- **`:8443` shows "web/dist not built".** You started the backend without building the UI
  (e.g. `make up`). Run `make start`.
- **`mneme.dev` won't resolve / cert not trusted.** Re-run `sudo make setup-host`, then
  `make verify-host`.
- **Connection refused while the container looks healthy.** Docker Desktop's port forwarder
  can occasionally wedge — quit and reopen Docker Desktop, then `make start`.
- **Health check fails only with Homebrew curl.** Use `/usr/bin/curl` — only the system curl
  trusts the mkcert CA in the macOS keychain.
- **Semantic search isn't finding things by meaning.** Confirm `.env` has a valid
  `MNEME_VOYAGE_API_KEY`, re-run `make start`, and give the background worker a moment to
  backfill. Check `make logs` for Voyage rate-limit messages (raise `MNEME_VOYAGE_RPM` or
  upgrade your Voyage tier if you see them).
