# Fairy Context Pack implementation, 2026-11-06

Part of the complete implementation queue. All three existing queries and all
three existing map/pack families are required; a parser or overview prerequisite
does not finish atlas-pack or the broader objective. Native Linux stays batched;
no push, dispatch, publication or target promotion is authorized here.

## Required sequence

- [x] Exact overview/show/around arguments, versions, direction/relation filters,
  canonical numeric bounds and locked selection, compared with actual Go parsers.
- [ ] Current-admitted overview projection/selection and canonical v1/v2/v3 packs,
  including independent record/byte/final-byte/work boundaries and checksums.
- [ ] Current-admitted show/around node/relation facts, specialization/cleanup
  identity, deterministic local traversal and explicit omissions under all bounds.
- [ ] Fresh Complete-bound source selection and actual dispatcher/help/error/
  lock/output acceptance at both Windows profiles. No child Go fallback.
- [ ] Original full-CLI/generated-consumer/self-build/distribution/hosted/Linux
  acceptance gates and the remaining implementation queue.

Owners: `docs/atlas/context-pack-v1.md`, `context-pack-v2.md`,
`context-pack-v3.md`, `fymap-query-v1.md`, `fymap-query-v2.md` and
`fymap-query-v3.md`. Go owners: `internal/cli/atlas_pack.go`,
`atlas_pack_relations.go`, `internal/atlasmap/query.go` and `internal/contextpack/`.
The actual query is `around`, not `walk`. Versions 1/2/3 require exact-current
map families 1/2/3; no conversion or relabeling is admitted.

## Exact argument selection

`project_pack_options` parses process arguments only, before workspace, lock,
map/cache, network or stdin I/O. All actions select an exact package UID and
positive record/selection/output limits. Show/around also require an original
visible/full source SymbolId, exact direction, one or more distinct relation
kinds and a positive relation limit. Only around accepts/requires max-depth,
canonical 0..64. Overview rejects every relation-only flag. Versions default to
1 and explicit 1/2/3 occur at most once; locked also occurs at most once.

Record/relation limits are 1..65,536 and selection/output limits are
1..1,073,741,824. UID is at most 256 bytes in manifest form. Relations preserve
the Go lexical ten-kind ordering through an owned bit set. The maximum around
invocation includes all ten distinct relation flags, all scalar flags and locked,
43 words including executable and command words; it is accepted without truncation.

`TestCandidatePackOptions` compares 317 actual Go parser cases and three unknown
query refusals per Windows profile (640 native observations total), including
all actions, versions, directions, relation kinds, flag permutations, repeated/
missing/unknown flags, canonical decimals and overflow, zero/max/one-short/
one-more boundaries, UID/Symbol lengths/case and the full maximum invocation.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidatePackOptions$'
```

PASS, 33.583s (body 33.47s; low 19.26s, optimized 13.35s), no selected skips.
Log: `.cache/pack-options-20261106.log`. Initial construction failed once on a
Go oracle uint64/uint32 assignment, then on inferred Fairy action type and on
returning unrelated static string origins. Explicit Go/Fairy numeric types and
one static action slice fix those harness/source issues without policy changes.

This is argument selection only, not current-source query or actual CLI
acceptance. The rest of the sequence and the full objective remain open.
The parser slice's authored check/locked pass with 656 source files and 8,207 MIR
functions. Scoped CLI vet, Go formatting, LF/EOF and diff checks pass.
No grammar, stable CLI, SDK ABI, dependency, DB, workflow/runner or hygiene-policy
change; native Linux/full CLI/full Go/self-build/distribution/hosted execution
is not established by this isolated leaf. No push or dispatch.
