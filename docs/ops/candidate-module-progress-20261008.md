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
