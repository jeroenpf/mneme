# Context bundle — mneme / search

## Memory
- **db**: postgres
- **editor**: nvim
- **search**: hybrid fts + vectors

## Env
_none_

## Active plan
**Retrieval quality** — phase 3/5 (in-progress)
Current phase: **Fusion**

## Next tasks
- [ ] normalize cross-type scores — Fusion `t-1`
- [ ] add relevance fixtures — Fusion `t-2`
- [ ] deep-link every hit — Navigation `t-3`

## Blockers
- Local embedding provider spike `provider`

## Deferred (from last session)
- normalize fusion scores
- excerpt from matched chunks

## Recent decisions
- **Reciprocal-rank fusion** — accepted (2026-07-19)
  RRF blends lexical and semantic ranks without tuning per-type score scales.
- **Recursive AST chunker** — accepted (2026-07-19)
  Walking every block type indexes tasks and tables the section chunker missed.

## Snippets
- **cosine similarity** [go] — Brute-force cosine over the embedding slice.
- **fts5 query** [sql] — bm25-ranked FTS5 match with a project filter.

## Recent journal
- sp-2: Landed the recursive chunker and stable chunk ids (2026-07-19)
- sp-1: Spiked pgvector index tuning (2026-07-19)
