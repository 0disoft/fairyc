# ADR0551 unit Body IR closure C emission, 2026-11-08

The private unit C backend now transports the canonical kind23 closure leaf by
value. Its generated struct contains an invoke pointer and four inline i32
capture slots. No pointer references a creator frame. Ordinary local, parameter,
return and non-repr(C) struct transport reuse the existing aggregate copy path.
There is no allocation or cleanup obligation.

Construction kind36 emits a literal-specific adapter selected by caller frame
and construction node. `module_unit_body_ir_closures::helper` supplies the actual
checked helper frame, rather than interpreting a source declaration as a frame
number. Its safe, pure, nongeneric i32 helper proof remains owned by the Body IR
verifier. The adapter forwards inline captures in parameter order and calls the
existing checked helper body, preserving its checked arithmetic/trap emission.
Capture chains are newest-first; construction fills slots in written order and
zeros every unused slot. Operand snapshots are evaluated by the source/CFG owner;
C construction only copies their already-materialized i32 values.

Invocation kind4 uses the Body IR call record's node index and resolves the
original callable expression through `ir::node`. It invokes the materialized
callable and its inline capture array with the materialized i32 argument. The
CFG owner evaluates the callable first, then the argument, each once. Value
calls never enter declaration lookup or foreign-call lookup.

Entry reachability includes every selected construction's actual helper frame,
including a factory that returns its value without invoking it. Adapters live
beside their constructor bodies; selected helper declarations are available in
the shared header. Monolithic and partitioned emission share this implementation.
`emit_closures` explicitly selects the private closure cleanup family. Older
project and aggregate entry points retain separate admission. Body/cleanup
artifact versions, source publication, generic admission and FFI rules are not
extended. Closure-disabled paths do not scan adapters or closure construction.
Shared function emission now follows entry-reachable CFG blocks; unreachable
completion traps and their labels are omitted rather than suppressing warnings.

## Validation boundary

`TestCandidateClosureUnitCControls` checks original Go frontend controls,
including semantic admission of all 17 native inputs, and fixture syntax. `TestCandidateClosureUnitCSource` checks the authored Fairy
pipeline and backend imports in a temporary workspace; its lock generation is
confined to that temporary workspace. Source performs no native build or run.
The parent owns native compiler CPU and Native test scheduling.

The executable Fairy fixture exercises source-to-Body-IR-to-C generation for
captures 0..4, returned factories, local/struct copies, different helpers through
one parameter, mutable i32 snapshots, a checked divide-by-zero helper, an
uninvoked returned closure, and a plain nonclosure control. It checks monolithic C, selected partition output and
separate admission. Merely source-checking the fixture does not execute these
assertions or prove generated native behavior. The parent must run the fixture
and generated C controls before claiming native closure execution.

No language contract, public/SDK ABI, DB, dependency lock, repository hygiene,
runner or remote state is changed by this unit. No commit is made in this shared
checkout; the parent owns integration and commits.

Executed checks: `GOMAXPROCS=1 go test -mod=readonly ./internal/cli -run
'^TestCandidateClosureUnitC(Controls|Source)$' -count=1` passed, as did
`git diff --check`. Native fixture assertions and generated C execution were
intentionally not run under the parent CPU reservation.

## Authored Native harness (not executed)

`TestCandidateClosureUnitCNative` builds one low-profile, real Fairy producer
through `candidateParityBuildParts` and
`BuildWindowsMSVCCompilerHostPartsWithStackReserve`, retaining the existing
two-CL compiler-host bound. Consumer builds keep the ordinary target driver.
The producer
uses the source-bound checked-bodies owner and `lower_checked`, then emits each
original program as single-buffer C and an entry-selected partition with one
frame per unit. It publishes exact decimal-length-framed members on stdout;
failures report the case and phase to stderr. The partition member retains the
existing `FYCPARTS1` packet format, decoded by `candidatebuild.Decode` without
expanding artifact or source-owner admission.

Seventeen original-source programs cover returned factories with captures 0..4,
noncommutative four-capture order, local/struct copies after factory exit,
mutable snapshots, separate instances, different helpers through one parameter,
42 and 84 results, exactly-once argument side effects and repeated body locals.
The trap controls distinguish unused trapping bodies (42), invocation division
by zero (82), argument division before body overflow (82), ordered body locals
(82), body overflow (81) and signed division overflow (83). Existing scalar
runtime trap categories are reused through `cruntime.ScalarSource(80)`.

For each program, the harness compiles both C forms under Windows low and
optimized consumer profiles, then uses `candidateAtlasProcess` to require the
independent expected exit code and empty stdout/stderr: 68 consumer builds/runs
when all pass, stopping at the first failed consumer.
String shape assertions are producer checks only; runtime success is logged
only after an actual generated executable returns the expected result. Source
checking does not execute the producer or these 68 consumers.

Final harness Source/Controls recheck passed in 35.388s with `GOMAXPROCS=1`
and the exact non-Native regex above. Scoped `git diff --check` passed. The
Native test was not started; the parent UnitBodyNative reservation was respected.

Compiler-host builder selection recheck: the same exact Source/Controls command
passed in 21.617s. Producer input, runtime support, 60-second per-command timeout,
16MiB stack reservation and two-minute producer execution timeout are unchanged.
Only the producer selects the existing two-CL compiler-host builder; generated
consumers retain ordinary target builders.

## Parent Windows native validation

`TestCandidateClosureUnitCNative`: PASS, package 140.466s, in the detached
`.cache/closure-validation-20261108` worktree at base `4bb341e6` with this slice
copied in. All 68 generated-C consumers (17 cases, two profiles, two C forms)
compiled with unchanged strict MSVC warnings and executed with the expected
42/84 result or 81/82/83 scalar trap code and empty stdout/stderr.
Log: `.cache/closure-unit-c-native-values-20261108.log` in the main checkout.

Earlier native attempts reproduced C4102 unused labels, C4702 unreachable
completion traps, C4244 implicit narrowing and C4101 unused place temporaries.
The emitter now computes reachable blocks in O(B+E), declares closure-family
temporaries only for reachable value-producing operations, and explicitly casts
the widened checked scalar result to its verified destination type. Arithmetic
checks still precede narrowing. Compiler warning and timeout policies are intact.
The failed logs are retained separately; they are not passes. A concurrent
unfinished artifact edit also interrupted one main-checkout attempt; the stable
worktree avoids mixing worker revisions into native input.

The tested input remains the unit source-to-C path. Project/source-bound restore,
cache, whole compiler/CLI/self-host, Linux and remote CI are separate work. No
normative source-of-truth, public API/ABI, DB, dependency, runner or workflow changed.
