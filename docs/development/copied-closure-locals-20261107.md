# Copied closure immutable locals, 2026-11-07

Status: the bounded Go/Fairy extension in
[ADR 0546](../adr/0546-copied-closure-immutable-locals.md) is implemented and
verified on Windows. This advances the wider closure queue; it does not deliver
first-class values or promote a closure ABI. Linux execution stays batched.

## Parallel ownership and implementation

Two retained Sol High workers own disjoint Go admission/tests/ADR and native
comparison tests. The coordinator owns Fairy admission/lifting, integration,
source/lock checks and the commit. No worker edits another owner's production
files or commits shared changes.

Both implementations admit at most sixteen immutable explicit/inferred i32
locals followed by a tail or final return. All initializers and the returned
expression share the existing 64-node budget. Names become visible only after
their initializer and are unique across captures, parameters and locals.
Unused bool/float/string initializers refuse; contextual i32 range checking stays
in ordinary typing, including integer literals larger than uint64.

Captures still snapshot at closure creation. Initializers run in source order
once per call; creating an unused closure does not run its initializers.
Fairy's tail-to-return lifting now retains every original local statement and
its entry/expression coordinates before appending the return. Explicit-return
bodies retain their original statements. Original arenas are append-only, so
two helper bodies cannot acquire each other's locals through an ID rewrite.
No call, mutation/control-flow, capture-type or escape boundary is widened.

## Windows observations

The Go worker runs these focused checks successfully, including existing copied
closure regressions, original AST immutability and original failure spans:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 -timeout 6m -v ./internal/closureprep ./internal/compiler -run '^(TestImmutableLocals|TestCopiedClosure)'
go vet -mod=readonly ./internal/closureprep ./internal/compiler
```

The compiler package takes 8.745s; four C11/C17 low/optimized native runs exit 42.
Tests cover 16/17 locals, shared 64/65 nodes including groups, initializer order,
unused traps, repeated calls, snapshot timing and generic-definition/instance
identity changes after editing or reordering unused initializers.

With `FAIRYC_RUN_PARITY_INTEGRATION=1`, the native worker runs
`TestCandidateCopiedClosureLocalsSource`: 39 source cases in both host profiles
pass in 45.54s. Current Go checks independently supply admission expectations and
written-function definition digests. Actual Fairy copies retain byte-equal
written AST encodings and original arena fields; the Go comparison uses the
definition digests, not an unrestricted full-AST stdout capture.

The final `TestCandidateCopiedClosureLocalsPipeline` passes in 196.16s. Both
103-part Fairy producers roundtrip/restore the existing checked artifacts and
emit actual C for three scenarios. Low/optimized consumers execute all twelve
combinations. Four ordinary runs return 42 with empty output. Eight trap runs
exit the independent observer's distinct code 73 only on the expected category
and exact original source byte range. This distinguishes an observed trap from
an accidental ordinary return. The existing SDK vector runtime remains linked.
Commands and failed/successful attempts are retained in
`.cache/copied-closure-locals-test-20261107.log` and its attempt log.

Initial harness attempts fail on a nonexistent Go exit constant, the unchanged
64KiB stdout allowance, an inappropriate trap-location helper and omitted
vector-runtime linkage. Those remain failures. The final harness compares
digests inside the existing output bound and supplies the real runtime; product
capture/work/deadline limits and warning levels are unchanged.

The coordinator's final existing `TestCandidateCopiedClosureCalls` replay passes
both profiles in 64.874s on the final product bytes. The preceding Calls/Identity
combined replay passes in 79.703s before the initializer-only literal refinement;
it is retained separately, not represented as another final-byte run.
Scoped CLI vet, gofmt and diff checks pass.

Fresh current-Go source checking uses bootstrap SHA-256
`fce79c8e772fccfd3956c2cc6e395fdb45b3ba9c318243816263aa0d88152fa5`.
Isolated `check` and `check --locked` pass in 18.661s / 13.288s:
755 source files / 9,542 MIR functions, empty stderr. All 1,226 authored files
and the final copied lock match the snapshot. Receipt:
`.cache/closure-locals-source-20261107-all-check.json`.

## Remaining gates

Actual full Fairy CLI/fresh-process disk-cache execution for this new family,
refreshed self-build/fixed-point comparisons, current distribution after the
remaining implementation, native Linux, allocator-failure injection, full Go
suite and hosted CI are not established by these focused runs. The earlier
`d0b84329` self-build and `a8a3e499` public ZIP receipts retain their original
identities; neither is relabelled as this extension's acceptance.

Source contracts changed: ADR 0496's forward reference, ADR 0546 and the candidate
closure architecture. No grammar production, public API/SDK/runtime ABI, DB,
dependency, repository-policy, runner or CLI flag change. No push, dispatch,
release, deployment or publication.
