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

This is storage, not `Vec<T>`. Typed element layout acquisition, safe push/get/pop,
initialized-prefix transitions, resource-element cleanup, ownership-returning
failure APIs, borrow invalidation, package/SDK integration and stable admission
remain separate work. The i32 fixture supplies its known four-byte layout and
seeds/discards one copy scalar in C; it is not a generic typed element API.
