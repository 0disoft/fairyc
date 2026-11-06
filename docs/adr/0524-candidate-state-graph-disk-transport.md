# ADR 0524: Transport graph envelopes through the existing raw disk store

- Status: Accepted for private candidate cache transport
- Date: 2026-11-07

The existing `project_cache_disk` byte store additionally recognizes `FYSGCAC0`.
Its `FYCACHE0`/`FYCACHE1` handling, required signatures, limits, generation and
staging names, no-replace publication, metadata-conditional repair and retention
are unchanged. A recognized prefix and a matching content address confer only
transport validity; the owning family's semantic restoration remains mandatory.

This reuses the actual rooted Windows filesystem adapter rather than another
graph-specific file implementation. The graph route/Complete admission adapter
remains a separate prerequisite. Linux execution stays batched. Windows tests
and exclusions: `docs/development/candidate-state-graph-disk-transport-20261107.md`.
