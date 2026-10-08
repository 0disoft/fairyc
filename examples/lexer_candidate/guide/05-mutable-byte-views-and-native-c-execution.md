[Candidate guide index](../README.md)

## Mutable byte views and native C execution

Version 0.27.0 lowers exclusive byte descriptors, mutable slices and indexed
writes to verified MIR and private C11 transport. `struct fy_bytes_mut` has a
mutable byte pointer and an element count; shared descriptors keep const
pointers. A write changes the original array storage. Child writes therefore
remain visible when the parent and original owner resume after the last use.
Exact mutable parameters are passed as descriptors; a returned child retains
the inferred caller origin and suspension checked by HIR.

For an indexed store, MIR freezes the descriptor, evaluates/checks the index,
then evaluates the RHS and performs the write. Compound assignments load the
previous value before RHS effects, apply checked byte arithmetic and store only
on success. Slice bounds evaluate once, left to right, and trap before pointer
arithmetic unless `start <= end <= len`; an empty end slice is valid.

MIR separately validates mutable backing arrays, store value/descriptor types,
matching frozen descriptors and checked indices, definite initialization and
local/parameter return origins. HIR owns exclusive-generation conflicts;
MIR origin verification does not claim a complete source-independent mutable
loan analysis. Invalid MIR is rejected before C rendering changes the caller's
buffer. Public grammar, SDK and runtime contracts are unchanged.

`samples/mutable-view-c-demo.fy` updates a returned payload slice, resumes the
parent and verifies the original array. Emit it with `build/lexer-candidate.exe
--c-file samples/mutable-view-c-demo.fy`. `verify_c.py --mutable` compares both
candidate profiles with independent Go MIR interpreter outcomes and executes the
C in MSVC low/optimized profiles. The mutable lane covers 20 Go-oracle programs,
102 C CLI checks and 46 native executions; the sample adds two native runs.
Exact source/tool hashes, commands and native
observations are in `docs/ops/evidence/mutable-c-file-cli-20261006.json`.

Mutable-to-shared call conversion, mutable text, field/nested-array borrows,
view-bearing aggregates, retained mutable siblings and mixed/recursive return
origins remain closed. Linux native validation stays deferred to the requested
batch; no performance or self-hosting completion is claimed.

## Cost baseline

`scanner::storage` reports source, separately owned cooked and diagnostic-token
bytes plus record counts. It does not report allocator capacity or peak heap/RSS.
The opt-in `fairyc_go_measure_lexer_stream` lane records five native driver
roundtrips after one warmup for each scenario/profile; ordinary tests skip it.
See [the measured baseline](../cost-baseline-2026-09-10.md) for raw samples, exact
build identities, included overhead and the limits on interpretation.

The separate [SDK heap observation](../heap-baseline-2026-09-10.md) counts actual
allocator requests, peak simultaneously live requested bytes, retained capacity
and balanced destruction, including three injected failures. Its instrumented
builds are not latency benchmarks or RSS measurements.

The [sparse cooked-storage comparison](../sparse-storage-2026-09-10.md) preserves the
old baseline and records a 41.06% reduction in peak requested heap bytes for the
same token-heavy input, with a small allocation-count increase. The private token
layout changed; the Stream consumer API and logical payload metrics did not.

## Array fields and rank-two array HIR

Version 0.28.0 retains rank-one and rank-two scalar arrays in local values,
function signatures and acyclic struct fields. Each dimension stays 1..64.
Postfix type constructors apply left to right: `u8[2][3]` is three rows of two
bytes. Nested literals and repeated rows retain ordered, context-typed children;
repeated values are evaluated once and copied. Array IDs preserve existing
rank-one IDs and use a disjoint canonical encoding for rank two.

Field/index assignment paths must end at a mutable named local; let bindings,
parameters and temporaries stay immutable. Whole-row and whole-field replacement
require exact type equality, and compound operators require scalar integers.
`tests/hir_aggregate_arrays.fy` checks retained layouts and child types.
Rank three, arrays of structs/views and field/index byte-view borrows remain
closed. Version 0.29.0 connects these paths to native C execution below.

## Array fields and rank-two native execution

Version 0.29.0 lowers field/index paths through `mir_places.fy`. Index operands
are evaluated once from outer to inner and each dimension is checked before
later index expressions, reads or RHS effects. Named value paths read their
current root after index effects; temporary receivers retain frozen values.
Compound writes load the previous element before RHS evaluation and perform
checked scalar arithmetic. Stores then rebuild the selected path from the
current root, preserving other rows and sibling fields changed by the RHS.
Whole-row/array-field copies and function arguments remain independent values.

C array wrappers are emitted with inner types before outer types and before
structs, including types referenced only by unused struct declarations.
The writer keeps its 2 MiB output limit and admits at most 1024 distinct
rank-two wrapper types in a unit. MIR validation reuses exact aggregate operand
types, checked-index provenance and definite initialization for each layer.
`tests/mir_aggregate_arrays.fy` mutates row operands, bound provenance, update
values and field-read shapes; failed rendering preserves the existing buffer.

Emit `samples/aggregate-array-c-demo.fy` using `--c-file`. The
`verify_c.py --aggregate-arrays` lane uses explicit fixed expectations and
source assertions from `--aggregate-array-cases` in the test utility: 21
programs, 106 CLI checks and 48 native runs; the sample adds two native runs.
These are
not Go MIR interpreter outcomes: the Go reference does not admit rank-two source
types. The existing rank-one/nominal/mutable lanes retain their Go interpreter
oracles. Exact command, source, SDK/tool and native image hashes are recorded in
`docs/ops/evidence/aggregate-array-c-file-cli-20261006.json`.

Ranks above two, arrays of structs/views, and field/index byte-view borrows
remain closed. Public grammar, SDK/runtime and public ABI are unchanged. Linux
native execution remains deferred; this step makes no performance or completed
self-hosting claim.

## Text access diagnostic

Version 0.29.1 reports `text indexing and slicing are not supported` for a
`ref str` receiver, instead of the unrelated `invalid nominal type` message.
This follows the existing surface-syntax contract: text `.len` and `.is_empty`
remain byte metadata; only byte views support bracket indexing and slicing.
The private type diagnostic is 21 and spans the complete access expression.
Type, HIR, MIR and C CLI modes report the same error before publishing output,
including literals, local aliases, parameters and text-returning calls.

`tests/types.fy::text_access_has_a_distinct_diagnostic` exercises the type checker
directly. Exact CLI diagnostics, unchanged input bytes, both candidate profiles
and representative native byte/text-metadata results are recorded in
`docs/ops/evidence/text-access-diagnostic-20261006.json`. This is a diagnostic
correction, with no new text slice syntax, SDK API or runtime behavior.


## Payloadless enum HIR

Version 0.30.0 retains declaration-specific payloadless enums and exhaustive
shallow switches. Enums have 1..128 distinct members and private type IDs in
131072..131583, separate from integers, structs and fixed arrays. Contextual
`.Member` construction works in typed bindings, assignment, parameters, return
values and conditional/switch joins. Each switch lists every member exactly
once; written arm order and arm-local scopes survive HIR lowering. Positional
`.Member` arguments are distinguished from unsupported labeled call arguments.

`tests/hir_enums.fy` checks owned layouts and retained arm order after the input
arenas are released. `verify_enums.py --phase hir` checks accepted values and
rejected empty enums and unsupported payload field types, duplicate members/cases, missing/unknown cases,
integer coercion, distinct declarations, equality, aggregate enum storage,
immutable assignments and borrowed-view joins. Types, HIR, MIR and C execution
are separate candidate milestones; version 0.30.0 retains HIR only. That milestone excluded payloads;
version 0.32.0 adds scalar payload HIR below. Enum comparison and enum
arrays/struct fields remain closed. The public grammar,
SDK/runtime and public ABI are unchanged.


## Payloadless enum native execution

Version 0.31.0 connects enum values and switches to verified MIR and C11.
Operation 21 constructs a declaration-relative tag and operation 22 tests one
member; exact enum types remain distinct in copies, parameters and return values.
The private C representation is a separate struct per enum containing a
`uint32_t` tag. Source enum comparison stays unsupported. Switch subjects are
frozen once, only the selected arm executes, and live arm results join through
exactly typed locals. Returning arms have no fallthrough edge. Existing lexical
defer/loop cleanup applies to each arm; defer still captures scalar arguments,
and direct enum defer arguments are rejected explicitly.

`tests/mir_enums.fy` checks retained tags/branches and rejects forged tags,
wrong declarations/operands, non-bool tests and uninitialized values. Failed MIR
or C rendering preserves existing output bytes. `verify_enums.py --phase full`
checks all four CLI modes in both candidate profiles and runs emitted C in both
native profiles. Fixed expectations are independently compared with Go MIR
interpreter results produced by the test-only `--enum-mir` oracle. This work also
fixed Go MIR call lowering: earlier by-value enum arguments are now copied
before later arguments can overwrite their source local (including inert payload
enums). No parsing or lowering in the Fairy CLI is delegated to that oracle.

Emit `samples/enum-c-demo.fy` with `--c-file`. The evidence file is
`docs/ops/evidence/enum-native-20261006.json`. Payload enums, enum equality,
enum arrays/struct fields and borrowed-view switch joins remain outside this
candidate slice. Public grammar, SDK/runtime and public ABI are unchanged.
Linux native validation and performance measurement remain deferred. These
checks do not constitute completed compiler self-hosting.


## Integer and bool payload enum HIR

Version 0.32.0 extends the Fairy-written candidate's type checking and owned HIR
with named payloads of `i32`, `i64`, `u8`, `u32`, `usize`, and `bool`. Each enum
has 1..128 members; a payload member has 1..16 fields. The shared parser field
arena remains capped at 1024 entries, including member records and payload fields.
Empty payload braces, nested aggregates, resource/view payloads, positional
patterns, enum equality, and enum storage inside arrays/structs remain closed.

Construction uses contextual `.Value { .amount = n, .valid = true }`; every field
must appear exactly once. Written initializer order is retained, and literal
bounds use the declared field type. An exhaustive switch binds every field with
`case .Value { .amount = amount, .valid = valid }: { ... }`. Bindings are immutable,
scoped to that arm block, and can shadow outer locals. `_` discards one selected
field; it creates no local binding. Payloadless members retain their existing
construction and pattern syntax.

Owned shape records preserve payload ranges and scalar types after syntax/type
arenas are released. Variant edges contain ordered Initializer expressions; Arm
edges contain ordered Capture expressions. A Capture identifies its payload field
and has zero (discard) or one immutable Binding ID in its edge range. The private
payload dump is `fairyc.enum-payload-hir/v1`; its D rows add payload start/count.
Payloadless dumps keep their previous format and output bytes.

Version 0.32.0 initially rejected payload layouts at the MIR boundary. Version
0.33.0 connects these retained layouts to MIR and native execution as described
below. The original HIR-only evidence remains a historical record.
`tests/hir_enum_payloads.fy` checks retained data and malformed ranges;
HIR evidence: `docs/ops/evidence/enum-payload-hir-20261007.json`.
The public language grammar and SDK contracts are unchanged.

## Integer and bool payload enum execution

Version 0.33.0 lowers payload construction to operation 21 with declaration-order
operands, tag tests to 22, and scalar captures to 23. Initializers run in written
order and freeze each value before the next initializer. A switch evaluates its
subject once and copies only the selected arm's named captures. Early return
during construction leaves no unused parent edges. `_` never reads union storage.

The owned MIR dump is `fairyc.enum-payload-mir/v1`; D rows retain payload ranges.
Member tags are dense declaration ordinals excluding payload-field records.
The verifier checks owner, field membership/type, argument count/type, unique
edge ownership and definite initialization before either dump or C publication.
`tests/mir_enum_payloads.fy` includes malformed-operation and read mutations.

The C transport uses one struct per enum declaration, a `uint32_t fy_tag`, and
named payload structs inside a union. Construction, assignment, arguments and
return use value copies. Each capture tests the active tag before accessing the
union and calls `fy_runtime_trap(7)` on mismatch. This dynamic guard covers a
structurally valid MIR extraction whose active member differs at runtime.

Use `--mir-file` or `--c-file` on `samples/enum-payload-demo.fy`. As with the
existing C entry contract, native main returns i32; i64 payload computations
remain available inside functions. The sample asserts its i64 result then
returns 0. Run `verify_enum_payloads.py --images <dev-image> <release-image>
--phase full --native-runner <native-runner> --receipt <path>` for deterministic
HIR/MIR/C, malformed inputs, fixed expected native results, and a test-only
corrupted C active-tag case. The latter must trap 7 in both native profiles and
is explicitly not a valid Fairy source program. Raw CLI/native observations and
digests are in `docs/ops/evidence/enum-payload-native-20261007.json`.

Enum comparisons, nested/resource/view payloads, enum array/struct storage,
borrowed-view switch joins, complete self-hosting, Linux validation and
performance measurement remain outside this slice. Stable compiler, public
language/SDK APIs and runner implementation are unchanged.

## Scalar Option type, HIR and MIR

Builtin `Option<i32|i64|u8|u32|usize|bool>` uses contextual `.some(value)` and
`.none`, with exactly one `.some(binding)` and `.none` switch arm. `_` discards
the capture; named captures are immutable and confined to their arm. The six
canonical private IDs 196609..196614 are disjoint from user enum declarations.
Each used instance owns a synthetic, declaration-free shape (binding 0), with
one payload field and two members. Repeated spellings share the same shape.

HIR retains `Option` as a distinct type category; its readable tag is `option:N`.
The owned payload HIR/MIR wire formats remain `fairyc.enum-payload-hir/v1` and
`fairyc.enum-payload-mir/v1`. MIR operations 21/22/23 preserve construction,
tag tests and typed captures. Synthetic layouts are bounded by the same field
and shape arenas and validated before publication. See `samples/option-demo.fy`,
`tests/hir_options.fy` and `verify_options.py --phase mir`.

Evidence: `docs/ops/evidence/candidate-option-mir-20261007.json`. Native Option
acceptance is recorded below; structured/resource payloads and Result
remain subsequent units in the owner's numbered plan. Source grammar and SDK
contracts are unchanged.

`tests/mir_options.fy` independently forges 5 invalid builtin layouts and 15
invalid operations/reads. It checks different Option instances, wrong member
tags and field ownership, missing or mistyped operands, uninitialized reads,
and preservation of the output buffer when verification fails. Existing enum
mutation tests run alongside it. Evidence:
`docs/ops/evidence/candidate-option-verifier-20261007.json`.

## Scalar Option native execution

Option C transport uses private `struct fy_o_N` names with a tag and payload
union, preserving ordinary value-copy semantics in assignment, arguments and
returns. `.none` sets its tag without evaluating or extracting a payload; each
`.some` capture checks the active tag before a union read. Generated C is
identified as `fairyc.candidate-option-c/v1`.

`verify_options.py --phase native --images <dev> <release> --native-runner <runner>
--receipt <path>` compares type/HIR/MIR/C output bytes from both candidate images,
fixed native return values in both C profiles, source preservation and artifact
hashes. It also deliberately corrupts a generated tag and branch decision,
requiring the unchanged extraction guard to trap 7. This is test-only C fault
injection rather than execution of a valid Fairy source program. Raw evidence:
`docs/ops/evidence/candidate-option-native-20261007.json`. Linux runs remain
deferred to the consolidated validation batch.

## Option control flow

Version 0.33.1 preserves Option values through if expressions, loop assignments,
break/continue, calls and early returns. Switch subjects are evaluated once and
captured payloads remain independent of later subject reassignment. A no-tail
arm ending in literal `assert(false);` contributes no value to the join, as
required by the existing language contract; at least one arm must contribute.
Direct-return Option arms retain the existing rules rather than acquiring the
separate Result arm exception. No general never type is introduced.

`tests/mir_option_flow.fy` checks that early payload returns publish no orphan
construction/call edges and that literal-assert arms lower without value joins.
`verify_options.py` adds normal control-flow outcomes plus overflow, division,
payload assertion and assertion-arm traps. Both candidate profiles must emit
identical HIR/MIR/C; emitted code executes in both native C profiles. Evidence:
`docs/ops/evidence/candidate-option-flow-20261007.json`. Linux execution remains
deferred to the consolidated batch. Source language, SDK and runtime are unchanged.

## Result scalar type identity

Version 0.34.0 recognizes `Result<T,E>` for the six admitted integer/bool scalar
types in each position. The 36 ordered pairs use private IDs 262145..262180;
both the success and error type participate in identity. Repeated spellings
share an ID, and Option, user enums and structs remain distinct. Parameters,
locals, assignments, calls and returns preserve these types and reject a
different success/error component without implicit conversion.

`verify_results.py --phase types --images <dev> <release> --receipt <path>`
checks all pairs plus changes to either component, malformed arity, invalid
payload types and existing closed storage boundaries. `tests/result_types.fy`
also checks the canonical mapping, retained symbol types and field rejection.
Evidence: `docs/ops/evidence/candidate-result-types-20261007.json`.
Construction/patterns, owned HIR, MIR/C execution, propagation and must-use
Flow are subsequent numbered units, not claims of this type-only unit.
The public language, SDK and runtime are unchanged; Linux runs stay deferred.

## Result constructors and patterns

Version 0.35.0 type-checks contextual `.ok(value)` and `.err(error)` with exactly
one correctly typed payload. A Result switch requires each variant once, with
one immutable arm-local binding (or `_`) per variant. Success and error captures
retain their own types even when the arms share the same binding spelling.
The subject is checked once. Missing/duplicate variants, malformed constructors,
wrong payloads, escaping bindings and incompatible continuing-arm types fail
before HIR publication.

Following the Go reference, no-tail arms ending in direct return or literal
`assert(false);` contribute no join value; when every Result arm terminates,
the switch has void type. This does not change the narrower Option arm rules.
Builtin pattern/capture facts now live in `tagged_check.fy` rather than adding
another independent switch implementation to the general type checker.
`verify_results.py --phase patterns` covers every scalar pair, both construction
alternatives and malformed/terminal-arm controls. `tests/result_patterns.fy`
checks retained capture types and field targets. Evidence:
`docs/ops/evidence/candidate-result-patterns-20261007.json`.
Owned Result HIR, MIR/C execution, `?` and must-use Flow remain later units.

## Owned scalar Result HIR

Version 0.36.0 retains a distinct Result type category, canonical success/error
layout, construction operands, arm-local captures and exact source byte spans.
Each used ordered pair owns one synthetic four-field shape: success payload,
ok member, error payload and err member. Repeated spellings share the shape;
its declaration binding is zero and it cannot acquire user visibility.
The existing `fairyc.enum-payload-hir/v1` transport carries these owned facts;
readable types use `result:success:error`. No parsed or typed owner survives
publication.

`tests/hir_results.fy` reads the retained graph after input-owner destruction,
checks both payloads/captures and determinism, and rejects seven forged builtin
layouts plus mismatched Result/enum type categories. `verify_results.py --phase
hir` compares typed acceptance and HIR bytes in both candidate profiles.
Evidence: `docs/ops/evidence/candidate-result-hir-20261007.json`.
Version 0.36.0 retained HIR only; the following unit adds MIR/native acceptance.
Propagation and must-use Flow remain later units.

## Scalar Result MIR and native execution

Version 0.37.0 executes both alternatives of the 36 ordered scalar
`Result<T,E>` instances. The owned layouts survive HIR destruction and use the
existing verified tagged operations: 21 constructs, 22 tests a member and 23
captures its typed payload. Tag 0 is `.ok`, tag 1 is `.err`. The MIR wire format
remains `fairyc.enum-payload-mir/v1`; no duplicate Result-only operation set is
introduced. Copies, assignments, arguments, return values, if/switch joins and
loop-carried values retain the exact ordered Result type. Subjects and earlier
call arguments are frozen before subsequent effects. An early return during
payload evaluation emits neither the parent construction nor unused edges.

The private C representation is `struct fy_r_<success-type>_<error-type>` with
a tag and independently typed union members. Captures check the active tag
before reading the union and trap with category 7 on mismatch. Discarded
captures do not read payload storage. Option and declaration-owned enums retain
their separate C types. This is internal C11 transport, not a public tagged ABI.

`tests/mir_results.fy` rejects 18 executable MIR mutations for each alternative,
covering instance/tag/payload/type and initialization errors, and verifies that
failed MIR/C output preserves the existing buffer. Both development and release
native API tests pass. `verify_result_execution.py` observes both candidate
profiles in all four CLI modes, then executes the generated C in low/optimized
MSVC profiles. Its 657 CLI observations and 116 native executions cover all 36
pairs in both alternatives, copy/call/return, written arm order, frozen subjects,
terminal arms, loops, scalar boundary values and payload traps. Four of those
native executions are deliberately corrupted generated-C tag/branch controls:
both inactive-payload directions trap 7. They are not valid Fairy source tests.

Emit `samples/result-c-demo.fy` with `--c-file`; the sample asserts an i64 value
above i32 range and a bool error. Six additional native executions cover this
sample and existing Option/user-enum coexistence. Source, SDK/compiler/image
digests and raw observations are retained in
`docs/ops/evidence/candidate-result-native-20261007.json`.
At 0.37.0, propagation, must-use, nested/aggregate/resource payloads and tagged
aggregate storage remained units 9..20 or later. Linux validation stays deferred to the
consolidated batch; this unit makes no self-hosting or performance claim.

## Result propagation type checking

Version 0.38.0 resolves and type-checks postfix `?` in admitted function bodies.
The operand must already have an exact scalar `Result<T,E>` type; the operator
does not infer a contextual Result for `.ok(...)` or apply to Option. The
enclosing function must return a Result with exactly the same error type.
Its success type may differ: `Result<i64,bool>` can supply an i64 to a function
returning `Result<bool,bool>`. The propagated expression retains the operand's
success type; it is not an unconditional return and does not satisfy a missing
return obligation. Function arguments, conditions, while bodies and immediately
evaluated scalar defer arguments are admitted. Classic for headers retain their
existing pure-expression restriction and reject propagation in every clause.

Private type errors 40/41 identify a non-Result operand or an incompatible
enclosing return and point to the single question-mark byte. The retained
expected/actual fields record the exact error components; integer widening does
not make them compatible. `tests/result_propagation_types.fy` verifies these
facts and spans through native API tests in both candidate profiles.
`verify_results.py --phase propagation-types` covers 43 accepted and 52 rejected
inputs in both profiles (190 CLI observations), including all ordered scalar
pairs, error mismatches, calls, repeated `?`, and header positions. Three actual
Go-bootstrap reference checks independently confirm a compatible differing
success type, E4102 and E4103. Evidence:
`docs/ops/evidence/candidate-result-propagation-types-20261007.json`.

Version 0.38.0 implemented type checking only. Its HIR/MIR/C publication rejected
accepted propagation syntax explicitly; six boundary checks confirmed that it
could not silently lose the error path. Version 0.39.0 connects retained
propagation and runtime early-return semantics as described below. Linux,
performance and self-hosting remain deferred.

The first two release builds hit the existing bootstrap's 60-second MSVC
command limit. Release validation uses a separate local bootstrap built with a
Go overlay changing only that private timeout constant to 120 seconds. The
original executable and public compiler source are preserved. The evidence
records both bootstrap identities, the exact overlay change and the failed
attempts; this is not a release of a new stable build-timeout option.

## Result propagation execution

Version 0.39.0 retains postfix `?` as owned HIR expression kind 22, with one
operand and the exact enclosing function binding. Assembly checks the operand,
success type, error compatibility and unused fields after syntax/type arenas
are destroyed. `src/mir_propagation.fy` lowers the frozen operand through the
existing verified tagged construction/test/capture operations 21..23. Success
extracts the `.ok` payload and continues. Failure captures `.err`, constructs
the enclosing function's Result type, emits the same lexical `defer` suffix
as a direct return and returns before later operands or statements execute.
Registration metadata remains intact for sibling branches. Ordinary traps retain
the existing trap behavior, including traps during a deferred call.

`tests/mir_propagation.fy` checks retained HIR, eight forged-node/owner/type
rejections, verified MIR and C rendering. `verify_result_execution.py
--propagation` checks all 36 ordered scalar pairs, both alternatives, differing
success types, once-only evaluation, frozen earlier arguments, skipped later
arguments, conditions, loops, early operand returns and short circuiting.
Generated native executions also check nested/sibling `defer` order and failed
registration. The C observer only records actual generated `mark` calls; it
does not implement propagation or cleanup. Genuine Fairy trap fixtures exercise
integer overflow, signed division overflow, invalid shifts, an operand assertion
and a deferred-call assertion. Safe signed/unsigned boundary controls also run.
All core product logic remains in Fairy and the public SDK.

The optimized MSVC run exposed an always-trapping literal operation whose caller
continuation was still emitted. `src/c_guard_facts.fy` now proves block-local
literal/copy/unary guard failures with signed magnitudes, without overflowing the compiler
at i64/usize boundaries. The existing C reachability plan removes those success
paths and nonreturning-call continuations. Mutable/parameter/join values keep their
runtime guards. No warning flags, fixture expectations or runtime trap policies
are relaxed; fourteen direct facts tests cover failing, safe and unknown values.

Sample: `samples/result-propagation-demo.fy`. Evidence, raw command observations,
source/SDK/bootstrap/native-image hashes and the validation-only timeout overlay
are in `docs/ops/evidence/candidate-result-propagation-native-20261007.json`.
Both candidate profiles passed check/build and three native API tests each.
The execution lane passed 937 CLI observations and 130 generated native runs;
the checked-in sample passed two further native runs. The initial MSVC failure
and its original emitted bytes are retained alongside the passing observations.
Result handling obligations, nested/aggregate/resource payloads and resource
cleanup remain subsequent units. Public grammar, SDK/runtime and stable CLI
contracts are unchanged.

