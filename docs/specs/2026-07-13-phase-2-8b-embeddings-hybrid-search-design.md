---
date: 2026-07-13
phase: 2.8b
status: approved-pending-review
authors: jeroenpfeil, claude
supersedes: none
---

# Embeddings & hybrid search — design

Design spec for the **embedding/vector half** of Phase 2.8, the follow-on
to 2.8a (unified FTS search). It adds semantic recall on top of the
existing `search` surface: a Voyage embedding client (`voyage-4-large`,
1024-dim), an async embed-on-write worker with startup reconciliation, and
a hybrid BM25 + vector ranking folded into the single `store.Search` SQL
that 2.8a deliberately shaped for exactly this addition. Completing 2.8b
closes `#sp-2-8` and finishes Phase 2.

Source plan: the Mneme implementation plan (in Mneme), Phase 2, sub-phase `#sp-2-8` (the four remaining task rows + the Vue coverage indicator).
Predecessor: [`2026-07-13-phase-2-8a-unified-search-design.md`](2026-07-13-phase-2-8a-unified-search-design.md) — 2.8a shipped the FTS-only superset; this spec slots the vector list into its RRF query with no rewrite.

## Goals & non-goals

**Goals**
- A Voyage embedding client (`voyage-4-large`, 1024-dim) behind an interface, with the model name configurable via env. The request pins `output_dimension: 1024` so the returned vectors always match the schema, whatever the model's default.
- Section-level chunking for documents (single-chunk for the short entity types), embedded asynchronously on write.
- An in-memory embed worker + self-healing startup reconciliation (no durable queue, no re-embedding of unchanged content).
- Hybrid ranking: fuse the existing per-source FTS RRF term with a per-source vector RRF term inside one SQL query, degrading byte-for-byte to 2.8a FTS-only when embeddings are unavailable.
- `GET /api/v1/search/status` embedding-coverage endpoint + a small coverage line on the Vue `/search` page.

**Non-goals**
- No durable job/outbox table (personal single-user scale; the in-memory worker + startup reconcile is self-healing).
- No hash column for staleness — reconciliation diffs `chunk_text` in place.
- **Memory** stays excluded from search (unchanged from 2.8a).
- No re-ranking, query expansion, or multi-vector retrieval. Single query embedding, single cosine list.

## Degradation posture (the load-bearing decision)

Embedding is gated on `MNEME_VOYAGE_API_KEY`. **Key present** → embeddings on, `search` runs hybrid. **Key absent** (CI, fresh dev, tests) → the embed client is nil, the worker no-ops, startup reconciliation is skipped, and `search` is exactly the 2.8a FTS-only path. No startup failure, no network calls, no errors surfaced to callers. `make test` stays hermetic with **no** key and **no** Voyage traffic. Additionally, a transient Voyage failure at query time logs and falls back to FTS-only — `search` never hard-fails because embeddings are down.

## Data layer

**Migration `013_embeddings_dim`.** The `embeddings` table (migration 005) is
`vector(1536)` / `model DEFAULT 'voyage-code-2'`; `voyage-4-large` is 1024-dim.
The table is empty until 2.8b runs anywhere, so the migration drops the ivfflat
index, recreates the `embedding` column as `vector(1024)`, sets the model default
to `'voyage-4-large'`, and rebuilds the index — no backfill, no data loss. The
`.down.sql` reverses to `vector(1536)` / `'voyage-code-2'`. Everything else about
the table (the `(source_type, source_id, chunk_id)` upsert key, `source_title`/
`chunk_text`/`project`/`created_at` columns) is unchanged and reconciliation diffs
`chunk_text` in place, so no hash column is needed.

New `Store` methods over the `embeddings` table:

```go
// Batch upsert by (source_type, source_id, chunk_id).
UpsertEmbeddings(ctx, rows []models.Embedding) error
// Prune section-chunks that no longer exist after a re-embed.
DeleteEmbeddingsExcept(ctx, sourceType, sourceID string, keepChunkIDs []string) error
// chunk_id -> chunk_text for a source; reconciliation diffs this against
// freshly-extracted chunks to skip unchanged ones (no API call).
EmbeddingsFor(ctx, sourceType, sourceID string) (map[string]string, error)
// All embeddable sources, for startup reconciliation.
SourceRefs(ctx) ([]SourceRef, error)
// {type, embedded, total} per type, for the status endpoint.
EmbeddingCoverage(ctx) ([]TypeCoverage, error)
```

`SourceRef{Type, ID string}` and `TypeCoverage{Type string; Embedded, Total int}` live in `internal/store`. `models.Embedding` already exists (Phase 1.2).

## `internal/embed` — client + chunker

**Client interface** (keeps store/worker/API decoupled from Voyage and hermetically testable):

```go
type Client interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Model() string
}
```

- `voyageClient` — POSTs to Voyage's embeddings endpoint, batches `texts`, `Authorization: Bearer $MNEME_VOYAGE_API_KEY`, model from `MNEME_VOYAGE_MODEL` (default `voyage-4-large`), and pins `output_dimension: 1024` so results match the schema regardless of the model's default. Wire shape verified live: `POST https://api.voyageai.com/v1/embeddings`, body `{"input":[...],"model":...,"input_type":"document"|"query","output_dimension":1024}`, response `{"data":[{"embedding":[...],"index":i}]}`. Batches are size-capped; oversized batches are split.
- A nil `Client` is the "embedding disabled" state, checked explicitly by every caller.

**Rate limiting & batching.** Each source is embedded in **one** request — a document's changed sections go in a single `input` array (≤128 per request; larger sources are split into 128-input batches). So a 10-section document is one Voyage call returning 10 vectors, not ten calls. The client retries on `429`/`5xx` with exponential backoff (honouring `Retry-After`), so a low rate-limit tier just runs slower rather than erroring or dropping jobs — the worker is single-goroutine, so slow embeds only lengthen the queue, never block a write. An optional `MNEME_VOYAGE_RPM` (default 0 = off) adds a proactive inter-job delay in the worker for accounts on the ~3 RPM no-payment-method tier; adding a payment method to the Voyage account raises the limit to 2000 RPM (Tier 1, no spend required) and makes the knob unnecessary. The 200M free-token allowance is unaffected by tier.

**Chunker** — a pure function, no I/O:

```go
type Chunk struct { ID, Text string }
func Chunks(sourceType string, src any) []Chunk
```

- **documents** → one chunk per section, walking `body.sections` (same recursion as `internal/mcp/blocks.go`'s `walkBlocks`); `ID` = section id, `Text` = `"{doc title} | {project} | {section title}: {content + task titles}"`.
- **decisions / snippets / solutions / journal** → a single chunk (`ID` = `"full"`), `Text` = the entity's salient fields joined (mirrors what each type's FTS vector already weights).

Because it's pure, the chunker is unit-tested directly against fixture sources.

## Async worker + wiring

**`embed.Worker`** — a buffered `chan SourceRef` + one goroutine bound to the server's signal context:

- Per job: load the source → `Chunks(...)` → `EmbeddingsFor(...)` diff (embed only new/changed `chunk_text`) → `Client.Embed(changed)` → `UpsertEmbeddings` → `DeleteEmbeddingsExcept(keep = current chunk ids)`.
- Errors are logged, never fatal; the next reconcile self-heals.
- Drains and exits cleanly on shutdown (ctx cancel).

**Enqueue hook** — after a successful write, the MCP write tools (`push_document`, the section/task edit tools, `log_decision`, `save_snippet`, `log_solution`, `append_journal`) call `enqueuer.Enqueue(SourceRef{...})`. A single small helper; when embedding is disabled the enqueuer is a no-op so the tools are agnostic to whether Voyage is configured.

**`main.go run()`** — build the client from env; if enabled, start the `Worker` goroutine and run startup reconciliation (enqueue every `SourceRef`; the diff step makes an all-warm DB cheap — no API calls when nothing changed). Pass the enqueuer into `mcp.New`. When disabled, none of this starts.

## Hybrid query (extend `store.Search`)

`Search` gains an optional query vector (`[]float32`, nil = FTS-only). When present, the SQL adds a `vhits` CTE alongside the existing `hits`:

- `vhits` selects the **best chunk per source**: `DISTINCT ON (source_type, source_id) source_id, 1 - (embedding <=> $qvec) AS sim ... ORDER BY source_type, source_id, embedding <=> $qvec`, then `row_number() OVER (ORDER BY sim DESC)` → `vec_term = 1.0/(k + rank_vec)` (k = 60, same constant as the FTS term).
- Final `score = coalesce(fts_term,0) + coalesce(vec_term,0)`, full-outer-joined per source so a source found by only one modality still ranks. Outer ordering `score DESC, updated_at DESC` is unchanged.
- **Fallback:** nil vector → the `vhits` CTE and its term are omitted; the query is 2.8a byte-for-byte.
- **Excerpt for a vector-only hit** (no FTS `ts_headline` match): use the best-matching chunk's `chunk_text` (truncated), so semantic-only hits still show a meaningful snippet. FTS-matched hits keep their `ts_headline` excerpt.

**Query embedding:** the `search` MCP tool and REST `/search` embed the query string once (one `Client.Embed`) when a client exists, then pass the vector to `store.Search`; on embed error they log and pass nil (FTS-only). No query-vector caching in v1.

## Surfaces

- **MCP `search`** — unchanged signature/output; internally embeds the query when enabled. Hybrid is transparent to the caller (no new arg, no toggle).
- **REST** — `GET /api/v1/search` unchanged externally. New read-only `GET /api/v1/search/status` → `{ items: [{type, embedded, total}], enabled: bool }`. CORS untouched; same `writeJSON`/`writeStoreError` adapters.
- **Vue** — `/search` gains a small coverage line sourced from `/search/status` (e.g. "semantic: 42/50 embedded" when enabled, "FTS-only — no embedding key" when not). Non-blocking; search works regardless. `web/src/api/search.ts` gains a `searchStatus()` call.

## Testing

- `embed.Client` is an interface → a `fakeClient` returning deterministic vectors drives store/worker/API tests. `make test` needs **no** key and makes **no** network calls.
- **Chunker**: pure-function unit tests over fixture documents/entities (section walk, short-entity single chunk, chunk-text format).
- **Store hybrid ranking**: seed embeddings directly with fake vectors; assert FTS-only, vector-only, and both-match fusion ordering, plus the nil-vector 2.8a-identical path.
- **Worker**: enqueue → assert upsert + stale-chunk prune + unchanged-chunk skip, against the fake client.
- **API**: `/search/status` shape + `enabled` flag; `/search` still 200/400 as in 2.8a.
- **Vue**: mocked-api coverage-line render (enabled vs disabled).
- **Live smoke**: one key-gated (env or build-tag) test hitting real Voyage to pin the wire format — skipped by default in `make test`, run manually.

TDD throughout, one commit per task, mirroring the 2.8a rhythm.

## Sequencing (bottom-up, for the plan)

1. `internal/embed` chunker (pure, TDD) + `Client` interface & Voyage impl (wire format).
2. Migration `013_embeddings_dim` (1536→1024) + store methods (`UpsertEmbeddings`, `DeleteEmbeddingsExcept`, `EmbeddingsFor`, `SourceRefs`, `EmbeddingCoverage`) — TDD with fake 1024-dim vectors.
3. `embed.Worker` + enqueuer + `main.go` wiring + startup reconciliation.
4. Hybrid `store.Search` (query vector, `vhits` CTE, RRF fusion, vector-only excerpt) — TDD.
5. Query embedding in the `search` MCP tool + REST `/search`; new `GET /api/v1/search/status`.
6. Vue coverage line + `searchStatus()`.
7. Live smoke + bookkeeping: tick the remaining `#sp-2-8` rows → 7/7, flip the sub-phase pip to done, close Phase 2.8.

## Decisions log (resolved during brainstorming)

- **Degrade to FTS-only without a key** — no boot dependency on Voyage; hermetic tests; query-time Voyage errors fall back, never hard-fail.
- **`voyage-4-large` at 1024 dims** — chosen over the original `voyage-code-2`/1536 after checking the live Voyage lineup: `voyage-4-large` is the current flagship (Jan 2026), best overall for Mneme's mostly-prose-plus-some-code corpus, and **free** (200M tokens/account shared — far beyond a personal tool's spend). It needs a one-off dim migration (013) since every current model is 1024-dim, not 1536. `voyage-code-3` (also 1024-dim, also free) is the code-specialized alternative — a one-word `MNEME_VOYAGE_MODEL` swap, no rework. Requests pin `output_dimension: 1024`.
- **Section-level chunks for documents**, single-chunk for short entities — better recall/section-precise excerpts; the vector list collapses to per-source via `DISTINCT ON` best chunk before RRF fusion.
- **In-memory worker + auto startup reconciliation** — self-healing (covers rows written while the key was absent, and jobs lost to a restart); `chunk_text` diff skips unchanged chunks so reconcile is cheap and doesn't waste API calls. No durable queue.
- **Hybrid fusion inside one SQL query** — the vector term is added to the 2.8a RRF shape with no rewrite; degrades to the identical FTS-only query when the vector is nil.
