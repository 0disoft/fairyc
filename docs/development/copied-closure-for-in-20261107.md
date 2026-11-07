# Direct copied-closure calls in for-in, 2026-11-07

Base: detached `6270e3673e8a5551a927422550a853c3cbb109e9`, in the dedicated
`parallel-closure-for-in-20261107` worktree. Parent main is not edited here.

## Supported boundary

This is the bounded follow-up to [ADR 0496](../adr/0496-explicit-copied-local-closures.md).
An existing unique untyped outer-function `let` can be called directly and
positionally from an otherwise admitted view/vector for-in body. It can also
remain active across a loop that does not call it. Existing for-in source types,
SDK vector authentication, owner holds, item lifetime and mutation checks still
decide whether the loop is executable. No grammar production changes.

Captures are still copied once at the outer binding, before iteration. Mutating
the original local or shadowing its name with an item does not update that
snapshot. The item shadows a same-named closure only in the loop body; iterable
evaluation uses the enclosing environment and the outer closure returns after
the loop. Nested scalar shadowing follows the ordinary lexical rules.

Closure declarations inside a loop remain rejected. Copy, pass, return, store,
assignment, mutable/implicit captures, resources/borrows/pointers, labeled calls,
oversized or impure closure bodies remain excluded. There is no retained,
asynchronous or cross-thread closure. The helper still returns i32 and cannot
retain a reference item. No heap environment, dynamic-call ABI, MIR opcode,
private allocation/header policy, deadline or budget is added or increased.

## Implementation

- Go `closureprep.Prepare` removes its blanket active-closure/for-in refusal.
  It rewrites the iterable with the outer closure environment and the preserved
  written body with a cloned environment excluding the item name. Generated
  scalar/vector loop blocks continue through their ordinary binding scopes.
- Fairy `copied_closure_lower` rewrites kind-18 iterable/target first, pushes a
  scalar shadow binding for the item while rewriting the body, then restores the
  enclosing environment and scope. This uses the existing bounded binding table.
  Helper/snapshot tokens and the reach-aware AST projection are unchanged.
- The original syntax and written source remain untouched. Existing package
  identity and cache formats are reused. Previously rejected active-closure
  loops have no admitted old executable artifact; ordinary source freshness,
  typing, Flow and artifact verification remain authoritative.

## Narrow acceptance

The shared view/vector sources live under
`internal/cli/testdata/copied_closure_for_in/`. They check first/last values, zero
iterations, an immediately preceding closure binding, immutable capture
snapshots, nested loops, item/local shadowing and post-loop closure use. Go
negative cases check escaped item names, calling a scalar item instead of the
outer closure, nested closure declarations, copy/return/store and vector owner
mutation. The SDK vector case uses actual SDK sources and the existing native
storage test helper; it does not fabricate a vector implementation.

The workspace copied-closure cache fixture now calls the closure from a copied
integer for-in, preserving its 42/43 body-change oracle, ordinary/generic paths,
memory/disk hits and reopened disk-session check.

`TestCandidateCopiedClosureCalls` replays the actual authored parser, private
lifting, names/calls and Values path for the view fixture. The vector fixture
uses authentic `std::vec` source and reaches the resolved call plan. It also
preserves the previous admission/refusal matrix and adds loop-specific
scope/escape refusals.
`TestCandidateCopiedClosurePipeline` includes the view fixture in its existing
complete compile, artifact encode/decode/restore and generated-C execution
probe. This is a small purpose-built compiler host, not a full authored CLI or
selfbuild/fixed-point run.

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run 'TestCopiedClosure' -count=1 -timeout 5m -v
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosureCalls$' -count=1 -timeout 6m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosurePipeline$' -count=1 -timeout 8m -v
```

Go compiler PASS (11.705s), workspace compiler PASS (3.348s). View native execution
passes C11/C17 low/optimized; SDK vector passes low/optimized, all exit 42 with
empty streams. Both source/SDK AST immutability checks and nine new negative
subcases pass. Cache tests pass ordinary/generic, memory/disk and restored-session
paths.

## Existing Fairy vector execution boundary

Go vector execution passes. The Fairy unit Values path still rejects a vector
iterable with code 2, independently of copied closures. Initial diagnostics were
module 0, bytes 208..217 (`ref items`); after using the existing explicitly typed
shared descriptor form, bytes 243..247 (`view`). Both failures happened after
private closure lifting and resolved call-plan construction.

The regression compares the closure program with a no-closure control that
replaces both literals with ordinary named arithmetic functions. Both must reach
the call plan and either succeed together or refuse with code 2 at the named
empty vector descriptor. The empty descriptor is scoped before pushes, so this
control does not introduce an owner-mutation/borrow violation. The live
descriptor for subsequent loops is created only after population. This proves
the existing Values boundary is preserved; it is not authored vector native
execution evidence. Completing Fairy vector Values/IR admission is a separate
task, outside the closure-preparation/lowering write scope here.

No Values, vector type, ownership, native allocator, ABI/header, timeout or cap
implementation was changed to make that control pass.

The generated-C pipeline PASS (169.458s): one purpose-built compiler host,
101 C parts, low/optimized host profiles, artifact encode/decode/restore, and
low/optimized generated consumers, all exit 42 with empty output. This validates
the view execution path. The full authored CLI was not rebuilt.

Final authored admission replay PASS, low and optimized (53.364s), after the
fixture descriptor scopes were adjusted to preserve owner holds. The earlier
comparison replay passed in 35.972s. The first fixture/control attempts are
retained as failure logs, including the control's initially misplaced import
order; these are not counted as passes. Logs are retained
only in this worktree's `.cache/closure-for-in/`. Native Linux, full authored
selfbuild, full source/locked integration, ZIPs, hosted CI and remote operations
are intentionally not run. The parent owns final lock integration; no
`fairy.lock` is committed by this worker. Native timings occur under concurrent
same-host work and are not a throughput or memory claim.

Passing logs: `go-tests.log`, `authored-admission-freeze.log` and
`authored-pipeline.log` under `.cache/closure-for-in/`. Full per-file SHA-256 and
local commit identities are retained in that directory's `receipt.json`.
First local detached commit: `cc7fd701dac2933a18d7cd9ec9ede0a455d6a429`
(Go preparation, native/cache regressions and shared fixtures). The follow-up
commit contains Fairy lowering, authored regression updates and contract notes.
