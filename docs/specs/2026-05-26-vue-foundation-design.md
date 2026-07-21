---
date: 2026-05-26
phase: 1.5+
status: approved-pending-review
authors: jeroenpfeil, claude
supersedes: none
---

# Vue foundation — structure & stack

Design spec for Mneme's frontend, executed across Phase 1.5
(scaffold), 1.6 (registry UI), and 1.7 (document viewer). Locks the
shape so the three sessions stay aligned and the design-system import
has a known destination.

Source plan: the Mneme implementation plan (in Mneme), Phase 1, sub-phases 1.5–1.7.

## Goals & non-goals

**Goals**
- One Vite + Vue 3 app under `web/`, embedded into the Go binary via `//go:embed web/dist`.
- Clean home for an externally-authored design system that arrives as a
  CSS tokens file plus React components, which we port to Vue.
- Body-renderer architecture matching the plan: structured JSON → `<component :is>` dispatch, no runtime template compilation.
- Lean dep tree appropriate for a single-developer LAN service.

**Non-goals**
- Multi-app workspace (no `apps/`, no `packages/`, no pnpm workspaces). Lift later only if a second consumer ever shows up.
- State store (no pinia in Phase 1). Composables + URL query state cover the registry+viewer needs.
- Headless component library (no reka-ui in Phase 1). The handful of widgets we need are hand-rolled. Revisit when interactive forms land.
- React preserved anywhere in the repo. Components are ported to Vue;
  the React originals stay outside as reference.

## Workspace shape

Single Vite app under `web/`. Mneme is one Go binary with one frontend
consumer; the monorepo overhead (turbo, pnpm workspaces, multiple
package.json files) buys nothing here. Easy to lift into an
`apps/` + `packages/` layout later if a second consumer materialises.

```
mneme/
  cmd/server/          existing Go entrypoint
  internal/            existing Go packages
  web/                 ONE Vite app
    package.json
    vite.config.ts
    tsconfig.json
    tsconfig.app.json
    tsconfig.node.json
    index.html
    public/
    src/               see "App src/ layout"
    dist/              built artefact, embedded by Go via //go:embed
```

## Stack

| Layer | Choice | Notes |
|---|---|---|
| Build | Vite 7, Vue 3.5, TypeScript 5 | |
| Routing | `vue-router` 5, history mode | `/` → RegistryView, `/doc/:id` → DocumentView |
| Reactive utils | `@vueuse/core` | |
| Styling | Tailwind v4 + `@tailwindcss/vite` | `tokens.css` is the source; Tailwind theme reads from it via `@theme` |
| State | Composables + URL query state | No pinia in Phase 1 |
| Headless primitives | None (hand-rolled) | No reka-ui in Phase 1 |
| Icons | `lucide-vue-next` | |
| Markdown (inline) | `marked`, block-features disabled | Lands in Phase 1.7 |
| Syntax highlighting | `prismjs`, lazy-loaded | Lands in Phase 1.7 |
| Diagrams | `mermaid`, ESM import, dark theme | Lands in Phase 1.7 |
| HTTP | Native `fetch` wrapped in `src/api/client.ts` | No axios |
| Tests | `vitest` + `@vue/test-utils` + `jsdom` | |
| Lint/format | `eslint` (with `eslint-plugin-vue`) + `prettier` | Match Hyperion's configs |

## App `src/` layout

```
web/src/
  api/
    client.ts             typed fetch wrapper; reads VITE_API_URL
    documents.ts          listDocuments(filter), getDocument(id)
    projects.ts           listProjects()
  design-system/          ported from Claude Design output
    tokens.css            single source of design truth (colors, spacing, type, motion, layout)
    Button/               one folder per component (Index.vue + types if needed)
      Index.vue
      index.ts
    Card/
    Tag/
    Tooltip/
    ...
    index.ts              barrel export
  blocks/                 Mneme-specific renderers for body.sections
    MSection.vue
    MSubphase.vue
    MTaskList.vue
    MCallout.vue
    MCode.vue
    MTable.vue
    MDiagram.vue
    MKeyValue.vue
    MText.vue
    BlockRenderer.vue     the <component :is="typeToComponent(block.type)" /> dispatcher
  pages/
    RegistryView.vue
    DocumentView.vue
  components/             app-level glue, not part of the design system
    DocCard.vue
    PhaseTracker.vue
    MetaHeader.vue
    Topbar.vue
    Sidebar.vue
  composables/
    useDocuments.ts       wraps GET /documents with reactive filter state
    useProjects.ts
    useDebounced.ts
  router/
    index.ts
  lib/
    markdown.ts           configured marked instance (inline-only)
    slug.ts               id helpers if needed
  main.ts                 app bootstrap, registers blocks globally
  main.css                imports tokens.css + tailwindcss + @theme mapping
  App.vue
```

**Folder rules of thumb:**

- `design-system/`: visual primitives that could theoretically be reused outside Mneme. Lives behind a single barrel export. No knowledge of Mneme's data model.
- `blocks/`: renderers for the JSON block types defined in the plan. Tightly coupled to the body schema. **Globally registered** in `main.ts` so `<component :is>` can resolve them by component name. Design-system primitives are *not* globally registered — they import normally via the barrel.
- `components/`: app-level composition (`DocCard`, `Sidebar`). Composes `design-system/` primitives. Aware of Mneme data shapes.
- `pages/`: route targets. Compose `components/` + fetch via `api/`.

## Styling integration

`tokens.css` (from Claude Design) is the single source of design truth.
Tailwind v4 reads from it via the `@theme` directive — utility classes
like `bg-surface`, `text-muted`, `border-strong` are auto-derived from
the custom properties. No palette duplication.

```css
/* web/src/main.css */
@import './design-system/tokens.css';
@import 'tailwindcss';

@theme {
  --color-bg: var(--bg);
  --color-surface: var(--bg-surface);
  --color-elevated: var(--bg-elevated);
  --color-hover: var(--bg-hover);
  --color-overlay: var(--bg-overlay);

  --color-text-primary: var(--text-primary);
  --color-text-secondary: var(--text-secondary);
  --color-text-muted: var(--text-muted);
  --color-text-faint: var(--text-faint);

  --color-accent: var(--accent);
  --color-accent-hover: var(--accent-hover);

  --color-border: var(--border);
  --color-border-soft: var(--border-soft);
  --color-border-strong: var(--border-strong);

  /* radii, spacing, motion duration likewise — one line per token */
}
```

**Usage policy in templates:**

- **Typography:** apply the semantic `.mn-display`, `.mn-h1`, `.mn-h2`, `.mn-body`, `.mn-body-sm`, `.mn-label`, `.mn-mono`, `.mn-code-inline` classes from `tokens.css`. Do not redefine these in Tailwind.
- **Color / borders / radii / shadows:** use Tailwind utilities derived from `@theme` (`bg-surface`, `text-muted`, `border`, `rounded-md`, `shadow-md`). Equivalent to using `var(--bg-surface)` etc. directly but more ergonomic.
- **Layout (flex, grid, spacing):** use Tailwind utilities (`flex`, `gap-4`, `p-6`, `grid-cols-[272px_1fr]`).
- **Component-internal styling that doesn't fit utilities:** scoped `<style>` in the SFC, referencing tokens directly (`var(--space-3)`).

## Design system import workflow

When Claude Design delivers a component:

1. Identify the React source — kept outside the repo as reference.
2. Create `web/src/design-system/<ComponentName>/Index.vue`.
3. Port markup → Vue template (`v-if`, slots in place of children).
4. Port logic → `<script setup lang="ts">` using composition API + `@vueuse/core`.
5. Replace any inline styles with Tailwind utilities or scoped `<style>` blocks referencing tokens. **No CSS-in-JS, no `style={}` props.**
6. Add an entry to `design-system/index.ts` for ergonomic imports.
7. Light vitest if there's logic worth covering (most primitives don't need it).

Design tokens travel as the literal `tokens.css` file — copy-paste in, no transformation. When the design system updates, `tokens.css` is replaced wholesale and `@theme` mapping in `main.css` is reviewed for new tokens.

## Build & dev-server integration

**Dev:**
- `cd web && npm run dev` → Vite dev server on `:5173`.
- Vite proxy in `vite.config.ts` forwards `/api/*` to the Go server on `:18080` so the SPA hits the same paths in dev and prod. (`/mcp` is server-to-server — Claude Code calls it directly, the browser never does — so no proxy entry.)
- Go server unchanged.

**Prod:**
- `npm run build` → emits to `web/dist/`.
- The Go server embeds `web/dist` with `//go:embed` (added in Phase 1.5, task 6 of the plan).
- API routes (`/api/v1/*`, `/health`, `/mcp`) take priority on the chi router. All other paths serve `index.html` for SPA history-mode routing.

**Docker** (already in place for Go): the production multi-stage build added in Phase 1.8 will gain a Node stage for the Vue build before the Go stage, so the embedded `dist/` is built from source in the image, not committed.

## Makefile — one-command dev workflow

Phase 1.5 adds a top-level `Makefile` that wraps the now-scattered dev
commands into a few sharp targets. The same file gains `make deploy` /
`make logs` / `make rollback` in Phase 1.8 (already planned), so this
spec just establishes the foundation.

| Target | What it does |
|---|---|
| `make dev` | `docker compose up -d` (Postgres + Go via air, healthchecks wait) → `cd web && npm install --silent && npm run dev`. Foreground process; `Ctrl-C` stops Vite, leaves compose running so the Go server stays warm. |
| `make up` | `docker compose up -d`. Backend only — useful before `make dev` if you want compose logs separate. |
| `make down` | `docker compose down`. Containers gone, volumes preserved. |
| `make reset` | `docker compose down -v`. **Destroys** the Postgres volume. Confirms first (prints what will be lost, requires `RESET=yes` env). |
| `make logs` | `docker compose logs -f --tail=100 app postgres`. |
| `make psql` | `docker compose exec postgres psql -U mneme -d mneme`. |
| `make test` | `go test ./... && cd web && npm test`. |
| `make build` | `cd web && npm run build && go build ./cmd/server`. |
| `make tidy` | `go mod tidy && cd web && npm install`. |
| `make clean` | `rm -rf web/dist web/node_modules/.vite`; leaves `node_modules/` and Go caches alone (those rarely cause real problems and are slow to rebuild). |

**Conventions:**
- `.PHONY` declared once at the top for every target (they're all phony).
- Variables at the top: `COMPOSE`, `WEB`, `GO_PKGS` — never hard-coded in target bodies.
- Help target: `make help` (or bare `make`) prints a table of targets with one-line descriptions. Implementation via `grep ## $(MAKEFILE_LIST)` so descriptions are inline comments next to each target.
- Each target's recipe is one logical command. If a target needs two steps that can run separately, split it into two targets and have the parent depend on both.

**Out of scope for Phase 1.5:** `make deploy`, `make logs-pi`, `make rollback`. Those land in Phase 1.8 alongside the multi-stage Dockerfile.

## Test strategy

- **Vitest** with `jsdom` for components and composables.
- **`@vue/test-utils`** for component mounting.
- **Per-component tests** for non-trivial logic only — most design system primitives are styling and don't need tests.
- **One test per page** to verify the page mounts and calls the expected API method (mocked at the `api/client.ts` boundary).
- **No E2E** in Phase 1. Manual smoke through the running service covers the loop.

## Conventions

- File names: `PascalCase.vue` for components, `camelCase.ts` for non-component modules.
- Imports: prefer the `@` alias for `src/` (`import Button from '@/design-system'`), configured in both `vite.config.ts` and `tsconfig.app.json`.
- `<script setup lang="ts">` everywhere.
- Composables prefixed `use*`, return reactive refs + plain functions (no classes).
- TypeScript: strict mode on; tsconfig follows `@vue/tsconfig`.
- No global mixins, no plugins outside `main.ts`.

## Open questions deferred

- **dev proxy port** — uses `18080` (host-mapped dev container port). If you switch to running `air` natively, the proxy target changes to `:8080`. Trivial to flip.
- **`reka-ui` for Phase 2** — when interactive forms / menus / dialogs arrive, revisit. Hand-rolling each is fine for Phase 1's read-only surface.
- **State store for Phase 2** — composables stay as long as state remains page-local. Pinia goes in the moment we need cross-route shared mutable state.
- **Markdown lib choice** — `marked` is the plan default; `markdown-it` is a fallback if `marked`'s inline-only mode doesn't behave.

## What this spec does NOT cover

- Specific component contracts (props, slots, events) for design-system entries — emerge as components are ported.
- Exact registry/viewer UX — defined in plan tasks 1.6 and 1.7.
- Pi deployment of the built frontend — addressed in 1.8.

The spec stops at "scaffold + folder rules + dep tree + integration paths". Implementation details are owned by the per-session plans.
