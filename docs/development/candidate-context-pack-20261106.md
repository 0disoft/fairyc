# Fairy Context Pack implementation, 2026-11-06

Part of the complete implementation queue. All three existing queries and all
three existing map/pack families are required; a parser or overview prerequisite
does not finish atlas-pack or the broader objective. Native Linux stays batched;
no push, dispatch, publication or target promotion is authorized here.

## Required sequence

- [x] Exact overview/show/around arguments, versions, direction/relation filters,
  canonical numeric bounds and locked selection, compared with actual Go parsers.
- [x] Current-admitted overview projection/selection and canonical v1/v2/v3 packs,
  including independent record/byte/final-byte/work boundaries and checksums.
- [x] Current-admitted show node/relation selection, specialization/cleanup
  identity, canonical v1/v2/v3 packs and explicit omissions under all bounds.
- [ ] Current-admitted around traversal, deterministic local node/relation
  selection, canonical packs and explicit omissions under all bounds.
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

## Go overview header transport prerequisite

Independent pack comparison reproduced a pre-existing Go distinction: source
partitions can retain nonnil empty feature/dependency slices, while map decode
reconstructs nil empties. `cloneQueryHeader` previously preserved that allocation
detail, changing overview JSON, selection costs/cutoffs and derived pack checksums
after an unchanged map round trip. This is not an identity/source change.

`internal/atlasmap/query_header_test.go` first reproduced the changed overview
against an actual checked source/map round trip; the original command failed at
`.cache/pack-header-before-20261106.log`. Header projection now detaches and
normalizes the four header array fields to `[]`, matching the existing empty-array
contract. Original/restored mandatory record limits at one-short/exact/one-more
and nonempty clone immutability are pinned separately.

The full `internal/atlasmap` and `internal/contextpack` package suites pass in
2.077s/0.202s (`.cache/pack-header-after-20261106.log`). Final focused header
controls pass in 0.242s. Existing experimental Go atlas-pack command regressions
pass in 6.631s; scoped atlasmap/contextpack/CLI vet, formatting/LF/diff checks pass.
The owning query and Context Pack v1 documents now explicitly state normalization;
v2/v3 retain the same overview projection. This corrects experimental overview
bytes/checksums and selection costs for empty header arrays; it changes no map
bytes, original partition, language/SDK ABI, DB, dependency or runner policy.
It is a Go prerequisite, not Fairy pack or actual CLI acceptance. No push,
dispatch or native Linux execution; the full objective remains open.

## Current-admitted overview packs

`project_pack_projection` explicitly maps checked wire fields to owning Go query
records; wire JSON is not a snapshot/query JSON record. `project_pack_overview`
selects the mandatory package, ordered completeness and modules with separate
record/selection-byte bounds, retaining record-limit precedence and exact known
omissions. `project_pack` serializes one body, independently domain-hashes it and
checks the complete envelope length before final copying. Only exact-current
v1/v2/v3 maps enter their matching pack family. This leaf performs no source,
filesystem, process or network discovery and emits empty text channels.

`TestCandidatePackOverview` uses the actual ordinary, scalar-generic and aggregate
source/map fixtures, both package partitions and all three owning Go encoders.
Its checksum oracle separately hashes the envelope's exact logical bytes.
All cumulative record costs are checked one-short/exact/one-more, including the
mandatory package and competing record/byte stops; zero/full-output and exact
global work limits, corrupt map, stale currentness and wrong family all refuse
atomically. Both Windows profiles pass 488 native observations each, 976 total:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 8m -v ./internal/cli -run '^TestCandidatePackOverview$'
```

PASS, 86.667s (body 86.48s; low 49.99s, optimized 35.04s), no selected skips;
`.cache/pack-overview-normalized-20261106.log`. Earlier attempts retained in
`pack-overview`, `pack-overview-retry`, `pack-overview-arrays` and
`pack-overview-omissions` logs are failures, not acceptance. They exposed an
unsupported harness cast, snapshot/wire empty-array and omission differences,
and the separately fixed Go header transport defect above. Final source/test
inputs were unchanged after the passing native replay.

Authored `check`/`check --locked` pass with 659 source files and 8,227 MIR
functions (`pack-overview-check`/`pack-overview-locked` logs). Scoped
atlasmap/contextpack/CLI vet, Go formatting, LF/EOF and diff checks pass.
This closes the overview encoder leaf only. Show/around, fresh source-backed
actual dispatch, generated consumers and every original final acceptance gate
remain open. No grammar, stable CLI, public SDK/runtime ABI, dependencies, DB,
workflow/runner or hygiene-policy change; no push or native Linux execution.

## Current-admitted show packs

`project_pack_graph` binds the already verified relation observations to source
and optional local-target SymbolId positions and builds ascending adjacency
lists. `project_pack_frontier` lazily merges those lists with a bounded owned
min-heap, consuming a self-call once even when both directions are selected.
No repeated whole-relation scan or workspace/dependency discovery supplies
query facts. `project_pack_context` projects only requested completeness plus
the existing version-specific generic uncertainty.

`project_pack_query` reserves the exact context/subject as one mandatory record,
selects ordered incoming/outgoing/both observations, and preserves relation-
before-record-before-byte refusal and exact known remaining counts. Relation
filters, unjoined selected targets and missing incoming packages remain explicit
omissions. The public private-host `encode_show` entry runs that selection itself
and uses the existing version-specific canonical body/checksum/envelope path;
it never accepts a caller-mutated query result or a map-version conversion.

`TestCandidatePackShow` uses the fifteen actual source/package/schema fixtures
from the all-kind relation corpus. All nodes compare against each actual Go
show encoder in all three directions and both full/visible selector forms.
Each kind, combined filters, generic/cleanup facts, mandatory and cumulative
selection bytes, every selected record/relation boundary and their precedence,
final output length and global work one-short/exact/one-more pass. Malformed/
missing selectors, stale/corrupt maps, wrong versions and invalid policies
publish nothing. The checksum is independently recomputed from logical bytes.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 12m -v ./internal/cli -run '^TestCandidatePackShow$'
```

PASS, 156.692s (body 156.49s; low 102.18s, optimized 51.85s), 1,210 native
observations per profile, 2,420 total, no selected skips;
`pack-show-frontier-20261106.log`. Earlier `pack-show` and `pack-show-names`
attempts failed on an import/function name collision and mutable borrowing of
a nested heap resource respectively. The final merge owns the heap directly
and keeps the deduplication cursor separately; no query policy was relaxed.

Authored check/locked pass with 667 source files and 8,294 MIR functions;
scoped CLI vet, Go formatting, LF/EOF and diff checks pass. This closes the
show encoder only. Around BFS/packs, source-backed actual dispatch, generated
consumers and every original implementation/final gate remain open. No grammar,
stable CLI, SDK/runtime ABI, DB, dependency, workflow/runner or hygiene-policy
change; no push, dispatch or native Linux execution.

## Go node transport prerequisite

Actual Fairy node comparison then reproduced the same allocation drift inside
Go query declarations: source-prepared empty type arguments, parameters, fields
and variants encode as `[]`, while unchanged map restoration encodes `null`.
`TestQueryNodesSurviveMapTransport` independently reproduces this for actual
ordinary v1, generic v2 and aggregate/cleanup v3 compiler partitions; all three
fail before the fix (`pack-node-go-before-20261106.log`). Both show/around facts
and mandatory byte boundaries are affected; this is not a source/map identity
change. Query copies now normalize nested declaration/type slices to the existing
map-restored nil-empty representation. Snapshot transport and caller data remain
unchanged. Concrete type arguments and aggregate fields follow the same rule.

Full atlasmap/contextpack suites pass in 3.291s/0.190s
(`pack-node-go-after-20261106.log`); existing experimental Go pack CLI regressions
pass in 2.503s (`pack-node-go-cli-20261106.log`). Scoped vet, Go formatting,
LF/EOF and diff checks pass. Owning query/pack v1 documents now state this
normalization explicitly; v2/v3 retain the same detached projection. This
corrects experimental query/pack bytes, checksums and byte limits for source-
prepared empty nested arrays. It does not implement Fairy show/around or grant
actual CLI, Linux, full Go-suite, self-build, distribution or hosted acceptance.
No grammar, SDK ABI, DB, dependency, workflow/runner or hygiene-policy change;
no push or dispatch. The complete implementation objective remains active.

## Current-admitted query nodes

`project_pack_snapshot` projects keys/spans, recursive types, parameters,
fields, enum payload variants and complete original/concrete declarations.
`project_pack_node` keeps original binders separate from concrete type arguments,
aggregate fields and function/cleanup relation counts. All ten relation kinds
have distinct count slots; snapshots use the canonical map-restored nil empties
described above. Neither module loads source or accepts a caller-built node as
compiler truth; node selection requires an exact-current Ready map.

`TestCandidatePackNode` compares every node against the actual independent Go
query API, in full SymbolId order. Twelve package/schema/source combinations
cover the existing ordinary/generic/aggregate corpus plus repr(C) fixed-array
fields, payload enums, shared owner methods, exclusive/view/array parameters,
nonempty cleanup and two generic aggregate/cleanup specializations with real
helper calls and borrows. Per Windows profile, 65 node records compare exactly
and 72 native processes exercise all-node bytes, out-of-range atomic rejection
and one-short/exact/one-more cumulative work boundaries. Both profiles pass:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 8m -v ./internal/cli -run '^TestCandidatePackNode$'
```

PASS, 25.382s (body 25.26s; low 10.62s, optimized 13.01s), 144 native processes
and 130 node comparisons, no selected skips; `pack-node-normalized-20261106.log`.
Earlier retained logs (`pack-node`, `pack-node-retry`, `pack-node-native`,
`pack-node-names`, `pack-node-view`, `pack-node-counts`, `pack-node-typed`) record
failed harness API names, a reserved local name, an inferred i32 index and the
independently fixed Go nested-array drift. They are not passing evidence.
Authored check/locked pass with 661 files and 8,237 MIR functions; scoped vet,
Go formatting, LF/EOF and diff checks pass. This is query-node projection only:
relations, show/around selection/traversal and full pack/actual dispatch remain
required, as do all original implementation/final gates. No grammar, stable CLI,
SDK ABI, DB, dependency, workflow/runner or hygiene-policy change; no push or
native Linux execution.

## Current-admitted relation projection and ordering

`project_pack_relations` expands the ten observed relation kinds from ordinary
functions, concrete functions and concrete aggregate cleanup bodies. Source/
target specialization, cleanup ownership, local/unjoined target identities,
optional spans, trap detail, certainty, C ABI/link name and escaping-borrow facts
retain their owning Go query fields and order. The total expansion must equal
the checked header count and is capped at 65,536 observations. It grants no new
source-family or dependency-map admission and performs no host I/O.

`project_pack_relation_order` uses stable bottom-up mergesort over owned indices:
O(R log R) comparisons and O(R) scratch, preserving equal observations. Ordering
matches the complete Go relation tuple, including numeric span offsets and both
implementation endpoints; raw JSON lexical order is not an ordering oracle.

`TestCandidatePackRelations` uses the node corpus plus three actual-source root
maps with Region placement, both borrow kinds, escaped field borrow, foreign
calls, self-calls, trap observations, resource moves/drops and `?` propagation.
The independent Go outgoing queries cover every admitted kind and explicitly
require seven optional fact families (source/target specialization, cleanup,
unjoined target, foreign link, escaping borrow and spanless detail). Reverse
input has the same result; an added equal observation stays duplicated. Global
work limits are independently checked one-short/exact/one-more after current
admission, with atomic failure output.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 8m -v ./internal/cli -run '^TestCandidatePackRelations$'
```

The final expanded assertion set passes Windows low/optimized in 28.754s (body
28.64s; low 11.69s, optimized 15.24s): 105 native processes and 65 independent
relation comparisons per profile, 210/130 total, no selected skips.
Log: `pack-relations-final-20261106.log`. The preceding all-kind byte replay
passed in 31.522s before adding the seven explicit coverage assertions; it is
not substituted for final-input acceptance. Earlier `pack-relations` and
`pack-relations-place` attempts fail on a non-place borrow fixture and missing
actual `?` propagation respectively; those are not passes.

Authored check/locked pass with 663 files and 8,252 MIR functions; scoped CLI
vet, Go formatting, LF/EOF and diff checks pass. This closes relation projection
and ordering, not show/around bounded selection, breadth-first traversal, pack
assembly or actual CLI. Those and all original implementation/final acceptance
gates remain open. No grammar, stable CLI, SDK ABI, DB, dependency,
workflow/runner or hygiene-policy change; no push or native Linux execution.
