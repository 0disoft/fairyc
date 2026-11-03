# Windows Fairy Atlas reader and impact, 2026-11-04

## Transport slice

`project_atlas_wire` reads the existing canonical `.fymap/v1` and `.fymap/v3`
JSON envelopes without a new transport schema. This first private slice rejects
specialization/type-parameter records rather than dropping them. The optional
ordinary v3 generic-interface header pair is retained. It validates exact field
names/order/types, canonical Go JSON spelling, no whitespace/trailing values,
duplicate decoded keys, declared unsigned integer forms, the version pair and
SHA-256 over the domain-separated length-prefixed canonical map document.

The returned `Document` is a **transport-only** view. Semantic validity and
current-source authority require the next reader layer. A checksum is neither
source freshness nor publisher authentication. Input is at most 4 MiB, JSON
structure depth at most 160, arrays at most 65,536 entries, with the existing
shared 16,777,216-unit work meter. Failure returns no usable document.

Executed: `go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlasWire$'`.
Windows MSVC low/optimized pass (8.506 seconds overall). Four real Go-generated
partition envelopes (two source packages, v1 and v3) are admitted. Sixteen
whitespace/trailing/truncation/duplicate/unknown/checksum/version/escape mutations
and a tiny byte-budget rejection execute in each profile. Go supplies fixture
bytes only; the native Fairy process owns decoding, canonical checks and hashing.

Linux execution is deferred. No CLI/MCP/edit dispatch, map production, source
freshness claim, public API/SDK ABI, dependency, DB, repository or runner setting
change, push or hosted CI accompanies this first slice.

## Semantic and current-source gates

`project_atlas_model`, `project_atlas_symbols` and `project_atlas_validate` check
header identity algorithms/nonzero digests, exact namespace references, canonical
file/module/declaration/symbol/function tables, spans, type forms, declared record
counts, call provenance/targets and completeness/omission consistency. SymbolId
signatures and full/visible hashes are rederived in Fairy, including owners,
parameters, effects, fields and variants. Neither a valid checksum nor a matching
stored visible/full pair can bypass declaration identity verification.

The bounded reader admits v1 and nongeneric v3 records. Generic declarations,
concrete specialization records, nominal type arguments and v2 remain rejected.
Rows are capped at 4,096 (call arrays at 65,536), references at 33, type depth at
64 and metadata strings at 4,096 decoded bytes. Comparisons and signature work
charge the same shared meter; no unbounded identifier search is introduced.

`project_atlas_current::read` additionally requires a detached independent current
expectation with compiler, ordinary/generic interface, package/source/namespace,
features/dependencies/references and every file digest/line-map/kind. This private
argument carries the same facts as Go `LoadExpectation`; it is not a new `.fymap`
version or stable file format. The caller must derive it from verified current
retained source, not copy it out of the map being validated. The gate performs no
filesystem scan, lock update or source compilation. A ready map is evidence only
relative to that caller-supplied current expectation.

Executed: `go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlas(Validation|Current)$'`.
MSVC low/optimized pass, 24.485 seconds overall. Checks use real v1/v3 source maps,
record/enum/foreign-call/resource fixtures, Go-rejected rechecksummed metadata,
counts, file identity, span, declaration SymbolId and provenance mutations.
Current expectations are independently serialized from the verified Go compiler
result; changes to ten identity fields, files, dependencies and unknown fields
reject. These are native Fairy readers, without a Go child or native C parser.
Current-source ingestion and graph joining remain separate from these APIs.

## Joined reverse-call impact

`project_atlas_join` consumes only independently current-checked Ready partitions.
It requires one v1/v3 schema, unique package instances, the complete direct-dependency
closure and agreement of reference UID/coordinate pairs. Every resolved call must
resolve to an exact declaration, with public cross-package visibility. Foreign
calls remain opaque and do not consume the resolved-call ceiling. Package/call
ceilings are explicit, at most 65,536, and shared work exhaustion rejects the join.
The immutable graph owns its retained partitions; owned resource fields are never
partially moved or mutated to construct the index.

`project_atlas_impact` canonicalizes visible/full selectors through the join and
performs deterministic breadth-first reverse traversal. Cycles and duplicate call
sites cannot duplicate symbols. The canonical call ledger orders capped selection
independently of partition input order. Depth is capped at 64, returned symbols at
4,096, with depth/symbol cutoffs reflected in partial state and traversalComplete.
Contexts retain current identities, map omissions and relevant call/generic
completeness. The existing three explanatory omissions always remain. The complete
result is sorted by depth/instance/visible id.

`project_atlas_impact_json` returns the existing fairyc.atlas-impact/v1 JSON,
including its Go-compatible field spelling and null/empty conventions. Output is
buffered and bounded to at most 4 MiB; failure returns no partial document. No
source body or host path is emitted. These modules have no filesystem, network,
process-execution or lock-update operation.

Executed on Windows amd64 with MSVC 14.51.36231 and Go 1.27.1:

- `go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlasImpact$'`: low and optimized pass, 50.54 seconds overall.
- `go test -mod=readonly -count=1 -timeout 4m ./internal/atlasmap -run '^TestImpact'`: pass.
- `go vet -mod=readonly ./internal/cli`: pass.
- `git diff --check` (including staged new files): pass.

Eight fixtures per native profile cover real v1/v3 maps from cyclic diamond and
foreign-call source workspaces, plus controlled canonical coverage metadata for
bounded/partial branches. Every visible/full declaration selector, five traversal
limit pairs and both input orders match Go's typed result and exact JSON bytes.
Cases also reject invalid limits/selectors, missing/duplicate partitions, mixed
schemas, too-small package/call/output budgets, missing target declarations and
private cross-package callees. Go decoding accepts the latter canonical mutations
before the Go join independently rejects them; these are graph checks, not merely
checksum failures. Exact resolved-call budgets pass with foreign calls excluded.

Linux execution remains queued; the native probe helper explicitly skips it.
A future Linux driver must run these same Fairy modules under GCC/Clang rather
than count the skip as evidence. Full CLI replay, candidate dispatch, independent
current-fact production, generic maps, MCP/edit integration, hosted CI and push
remain unexecuted/separate. Public CLI/API, SDK ABI, dependencies, DB, repository
hygiene settings and CI/runner settings did not change.

Source-of-truth updates: `docs/atlas/impact-mcp-edit-v1.md`,
`docs/development/ordered-work-20261031.md` and
`docs/development/source-of-truth-capabilities-contracts-and-atlas.md`.

## CLI argument leaf

`project_atlas_options` parses the existing impact selectors and explicit limits
before touching any workspace. It rejects missing, duplicate, unknown, overlong,
noncanonical, overflowing and out-of-range flags/values. `--locked` is optional
syntax only; the later command must always verify the current existing lock.
`TestCandidateAtlasOptions` compares the real native argument parser with Go's
`parseImpact` under MSVC low/optimized. This first leaf does not dispatch a command
or claim current-source/map production.

## Verified-current source producer

`project_atlas_source` freshly loads the locked workspace, completes ordinary
Fairy semantic/body/Flow/IR and public-interface verification, and hashes the
running native compiler through its host-owned executable path. It does not use
argv[0], a cached Atlas map, a Go child or a build companion as authority.
`project_atlas_facts` derives detached compiler/interface/package/feature/reference
and exact source-file/line-map facts from those retained owners. The expected
current JSON is built from those facts before a candidate map exists.

`project_atlas_declarations`, `project_atlas_calls`, `project_atlas_header` and
`project_atlas_produce` create bounded v3 partitions in memory, then re-enter the
existing transport/semantic/current-source readers and join. Source SymbolId
publication and validation share `project_atlas_symbols::derive`. Source parser
entries are filtered to actual parameters, and the transport's explicit mutable
parameter field is retained. Concrete compiler frame handles for ordinary
nongeneric source functions are not exposed as specialization identities.

This first producer admits local packages containing ordinary safe free functions
with no attributes and builtin void/bool/u8/i32/i64/usize parameter/result types.
SDK, foreign/unsafe/aggregate declarations, generics, interfaces, closures, other
signature forms and packages with no functions reject. Checked ordinary calls,
including nested calls and cycles, form the exact admitted reachability graph.
Non-call observations are deliberately not collected: their completeness and
header omissions explicitly retain candidate-atlas-not-collected. The generic
body omission remains visible. These are bounded partial impact facts, not a Go
completeness-parity claim or a new language-support restriction.

Executed: `go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateAtlasCommand$/low$'`.
The native MSVC command leaf passed in 119.07 seconds against Go's source-derived
identities, interface hashes and cyclic reverse-call graph. After adding scalar
parameter, nested-call and negative-command coverage, the actual main/dispatcher
also passed: `FAIRYC_RUN_PARITY_INTEGRATION=1 go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$'`
(248.04 seconds). The native process had empty PATH and no companion executable.
This low result covers visible/full selectors, parameter SymbolIds and independent
compiler/source/ordinary+generic interface identities, cutoffs, selector/package/
call/output ceilings, missing/stale locks, argument rejection before filesystem
access, stdout errors and exact workspace-tree/lock preservation.

`go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlasValidation$'`
passed low/optimized after sharing SymbolId derivation (12.656 seconds overall).
`go vet -mod=readonly ./internal/cli` also passed. Optimized full CLI acceptance is
recorded separately below when executed; Linux remains deferred.

## Authored CLI dispatch acceptance

`project_experimental_cli` dispatches Atlas to `project_atlas_cli` and delegates
existing fetch/experimental-help behavior to the unchanged fetch command owner.
The real authored `main` and `file_cli` entrypoint are retained in the Windows
acceptance image; there is no replacement test-only main in this acceptance.
The CLI accepts the existing explicit selectors/limits and optional --locked,
but always verifies the current existing lock. Diagnostics are bounded candidate
classifications rather than raw host paths, source bodies or OS errors.

Windows native execution records:

- `FAIRYC_RUN_PARITY_INTEGRATION=1 go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$'`: pass, 248.04 seconds (213.89 seconds inside the low profile).
- `FAIRYC_RUN_PARITY_INTEGRATION=1 FAIRYC_ATLAS_CLI_ARTIFACTS=<local ignored directory> go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/optimized$'`: pass, 203.02 seconds (157.32 seconds inside optimized). Additional coverage includes actual --locked queries, otherwise-valid aggregate/generic source rejection and root/experimental/fetch/Atlas help.
- Repeated optimized acceptance with the same C/profile/MSVC version and verified image digests: pass, 30.97 seconds, including 2.48 seconds inside the reused optimized profile. Every query still freshly verifies its workspace/source; only the test compiler image is reused.
- `go vet -mod=readonly ./internal/cli`: pass on the final test helpers.
- `git diff --check` including staged new files: pass.

Both successful full-profile runs execute with empty PATH and no Go/build
companion next to the native image. Ordinary scalar parameters, nested calls,
cycles and repeated call sites match Go's visible/full symbol selection, nodes,
depths, traversal cutoffs and current source/ordinary+generic interface facts.
The expected compiler identity is independently hashed from the actual native
executable. Non-call completeness is conservatively unavailable with explicit
omissions, so the complete JSON is not represented as byte-identical to Go's
richer source snapshot.

Queries leave the complete workspace tree and fairy.lock unchanged. Unknown
selectors, package/call/output ceilings and invalid arguments reject without a
result document. Missing/stale locks and unsupported source forms reject; stdout
failure returns 4. The final helper preflights otherwise-valid unsupported source
fixtures through Go before expensive C compilation. An earlier added generic
fixture used the wrong call spelling and failed this Go precondition; correcting
it to identity<i32>(1) changed only the test fixture, not product behavior. Earlier
producer field omissions were corrected before the successful full-profile runs;
failed/aborted development runs are not counted as passes.

Optional FAIRYC_ATLAS_CLI_ARTIFACTS is test-only. It retains a compiler image and a
source/image digest receipt outside Git tracking, reuses only matching generated
C/profile/toolchain-version/image inputs, and does not alter product cache policy
or attach any authority to cached workspace facts. Without it, test artifacts use
temporary directories as before. No workflow or runner setting changed.

Source-of-truth updates remain docs/atlas/impact-mcp-edit-v1.md,
docs/development/ordered-work-20261031.md and
docs/development/source-of-truth-capabilities-contracts-and-atlas.md. Linux
GCC/Clang replay is still batched, including the new argument/source/command/full
CLI probes; no Windows skip is counted as Linux evidence. Wider candidate source
admission, generic maps, MCP/edit integration, distribution and hosted CI remain
separate. No public CLI flag/schema, SDK ABI, dependency, DB, repository hygiene
setting, runner, push, deployment or release change accompanies this slice.
