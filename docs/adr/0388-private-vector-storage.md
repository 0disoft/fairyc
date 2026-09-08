# ADR 0388: Private element-sized vector storage

- Status: Accepted compiler-private storage prerequisite; no public Vec admission
- Date: 2026-09-08
- Depends on: ADR 0387

## Decision

Add a private, sequential storage transport for the element-sized allocation
required by issue 44. `internal/cstdlib/stage0_vector_storage.c` reuses the
SystemHeap allocator, not Buffer's byte-length contract. The descriptor is an
owned block followed by initialized element count, element byte size, and element
alignment. Byte capacity and element capacity are distinct: the former must be an
exact multiple of element size. Element size must be nonzero and a multiple of a
supported power-of-two alignment. Zero-sized and over-aligned elements are not
admitted by this transport.

The caller exclusively owns a live descriptor and all element lifetimes. It may
move the descriptor but must not create another live owner by copying it. During
reserve there may be no outstanding element borrows, pointers or callbacks that
can observe relocation. Elements must permit representation relocation; this
storage layer does not establish that property for an arbitrary Fairy type.

`fy_private_vector_initialize_v0` requires a canonical empty descriptor with null
data and zero metadata, records size/alignment without allocating, and rejects
reinitialization rather than losing an existing owner.

`fy_private_vector_reserve_v0` receives a minimum **element** capacity. A request
within capacity has no effect and makes no allocation. A request greater than
`SIZE_MAX / element_size` fails before multiplication or allocation. Growth starts
at up to eight elements, doubles within that bound, and switches to the exact
minimum before doubling could overflow. It allocates replacement storage, copies
only `length * element_size` initialized bytes, releases the old block, and then
publishes the new descriptor. Allocation failure leaves the original descriptor,
payload, length and allocation ownership unchanged. Spare capacity is not copied
or treated as initialized elements.

`fy_private_vector_release_empty_v0` refuses nonzero length: the caller must first
destroy or transfer every element. Successful release frees the block and returns
the descriptor to its canonical empty state. It is not an element destructor,
public drop implementation, or idempotent release of an already released owner.
Status values reuse SystemHeap's existing internal status vocabulary.

All exported transport arguments are `void *` handles and scalar sizes/statuses.
Descriptor access uses `memcpy` into/out of a local C representation, not an
incompatible C struct lvalue cast over generated-Fairy storage. Raw pointers must
still reference enough live descriptor storage; numeric validation cannot make an
arbitrary address safe. There is no thread-safety, reentrancy or public FFI promise.

## Target and exposure

`WindowsMSVCVectorStorageSource` composes the existing Windows heap adapter with
the new implementation. Product symbol/component selectors do not include it.
Generated-Fairy tests call the existing allowed `fairyc_fixture_` namespace through
thin C adapters. The private runtime names are not added to the compiler's public
foreign-symbol allowlist. Language 14, SDK sources/version, runtime ABI, stable CLI,
cache formats and target maturity remain unchanged.

## Evidence

`TestPrivateVectorStorageGeneratedNative`, selected by configured
`fairyc_go_test_vector_storage`, compiles an aggregate-only `Storage<i32>` source
through `CheckExplicitFiles` and executes it under low and optimized Windows amd64
MSVC. The source initializes storage, reserves capacity, invokes the C payload
fixture, grows from eight to sixteen elements, reads descriptor fields, moves the
resource and exits its scope. An independent C-side counter requires one successful
cleanup and zero live allocations after the source function returns.

The C fixture initializes one i32 payload, poisons spare old capacity differently
from fresh allocation bytes, and proves that only the initialized four-byte prefix
survives growth. Deterministic OOM, overflowing minimum capacity and premature
nonempty release preserve the exact descriptor bytes and live allocation count.
Additional C-side checks cover invalid size/alignment, duplicate initialization,
allocation-free reserve-zero, first-allocation failure and empty release/reset.
Static assertions and generated layout checks cover the Windows descriptor layout.

The initial fixture failed because it used unadmitted private link names; a second
attempt exposed missing MSVC heap adaptation. Both fixture-integration failures
were corrected without changing compiler symbol policy or SystemHeap behavior.
The final configured run passed both native profiles. No sanitizer, non-Windows
execution or independent compiler lineage is claimed.

## Remaining container work

### Implemented copy-element transitions

`fy_private_vector_push_copy_v0`, `fy_private_vector_read_copy_v0` and
`fy_private_vector_pop_copy_v0` now own the copy-element initialized-prefix
transitions. Each checks the supplied element byte size against the descriptor.
Push checks count overflow and reserves before copying the complete input and
incrementing length. OOM leaves both input and descriptor unchanged. Read accepts
only an index below length. Pop copies the last initialized element into the
output before decrementing length. Bad size, out-of-range read and empty pop
return invalid-argument status without changing descriptor or output. Physical
bytes beyond the new prefix are not zeroed but cannot be read by these APIs.

These raw operations require copyable, representation-relocatable elements. Input
and output spans must be live and disjoint from the descriptor and backing
allocation, with enough bytes for the matching element size. They cannot copy
resource ownership or keep an element borrow alive across mutation. The caller
still establishes the correspondence between a Fairy type and descriptor layout.

`TestPrivateVectorCopyScalars` supplies private Fairy `push<T>`, `read<T>` and
`pop<T>` wrappers using scalar inputs and borrowed outputs. Separate Windows
amd64 sources exercise `bool`, `u8`, `i32`, `u32` and `usize`, including nonzero,
negative and wide unsigned payloads. Each now uses the compiler's C storage
layout query described below; this is not a public type-layout intrinsic or a
safe arbitrary-type constructor.
Low and optimized native runs cover first-allocation OOM, eight-to-nine-element
growth OOM and retry, exact values, LIFO pop, wrong-size rejection, unchanged
outputs, rejection of a just-popped index, and one cleanup with zero live
allocations after draining and moving the resource. A mismatched scalar argument
is rejected before emission. The original poison-prefix reserve fixture remains
in the same configured lane. Foreign heap calls have no interpreter oracle here.

### Copy-element clear and nonempty cleanup

`fy_private_vector_clear_copy_v0` validates the descriptor and exact element size,
then resets only the initialized count. Allocation, capacity and layout metadata
are retained. It does not erase bytes, invoke destructors, allocate or free. Null,
invalid descriptor and wrong-size requests fail without changing the descriptor.
The unsafe caller must prove that discarded elements have no destruction
obligations and no element borrow survives; matching byte size is not that proof.
This operation must not discard resource elements.

The five scalar programs now clear empty and populated storage, clear repeatedly,
reject reads/pops of removed elements with unchanged outputs, and reuse retained
capacity. Their cleanup clears before calling the unchanged empty-only release.
Each leaves one live copy element before moving the vector and exiting scope;
low/optimized MSVC requires one successful release and zero live allocations.
The C probe checks unchanged allocation metadata and live count across clear,
and wrong-size/null rejection. Resource cleanup remains the explicit pop/drop
loop and keeps its independent nine-element destruction-order oracle.

### Owner-supplied empty-state resource transfer

`fy_private_vector_push_take_v0` adds a distinct ownership-transfer protocol over
the same checked byte-growth core. The declaring owner supplies a valid,
non-owning empty representation in a separate live span. On success the input is
copied into the new initialized slot and reset to that empty representation before
the call returns. On failure the donor and prefix remain unchanged.
`fy_private_vector_pop_take_v0` transfers the tail into an already empty output
and removes that slot from the initialized prefix. Neither operation invokes an
element destructor or admits resource use through the copy/read-copy APIs.

This is the existing Buffer/String owner-reset pattern, not permission to `burn`
droppable values. The raw caller proves exact representation size, relocation
permission, empty-state validity, disjoint spans and exclusive ownership. An
arbitrary type without a valid owner-supplied empty state is not covered. The byte
core is shared with copy operations, but their ownership contracts remain distinct.

The original `TestPrivateVectorResourceOwnership` executed a concrete owner-module `Item` and
`Items` implementation under both MSVC profiles. Each Item owns a real native
allocation. A consuming push returns `Result<u32, Item>`: failed first allocation
and failed eight-to-nine growth return the original Item, whose exact identity is
checked before retry. Successful donors become empty and can follow ordinary
cleanup. Explicit pop returns Item 9; after moving the vector, its Fairy cleanup
loop pops and drops Items 8 through 1 before releasing storage. The independent
C registry rejects duplicate/unknown releases and requires nine releases in that
order, no live Item allocation, one vector cleanup and no live vector allocation.
`TestPrivateVectorResourceDonorCannotBeReused` rejects a source borrow after the
consuming push path. Scalar and poison-prefix regressions run in the same lane.

That version used a concrete pointer-sized Item with a valid empty constructor and
compiler-queried Windows C storage layout. It does not instantiate a generic type with a nominal
resource argument or establish cleanup for every possible resource representation.
The later consuming-write integration removes that empty-constructor dependency.
No `burn` authority, public SDK symbol or runtime ABI changes.

### Compiler-private C storage layout acquisition

`cbackend.StorageLayout` validates the target profile and MIR, then reuses the
emitter's aggregate/resource layout preparation and canonical scalar mapping.
It returns the requested type, target, representation type, byte size and alignment.
The current query covers the five emitted Fairy scalars and admitted inert
structs/enums/resources. It rejects unsupported types, pointer/borrow requests,
tagged values, arrays, unverified MIR and unknown profiles. Its result conveys
no copyability, relocation, empty-state or destruction permission.

This is the backend representation, not a source-language `sizeof` promise.
In particular, this backend stores `bool` as `int32_t`, with size/alignment 4/4
on both current profiles. The original bool vector test incorrectly supplied
1/1; its 0/1 values did not reveal the partial-representation copy. The test now
initializes storage from the queried representation and statically compares the
computed constants against native `sizeof` and `_Alignof`. Scalar and Item
fixtures obtain their initialization sizes through test-only C adapters; Item
push/pop/cleanup reuse the initialized descriptor's element size.

The five scalar copy/OOM cases and Item move/failure/cleanup case execute with
these values in low/optimized MSVC. Backend tests separately check padded
struct, resource and enum layouts for both target profiles, preserved query
identity and zero-result rejection. The original i32 poison-prefix fixture
retains its explicit four-byte constants as an independent fixed-layout oracle.
No public foreign symbols, source grammar, stable entry points or SDK versions
change. The later sections record source-level acquisition and constructor
integration; a public SDK container is not admitted.

### Symbolic private storage-query lowering

Typed HIR can now carry an internal `storage-layout` expression with a resolved
target type and `size` or `alignment` selector. MIR retains that type in
`OwnerType`; it does not evaluate a value, borrow storage, allocate, consume an
owner or choose a numeric layout during lowering. The result is `usize`.

`mir.VerifyStorageQueries` admits this one additional operation after checking
its complete shape and known type, including occurrences in unreachable blocks.
Ordinary `mir.Verify` rejects it in every block. The default interpreter and MIR
artifact encode/decode therefore stay closed, without a new cache schema or an
implicit host-target choice. C emission explicitly uses the private verifier and
the same prepared storage layouts as `StorageLayout`, emitting a constant only
after selecting a supported target profile. It records `private-storage-layout`
in the unit's internal feature ledger.

`TestPrivateStorageQueryHIRAndConsumerBoundaries` inserts six queries into typed
HIR for bool, a padded struct and a resource, then checks symbolic MIR and default
consumer rejection. Mutation cases reject wrong result/destination/type/selector,
unexpected operands, ownership flags and traps. An unreachable-block control
separates private admission from normal dataflow reachability. Low/optimized MSVC
executes the six lowered queries and returns 42; generated aggregate assertions
check the corresponding C layouts. This is direct HIR-to-native evidence, not
parsing or semantic admission of a source-level intrinsic. The following section
records the subsequently implemented private generic source connection.

### Private generic source layout queries

`compiler.CheckStorageGenericFunctions` connects the import-free private nominal
generic lane to `storage_size<T>()` and `storage_alignment<T>()`. These are
private source operations, not stable language builtins. Existing parser syntax
is reused; ordinary generic, package, cache and CLI entry points do not enable
them. The two names are reserved only within this opt-in lane.

Expansion substitutes a direct scalar or module-local nominal type into a
detached private AST marker. Semantic analysis resolves the type and produces the
symbolic HIR query; the private MIR verifier admits it without choosing a target.
No value is constructed, evaluated, borrowed, copied or consumed. C emission is
still the first numeric layout selection. Uninvoked specializations fail parsing;
value arguments, wrong arity, unknown or unsupported types and shadowing fail
before verified MIR publication.

`TestPrivateStorageQuerySource` parses generic size/alignment helpers and calls
them for bool, a padded struct and an unconstructed resource. Six symbolic queries
retain source locations, the original AST is unchanged, and ordinary compiler,
nominal-generic, interpreter and artifact paths reject the private operations.
The complete source program returns 42 under low/optimized Windows MSVC.
These tests reuse the existing native query harness, not injected HIR or numeric
test adapters. SDK, public language and runtime ABI versions are unchanged.

### Type-driven construction and lifecycle integration

The five scalar copy programs now initialize inside `empty<T>()`, using the
source queries directly, then return the initialized resource by ownership move.
The Item-owning program does the same inside `empty_items()` with the nominal
Item type. Neither constructor receives a size/alignment argument or calls a
test-only C layout provider. The previously described numeric adapter setup has
been removed from these two integration paths.

Native C probes independently compare the returned descriptor with `sizeof` and
`_Alignof` of each element's C representation. They also require null data, zero
allocated bytes, zero allocation alignment, zero length and no live allocation
immediately after construction. They read through `memcpy`, not an incompatible
struct cast. Existing full scalar push/read/pop, initial/growth OOM preservation,
resource failure-return/retry, move and exact cleanup oracles run afterward under
low/optimized MSVC. Wrong scalar arguments and reuse of a moved Item donor are
checked through the same private source admission lane as their positive controls.

This closes type-driven initialization in the native integration programs, not
public SDK construction. The scalar generic helpers still require a copyable
element contract, and Item cleanup remains specific to that known empty-state
resource. Private descriptor fields are not an encapsulated public API.

### Private scalar package construction boundary

`compiler.CheckStorageGenericPackages` carries the same symbolic operations
through an already-resolved package graph, with exact coordinate/import bindings
and ordinary package visibility analysis. This path is separate from ordinary
generic packages, stable interfaces, cached compilation and CLI admission.
It admits scalar generic arguments only; module-local nominal admission does not
establish cross-package nominal binding.

The storage package integration fixture exports a generic resource and typed
factory/reserve/capacity functions while keeping descriptor fields and raw C
helpers private. An importing client constructs `Storage<bool>`, reserves nine
elements, observes capacity sixteen, moves the owner and leaves scope. Both MSVC
profiles return 42 after the runtime probe observes one release and no live
allocation. The client supplies no descriptor fields, raw pointer or layout
number. No element insertion or arbitrary element destruction is claimed here.

Negative clients cannot read a descriptor field, call its private raw release,
reuse the moved donor, or pass `Storage<bool>` to `reserve<i32>`. The positive
graph preserves its input AST and specialization plan under file reversal and
local package-ID renumbering; ordinary generic admission still rejects the
queries. This is compiler-private module encapsulation evidence, not SDK
publication or admission of a general `Vec<T>` element contract.

This integration exposed a missing resource-field owner check in semantic
analysis. The existing language-14 surface contract already makes such fields
private outside their defining module. Field projection now applies the same
package/module ownership check as resource record construction, with `E0103`
and the owner declaration span. Owner-local reads and public inert struct fields
remain admitted. The package aggregate fixture now uses an owner-defined
consuming accessor instead of reading a private field from its client.

A borrowed accessor for a package-specialized resource exposed an interpreter
unary-borrow type compatibility failure. Runtime resource, struct and enum
constructors retained the definition but omitted the value's semantic type tag.
They now retain that exact type; borrow formation still requires an exact owner
type and result-borrow match. Shared/mutable borrow checks retain the original
state object and leave the owner's type unchanged, and reject a same-named type
from another package.

The package aggregate fixture again uses the borrowed accessor, with interpreter
value 42 and one cleanup plus low/optimized native execution. This fixes nominal
runtime type retention; it does not admit target-dependent layout queries to the
interpreter or create a general mutable-place runtime model.

### Private representation-to-value ownership transfer

Typed HIR and MIR now have a private `storage-take` operation. It reads exactly
one backend representation through a materialized mutable void pointer and
produces a typed value without constructing a placeholder or empty value.
For resource results the ordinary binding, return, move and cleanup machinery
owns the resulting obligation. The source storage is neither cleared nor freed.
The unsafe caller must exclusively own one initialized, representation-relocatable
element, provide enough live disjoint storage, and retire the old element lifetime
without reading or dropping it again. An unsafe span or byte size does not prove
these obligations; there is no runtime pointer validation or safe arbitrary read.

`mir.VerifyStorageTransfers` admits this operation separately from ordinary and
layout-query-only verification, checks its shape and exact unsafe provenance in
every block, and checks live operands on reachable paths. The explicit
`cbackend.EmitStorageTransfers` additionally checks the actual pointer local type
and supported nonzero element representation. It emits `memcpy` into a typed
temporary, avoiding incompatible C lvalue access. Ordinary C emission,
interpreter and MIR artifact encode/decode stay closed. No new cache or stable
source operation is admitted.

`TestPrivateVectorStorageTakeMSVC` replaces one explicitly unsafe fixture HIR
expression with the private operation. A C-owned slot transfers one real allocation
into an Item whose destructor rejects null and duplicate handles. Both MSVC
profiles return 42 after the value crosses a consuming function, with one slot
export, one destruction and no live allocation. Boundary/mutation tests reject
missing provenance, malformed unreachable operations and scalar pointer operands.
This proves the HIR-to-native transfer prerequisite. The following section records
its later generic source and vector-pop integration.

### Private source take and vector-slot pop

`compiler.CheckStorageTransferFunctions` and
`genericsource.PrepareStorageTransferModule` enable `storage_take<T>(pointer)` in
an import-free private source module. The ordinary/query-only entry points do not
enable this name. Expansion substitutes the concrete generic type into a detached
AST marker; semantic analysis requires one mutable void pointer, an admitted
concrete type and an explicit unsafe context (`E5101` when absent). Source type
identity and unsafe provenance survive lowering into the preceding MIR operation.
The parser and stable language/SDK contracts are unchanged.

`fy_private_vector_pop_slot_v0` checks descriptor, exact element size and nonempty
prefix before decrementing length and returning the former tail address. Null,
invalid size/descriptor and empty-prefix requests return null without mutation.
Success transfers the tail's pending element obligation, not a safe borrow: the
caller must immediately materialize the typed value before mutating/releasing
storage or exiting. The helper does not allocate, copy, clear or destroy elements.

The first source-take integration used one source `take_slot<Item>` generic
specialization in both explicit pop and its cleanup loop, with no injected HIR
or empty Item construction on these paths. It retains first-allocation and growth
OOM ownership-return/retry and exact 9-through-1 destruction evidence under both
MSVC profiles. Native counters require nine slot transfers and eleven empty Item
constructions, all from the eleven push attempts; no vector-pop placeholder is
constructed in that version. Empty/wrong-size pops preserve descriptor bytes. Source tests cover
AST preservation, query-only rejection, unsafe/type/arity/name guards and moved
donor rejection. The consuming-write integration below removes push's remaining
empty-state dependency; generic Items cleanup and package/SDK integration remain
required.

### Private consuming source write

The same explicit transfer lane now admits `storage_put<T>(pointer, value)`.
It requires an exact mutable void pointer, an admitted concrete value type and
unsafe context. Resource Flow consumes the value operand; borrow analysis checks
that no live loan crosses that move. Scalars remain copy values at source level.
Lowering evaluates and materializes the destination before the value, transfers
any named resource obligation, and emits a void `storage-put` operation. MIR
verification consumes its materialized value operand so it cannot be reused.
The backend checks actual pointer/value local types and supported nonzero layout,
then uses `memcpy` to write the value without invoking its destructor.

The caller must own sufficient uninitialized, disjoint writable storage and
representation-relocation permission, and establish exactly one initialized
element obligation after the write. The primitive does not allocate, check raw
addresses, destroy an overwritten owner or return a recoverable failure. It
must follow successful capacity preparation, not consume a donor before an
allocation attempt. Standard/query-only source and MIR, ordinary C emission,
interpreter and artifact consumers remain closed.

The generated-Fairy source fixture moves a real allocated Item through a generic
writer into a C-owned slot. Its original donor and writer parameter do not run
cleanup; the receiving slot consumes and releases the allocation exactly once,
with no null-placeholder construction or live allocation afterward. Both MSVC
profiles return 42. Source tests reject wrong types/arity/reserved names, donor
reuse, missing unsafe (`E5101`) and a live loan crossing the move (`E2105`). MIR
mutation tests reject malformed writes and post-write temporary reuse; backend
tests separately reject pointer/value type substitution. AST preservation and
default-consumer rejection are also checked. The following section records this
primitive's reserve/write/prefix-commit integration in the vector fixture.

### Vector push without an empty element representation

`fy_private_vector_prepare_push_v0` validates size/count and reserves one tail
slot before publishing its address into a disjoint pointer-sized output. It does
not change the initialized count. Invalid input, overflow or OOM preserves both
descriptor and output. After successful preparation the caller consumes the value
with `storage_put<T>`, then calls `fy_private_vector_commit_push_v0` with that
exact tail address and element size. Commit checks available capacity/address,
increments length and leaves payload and allocation metadata alone. Invalid or
duplicate commit leaves the descriptor unchanged.

This is an unsafe sequential protocol, not a persistent reservation token or a
transaction across callbacks. No storage mutation, escaping slot, alias access or
callback may intervene; the caller proves the write initialized one owned element
before commit. A matching pointer alone cannot prove initialization. No failing
allocation occurs after the donor is consumed. A prepared but unwritten slot is
outside the initialized prefix; ordinary release rules still apply to the prefix.

The Item vector fixture now has no empty Item constructor, reset representation
or null-tolerant destructor. Its source push returns the original Item on failed
prepare and consumes it only on success. The native probe verifies nine successful
prepares, two OOM failures with exact descriptor/output preservation, nine typed
writes checked before commit, nine commits and nine tail transfers. Wrong-size,
null and duplicate commit controls preserve descriptor bytes. Explicit pop plus
scope cleanup releases Items 9 through 1 exactly once, with one vector release
and no live element/vector allocation under low/optimized MSVC. Source donor-reuse
and the scalar/poison-prefix regressions remain in the same lane. Container cleanup
was concrete Items code at that integration step. The generic fixture below
removes this limitation; package/SDK admission is not established.

### Generic container ownership fixture

The source fixture now declares `Items<T>` with typed cleanup, `empty_items<T>`,
`push<T>`, and `pop<T>`. Construction uses `storage_size<T>` and
`storage_alignment<T>`; cleanup takes each initialized tail as `T` before
releasing empty backing storage. Push returns the unchanged `T` on preparation
failure and consumes it only after successful allocation. No empty element
representation is required.

The same declarations execute with both owning `Item` and copy `i32` arguments.
Both low/optimized MSVC lanes verify nine writes and reverse-order tail transfers,
two OOM failures, one backing-storage release, and zero live allocations. The
resource lane additionally checks nine exact resource destructions and source
donor-reuse rejection. The scalar lane observes written/popped values independently
in C and checks the compiler-selected four-byte layout. This first fixture uses
import-free private source preparation, not stable generic package or public SDK admission.

### Private resolved-package ownership transfer

`CheckStorageTransferPackages` now combines exact nominal package bindings with
the transfer-enabled expander and MIR verifier. Existing ordinary and query-only
package entry points still reject the transfer operations; stable workspace,
artifact and SDK consumers are unchanged.

The container declarations above also execute in a separate `storage` package,
with public construction, push, pop and length functions and private descriptor
fields and raw helpers. The client owns `Item` and creates it through its own
factory. The container does not import the client or invent a client import;
exact private nominal bindings preserve that argument's owner through expansion,
verification, C emission and cleanup. Low/optimized MSVC execution retains both
OOM recoveries, nine typed writes and tail transfers, exact reverse-order element
destruction, one vector release and zero live allocations. Tests reject private
field/helper access, moved-container and popped-donor reuse and wrong element
arguments. Input ASTs remain unchanged, and reversing package input order while
renumbering local package IDs leaves the specialization plan unchanged.

This is a closed resolved-graph compiler entry point, not stable package-cache
identity, installed SDK source, or public `Vec<T>` admission.

### Transfer analysis source identities

`AnalyzeStorageTransferPackages` publishes ordinary source interfaces, generic
template-closure interfaces and per-package specialization IDs through the same
post-Flow publication helper as existing scalar generic analysis. Exact bound
nominal analysis still owns admission. Failed semantic analysis publishes no
interface maps or interface provenance. `CheckStorageTransferPackages` carries
those source facts into its result only after the transfer MIR verifier succeeds.

Interface provenance retains the exact semantic package and coordinate, with
detached getters. Package input reordering and local-ID renumbering preserve
interface hashes and specialization plans. A provider template-body change
invalidates its generic interface; a client nominal-definition change invalidates
the conservative specialization plan. These are distinct identities: the template
interface alone is not a cache key for instantiated nominal code. This step does
not admit storage-transfer MIR to ordinary artifact codecs, stable caches, Atlas
consumers or workspace entry points.

The existing scalar-argument aggregate cache fixtures now read resource payloads
through a provider-owned generic getter, rather than invalid cross-package private
field access. Their cold/warm, persistent-process, invalidation, provenance and
stable-entry rejection assertions remain active.

### Empty-safe pop and discarded result ownership

The private generic `pop<T>` now returns `Option<T>`. Zero initialized length
returns `.none` without calling the raw tail helper. Nonempty storage returns
`.some` with the former tail's sole ownership obligation. Module-local and package
clients match the result explicitly; the scalar fixture also drains the container
and verifies repeated empty observation after the final element.

Discarding a returned `Option<Item>` in a nested block exposed a general MIR
fallthrough-cleanup omission: destruction was delayed until function exit. Value
blocks now materialize their outgoing result before dropping live block-local
resources, preserving moved results and scalar values read from a resource.
Region and loop callers retain their existing cleanup/reset accounting. The
native discarded-Option fixture preserves exact 9-through-1 destruction; a direct
interpreter regression separately checks scalar-tail capture, resource-tail move
and the observable `ABDC` cleanup order. This conforms to the existing lexical
cleanup contract and does not add a new language or SDK surface.

### Exact private transfer MIR envelope

`mircodec.EncodeStorageTransfers` and `DecodeStorageTransfers` use the distinct
`fairyc.mir-storage-transfer-artifact/v0` schema with the existing exact MIR
version. They reuse the ordinary codec's positive byte budget, unknown-field and
trailing-data rejection and canonical-byte check, but verify with
`VerifyStorageTransfers` both before encoding and after decoding. Ordinary
encode/decode remain on their original schema and verifier.

The package-vector round trip preserves the exact MIR, deterministically
re-encodes it, and executes the decoded program under low/optimized MSVC with the
existing ownership/OOM counters. Tests reject ordinary-reader access, schema
substitution, over-budget and noncanonical bytes, and malformed storage writes
without publishing a decoded program. This envelope carries executable MIR only;
it does not authenticate a current source closure, carry generic provenance,
bind a target or admit persistent package-cache reuse.

### Private package artifact extraction and reassembly

`packageartifact.ExtractAllStorageTransfers`, `VerifyStorageTransfers` and
`ReassembleStorageTransfers` reuse the existing package identity, coordinate,
owned-function, explicit-import and stable-context checks with the private MIR
transfer verifier. Ordinary artifact entry points retain their original verifier.
Reassembly requires the complete exact process-local package closure, allocates
fresh internal function symbols, rewrites calls and resource drop links, and
verifies the combined program before returning it. Import body stubs remain
internal to artifact verification.

The vector fixture splits into two independently checked package artifacts and
executes the reassembled program under low/optimized MSVC with the existing
ownership counters. Missing/duplicate packages and inconsistent owner metadata
are rejected, package input order does not change reassembly, and caller-owned
artifacts are not mutated. These entry points do not yet select a transfer-aware
package codec, cache key or hit/fallback coordinator.

### Still required

The private process cache now has explicit `NewStorageTransferProcessCodec` and
`NewStorageTransferCache` constructors. Transfer envelopes use
`fairyc.package-mir-storage-process/v0` and a distinct process-identity prefix;
ordinary codec defaults remain unchanged. Both encode and decode verify the
package artifact. The shared bounded LRU retains copy isolation, eviction and
interface-mismatch quarantine behavior. Tests include two verified vector-package
cache hits followed by reassembly and low/optimized native execution, as well as
cross-codec/process rejection and size/canonicality limits. Fixture keys in this
codec lane are not production source keys. The explicit workspace coordinator
below supplies current-source identity and hit/fallback coordination; persistent
cache and SDK admission remain separate.

### Current-source process cache coordination

`workspacecompiler.NewStorageTransferSession` explicitly selects the private
transfer analyzer, process codec, fallback lowering, artifact extraction and
reassembly. It reuses the generic source/interface/dependency/specialization key
construction with an isolated transfer compatibility identity and the configured
target. Every call analyzes current source and rebuilds provenance before lookup.
Only complete verified hits with exact current package identities and coordinates
may skip lowering; missing, rejected or incompatible artifacts fall back to the
whole current compilation. Invalid current source never looks up retained code.

The existing two-package vector ownership fixture now also loads real manifests
and source digests, stores two cold artifacts and executes two warm hits under
low/optimized MSVC. Provider template-body and client nominal-cleanup edits both
invalidate the two package keys. Updated warm execution retains the original
OOM, typed-transfer, exactly-once cleanup and zero-live-allocation counters.
One-entry retention exercises mixed hit/miss fallback, and a lowering sentinel
separately proves warm checks skip lowering while source changes invoke it.

The initial constructor admitted memory only. The explicit persistent option
described below retains the same source-analysis gate. Stable `SourceSession`
selection still rejects generic aggregate source; no stable CLI, SDK, language or
runtime ABI promotion is made by this private coordinator.

### Private coordinate-stable transfer artifacts

`DescribeStorageTransferStableArtifact`, `EncodeStorageTransferStableArtifact`
and `DecodeStorageTransferStableArtifact` use the separate
`fairyc.stable-package-storage-mir/v0` envelope. They preserve the existing exact
package ownership, projected type context, canonical body graph and explicit
import checks. Ordinary package encoders, decoders and rebind entry points still
reject storage operations, including a private payload relabeled with ordinary
schema and version strings.

`RebindStorageTransferStableArtifacts` maps the complete package closure into
fresh package, function, foreign-symbol and local namespaces, then runs the
transfer MIR verifier before publishing executable code. Source entities for
`storage-take` and `storage-put` now follow the same package-neutral source-path
transport as calls and unsafe scopes. Structural decoding alone is not execution
authorization; damaged ownership provenance is rejected during final rebind.

The vector fixture changes all process-local namespaces, reverses package order,
re-extracts byte-identical stable artifacts, and executes the rebound code under
low/optimized MSVC with its existing ownership/OOM counters. It rejects incomplete
closures, schema substitution, unknown/trailing/noncanonical bytes, byte-limit
violations and damaged unsafe scope. This transport step alone does not select
persistent storage or establish current-source disk-cache compatibility.

### Exact-build persistent transfer cache

`NewStorageTransferStableCache` and its opened-root variant persist the private
stable artifacts in a `storage-transfer-v0` directory using the separate
`fairyc.storage-transfer-stable-cache/v0` envelope and storage-address domains.
They retain the existing positive byte/entry limits, opened-root containment,
immutable publication, retention and generation-matched quarantine checks.
Ordinary and transfer caches cannot read each other's records, including records
copied into the other directory and relabeled at the outer storage layer.

`StorageTransferSession` now accepts explicit persistent configuration, requires
an exact compiler executable digest and rejects mixed memory/disk policy. Its
compatibility key includes the private stable-artifact version, source and generic
interfaces, nominal specialization closure, dependencies and target. Every hit
still follows current source analysis; complete hits rebind into fresh namespaces
and pass the transfer verifier. Missing, rejected or partial entries recompile the
current program. `Close` releases the opened cache root; `PersistentStats` reports
the selected namespace, not an aggregate of unrelated cache directories.

The vector fixture exercises memory and disk cold/warm checks, provider-body and
client nominal-cleanup invalidation, invalid-source suppression, partial retention
fallback and updated warm native execution. A separate process reads two disk
hits with no lowering or stores and executes the ownership/OOM program under
low/optimized MSVC. Changing the compiler digest produces two misses. The shared
cache tests cover both schema families, invalid records, byte bounds and opened-
root escape rejection; a separate sentinel proves disk hits skip lowering.

This remains an explicit compiler-private cache lane. Stable workspace selection,
installed SDK container source and public `Vec<T>` admission remain separate work.

### Existing SDK resource integration

The private package fixture now includes all six unmodified embedded SDK modules
and specializes the candidate container with the SDK-owned `std::bytes::Buffer`.
No SDK module is filtered out, copied into the provider, or given a new owner.
The original source graph remains unchanged after compilation. The native helper
links the emitted stable SDK symbol closure alongside private vector storage,
sharing one instrumented heap implementation rather than duplicating it.

Low/optimized MSVC execute empty pop, nine allocated Buffer elements, two forced
growth failures with ownership-returning retry, explicit pop, container move and
automatic cleanup. Every popped slot retains the exact five-byte payload; all
nine elements and the vector allocation are released. Separate source rejections
cover SDK-private field access, moved-donor reuse and a surviving Buffer view
across donor transfer. This proves an existing SDK resource works through the
private candidate, not installed container source or stable `Vec<T>` admission.

### SDK-owned vector candidate

`sdkstdlib.LoadVectorCandidate` adds `candidates/std/vec.fy` to the complete
stable SDK bundle under a distinct candidate source identity. The candidate is
not installed by `Load`; stable source bytes, version, language revision and
runtime ABI are unchanged. Each load returns fresh source and AST objects.

The candidate owns `std::vec::Vec<T>` and its empty construction, length,
capacity, empty predicate, reserve, push, pop and clear operations. Fields and
raw helpers are private. Push returns the new length or the original element
on failure; pop transfers one owned element, and clear destroys the initialized
prefix in reverse order while retaining capacity. These are candidate contracts,
not a stable public API promise or borrowed element-view admission.

The transfer-only Windows C emitter admits exactly the six required private
runtime symbols with exact result and parameter types. Ordinary output, other
targets, unknown private symbols and incorrect signatures remain rejected.
This avoids replacing runtime calls with test-only symbol aliases.

The full candidate bundle is compiled with a separate client using SDK Buffer
elements. Low/optimized MSVC cover automatic cleanup, explicit clear with
retained capacity, zero reserve, failed and successful growth reserve, two
ownership-returning push retries and container moves. Instrumented allocation
counts require every Buffer and storage allocation to be freed. Source checks
reject private field/helper access and non-owner Vec construction. SDK identity
tests prove that the installed bundle is unaffected and candidate loads are
deterministic and detached. Stable workspace selection, installed source, element
access/borrow invalidation and public admission remain later steps.

### Manifest-loaded candidate workspace

`workspaceloader.LoadVectorCandidate` now applies the normal exact local manifest,
path, namespace and source checks while selecting the candidate SDK explicitly.
The SDK package and graph retain the candidate source identity. The ephemeral
selection survives root/package test overlays, including when only tests import
the SDK. Ordinary `Load` and ordinary test overlays do not admit `std::vec`.

The candidate runs through `StorageTransferSession` with real root/SDK instance
and coordinate identities. Memory and disk tests require two cold stores, two
warm hits without lowering, native ownership execution under low/optimized MSVC,
source-change invalidation, and rejection of invalid current source before cache
activity. Ordinary `SourceSession` still refuses the candidate workspace.

Full SDK disk projection exposed missing source-entity mapping for shared and
mutable byte-view raw-parts construction. Both kinds now retain package-neutral
source paths and rebind to the new owner through the existing checked mapping.
Direct tests cover fresh package numbers and wrong-owner/build-local rejection;
the candidate's disk-hit native execution covers the full SDK body closure.
There is no stable CLI option, manifest opt-in or installed SDK promotion here.

These are private candidate and transport contracts, not stable `Vec<T>` admission. Installed SDK container construction,
stable package/SDK nominal-resource argument admission, safe public operations,
general resource-element cleanup, public ownership-returning failure APIs, borrow
invalidation, package/SDK integration and stable admission
remain separate work. The i32 fixture supplies its known four-byte layout and
seeds/discards one copy scalar in C; it is not a generic typed element API.
The newer copy-element fixtures use the actual push/read/pop transport instead of
C seeding, but neither fixture promotes a safe public container or resource elements.
