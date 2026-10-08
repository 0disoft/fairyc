# Fairy candidate module progress

The existing Go package/module contracts remain the reference. These are
candidate implementation units, not additions to the public language.

## Owned declaration and type graph

`module_catalog.fy` records declarations by source-set module index and original
declaration index. Structs, resources, and enums share lookup rules with functions;
public visibility is checked for every qualified lookup. An alias changes the
lookup path, never the declaration identity. Module/import/name collisions and
missing providers fail before a catalog is published. The owner retains exact
source bytes and rejects resolution against a different source set.

`module_types.fy` interns nominal leaves by their exact module/declaration pair.
Option/Result and shared/mutable reference descriptors refer to canonical child
IDs. Existing scalar, C-scalar, raw-pointer, text, and byte-view descriptors remain
distinct. Failed resolution rolls back all speculative nodes. Graph publication
checks nominal coordinates and rejects forward/cyclic child references.

Construction uses separate local buffers/vectors and whole-resource publication;
it does not introduce mutable access to private resource fields. Bounds remain
32 modules, 32768 source bytes per module, 4096 descriptors, and depth 64.
These IDs belong only to the current source-set graph; package-instance identity,
persistent artifacts, and imported layout publication remain subsequent work.

The native dev/release API test executes the Fairy parser, catalog, and resolver
on real source fixtures. It covers equal file-local declaration numbers in
different modules, multiple aliases to one provider, private types, duplicate and
missing imports, wrong source-set inputs, rollback, and forged graph publication.
Evidence: `evidence/candidate-module-type-graph-20261008.json`.

This unit does **not** complete nominal multi-file body typing, HIR/MIR linking,
SDK admission, or native compilation of a multi-file user program. The older
scalar-only module CLI remains closed until those consumers can own the same
type and layout graph. Linux execution is deferred to the combined phase.

## Owned callable signatures

`module_signatures.fy` resolves provider and caller annotations into the same
canonical type graph, including nominal/resource/enum leaves, Option/Result,
references, C scalars, and the exact pointer output slot. The published owner
retains result/parameter IDs, module/declaration coordinates, public visibility,
effects, unsafe, must-use, and exact C linker bytes after the parser owners leave
scope. Conflicting C alias prototypes are rejected across modules; identical
prototypes retain their shared linker name.

Invalid declaration effects/must-use, private type imports, and void parameters
fail before publication. Generic signatures, receivers, body checking, and SDK
authentication remain outside this unit. Signature success is not a whole-program
check result. Evidence: `evidence/candidate-module-signatures-20261008.json`,
with actual Fairy dev/release native API execution.

## Owned AST composition

`module_ast_join.fy` copies parser-owned arenas into one source-set arena.
Original source bytes remain exact with a single newline between files; names
are not substituted and the concatenation is not reparsed. Expression, statement,
type, field, entry and attribute links shift by their own arena bases. Token
indices are mapped through their original byte positions, including trivia that
merges at the separator. `syntax.fy` exposes token facts and module-position
queries for these consumers.

The focused native test checks every source span against the original owners,
plus call arguments, literal cooking, assignment operators, query tokens,
tagged captures, switch and if bodies, for headers, cleanup receivers and enum
payload fields. Empty, generic and over-count inputs fail before publication.
Evidence: `evidence/candidate-module-join-20261008.json`.

This arena is bounded to 32 files and 32768 combined source bytes, 512
declarations, 4096 expressions/types/attributes/arguments, 8192 statements and
1024 entries/fields. Generic expansion, source-set binding and program lowering
are separate consumers; AST composition alone does not claim their completion.

## Nominal program consumers

`module_bind.fy` resolves named annotations and qualified expression uses through
the owned declaration catalog, then translates each exact module/declaration
coordinate to a unique declaration slot in the joined arena. Its input must
match both the signature owner's retained source set and the joined source bytes.
Aliases retain the same slot; identical names from different modules retain
different slots. These slots are transient program-owned identities, not stable
package IDs or persisted interface keys.

`module_program.fy` composes signature publication, AST join, binding, ordinary
name and type checking, owned HIR lowering, and MIR lowering. Name lookup and
duplicate checks stay within the source module. Bound nominal leaves flow into
borrow, tagged and array type consumers. Cross-module resource field access and
direct resource construction reject with private candidate code 6201; factories
and provider-owned borrowed access remain available. Callable effects, unsafe and
must-use are checked against the actual provider declarations.

The focused native API test renders verified C for imported structs/resources/
enums, aliases and duplicate helper names, and rejects mismatched nominal types,
private calls/representation, missing permissions, ignored must-use, and an
invalid provider with its original file-local position. It also rejects stale
signature inputs. Evidence: `evidence/candidate-module-program-20261008.json`.

This private API currently reuses one joined HIR/MIR owner (module number zero,
unique joined declaration numbers). The catalog-to-arena mapping is owned during
compilation; persistent package identity and per-file debug maps are not claimed.
The combined source/global-name limits remain 32768 bytes/128 globals. SDK,
generic and receiver admission, imported return-origin generalization, CLI
program output and actual execution of the emitted multi-file C remain separate
work. The existing scalar-only `--check-modules-file` command is unchanged.

## Multi-file C CLI

`--c-modules-file <source-list>` uses the same file loader and explicit relative
path-list rules as `--check-modules-file`, then compiles the whole source set
through `module_program.fy`. It publishes buffered C only after every provider
and caller body, owned HIR/MIR and C entry check succeeds. Diagnostics retain the
original path and byte range, with private phase/code values. C output requires
exactly one nonforeign `fn i32 main()` across the input set. This is an explicit
source list, not the production workspace/package manifest command.

`verify_module_program.py --phase cli` compares emitted C from development and
release candidate images, then executes separately compiled observed C at low
and optimized profiles. The resource case calls a separately compiled C helper
from its provider-owned cleanup; the expected single `A` trace checks exactly
one destruction after return. Other cases exercise imported struct values,
duplicate helper names and enum aliases. Private access, provider errors and
missing entry fail with empty stdout; input bytes are unchanged.

Evidence: `evidence/candidate-module-cli-20261008.json`. The previous 120-second
validation bootstrap timed out building the enlarged release CLI. A separate
validation-only 240-second Go driver overlay preserves the original compiler
source bytes; the receipt records its exact compiler and overlay hashes. This
does not resolve the production CLI timeout task (75).

## Owned repr(C) permission

`repr_c_declarations.fy` admits an exact single `@repr(C)` marker on nonempty
inert structs with C identifiers and closed scalar/C-scalar/raw-pointer fields.
Arrays, nested records, borrows, resources and tagged fields remain outside this
lane. Marker errors use private code 5106; unsupported field lanes use 5102.

`hir_shapes::Abi` retains the nominal declaration identity plus marker/name spans
in owned source bytes. HIR publication checks the actual binding, field layout
and marker bytes; MIR publication and verification recheck the same permission.
This is a separate layout proof, not a caller-supplied repr flag. HIR and MIR
observation schemas include Q rows for these records.

The focused native API tests cover declaration admission, exact retained spans,
owner teardown, HIR-to-MIR survival, missing/wrong marker bytes and forged field
lanes. Evidence: `evidence/candidate-module-repr-20261008.json`.
By-value foreign results and C names/layout transport are the next unit; this
declaration unit does not claim them. Linux remains deferred.

## repr(C) foreign results and C transport

Extern C results may now name a provider-owned repr(C) struct. Ordinary struct
parameters remain outside the closed foreign lane. Imported result annotations
resolve to the actual module/declaration before admission; a same-spelled plain
struct cannot grant ABI permission. Owned HIR and independently checked MIR
require the matching ABI record, and C output retains the verified external tag
and field names in definitions, prototypes, local values, constructors and
borrowed field access. Internal nominal names remain private numeric identities.

The separate C provider returns 8-byte and 20-byte records, plus mixed-width,
C-double and pointer fields. Development and release candidates emit identical C;
the low and optimized native executions verify all returned field values, aliases,
borrowed reads and nested ordinary wrappers. Nine invalid single-file fixtures
and two imported repr errors fail without output. Removing a valid MIR owner's
ABI proof fails verification and leaves an existing output buffer unchanged.

The larger field predicate exposed development-candidate stack exhaustion in
recursive MIR expression lowering; it also reproduces with a plain struct.
Switch, if and logical-expression snapshots now live in dedicated helpers rather
than every recursive expression frame. Source-depth accounting and ownership
joins stay unchanged. The unchanged five-field conjunction now executes through
the module CLI. This targeted repair does not establish a global stack bound.

Evidence: `evidence/candidate-module-repr-results-20261008.json` (six programs,
nine errors, twelve native executions), `evidence/candidate-module-repr-cli-20261008.json`
(seven module programs, six errors, fourteen native executions), plus three
development-native API checks. Both candidate build profiles succeeded with the
validation-only 240-second bootstrap. SDK symbols, nested repr(C) layouts,
by-value struct parameters and Linux execution remain subsequent work.

## Owner-based raw text and byte views

The candidate recognizes `str::from_raw_parts`, `byte_slice::from_raw_parts` and
`byte_slice::from_raw_parts_mut` as compiler operations inside an unsafe boundary.
Exactly three ordered arguments provide a safe owner borrow, a raw u8/void
pointer and usize length. Shared constructors require a shared owner; mutable
bytes require an exclusive owner and mutable pointer. A by-value resource or raw
pointer cannot supply owner provenance. CStr admission is still pending.

HIR RawView nodes retain all three operands and independently check lexical
unsafe permission and exact types. Origin summaries, retained loans and mutable
generation ancestry follow only the owner operand. MIR operation 38 retains the
owner even though C emission needs only the pointer and length; its independent
origin fixed point substitutes the owner's parameter at calls. Removed unsafe
permission, owner replaced with a pointer, and missing length edges fail before
any C is appended. Existing byte indexing bounds checks remain in force; text
bracket indexing remains unavailable.

Unsafe callers remain responsible for accessible pointer extent, lifetime and
valid UTF-8 for text. Construction performs no new allocation, pointer/owner
identity test, or UTF-8 scan. The native examples use valid backing storage and
exercise static/local/resource owners, parameter-return wrappers, UTF-8 text,
mutable updates, parent resumption and null/zero-length construction. Error
examples cover unsafe permission, argument types, local escape, donor movement
and simultaneous mutable siblings. Candidate donor-movement rejection currently
uses its existing private HIR E2101 loan diagnostic, not Go's E2105 spelling.

Evidence: `evidence/candidate-raw-views-20261008.json` and
`evidence/candidate-raw-view-modules-20261008.json`. SDK package authentication,
Vec specialization, CStr representation and Linux execution remain separate.

## SDK source authentication and Vec graph admission

`sha256.fy` computes bounded SHA-256 directly in Fairy. Checked arithmetic uses
16-bit limbs for modular additions and masks discarded rotation bits before
left shifts. `sdk_authority.fy` pins the exact bundled language/20 runtime/0
sources, including the optional vector module. A source-set module claiming the
reserved std namespace must match these bytes; a name or caller-supplied digest
does not grant authority.

The owned module type graph now records authenticated Vec instantiations with
the canonical provider declaration and one canonical element edge. Aliases of
the same provider reuse the same identity; differing scalar or resource elements
remain distinct. The six storage fields are opaque data followed by five usize
fields, independent of the element identity. Bare templates, safe borrow elements
including nested tagged borrows, mutated source bytes and forged provider
coordinates fail admission. Failed resolution rolls back speculative interning.

Evidence: `evidence/candidate-sdk-authority-20261008.json`. Native API tests use
an explicit SDK source argument because isolated native tests run in temporary
directories. The type graph does not yet instantiate SDK function/resource ASTs
or publish executable Vec HIR/MIR/C layouts; those are subsequent work in 46/47.
The existing ordinary module graph contract is included as the narrow regression.

`sdk_vector_plan.fy` now owns the catalog, canonical type graph and ordered
concrete Vec instance requests. Provider aliases share one request; differing
module-owned elements and nested Vec edges retain their exact identity. Module
signatures own this plan, with an empty request list for ordinary programs.
Publication rejects missing, extra, reordered or forged requests and caps the
list at 64. Duplicate canonical type nodes are rejected at graph publication.
The plan records the six-field Windows AMD64 descriptor transport (48 bytes,
alignment 8); element layout and executable template cloning remain subsequent
work. Three native API tests pass in development and release, plus the existing
module-signature contract test in development. Evidence:
`evidence/candidate-sdk-vector-plan-20261008.json` and
`evidence/candidate-sdk-vector-plan-signatures-20261008.json`.

The six authenticated Vec extern declarations now carry exact runtime linker
names, scalar/pointer results and parameter types through the existing owned
HIR/MIR permission checks. Declaration lookup uses the complete extern prefix:
cleanup's earlier pop/release calls cannot supply declaration authority. A native
API fixture in both profiles observes all six verified C prototypes and rejects
missing/wrong scopes, changed source bytes, result/parameter/effect changes and
a call-site substituted for the extern declaration. The process SDK permission
regression passes in development. Evidence:
`evidence/candidate-sdk-vector-foreign-20261008.json` and
`evidence/candidate-sdk-vector-process-regression-20261008.json`.
The fixture constructs owned MIR directly; executable Vec template ASTs,
element storage operations and actual Vec runtime execution remain pending.

## CStr storage and owner-based reconstruction

The candidate carries `ref CStr` as a separate immutable descriptor with data
and private storage extent. C literals retain exactly one trailing NUL; owned HIR
and MIR independently reject missing or interior NUL in literal payloads. The
storage extent includes the terminator and has no public len/index/metadata lane.
Bare CStr, mutable CStr and conversion to a mutable byte pointer remain rejected.

Unsafe `CStr::from_raw_parts` retains the separate safe owner's origin and loans,
not the raw pointer's address. Internal parameters/results, imported signatures
and a restricted Option payload retain this type. Generated C uses the verified
descriptor's data pointer for an explicit unsafe immutable pointer cast. Raw
construction trusts the caller to establish accessible storage and the CStr
terminator/lifetime invariants; it does not scan or allocate.

Six programs and eight errors were checked through both candidate profiles;
twelve native C executions inspect empty, ASCII and Korean literal bytes, raw
owner wrappers, parameter returns and tagged captures using a separate provider
translation unit. Imported CStr/raw-view programs add four native executions.
One native API test in each profile rejects mutated literal terminators and
removed unsafe/owner/length facts; failed C publication preserves a prefix.

Evidence: `evidence/candidate-cstr-native-20261008.json`,
`evidence/candidate-cstr-modules-20261008.json`, and
`evidence/candidate-cstr-api-20261008.json`. SDK bodies and Linux are still pending.

## Consistent bounded source-set admission

`candidate_limits.fy` owns the 64 KiB per-file, 1 MiB joined-program and 256-file
bounds. The manifest has a separate 64 KiB bound. File loading, scanner storage,
catalog ownership, AST joining/rebinding, generic source reconstruction and
owned HIR/MIR use these bounds consistently. Separators count toward the joined
limit before allocation. MIR pending-temporary source positions retain offsets
past 64 KiB without weakening their independent local/scope bounds.

Three selected native API tests pass in development and release: exact 256 and
257 inputs, exact aggregate bytes and separator overflow, and a resource program
located after a padded 64 KiB file. The scanner's former independent 32 KiB guard
was exposed by these tests and replaced with the shared program bound. Evidence:
`evidence/candidate-source-limits-20261008.json`. CLI/native transport evidence is
in `evidence/candidate-source-limits-cli-20261008.json`.

This change retains independent node, declaration, binding, depth and analysis
budgets. It admits larger input; it does not establish complete SDK-body or
self-source compilation. Runtime/cache source identity and Linux remain pending.

## Bounded large rank-one arrays

Rank-one scalar and inert aggregate arrays now admit lengths 1..4096. The earlier
1..64 scalar/composite IDs remain unchanged; extended scalar and owned element
keys use separate closed namespaces below tagged IDs. The count parser rejects
overflow before multiplication. Safe byte-array backing references recognize the
new canonical scalar IDs; arbitrary holes, resource/view elements and large
rank-two dimensions remain rejected. Rank two retains its existing 1..64 bounds.

C array definitions use bounded ordered ID vectors instead of indexing a 520-bit
table with an extended type ID. Repetition, field initialization, array/record
copies, element access and length queries retain their existing owned HIR/MIR
checks and runtime index checks. This implements the actual `u8[4096]`,
`usize[1153]`, `i64[1024]` and inert `Token[128]` forms without raising AST,
binding, recursion or output budgets.

One owned-transport/identity API test passes in both native profiles. Four
programs and six errors pass through both candidate images; eight low/optimized
native executions cover full record copies, amounts above i32, a mutable byte
subview and the exact index-boundary trap. A legacy rank-two program emits the
same C bytes as the prior candidate. Evidence:
`evidence/candidate-large-arrays-api-20261008.json` and
`evidence/candidate-large-arrays-cli-20261008.json`. Larger ranks and general
resource-array ownership are outside this unit; SDK instantiation remains pending.

## Imported scalar function specializations

Joined source sets retain function templates until private/public binding has
resolved their provider declaration. Expansion keys use that declaration and
ordered scalar arguments; two aliases share one specialization and same-named
caller/provider functions remain distinct. Nested template helpers retain the
provider module, nominal type bindings and declaration references during cloning.
The existing per-template, total, depth and work budgets remain unchanged.

Development runs three selected API tests; release runs the new provider-identity
test. An earlier release selection hit the observer's 360-second whole-selection
timeout, without a completed result. It is not counted as a passing run. The
observer now budgets each selected compilation and terminates its process tree
on a timeout, and can append a narrow continuation only when compiler, source,
test and lock digests are unchanged.

Generic aggregate and receiver declarations remain rejected. SDK Vec body
instantiation and persistent package identity are subsequent work. Evidence:
`evidence/candidate-module-generics-api-20261008.json` and
`evidence/candidate-module-generics-cli-20261008.json`.

## SDK enum errors and authenticated numeric bodies

Option/Result payloads retain user enum declaration IDs, including imported SDK
errors. A closed sixteen-bit enum leaf uses an unused prefix; existing scalar,
struct, resource, view and reference encodings are unchanged. The thirty-bit
compound budget remains enforced: two enum leaves need 34 bits and are rejected.
HIR shape validation requires the exact retained enum layout and MIR capture
validation requires the exact payload type. Removing the enum layout or changing
its ID fails before publication.

`std::num` runs from the complete authenticated bundled Fairy source through the
ordinary module, HIR, MIR and C path. Three programs cover i64 decimal limits,
syntax/range errors, checked integer conversions and local enum Option/Result
storage and switches. Three errors reject modified/spoofed SDK sources and a
same-spelled but distinct caller enum. The same assertions run through the Go
bootstrap in development and release; Python only writes fixtures and compares
outcomes. Evidence: `evidence/candidate-enum-tags-api-20261008.json` and
`evidence/candidate-sdk-num-cli-20261008.json`.

SDK Vec specialization, storage intrinsics, allocation/file/console runtime
symbols and the candidate project driver remain pending. This numeric-body unit
does not establish complete SDK compilation or self-hosting.

## Authenticated process argument runtime

`sdk_scopes.fy` retains exact SDK file intervals in the owned shape table. Joined
module intervals include original leading/trailing trivia; SDK authentication
rehashes those unchanged bytes. HIR publication and independent MIR verification
require the actual process declaration/linker positions, zero raw effects,
exact parameter count/types, and the result's authenticated repr(C) field layout.
An ordinary module cannot acquire a reserved runtime symbol merely by importing
the real SDK. Scope records survive parser/typed owner removal and HIR/MIR copies.

The original `std::process` Fairy bodies now execute through module binding,
HIR/MIR and generated C against the original Windows argument runtime. Native
programs check the argument count, executable-name copy, exact buffer-shortage
length, unchanged failure buffer, out-of-range status, Korean bytes and an empty
forwarded argument. Go-bootstrap reference runs exercise the same assertions.
Modified SDK bytes, spoofed std modules and stolen linker names fail without C
output. An API test in both native profiles rejects removed scope authority,
changed result types/layouts and modified source bytes before C publication.

The combined case exposed an existing borrow-analysis crash: `separate` queried
the nonexistent alternate expression of an if without else. The guard now skips
that nonexistent arm; it neither separates a live loan nor removes competing
access checks. A 180-byte ordinary-source reproducer failed with status 134 in
the previous candidate and emits/runs successfully after the guard. The same
regression is included in the process CLI lane.

Evidence: `evidence/candidate-sdk-foreign-api-20261008.json` (SDK authority tests
before the subsequent no-else guard), `evidence/candidate-sdk-process-cli-20261008.json`
(final candidate source and image identities), and
`evidence/candidate-borrow-if-no-else-20261008.json` (before/after reproducer).
These are bounded Windows results; Vec/storage, other SDK runtime components,
project commands and Linux execution remain pending.


## Authenticated scalar Vec AST specialization

- Validation: `check`; one native API in development and release.
- Evidence: `evidence/candidate-sdk-vector-expand-20261008.json`.
- Shared aliases produce one concrete Vec resource with six cloned fields and its
  original nonempty cleanup. Empty/len and the cleanup helper share scalar
  specializations; original SDK bytes and module ownership are retained.
- Closed storage operation names retain their substituted scalar arguments and
  require authenticated SDK spans. Their HIR/MIR/C execution remains pending.
- Resource elements, ordinary generic resources and forged SDK bytes remain
  rejected. Uninstantiated SDK bodies are not published as executable functions.
- A generic name is checked against locals in its caller's declaration. An
  unrelated function parameter no longer shadows a template; a local shadow at
  the bare call still fails E4501. Existing specialization budgets are retained.


## Scalar Vec storage execution

- Validation: `check`, development/release candidate builds, one mutation API
  per profile, four positive and seven rejected programs per candidate image,
  and sixteen MSVC low/optimized executions of emitted C.
- Evidence: `evidence/candidate-sdk-vector-storage-api-20261008.json` and
  `evidence/candidate-sdk-vector-storage-cli-20261008.json`.
- Closed SDK storage operations retain scalar element/result/operand types and
  authenticated source sites through owned HIR and independent MIR verification.
  Scalar size/alignment use C layout; take/put use typed memcpy. A put has no
  result local, and no other operation receives that void-result exception.
- Native programs cover empty/len/capacity/reserve/push/pop/clear, all six scalar
  layouts, first-allocation and growth failures, returned scalar donors,
  unchanged failure length/capacity/data, reverse prefix cleanup and zero live
  allocations. The observer compiles the original heap/vector runtime bytes;
  allocator and tail-pop hooks provide only failure/trace observations.
- Removed templates leave parser field gaps. Owned nominal, enum and builtin
  tag fields are compacted and all HIR field references are remapped. The
  provider-first native case covers struct and enum fields following the SDK.
- A cleanup receiver can be mutably borrowed and is bound to its concrete
  resource declaration, so distinct Vec scalar instances cannot share a receiver.
  Ordinary immutable owners still reject mutable borrowing.
- Mutations reject changed operation/result/element/unsafe/argument/source-site
  facts and damaged SDK bytes before C publication, preserving an output prefix.
- Resource/nominal elements, owner-bound view borrowing, Linux and
  self-hosting remain pending. This does not promote the candidate SDK/API.
- Test observer timeouts terminate the process tree. Receipts store C digests
  and byte counts rather than duplicate generated C; no CI or dependency change.

## Scalar Vec element borrows

- The actual authenticated SDK `at<T>` and `at_mut<T>` ASTs execute for all six
  scalar element types. The SDK checks the initialized-prefix bound before
  scalar-size address calculation. Private storage aliases carry both the safe
  Vec owner and raw slot through owned HIR/MIR; the raw pointer never supplies
  the lifetime origin. Borrow owners must use the authenticated six-field Vec
  layout, and unsafe/source-site/result/operand metadata is verified separately.
- Shared aliases permit shared owner observations. Structural mutation,
  replacement and moves wait for the last retained use. Exclusive aliases block
  other owner access; shared child reborrows suspend their parent until last use.
  Independent vectors retain separate owners. Ordinary SDK cleanup is unchanged.
- Specializations share source spans but have distinct scopes. Return-origin
  summaries now select return statements by scope, so `at` instances of different
  scalar types cannot merge each other's parameter origins.
- Evidence: `evidence/candidate-sdk-vector-element-api-20261008.json` and
  `evidence/candidate-sdk-vector-element-cli-20261008.json`. The native API
  rejects seven borrowed-operation mutations and preserves an existing C output
  prefix on rejection, alongside the previous seven storage mutations.
- Validation: development/release builds and two development native API tests;
  four positive programs, three bounds traps and sixteen rejected programs per
  candidate, with twenty-eight MSVC low/optimized executions. The previous four
  storage programs and seven rejections run against both current candidates,
  with sixteen native executions in
  `evidence/candidate-sdk-vector-element-storage-regression-20261008.json`.
- The release API file selection exceeded the observer's 360-second bound while
  the native compiler was still optimizing. Its process tree was terminated.
  The API receipt retains the timeout and has `passed: false`; release mutation
  API execution is not counted as a pass. Both release candidate generation and
  its low/optimized emitted-C executions above completed successfully.
- Non-byte Vec views, resource/nominal elements, Linux and stage1/stage2 remain pending.
  No language/SDK/runtime contract, CI workflow or dependency version changed.

## Vec byte views

- Authenticated SDK `view<u8>`, `view_mut<u8>`, `view_u8` and `view_mut_u8`
  retain the existing byte descriptors over the initialized prefix. Construction
  neither copies elements nor allocates. The private SDK transfer records its
  safe Vec owner, opaque data pointer and exact usize length in source order.
- Owned HIR validates the owner layout, source authority, mutability, element
  width and length. MIR independently checks all three operands, claims the
  length edge and checks its definite initialization. Raw view operation 38
  also reads its length edge during definite-initialization verification.
- Named non-generic byte exports are retained only when requested by an exact
  bound SDK declaration. Other SDK bodies stay closed, and unrequested byte
  exports introduce no extra Vec instance into earlier scalar storage programs.
- Existing origin and generation checks cover aliases, returned subviews,
  suspended mutable parents, structural mutation, replacement, move and argument
  evaluation. Byte metadata, loads, stores and slices use the existing path.
  Empty slicing now selects a null pointer directly instead of adding zero to it.
- Evidence: `evidence/candidate-sdk-vector-byte-view-api-20261008.json` and
  `evidence/candidate-sdk-vector-byte-view-cli-20261008.json`. The development API
  checks eight altered view/result/element/length/owner/unsafe/source facts and
  preserves an existing output prefix on rejection. Prior element and storage
  programs have separate current-development regression receipts.
- Non-byte scalar and nominal views, resource elements, Linux and stage1/stage2
  remain pending. Release mutation API execution remains deferred after the
  previous bounded timeout; release candidate/native execution is separate.
  No SDK/runtime ABI, language contract, dependency version or CI change.
- Validation: development/release candidate builds; one development native API
  test; five positive programs, three bounds traps and seventeen rejected
  programs per candidate, with thirty-two MSVC low/optimized executions.
  Current-development element and storage regressions execute fourteen and
  eight native runs respectively. Release mutation API validation is deferred,
  and Linux/self-hosting are outside this unit.

## Scalar Vec views

- Authenticated SDK `view<T>` and `view_mut<T>` now execute for i32, i64,
  u32, usize and bool as well as u8. Requested named integer exports use the
  original bound SDK declarations. All storage transfers retain the safe owner,
  opaque data pointer and initialized-prefix element count.
- Closed type identities keep bool distinct from i32 despite equal C storage
  widths. Owned HIR, module signatures, MIR loads/stores/slices and C descriptor
  pointers retain the exact scalar element and mutability. Byte IDs stay intact.
  Non-byte Option/Result borrow leaves use a separate canonical encoding escape.
- Returned shared/exclusive subviews and tagged shared views retain their owner
  and generation. Structural mutation, exclusive owner observations and parent
  writes while a child remains live are rejected. Final-use release permits
  growth and new views. Empty null-backed slices avoid pointer arithmetic.
- Validation: check, development/release candidate builds; 22 positive/bounds
  programs and 34 rejected programs per candidate, with 88 MSVC low/optimized
  executions total. The corpus includes existing byte views, all scalar extrema,
  bool values, fixed-array views, named integer exports, initialized-prefix bounds,
  empty end slices, tagged/returned origins and bool/i32 annotation mismatches.
- One development native API test verifies ten altered storage-view facts,
  including bool/i32 result relabeling, and preserves an existing C output
  prefix on rejection. All ten non-byte view IDs round-trip through Option and
  Result identities. The existing exhaustive HIR test recognizes ScalarView.
- Evidence: `evidence/candidate-sdk-vector-scalar-view-api-20261008.json`,
  `evidence/candidate-sdk-vector-scalar-view-cli-dev-20261008.json` and
  `evidence/candidate-sdk-vector-scalar-view-cli-20261008.json`.
- Remaining: resource/nominal Vec elements and views, release mutation API,
  Linux execution and stage1/stage2. Release mutation API remains deferred after
  the earlier bounded optimizer timeout. No SDK/runtime ABI, language contract,
  dependency version or CI change; item 47 is complete for scalar Vec execution.

## Resource Vec AST preparation

- Authenticated SDK templates accept a bound direct resource declaration as an
  element argument. Concrete instances retain the joined declaration identity;
  two distinct resource elements keep separate Vec and take-slot instances.
  Original SDK bytes and cleanup bodies remain unchanged. Ordinary user generic
  functions and nominal struct elements retain their earlier admission boundary.
- Check and two development native API tests pass. The existing scalar expansion
  observer now counts storage calls only inside executable declarations; removed
  named SDK view bodies remain in the owned AST and are not instantiated calls.
- Evidence: `evidence/candidate-sdk-vector-resource-expand-20261008.json`.
  Resource storage execution, ownership transfers and cleanup are still pending.
  This prerequisite alone does not complete item 48. No SDK, runtime or CI change.

## Resource Vec storage and cleanup

- Bound direct resource elements execute SDK size/alignment, take/put,
  empty/reserve/push/pop/truncate/clear and ordinary Vec cleanup. Storage queries
  require an owned HIR/MIR shape. Typed C memcpy relocates the representation;
  put consumes its donor in both lowering and independent MIR ownership checks.
  Take creates a new owned value using the existing temporary obligation path.
- Native observations cover retained resource borrows and scalar field writes,
  successful push/pop transfers, first-allocation and growth failure donor return,
  unchanged failure length/capacity, reserve without destruction, reverse prefix
  cleanup, nested resource fields and zero remaining allocator blocks.
- Short-circuit RHS-only owned temporaries now drop before the skipped/executed
  edges join. Their boolean result is frozen first. A failed push result on an
  executed RHS destroys its donor exactly once; skipped RHS calls create none.
- Validation: check and development/release candidate builds; five programs and
  four rejections per candidate, with twenty MSVC low/optimized executions total.
  Two development native API tests verify resource/scalar storage mutations and
  preserve a preexisting C output prefix. Current-development scalar storage
  regression runs four programs, six rejections and eight native executions.
- Evidence: `evidence/candidate-sdk-vector-resource-api-20261008.json`,
  `evidence/candidate-sdk-vector-resource-cli-dev-20261008.json`,
  `evidence/candidate-sdk-vector-resource-cli-release-20261008.json` and
  `evidence/candidate-sdk-vector-resource-scalar-regression-20261008.json`.
- Item 48 still needs actual scanner/diagnostic integration: scanner Token is an
  inert struct; cooked token tables and diagnostics own bytes::Buffer. Struct
  element Vec and item 49 Buffer execution are prerequisites. Resource views,
  release mutation API, Linux and self-hosting remain pending. No SDK source,
  runtime ABI, dependency version or CI change.
