# Fairy generic aggregate implementation, 2026-11-05

Scope remains the actual Fairy compiler's aggregate execution and its Atlas,
interface/cache and CLI integration. This plan does not replace that scope with
source-only projections. ADR 0387 and the existing Go implementation own the
admitted scalar-parameter struct/resource semantics. Native Linux remains batched.

## Commit sequence

1. **Source plan prerequisite, implemented.** The explicit compiler-private
   `project_source_plan::build_aggregate_sources` route shares the current binding,
   catalog, root/dependency and specialization owners. It admits scalar-parameter
   structs/resources, preserves source kind, ordered arguments and cleanup-helper
   edges, and retains exact package-coordinate `g0_`/public-closure identities.
   User aggregate planning does not acquire SDK storage intrinsics or nominal
   argument admission. Ordinary `build_sources` still closes aggregate execution.
2. **Concrete aggregate types and fields, preparation implemented.** Materialize each original declaration
   and ordered argument tuple once; substitute fields structurally, preserve
   source ownership, and reject recursive/unresolved layouts without partial
   publication. Integrate with actual signatures, annotations and construction.
3. **Ownership, cleanup and executable MIR.** Connect resource instances and
   cleanup-helper calls to Flow, kind-aware identities, MIR verification and both
   C emission forms. Require native values and exact cleanup traces against Go;
   planning success alone is insufficient.
4. **Interfaces, restoration and Atlas production.** Keep original templates as
   source exports, bind instance/type/cleanup identities to current source,
   preserve definition-dependent invalidation and emit concrete aggregate and
   cleanup facts from Complete. Include SDK source families through their actual
   authenticated host path.
5. **Command and final acceptance.** Run actual check/build/run/test and Atlas
   command/full-CLI and persistent restoration controls, then collect Linux replay,
   consecutive self-build, distribution and hosted evidence under the main queue.

## Source-plan evidence

`TestCandidateGenericAggregatePlan` compares actual Fairy MSVC low/optimized
execution with `genericsource.PreparePackages` over three packages. Five positive
variants each contain three roots, nine specializations and five public maker
templates, including an unused aggregate. It compares ordered bindings, original
declaration IDs, `g0_` IDs, dependency-first order and public template closures.
Cleanup helper edits change dependent identities; private/unused edits preserve
concrete identity, and reversed module order plus rebased numeric owners preserve
the coordinate-bound facts.

Seven negative variants cover private aggregate access, wrong arity, an active
recursive field, nonscalar arguments, omitted arguments in a written annotation,
duplicate binders and generated names. All reject in Go and Fairy. Each positive
variant separately verifies the unchanged ordinary aggregate gate and work
exhaustion. Source planning is not concrete field substitution, type checking of
all template bodies, resource cleanup, native aggregate execution or CLI support.

Command: `go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateGenericAggregatePlan$'`.
Passes in 16.21s on Windows; source-planner processes run with empty PATH.

Existing generic-function Atlas production, copied-closure identity and static
operation identity regressions pass in both Windows profiles (149.254s):

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidate(StaticOperationIdentity|CopiedClosureIdentity|AtlasGenericProduction)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_plan_test.go
git diff --check
```

Scoped vet, formatting and diff checks pass. During probe preparation, an omitted
type-argument control was moved into an explicit type annotation: bare untyped
record construction is left to ordinary semantic analysis by the Go preparation
adapter. A missing copied test dependency and an explicit test-reader return were
also repaired before the final native pass.

No stable public API/SDK ABI, DB, dependency, repository-hygiene or runner change.
No push, CI dispatch, release or deployment. The full implementation queue remains open.

## Concrete type preparation evidence

The compiler-private `generic_aggregate_types::build` now consumes the sealed
original plan, reauthenticates every supplied source, and materializes concrete
aggregate fields in dependency order. Its retained catalog includes original
parameter arity. `module_types` uses distinct kind 20 (struct instance), 21
(resource instance) and 22 (ordered scalar argument tuple); kind 13 remains the
SDK Vec representation. Instance equality uses declaration coordinates and the
ordered tuple, not a generated spelling. Duplicate requests share a type ID.

The preparer resolves original field roots, concrete function results/parameters,
written local annotations, casts and explicit aggregate constructors into the
same type graph. Every aggregate value type must join exactly one materialized
source instance before publication. Field order, source module/declaration,
original field indices and raw spans remain intact. A failing preparation returns
no Prepared object; individual type resolution rolls back appended graph nodes.

`TestCandidateGenericAggregateTypes` uses actual detached fields and signatures
from Go `genericsource.PreparePackages` as its independent oracle. The positive
matrix has nineteen three-package variants: the original five source-plan
variants, eleven additional scalar descriptors (covering all thirteen together),
a written local annotation, reversed ordered arguments and structural fields.
Each has five aggregate instances and four concrete functions. Nested generic
records, Option/Result, grouping, arrays, borrows and raw pointers are represented
as types. The fixture compares field names/order/spans and types, concrete
results/parameters, and verifies that five constructors select shared instances.
This is type preparation, not a claim that those fixture bodies pass ordinary
value/ownership checking or execute.

The seven existing source rejection controls remain required. Additional native
controls check exact-source drift, exhausted preparation work, an unresolved
active field, invalid type roots and arity, and type-graph capacity: 4,094 existing
nodes plus a one-argument tuple and instance fill the 4,096-node limit; at 4,095
the partial tuple is rolled back; at 4,096 failure preserves the graph. Repeating
an existing instance at full capacity returns the same ID without growth.

The ordinary `module_types::assemble` gate still refuses these private aggregate
and tuple records. The ordinary `module_signatures::Program` and Interfaces
owners, full field/layout validity, construction value checking, ownership,
cleanup, Flow/MIR/C, interface/cache/Atlas publication and full CLI execution
remain required follow-up work. In particular this preparation does not attest
layout size/alignment, borrowed fields in stored records, or native aggregate
execution. No broad implementation-queue family closes here.

Final focused replay passes in 42.380s on Windows, with MSVC low/optimized:

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateGenericAggregate(Types|Plan)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_plan_test.go internal/cli/candidate_generic_aggregate_types_test.go
git diff --check
```

The plan probe passes in 22.83s and concrete-type probe in 19.44s. Scoped vet,
formatting and diff checks pass. Probe processes use empty PATH. During fixture
construction the harness was corrected to count written parameter entries rather
than all entries, retain the extra local-annotation root, and render the existing
nominal-pointer mutable bit correctly; those initial failures are not passes.

A development regression run with `FAIRYC_RUN_PARITY_INTEGRATION=1` also passes
`TestCandidate(AtlasGenericProduction|AtlasTypes|GenericTransportPipeline|GenericTransportAdmission)`
in 406.382s. Its earlier Atlas/compiler-host snapshots precede the final private
aggregate group/pointer/argument-guard edits; the final scalar/signature admission
probes and focused aggregate replay attest the final source. This run is not a
refreshed full CLI or consecutive self-build result. Linux remains batched; self-build, clean-machine distribution and
same-commit hosted CI remain open. No stable public API/SDK ABI, DB, dependencies,
repository-hygiene or runner configuration changed.
