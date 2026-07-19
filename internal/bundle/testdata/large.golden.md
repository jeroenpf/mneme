# Context bundle — mneme

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
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch A
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch B
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch C
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch D
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch E
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch F
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch G
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch H
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch I
- backfill embeddings for legacy documents pushed before the chunker rewrite, batch J

## Recent decisions
- **Reciprocal-rank fusion** — accepted (2026-07-19)
  RRF blends lexical and semantic ranks without tuning per-type score scales.
- **Recursive AST chunker** — accepted (2026-07-19)
  Walking every block type indexes tasks and tables the section chunker missed.
- **Decision A** — accepted (2026-07-19)
  A deliberately verbose rationale that spells out the trade-offs weighed, the alternatives rejected, and the follow-on consequences in more than enough detail to consume a…
- **Decision B** — accepted (2026-07-19)
  A deliberately verbose rationale that spells out the trade-offs weighed, the alternatives rejected, and the follow-on consequences in more than enough detail to consume a…
- **Decision C** — accepted (2026-07-19)
  A deliberately verbose rationale that spells out the trade-offs weighed, the alternatives rejected, and the follow-on consequences in more than enough detail to consume a…

## Snippets
- **Pattern A** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…
- **Pattern B** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…
- **Pattern C** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…
- **Pattern D** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…
- **Pattern E** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…
- **Pattern F** [go] — A thoroughly documented reusable pattern with plenty of surrounding prose describing when to reach for it, the edge…

## Recent journal
- sp-2: Landed the recursive chunker and stable chunk ids (2026-07-19)
- sp-1: Spiked pgvector index tuning (2026-07-19)
- sp-x: A detailed session summary covering everything built, everything consciously deferred, and every behavioural change shipped across a long and productive day of focused implementation work on the project. #A (2026-07-19)
