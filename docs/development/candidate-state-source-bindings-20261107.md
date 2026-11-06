# Fairy state graph source bindings — 2026-11-07

This worker owns only the new `module_state_graph_source_bindings.fy` and this
receipt. The parent owns source/context/flow integration; the operand worker
owns `module_state_graph_source_operands.fy`. No shared files were changed here.

## Lowering implemented

The exported `bind` API takes the parser, statement ID, state layouts, lexical
environment, separate value/borrow counters, literal data, emitted operations,
source spans and the existing bounded work ledger. It returns the existing
private statuses: `0`, malformed/closed shape `1001`, work/size/ID bound `1002`,
or allocation `1003`.

Behavior follows `internal/compiler/state_graph_operands.go` and
`internal/compiler/state_graph_borrows.go`:

- Only immutable named locals are accepted. Existing environment names and any
  parsed function declaration name cannot be shadowed. Aliases allocate no new
  operation or ID, and retain codec and read-presence IDs. A found-result alias
  is an ordinary bool with no invented presence result.
- An annotation must exactly match a known codec `1..7`. Borrow annotations are
  rejected. Direct storage calls and grouped local aliases remain distinct,
  matching Go's syntax admission.
- Root-only cell/map borrow emits a fresh borrow ID and an ordinary presence
  value ID. Map borrow requires an exact literal key, never a value-reference
  key. Those IDs remain in separate namespaces.
- Borrow aliases retain their original ID. View/copy require an available
  borrowed local, with grouping allowed for that argument. View accepts literal
  u32 offset and positive literal u32 length, retains presence, and creates a
  fresh borrow ID. Copy produces an ordinary value with no presence result.
- Every emitted operation uses the contextual capability, cost one, zeroed
  unused fields, the expression span, and `context::emit`. Helper-owned borrow,
  view and copy calls are rejected.

This is a lowering helper, not an authorization boundary. The parent must run
the independent program verifier before publishing a proof. It owns actual
borrow invalidation, exact-key mutation, helper write effects, edge escapes and
type/availability validation. Runtime view range checks are not invented here.
On failure the parent discards the partial lowering; this API does not promise
rollback of counters, data or emitted operations.

## Previous verification performed by the bindings worker

- Current Go CLI `experimental check-syntax` passed with the candidate `src`
  directory as the source root.
- An isolated temporary workspace containing the actual 58-module authored
  import closure, the example manifest and a trivial entry point passed the
  current Go CLI `check`: **67 source files, 943 verified MIR functions**.
  The CLI was freshly built from current Go source. This did not build the
  complete Fairy compiler or change the repository lockfile.
- The same isolated workspace passed `check --locked`.
- `git diff --check`, strict UTF-8 decoding, LF-only text and final-newline
  checks passed for both owned files.

Preliminary invocations with the wrong source root or without standard-library
workspace loading failed and are not counted as passes. The first isolated
workspace check also caught the operand worker's transient missing return in
`ungroup`; that worker corrected its own file before the successful check.

No new Go tests were written by the bindings worker. Current authored-source
Windows native and independent whole-program comparisons are recorded below.
Linux, allocation-failure injection, ordinary Complete/cache reuse and actual
state execution remain open.

No source-of-truth contracts, public CLI/SDK API, DB, dependencies, CI/runner
settings or repository hygiene policies were changed. No staging, commit, push,
release or deployment was performed by this worker.

## Integrated authored-source validation, 2026-11-07

The parent independently compared the actual source producer and restored
root/helper program with Go: 68 authored cases / 136 Windows low/optimized
processes, package 24.792s. Retained source and artifact regressions bring this
slice to 374 native processes and eight ownership refusals. Fresh bootstrap
`check` and `check --locked` pass 716 files / 8,917 MIR functions; scoped vet
and source hygiene pass. The separate operation-limit suite is not included.

Exact commands, work/ownership controls, failed-attempt accounting and full
integration records are in
[the source receipt](candidate-state-graph-source-20261107.md#integrated-authored-source-validation-2026-11-07).
The earlier module-specific checks above remain historical evidence.
Linux execution and allocator-failure injection remain deferred; ordinary
Complete/package/cache/command/execution admission remains separate.
