# Getting started with Mneme (portable binary)

Mneme ships as a **single self-contained binary** — no Docker, no Postgres, no external
services. Storage is a local SQLite file; everything the UI needs is embedded. This is the
path for trying Mneme on your own machine.

The whole flow is three steps: **download → `mneme init` → `mneme server`.**

## 1. Get the binary

Grab the `mneme` binary for your machine (or build it yourself — see
[Building from source](#building-from-source)). It is pure Go: one file, nothing to install
alongside it.

```bash
chmod +x mneme
./mneme --help
```

## 2. `mneme init`

Run the setup wizard. It asks a few questions and writes `~/.mneme/settings.toml` (mode
`0600`):

```bash
./mneme init
```

- **Data** — a **SQLite file** (default `~/.mneme/mneme.db`) or an existing PostgreSQL DSN.
  SQLite is the zero-setup default.
- **Embeddings** — **off by default** (lexical / full-text search only, nothing leaves your
  machine). Turn it on to add semantic search via a [Voyage](https://voyageai.com) API key;
  the key is stored in `settings.toml` in plaintext (same trust model as a `.env`).
- **Networking** — pick one:
  - **localhost + plain HTTP** (default): serves on `http://localhost:8765`, loopback only,
    zero certificate setup. The wizard bumps to the next free port if 8765 is taken.
  - **mneme.dev + HTTPS** (opt-in): the wizard runs `mkcert` to install a locally-trusted CA
    and issue a certificate, and adds the `mneme.dev → 127.0.0.1` entry to `/etc/hosts`
    (needs `sudo`; it prints the exact command if it can't write the file itself). Serves on
    `https://mneme.dev:8443`. Requires `mkcert` on your PATH (`brew install mkcert`).

The wizard finishes by offering to start the server immediately.

## 3. `mneme server`

Start the service (this is also the default the wizard offers to run for you):

```bash
./mneme server
```

It applies migrations to the SQLite file on first run, then serves the REST API, the MCP
endpoint at `/mcp`, and the embedded web UI. Open the URL the wizard reported
(`http://localhost:8765` by default) and you'll see the viewer.

Stop it with `Ctrl-C` — it drains connections and shuts down cleanly.

## Check your setup: `mneme doctor`

One command diagnoses the whole install and exits non-zero if anything is broken:

```bash
./mneme doctor
```

It scores configuration, the database + migrations, networking (certificate / hosts in HTTPS
mode), full-text search, and the embeddings provider.

## Connect Claude Code

Point Claude Code at the MCP endpoint (use the URL for your chosen mode):

```bash
# localhost mode
claude mcp add --transport http mneme http://localhost:8765/mcp

# mneme.dev mode (the mkcert CA the wizard installed makes this trusted)
claude mcp add --transport http mneme https://mneme.dev:8443/mcp
```

Then register your project once (`create_project`) and follow
[Using Mneme in a project](using-mneme.md).

## Configuration & precedence

Settings resolve highest-wins: **command flag → `MNEME_*` env var → `~/.mneme/settings.toml`
→ built-in default.** So the file is the friendly front door, env vars still override it
(for CI or advanced setups), and a flag beats everything:

```bash
./mneme server --port 9000            # flag wins
MNEME_DSN=sqlite:///data/x.db ./mneme server   # env overrides the file
./mneme server --config /path/to/settings.toml # point at a different file
```

Key env vars: `MNEME_DSN`, `MNEME_HOST` (bind interface; defaults to loopback),
`MNEME_PORT`, `MNEME_VOYAGE_API_KEY`, `MNEME_TLS_CERT` / `MNEME_TLS_KEY`.

## Security posture

- **Loopback only.** The server binds `127.0.0.1` by default — it is not reachable from the
  LAN. Set `MNEME_HOST=0.0.0.0` only if you deliberately want that.
- **Origin validation.** Every request with a browser `Origin` header must match the
  configured allow-list, on `/mcp` and every route — the real defense against DNS-rebinding,
  enforced in both HTTP and HTTPS modes. Native clients (Claude Code, curl) send no Origin
  and are unaffected.
- **Your data stays local.** With embeddings off, nothing leaves the machine. With embeddings
  on, document text is sent to Voyage to compute vectors — that is the one external flow, and
  it is opt-in.

## Building from source

```bash
make build-portable
```

This builds the SPA, embeds it, and produces a stripped, `CGO_ENABLED=0` static `./mneme`
binary with no external runtime dependencies. (`make dev` / `make start` remain the
Docker-based developer loop; the release binary does not include the live-reload tooling.)
