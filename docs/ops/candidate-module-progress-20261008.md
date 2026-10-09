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

The current arena is bounded to 256 files and 1048576 combined source bytes,
512 declarations, 16384 expressions, 4096 types/attributes/arguments, 8192
statements and 1024 entries/fields. Per-file parser limits remain separate. Generic expansion, source-set binding and program lowering
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

## Struct Vec storage

- Authenticated SDK templates specialize direct bound struct elements as well as
  resources. Generic identity, storage size/alignment, typed memcpy and element
  references retain the exact nominal declaration. Struct fields still cannot
  own resources; ordinary user generic functions and enum elements stay closed.
- Development/release candidate builds pass. Three positive programs and one
  SDK assertion bounds trap, plus four rejections, pass per candidate with
  sixteen MSVC low/optimized executions. Programs cover a five-field Token-shaped
  struct, copy-donor reuse, allocation failure preservation, growth, shared and
  exclusive element references, and padded nested structs with tagged fields.
- Two development native API tests pass, including seven storage/authority
  mutations with output preservation and the existing scalar AST observer.
  Evidence: `evidence/candidate-sdk-vector-struct-api-20261008.json`,
  `evidence/candidate-sdk-vector-struct-cli-dev-20261008.json` and
  `evidence/candidate-sdk-vector-struct-cli-release-20261008.json`.
- The Token-shaped program verifies storage behavior; actual scanner execution
  still needs Buffer support for cooked text and diagnostics. Item 48 remains
  partial until that integration executes. Release mutation API, Linux and
  self-hosting are pending. No SDK/runtime ABI, dependency or CI change.

## Buffer execution

- The candidate executes the original authenticated `std::bytes` bodies and
  seven exact Buffer runtime extern declarations. Source coordinates, signatures,
  unsafe permission and SDK authority survive parser-owner removal and are
  rechecked before C publication. User declarations cannot claim these symbols.
- Development/release candidates each pass four programs and five rejections;
  sixteen MSVC low/optimized executions cover empty/reserved buffers, append,
  zeroed growth, truncate/clear, nested mutable slices, copies containing NUL,
  whole-resource moves, integer formatting at scalar limits, allocation failure
  preservation and zero live allocator blocks. Retained views prevent mutation
  or owner moves; shared stores and local-view escape remain rejected.
- Borrow analysis reuses fixed origins and successful generations, and checks
  competing owners before path traversal. Existing work/depth bounds remain.
  Recursive `if` lowering keeps its two ownership snapshots in explicit owned
  buffers; allocation failure aborts lowering. This removes the native stack
  overflow reproduced by the SDK decimal-digit conditional chain.
- Two development native API tests pass: seven Buffer authority/signature
  mutations are rejected, failed C rendering preserves an existing prefix, and the existing mutable
  parent/sibling/call-conflict test remains rejecting. The resource Vec regression
  passes five programs, four rejections and ten native executions.
- Evidence: `evidence/candidate-sdk-buffer-api-20261008.json`,
  `evidence/candidate-sdk-buffer-cli-20261008.json` and
  `evidence/candidate-sdk-buffer-resource-regression-20261008.json`.
- Item 49 remains partial: String/text execution is pending. Actual scanner
  Token/cooked-buffer/diagnostic integration in item 48, release mutation API,
  Linux and self-hosting remain pending. No SDK/runtime ABI, public language,
  dependency, database or CI change.

## Struct pointers for SDK text prerequisites

- Raw pointers can retain an inert struct's exact declaration identity. The
  source-set graph represents their pointee as a canonical child, preserving
  provider module/declaration and mutable permission across imports and joins.
  Nullable scalar/void pointers and the existing mutable void** output lane keep
  their previous IDs. Resources and pointer dereferences remain outside this unit.
- Foreign struct-pointer parameters/results require the pointee's owned repr(C)
  proof in parsing, source-set admission, HIR and independent MIR verification.
  C declarations use that proof's external tag through the pointer. Plain structs
  cannot claim this ABI; unsafe and exact safe-reference cast rules still apply.
- Check and development/release builds pass. Each candidate passes five programs
  and five rejections, with twenty low/optimized native executions covering
  imported shared/mutable pointers, foreign results, typed null, module forwarding
  and the existing opaque output slot. Wrong pointees, stronger mutable permission,
  missing unsafe, ordinary struct extern pointers and resource pointers reject.
- Two development native API tests pass. Removing owned ABI proof from pointer
  parameters/results fails verification and preserves an existing C prefix; the
  existing pointer namespace regression passes. Current-development Buffer
  regression passes four programs, five rejections and eight native executions.
- Evidence: `evidence/candidate-nominal-pointer-api-20261008.json`,
  `evidence/candidate-nominal-pointer-cli-20261008.json` and
  `evidence/candidate-nominal-pointer-buffer-regression-20261008.json`.
- This prerequisite does not execute String/CString SDK bodies yet. Item 49,
  actual scanner integration, release mutation API, Linux and self-hosting remain
  pending. No SDK/runtime ABI, public language, dependency, database or CI change.

## String/CString and UTF-8 execution (2026-10-09)

- The candidate executes original authenticated `std::text` and `std::bytes`
  bodies through eight exact text runtime extern declarations. Raw text results
  and shared raw-view pointers require the exact owned SDK declaration, repr(C)
  proof and field types in HIR and independent MIR checks. Source spelling or an
  equally sized user struct cannot grant the runtime authority.
- Development/release builds pass. Each candidate passes six programs and seven
  rejections, with twenty-four MSVC low/optimized executions. Observations cover
  Korean UTF-8, empty strings, append, copy/move, Buffer-to-String-to-Buffer pointer
  preservation without allocation, invalid UTF-8 returning its intact donor,
  String/CString allocation failures, append failure content preservation,
  CString terminating/interior NUL rules, borrowed UTF-8 validation and no live
  allocator blocks at scope exit. Both candidates produce identical C per case.
- An old type-only guard rejected every Option/Result switch returning a byte
  view or text borrow, before the existing HIR/MIR origin checks could run. It is
  removed. The Go reference accepts the UTF-8 switch extraction. Mixed origins,
  local view escape, retained-owner mutation and String-to-Buffer transfer still
  reject through origin/lifetime checks. No public syntax or language addition.
- Two development native API tests pass. Eleven text authority/signature/record
  mutations reject; failed C rendering preserves a preexisting output prefix.
  The struct-pointer ABI-proof regression passes, as does the current-development
  Buffer suite (four programs, five rejections and eight native executions).
- Evidence: `evidence/candidate-sdk-text-api-20261009.json`,
  `evidence/candidate-sdk-text-cli-20261009.json`,
  `evidence/candidate-sdk-text-reference-20261009.json` and
  `evidence/candidate-sdk-text-buffer-regression-20261009.json`.
- Item 49's Buffer/String/CString execution is complete for this candidate lane.
  Item 50 still needs console SDK execution and project CLI integration. Actual
  scanner storage/diagnostic integration, release mutation API, Linux and
  self-hosting remain pending. No SDK/runtime ABI, dependency, database or CI change.

## Console SDK execution (2026-10-09)

- The candidate admits nine exact authenticated `std::io` runtime declarations.
  HIR and independent MIR checks require the original owned write/read result
  records, exact scalar/pointer signatures and the `console` effect. Removing
  that effect, authority, ABI proof or source identity rejects before C output.
- Development and release native API tests each pass eleven mutations, including
  result/parameter types, effect removal, wrong source module, declaration spans,
  write/read record fields and missing repr(C) proof. Failed C rendering preserves
  an existing output prefix after parser owners have been released.
- Development/release console candidates each pass five programs and four
  rejections through twenty total low/optimized MSVC executions. The two
  candidates emit identical C for all five programs. Exact stdout/stderr include Korean UTF-8,
  embedded NUL, i32/u32/usize limits, i64 minimum/maximum/zero and bool formatting.
  Stdin covers zero-length reads, two-byte partial reads, embedded NUL and EOF;
  invalid standard handles preserve the raw error status/count and input buffer.
  Missing direct/transitive console effects, shared stdin and stolen symbols reject.
- The test-only native consumer accepts caller input and exact expected stderr.
  The current development text regression passes six programs, seven rejections
  and twelve native executions with the same consumer.
- Evidence: `evidence/candidate-sdk-io-api-20261009.json`,
  `evidence/candidate-sdk-io-api-release-20261009.json`,
  `evidence/candidate-sdk-io-cli-dev-20261009.json`,
  `evidence/candidate-sdk-io-cli-release-20261009.json` and
  `evidence/candidate-sdk-io-text-regression-20261009.json`.
- Item 50 still needs project CLI integration. Actual scanner Token/cooked-text/
  diagnostics integration, File SDK execution, broader release mutation API,
  Linux and self-hosting remain pending. No SDK/runtime ABI, public language,
  dependency, database or CI change.

## Joined body capacity for scanner integration (2026-10-09)

- Original scanner/number/string/diagnostic modules plus authenticated bytes,
  vector and console SDK bodies exceed the old 4096-expression joined AST cap.
  `candidate_limits::program_expressions()` now bounds joined AST, generic
  expansion, typing and owned HIR at 16384. Vector specialization's two-node
  reservation shares that cap. HIR canonical integer/literal payload producers
  and independent validation share the derived 344064-byte cap.
- Unsafe lexical visitation uses an explicitly sized owned Buffer, rather than
  indexing a fixed 4096-byte table with larger expression IDs. Allocation failure
  returns a lowering problem. Mutable parameter generations start after the
  actual expression arena, preserving disjoint parameter/expression identities.
  Existing depth, work and other arena limits remain independent.
- Program name resolution permits up to 512 globals, within its declaration cap.
  Single-file resolution retains 128 globals. Global lookup/duplicate checks reuse
  the existing owned Symbol table and exact module identity; local names retain
  priority. This removes the duplicate global array whose enlarged by-value
  context reproduced native stack exhaustion even for `fn i32 main(){return 7;}`.
- Check and development build pass. A wide program with more than 4096 expression
  nodes and 128 globals executes under both low/optimized MSVC C profiles. Foreign
  unsafe calls and mutable-parameter reborrows occur after the old ID boundary.
  Four rejections cover missing unsafe, a retained-child borrow conflict, joined
  AST overflow and the unchanged single-file global limit. The current-development
  text regression passes six programs, seven rejections and twelve native runs.
- Evidence: `evidence/candidate-program-capacity-dev-20261009.json`,
  `evidence/candidate-program-capacity-text-regression-20261009.json` and
  `evidence/candidate-scanner-admission-20261009.json`.
- The unchanged original scanner now passes composition but still rejects its
  byte `for-in` statement during name analysis. Item 48's actual scanner execution
  remains incomplete; item 59 owns that language prerequisite. Release candidate
  validation is deferred to the next scanner/byte-loop unit to avoid rebuilding
  the same compiler twice. Linux and self-hosting are pending. No SDK/runtime ABI,
  dependency, database or CI change.

## Copied byte iteration for scanner integration (2026-10-09)

- The candidate admits existing copied `u8` for-in over a named immutable shared
  byte descriptor or its direct checked slice. Name/type analysis gives the item
  an immutable body binding, and generic specialization remaps its exact item
  type. Owned HIR normalizes the capture, checked index, counter and loop into
  existing block/For/Index facts. No SDK source or source identifier is synthesized.
- The descriptor and slice bounds evaluate once before entering the loop. Hidden
  repeated nodes start at the body, so a mutable borrow used to calculate a bound
  ends with its call rather than being held on every backedge. Continue targets
  the counter post; break and return retain existing cleanup and ownership checks.
- Development/release builds each pass four programs (including a reversed-bound
  trap) and seven rejections, with sixteen total low/optimized C executions. They
  emit identical C. Cases cover empty/sliced/nested loops, item shadowing, scalar
  generic specialization, one-time effectful bounds, resource cleanup and release
  of the owner after exit. Mutation/move inside the loop, item assignment/escape,
  wrong item types, computed roots and mutable descriptors reject.
- One native API test verifies owned HIR after parser/type owners are released,
  exact hidden scopes, immutable byte items and break/continue loop targets. The
  resulting MIR also passes its independent verifier. The large joined-program
  regression passes both compiler profiles and four generated C executions,
  completing the capacity unit's deferred release coverage.
- Evidence: `evidence/candidate-byte-loops-dev-20261009.json`,
  `evidence/candidate-byte-loops-release-20261009.json`,
  `evidence/candidate-byte-loops-api-20261009.json` and
  `evidence/candidate-program-capacity-profiles-20261009.json`.
- The unchanged scanner now passes byte-loop name analysis and reaches type
  checking. Its next rejection is the shared view of the fixed byte-array field
  `ref encoded.bytes[..encoded.length]` in `strings.fy`. Evidence:
  `evidence/candidate-scanner-byte-loop-admission-20261009.json`. Item 48's actual
  scanner execution remains incomplete; item 59 still owns region allocation.
  Other iteration forms, release mutation API, Linux and self-hosting are pending.
  No public syntax, SDK/runtime ABI, dependency, database or CI change.

## Fixed byte-array field subranges for scanner integration (2026-10-09)

- Type analysis now recognizes a fixed-array slice inside an explicit shared or
  exclusive borrow, retaining the named place root, exact view element type and
  checked usize bounds. The candidate supports the scanner's
  `ref encoded.bytes[..encoded.length]` source directly. Original AST/source
  bytes remain owned and unchanged; the existing whole-owner checks still reject
  mutation during bound evaluation and retained-view use.
- The new `hir_slices` helper normalizes a projected array borrow followed by the
  existing Slice expression. MIR address projections select original field
  storage, never a copied array field. Default bounds and ordinary descriptor
  slices share the same helper. No new MIR operation or runtime symbol is added.
- Development/release each pass six programs (including two bounds traps), seven
  rejections and twenty-four total low/optimized C executions. Generated C is
  identical across candidates. Evidence covers nested shared fields, a view
  returned from its borrowed parameter, pointer identity between whole and ranged
  field views, exclusive writes visible in original storage, one-time bound
  effects, empty end slices and named u8/i64 array subranges. Local escape,
  whole-owner/sibling mutation, competing bound effects, immutable exclusive
  borrows, temporary roots and wrong bound types reject.
- One development native API test verifies retained owned HIR after parser/type
  owners are released, then checks nested address projections and array-to-view
  operations in independently verified MIR and successful C rendering. The byte
  loop regression passes both final profiles with sixteen native executions.
  A first MSVC host launch failed with 0xc0000142; retrying unchanged sources
  completed the suite. This is recorded in the API receipt.
- Evidence: `evidence/candidate-array-field-views-dev-20261009.json`,
  `evidence/candidate-array-field-views-release-20261009.json`,
  `evidence/candidate-array-field-views-api-20261009.json` and
  `evidence/candidate-array-field-byte-loop-regression-20261009.json`.
- The unchanged scanner passes type analysis and reaches HIR. Its payload
  selector combines views of the input parameter and local cooked-value storage,
  which the candidate still rejects as mixed origin (E2103). Evidence:
  `evidence/candidate-scanner-field-view-admission-20261009.json`. Item 48's
  complete scanner execution remains pending. Release API, Linux and self-hosting
  are also pending. No public language, SDK/runtime ABI, dependency, DB or CI change.

## Mixed-origin analysis set prerequisite (2026-10-09)

- `borrow_origins` implements analysis-local canonical sorted owner sets. Static
  and Unknown retain their distinct existing identities. HIR binding IDs and MIR
  local/parameter IDs remain leaves; set IDs use a disjoint namespace and are
  never published in HIR/MIR. Equal unions intern to one ID for fixed-point
  convergence. Membership/intersection retain every possible backing owner.
- Check and one development native API test pass. The test covers commutative,
  idempotent and associative unions, all owner membership, disjoint/intersecting
  sets, Static/Unknown handling and work-exhaustion rejection. Evidence:
  `evidence/candidate-borrow-origin-sets-20261009.json`.
- Bounds are 4096 sets, 32768 retained members, 1152 members per set and the
  shared 1048576 work budget. Allocation/work failures remain explicit. HIR and
  independent MIR wiring, scanner execution and release validation are pending.
  No public API/runtime/schema, dependency, DB or CI change.

## Local mixed-origin HIR/MIR wiring (2026-10-09)

- HIR selections retain canonical unions of every possible local backing owner.
  Initializer/scope checks inspect each member. Assignment, replacement and
  resource move holds test membership; exclusive generations test intersection
  so either incoming parent stays frozen until the selected child expires.
  Static branches add no owner and Unknown remains distinct. Singleton checks
  retain their constant-time path without introducing set traversal work.
- Independent MIR provenance reconstructs unions from owned operations and CFG
  merge destinations. A caller's single-parameter return summary can substitute
  a local union argument. Function returns still require Static or one exact
  input parameter; mixed/ambiguous returns remain E2103. No source origin hint or
  published HIR/MIR field grants this proof. Set/work/allocation bounds remain.
- Development/release each pass four programs, eight rejections and sixteen total
  low/optimized C executions, with identical generated C. Tests cover the actual
  input-versus-cooked-Vec selection shape, borrowed identity forwarding, byte
  iteration, tagged extraction, Static joins, exclusive owner selection, nested
  and scalar-generic joins. Either owner's modification, move and parent access,
  inner-owner escape and ambiguous input/local returns reject.
- Two development native API tests pass. Canonical set convergence remains valid;
  an independently reconstructed MIR whose scalar return is forged into a mixed
  local-view return rejects E2103 and preserves a preexisting C-output prefix.
  The development String/CString/UTF-8 regression passes six programs, seven
  rejections and twelve native executions.
- Evidence: `evidence/candidate-mixed-origins-dev-20261009.json`,
  `evidence/candidate-mixed-origins-release-20261009.json`,
  `evidence/candidate-mixed-origins-api-20261009.json` and
  `evidence/candidate-mixed-origin-text-regression-20261009.json`.
- The unchanged scanner now passes mixed-origin admission but still exits at the
  HIR analysis limit (phase 6, code 2, span 0..0). The precise budget/round guard
  has not yet been profiled; no limit has been raised. Evidence:
  `evidence/candidate-scanner-mixed-origin-admission-20261009.json`. Item 48's
  complete scanner execution remains pending. Release API, Linux and self-hosting
  are pending. No public language, runtime ABI/schema, dependency, DB or CI change.

## Indexed HIR borrow reachability and origins (2026-10-09)

- Definite-exit suffixes now form a sorted containment index. Reachability is
  computed once per expression/statement instead of scanning all blocks for
  every origin query. Existing conservative outcomes for zero-iteration loops,
  branches, return/break/continue and depth/work exhaustion remain.
- Origin memoization starts only after return summaries converge. Cached facts
  retain their required traversal depth, so reuse cannot admit an over-depth
  alias chain. Unknown recursive returns still reject E2103. The 1048576 work
  limit remains unchanged and index allocation/work failures stay explicit.
- Development native API tests pass for indexed-versus-original reachability,
  budget exhaustion and independent mixed-origin MIR mutation rejection.
  Development/release mixed-origin and String/CString/UTF-8 execution receipts
  are `evidence/candidate-borrow-index-{mixed,text}-{dev,release}-20261009.json`.
- Diagnostic-only guard profiling establishes that the unchanged scanner now
  passes initial HIR origins and assignment/move holds. Its next failure is the
  exclusive competing-access scan (guard 60112), which still compares every
  fresh loan against every expression. Evidence:
  `evidence/candidate-scanner-reach-limit-profile-20261009.json`. Production
  diagnostic codes remain unchanged; complete scanner execution is pending.
- No public language/API, HIR/MIR schema, runtime ABI, dependency, DB or CI change.

## Indexed exclusive accesses and scanner SDK execution (2026-10-09)

- Exclusive checking indexes whole-place references and every descriptor owner.
  Each loan merges only those owner lists, in ascending expression order with
  duplicates removed. Mixed-owner parents, alias ancestry, overlapping holds,
  path separation and the original first-error order remain checked. Generation
  and access reachability reuse the owned cache. The work limit is unchanged;
  index links are bounded at 32768 and all allocation failures remain explicit.
- Heads and the reusable merge cursor use owned vectors rather than copying
  large fixed arrays through candidate native frames. Development API tests
  compare indexed candidates with exhaustive owner membership/order and verify
  work exhaustion; independent mixed-origin MIR mutation rejection also passes.
  Development/release each pass four mixed programs and ten rejections, with
  sixteen native executions total. Evidence:
  `evidence/candidate-borrow-access-api-20261009.json` and
  `evidence/candidate-borrow-access-mixed-small-{dev,release}-20261009.json`.
- The release candidate compiles the original scanner/diagnostic/numeric/string
  sources plus Vec, Buffer and console SDK bodies. Its generated C runs in low
  and optimized native profiles. Tests cover lossless raw spans and EOF, input
  versus cooked-Vec payload origins, keyword and escaped string/byte payloads,
  unterminated-comment repair, byte-budget E0010 and zero live heap blocks after
  releasing nested storage. Cooked literal views retain their existing private
  hex-text encoding. Evidence: `evidence/candidate-scanner-sdk-release-20261009.json`.
- The default development candidate passes HIR budgets but exits with Windows
  stack overflow (0xC00000FD) during scanner C compilation. A diagnostic copy
  with a larger PE stack reserve completes that compilation; it is not an
  accepted candidate or native-execution result. Development scanner execution,
  full differential coverage, Linux and stage1/stage2 remain pending.
- No public language/API, HIR/MIR schema, runtime ABI, dependency, DB or CI change.

## Default-stack scanner compilation (2026-10-09)

- Diagnostic phase stops verify that default-stack HIR lowering completes, while
  entering MIR lowering overflows the Windows stack before C rendering. Probe
  source and lock bytes were restored exactly. Evidence:
  `evidence/candidate-scanner-stack-phases-20261009.json`.
- MIR expression dispatch now selects separate extended, view, aggregate/math
  and call/block helpers. Existing operation bodies, child order, error spans and
  depth accounting are retained. Aggregate temporaries for unrelated kinds no
  longer share every recursive expression frame. No stack reserve, work budget
  or recursion limit was increased.
- The default development and release candidates both compile the original
  scanner and SDK source set. Generated C executes Token/raw/cooked/diagnostic
  storage, E0010 and nested resource cleanup in both native profiles. Development
  and release produce byte-identical scanner C. Evidence:
  `evidence/candidate-scanner-sdk-dev-20261009.json`,
  `evidence/candidate-scanner-sdk-final-release-20261009.json` and
  `evidence/candidate-mir-frames-parity-20261009.json`.
- Mixed-origin and String/CString/UTF-8 regressions remain accepted/rejected as
  before. Broader scanner differential coverage, project commands, Linux and
  stage1/stage2 remain pending. No public language/API, HIR/MIR schema, runtime
  ABI, dependency, DB or CI change.

## Candidate-emitted scanner differential corpus (2026-10-09)

- The existing Go-reference corpus runner now also accepts a candidate-emitted C
  unit through an opt-in test. Go computes inputs and expected observations; the
  tested native process recognizes bytes through the original Fairy scanner.
  A small test-only console framing fixture preserves the binary protocol.
- All 303 checked-in Fairy fixtures and two real project sources are compared
  under current/previous recognition, insert-NUL/delete-first/invalid-byte
  mutations, and byte/token/diagnostic/work budget variations. Long tokens,
  strings and comments add boundary cases. The 7670 framed observations match
  exact token kinds, trivia, cooked bytes, diagnostics, repair, source ranges and
  process status for both candidate profiles and both native C profiles.
- Four native builds are reused across 448 bounded corpus batches, rather than
  building every fixture. The 23 malformed/truncated frames per native profile
  preserve their exact exit/status behavior. Totals: 30680 observations and 92
  malformed-frame executions. Both candidates emit identical C; complete input
  and expected stream digests plus native/compiler/runtime identities are saved.
- Evidence: `evidence/candidate-scanner-corpus-dev-20261009.json`,
  `evidence/candidate-scanner-corpus-release-20261009.json` and
  `evidence/candidate-scanner-corpus-parity-20261009.json`. Item 48's scanner
  storage/integration scope is complete. File SDK/project commands, whole-source
  compilation, stage1/stage2, promotion and Linux checks remain separate gates.
- Only test harnesses, a test fixture and progress documentation changed. No
  public language/API, compiler/runtime implementation, dependency, DB or CI
  workflow change. Without an explicit C input the new native test skips.


## File SDK extern authority (item 51, 2026-10-09)

- The Fairy candidate admits the exact authenticated `std::fs` source's 37
  extern declarations and nine C records. Owned MIR verifies the original
  declaration/link spans, filesystem effect, result layout, scalar parameters
  and nominal pointer parameters, including pointer mutability.
- The native Fairy test releases its Parsed owner before verifying MIR and
  rendering all prototypes. It rejects 14 authority/source/signature/effect/ABI
  mutations, preserves an existing C output prefix on rejection, and rejects a
  user declaration that claims the reserved file-open runtime symbol.
- Development and release native API evidence:
  `evidence/candidate-sdk-file-api-20261009.json`. The shared observer accepts
  `--sdk-source` and records its digest, so the file test receives the absolute
  bundled SDK path as its first program argument.
- This completes the extern authority slice only. Compiling the entire original
  file SDK still fails at its `Result<FileKind, FileError>` return annotation:
  the current enum plus struct payload encoding needs 30 payload bits, beyond
  the existing 28-bit Result operand bound. No source, stack, work or depth
  bounds were raised. File body execution, candidate compilation of
  `file_cli::load`, project command integration and Linux checks remain.
- Candidate Fairy implementation, native API test, observer, lock and progress
  documents changed. No public language/SDK/runtime implementation, dependency,
  DB, CI workflow or release change.


## Wide Result declaration pairs (2026-10-09)

- Added a disjoint private Result identity band for two declaration leaves,
  each retaining its nine-bit struct/resource/enum index and optional single
  Option layer. The fallback is used only when the existing prefix encoding
  cannot fit, so previous valid IDs retain their values and aliases reject.
- Updated owned C initialization and array-layout verification to recognize
  the new non-scalar tagged identities. No stack, work, depth or source bounds
  increased. The compact pair cannot itself be nested in another builtin.
- Development native tests: all six `tests/nested_tagged.fy` tests passed,
  including 512 declaration-index roundtrips and owned HIR/MIR/C survival.
  Both candidate builds passed. Three candidate-emitted programs cover both
  arms of enum/record, record/enum and Option-record/record Results. Their C is
  identical across candidates and passes six native low/optimized executions;
  wrong nominal identity and unsupported nesting reject in both candidates.
  Evidence: `evidence/candidate-result-pairs-20261009.json`.
- This resolves the tagged identity blocker recorded above for the File SDK.
  The original complete File SDK now compiles to C with the development
  candidate, and an initial original-runtime open/close probe passed in both
  native profiles. Complete file lifecycle and project-driver checks remain.


## File SDK native lifecycle (item 51 complete, 2026-10-09)

- Both candidate profiles compile the unmodified original File and Buffer SDK
  bodies. Original Windows runtime execution covers UTF-8 creation/open,
  write/write_all, sync, replacement, read/EOF, FileKind Result, close and
  removal. The fixture uses Korean paths and UTF-8/NUL bytes, checks the unused
  read-buffer suffix, rejects an interior-NUL path, and observes missing files
  after replacement and removal. Both test files are removed on success.
- Both candidates reject missing filesystem authority and a user declaration
  claiming the reserved file runtime symbol. The successful program runs in
  native low/optimized profiles for both candidates: four executions. Evidence:
  `evidence/candidate-sdk-file-lifecycle-20261009.json`.
- The native observer includes original runtime source files in its bounded
  wrapper, retaining each file digest in both the receipt and wrapper text so
  native cache invalidation follows library changes. The MSVC allocator probe
  uses the same explicit 16-byte maximum alignment as existing SDK probes.
  No compiler or native runner limits changed.
- Item 51's File transport and lifecycle slice is complete. Candidate compilation
  of `file_cli::load` and actual project commands remain item 50 work. Linux,
  stage1/stage2 and promotion remain later gates; native sync is not a power-loss
  experiment. SDK receipt appends now require matching program inputs, and the
  Result-pair observer keeps generated C/native products in the cache.
- Only observers, evidence and progress documents changed in this slice. No
  public SDK/runtime/API, dependency, DB or CI workflow change.


## Candidate file loader integration (item 50, 2026-10-09)

- Extracted the actual file reader into `file_source.fy`. Source and manifest
  reads share the same bounded helper while retaining their distinct public
  limits and manifest path gate. `file_cli::load` forwards to this leaf and
  `module_cli` imports it directly, removing the prior CLI import cycle.
- Both Fairy candidates compile this original loader with the authenticated
  File/text/Buffer SDKs. Native low/optimized execution checks empty, Korean-path,
  exact 65536-byte and oversized inputs, missing/invalid UTF-8 paths, injected
  Buffer allocation failure and the manifest path limit. Original runtime
  allocation and Windows handle hooks show no retained blocks or handles after
  the success and early-error paths. Four native executions passed; candidate
  profiles emitted identical C.
- Evidence: `evidence/candidate-file-source-dev-20261009.json`,
  `evidence/candidate-file-source-release-20261009.json`,
  `evidence/candidate-file-source-cli-regression-20261009.json`.
  Stage0 check and both candidate builds passed. The five existing file/help
  command observations keep exact stdout/stderr/exit across prior development,
  new development and new release images. The Go legacy CLI dependency test now
  copies the actual authored source inventory rather than a stale manual list;
  its workspace check passed.
- This proves the actual loader component and Stage0 CLI compatibility. Entire
  candidate-emitted CLI dispatch, broader project commands, Linux and stage1/2
  remain. No source/work/depth/stack limits increased; no public language, SDK,
  runtime, DB, dependency or CI workflow change.


## File token route and measured admission blocker (item 50, 2026-10-09)

- Factored the original token-file behavior into `file_tokens.fy`, independent
  of parser/type/module command dispatch, and routed `file_cli::run` to it. The
  token mode is captured before the argument buffer is reused for the path.
  Stage0 check, both builds and the Go CLI dependency check passed. Fourteen
  existing file/help/module command cases have identical stdout/stderr/exit
  across prior development, new development and new release images. Evidence:
  `evidence/candidate-file-token-route-regression-20261009.json`.
- The authored route is ready for direct candidate-C observation, but compiling
  it with all original required SDK modules rejects before producing any C:
  module AST join phase 2 code 10. The failed probe is retained, with
  `passed=false`, in `evidence/candidate-file-tokens-dev-20261009.json`.
- An isolated native Fairy probe measured the exact original parser arenas:
  1244 entries, 2720 types, 139 fields, 395 declarations and 10094 expressions.
  It verified join rejection with the unchanged 1024-entry bound. The other
  measured arenas remain within their existing limits. Evidence:
  `evidence/candidate-file-token-join-budget-20261009.json`.
- Item 50 remains open. The next prerequisite is authenticated-SDK AST
  compaction/reachability, retaining original source bytes, spans, full user
  validation and fail-closed budgets. Raising the AST or analysis limits is not
  part of this unit. The opt-in token observer retains exact prior authored CLI
  stream/status expectations; it has no successful native token-route results
  yet. The budget probe uses native JSON capture and explicit manifest/root
  arguments, and is not a default corpus test.


## Authenticated SDK dependency plan (2026-10-09)

- Added a bounded source-matched SDK dependency plan. Every user declaration,
  SDK non-function declaration and resource cleanup remains a root; authenticated
  ordinary SDK functions are retained through direct/alias name references and
  transitive dependencies. Private references retain their original target so
  downstream privacy diagnostics remain available. Declaration ownership uses
  binary search and a single queue, within the unchanged 1048576 work budget.
- Added explicit scanner ownership snapshots that copy the original source,
  token descriptors, cooked storage and diagnostics without rescanning. Current
  and previous lexer revisions retain their exact token/trivia/span/cooked data.
- Check and release native API tests (3) passed. The final development dependency
  test also passed, including source-set mismatch and work exhaustion. Evidence:
  `evidence/candidate-sdk-reachability-api-20261009.json`.
- This is the dependency/snapshot prerequisite only. AST arena compaction and
  pipeline integration remain the next unit; item 50 is still open. No language,
  SDK/runtime, DB, dependency, CI workflow or analysis bound changed.

## SDK AST arena compaction (2026-10-09)

- `sdk_ast_compact.fy` remaps expression, statement, entry and type IDs after
  authenticated SDK function reachability. Missing or non-contiguous retained
  links reject publication. Original source owners, token indices, spans,
  fields, attributes and all user declarations remain intact.
- `syntax::copy_source` copies the private scanner through its explicit snapshot
  API. The implementation does not expose mutable private resource storage.
- Development/release native API tests (2 per profile) passed: reduced SDK arenas can be joined and
  rebound with rebuilt signatures, an unused invalid user function still fails
  typing, and a private SDK call still fails at its original byte position.
  Evidence: `evidence/candidate-sdk-ast-compact-api-20261009.json`.
- The original CLI pipeline is unchanged in this unit. Item 50 remains open;
  token-route integration follows through an opt-in compiler API. No language,
  SDK/runtime, DB, dependency, CI workflow or analysis bound changed.

## Opt-in SDK compilation pipeline (2026-10-09)

- `module_program::compile_sdk` authenticates the complete input, compacts SDK
  arenas and rebuilds signatures before the existing join/bind/HIR/MIR path.
  `module_cli::emit_sdk` exposes it to the Fairy observation entry in
  `probes/sdk_module_driver.fy`; the original CLI dispatcher still uses `emit`.
- SDK dependency lookup caches exact caller/module spellings, including misses.
  Collisions compare original source bytes. Its 4096 heap slots and the existing
  1048576 shared work budget fail closed; no stack or analysis bound increased.
- Check and the development observation-driver build passed. The token program
  now passes compaction and join, then rejects at HIR phase 6 code 2101 in
  `token_output::hex`. The failed observation is retained as `passed=false` in
  `evidence/candidate-file-tokens-compact-dev-20261009.json`; driver inputs and
  image identity are in `evidence/candidate-sdk-module-driver-dev-20261009.json`.
- A small source probe reproduces the rejection with a byte loop and two
  consecutive mutable calls; the same calls outside that loop are admitted.
  The next unit corrects the byte-loop lifetime provenance. Item 50 is open;
  no token-route native success is claimed here.

## Byte-loop induction provenance (2026-10-09)

- The hidden byte-loop induction target now retains the one-byte body opener
  span, rather than the entire repeated body. HIR/MIR assignment containment
  therefore cannot extend every authored call argument's loan to loop exit.
  The target remains nonempty for the independent MIR range verifier.
- The original driver reproduces HIR 2101 for consecutive mutable calls in a
  byte loop. The fixed development driver admits that program and explicit
  reborrows; both execute in low/optimized MSVC profiles with exact mutation
  counts. A later argument that mutates the held owner still rejects with 2101.
  Evidence: `evidence/candidate-byte-loop-calls-dev-20261009.json` and
  `evidence/candidate-sdk-module-driver-dev-fixed-20261009.json`.
- Check passed. Release-driver and full token-route observations follow; no
  analysis budget, source bound, SDK ABI, database or CI workflow changed.

## Authored file-token path execution (2026-10-09)

- The opt-in driver now compiles `file_tokens::run`, its real scanner/output
  dependencies and the complete original required SDK modules to owned MIR/C.
  The former 1024-entry join rejection is removed through SDK arena compaction,
  without changing that bound or the existing work/depth/stack limits.
- Development and release candidates emit identical C (769317 bytes). One
  authored Fairy program asserts the prior CLI's status for each of four input
  cases and produces their exact concatenated stdout/stderr: Korean UTF-8 with
  CRLF/documentation/cooked data, malformed string, 65537-byte input and missing
  file. Both candidates' generated C runs in low/optimized MSVC profiles (4
  executions, each covering all four cases).
- Each candidate also rejects an unused invalid user function, a private SDK
  call and missing filesystem effects. The byte-loop regression passes in both
  candidate profiles (2 programs, 1 rejection, 8 native executions total).
- Evidence: `evidence/candidate-file-tokens-fixed-dev-20261009.json`,
  `evidence/candidate-file-tokens-fixed-release-20261009.json`,
  `evidence/candidate-byte-loop-calls-release-20261009.json` and
  `evidence/candidate-sdk-module-driver-release-fixed-20261009.json`.
  The development receipt retains the initial older-runner failure and its
  explicit correction: exact generated C and source digests were reused with
  the existing runner that consumes expected stderr. No compiler pass was
  repeated to repair that observation harness mismatch.
- `build_sdk_module_driver.py --compiler <stage0> --profiles development release
  --receipt <json>` builds the opt-in probe. Then run `verify_file_tokens.py
  --images <driver-dev> <driver-release> --native-runner <console-aware-runner>
  --receipt <json>`. The pre-fix driver build receipt remains at
  `evidence/candidate-sdk-module-driver-build-20261009.json` for the regression
  control's image identity.
- Item 50 still contains the complete command-dispatcher integration. The
  authored production CLI keeps its original full-AST route. Linux, whole-CLI
  native execution and stage1/stage2 promotion have not been run. No public
  language, SDK/runtime ABI, DB, dependency or CI workflow changed.

## MIR self-source parser admission (2026-10-09)

- The actual full-dispatcher source inventory initially contains 137 imported
  files and 1038311 bytes. Its Fairy compilation rejects at parsing. The native
  declaration CLI reproduces the parser limit in `hir_model`, `mir_lower`,
  `type_check` and `hir_lower`; the other eight largest files parse successfully.
  This is a node-budget failure, not a 64KiB file-byte failure. Evidence:
  `evidence/candidate-dispatcher-admission-20261009.json` and
  `evidence/candidate-self-parse-probe-20261009.json`.
- MIR lowering now separates extended values, scalar/views, aggregate/calls,
  control flow and statements from its coordinator. All six files parse within
  the unchanged limits; each is at most 15044 bytes. The original 13 function
  bodies are identical after accounting for module qualification and internal
  visibility. Shared context, source-depth increments and final MIR verification
  remain intact. No new forwarding runtime frame or allocation is introduced.
- Check and a rebuilt development candidate pass. Three programs cover views,
  arrays/casts, short circuit, enum/Result branches, foreign calls, factory moves
  and exact cleanup counts. Their emitted C is byte-identical to the pre-split
  release candidate, and all twelve low/optimized MSVC executions pass. A real
  later-argument loan conflict still rejects. Evidence:
  `evidence/candidate-mir-source-split-20261009.json`,
  `evidence/candidate-mir-source-split-cli-20261009.json` and
  `evidence/candidate-sdk-module-driver-mir-split-dev-20261009.json`.
- The updated full-dispatcher inventory is 142 files and 1041199 bytes. It still
  rejects at parsing; the remaining self-source units are `hir_model`,
  `type_check` and `hir_lower`. This does not complete item 50 or self-hosting.
  `evidence/candidate-dispatcher-admission-after-mir-split-20261009.json` retains
  that rejection. Linux, whole CLI execution and new release-candidate rebuilding
  were not repeated for this body-equivalent split. SDK/runtime, DB, dependency,
  CI runner settings and all admission budgets are unchanged.

## HIR owned-model self-source admission (2026-10-09)

- `hir_model.fy` now keeps the owned types, queries, admission caps and final
  factory. Symbol/function, expression shape, pointer/storage, value/view/enum,
  control/drop and lexical unsafe proofs live in eight focused helpers. All
  nine files parse under the unchanged limits; the model is 20360 bytes.
- The six extracted guard regions and the enum/unsafe proof bodies match their
  previous source after module qualification and ref-parameter normalization.
  The coordinator preserves proof order and returns no successful owner until
  the lexical unsafe proof also passes. Evidence:
  `evidence/candidate-hir-model-source-split-20261009.json`.
- Check and the rebuilt development driver pass. Existing native mutation
  testing rejects all 13 forged drop/receiver links. Three focused programs
  emit the same C as the pre-split release driver and pass twelve native
  low/optimized MSVC executions; a conflicting argument borrow still rejects.
  Evidence: `evidence/candidate-hir-model-split-mutation-20261009.json`,
  `evidence/candidate-hir-model-split-cli-20261009.json` and
  `evidence/candidate-sdk-module-driver-hir-split-dev-20261009.json`.
- The full-dispatcher import input now has 150 files and 1047406 joined bytes,
  still within the 1MiB source budget. It continues to reject at parsing:
  `type_check.fy` and `hir_lower.fy` self-source work remains. Evidence:
  `evidence/candidate-dispatcher-admission-after-hir-split-20261009.json`.
  New release-candidate rebuilding, Linux and whole-CLI native execution were
  not repeated for this proof-preserving split. Source/AST/work/depth limits,
  public language, SDK/runtime ABI, DB, dependency and CI settings are unchanged.

## Dispatcher diagnostic source deduplication (2026-10-09)

- `file_cli::usage` owns the identical usage text previously repeated in three
  failure branches. It returns the same static borrowed literal; command modes,
  argument bounds and console effects remain unchanged.
- Check and the actual development CLI build pass. Eighteen existing and
  missing/extra/unknown/oversized-argument cases retain exact stdout, stderr and
  status against the prior authored CLI. Evidence:
  `evidence/candidate-cli-shared-usage-20261009.json`.
- This reduces full-dispatcher source bytes within the unchanged 1MiB bound,
  leaving room for the remaining type-check/HIR-lowering source partitions.
  It is source deduplication, not a whole-dispatcher candidate-C execution proof.

## Type-check self-source admission (2026-10-09)

- `type_check.fy` retains the owned result, value/error/table helpers, block
  coordinator and kind dispatch. Tagged, scalar, call, access and record/control
  expressions, statements and declaration collection now have seven helpers.
  All eight files parse under the unchanged limits. Seven extracted regions and
  the retained helper bodies match the previous source after qualification;
  original source-depth accounting and contextual variant dispatch are retained.
- Check and the rebuilt development SDK driver pass. Three focused programs
  emit byte-identical C before/after the split and pass twelve native MSVC runs.
  Eight type-error inputs preserve exact rejection status and diagnostics; the
  existing later-argument borrow conflict also rejects. Evidence:
  `evidence/candidate-type-source-split-20261009.json`,
  `evidence/candidate-type-source-split-cli-20261009.json` and
  `evidence/candidate-sdk-module-driver-type-split-dev-20261009.json`.
- The full dispatcher inventory is now 157 files and 1054271 joined bytes.
  It still reports `parser limit`; `hir_lower.fy` remains to be partitioned.
  Import/helper overhead also puts the complete input above 1MiB, so source
  deduplication must precede full admission. Evidence:
  `evidence/candidate-dispatcher-admission-after-type-split-20261009.json`.
  Release-candidate rebuilding, Linux and whole-dispatcher native execution were
  not repeated. Public language, SDK/runtime ABI, DB, dependencies, CI runner
  settings and all admission bounds are unchanged.

## HIR-lowering self-source admission (2026-10-09)

- `hir_lower.fy` retains the block/program coordinator. Binding/signature
  preparation, expression lowering and statement lowering have three helpers.
  All four files parse under existing bounds; all five function bodies are
  identical after module qualification, including depth and edge ordering.
- Check and the development SDK driver build pass. Three focused programs
  retain byte-identical generated C and pass twelve native MSVC runs; the
  argument-borrow rejection is retained. Evidence:
  `evidence/candidate-hir-lower-source-split-20261009.json`,
  `evidence/candidate-hir-lower-source-split-cli-20261009.json` and
  `evidence/candidate-sdk-module-driver-hir-lower-split-dev-20261009.json`.
- The actual complete dispatcher now passes parsing and reaches module-program
  phase 1, code 10 at the cumulative source-byte cap. Its input is 160 files and
  1055400 joined bytes. Source deduplication remains necessary before further
  self-source admission. Evidence:
  `evidence/candidate-dispatcher-admission-after-hir-lower-split-20261009.json`.
  Release-candidate rebuilding, Linux and whole-dispatcher native execution were
  not repeated. Language, SDK/runtime, DB, dependency, runner and budget
  contracts are unchanged.

## Dispatcher source budget and shared frontend stages (2026-10-09)

- Declaration/name/type/HIR CLI stages now return their existing owned results
  through shared `compile` helpers. MIR reuses those stages while preserving its
  legacy HIR-error wording. Six output paths share the existing exact byte-write
  count check. Help reuses the usage prefix followed by its unchanged suffix.
  Unused imports and repeated module qualifications were reduced.
- HIR initializer/capture/arm/borrow/slice/byte-loop nodes use one typed default
  factory. Tagged kind/type fields are selected during construction rather than
  assigned afterward. MIR publication copies the operation and overwrites only
  function, block and next, preserving its existing link and ownership behavior.
- Check, the authored development CLI build and SDK driver build pass. Forty-six
  CLI cases retain exact stdout/stderr/status, including parser/name/type/HIR
  errors. Eight type-error cases retain exact diagnostics. Five focused programs
  (including payload captures and default slice bounds with byte loops) retain
  byte-identical generated C and pass twenty native MSVC executions. Evidence:
  `evidence/candidate-cli-source-dedup-20261009.json`,
  `evidence/candidate-type-source-dedup-regression-20261009.json`,
  `evidence/candidate-source-dedup-native-20261009.json`,
  `evidence/candidate-source-dedup-c-parity-20261009.json` and
  `evidence/candidate-sdk-module-driver-source-dedup-dev-20261009.json`.
- The actual 160-file dispatcher input shrank by 6884 bytes to 1048516 joined
  bytes, 60 bytes below 1MiB. It passes parsing and the source cap, then rejects
  at module-program phase 1 code 9 on the by-value `usize[20]` parameter of
  `mir_output::row`. Source-set signature type resolution currently has no
  fixed-array branch. That is the next measured blocker; very little source
  budget remains for further implementation. Evidence:
  `evidence/candidate-dispatcher-admission-after-source-dedup-20261009.json`.
- Public language and SDK/runtime ABI, DB, dependency and CI runner settings
  are unchanged. Compiler-internal CLI/factory helper APIs changed. Release
  candidate rebuilding, Linux and full-dispatcher native execution remain
  unverified; the unchanged admission bounds still reject the complete program.

## Source-set fixed-array signatures (2026-10-09)

- Detailed AST remapping notes moved verbatim from `module_ast_join.fy` to
  `docs/architecture/candidate-source-join-v0.md`. All non-comment source text
  is unchanged; check passes. This provides source budget for implementation
  without increasing a bound. Evidence:
  `evidence/candidate-join-documentation-split-20261009.json`.
- Source-set graph kind 16 owns a canonical element edge and exact array length.
  Resolution accepts the existing rank-one 1..4096 and rank-two 1..64 shapes,
  retaining module/declaration identity for nominal/enum leaves and rejecting
  resource/borrow elements, larger rank-two and higher ranks. Graph validation
  rejects malformed/forward/self edges, lengths and scalar/module payloads;
  failed resolution rolls back speculative child interning.
- Development/release SDK drivers build. Three exact API tests pass in each
  native profile, including prior nominal signatures, canonical alias/length
  identity, privacy/FFI rejection and forged graph/rollback cases. Five actual
  source-set programs pass across both candidate profiles with byte-identical
  C and twenty native MSVC executions: array arguments/results, usize[1153],
  imported nominal arrays, rank two, enum and Option arrays. Ten rejection
  inputs per candidate cover mismatched length/type/nominal owners and bounds,
  resource/borrow elements and foreign arrays. Evidence:
  `evidence/candidate-module-array-signatures-api-20261009.json`,
  `evidence/candidate-module-array-signatures-dev-20261009.json`,
  `evidence/candidate-module-array-signatures-release-20261009.json` and
  `evidence/candidate-module-array-signatures-parity-20261009.json`.
- The actual dispatcher passes signature registration, including
  `mir_output::row`'s usize[20] argument. Its 160-file input is 1046965 joined
  bytes. It now reaches SDK compaction phase 8 code 11 on the capped reference
  index; `sdk_reachability` currently conflates 16384 indexed uses with an
  allocation failure. That index/work representation is the next blocker.
  Evidence: `evidence/candidate-dispatcher-admission-after-array-signatures-20261009.json`.
- This extends the private source-set type graph, not the language or SDK ABI.
  Source/AST/work/depth/stack budgets, DB, dependencies and CI settings are
  unchanged. Linux and full-dispatcher C/native execution remain unverified;
  stage1/stage2 and promotion remain open.

## Reachability reference-limit classification (2026-10-09)

- The existing 16384-use cap now returns limit code 10 separately from a failed
  Vec push's allocation code 11. All other source text and the index/work bounds
  are unchanged. The rebuilt development driver confirms the full dispatcher
  stops at phase 8 code 10 at the same source span, proving the capped index is
  the measured cause. Current input: 160 files, 1047016 joined bytes.
- Two existing native SDK compaction tests pass with the actual SDK fixture
  path forwarded: private references retain their diagnostic, and exact source
  spans/user validation survive compaction. The initial invocation omitted that
  required path and failed; the receipt records that command error and the
  corrected run. Evidence:
  `evidence/candidate-reachability-limit-classification-20261009.json` and
  `evidence/candidate-dispatcher-admission-reach-limit-classified-20261009.json`.
- Index reduction remains open. Release rebuilding, Linux and whole-dispatcher
  C/native execution were not repeated for this diagnostic-only change.

## Reachability declaration ranges (2026-10-09)

- The per-name linked `Use` list is replaced by a first/stop expression range
  per declaration. Queued declarations visit every original name expression;
  the interval is checked against the declaration's source span. No user or
  SDK target is filtered out by spelling or call shape. This removes per-name
  storage pressure while keeping the 4096-row/cache and 1MiB work bounds.
- Development check/build and three exact native API tests pass. Twenty-two
  real parsed modules with 17600 repeated local references retain every user
  body, source byte and token span through compaction. Existing private-target
  and unused-invalid-user compaction checks pass. The real file-token closure
  emits byte-identical C before/after and retains its four exact file cases
  across four native executions; three rejection cases are retained. Evidence:
  `evidence/candidate-reachability-ranges-20261009.json` and
  `evidence/candidate-file-tokens-reach-ranges-20261009.json`.
- The 160-file full dispatcher is 1046901 joined bytes and passes the former
  reference-list blocker. It now reaches the work bound during name lookup in
  `numbers.fy`, phase 8 code 10. Catalog lookup still scans global tables and
  reachability charges their whole sizes for each cache miss. Reducing those
  lookup scans is next. Evidence:
  `evidence/candidate-dispatcher-admission-reach-ranges-20261009.json`.
- Release-candidate rebuilding, Linux and full-dispatcher native execution were
  not repeated. Internal Row storage changed; language/SDK ABI, DB, dependency,
  CI runner settings and all numeric admission bounds are unchanged.

## Module-local catalog lookup (2026-10-09)

- Lookup finds each module's ordered import/declaration range by binary search.
  Reachability charges the rows actually visited, reuses declaration ownership
  for adjacent expressions, and skips unqualified user names because every user
  declaration survives. Qualified names, SDK-local dependencies and private
  targets still enter the ordinary lookup and validation paths.
- Two native catalog tests and three SDK compaction tests pass in development.
  The ordered-table case checks 22 modules and 1364 declarations, including
  aliases with trivia, missing names, private targets and both end modules.
- The 160-file dispatcher (1048411 joined bytes) passes name reachability and
  reaches phase 8 code 10 at `std/vec.fy:0..0` during AST compaction. Reducing
  repeated ownership searches and retained-copy work is next. Evidence:
  `evidence/candidate-catalog-ranges-20261009.json` and
  `evidence/candidate-dispatcher-admission-catalog-ranges-20261009.json`.
- Release rebuilding, Linux and full-dispatcher C/native execution remain open.
  Numeric bounds, language/SDK ABI, DB and CI settings are unchanged.

## Cached AST ownership and token-qualified names (2026-10-09)

- User-local names are identified by the parser's first token end instead of
  rescanning spelling bytes. Qualified names with intervening trivia still use
  catalog lookup. SDK arena mapping reuses a containing function row; the
  enclosing module row is never cached as a function owner. User nodes retain
  their identity mappings without declaration searches.
- Three development native compaction tests pass. The dense fixture now uses
  17600 long local names across 22 modules and requires reachability work below
  50000. Exact sources/tokens/arenas, private-target diagnostics and invalid
  unused-user checking remain covered, including qualified-name trivia.
- The dispatcher remains within 1MiB at 1048534 joined bytes and advances to
  `bodies.fy:0..0`, still phase 8 code 10 during copying. Moving unchanged user
  AST owners through the CLI can eliminate their duplicate arena/scanner copy.
  Evidence: `evidence/candidate-ast-owner-cache-20261009.json` and
  `evidence/candidate-dispatcher-admission-owner-cache-20261009.json`.
- Development check/build pass. Release/Linux and full-dispatcher C/native
  execution remain open; no numeric limits, SDK ABI, DB or CI settings changed.

## Moving unchanged user AST owners (2026-10-09)

- `sdk_ast_compact::compact_in_place` validates original metadata and computes
  every SDK replacement before touching the input vector. It reserves transfer
  storage, moves user AST/scanner owners in manifest order, and replaces only
  SDK owners. Copy or reserve failure leaves the original vector intact. The
  borrowed compaction API remains available for exact snapshot comparison.
- The opt-in `module_program::compile_sdk` now takes a mutable source vector;
  its CLI caller retains exact source owners for diagnostics after compilation
  fails. Signatures are rebuilt against the compacted declaration coordinates.
- Three development native tests pass. Twenty-two modules with 17600 long
  local references move and retain all source/token/arena observations under
  50000 total work units. Tests also compare copied versus moved SDK snapshots,
  reject stale original metadata without changing the input, and retain typing
  rejection for invalid unused user code.
- The actual dispatcher (160 files, 1048460 joined bytes) passes SDK compaction
  and reaches phase 2 code 10 at the joined scanner. Its error span `0..0` maps
  to `scanner::collect` storage rejection, rather than a per-module AST cap.
  The scanner's fixed 65536-byte cooked pool is the next storage-reduction
  candidate; no completed joined AST, C output or full-dispatcher native run is
  claimed. Evidence: `evidence/candidate-ast-owner-transfer-20261009.json` and
  `evidence/candidate-dispatcher-admission-in-place-20261009.json`.
- The real file-token closure still emits byte-identical C to the saved catalog
  baseline. Four exact file cases, three rejection cases and two native runs
  pass. Evidence: `evidence/candidate-file-tokens-in-place-20261009.json`.
- Private arena and manifest notes moved into
  `docs/architecture/candidate-source-join-v0.md` to retain the 1MiB input bound.
  Release rebuilding and Linux remain open. Only the candidate mutable-vector
  API changed; numeric bounds, language/SDK ABI, DB and CI settings did not.

## Joined cooked storage sharing (2026-10-09)

- The joined scanner shares equal nonempty canonical cooked buffers through a
  private 256-slot index. Exact comparison handles collisions and a charged
  table search retains values added after the index fills. The 65536-byte pool
  bound now counts distinct owned payloads; ordinary file scanning is unchanged.
  Work exhaustion retains E0010 and the module join rejects the incomplete
  stream before publishing any AST arenas.
- Mutable record parameters can expose scalar fixed-array field storage through
  the existing mutable view path. HIR and MIR retain the original record owner;
  shared parameters and access to a sibling while the view is held are rejected.
  The scanner reads and writes its index through this admitted path.
- Development check and SDK observation-driver build pass. Eight native API
  tests cover sharing, hash collisions, cross-kind equality, exact snapshots,
  pool capacity, full-index fallback, work exhaustion and array field storage.
  Seven array-view programs, nine rejections and fourteen low/optimized native
  runs pass. Four file-token cases retain exact reference stdout/stderr/status;
  three rejections and two low/optimized native runs also pass. Generated C
  changed with the scanner implementation; no byte-identical C claim is made.
- The authored storage probe observes the actual 160-file dispatcher at 1045810
  joined bytes: logical cooked data is 69949 bytes, distinct owned data is 41226
  bytes, and joined scanning has zero diagnostics. The dispatcher advances to
  AST joining at module 18 (`module_cli.fy`), where the compacted source set has
  1049 entries against the unchanged 1024-entry bound. Other cumulative counts
  there are 286 declarations, 10562 expressions, 3705 statements, 1752 types,
  68 fields, 86 attributes and 93 attribute arguments. Arena partitioning or
  reduction is next; full dispatcher C/native and stage1/stage2 remain open.
- HIR/MIR numeric notes moved into the architecture model documents to keep
  the actual input below 1MiB. Language/SDK ABI, numeric limits, DB and CI are
  unchanged. Release rebuilding and Linux validation remain open.
  Evidence: `evidence/candidate-cooked-storage-20261009.json`,
  `evidence/candidate-dispatcher-admission-cooked-intern-20261009.json`,
  `evidence/candidate-sdk-module-driver-cooked-intern-dev-20261009.json`,
  `evidence/candidate-file-tokens-cooked-intern-20261009.json` and
  `evidence/candidate-mutable-record-array-view-native-20261009.json`.

## Bounded whole-module AST units (2026-10-09)

- `module_units` measures actual parser arenas and groups manifest-ordered
  whole modules into maximal contiguous ranges. The existing source-set bounds
  and every joined arena cap remain unchanged. No unused user body is discarded.
- `module_ast_join::join_range` borrows the original source vector and remaps
  only its selected range. Each returned AST starts at local byte offset zero,
  while rejection positions retain the original full-vector module index.
  Existing `join` behavior is retained by delegation to the full range.
- Development check and probe build pass. Eight native API tests pass, including
  a 1206-entry input that the full join rejects but two bounded owners preserve,
  exact bytes, original error coordinates, invalid ranges, and the existing
  module-join and cooked-storage regressions.
- The actual dispatcher is 160 modules and 1046245 joined bytes. All modules
  join successfully into twelve bounded owners, retaining 152020 expressions
  and 11509 entries. Observed group maxima are 305 declarations, 16234
  expressions, 4002 statements, 1017 entries, 1899 types, 342 fields, 159
  attributes and 163 attribute arguments. No arena bound was increased.
- This completes the AST grouping layer, not compiler integration. The current
  CLI still attempts one joined owner. Cross-unit names/types, HIR/MIR targets,
  C symbols and final linking are next. Full C/native compilation, stage1/stage2,
  release rebuilding and Linux remain open. The architecture owner is
  `docs/architecture/candidate-compilation-units-v0.md`; evidence is
  `evidence/candidate-module-units-20261009.json`. Language/SDK ABI, DB, CI and
  runner implementations are unchanged.

## Owned names across AST units (2026-10-09)

- `module_unit_bind` validates the whole live source set against signatures,
  creates one bounded range AST, and owns local definitions and imported named
  type/value targets in original module/declaration coordinates. Repeated
  provider aliases share a target; equal declaration numbers in other providers
  remain distinct. Coordinates survive original parser/signature owner removal.
- Private imported types/functions and stale providers outside the selected
  range fail closed with original module-local spans. Generic parameters,
  builtin types and raw compiler intrinsics keep their existing handling. This
  layer records declarations; local body typing is not yet connected.
- Development check and observation-probe build pass. Five native regressions
  pass, including identity/alias/owner retention, privacy, stale provider bytes,
  generic parameters, original errors, invalid ranges and partition behavior.
- The actual dispatcher is 160 modules and 1046447 joined bytes. All twelve
  groups bind successfully, retaining 152032 expressions and 11513 entries.
  Per-group unique targets sum to 2146 (maximum 252); measured work peaks at
  134589 within the existing 1048576 budget. The probe compares every retained
  coordinate to its original parser declaration. No numeric cap was increased.
- Next: consume these coordinates in unit body name/type checking, including
  exact imported layouts/signatures, then HIR/MIR/C and CLI integration. Full
  dispatcher compilation, stage1/stage2, release rebuilding and Linux remain
  open. Internal candidate query/helper APIs expanded; language/SDK ABI, DB,
  CI and runner implementations are unchanged. Evidence:
  `evidence/candidate-module-unit-bind-20261009.json`; architecture:
  `docs/architecture/candidate-compilation-units-v0.md`.

## Owned provider interfaces for units (2026-10-09)

- `module_unit_interfaces` follows selected target/signature types through a
  deduplicated canonical graph queue. It owns needed provider field names/types,
  enum payload ranges, callable arguments/results and declaration flags,
  foreign link names, repr(C) admission, resource abilities and declared cleanup
  authority. Generic function interfaces are explicitly deferred templates.
- Retained bindings are checked against live caller bytes and fresh catalog
  lookups, so rebuilding metadata cannot make a stale private target usable.
  Canonical field queries preserve resource module privacy and reject ordinary
  enum payload access. Containers stay local until the owned result is built.
- Development check, probe build and six native regressions pass. Tests exercise
  owner removal, nested nominal/array fields, payload ranges, callable/foreign
  flags, field queries, templates, stale callers/providers and binding behavior.
- All 160 dispatcher modules (1046575 bytes) now collect provider interfaces in
  thirteen AST groups. Maxima are 131 shapes, 910 fields, 198 callables and 401
  graph types. Projection work peaks at 10629 and binding work at 146397; the
  existing numeric caps remain fixed. Original shape/field spans and names are
  independently compared by the Fairy probe. Evidence:
  `evidence/candidate-module-unit-interfaces-20261009.json`.
- Repr(C) lane comments moved unchanged to
  `docs/architecture/candidate-repr-c-model-v0.md` to keep input within 1MiB.
  Next: local annotation resolution and actual body checking against these
  interfaces, then HIR/MIR/C/CLI integration. Full dispatcher compilation,
  stage1/stage2, release rebuilding and Linux remain open. Internal compiler
  query APIs expanded; language/SDK ABI, DB, CI and runner implementations did
  not change.

## Local annotation roots (2026-10-09)

Fairy C's module_unit_interfaces now maps declaration results, parameter and
local entries, field types, cast targets, explicit specialization arguments, and
byte-loop binder types from joined AST root IDs to canonical graph IDs.
Resolution runs against the original Parsed owners and original module
coordinates while preserving original error spans, and local annotation types
seed provider field projection. Generic declaration roots carry required=true,
deferred=true, and type=0 until specialization, whereas all unrequested child
and list nodes carry required=false. Interfaces owns this mapping, so no
borrowed owner survives.

Eight development native regressions pass. The actual dispatcher processes 160
modules (1046575 bytes) in thirteen units, with 10289 concrete annotation roots
and 69 deferred template roots. Group maxima are 409 canonical graph types, 131
shapes, 910 fields and 198 callables; projection work peaks at 27474. Existing
numeric limits remain fixed. Evidence: `evidence/candidate-module-unit-annotations-20261009.json`.

Next: specialize deferred templates and connect local symbol/expression/body
typing to the owned annotations, provider fields and callable signatures.
Annotation collection does not validate initializer, argument or return
expression compatibility. CLI/HIR/MIR/C unit integration, target layouts,
transitive cleanup effects, stage1/stage2, release rebuilding and Linux remain
open.

## Owned generic signature instances (2026-10-09)

`module_unit_generics` builds an owned per-unit plan of explicit generic-call
signature instances. Requests are keyed by the original provider module and
declaration together with the ordered canonical type arguments, so alias and
repeated requests share one entry while distinct providers and argument order
stay separate. Each instance owns its result and parameter graph IDs, effects,
must-use and unsafe flags, and per-expression request identities. Canonical type
resolution substitutes generic names without touching the original AST or source
bytes, which keeps provider error spans intact, and admits a defined set of
generic wrappers: safe borrows, scalar slices, pointers, fixed arrays,
Option/Result, and the authenticated SDK Vec.

Admission stays narrow. Ordinary user functions accept only the six existing
builtin scalars, and exact authenticated `std::vec` sources additionally accept
named non-generic structs and resources; unrequested generic-body calls remain
explicitly deferred, and exact SDK storage intrinsics are flagged separately.
The existing caps are retained: 8 type args, 16 instances per template, 64 per
unit, canonical graph 4096, parameters 32768, and work 1048576. Failed plan
collection rolls graph appends back, and the interface builder follows the
resulting signature types for provider fields.

Development check and probe build pass, along with thirteen native regressions
covering alias reuse, provider/argument-order identity, owned signature queries,
pointer/slice/array substitution, rejected argument/contract/shadow cases,
atomic graph rollback, the 64-instance cap and exact SDK resource admission. The
actual 160-module dispatcher remains within 1MiB at 1048186 bytes. The initial
thirteen AST groups pass interface specialization for the first two groups (12
and 61 instances), then reject module 37 at 4953..4970 with E4503. AST-only
packing must now refine its ranges under the existing specialization limits.
Full dispatcher interface planning is incomplete. Evidence:
`docs/ops/evidence/candidate-module-unit-generics-20261009.json`.

After range refinement, the owned instance keys and canonical signatures must
drive template body cloning, nested specialization and recursion checks, local
expression/body typing, and HIR/MIR/C/CLI consumers. Target layout, transitive
cleanup effects, stage1/stage2, release rebuilding and Linux remain open. Public
language/SDK ABI, DB, CI and runner implementations are unchanged.

## Semantic pressure range refinement (2026-10-09)

Fairy C's module_unit_plan starts from the existing AST arena grouping and, for
each range, checks binding together with owned interfaces and signature
specialization. It bisects only pressure failures code10 or E4503 while
preserving ordered source module/declaration coordinates and every input
declaration and body; ordinary semantic, privacy, and stale errors return
immediately, and an irreducible single-module error stays rejected. Planning is
bounded by 1048576 work and depth 10, and each admission retains the existing
arena/type/field/shape, 64-total-instance, and 16-per-template bounds. The
planner returns owned ranges and observed admission maxima, and the current
observation independently reconstructs each admitted interface afterward.

Development check, probe build and two new native planner regressions pass. The
earlier thirteen generic/interface/type regressions remain valid after
rechecking their original compiler, source, SDK fixture and test hashes. The
complete 160-module dispatcher (1048186 bytes) now admits fifteen ranges after
seventeen attempts and two splits. Planning work is 197; observed binding and
admitted-interface work maxima are 146311 and 81469. All fifteen interfaces are
reconstructed and checked against input syntax. Per-range maxima are 61
specialization instances, 433 graph types, 131 shapes, 907 fields and 164
callables. Per-range instance counts sum to 476, linking 1281 explicit requests;
fourteen generic-body requests remain deferred. Annotation counts are 10327
concrete and 69 deferred. Evidence: `docs/ops/evidence/candidate-module-unit-plan-20261009.json`.

Next: clone owned template bodies using provider coordinates and the ordered
type arguments, then process forwarded nested requests with recursion/depth
checks. Graph IDs remain owner-local; cross-unit instance/link deduplication
must compare structural type identities. Local expression/body typing, target
layouts, transitive cleanup effects, HIR/MIR/C/CLI integration, stage1/stage2,
release rebuilding and Linux remain open. Public language/SDK ABI, DB, CI and
runner implementations are unchanged.

## Transitive generic function request graphs (2026-10-09)

This stage expands the generic function request graph from each owned signature
instance, reading the original provider AST without modifying source or copying
bodies yet. Forwarded parameter arguments are substituted in provider
coordinates, and private same-provider calls plus imported visibility are
resolved against fresh metadata. Function instances are deduplicated by provider
module/declaration together with ordered graph IDs. DFS active/completed states
reject direct, indirect, and changing-argument specialization recursion, with
the existing depth-32 bound also applying to a previously completed leaf at that
call depth. Owned dependency records are keyed by caller instance and original
provider expression ID. The design differentiates function edges, authenticated
SDK storage intrinsic element/slice facts, and authenticated Vec<T> aggregate
construction; the aggregate branch fixed an initial assertion failure in the SDK
resource-element native case. Constructor typing uses canonical graph interning
with final catalog graph verification, while interface projection follows
dependency type seeds and keeps materialized Vec shapes capped at the existing
64. Other existing bounds hold: 64 function instances, 16 per template, 8 args,
graph 4096, dependency records 16384, and work 1048576.

Development check, probe build and ten native regressions pass. They cover
direct/indirect/changing-argument recursion, a depth-32 request to an already
completed leaf, ordered argument forwarding through a private provider helper,
alias reuse, rollback, quota and semantic-error preservation, and exact SDK
resource admission. The complete dispatcher is 160 modules and 1048277 bytes,
admitted as eighteen units after five splits and twenty-three attempts. Per-unit
maxima are 57 function instances, 433 canonical graph types, 124 shapes, 888
fields and 164 callables; binding/interface work peaks at 146700/90658. Per-unit
function instance counts sum to 592 for 1281 initial calls, with 116 nested
function edges, 594 private storage intrinsic records and 100 Vec aggregate
records. Original template syntax still carries fourteen deferred request slots;
instance-specific dependency records resolve those sites separately. Evidence:
`docs/ops/evidence/candidate-module-unit-generic-dependencies-20261009.json`.

Next: own the provider function AST bodies and authenticated SDK Vec cleanup
bodies, specialize resource cleanup contexts and their nested/implicit drop
dependencies, and connect local symbol/expression/body typing. Canonical graph
IDs stay owner-local; structural cross-unit instance/link deduplication, target
layouts, transitive cleanup effects, HIR/MIR/C/CLI integration and stage1/stage2
remain open. Release rebuilding and Linux are deferred. Public language/SDK ABI,
DB, CI and runner implementations are unchanged.

## Source-stamped generic plan validation (2026-10-09)

Generic plans retain original module byte lengths/SHA256 and their canonical
graph prefix. Development native tests pass 11/11, including changed provider
bodies with refreshed metadata, missing modules, shortened/altered prefixes and
permitted suffix extension. The unchanged 160-module dispatcher corpus still
admits 18 units after 23 attempts and 5 splits; binding/interface work peaks at
146700/91081. Development probe check/build and source/tool identities pass.
Bounds remain unchanged. Provider function and SDK Vec cleanup body ownership
remain next; release, Linux and stage1/stage2 were not run in this unit.
Evidence: `docs/ops/evidence/candidate-module-unit-source-stamps-20261009.json`.

## Owned generic function body arenas (2026-10-09)

`module_unit_bodies` owns requested concrete function ASTs, ordered signature
arguments/parameters, canonical annotation overlays and remapped dependency
edges. `module_unit_body_ast` selects only the requested declaration's nodes and
copies its attributes; source spans and token IDs still address a scanner stream
owned once per provider module. Original sources, signatures and interfaces may
be released without invalidating the retained body plan. The existing hex
cooked-literal representation is preserved without decoding or textual type
substitution.

Admission counts original plus cloned nodes under existing byte, declaration,
expression, statement, entry, type and attribute bounds. Capacity pressure
refines ranges using the existing code 10/E4503 rule. Development check/probe
build and seven native tests pass. The full 160-module corpus passes in 25 units
after 37 attempts/12 splits; per-unit counts sum to 615 owned bodies, 3249
resolved roots and 838 edges. Body work peaks at 93777; combined expressions
peak at 16293 and entries at 951. The initial fixture assertions expected one
storage intrinsic in SDK empty and decoded string bytes, but the existing source
has two storage intrinsics and scanner cooked bytes are hex encoded. Corrected
assertions pass. Evidence:
`docs/ops/evidence/candidate-module-unit-bodies-20261009.json`.

Next: SDK Vec cleanup AST ownership and resource cleanup specialization with
nested/implicit drop dependencies. Local expression/body typing, new local type
field projection, structural cross-unit deduplication, byte layouts and cleanup
effects, HIR/MIR/C/CLI, whole dispatcher native compilation and stage1/stage2
remain open. Release rebuilding, Linux and push are deferred. Public language
and SDK ABI, DB, CI and runner implementations are unchanged.

## Owned cleanup contexts and SDK leaf inlining (2026-10-09)

Required resource/Vec shapes request concrete cleanup contexts and owned AST
arenas, retaining dropping receiver kind 10, original positions, declared
effects/abilities, ordered element arguments and per-context annotation roots.
Cleanup request edges use their own owner table; normal function instance IDs
and scalar/SDK argument restrictions remain unchanged. Immutable local-vector
builders avoid mutable resource-field projection. Original plus cloned node
admission includes cleanup arenas, and expansion failures roll added types back.

The initial full-corpus attempt failed at module 7, bytes 451..463, with E4503:
cleanup specialization exceeded the existing `take_slot<T>` per-template quota.
Only the exact authenticated SDK leaf is folded in cleanup contexts after its
single unsafe forwarding expression and signature are structurally checked.
The edge retains helper declaration and unsafe provenance for later typing and
lowering. General calls and all capacity limits remain unchanged.

Seven development native tests, development check/probe build and source/tool
hashes pass. The 160-module, 1048472-byte corpus passes as 26 units after 39
attempts/13 splits. Per-unit totals: 625 function bodies, 763 cleanup contexts,
3624 cleanup annotation roots and 595 cleanup request edges. Body work peaks at
495691; combined entries at 1015 within the 1024 cap. Evidence:
`docs/ops/evidence/candidate-module-unit-cleanups-20261009.json`.

Next: implicit/nested drop dependencies and transitive cleanup effects. Body
symbol/expression typing with inline provenance, new local type field projection,
cross-unit structural deduplication, target layouts, HIR/MIR/C/CLI and
stage1/stage2 remain open. Release rebuilding, Linux and push were not run.
Public language/SDK ABI, SDK sources, DB, CI and runner implementations are
unchanged.

### Type-level nested drop graph (2026-10-09)

`module_unit_drops` derives an immutable type-level graph from owned interfaces.
Field and Vec edges retain provider coordinates; Option, Result and array edges
are structural. Declared cleanup effects and no-drop abilities propagate through
contained values. References and pointers have no drop obligations. Recursive
Vec containment reaches a bounded monotone fixed point. Unprojected nominal
metadata remains explicitly incomplete and `required()` rejects it with code 9.
Admission includes drop planning; code 10 uses the existing range refinement.
Type, dependency and work bounds remain unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-drops-20261009.json`.

Development check/probe check/build and seven native tests pass. The current
160-module, 1048472-byte input retains 26 units after 39 attempts and 13 splits.
Drop edge counts sum to 9878 across units, with maxima of 425 graph nodes,
660 edges and 6036 work. Combined original/body entries remain 1015/1024.

Cleanup-body locals, inferred expression types, move-aware obligations and
generic-body local field expansion remain next. Full dispatcher native
compilation, HIR/MIR/C/CLI, stage1/stage2, release rebuilding, Linux and push
remain pending. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged.

### Generic-body and cleanup type-demand closure (2026-10-09)

`module_unit_body_types` scans the original provider AST for requested generic
function instances and resource/SDK Vec cleanup contexts. It collects explicit
local, parameter and return types, casts, explicit generic arguments, byte-loop
types and named non-generic record constructor types. Each request records the
context kind/ID, provider module/declaration, original type root or constructor
expression ID, canonical type and source span.

Interfaces add these types to the existing pending queue and collect their
fields and cleanup contexts. Newly requested cleanup helpers are then scanned
in the same closure, until no further types or cleanups remain. Each function
instance and cleanup context is scanned once. Unrequested generic templates
remain deferred. Failed collection rolls graph additions back. The existing
source/hash/privacy and scalar-only user generic boundaries remain unchanged.
New local field/cleanup discovery is implemented; expression type matching,
name binding, inference from ordinary function-call results, move-aware cleanup
obligations and inline provenance consumption remain open. Public language/SDK
ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-body-types-20261009.json`.

Development check/probe check/build and eight native regressions pass. The
current 160-module, 1048472-byte dispatcher input retains 26 units after 39
attempts and 13 splits. Per-unit demand counts sum to 6913 (3289 function and
3624 cleanup annotations), with at most 443 demands per unit. Interface work
peaks at 265344 within 1048576; combined original/body entries remain 1015/1024.
Every admitted unit verifies unchanged canonical graph size after cloning.
The private-constructor discovery path is exercised by native regressions;
the dispatcher corpus has no such additional constructor requests. Release
rebuilding, Linux, whole dispatcher native compilation and stage1/stage2 were
not run in this unit.

### Owned lexical binding for concrete bodies (2026-10-09)

`module_unit_names`, `module_unit_name_model` and `module_unit_name_walk` resolve
requested generic function and resource/SDK Vec cleanup ASTs. Current source
stamps, canonical graphs, ordered arguments and provider streams are checked
before publishing any result. Parameters, let/var declarations, payload patterns,
byte iteration, regions and the dropping receiver have separate symbol kinds.
Explicit annotation types remain canonical; inferred locals and payload types
remain unresolved. Entry/pattern/use tables and provider module/declaration
coordinates retain original expression IDs and spans.

Initializers resolve before local registration. Nested scopes restore the active
name stack; switch bindings stay in their arm, classic for post expressions are
resolved before body locals, and byte iterable expressions precede iteration
bindings. Statement lists are reversed once for linear source-order traversal.
The existing 128-active-local, depth-64 and 1048576-work bounds are maintained;
new aggregate symbol/reference tables use the existing 16384 arena bound.
Capacity failures use existing unit refinement. Duplicate, missing, private,
out-of-scope and stale references return their original provider locations.
Resource fields remain private and immutable table accessors preserve the
existing resource-field borrowing contract.

Cleanup `this` is kind 10 with a resource type and no ordinary by-value parameter.
References record lexical unsafe function/block context; SDK storage names are
classified only in authenticated std::vec source. The safe `take_slot<T>` wrapper
has an unsafe intrinsic in its body: its source call can have `unsafe_context`
false while the separate request edge retains `inline_unsafe` provenance.
Binding records these facts without granting unsafe authority or performing
expression/flow checks. Ordinary unit bodies, expression typing, implicit drop
ownership/effect checks, inline provenance consumption, structural links,
layouts, HIR/MIR/C/CLI and full dispatcher native/stage1/stage2 remain pending.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus | 160 modules; 1048472 bytes |
| Admitted units | 26; 39 attempts; 13 splits |
| Per-unit function / cleanup totals | 625 / 763 |
| Per-unit symbol / reference totals | 3963 / 10161 |
| Maximum symbols / references | 260 / 648 |
| Maximum resolved-unit name work | 10639 / 1048576 |
| Maximum combined entries | 1015 / 1024 |

### Lexical binding for ordinary unit bodies (2026-10-09)

module_unit_ordinary_names binds every non-generic function body within an
admitted original module range, reusing module_unit_body_ast and
module_unit_name_walk with borrowed original scanner streams. The temporary
body AST is discarded, while immutable resolved name tables retain the
original module and declaration, expression, and entry IDs and spans. Explicit
canonical annotation roots are read from the exact same bounded interface
range, and interface range mismatches and stale source stamps are rejected
before publication. Generic templates stay deferred, and requested generic and
cleanup binding stays in module_unit_names. Unit admission accounts for
combined ordinary, generic, and cleanup symbols, references, and name work
under the existing 16384, 16384, and 1048576 bounds, with the existing 128
active locals and depth 64 unchanged. Expression type checking, ordinary-call
result inference, move-aware drop obligations, unsafe and inline provenance
consumption, cross-unit structural dedup and layout, HIR, MIR, C, and CLI, and
the full dispatcher native, stage1, and stage2 remain unfinished. The public
language and SDK ABI, SDK sources, DB, CI, and runners are unchanged.

Evidence: `docs/ops/evidence/candidate-module-unit-ordinary-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Ordinary functions / symbols / references | 1175 / 7479 / 48136 |
| Maximum combined symbols / references | 723 / 4094 (each <= 16384) |
| Maximum combined name work | 751853 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Observation execution timeout | Initial 300s run timed out after 23 units; final 600s selection passed all 26 |

### Declared call results and interface closure (2026-10-09)

The fairy compiler's module_unit_calls records declared-call canonical results
and parameters, along with provider and original expression coordinates. It
also records effects, unsafe, and foreign flags after lexical binding.
module_unit_call_closure repeatedly seeds unseen ordinary callable signatures
into the existing bounded field and cleanup closure. Cleanup helper calls can
request more result types. Local name shadows and stale input are rejected.
Generic calls, closed builtins, and variant constructors stay separate.
Existing arena bounds remain unchanged, with at most 64 rounds and 1048576
units of call-projection work. Full expression and argument type matching,
inferred local types, ownership and effects, unit HIR/MIR/C/CLI, and self-
hosting all remain pending.

Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-call-results-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Per-unit declared / specialized call-site totals | 11189 / 2622 |
| Per-unit variant / closed-builtin totals | 610 / 4 |
| Maximum closure rounds / work | 2 / 64; 305763 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Native closure-chain fixture | New call-result fields and cleanup-returned resources close after at least 3 rounds |
| Array reference regression | Kind-24 element links use right; a declared call beneath a cast retains its callee binding |
| Remaining expression consumers | Argument/operator matching, inferred-local symbol types, ownership and unsafe/inline checks |
| Deferred checks | Release, Linux, full dispatcher native and stage1/stage2; no push |

### Known values, inferred locals and matched returns (2026-10-09)

`module_unit_values` collects generic, cleanup and ordinary contexts. Its
immutable value, local and return tables retain provider module/declaration,
original expression/entry IDs and source spans. Declared and explicitly
specialized function results seed known values. Direct literals take local or
return context through parentheses; integer range errors are retained.
Unannotated locals infer from known initializer types, and local references and
parentheses propagate those types. Only resolved returns are matched.
Operators, casts, fields, arguments, variants, joins and builtin result typing
remain deferred, along with ownership, effects and unsafe/inline validation.
Existing canonical IDs are preserved; missing primitive/literal descriptors
append to a separate bounded graph. Stale source stamps are rejected before
publication. Existing arenas, 64 fixed-point rounds and 1048576 work remain
bounded, with code-10 admission using the existing range-split planner.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher input snapshot | 160 modules; 1048562 bytes |
| Admitted units / attempts / splits | 26 / 39 / 13 |
| Per-unit context / expression totals | 2563 / 186137 |
| Known / unresolved / non-value expression totals | 58300 / 68976 / 58861 |
| Local / inferred local totals | 11442 / 539 |
| Return / checked return totals | 4783 / 2876 |
| Maximum value work / graph | 754886 / 1048576; 427 / 4096 |
| Maximum combined entries / type syntax nodes | 1015 / 1024; 2242 / 4096 |
| Preserved local / iteration bounds | 128 active locals; 64 fixed-point rounds |


The same dispatcher input snapshot is reused for comparison. The new value
modules are tested as implementation code, but are not added to the 1MiB input
snapshot. This observation is not full current-source dispatcher compilation.
Argument/operator consumers and move-aware cleanup, unsafe/inline provenance,
structural cross-unit links/layouts and unit HIR/MIR/C/CLI remain unfinished.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Release, Linux, full dispatcher native and stage1/stage2 remain deferred; no push.
Evidence: `docs/ops/evidence/candidate-module-unit-values-20261009.json`.

### Context-driven scalar operators (2026-10-09)

`module_unit_value_scalar` types builtin integer arithmetic, bitwise/shift,
comparisons, boolean logic/negation and signed unary operators within local
initializer and return roots. Pure integer literal trees remain flexible; a
typed peer anchors their scalar type while original source links retain
execution order. Direct signed i32/i64 minimum operands retain a unary minimum
marker. Unknown field/cast/builtin anchors keep literal peers unresolved, avoiding
premature i32 range errors. A primitive descriptor cache per value plan replaces
repeated graph scans. Existing canonical IDs, arenas, 64 rounds and 1MiB work
remain bounded across ordinary, requested generic and cleanup contexts.

| Observation | Result |
| --- | --- |
| Development check | 185 source files; 2298 MIR functions; passed |
| Selected Fairy native tests | 8/8 passed |
| Reference fixture acceptance / selected primary code and span | 8/8 matched; additional cascade output retained |
| Unsupported reference case | Experimental exit 2 and span, distinct from an E diagnostic |
| Integer range correction | E1013, replacing the earlier internal 1008 |

Call arguments and other expression/control/place/ownership consumers, inline
and unsafe provenance, unit HIR/MIR/C/CLI and self-hosting remain unfinished.
Full dispatcher input observations are grouped with the next argument unit;
this unit makes no new full-corpus claim. Release, Linux, full dispatcher native,
stage1/stage2 and push were not run. Public language/SDK ABI, SDK sources, DB,
CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-operators-20261009.json`.

### Bounded call argument matching (2026-10-09)

Fairy C's module_unit_argument_collect/types now adds immutable invocation and argument records across generic, cleanup, and ordinary contexts, preserving the provider module/declaration, specialization instance, ordered original expression IDs, and spans. Declared and explicit generic calls validate arity (E1010) and known argument types (E1006), along with contextual scalar literals/operators and range (E1013); positional labels are only allowed for same-module private non-generic ordinary functions per ADR0396, requiring the exact name at the same position or else E1014. Converting ref mut T to ref T records a shared reborrow, while ownership/place validation remains pending, and unknown field, cast, borrow, builtin, and variant types stay unresolved. Existing arena, work, depth, and round limits are unchanged, and code10 splits only oversized units.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2338 MIR functions; passed |
| Selected development native tests | 9/9 passed |
| Reference acceptance/code/span | 10 matching cases; one recorded callable-value difference |
| Fixed dispatcher input snapshot | 160 modules; 1048562 bytes |
| Units / attempts / splits | 26 / 39 / 13 |
| Invocation / checked invocation totals | 12590 / 6167 |
| Argument / checked argument totals | 29822 / 20527 |
| Shared reborrow argument records | 402 |
| Maximum calls / arguments | 919 / 2687; each bounded to 16384 |
| Maximum value work / graph | 737134 / 1048576; 427 / 4096 |


The observation reuses the fixed 160-module, 1MiB input snapshot; new analysis
modules are compiled as implementation code. It does not establish full
current-source dispatcher compilation. Arguments whose field/cast/borrow/variant
or builtin types are unresolved stay pending, even when the call result is known.
The reference compiler accepts the callable-name argument fixture; the candidate
retains its existing non-value callable classification and reports E1006. That
case is recorded as a difference, not counted among the matching fixtures.
Move-aware cleanup, ownership/place and unsafe/inline validation, structural
cross-unit deduplication/layouts and unit HIR/MIR/C/CLI remain pending.
Release, Linux, full dispatcher native, stage1/stage2 and push were not run.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arguments-20261009.json`.

### Canonical scalar cast targets (2026-10-09)

module_unit_value_model.Expression now keeps the canonical cast target separate from the contextual expected/result type. module_unit_value_collect seeds these targets from owned generic/cleanup annotations or the original ordinary source type offsets, while retaining provider original expression/type IDs and spans and leaving stale input checks unchanged. module_unit_value_scalar applies the existing scalar_casts::allowed rules, including widening to i64 and the exact i32 / c::int bridge. Operands are checked without target context, so oversized unsuffixed literals still fail E1013. Unsupported cast pairs keep the experimental code2 and the full cast span rather than an invented E1052. Known casts propagate to inferred locals, scalar peer operators, call arguments, and returns. Unknown operands and raw-pointer conversions remain pending until the field/place/unsafe work. The existing graph/arena/depth64/round64/work1MiB bounds are unchanged. Full dispatcher observation is deferred to the next field/place unit; this does not claim a new corpus pass, stage1/stage2, or Release/Linux/push.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2340 MIR functions; passed |
| Selected development Fairy native tests | 8/8 passed |
| Reference acceptance/code/span | 7/7 matched |
| Unsupported conversion | Experimental code 2 and full cast span; no invented E1052 |
| Retained bounds | 4096 types; 16384 expressions; depth/rounds 64; work 1048576 |


Full dispatcher observation is bundled with the next field/place unit; this unit
claims no new corpus pass. Field/place, raw-pointer casts and unsafe provenance,
move-aware cleanup, structural cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. Release, Linux, full dispatcher native, stage1/stage2 and push were not
run. Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-casts-20261009.json`.
