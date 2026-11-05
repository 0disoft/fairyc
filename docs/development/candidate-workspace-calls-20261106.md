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
- [ ] Fresh Complete/source-selected production and actual dispatcher/help/
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
