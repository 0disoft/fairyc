# Partial aggregate resource cleanup, 2026-11-05

The aggregate native acceptance work exposed a Go reference compiler defect:
`Parent{.a=Leaf{...},.n=fallible()?,...}` returned on the error edge without
dropping the already completed owning field. The same loss occurs for a call
result or moved named owner. This violates the existing left-to-right and
reverse-completion rules in `docs/mir/drop-elaboration-v0.md`; it is not a new
language feature or a change to trap/unwind policy.

`lowerAggregateConstruction` now retains each completed owning prefix field as a
private ordinary typed resource obligation while later initializers run. Existing
function-exit/propagation cleanup discharges it on early return. Once the parent
is complete, explicit moves discharge the private copies and transfer exactly
one obligation to the parent. The final initializer needs no extra prefix slot.
Existing MIR operations, verification, codec and C drop glue remain the owners.
No source-visible binding, public ABI, dependency, DB or cache-wire change.

The original six-case interpreter regression fails in four cases before the fix:
inline/call/moved first fields and reordered completed fields all lose cleanup.
The final eight-case test passes, including failure before any field, error after
one/two fields, moved roots, nested branch propagation, older-local ordering and
successful parent transfer without double-drop. Each case round-trips MIR through
the existing codec and emits both Linux GNU and Windows MSVC C. Emitting Linux C
is not native Linux execution.

Executed and passed on Windows:

```powershell
go test -mod=readonly -count=1 ./internal/compiler -run '^TestPartialAggregateConstructionCleanup$'
go test -mod=readonly -count=1 -timeout 6m ./internal/mir
go test -mod=readonly -count=1 -timeout 6m ./internal/mir ./internal/compiler ./internal/cbackend ./internal/interpreter -run 'Partial|NestedOwning|IgnoredResource|AggregateConstruction|AggregateField'
go vet -mod=readonly ./internal/mir ./internal/compiler
gofmt -l internal/mir/lower_aggregates.go internal/compiler/partial_aggregate_cleanup_test.go
git diff --check
```

The broad selected regression ran matching MIR, compiler and interpreter tests;
`internal/cbackend` reports no matching tests, not a backend suite pass. C emission
is verified by the new compiler test itself. Full MIR tests pass in 0.435s; the
final eight-case test passes in 0.196s. Earlier failing commands are not passes.

Actual native Go/Fairy C trace comparison follows in the aggregate C execution
slice. Full CLI/cache, Linux execution, self-build, distribution and hosted CI
remain separate. No push, release, deployment or runner changes.


The subsequent final `TestCandidateGenericAggregateC` replay passes in 146.16s.
It includes actual Windows low/optimized Go reference C execution of the failing
partial-construction case and the same source through both Fairy host and C
consumer profiles. All observe exactly the single initialized child (`3,`) and
no outer cleanup; the successful nested/defer case observes `8,9,2,1,`.
The aggregate implementation plan owns the full 72-execution matrix and private
C consumer/strict-warning distinction. This is not Linux or full CLI evidence.
