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
negative and wide unsigned payloads. Each uses its known target size/alignment;
this is not a public type-layout intrinsic or a safe arbitrary-type constructor.
Low and optimized native runs cover first-allocation OOM, eight-to-nine-element
growth OOM and retry, exact values, LIFO pop, wrong-size rejection, unchanged
outputs, rejection of a just-popped index, and one cleanup with zero live
allocations after draining and moving the resource. A mismatched scalar argument
is rejected before emission. The original poison-prefix reserve fixture remains
in the same configured lane. Foreign heap calls have no interpreter oracle here.

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

`TestPrivateVectorResourceOwnership` executes a concrete owner-module `Item` and
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

The fixture uses a concrete pointer-sized Item with a valid empty constructor and
known Windows layout. It does not instantiate a generic type with a nominal
resource argument or establish cleanup for every possible resource representation.
No source ownership rule, `burn` authority, public SDK symbol or runtime ABI changes.

### Still required

This is private storage and transfer evidence, not `Vec<T>`. Typed element layout
acquisition, generic nominal/resource argument admission, safe public operations,
general resource-element cleanup, public ownership-returning failure APIs, borrow
invalidation, package/SDK integration and stable admission
remain separate work. The i32 fixture supplies its known four-byte layout and
seeds/discards one copy scalar in C; it is not a generic typed element API.
The newer copy-element fixtures use the actual push/read/pop transport instead of
C seeding, but neither fixture promotes a safe public container or resource elements.
