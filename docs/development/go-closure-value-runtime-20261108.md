# Go closure backend/interpreter slice, 2026-11-08

This implements the backend/interpreter side of accepted ADR 0551 in the shared
isolated `.cache/closure-values-20261108` checkout. Parent-owned MIR model,
lowering and verification are dependencies. After the parent corrected its target
gate, focused interpreter/emission checks and all 28 native executions passed,
including real source-to-semantic-to-MIR-to-C execution. Earlier gate failures
are retained below as resolved receipts, not current blockers.

## Contract and representation

- `mir.OperationClosureCreate` (`closure-create`) carries exactly
  `hir.TypeClosureI32`, the lifted target `Symbol`, zero through four already
  evaluated i32 capture operands, and an ordinary destination local.
- The target has `ClosureTarget=true`, `ClosureCaptures` matching construction,
  i32 parameters ordered captures then value argument, and i32 result. The bool
  deliberately distinguishes a zero-capture target. The parent verifies target
  purity, safety, body bounds, provenance and must-live initialization.
- `mir.TerminatorClosureCall` (`closure-call`) has arguments `[callable, value]`,
  `Callee=0` (not `hir.NoSymbol`, which is -1), no foreign ABI, a destination and
  one normal successor. Lowering freezes the callable before lowering the value
  argument. Both consumers retain this ordering and existing call/trap rules.

C uses a private `fy_closure_i32` struct containing an invoke pointer and four
inline i32 captures. Construction initializes all four slots, with zero in unused
slots, and never invokes the body. A target-specific adapter passes captures in
written order, then the value argument, to the checked ordinary helper. Invocation
snapshots the callable and argument in separate C declarations. No environment
points into the creator's frame and no allocation/free/reference-count operation
is emitted for construction, copying or invocation. This is a private backend
layout, not a source size, field, pointer or FFI contract.

Construction adds a reachability edge to the target even when the creator never
calls it. Adapters are emitted only for construction sites in reachable blocks.
Single-unit emission, measured-body single-unit emission and partitioned output
all have adapter integration; shared partition headers declare adapters and the
entry unit defines them once. Closure-free emission adds no closure declarations
or adapter inventory allocations.

The interpreter keeps immutable target/capture records in execution-owned host
storage. Compact `runtimeValue` carries an integer record handle; its layout and
scalar payload remain unchanged. Captures contain four inline i32 values, no
creator-local references. Ordinary copies share the immutable record and allocate
nothing. The host storage is lazy and scoped to one execution, with construction
bounded by the existing step budget. This is interpreter bookkeeping, not a new
language allocator, retained async environment or cleanup obligation. Invocation
passes captures and the argument through the existing checked `call` path and
call-depth limit. No limit was raised.

Only an ordinary struct's direct closure field is added to aggregate transport.
Private layout checks reject repr(C), nested closure structs, arrays, tagged
payloads, resource storage and borrowed/pointer transport of closure-bearing
structs. Direct closure wrappers and foreign transport remain closed. Shared
scalar/tagged predicates were not widened. No equality, cast or pointer access to
the closure representation was added.

## Changed files

New helpers and focused tests:

- `internal/cbackend/closure_values.go`
- `internal/cbackend/closure_values_admission_test.go`
- `internal/cbackend/closure_values_test.go`
- `internal/cbackend/closure_values_source_test.go`
- `internal/interpreter/closure_values.go`
- `internal/interpreter/closure_values_test.go`

Integration changes:

- `internal/cbackend/aggregate_types.go`
- `internal/cbackend/analysis_functions.go`
- `internal/cbackend/analysis_locals.go`
- `internal/cbackend/analysis_reachability.go`
- `internal/cbackend/analysis_views.go`
- `internal/cbackend/arrays.go`
- `internal/cbackend/emit.go`
- `internal/cbackend/operations.go`
- `internal/cbackend/partition.go`
- `internal/cbackend/preamble.go`
- `internal/cbackend/resource_layout.go`
- `internal/cbackend/tagged_types.go`
- `internal/cbackend/terminators.go`
- `internal/cbackend/value_types.go`
- `internal/interpreter/execute_call.go`
- `internal/interpreter/execution_state.go`
- `internal/interpreter/value_state.go`
- This development receipt.

This slice changes no MIR/HIR/semantic/type/grammar/Fairy/ADR/cache/queue/lock
files, public API, dependencies, database, runner or repository configuration.
No commit or remote action was performed.

## Executed validation receipt

Windows, `GOMAXPROCS=1`, all commands with `-mod=readonly`:

```powershell
go test -mod=readonly -count=1 ./internal/cbackend ./internal/interpreter -run '^$'
go test -mod=readonly -count=1 ./internal/cbackend ./internal/interpreter -run '^(TestClosureValue(PrivateLayoutBoundaries|SnapshotCopyAndZeroSlots|RejectsForgedRuntimeState)$|TestEmitScalarControlUnit$|TestSmallBuildMatchesStreamingUnit$|TestCompactScalarPayloadPreservesEveryBit$|TestPreparedProgram|TestValueCopy|TestCompositeResultCandidateBoundaries$|TestControlOperandsPreserveSourceOrderAndReturn$|TestEmitCopyScalarTaggedFieldsInsidePrivateAggregates$|TestEmitAggregateBearingTaggedFieldsInsidePrivateAggregates$)' -v
go vet -mod=readonly ./internal/cbackend ./internal/interpreter
git diff --check
```

All passed. Compile-only reported cbackend 0.040s / interpreter 0.043s. The focused
passing run reported cbackend 0.062s / interpreter 0.059s. Vet completed with no
output. Independent tests cover every capture count 0..4, signed i32 extremes,
snapshot independence from subsequently changed creator locals, zero unused
slots, zero-allocation copies, forged handle rejection, private aggregate layout
and wrapper/FFI restrictions. Existing scalar emission, all four small-build
profiles (including callback transport), compact scalar bits, aggregate copies,
prepared execution reuse, operand ordering and tagged aggregate regressions passed.

An initial test-only native harness build referred to `NativeOutcome.Value`;
that was corrected to the existing `ExitCode` field before the passing compile
and focused runs.

### Resolved parent gate failures

The initial readiness command actually executed was:

```powershell
go test -mod=readonly -count=1 ./internal/cbackend -run '^TestClosureValueRuntimeMIRAndEmission$' -v
```

It failed before execution at the parent MIR gate (cbackend 0.056s): all six
fixtures reported `V1002`, local 100 used before initialization at
`block:0.terminator` and `block:1.terminator`; the five invoked fixtures also
reported local 106 at `block:1.terminator`. These are the synthetic construction
and value-call result locals. No verification was bypassed and no MIR scaffolding
was changed to make tests run.

After the parent wired closure verification, its shared focused run (session
43902, exit 1) passed interpreter tests but rejected both synthetic targets with
`V1021`, invalid signature or lifetime/effect facts. This worker then executed
only the following diagnostic case, without repeating the native matrix:

```powershell
go test -mod=readonly -count=1 ./internal/cbackend -run '^TestClosureValueRuntimeMIRAndEmission/zero$' -v
```

It failed at the same gate (0.137s). The new failure logging records each target's
complete JSON. Both `main::identity` and `main::target` have a single benign
`CleanupPaths` entry: `ExitKind="return"`, `Actions=[]`. Effects, resources,
unsafe operations, calls, loans, drop flags, region values, reassignments, moves,
return origins and region scopes are all empty. Their signatures are the correct
zero-capture i32-to-i32 signature. The parent predicate
`len(fn.CleanupPaths) != 0` therefore rejects normal lowerer metadata; checking
actual cleanup actions is the parent-owned next correction. No target metadata
was erased to bypass the gate. The parent subsequently allowed empty-action
ordinary return cleanup facts while retaining rejection of executable cleanup.
The successful runtime checks below followed that correction.

## Successful runtime checks after the parent gate

Executed with `GOMAXPROCS=1`:

```powershell
go test -mod=readonly -count=1 ./internal/cbackend ./internal/interpreter -run '^TestClosureValue' -v
go test -mod=readonly -count=1 ./internal/cbackend -run '^TestClosureValueSourceExecutionAndEmission$' -v
go test -mod=readonly -count=1 ./internal/cbackend -run '^TestClosureValueSourceNative$' -v
go vet -mod=readonly ./internal/cbackend ./internal/interpreter
git diff --check
```

All passed. The first command reported cbackend 18.614s and interpreter 0.141s.
It passed all six synthetic MIR/emission fixtures and independent private-layout
and interpreter copy tests. Native results were:

| Executed native group | Cases | Modes | Executions | Test duration |
| --- | --- | --- | --- | --- |
| Synthetic MIR C11 | 4 | low, optimized | 8 | 10.54s |
| Same emitted source consumed as Windows C17 | 4 | low, optimized | 8 | 6.98s |
| Partitioned C11 adapters | 1 | low, optimized | 2 | 1.02s |
| Real source through semantic/MIR/C11 | 5 | low, optimized | 10 | 20.50s |

The source interpreter/emission command passed in 0.079s. Its native command
reported package time 20.640s. Final vet and whitespace checks passed with no
output. Earlier valid matrix results were reused; the same native matrix was not
repeated after adding the independent source tests.

The readiness fixture starts from ordinary lowered checked arithmetic, then adds
explicit closure MIR. It covers returned factories after creator exit, capture
snapshot after changing the original scalar, two distinct helpers through one
parameter, immutable closure copies, struct copy/extraction, 0/1/4 captures,
division traps on invocation and an unused trapping body that must not run.
Emission checks also require creation reachability, deterministic output,
unchanged input MIR and partition adapter declarations.

`TestClosureValueRuntimeNative` executed C11 low/optimized runs for returned
struct copies, zero captures, unused trapping bodies and invocation traps.
`TestClosureValueRuntimeWindowsC17` executed the same cases with unchanged C11
source consumed as C17 at both optimization levels. The partitioned native test
executed adapters across translation units at both levels.

The new source tests use the ordinary frontend, closure preparation, semantic
checker and MIR lowerer without patching any MIR or target metadata. Their five
cases cover returned 4-capture factories with a changed creator scalar, a second
zero-capture target through the same callable parameter, ordinary struct copies
and extracted callable fields, a side-effecting argument evaluated once, an
argument division trap preceding a potential body overflow, an unused body with
a trapping local, and invocation of that trapping local. Interpreter results,
trap categories and strict native exit/output receipts agree. Source execution
also checks input MIR immutability and both Linux and Windows emission profiles;
native execution evidence is Windows MSVC only.

Full-suite, Fairy parity and cache restoration checks were not run in this slice.
Package/cache acceptance is explicitly outside this source/MIR execution stage;
the parent owns rejection of closure values at legacy artifact boundaries until
stable helper identities are implemented. Parent transport-guard completion and
later restored-artifact/CLI acceptance are not certified by these backend tests.
