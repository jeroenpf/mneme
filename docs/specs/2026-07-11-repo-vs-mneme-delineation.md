# Repo vs. Mneme — scope delineation

**Date:** 2026-07-11
**Status:** accepted

## Context

Mneme is exclusively a local development tool. Its real competitor is not git — it is
the existing workflow of gitignored `.architecture/*.md` files plus Claude Code's native
file tools. Mneme earns its keep through three things markdown files cannot do:
cross-project recall, stable block addressability with server-side semantics
(`tick_task`, `advance_phase`), and document lifecycle beyond a single repo.
That only works if the boundary between "belongs in the repo" and "belongs in Mneme"
is crisp.

## Decision

The boundary is a **lifecycle, not a topic**:

> **Git owns the code. Mneme owns the work.**

- **Repo (git):** documents that describe *the code* — present-tense, durable, needed by
  anyone (human or agent) holding only a clone. README, CLAUDE.md, accepted ADRs and
  specs, API contracts, runbooks.
- **Mneme (local):** documents that describe *the work* — past- and future-tense,
  evolving, personal. Plans, session state, journals, error/solution notes, brainstorms,
  cross-project conventions.

Documents are **born in Mneme** (cheap, mutable, searchable). The few that harden into
durable decisions **graduate to the repo** as committed markdown.

### Litmus tests

- *"Fresh clone on a new machine, Mneme unreachable — is something missing to understand
  or build this?"* → repo.
- *"Will this be stale or misleading in a month?"* / *"Is this about my process rather
  than the artifact?"* → Mneme.

### Doctype mapping

| Doctype | Home |
|---|---|
| `plan`, `report`, `brainstorm`, `journal` | Mneme-native; never committed |
| `spec`, `adr` | Drafted in Mneme; graduate to the repo once accepted |

## Bridge — four rules

1. **Pointer, never copy.** A document lives on exactly one side; the other side holds a
   reference. Each repo's CLAUDE.md carries one line: *"Plans and working memory: Mneme
   project `<slug>` via MCP."* Mneme docs reference repo files by path — the MCP
   `Instructions` string already tells the LLM to read repo files from disk.
2. **Graduation ritual.** When a plan completes or a decision solidifies: distill it to a
   committed md spec/ADR, then archive the Mneme doc with
   `meta.superseded_by: <repo path>`. Convention first; an `export_document` tool if the
   friction shows up (queued in plan §1.9).
3. **Project binding.** A Mneme project binds to a git remote / root path (the
   `Document.repo` field exists for this), so a Claude Code session in that directory
   auto-discovers its documents. Pairs with the §1.9 orientation-doc convention.
4. **No repo ingestion.** Mneme never indexes or serves what git owns. Claude Code reads
   and greps repo files natively; ingesting them would make Mneme a stale cache of git
   and split the source of truth. This drops the former §1.9 items "repo-file MCP
   resources" and "repo markdown ingestion".

## Consequences

- The primary consumer is Claude Code over MCP. The Vue UI is a **read-mostly viewer** —
  browsing and review only, no editing features (plan §1.6/§1.7 stay lean).
- Phase 2 (cross-project memory & recall) is the differentiated value over plain md
  files; Phase 1 polish must not delay it.
- Writing to Mneme must stay near **zero-ceremony**: a journal/brainstorm entry costs one
  call and no schema negotiation (plan §1.9). Structured blocks remain for `plan`-type
  docs, where tasks and phases earn their cost. A memory system that is expensive to
  write to dies.
- Bootstrap exception (resolved 2026-07-20): the mneme repo originally tracked
  `.architecture/` because the plan predated a working Mneme. That plan has since graduated
  into Mneme; the repo now keeps only distilled specs under `docs/specs/` (see
  `architecture.md`), and `.architecture/` is gitignored local scratch.
