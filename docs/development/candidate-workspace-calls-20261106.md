# Fairy workspace call catalog implementation, 2026-11-06

Part of the original complete implementation queue; finishing a prerequisite
does not finish the calls command or broader objective. Native Linux remains
batched. No push, dispatch, publication or target promotion is authorized here.

## Required sequence

- [x] Exact `atlas-calls list` arguments, versions, directions and positive
  package/call/output bounds, compared with the actual Go parser.
- [ ] Current-joined graph output v0/v1/v2 with exact source/cleanup/
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
