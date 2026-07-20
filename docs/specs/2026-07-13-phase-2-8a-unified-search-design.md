---
date: 2026-07-13
phase: 2.8a
status: approved-pending-review
authors: jeroenpfeil, claude
supersedes: none
---

# Unified FTS search — design

Design spec for the **FTS-first half** of Phase 2.8 (Unified search &
embeddings). Ships one `search` surface spanning documents, decisions,
snippets, solutions, and journal — ranked by full-text relevance — as an
MCP tool, a REST endpoint, and a Vue page. The embedding/vector half
(Voyage client, async worker, hybrid cosine ranking) is a **separate
follow-on session, 2.8b**; this spec is deliberately shaped so that work
slots in without reshaping the query.

Source plan: the Mneme implementation plan (in Mneme), Phase 2, sub-phase `#sp-2-8`.

## Goals & non-goals

**Goals**
- One unified `search(q, types?, project?, limit?)` over five content types, returning a common ranked hit shape (`type, id, title, excerpt, project, score`).
- A single SQL `UNION ALL` query with coherent cross-type ranking and DB-side limiting — one round-trip, forward-compatible with a vector list.
- Add full-text search to journal entries (the one covered type that lacks it today).
- Expose it identically through MCP (`search`), REST (`GET /api/v1/search`), and a Vue `/search` page — the same read-mostly posture as the rest of Phase 2.
- Keep `search_documents` working as a thin alias so nothing that calls it breaks.

**Non-goals (deferred to 2.8b — the embeddings session)**
- `internal/embed` (Voyage AI `voyage-code-2` client) and the section-chunk extractor.
- Async embed-on-write worker (Go channel + goroutine), backfill of existing rows.
- The vector/cosine half of the query and true hybrid RRF fusion (BM25 **+** vector).
- `GET /api/v1/search/status` embedding-coverage endpoint and the Vue coverage indicator — there is no embedding coverage to report in FTS-only mode.
- **Memory** as a searchable type. Memory is small key/value context already delivered wholesale by `get_memory` / `get_context_bundle`; searching it adds little. It joins search only if a concrete need appears.

## Content coverage

| Type | Has `search_vector` today | Action |
|---|---|---|
| documents | yes | reuse |
| decisions | yes | reuse |
| snippets | yes | reuse |
| solutions | yes | reuse |
| journal | **no** | add `search_vector` (migration 012) |
| memory | no | excluded (see non-goals) |

## Data layer

### Migration `012_journal_fts`
Add a `search_vector` column to `journal_entries`, mirroring the exact
mechanism the other four tables already use (generated column **or**
trigger — match whichever `008_decisions` / `011_solutions` use, do not
invent a third pattern), plus its GIN index. Weighting:
`summary` = A, `session_ref` = B, `accomplished` + `deferred` (array text
joined) = C. Backfill existing rows in the same migration. A matching
`012_journal_fts.down.sql` drops the index and column.

### `models.SearchHit`
```go
type SearchHit struct {
    Type      string    `json:"type"`      // documents|decisions|snippets|solutions|journal
    ID        string    `json:"id"`
    Title     string    `json:"title"`     // per-type label (see mapping)
    Excerpt   string    `json:"excerpt"`   // ts_headline snippet
    Project   *string   `json:"project,omitempty"`
    Score     float64   `json:"score"`
    UpdatedAt time.Time `json:"updated_at"`
}
```
Per-type `Title` mapping (types without a `title` column): solutions →
`error_description`, journal → `summary`. documents/decisions/snippets use
their `title`.

### `store.Search`
```go
type SearchFilter struct {
    Types   []string // empty => all five
    Project *string
    Limit   int      // 0 => sane default (e.g. 20)
}
func (s *PostgresStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error)
```
Implementation: one query that `UNION ALL`s a projected `SELECT` per
requested type. Each branch:
- matches with `search_vector @@ websearch_to_tsquery('english', $q)` (same google-style syntax already used by `SearchDocuments`),
- scores with `ts_rank(search_vector, websearch_to_tsquery(...))`,
- builds `Excerpt` with `ts_headline` over the branch's primary text,
- projects `type` as a literal, plus id/title/project/updated_at.

Only requested type-branches are emitted (built dynamically, parameterised
— never string-interpolated user input). The optional project filter is
AND-ed into each branch. The outer query orders by the fused score
(below) then `updated_at DESC`, and applies `LIMIT`.

### Cross-type ranking
Raw `ts_rank` scales differ between tables, so ranking them in one list
needs normalisation. Use the **reciprocal-rank** term: within each type
branch compute `row_number() OVER (ORDER BY ts_rank DESC)` = `rank`, then
`score = 1.0 / (k + rank)` (k ≈ 60, the conventional RRF constant). The
outer query sorts by this `score`. This is exactly the RRF contribution of
a single ranked list — so 2.8b adds the vector list as a second
`1/(k + rank_vec)` term per row and the shape becomes true hybrid fusion
with no rewrite.

## MCP surface
- **`search`** — args `q` (required), `types?` (default all five), `project?`, `limit?` (default 10). Returns `{ results: []SearchHit }`. Unknown `type` value → friendly error. Errors translated via the existing `translateStoreErr`.
- **`search_documents`** — kept as a thin alias that calls `search` with `types=["documents"]`, preserving its current output contract for existing callers. (Per roadmap: "Replaces `search_documents` — keep old tool as alias.")

## REST surface
- **`GET /api/v1/search?q=&types=&project=&limit=`** → `{ items: []SearchHit }` (200). `q` required → 400 when absent (via `writeError`). `types` is a comma-separated list; unknown type → 400. Read-only GET; CORS untouched. Same `writeJSON` / `writeStoreError` adapters as every other Phase 2 handler.

## Vue surface
- **`/search` page** — results grouped by type; each hit links to its document (`/doc/:id`) or the relevant entity page. No embedding-coverage indicator this session.
- **Topbar** — the existing search input keeps its instant registry-filter behaviour on the registry page; pressing **Enter** navigates to `/search?q=…` for global search. The topbar search is **not** replaced. A `search →` affordance is unnecessary since the input itself is the entry point.

## Testing
TDD throughout, one commit per task, mirroring the 2.6 rhythm.
- **Store** (testcontainers): a hit from each of the five types; `types` filter narrows correctly; `project` filter; ranking order across types; journal FTS matches on summary/accomplished; empty query and no-match behaviour.
- **MCP**: `search` round-trip returns ranked hits; `types`/`project`/`limit` honoured; `search_documents` alias still returns documents.
- **API**: `/search` 200 with results; missing `q` → 400; unknown type → 400; `types`/`project`/`limit` params.
- **Vue**: mocked-api render, grouping by type, navigation on click; topbar Enter routes to `/search`.

## Sequencing (bottom-up, for the plan)
1. Migration `012_journal_fts` + store `Search` (`SearchFilter`, `SearchHit`, the UNION-ALL query, ranking) — TDD at the store layer.
2. MCP `search` tool + `search_documents` alias.
3. REST `GET /api/v1/search`.
4. Vue `/search` page + topbar Enter→`/search`.
5. Smoke + bookkeeping (mark part of `#sp-2-8`; the sub-phase completes with 2.8b).

## Decisions log (resolved during brainstorming)
- **Provider deferred, not dropped.** Embeddings use Voyage AI in 2.8b; this session is FTS-only with the query shaped for a clean vector add.
- **Split into 2.8a (FTS) + 2.8b (vectors)** to ship a working unified search sooner and de-risk the external API dependency.
- **Coverage = 5 types + journal FTS, memory excluded.**
- **Single SQL `UNION ALL`** (not Go-side fan-out) for coherent DB-side ranking, one round-trip, and vector-forward-compatibility.
- **Topbar search kept**; Enter navigates to `/search`, registry instant-filter preserved.
