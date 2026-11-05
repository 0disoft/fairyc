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
2. **Concrete types/fields/signatures, field contracts and Interfaces preparation implemented.**
   Each original declaration and ordered argument tuple is materialized once;
   fields substitute structurally and preserve source ownership. Signature,
   annotation and constructor type requests use the actual graph. Ordinary
   field/layout contracts now check specialized shape facts and finite by-value
   layouts. The ordinary Interfaces owner now consumes that route with retained
   constructor, specialization and cleanup facts. Construction value checking is
   implemented below, as is ownership/Flow checking. Executable lowering remains
   open.
3. **Ownership/Flow, cleanup/body MIR and single-buffer/partitioned CFG/C execution implemented.**
   Resource instances now enter the ordinary move/loan and cleanup-policy owner.
   Connect checked bodies to kind-aware identities, MIR verification and both C
   emission forms. Require native values and exact cleanup traces against Go;
   checked cleanup requests alone are insufficient.
4. **Unit/project source identities and linking implemented; interfaces, restoration and Atlas production remain.** Keep original templates as
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

At that preparation checkpoint, ordinary `module_types::assemble` still refused
these private aggregate and tuple records. The signature-only follow-up below
now joins the existing Program owner through a private route. Interfaces, full
field/layout validity, construction value checking, ownership,
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

## Ordinary signature integration

`module_signatures::build_aggregates` now consumes the authenticated concrete
preparation and publishes the existing Program owner with its retained original
plan/fields. Ordinary `build`/`build_catalog` remain closed for explicit user
aggregate instances. Source reauthentication precedes signature publication;
work exhaustion, declaration checks and existing effect/foreign diagnostics return
no partial Program. Generic methods remain outside the admitted slice.

The type Graph records whether it was admitted through the private aggregate
validator. That validator checks original declaration kind and arity, ordered
scalar tuple tails, bounds, backwards edges and uniqueness. Tuples cannot become
Option/Result/borrow payloads. Struct/resource instances stay distinct from SDK
Vec. `sdk_vector_plan` keeps that graph mode through actual signature resolution,
specialization, ordered batch queries and owned graph reconstruction; ordinary
callers keep the existing path. Source-only extended scalar handles materialize
into concrete scalar nodes when the private specialization resolver needs them.

`TestCandidateGenericAggregateSignatures` independently runs Go
`AnalyzeGenericPackages`, requiring completion through Flow, no diagnostics and
no unsupported forms for each accepted input. Detached Go expansion supplies the
expected ordinary signature types; Fairy uses its actual Program owner. Seven
three-package positive variants cover concrete resource return/borrow, ordered
struct arguments, owner/module reorder, f64, Result, borrow and `@must_use`.
Forty ordinary signatures are compared per native profile. Three rejected inputs
cover invalid must-use result, foreign generic aggregate and generic owner method;
Go rejection includes its explicit unsupported-source channel.

Additional controls retain the ordinary signature gate, exact source drift/work
refusal, mode-preserving owned graphs, tuple escape, wrong arity and ordered batch
failure without graph growth. The probe's supplied sources are Go Flow-valid in
accepted cases, but no candidate body Flow/MIR, cleanup execution or executable
aggregate behavior is claimed by this signature-only admission.

Actual field/layout contracts and ordinary Interfaces wiring, construction value
checking, ownership/cleanup, executable MIR/C, interface/cache/Atlas and command
acceptance remain required. The complete queue remains open; native Linux stays
batched. No stable API/SDK ABI, DB, dependencies, hygiene or runner change.

Final signature probe passes in 23.01s on Windows MSVC low/optimized, including
all seven extended scalar handles through single and ordered batch specialization,
plus invalid-handle refusal without graph growth:

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateGenericAggregateSignatures$'
```

The final scalar-handle controls were added after the combined replay began;
that combined replay's production inputs are unchanged. The focused probe above
contains the final test inputs. Scoped vet, gofmt and diff checks pass. Native
processes use empty PATH. No Linux, full candidate CLI, consecutive self-build,
clean-machine or hosted same-commit claim is made by these tests.

The combined Windows replay passes in 184.063s with
`FAIRYC_RUN_PARITY_INTEGRATION=1`:

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidate(GenericAggregatePlan|GenericAggregateTypes|GenericAggregateSignatures|AtlasTypes|GenericTransportAdmission)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_types_test.go internal/cli/candidate_generic_aggregate_signatures_test.go
git diff --check
```

All selected tests ran; no selected native test was skipped. The final focused
signature replay above supplements its initial signature fixture with the last
scalar-handle controls. Linux remains deferred at the user's request.

## Concrete field and finite-layout contracts

The shape builder now joins kind-20/21 instances to exactly one retained prepared
instance and the current original source. It preserves concrete field types,
order, names, raw spans, source field indices, resource abilities and cleanup
declarations. Invalid IDs, changed source, altered instance records and exhausted
work reject before publishing shape or field records in the refusal controls.

An explicit aggregate field-contract route rejects borrowed stored fields,
resource ownership inside an inert struct, unsupported array elements and
recursive by-value layouts. Layout traversal follows records, Option/Result,
tagged payloads and arrays; raw pointers terminate traversal. The existing
ordinary entry points and array scalar boundary remain unchanged. These are
finite structural layout checks, not target size/alignment/offset evidence.

For generic `@repr(C)` declarations, `repr_c_declarations::read_template` checks
the original marker, attribute arity and C names. The shape owner then checks
the authenticated concrete field value lane. The ordinary `read` and `layout`
routes remain unchanged and cannot erase an unresolved template's parameters.
No generic aggregate foreign-call or C ABI admission is claimed here.

`TestCandidateGenericAggregateLayouts` runs fourteen three-package inputs in
actual MSVC low/optimized processes with empty PATH. Nine accepted variants use
Go `AnalyzeGenericPackages` through Flow and detached concrete Go fields as
independent expectations. They cover module/owner reorder, nested structs and
resources, Option/Result, scalar and record arrays, a nominal raw-pointer cycle,
repr(C), a zero-field resource and a second scalar resource instance. Four
source inputs reject in Go and Fairy (hidden resource ownership, borrowed field,
duplicate field and an unsupported f64 array element). The remaining direct
by-value recursion input passes Go Flow but must independently reject in the
Fairy finite-layout contract; it is not counted as a Go rejection comparison.

Native guards also check source drift, invalid IDs, forged instance kind, work
exhaustion, the unchanged ordinary generic-declaration gate, ordinary repr(C)
template refusal, malformed repr arguments and reserved C member names.

The focused Windows probe passes in 23.90s:

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateGenericAggregateLayouts$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_layouts_test.go
git diff --check
```

Initial harness failures were corrected before this pass: Flow was incorrectly
treated as a Go layout-cycle oracle, the type renderer read the frozen signature
prefix instead of the working shape graph, and one guard overlapped mutable and
shared graph arguments. Go's unsupported f64 array control also preserved the
existing array-element boundary rather than widening it. A generic repr(C)
declaration refusal exposed the source-T versus concrete-field join fixed above.

This completes the field/shape prerequisite only. Actual Interfaces construction,
constructor demands, cleanup specialization, ownership/Flow/MIR/C execution,
interface/cache/Atlas and full commands still need connection. Native Linux,
self-build, clean-machine distribution and same-commit hosted CI remain open.
No stable public API/SDK ABI, DB, dependency, hygiene or runner change.

The final combined Windows replay passes in 168.073s with all five selected
roots and both MSVC profiles executed; none is skipped:

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidate(GenericAggregate(Plan|Types|Signatures|Layouts)|AtlasTypes)$'
```

The ordinary Atlas type matrix includes repr(C), resource/enum fields,
Option/Result, nominal/raw-pointer types and arrays, exercising the preserved
ordinary route alongside the new private field/layout entry points. Scoped vet,
Go formatting and diff checks pass. No full candidate CLI, Linux, self-build,
clean-machine or hosted claim is made by this replay.

## Actual Interfaces, constructors and cleanup specialization

`module_unit_interfaces::build` and `build_unit_seeded` now consume the private
aggregate-capable signature owner. The source join has an explicit aggregate
route, and binding preserves struct as well as resource generic parameters.
The ordinary source join and ordinary signature entry remain closed for user
aggregate templates. Source/current-bound validation still precedes publication.

The new `module_unit_aggregate_plan` matches the original module/declaration and
ordered scalar arguments against the retained sealed source plan. Constructors
join exact prepared source uses rather than being instantiated as functions.
Ordinary calls retain an aggregate type slot; specialized function and cleanup
dependencies retain the same type in their existing dependency records. Body
type requests independently preserve the record expression and selected context.
Missing contexts/constructor facts reject instead of adding an orphan instance.

Resource instances seed cleanup preparation with their exact retained argument
tuple, abilities, effects and source coordinates. Generic cleanup helpers now
receive forwarded T arguments. A private scalar-only argument resolver handles
these resource contexts while rejecting composite arguments and rolling back
temporary graph nodes. Ordinary and authenticated SDK-vector argument resolution
keep their previous route. Stored-field contracts run before Interfaces escapes.
Field queries recognize both aggregate kinds and retain resource field privacy.

`TestCandidateGenericAggregateInterfaces` reuses the independent Go Flow/expanded
field oracle for the layout inputs and adds four accepted source variants:
ordinary construction, a generic factory called across packages, an extended
scalar factory argument and a constructor inside generic resource cleanup.
There are thirteen accepted and five refused source inputs in each native profile;
the direct by-value cycle remains a separate Fairy invariant, not a Go Flow
rejection. Actual complete-source Interfaces fields match the Go expectations.

Every accepted input also constructs both retained-import and compilation-unit
Interfaces for the root package alone. The factory root must retain exactly its
Cell/Holder shapes, factory/helper specializations and resource cleanup; unused
Pair shapes do not enter that selected unit. Constructor call/dependency facts
must join exactly one body-type request. Resource cleanup tuples and field
privacy are checked against the retained graph. Additional native controls
exercise the unchanged ordinary join gate, source drift, work exhaustion,
invalid scalar bindings, missing constructor facts and composite argument refusal
without graph growth.

Final focused Windows MSVC low/optimized probe passes in 38.26s:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateGenericAggregateInterfaces$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_layouts_test.go internal/cli/candidate_generic_aggregate_interfaces_test.go
git diff --check
```

The first execution exposed the existing source-join aggregate refusal; resource
argument forwarding also required the private resolver above. A reserved-word
variable in the final test guards was corrected before the final pass. Earlier
failed runs are not counted as passes.

Interfaces preparation does not check construction values or resource body Flow,
execute cleanup, lower/verify aggregate MIR, emit/link aggregate C, or provide
cache/Atlas/full CLI acceptance. Those remain required next steps under the same
objective. Linux stays batched. No stable API/SDK ABI, DB, dependency, hygiene or
runner change; no push or hosted dispatch.

The final combined Windows replay passes in 277.633s. All seven selected roots
execute with no skips, including both native profiles and the three legacy
scalar-argument/concrete-type/signature-diagnostic fixtures:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidate(GenericAggregate(Plan|Types|Signatures|Layouts|Interfaces)|AtlasTypes|GenericTransportAdmission)$'
```

This covers final production and fixture inputs. Scoped vet, formatting and diff
checks pass. Full candidate CLI, Linux, consecutive self-build, clean-machine
distribution and hosted same-commit acceptance remain separate open gates.

## Construction values and type-level drop obligations

The existing call-closure and value owners now consume kind-20/21 concrete
aggregates. Ordinary constructors retain their selected aggregate seed; concrete
function and cleanup contexts use the existing dependency seeds. Record checking
validates every specialized field, rejects duplicate/missing/unknown initializers,
and preserves resource construction privacy. Record arrays admit the same inert
field lane as ordinary records, without admitting new scalar-array types.

Generic resources now retain their real type-level drop obligation, cleanup
context, effects and abilities. Their fields preserve the existing exclusive
borrow restriction. Whole-owner borrows and explicit same-module burn use the
ordinary checks; a drop-enabled resource still cannot burn. These are value and
drop-plan facts, not executed cleanup or ownership/Flow acceptance.

`TestCandidateGenericAggregateValues` runs thirty-five three-package inputs in
actual MSVC low/optimized processes with empty PATH. Twenty accepted sources pass
Go `AnalyzeGenericPackages` through Flow, while Fairy runs its actual call closure
and value checking. Fourteen sources reject in both compilers. The direct
by-value layout cycle remains the separate Fairy layout invariant inherited from
the earlier probe, not a Go Flow rejection. Detached Go shapes/fields remain the
independent source/type oracle.

The matrix covers ordinary/imported constructors, generic factories, nested
constructors in cleanup, reordered fields, generic record arrays, f64 transport,
whole-resource shared borrows, mutable struct borrows, and explicit burn. Integer
endpoints -2147483648 and 2147483647 accept; the immediately adjacent out-of-range
values reject. Missing/duplicate/unknown fields, wrong/nested-wrong field types,
private resource construction and exclusive resource-field borrows reject. The
probe requires checked calls/arguments/returns, concrete local types, original
constructor spans/types, and complete resource drop contexts.

The focused Windows probe passes in 54.84s:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateGenericAggregateValues$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_values_test.go
git diff --check
```

The initial probe exposed omitted resource drop classification before these
changes. A later burn fixture initially exported a private type; its declaration
was made explicitly public before the final pass. Those failures are not passes.
The previous smaller matrix passed before expansion and is superseded by the
thirty-five-input replay above.

Move/loan analysis, concrete cleanup execution, verified executable MIR, C
emission/linking, interface/cache/Atlas production and actual aggregate commands
remain required follow-ups. Linux stays batched; self-build, clean-machine and
same-commit hosted acceptance remain open. No stable API/SDK ABI, DB, dependency,
repository-hygiene or runner change; no push or hosted dispatch.

The final five-root Windows regression replay passes in 212.548s with
`FAIRYC_RUN_PARITY_INTEGRATION=1`, with no selected skips:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidate(GenericAggregateInterfaces|AtlasTypes|GenericTransportAdmission|StaticOperationIdentity|CopiedClosureIdentity)$'
```

Both native profiles execute where provided; the legacy scalar admission group
also runs its three authored fixtures. Scoped vet, formatting and diff checks
pass. This supplements the final construction-values replay above without
claiming ownership/Flow or executable aggregate acceptance.

## Actual ownership/Flow and cleanup-policy admission

Kind-21 resource instances now enter the existing move-only classification.
Option/Result wrappers inherit that ownership through the same walker. Their
cleanup requests also enter both no-drop blockers and declared cleanup-effect
requirements, including nested owning fields. Struct instances remain inert;
specialization does not grant resource copy or partial-move permission.

`TestCandidateGenericAggregateFlow` runs the actual
`module_unit_checked_bodies::build` factory, not isolated value or source-plan
results. That owner checks values, moves, origin/loan diagnostics, cleanup
abilities/effects and Result handling before retaining the checked body. The
test helper is shared with the value-only probe, while the two authored Fairy
assertion modules remain independent.

The matrix contains fifty-two three-package sources per native profile: the
complete thirty-five-input value matrix plus seventeen ownership/policy inputs.
Twenty-five accepted sources independently pass Go Flow. Twenty-six sources
reject in Go and Fairy; the remaining direct layout cycle retains the earlier
separate Fairy invariant. Positive cases cover scoped resource locals, ownership
transfer into calls, nested and Option-wrapped resources, declared cleanup
effects and the earlier explicit burn and borrow lanes. Negative cases cover
reuse after transfer, double consumption, reads through a moved owner, duplicated
resource fields, partial field moves, inconsistent branch state, a consumed
cleanup receiver, implicit destruction of direct/nested no-drop resources,
missing cleanup effects and an escaping local borrow.

Independent fact assertions require two distinct named locals to retain exactly
two source-linked scope-cleanup obligations. A consumed call parameter retains
its obligation; the transferring caller and returning owner retain none for the
transferred value. These are semantic cleanup requests, not backend drop-order
or executed-cleanup observations. Fresh checked-source validation succeeds and
its separate 16 MiB source-work limit rejects exhaustion.

The corrected probe fails at the two-obligation control with the prior production
classification. During fixture preparation, cleanup effect placement, the source
validation budget and explicit consuming-operation controls were corrected
against the existing contracts; earlier failures are not passes. A focused
Windows low replay then passes in 29.23s.

Final Windows MSVC low/optimized replay passes in 139.713s, with no selected
skips. The Flow probe passes in 83.12s and the shared-harness value regression
passes in 56.47s:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregate(Flow|Values)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_values_test.go internal/cli/candidate_generic_aggregate_flow_test.go
git diff --check
```

Scoped vet, formatting and diff checks pass. Both probes execute with empty PATH.

Concrete cleanup execution, aggregate MIR/C, interface/cache/Atlas production and
full commands remain required. Native Linux, consecutive self-build, distribution
and hosted same-commit evidence remain open. No stable API/SDK ABI, DB,
dependency, repository-hygiene or runner configuration change.

## Source-independent aggregate cleanup MIR

The checked-body lowerer now retains kind-21 resource recipes, fields, abilities,
effects and exact cleanup-frame receiver types in the existing source-independent
cleanup MIR. Its private aggregate constructor admits concrete struct/resource
descriptors and backwards scalar-argument tuples of length one through eight.
Ordinary, project and artifact reconstruction remain explicitly closed for those
types; the private mode is not silently serialized or inferred during restoration.
The private cleanup MIR contract documents this boundary.

Tuple metadata cannot escape as a frame result/receiver, resource field, operand,
array or tagged/borrow payload. Canonical identities, frame/effect joins, nested
drop requirements and existing work/table bounds still run before publication.
Original declaration arity remains a checked-source fact; this detached owner
does not invent it after releasing the catalog.

`TestCandidateGenericAggregateCleanupMIR` reuses all fifty-two Flow inputs and
adds an eight-scalar-parameter aggregate, for fifty-three source inputs per
profile. Twenty-six accepted inputs pass independent Go Flow/expanded-field
expectations; twenty-six reject in both compilers and the direct by-value cycle
retains its independent Fairy layout refusal. The native probe compares every
semantic drop request's operand/frame/type/span/exit and all generic resource
recipes/field order against the retained checked owner.

An unchanged aggregate reconstruction succeeds. Thirteen malformed or ordinary
reconstruction controls reject: ordinary/project/artifact gates, self-referential
tuples, a non-tuple instance argument, lost resource glue, tuple results/receivers,
tuple fields/operands, Option/array tuple payloads and excessive tuple depth.
Field/operand mutations run when those tables are present; the matrix includes
both nonempty tables and zero-field resources.

The final focused Windows MSVC low/optimized replay passes in 72.19s with no
selected skips:

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateGenericAggregateCleanupMIR$'
```

The initial fixture used a reserved variable spelling; it was corrected before
the native passes. Earlier failures are not passes. The smaller low replay is
superseded by the final thirteen-control matrix above.

This completes cleanup metadata lowering/verification only. Whole-body aggregate
MIR, source-bound identities, CFG/drop/defer sequencing, both C emission forms,
actual cleanup traces, interface/cache/Atlas, full commands and final acceptance
remain required. The overall implementation queue remains active; Linux stays
batched. No stable API/SDK ABI, DB, dependency, hygiene or runner change; the
private cleanup MIR architecture contract is updated explicitly.

The ordinary cleanup MIR fixture's four authored checks are now exposed through
`TestCandidateOrdinaryCleanupMIR` on both Windows profiles. Its first run exposed
a stale 4 MiB project-work expectation, while `candidate_limits::project_work`
already specifies 16 MiB in unchanged production code. The fixture now reads the
owning limit, checking one below, exact and one above, while preserving the
unit-versus-project and malformed-handoff controls. Product work/time limits are
unchanged. This is a test correction, not a relaxed production gate.

The combined ordinary/Flow command initially failed only in that old fixture;
its unchanged fifty-two-input Flow root passed both profiles in 74.47s. That
result is reused only for those unchanged inputs. The final ordinary-fixture
replay, including the one-below-boundary control, passes in 46.26s with both
profiles and all four authored checks executed:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateOrdinaryCleanupMIR$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_flow_test.go internal/cli/candidate_generic_aggregate_cleanup_mir_test.go
git diff --check
```

Scoped vet, formatting and diff checks pass. There are no selected native skips.
These separate valid results do not turn the earlier failed combined command
into a pass. Whole-body MIR/C, restoration/identity/Atlas/full commands and the
remaining queue stay open.


## Owned aggregate body MIR

The existing `module_unit_body_ir_lower::lower_checked` now has explicit aggregate
acceptance through the verified cleanup owner. Detached bodies preserve concrete
frame context, nodes, locals, calls, field identities and layouts without retaining
parsed source owners. Kind-22 argument tuples remain identity metadata and cannot
be expression results/expectations/cast targets, builtin element types, local
values/expectations or invocation argument types. Shared layout, field, callable
and module-coordinate checks retain that separation as well.

`TestCandidateGenericAggregateBodyMIR` reuses all fifty-three cleanup/Flow inputs:
twenty-six accepted source variants, twenty-six paired Go/Fairy rejections and the
independent Fairy by-value-cycle refusal. It checks the ordinary whole-body owner
against the retained checked values and shapes. Unchanged reconstructed bodies
pass; eight tuple-in-value mutations, invalid frame IDs and exhausted body work
reject. Argument mutations run where calls have arguments; zero-field resources
also remain in the accepted matrix. No ordinary or artifact cleanup gate is opened.

The initial low replay reproduces the missing guard: an expression result changed
to a tuple is accepted with code 0. The corrected verifier rejects it with 1001.
Test preparation also fixed selection of empty cleanup receivers as expression
mutation targets, and an explicit u32 expected-code annotation. These failed
commands are not passing evidence. Diagnostic labels identify body stage/mutation.

`TestCandidateOrdinaryBodyMIR` runs all five existing authored ordinary-body checks:
resource/call frames, literals/assignment/for links, specialized partial cleanup,
malformed tables/work and UTF-8/C-string validation. Together with the unchanged
aggregate cleanup/Flow roots it is the focused regression for this verifier change.

CFG placement, actual drop/defer traces, both C emission forms, source-bound
identities/restoration/cache/Atlas, actual commands and all remaining queue gates
stay open. Native Linux remains batched. No stable API/SDK ABI, DB, dependency,
repository hygiene or runner policy changes. Final command results follow below.


Windows MSVC low/optimized final-input results (no selected skips): aggregate
body MIR 78.81s; unchanged aggregate cleanup MIR 75.92s; unchanged aggregate Flow
70.55s. Their combined command exited 1 because the added ordinary test's isolated
module copy omitted `module_unit_body_ir_lookup`; this is not a passing combined
command. After adding that dependency, the five-check ordinary body replay passes
both profiles in 50.93s. The other three results are reused for unchanged inputs.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidate(GenericAggregateBodyMIR|OrdinaryBodyMIR|GenericAggregateCleanupMIR|GenericAggregateFlow)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateOrdinaryBodyMIR$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_body_mir_test.go internal/cli/candidate_generic_aggregate_cleanup_mir_test.go
git diff --check
```

Scoped vet, formatting and diff checks pass. Native probes use empty PATH.
No full candidate CLI, Linux, self-build, distribution, clean-machine or hosted
same-commit result is claimed. The complete implementation queue remains active.


## Single-buffer aggregate C execution

`module_unit_c_aggregate` now gives kind-20/21 instances distinct private C record
names, forward declarations, recursively defined concrete fields, ordinary
constructors and kind-21 cleanup recipes. Arrays may contain admitted inert
kind-20 records. Scalar-only arguments and existing field/ownership restrictions
still come from the checked graph; kind-22 tuples have no C value spelling.
The existing checked body-to-C entry revalidates bodies and lowers/verifies CFGs
before emission. No parser, AST join or reference frontend substitutes for it.

The C probe contains the fourteen baseline field/layout inputs plus six executed
programs: constructed copies, two scalar specializations, repeated record arrays,
nested cleanup/defer, partial construction with Result propagation and loop
continue/break cleanup. Fifteen accepted inputs emit C; four paired source
rejections and the independent Fairy by-value-cycle rejection emit no program.
The baseline entry returns zero and does not execute every unused declaration.
The additional six entries return 42 and assert values or exact cleanup traces.

Each emitted single-buffer C program is consumed unchanged by the existing private
candidate C builder at low/optimized. The inert header/literal companions add no
program behavior. This uses the existing candidate consumer policy, not the stable
single-unit `/W4 /WX` API: an initial attempt with that API reports pre-existing
unreferenced-label and checked-helper narrowing warnings. No compiler flag,
warning suppression, product timeout or runner policy is changed to obtain a pass.
Strict standalone C-warning cleanup is not attested by this private execution path.

Both Go-generated and Fairy-generated native programs independently check fixed
expected return values and exact comma-separated observer bytes. The partial
construction case exposed the Go MIR defect fixed separately in `74240245`; it
is kept as a native regression, not removed from the matrix. The initial observer
used CRT newlines and was changed to commas so bytes do not depend on text-mode
newline translation. The original Go entry lookup was also corrected to use the
actual package-qualified module identity. Earlier failed commands are not passes.

Partitioned output, source-bound identities, project linking/restoration/cache,
Atlas/SDK and actual `check/build/run/test` remain separate required steps. No
stable source/FFI ABI or cache wire promotion follows from private C execution.
Native Linux, self-build, distribution and hosted CI remain open. Final Windows
command results follow below.


The final twenty-input Windows MSVC replay passes in 146.16s, with candidate
low/optimized hosts (35.79s/93.44s) and no selected skips. Fifteen Fairy C programs
run under two consumer profiles for each host: 60 actual native executions.
The six Go reference programs separately run under both profiles: 12 executions.
All 72 match fixed expected exit/output bytes. Each candidate source also passes
its independent Go Flow/expanded-field oracle or the specified rejection control.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregateC$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_c_test.go internal/cli/candidate_generic_aggregate_values_test.go
git diff --check
```

Scoped vet, formatting and diff checks pass. The C emission/source programs and
observer bytes are final inputs of the command above. This closes the bounded
single-buffer native aggregate step, including the named CFG cleanup traces.
Partitioned emission, broader CFG/resource forms, source identities/restoration,
cache/Atlas/SDK/full CLI and every final acceptance family remain open. The
strict standalone C-warning attempt remains failed/unclaimed; it is not erased
by the private consumer pass. Native Linux remains deferred at the user's request.
No stable public API/SDK ABI, DB, dependencies, hygiene, runners or timeout policy
changed. No push, release, deployment or hosted dispatch.


## Partitioned aggregate C execution

The compiler-private `module_unit_c_emit::emit_aggregates` route now emits the
existing bounded shared header, shared literals and contiguous body units from
verified aggregate-enabled body IR. It retains the same CFG, SDK, call/drop
reachability and output/work accounting owners as ordinary partitioning. The
caller retains its checked input; an error returns no successful partial Emission.
The route does not admit source-authenticated project linking or artifact restore.
Ordinary partition/test/scoped entry points retain their old project gate.

`TestCandidateGenericAggregatePartitionC` replays the same twenty source inputs
as the single-buffer test. Fifteen accepted programs each emit two packets:
all frames split at one frame, and entry-reachable frames split at two frames.
The Fairy verifier checks exact range coverage, nonempty selected frames, unchanged
aggregate/project mode, and work limits. The independent Go packet decoder checks
markers, length/canonical framing, literal identities, ordered original frame
symbols and exactly one entry before native compilation.

Each accepted input also rejects zero/over-limit partition sizes, zero/out-of-range
or cleanup entry, ordinary test/partition/scoped aggregate admission, a plain program
at the private aggregate entry and kind-22 C value spelling. The four paired source
rejections and independent Fairy by-value-cycle refusal still produce no C packet.
Both output modes use the same fixed exit/output expectations as the single-buffer
and Go reference tests, including nested/defer/partial/loop cleanup order.

Windows results, no selected skips:

- Single-buffer regression: 149.55s; low 35.15s, optimized 98.70s. The product,
  generated Fairy probe, runtime and native fixture inputs are unchanged by the
  subsequent Go-only replay-helper extraction, so these results are reused.
- Partitioned low host: 104.40s total, 92.18s subtest; 60 consumer executions.
- Partitioned optimized host: 162.66s total, 149.81s subtest; 60 consumer executions.
- All 120 partitioned consumer executions match the fixed expected bytes/status.
  The separate single-buffer/Go regression accounts for 60/12 native executions;
  reference executions are not rerun or counted again by the partition-only roots.
- Scoped vet, packet/decode tests, Go formatting and diff checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregateC$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregatePartitionC$/low$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregatePartitionC$/optimized$'
go test -mod=readonly -count=1 ./internal/candidatebuild -run 'Packet|Decode'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_c_test.go
git diff --check
```

The standalone fixture parser helper/import precede its exported verifier and are
not copied by the native replay harness; their addition changes no generated probe
input. Private consumer flags, default timeouts and strict standalone warning
boundaries are unchanged. Source identities/restoration/cache/Atlas/SDK, actual
commands, broader CFG/resource families and every final acceptance gate remain open.
No native Linux/full CLI/self-build/distribution/clean-machine/hosted result is
claimed. No stable public API/SDK ABI, DB, dependencies, hygiene or runner changes;
no push, release, deployment or hosted dispatch.


## Aggregate unit source identities

The actual `module_source_specializations` bridge now selects kind-20/21 type
handles through their ordered scalar tuple, original module/declaration and the
existing completed source-plan selector. Before selection, every source-prepared
aggregate graph descriptor must match the typed graph prefix, including tuple
links. Merely matching a declaration cannot erase or exchange argument order.
Kind-22 tuples retain zero identity rows because they are metadata, not instances.
The ordinary SDK Vec selector is unchanged.

The existing `module_identified_ir` owner now retains aggregate type rows and can
bind each generic resource cleanup frame to exactly its receiver's source identity.
Its late-type guard rejects an unselected Vec or aggregate instance. Function and
cleanup rows retain their original declaration and specialization call endpoints;
ordinary rows stay zero. This is a source-bound compilation-unit owner, not linked
project identity, artifact admission, cache restoration or Complete/Atlas publication.

The first identity probe failed before execution because its isolated source copy
omitted dependencies. After selecting the existing C-emitter import closure and
using a separate template-catalog alias, the real low native probe reproduced error
9 at cleanup frame handle kind 4/index 2: its aggregate receiver had a zero type row.
The selector fix passed the low checkpoint in 27.72s. A subsequent full aggregate
replay passed in 78.33s, while the newly exposed ordinary fixture failed only on
missing isolated generic closure/JSON dependencies. Adding those test dependencies
makes its three original SDK Vec checks pass both Windows profiles in 36.18s
(low 6.79s, optimized 24.86s). None of the failed combined commands is a whole pass.

The aggregate verifier uses the same twenty-input Go Flow/expanded-field oracle:
fifteen accepted inputs, four paired rejections and one independent Fairy layout
refusal. It compares every aggregate row with the retained source-plan node and
its original declaration, checks zero tuple/ordinary rows, exact generic cleanup
receiver identity and each concrete/ordinary call target. The shared field oracle
independently compares source-plan `g0_` bytes with Go expansion. Exact 16 MiB work,
one extra work unit, already-exhausted work and otherwise identical source bytes
with one trailing newline check bounded current-source refusal. This is not a new
Go/Fairy whole-project artifact or full CLI comparison.

A final aggregate replay after extending the late-instance guard follows below.
The full implementation queue remains active. Native Linux, project linking,
restoration/cache/Atlas/SDK production, actual commands and final self-build,
distribution/clean-machine/hosted acceptance remain required. No stable public
API/SDK ABI, DB, dependency, hygiene, runner or timeout change; no push or dispatch.


Final aggregate replay passes in 79.73s (low 12.25s, optimized 56.07s), no selected
skips. The ordinary three-check SDK Vec replay above uses unchanged generated
inputs and is reused; it does not import the extended identified-unit late guard.
Scoped vet, formatting and diff checks pass on the final files.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateGenericAggregateIdentities$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateOrdinarySourceIdentities$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_identity_test.go
git diff --check
```

Only the bounded source-bound unit identity step is complete. Project type/frame
rebasing and linking, exact current-source restoration/cache, Atlas/SDK and actual
commands remain next; the complete implementation objective is not finished.


## Aggregate project linking and named partitioned C

The existing identified-unit builder now also accepts an explicit tighter source
partition bound while preserving ordinary grouping, checking, capacity retries,
current-source provenance and whole-plan publication. Its existing `build` entry
uses the unchanged maximum-source bound. The aggregate probe forces three
independently checked source units; it does not substitute one joined body for
project linking.

Project type interning now remaps both links of kind-22 ordered scalar tuples.
The regression constructs two source-prefix-valid graphs with different local
IDs for the same two-element tuple and verifies shared canonical head/tail IDs.
An extra Option pad makes an unchanged local second link point at the wrong
kind, so merely retaining the original tuple index cannot pass. Array lengths
remain lengths and raw-pointer bits remain bits.

Project cleanup reconstruction retains kind-21 resource recipes, exact rebound
cleanup receivers/field types and the explicit aggregate admission mode through
`assemble_project_aggregates`. Ordinary `assemble_project` and artifact constructors
remain aggregate-closed. Project identity merging accepts exact identified
aggregate type rows alongside SDK Vec rows; concrete functions and cleanup rows
still merge through the existing original-source/frame/receiver checks. The
normal named project C emitter then consumes that linked owner.

The first low replay fails at checking stage 6 with error 1001. Per-source units
retain global signature types, including generic resources not carried by any
selected body. Cleanup lowering already permits missing storage for unused
ordinary/Vec resources, but omitted kind 21. It now applies the same liveness
rule to kind 21; a resource carried by a body still requires its actual layout.
The fix preserves the existing used-type traversal and rejects used missing shapes.
The failed command is not counted as a pass.

The same twenty-input source/Go Flow/expanded-field matrix has fifteen accepted
programs, four paired rejections and one independent Fairy cycle refusal. Each
accepted program verifies source-unit ranges and linked aggregate/project mode,
nonempty aggregate identities, exact cleanup receiver identity and specialized
call targets. Its two named C packets select all bodies split at one frame, or
entry-reachable bodies split at two. Native consumers check fixed values and exact
nested/defer/partial/loop cleanup bytes under unchanged private C flags.

The corrected low command passes in 104.29s (89.64s subtest), producing 60 native
consumer executions. Ordinary type-graph regressions pass both Windows profiles
in 53.46s (9.20s/36.18s): independent units, reordered wrappers, array lengths,
256/257 units, 4096-node exhaustion and signature-prefix refusal. Ordinary raw
project-link regressions pass both profiles in 77.14s (15.16s/47.40s), retaining
missing/duplicate ordinary-body refusal and four native generic-namespace runs.
Those two roots complete together in 130.709s without selected skips.

The Go-only native roots now explicitly skip unsupported hosts before resolving
MSVC; Linux execution remains deferred, not a claimed pass. Their Windows
prepared-source/native-consumer inputs are unchanged by this host-selection guard.
The final optimized aggregate results and commands follow below.

This closes bounded source-authenticated unit-to-project linking and named
partitioned aggregate C. Complete public interfaces, current-source restoration,
persistent caches, Atlas/SDK production and actual CLI commands remain required.
Neither a wire/schema nor artifact admission, stable API/SDK ABI, DB, dependency,
hygiene, runner or timeout policy is changed. Full CLI, Linux, self-build,
distribution/clean-machine and hosted CI remain open; no push or dispatch.


Optimized aggregate replay passes in 151.22s (135.85s subtest), no selected skips.
All 120 aggregate project C consumer executions pass; the separate ordinary
project regression accounts for four more, not a Go reference or Linux result.
Scoped vet, formatting and diff checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregateProjectC$/low$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateGenericAggregateProjectC$/optimized$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateOrdinaryProject(Types|Link)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_c_test.go internal/cli/candidate_generic_aggregate_project_test.go
git diff --check
```

The Go shared replay helper selects the original single/partition source inputs
unchanged and adds the project case. Its new host guard changes no Windows
prepared/native inputs. The full implementation queue remains active: interface
publication, restoration/cache/Atlas/SDK/actual commands and all final gates are
still required. Existing strict standalone warning limitations remain separate
from this unchanged private native-consumer acceptance.
