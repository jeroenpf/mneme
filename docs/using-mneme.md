# Using Mneme in a project

Mneme is the source of truth for a project's **evolving, local work knowledge** — plans and
notes while they're in flight, decisions, code snippets, the dev journal, solved errors, and
the non-secret env registry. A coding agent (Claude Code) reads from and writes to Mneme over
MCP, so it starts each session already oriented instead of re-deriving context.

> **Mneme is not a substitute for the repo.** It is a *local* service — a single machine, not
> version-controlled, backed up, or shared. Anything durable, shareable, or that belongs under
> version control stays in the repo: code, the README, accepted specs and ADRs. Mneme holds
> only the local, in-flight work knowledge that does **not** belong in git. When a doc hardens,
> it graduates to the repo as markdown and Mneme keeps a pointer to it — never a copy. Nothing
> here replaces committing to git.

This guide is the few concrete steps a **new** project adds to start using Mneme.

## 1. Register the MCP server

Mneme must be running (`make dev`, or the app container up) and the host set up
(`make setup-host` once — it installs the mkcert CA into the macOS trust store). Then, from
the project directory:

```bash
claude mcp add --transport http mneme https://mneme.dev:8443/mcp
```

**Cert trust:** nothing extra is needed. Because `make setup-host` installed the mkcert CA
into the system trust store, Claude Code trusts `https://mneme.dev:8443` out of the box —
you do **not** need `NODE_EXTRA_CA_CERTS` or any other CA flag. Confirm it connected:

```bash
claude mcp list
# mneme: https://mneme.dev:8443/mcp (HTTP) - ✔ Connected
```

If Claude Code is already open, run `/mcp` to reconnect and pick up the server (and any
newly added tools).

> **Scope:** `claude mcp add` defaults to **local** scope (stored in `~/.claude.json` for
> this project only). To commit the registration so teammates/other checkouts get it, add
> `--scope project` — that writes a `.mcp.json` in the repo root instead.

## 2. Register the project in Mneme

Mneme's documents, decisions, and env entries are keyed by a project slug, and it must
exist first. Once, ask the agent (or call the tool):

```
create_project(slug: "<your-slug>", name: "<Human Name>")
```

## 3. Tell the agent to use Mneme

Add this block to the project's `AGENTS.md` (or `CLAUDE.md`). It complements — it does not
duplicate — the `instructions` string Mneme's MCP server already pushes on connect; keep it
lean:

```markdown
## Mneme — source of truth for local work knowledge

This project uses **Mneme** (local MCP server) for local, evolving work knowledge that does
not belong in the repo — plans and notes in flight, decisions, code snippets, the dev
journal, solved errors, and the env registry. It is **not** a substitute for git: code and
hardened docs (README, accepted specs/ADRs) live in the repo; Mneme holds the work in flight.

- **At session start:** call `get_context_bundle(project: "<slug>")` — one call returns
  merged memory, the active plan's status, recent decisions, relevant snippets, recent
  journal, and the env registry (ports / URLs / service names).
- **Before assuming something is missing:** `search(q, ...)` across all knowledge types.
- **Record as a side-effect of working:** `log_decision` (why you chose X),
  `save_snippet` (a reusable pattern), `append_journal` (what you did / deferred),
  `log_solution` (an error + its fix), `set_env` (a non-secret port / URL / service name).
- **Never put secrets in `set_env`** — it is plaintext, non-secret config only.

Project slug: `<slug>`.
```

## The env registry

`set_env` / `get_env` are where a project's **non-secret local config** lives — ports,
service names, local URLs, Docker service names. Record them once and the agent stops asking
"what port does X run on?"; it reads them automatically from `get_context_bundle` at session
start. Browse and edit them in the UI at `https://mneme.dev:8443/env`.

**Secrets never go here.** No API keys, tokens, passwords, or credentialed connection
strings — the registry is plaintext and unencrypted. Keep secrets in your usual secret
manager / `.env` files that stay out of Mneme.
