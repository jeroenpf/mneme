# Mneme UI — three reading-first themes + a persistent nav rail

**Date:** 2026-07-14
**Status:** accepted

## Context

Mneme's UI is a read-mostly viewer for long-form work docs — plans, journals,
decisions. Two things worked against that reading:

- **A single dark, neon-cyan theme** (base `#0d0e11` + `#22d3ee`, Syne / DM Sans).
  It reads poorly for extended editorial text, and there was no light option or
  any choice at all.
- **Per-page top menus.** Ten pages each hand-rolled their own
  `<header class="topbar">`, so primary navigation *changed or disappeared* as you
  moved between routes — disorienting for a knowledge base you browse constantly.

The design system was already fully token-driven (`web/src/design-system/tokens.css`;
components use only `var(--…)`, and both Prism and mermaid read tokens), which made a
palette-level change cheap. An interactive mockup drove the visual target:
[`design/theme-explorations.html`](../../design/theme-explorations.html).

## Decision

### Three reading-first themes, runtime-switchable

Replace the dark-only palette with three fixed, editorial themes selected by a
`[data-theme]` attribute on `<html>`:

| Theme | Character | Accent |
|---|---|---|
| **Paper** | warm editorial light (default) | steel blue `#2f5d80` |
| **Slate** | cool modern light, rounder corners, Inter-led display | teal `#0f7c78` |
| **Ink** | warm charcoal `#1a1918` (not pure black), a night option | amber-gold `#e6975a` |

- `tokens.css` holds theme-**invariant** tokens (spacing, type scale, motion, layout)
  in `:root`, plus per-theme override blocks (`:root[data-theme="…"]`) for palette,
  fonts, and radii. Type is Archivo / Inter / IBM Plex Mono.
- A tiny `useTheme` composable owns the active theme: initialise from `localStorage`
  (`mneme.theme`), else `prefers-color-scheme` (dark → Ink); write `data-theme`; persist
  on change. `index.html` ships `data-theme="paper"` so the app paints before JS runs.
- **No theme-authoring UI and no server-side preference** — three fixed themes, one
  picker, preference in `localStorage`, consistent with Mneme's single-user local design.
- Mermaid re-themes live: `mermaid.ts` re-initialises from the active theme's tokens
  (`darkMode` derived from a `--mermaid-dark` token) and re-renders on switch. It is driven
  from `MDiagram`'s render effect rather than app startup, so the large mermaid chunk stays
  lazy-loaded on diagram-less routes.

### One persistent nav rail

A single `AppShell` layout wraps `<RouterView>` and renders a left rail identical on every
route:

> **Primary wayfinding lives in one place that never moves; page chrome is stripped.**

- The rail holds the brand, project chip, a **global** search (Enter → `/search`, `/`
  focuses it), the eight primary destinations as `<RouterLink>`s with route-driven active
  state (`router-link-exact-active`), and the `ThemePicker`. It is sticky
  (`position: sticky; top: 0; height: 100vh`) so it — and the picker at its foot — stay
  pinned while content scrolls.
- The ten per-page `<header class="topbar">` menus are removed and `Topbar.vue` is deleted.
- **Registry filter vs. global search are split by purpose:** the rail owns the one global
  search; the registry's live filter (debounced query → URL → API) is a separate in-content
  searchbar. This removes the previous duplicate search box and duplicate `/` listener.
- **Document pages** become three columns — rail | contextual doc rail | content — where the
  doc rail is a **connected phase spine** (done / current / todo nodes threaded by a
  connector) plus a section table-of-contents, and the sticky doc sidebar mirrors the rail's
  behaviour. The document's status pill moved from its old topbar into `MetaHeader`; the back
  button is gone (the rail owns wayfinding, the browser owns Back).

## Alternatives considered

- **Keep / tune the dark theme, or keep it as a fourth theme** — rejected; the direction was
  the problem, so the dark-only palette is removed, and Ink covers night reading.
- **Light themes only** — rejected; a good dark reading option was still wanted (→ Ink).
- **A theme-authoring UI with custom palettes** — rejected; overkill for a single-user tool.
- **A persistent top tab-bar instead of a rail** — rejected; 8+ destinations crowd a top bar,
  and a vertical rail matches the knowledge-base mental model.
- **Re-theme the per-page menus without unifying them** — rejected; leaves the shifting-nav
  problem unsolved.
- **Keep `Topbar` trimmed to a search-only component** — rejected; a misnamed, still-redundant
  chrome bar once nav + global search live in the rail.
- **Drop the registry filter and rely on the rail's global search** — rejected; loses live
  in-registry filtering, a real feature.
- **Fold the registry search into `FilterToolbar`** — rejected; muddies that component's clean
  controlled contract with a debounce and an extra concern.
- **Call mermaid `applyTheme()` at app startup** — rejected; it would eager-import the largest
  chunk on every page load, defeating the lazy split.

## Consequences

- **Token-pure is now load-bearing.** Every color flows through a token; a component that
  hardcodes a hex renders unstyled in some theme. The build's grep check and review enforce it.
- **Nav coverage consolidated.** Deleting the per-page topbars and `Topbar.vue` moved the
  `data-test="to-*"` link assertions, active-state, and search/`/`-shortcut coverage into
  `AppShell.test.ts` — a superset of what was removed, so nothing was lost.
- **The rail is the app frame.** Any new top-level destination is one entry in `AppShell`'s
  `NAV` list; pages no longer carry navigation and stay content-only.
- **Local-only, as designed.** Theme preference is `localStorage`; there is no server round-trip,
  consistent with the repo-vs-Mneme delineation (the UI stays a read-mostly viewer).

## Provenance

Graduated from the Mneme plan `ui-themes-and-nav` (5 phases, complete) and its decision log:
"Three reading-first themes; abolish the dark-only palette", "Persistent nav rail replaces
per-page top menus", "Registry keeps a purpose-built filter searchbar; the rail owns global
search", and "Drive mermaid applyTheme() from MDiagram, not app startup". Visual reference:
[`design/theme-explorations.html`](../../design/theme-explorations.html).
