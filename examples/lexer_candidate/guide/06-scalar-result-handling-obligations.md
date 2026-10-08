[Candidate guide index](../README.md)

## Scalar Result handling obligations

Version 0.40.0 tracks pending obligations for exact scalar Result parameters and
locals. Return, discard, exhaustive switch, `.is_ok`, `?`, local transfer and
by-value call arguments handle or transfer them. Reassignment checks the old
destination before producing a new generation; copyable payloads remain copyable.
Normal scope/function exits, explicit returns, branch joins, zero-iteration paths
and loop backedges cannot silently lose pending obligations. Private type problem
42 points to the earliest pending producer or unhandled evaluation. `.is_ok` owns
one HIR operand and emits the existing verified MIR tag test, with no payload read.

`result_flow_graph.fy` constructs a bounded control graph; `result_flow.fy` computes
reachable pending producers to a fixed point. Limits are depth 64, 8192 graph
nodes, 262144 analysis cells and 16777216 propagation-cell visits per function;
allocation/limit errors fail closed. Bare Result calls are checked even in a
function without Result locals. The existing conservative literal-condition and
ordinary-trap handling rules remain visible rather than being optimized away.

`tests/result_handling.fy` covers exact diagnostic spans, branch/loop obligations,
propagation exits and retained `.is_ok` HIR/MIR. `verify_result_execution.py
--must-use` checks both candidate images in all four semantic/HIR/MIR/C modes and
compiles their actual generated C through the shared native driver. The 29 normal
fixtures and 27 rejection fixtures passed 449 CLI observations and 58 low/optimized
native executions. Both profiles passed check/build and seven selected native API
tests each. The 237 earlier positive Result fixtures pass type checking after nine
fixtures received explicit handling at their previously unchecked positions.

The initial verification exposed one reference discrepancy. For
`fn Result<i32,bool> f(Result<i32,bool> r){let Result<i32,bool> other=.ok(1);
let n=r?;discard other;return .ok(n);}`, Go Result Flow transfers only `r` and
accepts the source. The candidate also checks the implicit error-return edge and
rejects pending `other`; normal-path discard occurs after that edge. This follows
the candidate's reading of the earlier-return obligation in
`docs/language/0004-results-traps-and-control-flow.md`, but does not establish
differential equivalence with Go. The follow-up fixes Go Result Flow to check the
implicit error-return edge after transferring the operand and to preserve terminal
operand reachability. Six regression inputs, the related Go Result tests and all
24 Result fixtures pass. The old propagation fixture now explicitly handles its
copyable fallback before `?`; its exact source span expectation was updated.
All eight reference cases agree in both candidate profiles against a separately
built corrected Go compiler. The fixed bootstrap EXEs were preserved. Unit 11 is
complete; full compiler promotion still requires the later self-hosting gates.

Raw commands, the reachable discrepancy reproducer, source/SDK/bootstrap/image
digests and the existing validation-only release timeout overlay are retained in
`docs/ops/evidence/candidate-result-handling-20261007.json`; the correction and
comparison evidence is `candidate-result-handling-alignment-20261007.json`. Linux, performance,
self-hosting and the full repository suite remain deferred. Aggregate/resource
payloads and tagged storage remain units 13..20 or later. The remaining
work inventory is `docs/ops/candidate-remaining-work-20261007.md`.

## Nested Option and Result values

Version 0.41.0 extends the Fairy candidate to bounded combinations of Option and
Result with integer/bool leaves. Type identities are canonical across source
order; earlier scalar identities are preserved. Nested identities use a closed
30-bit prefix encoding with a separate marker, a resolver depth bound of 16,
and an encoding-size bound. Twelve successive Option wrappers fit; a thirteenth
does not. These bounds do not admit arbitrary recursive value types.

The owned HIR retains every instantiated child layout, including children only
spelled beneath another type annotation. HIR validates child identities and
acyclic layout dependencies; MIR retains nested constructors, copies, calls,
returns, switch payloads and Result propagation. Generated C defines children
before parents. Capturing a nested Result introduces a handling obligation that
must survive ordinary exits, early returns, break and continue.

The maximum-depth native case exposed MSVC C1054: implicit C aggregate zero
initialization recursively expanded inactive payloads beyond the C compiler's
initializer limit. Nested builtin locals and constructors now zero their scalar
storage with `memset`, set the tag, and assign the active payload from frozen MIR
operands. This applies only to nested integer/bool tagged layouts; pointer-bearing
or resource payload initialization is not admitted by this change.

`verify_result_execution.py --nested-tagged` compares type/HIR/MIR/C output in
development and release, then executes generated C under both MSVC profiles.
Fixtures cover nested success and error alternatives together, scalar boundary
values, copy order, nested propagation, capture handling, exhaustive patterns,
identity mismatches, void rejection and the encoding bound. Exact successful
observations and the initial backend failure are retained in
`docs/ops/evidence/candidate-nested-tagged-20261007.json`.

The final lane passes 52 source fixtures, 417 CLI observations and 78 generated
native executions, plus 16 native API tests across both candidate profiles and
six existing nominal/array/mutable-view sample executions. Seventeen selected Go
reference cases agree. The comparison exposed a Go Result Flow omission on
captured Results when break/continue exits their case scope; the reference now
checks exactly those exited edges, preserving captures across inner-loop exits.
Four rejecting controls and five handled/inner-loop controls cover that fix;
related semantic checks and the existing Result fixture manifest pass. The old
bootstrap images remain unchanged; the corrected reference is built separately.

Struct/diagnostic/resource payloads, aggregate tagged storage, broader type
identity representation and compiler self-hosting remain later units. Linux,
performance and the whole repository suite stay deferred to the owner's batch.

## Struct and diagnostic Option/Result payloads

Version 0.42.0 admits directly named same-file struct leaves in Option and Result.
Struct identity remains declaration-based: equal field shapes from different
declarations cannot be substituted. `nominal_identity.fy` shares name resolution
between ordinary structs and tagged leaves without adding a type-resolution
import cycle. Earlier scalar and nested-scalar identities are unchanged.

The closed prefix encoding adds a fourteen-bit nominal leaf escape for the 512
declaration slots. `Result<Token,Problem>` fits the thirty-bit encoding with both
struct alternatives; nested types still obey its size/depth bounds. For example,
`Result<Option<Token>,Problem>` needs thirty-two payload bits and is rejected.
This candidate bound is separate from the public Go compiler's accepted language.

Payload structs may contain the previously admitted scalar, fixed scalar-array,
and nested ordinary-struct fields. Empty structs work. Pointer/view-bearing
structs, resource payloads and tagged values stored in struct/array fields remain
later work. Zeroing these admitted nested layouts still addresses integer/bool
storage only, including its aggregate containers.

Owned HIR preserves nominal child layouts and exact capture spans after input
owners are destroyed. HIR rejects missing children or a different nominal payload
identity; MIR independently checks the extracted field, owner and destination
type. The existing constructor freezing, by-value copies, ordinary field access,
switch handling and `?` success/error propagation execute these nominal values.
Children are emitted before parent C records.

`samples/struct-payload-demo.fy` constructs both a Token and a source-span Problem.
Emit it with `build/lexer-candidate.exe --c-file samples/struct-payload-demo.fy`.
`verify_result_execution.py --struct-payloads --images <dev> <release>
--native-runner <runner> --receipt <path>` verifies the feature. Windows
check/build/native-test, both candidate profiles, generated-C execution and the
sample are recorded in `docs/ops/evidence/candidate-struct-payloads-20261007.json`.
The lane passes 27 fixtures, 217 CLI observations, 30 generated native runs,
16 native API tests and two sample runs. Existing nested regressions pass
417 CLI observations and 78 native runs. Go reference checks accept 15 programs
and reject six with diagnostics; missing-field and unknown-type controls return
its known experimental-subset unsupported status. Go semantic-only checking also
accepts a recursive value layout that the candidate rejects at its earlier
owned-layout boundary. These reference limitations are recorded separately,
rather than claiming exact diagnostic/phase equivalence for those controls.
Linux, performance, full-repository validation and stage1/stage2 remain deferred.

## Tagged struct fields

Version 0.43.0 admits same-file enum, Option and Result fields in the existing
copy-only struct subset. Construction, whole-struct copies, field reads and
writes, selected enum payloads, and Result propagation retain declaration-based
type identity. As in the Go reference, a tagged-containing struct permits direct
field replacement; deeper stores stay outside the plain stored-struct boundary.
Nested tagged values may be read and copied, edited in a local copy, then put back
by replacing the direct outer field. A field assignment evaluates its receiver without treating the
target Result value as an unhandled read; an ordinary unused Result field read
still fails the handling check. RHS effects on sibling fields remain visible.

`nominal_types.fy` follows nominal children through both Result alternatives and
Option payloads, rejecting direct and indirect by-value cycles. Owned HIR admits
the tagged child layouts and existing MIR verifies field owner and value types.
Generated C zeroes the admitted pointer-free aggregate locals with `memset`,
including structs containing deep tagged fields, to avoid MSVC's recursive
initializer limit. Resource and pointer-bearing layouts remain outside this
subset and need their own construction and ownership rules.

`tests/tagged_fields.fy` checks retained layouts, MIR field operations, recursive
types and unused Result reads. `verify_result_execution.py --tagged-fields`
uses `verify_tagged_field_cases.py` to observe both candidate profiles and actual
generated-C native execution. Its normal cases cover nominal enum identity,
copies, i64 values, nested ordinary structs, RHS sibling mutation, propagation
and a twelve-deep Option field; errors cover mismatched payloads, reversed Result
alternatives, missing fields and cycles. Checked arithmetic overflow remains a
native trap. Raw commands and tool/source/SDK/image hashes are recorded in
`docs/ops/evidence/candidate-tagged-fields-20261007.json`, including the existing
struct-payload regression and the validation-only bootstrap timeout overlay.

The field lane passes 20 fixtures, 161 CLI observations, 20 generated native runs,
four native API tests and fifteen aligned Go reference cases. Existing struct
payloads pass 217 CLI observations and thirty native runs. The initial reference
comparison exposed candidate overacceptance of a deep tagged field store; the
candidate now rejects it and a scalar deep store whose root also contains a tag.
The positive nested case uses the admitted copy/edit/replace path. Recursive
layouts and missing-field checks retain the separately recorded Go reference
limitations from the preceding unit.

This completes only the struct-field part of unit 14. Tagged array storage is
the next part; the progress ledger keeps unit 14 open until that part lands.
Linux, performance, whole-repository validation and stage1/stage2 remain deferred.

## Composite array type and owned HIR prerequisite

Version 0.44.0 extends the private candidate's array type and HIR paths to named
copy-only struct, enum, Option and Result elements. Existing scalar rank-one and
rank-two identities remain unchanged. Each dimension stays within 1..64; views,
resources and higher ranks are excluded. Composite rank-two arrays currently
require a written rank-one child type; fully inferred composite rank-two types
are not admitted by this representation.

`array_types.fy` reserves a disjoint identity range. Direct nominal/enum and
scalar-tag keys preserve declaration or payload identity; larger tags and
rank-two children use their first written type node. `hir_array_shapes.fy`
collects one owned element record per composite array, including inferred
rank-one arrays and unused declarations. The source-independent shape verifier
rejects missing child layouts, altered encoded element types and by-value cycles.
HIR construction/indexing uses these element records rather than a scalar-only
radix. A repeated Result initializer transfers its handling obligation to storage.

The ordinary source CLI accepts `--check-types-file` and `--hir-file` for the
cases in `verify_tagged_array_cases.py`. `--mir-file` and `--c-file` deliberately
return an error without stdout for this new representation: executable array
lowering and dependency-first C definitions are the next unit-14 part. Existing
scalar arrays and field behavior keep their executable paths.

`verify_tagged_arrays_hir.py --images <dev> <release> --receipt <path>` observes
25 source cases and 160 CLI results in both profiles. Four native API tests
exercise the Fairy type/HIR implementation, including retained element facts,
wrong/missing layouts, cycles and the temporary MIR gate; these are not native
executions of the array-element programs. Thirty-two existing field CLI
observations provide a narrow regression. Builds, locks, source/SDK/tool/image
hashes and raw commands are in
`docs/ops/evidence/candidate-tagged-array-hir-20261007.json`.

The pinned Go reference accepts the inferred nominal-array case. Tagged-array
read controls return its experimental subset-unsupported exit 2; store cases
instead report E1008. Those differences are recorded, not treated as equivalent
acceptance or a public Go language change. Unit 14 remains incomplete until the
candidate's composite-array MIR and native execution work lands.

## Composite array MIR and native execution

Version 0.45.0 connects the preceding owned array layouts to MIR and C execution.
Enum, Option, Result and copy-only struct elements support construction, repeated
initializers, indexing, assignment, value copies, calls and returns, and storage
in struct fields. Each dimension remains 1..64, with at most two dimensions.
Resource/view elements and higher ranks remain excluded. Rank-two identities now
pack both lengths and the child key, admitting fully inferred struct arrays.

MIR reads and stores use the owned element type. Checked indexes must dominate
access, retain the correct array length, and have one definition. The verifier
rejects attempts to reinterpret array construction or element metadata as nominal
struct operations. C definitions emit child layouts before containing arrays and
structs. Direct element assignments avoid recursive aggregate initializers that
can exceed MSVC's nesting limit. Repeated initializers evaluate once; ordinary
initializers keep written order. Bounds checks precede assignment RHS evaluation,
and stores preserve sibling edits made while evaluating that RHS.
Type-node keys use their own 4096-node bound rather than the 1024-entry
declaration conversion. A 350-field source with a late nested-tag array reproduces
the prior assertion exit and now checks, lowers and executes normally.

`verify_result_execution.py --tagged-arrays --images <dev> <release>
--native-runner <runner> --receipt <path>` checks source diagnostics, HIR/MIR/C
output and actual generated native programs. Test-only observation of emitted
`mark` calls measures initializer ordering without replacing product logic.
`tests/tagged_arrays_native.fy` additionally rejects forged element/index/nominal
operations before output publication and checks fully inferred rank-two layouts.
Builds, API regressions, raw commands, source/SDK/compiler and EXE hashes are in
`docs/ops/evidence/candidate-tagged-array-native-20261007.json`.

The preceding Go tagged-array subset limitations remain separately recorded.
This completes unit 14 after its execution evidence passes; resource identity and
ownership are units 15..20. Linux, performance and stage1/stage2 remain deferred.

## Resource type identity and owned HIR

Version 0.46.0 completes unit 15. Resource declaration identities use the disjoint
98304..98815 range; tagged payload encoding preserves their category separately
from inert structs. Owned HIR retains resource bindings and field layouts after
the parsed source owner is removed. Copyability, drop requirements and field
ownership follow potentially active Option/Result payloads. Inert structs and
arrays cannot hide resource ownership; missing children, forged ownership and
nonfinite by-value layouts reject.

This unit requires an explicit empty cleanup block. Missing/nonempty cleanup,
abilities and resource construction remain closed. Both ordinary lowering and
the MIR assembly API reject resource tables before the copy-only backend can
publish them. Native API tests execute compiler logic, not resource programs.
Construction HIR, moves and lifetime execution follow in units 16..30.

The maintained `verify_resource_types.py` observes 11 positive and 21 negative
fixtures in dev and release: 256 CLI observations. Five native compiler API tests
pass in each profile, and a focused prior array regression adds 57 CLI
observations and 10 generated native executions. Source, lock/SDK, bootstrap and
image digests plus raw commands are recorded in
`docs/ops/evidence/candidate-resource-types-20261007.json`.
Twenty-seven Go reference observations retain the cycle-acceptance difference
and fixed-array resource-field subset limitation separately. The final receipt
uses the current required-cleanup fixtures, replacing an earlier stale receipt
that included implicit cleanup. Linux, performance and stage1/stage2 remain
deferred.

## Explicit resource factory and construction HIR

Version 0.47.0 connects same-module resource record literals and ordinary
resource-returning factory calls to owned HIR. Factories returning Result keep
their success/error payload identities and explicit propagation. Construction
requires every declared field exactly once, with exact types and declaration
identity; written initializer order and field spans survive the input owner.
The existing record shorthand is unchanged. Resource names are not callable
constructors and no implicit conversion is introduced.

`tests/resource_factories.fy` checks initializer/call order, nested ownership,
both Result alternatives and six forged construction/layout mutations.
`verify_resource_factories.py --images <dev> <release> --receipt <path>` observes
the source fixtures through type/HIR/MIR/C CLI modes. Resource MIR/C publication
remains closed until movement and cleanup are connected. Cross-module resource
visibility and imported factories remain part of later module integration;
this same-module unit makes no imported construction claim.

The sample is `samples/resource-factory-demo.fy`. Final validation evidence is
`docs/ops/evidence/candidate-resource-factories-20261007.json`, including the
focused unit-15 regression and separate Go fixed-array resource-field limitation.

## Whole-root resource transfer HIR

Version 0.48.0 records typed whole-root ownership transfer as an owned `Move`
expression. Binding initializers, replacement RHS values, by-value arguments,
returns, record payload installation, consuming tagged switches, propagation and
resource-bearing Result `is_ok` retain the exact donor binding and source span.
Copy-field inspection and assignment places retain nonconsuming references.
The new source classifier uses a separate slot flag, preserving field/member
identity metadata. There is no source-level move keyword or implicit copy.

Partial resource-field transfers reject as E1103 at the consuming field span;
nested copy-field reads remain admitted. HIR assembly refuses Move records with
copyable types, missing/non-value donor bindings or forged operand metadata.
`tests/resource_moves.fy` exercises ownership classification after the source
owner is discarded and checks partial-field diagnostics.
`verify_resource_moves.py` observes source CLI acceptance and rejection in both
profiles. Evidence is `docs/ops/evidence/candidate-resource-moves-20261007.json`.
This is unit 17 transfer representation, not liveness or cleanup execution:
units 18..20 add use-after-move and branch/loop flow, and resource MIR/C remains
closed until the later cleanup units. Linux and stage1/stage2 remain deferred.

## Resource use-after-move and transfer provenance

Version 0.49.0 checks resource-root liveness after exact type/consumption
classification. A shared control-graph builder preserves ordinary evaluation,
returns, propagation error exits, lexical scope exit and capture installation
for both the existing Result-handling analysis and the new ownership analysis.
A moved root cannot be read, inspected through a copy field, returned or used
as a second argument/initializer. Fresh reassignment reinstalls a live value;
shadowed bindings keep separate identities and unreachable uses are ignored.

The bounded worklist retains Live/Dead reachability and the earliest transfer
span. E1101 diagnostics identify both the rejected use and original move.
`tests/resource_dead.fy` checks exact spans, replacement, shadowing, argument
order, propagation and consuming Result inspection. `verify_resource_dead.py`
observes the source CLI and requires the related move range in each rejection.
Evidence is `docs/ops/evidence/candidate-resource-dead-20261007.json`.
Unit 18 establishes root-use rejection. Units 19/20 add explicit rejection of
ambiguous branch joins even without a later use, and equality at loop back/exit
edges. The conservative dataflow union is not a runtime conditional drop flag;
resource MIR/C publication stays closed until later cleanup integration.

The pinned Go reference incorrectly accepted returning a resource-bearing Result
after extracting it with `?`. Its resource flow now consumes the operand on both
alternatives and checks cleanup effects at the possible error return. Focused Go
Resource/Result propagation tests and a newly built reference reproduce the
correct E1101 rejection; the candidate source still checks with that reference.
The receipt keeps original images and before/after tool identities separately.

## Resource branch joins

Version 0.50.0 marks `if`, `switch`, `&&` and `||` ownership joins in the
shared control graph. All reachable incoming root states must agree, even when
there is no subsequent read. A Live/Dead disagreement rejects E1104 at the join
and retains the original transfer range. Both-moved branches, fresh replacement,
branch-local owners and terminal alternatives are handled separately; an exited
path does not participate in the join.

`tests/resource_branches.fy` executes three compiler API tests per native profile
for exact join/move spans, equal or exited paths and short-circuit consumption.
`verify_resource_branches.py` checks 17 source fixtures across both profiles and
all four check/HIR/MIR/C observation modes (136 observations). The unit-18 lane
passes another 18 fixtures / 144 observations with the same images.
Evidence is `docs/ops/evidence/candidate-resource-branches-20261007.json`.

The Go reference previously treated a short-circuit right-hand side as always
evaluated. Its Resource and Result analyses now merge the skipped and evaluated
paths. Focused Go regressions pass, and a newly hashed reference agrees on all
17 acceptance/rejection controls. It can emit additional E4101 Result obligations
alongside E1104; this is not a claim of identical full diagnostic lists.
The receipt retains raw observations and the pre-fix tool identity.

Loop/backedge/exit equality remains unit 20. Resource source execution remains
closed before MIR/C publication; the native tests above execute compiler logic.
Linux, performance and stage1/stage2 promotion remain deferred.

## Resource loop edges

Version 0.51.0 records each reachable normal-body, `break` and `continue` edge
against the resource state immediately after its loop condition. Exited local
and capture scopes are killed before comparison, so newly created body owners
do not masquerade as mismatches of enclosing owners. Classic `for` continues
still reach the post clause; breaks skip it. Returns and propagation error exits
do not re-enter a loop. Restoring a moved owner preserves its live state without
requiring the same allocation generation on every iteration.

E1105 includes the responsible loop range, loop-header range and available move
provenance. A nested mismatch retains the smallest responsible loop for that
transfer instead of reporting only a contaminated outer fixed point. Reading a
condition-consumed owner on the next iteration remains E1101 at the exact use.

`tests/resource_loops.fy` executes five native compiler API tests per profile;
`verify_resource_loops.py` observes 33 accepted/rejected source fixtures in both
profiles and all four CLI modes (264 observations). Unit-19 branch and unit-18
use-after-move regressions retain their own raw observations with these images.
Evidence is `docs/ops/evidence/candidate-resource-loops-20261007.json`.

