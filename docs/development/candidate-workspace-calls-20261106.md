# Fairy workspace call catalog implementation, 2026-11-06

Part of the original complete implementation queue; finishing a prerequisite
does not finish the calls command or broader objective. Native Linux remains
batched. No push, dispatch, publication or target promotion is authorized here.

## Required sequence

- [x] Exact `atlas-calls list` arguments, versions, directions and positive
  package/call/output bounds, compared with the actual Go parser.
- [x] Current-joined graph output v0/v1/v2 with exact source/cleanup/
  specialization identity, canonical order/deduplication, context, whole-graph
  budget/usage, independently verified checksums and atomic byte/work refusal.
- [x] Fresh Complete/source-selected production and actual dispatcher/help/
  lock/error/output acceptance under both Windows profiles. No child Go fallback.
- [ ] Remaining full authored-main, self-build, distribution, same-commit hosted
  and batched Linux gates from the original queue.

Owners: `docs/atlas/workspace-call-catalog-v0.md`,
`workspace-call-catalog-v1.md`, `workspace-call-catalog-v2.md`,
`workspace-call-join-v0.md`; Go parser/implementation: `internal/cli/atlas_calls.go`
and `internal/workspacecallcatalog/`. Catalog 0/1/2 require current-admitted maps
1/2/3 respectively. No relabeling/conversion or implementation-ID selectors.

## Argument selection

`project_calls_options.fy` validates only process arguments. Package UID, visible
or full source SymbolId, direction and all three limits are mandatory. Default
catalog 0 equals explicit 0; versions 1 and 2 select the other existing families.
The seven scalar flags and locked switch appear at most once, in any order.
Canonical decimal package/call limits are 1..65,536; output is 1..1,073,741,824.
The exact maximum invocation is nineteen words including executable/command.
Argument selection performs no source, lock, map/cache, network or stdin I/O.

`TestCandidateCallsOptions` compares 87 actual Go parser cases plus three invalid
actions in each Windows profile, 180 native processes total. It checks missing/
repeated/unknown flags, both selector forms and widths/case, implementation-ID
refusal, all directions, version/map mapping, canonical decimals, overflow,
limits at one-short/exact/one-more, UID length and the full maximum invocation.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateCallsOptions$'
```

PASS in 13.161s (test body 13.05s; low 5.73s, optimized 6.46s), no selected skips.
Log: `.cache/calls-options-20261106.log`. The initial conditional return of three
different static literals failed existing borrow-origin admission before native
execution; the getter now selects slices of one static literal, as the existing
symbol-kind selector does. That failed attempt is retained separately at
`.cache/calls-options-before-20261106.log`; no language/lifetime rule changed.

Authored `check`/`check --locked` pass with 651 source files and 8,173 MIR
functions. Scoped CLI vet, Go formatting, LF and diff checks pass. This closes
argument selection only; joined-call rendering, actual calls dispatch, full CLI
and every original final gate remain open. No public CLI/SDK ABI, grammar,
dependency, DB, workflow/runner or repository-policy changes; no push or dispatch.

## Current-joined canonical output

`project_calls_catalog::encode` consumes only a current-admitted joined Graph.
The Graph now retains its actual construction limits; a request with different
package/call limits, map version or nonunique exact package UID is rejected.
Selection resolves visible/full original source SymbolIds and preserves outgoing,
incoming and both directions. Context includes the selected package's independent
current map identity, map omissions and call/generic completeness, while usage
continues to describe the whole joined ledger even for an empty selection.

`project_calls_order` sorts owned edge indices with bottom-up merge sort,
O(N log N) comparisons and O(N) scratch, leaving the Graph unchanged. Order and
deduplication include original endpoints, cleanup-body marker, exact source/target
specializations and complete spans. Canonical JSON field order and the separate
v0/v1/v2 domain-separated SHA-256 envelopes match the existing Go contracts.
Output is held privately until every write/hash/final-byte/work bound succeeds.

`TestCandidateCallsCatalog` uses three real independently checked workspaces and
the actual Go v1/v2/v3 join and catalog factories. Its final Windows low/optimized
matrix passes 159 native observations per profile (318 total), including all
source symbols and directions, reversed partition order/full selectors, recursive
self-call deduplication, unused declarations, cross-package specializations and
two concrete resource cleanups. The Go envelope checksums are recomputed using
independent SHA-256 and length framing before native byte comparison.

The final matrix also checks exact/one-short whole-graph package/call budgets;
one-short/exact/one-more final output and measured encoder-work limits;
independent reader and encoder cross-version refusal; changed budget echoes;
duplicate/missing packages; unknown UID/symbol; corrupt maps and stale independent
current expectations. All failures return no partial catalog. This isolated probe
uses private exit 7 for rejection; product command 2/3/4/7 classification is not
inferred from that sentinel and remains part of the actual dispatcher work.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 8m -v ./internal/cli -run '^TestCandidate(CallsCatalog|CallsOptions|AtlasGenericJoin|AtlasImpact)$'
```

PASS, 183.703s, no selected skips. Catalog: 46.72s; existing generic join: 25.75s;
existing impact: 96.98s; parser: 14.07s. Log:
`.cache/calls-catalog-core-final-20261106.log`. The initial 300-observation catalog
run passed in 29.704s before the extra corrupt/stale/encoder-version controls;
it is not added to the final 318-observation count. The original uncommitted
sort failed E1008 because it mutated an immutable parameter; moving the owned
vector into a mutable local fixes it without admitting caller alias mutation.

Authored check/locked pass at this core slice with 653 source files and 8,185 MIR
functions. Scoped CLI vet, formatting, LF and diff checks pass. This closes the
private joined-output API only. Fresh Complete-backed command assembly, actual
dispatch/help/error/lock behavior, full CLI, generated consumers and all original
self-build/distribution/hosted/Linux gates remain open. No grammar, stable CLI,
SDK ABI, dependency, DB, runner/workflow or repository-policy change; no push or
dispatch. The broader implementation objective remains active.

## Fresh source-backed authored CLI

`project_calls_source::query` freshly resolves the requested lock policy,
selects the exact requested source/map family, builds one ordinary/aggregate
Complete and derives every package from that same Complete. All packages re-enter
independent current-source admission before the bounded workspace join and
catalog encoder. Neither an existing map/cache nor a child Go compiler supplies
facts. The established SDK source/query work-ledger separation is retained.

`project_calls_cli` and `project_experimental_cli` now route actual
`experimental atlas-calls list`. Arguments/help precede workspace I/O. Product
exit codes are 2 for invalid selectors/join/output budgets, 3 for source/family
refusal, 4 for internal preparation/stdout failure and 7 for workspace/lock failure.
Successful output is published once, without a trailing newline; failure never
publishes a partial catalog. The command implements the existing experimental
contract rather than changing stable CLI, grammar or SDK ABI.

`candidate_calls_command_test.go` uses independent Go source declarations/calls,
current identities and native image SHA-256. The shared test-only
`candidateCatalogCommandPartition` projection declares only the existing
non-call collection omissions; it does not learn IDs, edges, bounds or catalog
bytes from the candidate. Its 26 ordinary command invocations cover all three
catalog versions/directions, visible/full selectors, recursive self-call
deduplication, empty selection, exact byte/join bounds and atomic refusals,
missing/duplicate/unknown argument policy before I/O, stdout failure, locked
immutability and stale-source refusal. The unlocked freshness test independently
discovers the new source selector, restores the exact stale lock, then requires
the native invocation alone to publish the expected new lock and catalog.

`candidate_calls_source_command_test.go` adds ten native invocations for used
cross-package function generics in v1/v2, unused templates, two cross-package
cleanup specializations and distinct owner-qualified namespaces, including
lower-version source-family refusals. Two SDK invocations cover ordinary SDK v0
and used Vec v2 with the unchanged SDK-specific diagnostic cap. All locked source
and SDK workspaces remain byte-identical. Four new help forms join the existing
ten help probes. These three command groups are appended to the full authored-main
verifier without deleting or reordering its original thirty-five registrations.
The helper clears PATH, so a child Go or C compiler cannot satisfy these queries.

The final source and generated image fingerprints are recorded by each command
below; reuse requires exact generated-C/profile/toolchain/stack and image digest
agreement. No observation from an old compiler image is promoted to this one.

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-calls-cli'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/optimized$/^(calls-ordinary|calls-sources|calls-sdk|help|catalog-sources)$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$/^(calls-ordinary|help)$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/low$/^(calls-sources|calls-sdk)$'
```

| Final selection | Result | Log under `.cache/` |
| --- | --- | --- |
| Optimized three calls groups, help and existing symbols source regression | PASS, 164.136s | `calls-cli-optimized-20261106.log` |
| Low ordinary calls and help | PASS, 369.726s | `calls-cli-low-ordinary-20261106.log` |
| Low source-family and SDK calls | PASS, 271.206s | `calls-cli-low-sources-20261106.log` |

The optimized selection verifies 65 native invocations: 26 ordinary calls,
ten source-family calls, two SDK calls, fourteen help and thirteen existing
symbols source checks. The two low selections verify another 52 native
invocations (40 ordinary/help and 12 source/SDK), 117 full-main observations
total. Four of thirty-eight registrations pass both profiles; the additional
existing symbols source group passes optimized. The other thirty-three
registrations, plus low symbols-source regression, are not attested by these
selections. The low SDK executions take 82.96s/79.68s under the unchanged
SDK-specific cap; they are observations, not a product timeout/performance claim.

| Profile | Generated-C/profile/toolchain/stack source fingerprint | Image SHA-256 |
| --- | --- | --- |
| Low | `ff5e6c4177b811f5ca3ebdb51317693aa2224c9e91cb39ab482e3ff6e6473cd8` | `e96e871adc33ba6d906ec4c978459290cddc8b65a9d7c489901714fb1e85821f` |
| Optimized | `28e6e166a6c825f9fc6a61beac70102af5f57ec8606e0e1fcd757b8d2cd00af4` | `2c4e019a3ca25a782f84b444a9fd3f035bc2420151dd883428ed4dc4dd18f2f2` |

The Go calls/symbols command regression passes in 3.781s; scoped CLI vet,
formatting, LF/diff checks and authored check/locked (655 source files,
8,190 MIR functions) pass. No stable CLI, grammar, SDK ABI, dependency, DB,
workflow/runner or repository-policy change.

Actual calls command acceptance is scoped to the admitted producer families.
The remaining full-main verifiers, generated consumers, broader source/non-call
coverage, atlas-pack and every original self-build/distribution/hosted/Linux gate
remain required. Native Linux stays batched; no push, dispatch or publication.
The complete implementation objective remains active.
