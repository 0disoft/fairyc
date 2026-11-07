# Go inline same-module calls, 2026-11-07

Owner: [ADR 0548](../adr/0548-inline-same-module-call-graphs.md). This is a
Windows implementation checkpoint, not completion of general inline bodies.

## Implementation

The HIR encoder accepts actual resolved same-module public inline declarations
and preserves all no-call v1/v2 bytes. Call-bearing bodies use the independently
specified v3 tuple. Semantic checking finalizes markers after all module bodies
are checked, so forward declarations and reverse-written graphs have the same
rules. Pure i32 call arguments/initializers retain ordinary ordered evaluation.
The package inline-class owner builds one module resolver and rechecks every
export with it; changing a reachable leaf changes the existing package class.

MIR verification validates the actual executable call graph again, refusing
unmarked, foreign, other-module, unsafe/effectful targets, cycles and more than
64 distinct reachable declarations. Existing operation/CFG budgets remain.
No public API/ABI, cache schema/key version, DB, dependency, runner, product
timeout or target change is made. The internal HIR helper keeps old callers
through an optional resolver argument.

## Actual Windows checks

With `GOMAXPROCS=1`, the focused run passes:

```powershell
go test -mod=readonly -p=1 -count=1 -parallel=1 -timeout 6m -v ./internal/compiler ./internal/inlinebody ./internal/mir -run '^(TestInlineCalls|TestInlineLocals|TestCrossPackageInline|Test.*Inline)'
```

Compiler 5.766s, MIR 0.049s; `inlinebody` has no package-local tests. The compiler
matrix includes the independent literal v3 digest, no-call identity regression,
real execution/restored MIR, argument/initializer trap order, forward/reverse
64/65-function shared DAGs, 64/65 expression nodes and forged call/marker/target
refusals. Twelve actual native images cover value/division-first/overflow-first
under C11/C17 and low/optimized, returning 42/195/194 with empty streams.
Log: `.cache/go-inline-calls-20261107-attempt1.log`.

The maximal nested-node test was added after the first test binary was built,
so it is verified by the separate final selection rather than credited above:

```powershell
go test -mod=readonly -p=1 -count=1 -parallel=1 -timeout 3m -v ./internal/compiler ./internal/cbackend -run '^(TestInlineCallsMaxNestedNodeBudget|TestPartitionedInlineBody|TestPartitionedZeroParameterInlineTraps|TestPartitionedInlineNestedTraps)$'
```

The final selection passes. The 63 nested calls plus one parameter node produce
zero MIR operations, 63 call terminators and 64 blocks, return 42, and retain
the existing operation budget. The next node is refused. C backend 10.037s
also replays existing parameterized/zero-parameter nested traps and partitioned
inline bodies under both profiles. Log:
`.cache/go-inline-calls-final-boundary-20261107.log`.

Scoped `go vet -mod=readonly ./internal/hir ./internal/semantic ./internal/mir
./internal/compiler ./internal/inlinebody` passes; gofmt and diff whitespace
checks pass. Native Linux remains batched. Fairy source/artifact/CLI/cache,
full-suite/self-build/fixed-point, distribution and hosted CI are independent
gates. No push, dispatch or publication occurs.
