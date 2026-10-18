[Candidate guide index](../README.md)

## Generic declarations and SDK observations

Functions, structs and resources preserve optional nonempty generic parameter
lists, including source order, trailing commas and duplicate names. Duplicate-name
validity and generic aggregate admission remain semantic concerns. This does not
add parameter constraints, defaults, specialization or name resolution.

`Declaration.generic_parameters` is a type-arena link root; zero means no written
list. `generic_start`/`generic_end` retain its complete angle-bracket span.
`generic_parameter_count` and `generic_parameter_at` expose ordered name nodes.
Private type-arena kind 13 is a parameter-name leaf and 14 links that leaf in
`left` to its preceding link in `right`. Neither kind is a value type. They share
the existing 4096-node budget and declaration rollback with ordinary type syntax;
two nodes are charged per parameter before successful declaration publication.

The SDK corpus reads unchanged sources as bounded input data, not as candidate
modules. `std::vec`, `std::bytes`, `std::text`, `std::alloc`, `std::process`, `std::io` and `std::fs`
are required-complete cases:
both native profiles must return status 0, zero problems and the exact whole Go
declaration list (kind, full span and name span). Exhaustion or partial parsing
fails immediately instead of falling back to a prefix observation. An independent
required inventory rejects omissions, duplicate disk/name identities and renamed
required mappings; each profile must finish all required files. Missing source
files fail at read time.

All seven current cases are required under the same source/node limits.
Logs identify each file's required status.
IO source is parsed as data;
its console functions and effects are not executed by the corpus probe. Likewise,
FS input is only parsed; filesystem SDK operations are not executed. The Go oracle
retains enum declarations as comparison kind 11; payloadless and payload-bearing
member syntax are supported by the candidate.
Future observation-only files may retain partial results with ordered Go-CST
subsequence checks and at most 128 original complete top-level prefix probes on
exhaustion. They do not weaken the required cases. `fairyc_go_test_parser_sdk_gate`
runs the inventory/outcome rejection controls and native corpus without unrelated
syntax suites; `fairyc_go_test_parser_sdk` also runs the corpus. Complete syntax
is not proof of SDK execution or semantic conformance. See the
[SDK syntax report](../parser-sdk-syntax-2026-09-12.md).

## Verified scalar C emission (`--c-file`)

Version 0.15.0 adds a Fairy-written C backend over the verified scalar MIR:

```sh
lexer-candidate --c-file samples/mir-defer-demo.fy
```

It accepts one module with exactly one `fn i32 main()` without parameters. The
existing scalar types, checked arithmetic, calls, branches, loops and lexical
defer are emitted as deterministic C11. The transport requires a 64-bit,
8-bit-byte, two's-complement target. Private numeric `fy_` symbols avoid source
identifier interpolation. Link the unit with the private scalar runtime providing
`fy_runtime_trap`; it is not a standalone public C ABI. Defining
`FY_CANDIDATE_NO_MAIN` omits the generated main wrapper.

`src/c_flow.fy` conservatively proves nonreturning closed callees and retains only
reachable blocks. A block-local final bool constant or literal zero divisor may select one successor;
unknown conditions and calls keep both possible paths. The bounded monotone pass
does not assume recursive calls return or trap. Emission omits normal continuations
of proven nonreturning calls and emits only required arithmetic helpers, keeping
strict C warning checks enabled. Arithmetic uses unsigned bit transport for signed
operations and explicit overflow, divisor and shift guards.

The same 32768-byte source bound applies. C output is buffered under a 2 MiB cap
before stdout publication. Invalid MIR/entry rejection preserves the caller's
existing Buffer. A later writer allocation/size failure may leave an unpublished
prefix in that internal Buffer; CLI failures still emit no C bytes. The writer
also rejects caller buffers already above its cap without arithmetic underflow.

Portable verification uses Python only for fixtures and observation, with a Go
test runner invoking the shared C driver; code generation remains Fairy C:

```sh
go build -o .cache/c-file-cli/native-runner ./tools/candidate/fairyc-candidate-c-native
python examples/lexer_candidate/verify_c.py --images /path/to/dev /path/to/release --reference .cache/mir-defer-file-cli/reference.json --native-runner .cache/c-file-cli/native-runner --receipt .cache/c-file-cli/c-receipt.json
```

Both candidate images emit byte-identical C for 106 Go-reference fixtures.
Native comparison observes the full i32 return separately from process status,
checks the private trap receipt and exact stdout, and adds a trace hook only to
test copies of the explicit defer trace function. The original generated main
also runs in a normal-return and a trap case. Windows low/optimized verification
covers 216 native executions, 228 C CLI executions and the existing 1256 CLI
executions. `tests/c_emit.fy` checks entry rejection, Buffer preservation,
determinism and the writer cap in both native profiles.
The Windows test runner can reuse an earlier successful build with the same
source/runtime hashes, compiler path/version and profile after verifying the
saved image hash. It always executes that image again and compares fresh results.
Linux builds are not reused by this runner.

Evidence: `docs/ops/evidence/c-file-cli-20261006.json`. Linux-target C emission is
checked; Linux native execution is deferred to the owner's consolidated batch.
Compiler/SDK inputs are unchanged. Compound values, ownership/cleanup lowering,
complete verifier coverage, package linking and a self-compiling compiler remain
subsequent work. No self-hosting promotion or performance claim is made.

## Same-file nominal struct type checking (historical: 0.16.0)

This section records the 0.16.0 scope and verification. Later HIR/MIR admission
is described in the versioned sections below.

Version 0.16.0 connects plain struct declarations to the existing name and type
checker. Named identity is the defining declaration, so equal field shapes do not
make two structs interchangeable. Scalar fields and acyclic nested plain structs
support record literals, reordered and shorthand initializers, copied bindings,
whole-value assignment, field reads/writes and internal function arguments/returns.
Every declared field must be supplied exactly once. Field writes require a mutable
local root; parameter, let-bound and temporary roots stay immutable. Scalar field
initializers retain contextual integer range/minimum checks.

`src/nominal_types.fy` owns the declaration lookup and bounded acyclic field graph.
Private type IDs are 256 plus the zero-based declaration index; scalar IDs stay
unchanged. The binding dump adds `struct` symbols. The typed result retains
declaration-based symbol/expression type IDs and `type_check::field_target` links to
the parser field arena, preserving written initializer order. No new language
syntax, public SDK type or runtime ABI is introduced.

The existing 128-global, 512-declaration, 1024-field and depth-64 bounds apply.
Resources, borrowed/array fields, generic structs, methods, attributes and imported
nominal definitions require subsequent components. At version 0.16.0 the scalar
HIR/MIR/C modes rejected nominal input with empty stdout. Version 0.17.0 below
admitted nominal HIR while MIR/C still rejected it. The type-checking unit does
not execute input struct operations.

```sh
go run ./tools/dev/fairyc-lexer-file-oracle --nominal .cache/nominal-types-file-cli/reference.json
python examples/lexer_candidate/verify_nominal.py --images /path/to/dev /path/to/release --reference .cache/nominal-types-file-cli/reference.json --receipt .cache/nominal-types-file-cli/nominal-receipt.json
```

Windows development/release verification compares 23 semantic outcomes per
profile with the Go reference, checks three explicit candidate rejection controls
per profile and six scalar-backend rejection controls, totaling 58 CLI runs.
The reference records Go semantic facts separately from later MIR errors: unused
by-value cycles pass Go semantics but are rejected by this candidate's finite
layout prerequisite; a bare type-name value also passes Go semantics but fails
Go MIR lowering and is rejected here. These three controls are not counted as
semantic-parity cases. Two native API tests per profile check retained field links,
nested mutation, nominal identity, empty structs, shorthand and malformed inputs.
The existing 1256 CLI runs pass with updated independent struct declaration
expectations. Linux native execution stays deferred to the consolidated batch.

During comparison, reordered aggregate initializers exposed a Go MIR defect:
the lowerer retained written field order while the verifier required declaration
order, and mutable source reads could change during later initializers. The
separate compiler regression now checks seven struct/resource cases through
verified MIR, interpreter results and both Windows native profiles. It evaluates
initializers in written order, captures copied source values, then normalizes
only construction operands; an initializer return stops remaining construction.

Evidence: `docs/ops/evidence/nominal-types-file-cli-20261006.json`.

## Owned nominal HIR (historical: 0.17.0)

This section records the 0.17.0 scope and verification, including its then-current
MIR/C rejection controls. Version 0.18.0 MIR admission is described below.

Version 0.17.0 retains same-file plain struct layouts and semantic trees after
the parsed, resolved and typed input owners are destroyed. The candidate owns
declaration-based type IDs, field types/name spans, constructor targets, written
initializer order, field accesses, nested assignment places and nominal function
signatures. It supports scalar and acyclic nested struct fields, empty structs,
forward declarations, shorthand, whole-value assignment and struct-valued
branches/calls/returns. At 0.17.0, input struct code did not execute: MIR lowering
rejected the nominal table before publishing a program; MIR/C errors published no
stdout.

`src/hir_shapes.fy` owns checked layout tables: at most 512 shapes, 1024 fields
and depth 64. Unknown field types, cycles, wrong owners/ranges and reused
identities are rejected before graph traversal. `src/hir_shape_lower.fy` copies
those facts from checked input. HIR assembly checks declaration/field identities,
field operand types, postorder edges, initializer uniqueness and complete
construction. A forged nominal variant cannot alias a scalar tag.

Scalar-only HIR retains byte-identical `fairyc.scalar-hir/v1`. Nominal HIR uses
the separate private `fairyc.nominal-hir/v1` transport: `N` rows retain type,
binding, field start/count and visibility; `D` rows retain field ID, owner/type and name span.
Named types render as `nominal:<declaration-type-id>`. Expression kinds add
`construct`, `field` and ordered `initializer`; their binding slots identify
the nominal declaration or field respectively. Scalar tags and record columns
are unchanged. This is an example-private format, not a stable CLI/SDK/ABI change.

```sh
go run ./tools/dev/fairyc-lexer-file-oracle --nominal-hir .cache/nominal-hir-file-cli/reference.json
python examples/lexer_candidate/verify_nominal_hir.py --images /path/to/dev /path/to/release --reference .cache/nominal-hir-file-cli/reference.json --receipt .cache/nominal-hir-file-cli/hir-receipt.json
```

Independent Go semantic HIR supplies 15 positive fixtures. Both native candidate
profiles compare complete function trees and field layouts (30 comparisons),
including initializer mutation/order and early return. The nominal checker
corpus retains 58 CLI runs: 46 semantic comparisons, six explicit candidate
rejections, two accepted HIR controls and four MIR/C rejections. Native API tests
check retained owners, deterministic rendering and malformed layout rejection.
The 1256 scalar CLI regression executions remain unchanged.
Sample: `samples/nominal_hir.fy`.

Evidence: `docs/ops/evidence/nominal-hir-file-cli-20261006.json`. At 0.17.0, struct
MIR, verification and C execution remained next; Linux native execution was
deferred to the consolidated batch. No self-hosting promotion is claimed.

## Owned nominal MIR

Example version 0.18.0 extends `--mir-file` to same-file plain, acyclic copy
structs with scalar or nested struct fields. `src/mir_lower.fy` consumes HIR;
`src/mir_model.fy` owns copied nominal layouts, source bytes and MIR arenas after
HIR destruction. Declaration-based identities remain distinct. Nominal function
signatures, ordered calls, struct-valued branches/returns, whole-value assignment
and nested field assignment lower into the existing control-flow graph.
Resources, borrowed/array fields, generic structs and imported nominal definitions
remain outside this subset. The existing 512-shape, 1024-field and depth-64 limits
apply alongside the scalar MIR budgets.

Three operation kinds carry struct values:

- **6, construct:** evaluate initializers in written order and freeze their
  values, then append operands in declaration order. A returning initializer
  stops construction. Call arguments are likewise frozen before later arguments
  can mutate their source.
- **7, field read:** copy a field from an evaluated, frozen owner value.
- **8, functional update:** produce a replacement owner value. `src/mir_nominal.fy`
  rereads the current assignment root and nested owners after RHS evaluation,
  preserving sibling changes made by that RHS. Compound assignment freezes the
  old target field before the RHS, computes the checked result, rebuilds the
  containing structs and copies the replacement into the mutable root.

`src/mir_verify_nominal.fy` checks constructor arity and declaration-order field
types, field ownership, operand/result types and unused payload slots. The
existing verifier checks edge ownership, calls/returns and definite initialization,
including every constructor operand. Lowering and rendering both require a valid
graph; invalid rendering preserves the caller's Buffer.

`src/mir_output.fy` emits private `fairyc.nominal-mir/v1`: `N` rows contain type,
binding, field start/count and visibility; `D` rows contain field ID, owner/type
and name span. Nominal `O` rows append `argument_start` and `argument_count` after
the literal payload column; constructor operands use the existing `A` edge arena.
Scalar-only output keeps `fairyc.scalar-mir/v1` and its columns unchanged.
Version 0.18.0 belongs to this example; it adds no public SDK, stable CLI or ABI
change.

`verify_nominal_mir.py` compares 20 Go oracle seeds across development/release
candidate images: **40 comparisons passed**. The candidate is native compiled
Fairy C. The script checks layouts and signatures, then independently executes
emitted MIR through the test-only Python observer against expected Go MIR
interpreter outcomes. Cases cover initializer order and
early return, copied call arguments, nested/sibling mutation, branches, loops,
arithmetic traps and defer traces. The separate Go call-argument snapshot fix is
commit `58350fb9`.

The 0.18.0 regression runs also passed: **58 nominal CLI runs**, **30 nominal HIR
comparisons** and **1256 scalar CLI runs**. Nominal CLI controls now accept HIR
and MIR and rejected C output. Native API final checks passed:
`tests/mir_nominal.fy` (2), `tests/hir_nominal.fy` (3) and `tests/mir_verify.fy` (1),
six tests per development/release profile, including ten nominal MIR mutations.
These historical comparisons established no empty-struct parity. Bootstrap
commit `8bd0e831` subsequently connected private empty construction, copying,
arguments, returns and nesting through MIR, the interpreter and C execution.

Version 0.18.0 still rejected nominal C output. Its evidence is recorded in
`docs/ops/evidence/nominal-mir-file-cli-20261006.json`.

## Owned nominal C

Version 0.19.0 extends `--c-file` to the same-file plain copy structs admitted by
nominal MIR. `src/c_nominal.fy` emits dependencies before their by-value owners,
constructs fields from the already frozen MIR operands, reads fields and copies
an owner before updating one field. Function signatures, arguments, returns and
locals retain the nominal type. Forward type references, nested layouts and distinct
types with identical fields keep their existing source identities.

C symbols use numeric arena IDs rather than source names. Private empty structs
contain one initialized dummy byte; this byte is not a source field and is not a
public ABI. Scalar-only units retain `fairyc.candidate-scalar-c/v1`; nominal units
use `fairyc.candidate-nominal-c/v1`. The existing 2 MiB output bound and 64-bit C11
target guards remain. Invalid MIR is rejected before appending output.

```text
fairyc build --release --locked
build/lexer-candidate.exe --c-file samples/nominal-c-demo.fy
```

The generated unit still needs the private trap runtime when compiled. The
portable `verify_c.py --nominal` harness invokes the existing native C runner;
scripts provide fixtures and observations, not product parsing or C generation.
It compares Go-interpreter outcomes for constructors, initializer ordering,
copied arguments, nested/sibling updates, control flow, traps and defer traces.
The initial run used a fixed native expectation for an empty-field RHS because
the Go bootstrap rejected that store. The field-store correction now admits it,
and `verify_c.py` takes that expectation from the Go MIR interpreter, alongside
empty-field early return and nested replacement. `@repr(C)`, resources, arrays/views, enums, generics
and multi-file nominal types remained outside version 0.19.0. Version 0.21.0
adds the scalar fixed arrays described below; views remain a separate step.

Windows MSVC development/release evidence, raw invocations, expected outcomes,
source and compiler identities and EXE hashes are recorded in
`docs/ops/evidence/nominal-c-file-cli-20261006.json`. Linux native execution is
deferred to the owner's consolidated batch; no performance measurement is claimed.

The initial focused run passed 48 nominal MIR comparisons, 74 nominal C CLI runs,
56 nominal native executions (24 Go-reference cases plus one fixed control and
three standalone images across both profiles), 58 nominal type/HIR/MIR/C CLI
controls, 20 native scalar regressions and three native API tests per profile.

The field-store follow-up passed 12 MIR comparisons, 36 C CLI runs and 18 Windows
native executions, restricted to three corrected cases and three prior controls.
All six expected outcomes came from the Go interpreter. Raw invocations, source
digests, reused candidate/compiler identities and fresh native EXE hashes are in
`docs/ops/evidence/empty-struct-field-store-20261006.json`; the original evidence
above retains its historical fixed-control classification.

## Owned scalar fixed arrays

Version 0.20.0 admits scalar fixed-array types and literals into name resolution,
type checking and owned HIR. Elements are `i32`, `i64`, `u8`, `u32`, `usize` or
`bool`, with literal lengths 1..64. Ordinary and repeated initializers preserve
written order; the repeated value is represented once. Exact element types and
lengths govern copies, function arguments and returns. Indexes use `usize` and
`.len` returns `usize`. Indexed assignment requires a mutable named array local.
Empty, oversized, nested or nominal-element arrays, array fields, slices and
borrowed arrays remain outside this component. Version 0.20.0 retained these
forms in HIR; version 0.21.0 connects them to verified MIR and native C execution.

`tests/hir_arrays.fy` checks retained types, element order, repeat representation,
index/length nodes and type-ID limits alongside existing nominal HIR controls.

Version 0.21.0 lowers construction, value copies, arguments, returns, length,
indexed reads and indexed assignment through `src/mir_arrays.fy` and
`src/mir_lower.fy`. Initializer operands are frozen in written order. A repeated
initializer evaluates its expression once, including any effects or early return.
An indexed store freezes and checks its index before evaluating its RHS; a
compound store also freezes the old element before the RHS. Replacing the whole
array during the RHS preserves the replacement's other elements.

`src/mir_verify_arrays.fy` checks exact element and length types. Indexed loads
and stores require a single-definition checked-index temporary for the same
array type, with definite initialization proving that the check dominates use.
`tests/mir_arrays.fy` rejects nine malformed MIR variants and confirms that a
rejected render preserves an existing output buffer.

`src/c_arrays.fy` emits each used array type as a private C struct with a fixed
`fy_items` field. Struct assignment preserves by-value array copies. Bounds trap
with category 6 before indexing; arithmetic uses the existing checked scalar
operations. Array units use `fairyc.candidate-array-c/v1`, array HIR/MIR dumps
use `fairyc.array-hir/v1` and `fairyc.array-mir/v1`. The scalar and nominal C
formats remain unchanged for programs without arrays. This transport retains
the existing 64-bit C11 guards and 2 MiB output bound; it adds no public ABI.

```text
fairyc build --release --locked
build/lexer-candidate.exe --c-file samples/array-c-demo.fy
```

The sample copies an array, updates the original and returns 3 from the copy.
Generated C still requires the private trap runtime when compiled. The portable
`verify_c.py --arrays` harness compares output from separate development/release
candidate images and runs emitted C against Go MIR-interpreter expectations.
It covers all six scalar elements, lengths 1 and 64, evaluation order, early
returns, loops, bounds and overflow traps. Scripts supply fixtures and observe
results; Fairy C owns parsing, lowering, verification and C generation.

Windows MSVC results and source/compiler/SDK identities are recorded in
`docs/ops/evidence/scalar-array-c-file-cli-20261006.json`. The run passed 68 array
CLI checks, 17 identical C units and 40 native executions, plus 54 nominal and
20 scalar native regressions. Three native API tests per candidate profile cover
HIR retention, temporary-array dump identity, nine MIR mutations and C output
controls. The Go bootstrap's array-call argument snapshot defect was corrected
in `5945ae28`; the reference now agrees with the candidate's by-value result.
Linux native validation
remains deferred to the owner's consolidated batch. No build-cost or performance
measurement is claimed. Array fields, nested or nominal-element arrays, borrowed
arrays, slices/views and array iteration remain separate candidate work.

## Owned static text and byte literals

Version 0.22.0 retains decoded `"..."` and `b"..."` bytes in owned HIR after the
parser and scanner are released. Their exact types are immutable `ref str` and
`ref u8[]`, with Static origins. Text remains UTF-8; byte literals may contain
any octet, including NUL and invalid UTF-8. Existing scanner escape validation
owns decoding. C strings, mutable views, local array borrows and literal-bearing
struct fields remain closed in this candidate slice.

Locals, arguments and literal-returning functions retain those types. Text and
byte views expose byte `.len` and `.is_empty`; only byte views admit indexed
reads with `usize` indexes. Text indexing, writes through byte views and implicit
text/byte conversion are rejected. HIR stores cooked bytes, not source spelling,
and `fairyc.literal-hir/v1` uses hex payloads so embedded NUL, tabs and newlines
cannot split records. Version 0.23.0 connects these retained values to MIR/C below.

`tests/hir_literals.fy` checks parser-independent ownership, distinct types,
Unicode escapes, empty text and embedded binary bytes. Windows native development
and release checks cover this retained HIR step; Linux runs remain deferred.

## Static literal MIR and native C execution

Version 0.23.0 executes immutable Static text and byte descriptors through
verified MIR and private C11 transport. `--mir-file` uses `fairyc.literal-mir/v1`
with hex literal payloads; `--c-file` emits static byte arrays and distinct
pointer/length structs for text and bytes. Empty values have length zero, and
embedded NUL does not terminate either type. Descriptors may be copied, passed,
returned and reassigned without copying their static storage.

Byte reads freeze the receiver before index effects, check `usize` bounds, then
load the byte. `.len` counts bytes and `.is_empty` tests length zero. The MIR
verifier requires the checked index and frozen descriptor to match, with one
definition each and initialization dominating the load. Invalid MIR is rejected
before C rendering changes the caller's buffer. Public grammar, SDK and runtime
contracts are unchanged.

`samples/literal-c-demo.fy` demonstrates Korean text, binary bytes and a
literal-returning function. For example, `build/lexer-candidate.exe --c-file
samples/literal-c-demo.fy` emits its C unit. `verify_c.py --literals` compares
development/release candidate output and Windows native execution with the Go
MIR interpreter. Exact observations are recorded in
`docs/ops/evidence/literal-c-file-cli-20261006.json`.

The run covers 16 programs, 72 C CLI checks and 38 native executions across
MSVC low/optimized profiles, plus array, nominal and scalar regressions.
Three native API tests per candidate profile cover HIR ownership, seven malformed
literal MIR mutations and C output preservation. The independent comparison
exposed the Go bootstrap's delayed descriptor selection; commit `4d5afa17`
fixes indexed reads and slice bounds, with five interpreter/native regressions.
The pinned compiler used to build the candidate remains unchanged.

C strings, view slicing, local-array borrows, mutable views, literal-bearing
struct fields, text indexing and writes through byte views remain outside this
candidate slice. Linux native validation remains deferred to the consolidated
batch. No performance or self-hosting completion is claimed.

## Shared array view HIR

Version 0.24.0 adds owned HIR for shared `u8[N]` array borrows in an expected
`ref u8[]` context and for half-open slicing of byte views. Bounds may be
omitted. Array places must be named, lengths remain 1..64, and text, mutable,
temporary-array and field-array borrows remain closed. HIR origin checking
rejects local escapes and mutation while a shared alias or call argument is
still used. A loop-held descriptor remains live across its backedge.
Non-Static local descriptors must be `let`; mixed-origin or recursive
view-return summaries are not yet admitted. Parameter-derived returns preserve
the single inferred parameter origin. Version 0.25.0 connects these values to
MIR/C execution below.
The HIR lane passed 17 positive/negative CLI cases and the retained-graph native
API test in development/release profiles.

## Shared array views, slices and native C execution

Version 0.25.0 executes shared byte-array borrows and half-open byte slices
through verified MIR and private C11 transport. `--mir-file` publishes
`fairyc.view-mir/v1`; `--c-file` emits pointer/length descriptors backed by the
original named `u8[N]` array. Slice bounds are evaluated once, left to right,
and checked before pointer arithmetic. Reversed and out-of-range slices trap
with category 6; empty and one-past slices are valid.

HIR tracks aliases, loop-held loans and descriptors held through later call
arguments, index expressions and slice bounds. Writes before the borrow or
after its last use are accepted. Half-open source spans keep adjacent completed
writes separate from the borrow that follows. MIR independently recomputes
Static, local-array and parameter origins; local-backed returns, unknown
consumed descriptors and mixed return origins are rejected before C output.
Invalid operation shapes and uninitialized bounds are rejected by the existing
shape and flow checks. Failed rendering preserves the caller's output buffer.

`samples/shared-view-c-demo.fy` demonstrates a local array, a parameter-derived
return and slicing. Run `build/lexer-candidate.exe --c-file
samples/shared-view-c-demo.fy` to emit its C unit. `verify_c.py --views` compares
both candidate profiles with the Go MIR interpreter and runs the emitted C in
MSVC low/optimized profiles. The source, tool hashes and exact observations
are recorded in `docs/ops/evidence/view-c-file-cli-20261006.json`.
The view lane covers 24 Go-oracle programs, 96 C CLI checks and 54 native
executions, including reversed/out-of-range slices and early returns from bounds.
Three native API tests per candidate profile check retained HIR, five malformed
MIR controls and output preservation. Literal, array, nominal and scalar lanes
provide the scoped regressions; the sample also runs in both C profiles.

Mutable views, field/nested-array borrows, text slicing, C strings,
view-bearing aggregate fields and mixed/recursive view-return origins remain
outside this candidate slice. Public grammar, SDK and runtime are unchanged.
Linux execution remains deferred to the requested consolidated batch;
performance and self-hosting have not been measured here.

## Mutable byte-view HIR

Version 0.26.0 retains `ref mut u8[]` separately from immutable byte descriptors.
An explicit expected mutable-view type permits `ref mut` of a named mutable
`u8[N]` local. Named immutable descriptors support byte reads, plain/compound
indexed writes, metadata and mutable child slices. Descriptors themselves cannot
be reassigned; temporary-array and field-array borrows remain closed.

Exclusive generation checking rejects owner reads/writes while a mutable loan
is live, shared/exclusive conflicts in either order, parent/parent-alias access
while a child remains used, and retained siblings even for disjoint intervals.
Exact mutable call arguments create bounded reborrows held through later
arguments. A mutable return preserves one inferred parameter origin and keeps
the caller's parent frozen until its last use. Child uses extend ancestor holds;
parent and owner access resumes after the appropriate last use. Loop-held loans
and indexed-store receivers stay live across backedges and RHS evaluation.

`tests/hir_mutable.fy` covers retained mutable nodes, parent/sibling/call conflicts
and local-return escape rejection. The initial HIR lane includes 30 positive and
negative CLI cases. Mutable-to-shared call conversion, mutable text, field/nested
arrays, mixed/recursive return origins and deferred view arguments remain closed.
Version 0.27.0 connects these retained nodes to execution below.

