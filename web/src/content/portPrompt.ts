// The one-time "port your existing workflow to mneme" prompt offered on the
// Help page. It lives outside the component because it is long, and because
// interpolating the live install endpoints makes it worth testing on its own.
//
// The prompt is written for the *user's* agent, run once inside a target
// repo: it briefs the agent on what mneme is, then walks it through
// inventory → mapping → instruction rewrite → user-confirmed migration.

export interface PortPromptInstall {
  /** Base URL of this install, e.g. http://localhost:8901 */
  url: string
  /** MCP endpoint of this install, e.g. http://localhost:8901/mcp */
  mcpEndpoint: string
}

export function buildPortPrompt({ url, mcpEndpoint }: PortPromptInstall): string {
  return `You are working in a repository whose development workflow I want to port to
Mneme. Read all of this before acting, then work the tasks in order.

## What Mneme is

Mneme is a local, single-user knowledge service for AI-assisted development.
It runs on this machine; you reach it as the MCP server named \`mneme\`
(endpoint: ${mcpEndpoint}). It is the durable home for knowledge that is
still evolving — plans, decisions, notes — so future sessions start with
context instead of amnesia.

Its content types:

- **Documents** — plans, specs, ADRs, reports, brainstorms. Bodies are
  structured blocks (sections, task lists, tables) with stable ids. Create or
  fully rewrite with \`push_document\`; edit surgically with \`add_section\`,
  \`update_section\`, \`add_task\`, \`update_task\`, \`tick_task\`,
  \`advance_phase\` (prefer the surgical tools — they are server-validated
  and far cheaper than re-pushing a whole document).
- **Decisions** — durable choices with rationale: \`log_decision\`,
  \`query_decisions\`.
- **Journal** — an append-only dev log per project: \`append_journal\`.
- **Snippets** — reusable commands and config: \`save_snippet\`,
  \`search_snippets\`.
- **Solutions** — problem→fix pairs worth remembering: \`log_solution\`,
  \`find_solution\`.
- **Memory** — durable free-form facts by scope: \`set_memory\`,
  \`get_memory\`.
- **Env** — key/value facts about the dev environment: \`set_env\`,
  \`get_env\`.

Two verbs matter most day to day: \`get_context_bundle(project)\` — one call
returning memory, active-plan status, recent decisions, snippets and journal
as a session-start digest — and \`search(q)\` — one ranked query across
every content type, to run before assuming something does not exist.

## The delineation rule

The repo owns the code and its durable, present-tense documentation: README,
contributor docs, accepted specs and ADRs — whatever describes the artifact
as it is. Mneme owns the evolving work around it: plans, in-flight specs,
brainstorms, journals, the decision log. Documents are born in Mneme and, if
they harden into durable documentation, graduate to the repo as markdown.
Across the line use pointers (a \`mneme://\` reference in the repo, a repo
path in a Mneme doc) — never copies. Never bulk-import repo files into
Mneme.

## Task 0 — verify connectivity

Call \`list_documents\` on the \`mneme\` server. If its tools are not
available, stop and tell me to wire it up first via the "Connect an agent"
section at ${url}/help.

## Task 1 — inventory the current workflow

Survey how this repo manages development knowledge today:

- Agent instruction files: CLAUDE.md, AGENTS.md, GEMINI.md, .cursorrules,
  .github/copilot-instructions.md, and anything similar.
- In-repo planning systems: docs/ trees (specs, plans, adr/, rfcs/),
  .architecture/-style directories, TODO / ROADMAP / NOTES files.
- Workflow plugins or skills that prescribe where documents go (for example
  superpowers, which writes specs and plans under docs/superpowers/).
- External systems referenced from the repo or its instructions: Notion,
  Linear, Jira, Obsidian, wikis.

Use git history to tell living documents from fossils. Then report back:
what exists, where it lives, and what each piece is actually used for.

## Task 2 — propose the mapping

For each artifact class you found, state where it lives from now on.
Defaults to apply unless something argues otherwise:

- Active plans and in-flight specs → Mneme documents (type plan / spec).
- Brainstorms and scratch notes → Mneme brainstorm docs or journal entries.
- Accepted ADRs and hardened specs → stay in the repo; log *new* decisions
  in Mneme as they happen.
- Setup/run commands worth reusing → snippets; machine and tooling facts →
  env.
- Conventions and preferences that are not repo documentation → memory.

Where a workflow skill prescribes writing specs or plans to repo paths, keep
the process but redirect its output to Mneme.

## Task 3 — rewrite the agent instructions

Update this repo's instruction file(s) (CLAUDE.md / AGENTS.md / …) so every
future session uses Mneme by habit:

- If the project is not registered yet: \`create_project(slug, name)\` —
  derive the slug from the repo name and confirm it with me.
- Session start: \`get_context_bundle("<slug>")\`.
- Before assuming something is missing: \`search()\`.
- New plans, specs and brainstorms are created in Mneme, not as repo files;
  they graduate to the repo only when they harden.
- \`log_decision\` for durable choices as they are made; \`append_journal\`
  at the end of significant sessions.
- Prefer the surgical document tools over full rewrites.
- State the delineation rule briefly, including graduation and
  pointers-not-copies.

Show me the diff before writing anything.

## Task 4 — propose the migration, then STOP

Produce a migration table: every artifact from task 1 → import into Mneme
(as which type), stay in the repo, or stay put with a pointer left behind.
Recommend importing only what is alive and still evolving; leave history
where it is.

Present the table and wait for my explicit confirmation. Do not migrate
anything before I confirm. Once I confirm: register the project if needed,
import the approved items with sensible metadata, leave a pointer at each
origin, and delete nothing without separate, explicit approval.

Finish with a short report: what changed, what was migrated, and anything I
should still do by hand.`
}
