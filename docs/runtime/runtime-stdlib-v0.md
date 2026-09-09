# Runtime and Standard Library v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0007-runtime-stdlib-and-allocator-boundary.md`
- Target source: `docs/abi/c-abi-v0.md`

## Three layers

### Compiler-known operations

The compiler and MIR directly own checked, wrapping, saturating, and checked-result arithmetic; division and shift validation; bounds checks and traps; layout queries; bit copy and bit cast; built-in `Option` and `Result` tags and active-payload movement; resource move and drop glue; borrow and region markers; raw memory operations; and slice construction and indexing.

The compiler also owns the current bounded scalar fixed-array value, list-construction, internal
copy transport, exact length observation, checked indexing, and contextual shared `u8[N]` to
`ref u8[]` view construction. They require no runtime allocation or library entry point. The
provisional Stage 0 by-value implementation admits `1..=4096` elements; that ceiling is not a
stable language or ABI limit. Empty/repeated literals, array fields, mutation, general slicing,
properties other than `.len`, other array borrows, and ABI exposure remain outside this slice.
Safe-borrow reassignment is compiler-owned local descriptor transport after Flow proves branch and
loop-edge provenance and loans; it adds no runtime allocation, phi object, or library entry point.
Internal borrow-return summaries are likewise Flow-only proof metadata. They select an existing
call result descriptor and add no runtime table, dispatch, allocation, or library entry point.
Compiler-known `Region.place(T)` returns `Result<ref T, std::alloc::AllocError>`. MIR retains the
placed type and complete tagged result; the `.ok` borrow carries Region provenance and `.err` carries
only allocation status. The interpreter's abstract Region always selects `.ok`. The constrained C
backend obtains the tag from the target Arena allocation status, initializes storage only after
successful allocation, and publishes no borrow on failure.

Heap allocation, files, UTF-8 validation, strings, buffers, formatting, console access, and environment access are not intrinsics.

The interpreter fixture harness may replace explicitly manifest-bound call symbols with bounded
in-memory stdout or stderr byte sinks. This is test configuration, not a compiler-known operation,
runtime symbol, standard-library API, magic source name, or permission to access host streams.

### Thin runtime

The Stage 0 runtime owns only:

- non-returning trap dispatch with bounded source location data;
- process-entry adaptation;
- a canonical empty-storage sentinel whose pointer identity is not a language guarantee;
- a small set of memory helpers needed to keep generated C defined.

The current scalar C transport names one internal non-returning symbol,
`fy_runtime_trap(uint32_t category)`. Category codes 1 through 7 correspond exactly to the canonical
MIR trap identities and do not depend on which traps a translation unit uses. The symbol is an
implementation boundary, not a stable public runtime ABI or C header.

`internal/cruntime/stage0_scalar.c` is the embedded provisional scalar implementation shared by the
experimental build and native differential runner. It maps those codes to caller-bound reserved
process statuses with `_Exit` and performs no cleanup. Receipt-free product builds emit no runtime
output and currently bind base `128`; differential fixtures bind base `192` so their small
ordinary-return oracle stays disjoint. The Windows workspace-run rendering alone inserts one
unbuffered compiler-private trap token before `_Exit`. The CLI consumes and validates that token so
a normal `i32` sharing a reserved OS status is not mistaken for a trap. The token is not program
stderr, a public runtime ABI, a C header, or serialized compatibility data. The complete product
runtime, bounded source-location payload, runtime ABI version suffix, and distribution bundle remain
unimplemented.

`internal/cruntime/stage0_drop_trace.c` is a separate Linux-only conformance observer appended only
for the native differential lane. It records one fixed little-endian `uint32_t` site ID to stderr
for each compiler-owned resource leaf drop and terminates with reserved status 254 if the record
cannot be written. The ID maps to deterministic source/type/span metadata retained by the Go test
runner. It is not linked by `experimental build`, emitted by `experimental emit-c`, exposed to Fairy
C source, or part of a public runtime ABI or output API.

It does not own a general allocator API, collections, files, strings, formatting, unwinding, logging, reflection, garbage collection, scheduling, or async execution. Runtime helpers use a major-suffixed symbol namespace and an exact pre-1 ABI match.

### Standard library

The initial library surface is:

```text
std::alloc  SystemHeap, Arena, AllocError
std::bytes  Buffer with fallible bulk/single-byte append and canonical scalar text append
std::text   String, CStr, CString, allocation-free UTF-8 validation and immutable text-byte views, and zero-copy Buffer-to-String conversion
std::fs     target-neutral File open/read/write/sync/close, absolute and bounded relative/end positioning, and logical size plus Windows MSVC exact I/O, owned-Buffer reads, path mutation, and other metadata
std::io     exact stdout/stderr byte writes, decimal i32 output, and one bounded stdin byte read
std::process effect-free argument count and exact UTF-8 argument copy
std::vec    on-demand Windows move-only Vec<T> with fallible growth and immutable element borrows
std::c      C scalar types and raw wrapper helpers
```

The standard library is supplied as an explicit package instance under
`docs/package/package-graph-v0.md`. ADR 0147 embeds the current Stage 0 source package in the
compiler and injects it only for actual `std` or `std::*` imports. SDK `0.0.0-stage0.59` exports
the accepted public `std::alloc`, `std::bytes`, `std::text`, target-neutral File core, positioning,
and logical size, Windows MSVC `std::fs` extensions, console I/O, scalar output, and effect-free
`std::process` argument slices recorded by ADRs 0205 through 0311 and ADRs 0331 through 0334.
The current Windows filesystem slice includes owned File I/O and metadata, bounded path mutation,
recursive parent creation, and ADR 0311's bounded recursive directory removal with exact completed-
object progress. The remaining standard-library list stays planned and may be backed only by
provisional target C components or examples. The `std::*` prefix creates neither an implicit
prelude nor authority to discover host installations.

Built-in `Option`, `Result`, slices, `str`, and `CStr` get compiler-known view semantics; library modules provide operations over them.

Safe console, file, environment, process, network, clock, and random APIs must declare the
matching source effect from `docs/language/0008-effects-and-capabilities-v0.md`. A target-specific
raw foreign declaration and its safe wrapper carry the same external effect even though only the
raw call also requires an `unsafe` boundary. The implemented effect checker does not itself expose
any host operation.

## Allocation v0

There is no generic allocator interface, borrowed allocator context, or hidden default allocator. `SystemHeap` is an inert zero-size handle whose concrete implementation is selected by the target standard library. `Arena` is a resource and lexical `Region` is the only safe bulk-allocation capability.

Allocation is fallible and reports at least out-of-memory, size overflow, unsupported alignment, and capacity exhaustion. Stage 0 does not trap on OOM by default.

Zero-size requests do not call the platform allocator. They produce length and capacity zero; deallocation is a no-op; empty raw pointer identity is unspecified. Public raw `realloc` is excluded. `Buffer.reserve` checks size overflow, leaves the old allocation unchanged on failure, and cannot run with a live borrow into the buffer.

The first provisional SystemHeap C slice lives in `internal/cstdlib`, not the thin runtime. On
`x86_64-linux-gnu` it admits only nonzero power-of-two alignments at or below
`_Alignof(max_align_t)`. MSVC C does not expose `max_align_t` through `<stddef.h>`, so the
`x86_64-windows-msvc` support resolver supplies the documented 16-byte x64 `malloc` fundamental
alignment ceiling. Over-aligned requests fail closed on both targets. Zero-size requests still
validate alignment but never call `malloc`. Growth and shrink use allocate-copy-release rather than
raw `realloc`, so validation and allocation failures leave the original block unchanged.

This helper is not yet a source-visible `std::alloc` module. Linux product linkage selects it for
exact called symbols; Windows `build-target` uses a private pointer/`usize`/`usize` record to request
eight bytes, observe size/alignment, release, and observe normalized zero state at both MSVC
profiles. The same component is selected once for Buffer and Arena dependency closures. A separate
internal Linux generated Fairy caller
declares the exact `void *`, `size_t`, `size_t` block and observes allocation plus normalized release
through a bounded C test observer. Its independent closed Linux probe still checks empty normalization,
alignment, prefix preservation, failure rollback, malformed-block rejection, shrink, and release
under Clang/GCC at low and optimized profiles. ADR 0214 records the first hosted success. Neither
path deterministically forces host OOM.

## `Buffer`

`Buffer` is a resource owning initialized bytes from `SystemHeap`. It starts with length zero, exposes only `[0, len)`, and may grow only through fallible methods.

The minimum operations are construction with capacity, copy from a byte slice, length and capacity inspection, immutable and mutable initialized-byte views, reserve, append, zero-initialized resize, truncate, and clear. Safe code cannot access spare capacity or raw parts. ADR 0390 additionally admits the SDK-owned Windows `Vec<T>` described below.

The provisional C Buffer slice implements the storage operations behind that surface. Its invariant
is `len <= capacity`; reserve copies only initialized bytes and publishes replacement storage only
after allocation succeeds. Append checks length overflow and
rebases a source inside the initialized prefix across relocation before an overlap-safe `memmove`.
A source into spare capacity or past initialized bytes is rejected. Growing resize zeroes every
newly visible byte. Capacity growth policy, spare bytes, and raw parts remain unobservable.

One internal generated Fairy caller now declares the exact nested SystemHeap-block plus `size_t`
record, calls the real initialize, zeroed-resize, and drop helpers, observes the initialized prefix,
and verifies normalized release through bounded C observers. Windows `build-target` now admits
initialize, zeroed-resize, and drop, selects Buffer plus its SystemHeap dependency exactly once,
observes capacity eight and length four, then observes normalized drop state under both MSVC
profiles. That Windows path does not read the initialized bytes, while the Linux C-only and source
probes retain byte-level zeroing evidence. Those earlier paths remain closed source-to-C
transports. ADR 0246 now connects the complete seven-operation runtime slice to a source-visible move-only
`std::bytes::Buffer`. `with_capacity` and `copy` construct canonical owned state; `reserve`,
`append`, `resize_zeroed`, `truncate`, and `clear` return `Result<usize, BufferError>` and preserve
the prior valid state on failure. `len`, `capacity`, and `is_empty` expose scalar metadata. `view`
returns an owner-bound shared view of exactly the initialized prefix, while `view_mut` returns an
exclusive view of the same prefix without exposing capacity or changing length. Cleanup invokes
drop once and leaves no source-visible error path.

`examples/std_bytes_quickstart` uses only safe source. Its local Windows development and release
builds execute copy, reserve, append, zeroed growth, checked exclusive stores, checked shared reads,
truncate, clear, and implicit cleanup with result zero. Private layout, spare capacity,
uninitialized bytes, allocator selection, generic element storage, and a foreign slice ABI remain
outside the public contract.

The closed C-only native probe configures four Clang/GCC low/optimized executions for construction,
ordinary and self-overlap append, overflow rollback, no-op reserve, zeroed growth, truncate, clear,
malformed-state rejection, and drop. The shared probe-only allocator now additionally forces
nonempty construction, reserve, self-overlap append growth, and zeroed-resize growth failure. Each
growth failure must preserve pointer, capacity, alignment, initialized length, bytes, and one live
allocation; recovery plus final drop must return the live count to zero. The raw caller and
target-specific implementation execute as separate translation units at low and optimized x64 MSVC
profiles. ADR 0256 additionally executes generated-Fairy `with_capacity` and `resize_zeroed`
failure arms at both x64 MSVC profiles; the raw caller retains the byte-level failure matrix.
The four Linux executions cannot run on this Windows host, but ADR 0214 records their
first hosted success. Product linkage contains no probe controls.

## `Arena` and `Region`

`Arena` is created explicitly from `SystemHeap`. Region syntax exposes only:

- `alloc_zeroed_bytes(size)`;
- `copy_bytes(source)`;
- `place(value)` for a fully initialized inert value inferred from the expression.

There is no safe uninitialized byte allocation, generic `alloc<T>` syntax, resource placement, resource-aware reset, or fallible reset.

The bounded Region fixture surface still exports its own opaque `std::alloc::Arena` test resource.
The compiler-bundled SDK separately exposes the move-only runtime-backed Arena accepted by
`region name in expression`, plus `create() -> Result<Arena, AllocError>` and exact status
inspection. It exposes no capacity query, growth policy, ordinary allocation method, node format,
or public representation guarantee.

The provisional C Arena slice now models one active Region per Arena. Begin captures a private mark
and generation; allocation publishes a tracking node only after both byte storage and node storage
succeed; reset frees allocations back to the mark, advances the generation, and invalidates the
Region. A second simultaneous Region, stale post-reset use, or Arena drop during an active Region
fails closed. Zero-size placement validates alignment without allocating. Representation, node
granularity, and growth policy remain private.

One internal generated Fairy caller declares exact target-specific node, Arena, and Region
records, initializes and attaches them through the real helpers, obtains eight initialized zero
bytes through an exact `void **` output slot, then resets the Region and drops the Arena. Bounded C
observers check the allocation, generation transitions, active flags, released node chain, and
invalidated Region. This does not expose those records as the public representation. Product builds
select Arena plus SystemHeap for actual admitted calls only.

Windows target resolution now admits initialization, begin, zeroed allocation, copied placement,
reset, and drop. The checked-in `build-target` example exercises the five lifecycle calls other than
copied placement.
Its checked-in FY program uses private 32/24/32-byte node/Arena/Region records and target-width
`usize` generation fields, which are layout-compatible with the selected Windows x64 C provider but
are not a public ABI. Both MSVC profiles observe generation 1 after initialization, one active Region
and non-null allocation/node state, generation 2 with cleared Region state after reset, and generation
3 after drop. The FY subset cannot safely dereference the returned raw storage, so this Windows path
does not inspect zeroed payload bytes. A separate raw C caller supplies Windows evidence for zeroed
and copied bytes, simultaneous and stale Region rejection, and active-drop preservation. Its shared
probe-only allocator forces payload failure before publication and tracking-node failure after
payload allocation, requiring unchanged Arena/output state and zero live allocations after rollback.
The checked-in safe `examples/regions` workspace now crosses the complete source-level path:
fallible public Arena construction, lexical Region enter, `place(i32)`, `.ok` dereference or `.err`
handling, reset, and automatic Arena drop. Development and release MSVC builds both execute with
result 42 after independently placing and dereferencing an `i32`, a scalar fixed array, and an inert
struct. Generated placement derives size and alignment from each placed C type and constructs
`AllocError` from a nonzero runtime status. A separate Windows host-smoke workspace now compiles the
same source-level operation through verified MIR and constrained C, then a test-only outer C entry
links the existing probe allocator without changing the generated function body. Payload allocation
and tracking-node allocation fail independently at both MSVC profiles; all four generated-Fairy
executions select `.err`, observe status 1, complete lexical reset and automatic Arena drop, and
finish with zero live probe allocations. The product runtime and public SDK still contain no
failure-injection control. The raw C probe retains the broader failure-atomic and byte-level matrix.

Its four-way C-only native probe is configured for exclusive begin, zeroed allocation, copy, allocation
rollback, empty normalization, active-drop rejection, reset, stale-use rejection, and final drop,
but no Linux execution receipt has been observed. The same caller and target-specific implementation
execute as separate translation units at low and optimized x64 MSVC profiles, including both forced
allocation stages, recovery zeroing, and final live-allocation cleanup. Product linkage retains the
ordinary allocator and contains no probe controls.

## Text types

`str` is an unsized, immutable, valid-UTF-8 language type used as `ref str`. UTF-8 validation of `ref u8[]` returns a borrow with the input's origin and performs no copy or allocation. Byte slices never convert implicitly to text.

The provisional text-view C slice performs a complete allocation-free UTF-8 scan. It rejects stray
continuations, overlong forms, surrogate encodings, values above U+10FFFF, truncated sequences, and
obsolete leaders. ASCII NUL remains valid text. Slicing accepts only ordered in-range byte offsets
that land on code-point boundaries and leaves output unchanged on failure. Empty storage identity
is normalized but not observable; compiler borrow origins remain outside the C record.

The four-way native probe for these validation and slicing cases cannot run on this Windows host,
but ADR 0214 records its first hosted Linux execution. Its canonical validator additionally exposes the internal ABI-major symbols
`fy_std_text_utf8_valid_v0` and `fy_std_text_validate_utf8_v0`. One Fairy source fixture calls the
empty-input predicate through raw FFI; another constructs the exact flat `@repr(C)` view record,
passes its verified direct-local raw address, and observes the C helper replace sentinel length 99
with zero. Those raw fixtures are not themselves the public safe `std::text` API. ADR 0248
separately exposes the predicate through an owner-bound borrowed `str` Result over arbitrary input
storage. Neither path implements Unicode normalization, grapheme segmentation, case conversion, or
locale behavior.

`String` is a resource owning a UTF-8 `Buffer`. The public SDK can copy from `ref str`, expose
`ref str`, and append fallibly. The runtime also has a consuming Buffer conversion with
failure-preserved ownership, but ADR 0247 deliberately leaves that operation private until the
source error can return the original Buffer without widening the public contract. Mutable raw bytes
are not exposed.

The provisional String C slice copies from valid text, appends through overlap-safe Buffer growth,
and exposes only immutable text. Buffer conversion validates the initialized prefix and moves the
Buffer only on success; failure leaves both input and destination unchanged so a later source
wrapper can return the original Buffer in its error resource. The internal operation result keeps
high-level text failure separate from the underlying allocation status without freezing source
`Result` representation.

`CStr` is an unsized borrowed NUL-terminated byte sequence with no interior NUL and no UTF-8 promise. `CString` is the owned resource form. Conversion from `ref str` copies bytes, rejects interior NUL, appends one terminator, and performs no locale conversion.

The provisional CStr validator consumes a bounded storage length, requires exactly the final byte
to be NUL, rejects earlier NUL, and reports a view length excluding the terminator. CString creation
checks length overflow, rejects text containing NUL, and publishes its Buffer only after the copied
bytes and one terminator are complete. Empty CString therefore owns one terminator byte while its
CStr view has length zero.

The owned-text four-way Linux native probe is configured for these ownership, rollback, overlap,
UTF-8, NUL, terminator, and empty-value cases but remains locally unobserved. The same raw probe now
links against the target-specific owned-text closure as separate caller and callee translation units
and executes at low and optimized x64 MSVC profiles. It additionally checks opaque storage null and
malformed rejection, failure preservation, self-overlap, deterministic size overflow, normal drop,
and repeated drop. Its probe-only one-shot allocator also forces String/CString initial allocation
failure and String append reallocation failure, requires exact storage/byte preservation, and tracks
live allocations back to zero. Product linkage retains the ordinary target allocator. A
separate generated-Fairy source
probe constructs the exact CStr view transport, obtains three bounded nonempty storage cases from
the companion C unit, and calls the real `fy_std_cstr_validate_v0`. It requires valid storage to
publish a length excluding the terminator, interior NUL to return status 4, missing final NUL to
return status 3, and both failures to leave the prior valid view unchanged. This is raw internal FFI
evidence.

A second generated-Fairy source probe uses the exact nested SystemHeap block, Buffer, text view,
String, CStr view, CString, and two-`u32` operation-result records. It calls the real String copy,
immutable view, append, second view, and drop helpers, then the real CString from-text, CStr view, and
drop helpers. C observers require `Fairy`, `FairyFairy`, the CString terminator immediately after the
logical view, and canonical empty storage after both drops. All operation and allocation statuses
must be zero. The C-only owned-text probe remains responsible for invalid UTF-8, failed Buffer moves,
self-overlap, interior-NUL rollback, empty CString, malformed-state negatives, opaque-storage
failure preservation, and repeated drop. That raw probe does not define wrapper-produced borrow
origins or a source package contract. ADR 0247 promotes the separately proven safe wrapper shape
into compiler-bundled `std::text`, owning source resources, `Result` mapping, and owner-bound
borrowed views. A source `c"..."` literal now has a
verified `Static` origin and compiler-owned C storage. Product build linkage is now automatic only for actual calls to the closed
internal symbols; that raw link transport does not itself define the public safe API.
ADR 0257 additionally executes a generated-Fairy `String.append` growth-failure `.err` arm:
`text::copy("Fairy")` succeeds on the first probe allocation, the second append-growth allocation
fails, and the source arm observes statuses 1/1, `len == 5`, non-empty, and a five-byte borrowed
view before the automatic String drop returns live probe allocations to zero at both MSVC profiles.
ADR 0259 additionally executes the generated-Fairy `CString` construction-failure `.err` arm:
`text::cstring_from_text("Fairy")` fails on the first probe allocation, the source arm observes
statuses 1/1, and the automatic CString drop returns live probe allocations to zero at both MSVC
profiles. ADR 0260 additionally executes the generated-Fairy `text::copy` initial-allocation
failure `.err` arm: the first String storage allocation fails, the source arm observes statuses
1/1, and the automatic String drop returns live probe allocations to zero at both MSVC profiles.
The raw C probe retains the byte-level failure matrix including interior-NUL rejection,
the exact terminator byte, empty and overflow storage, and repeated drop.

Windows `build-target` now admits the exact CStr validator plus String copy/view/append/drop and
CString from-text/view/drop entry points. Its checked-in FY program uses private target-width
pointer/`usize` records and the existing eight-byte two-`u32` result transport. Both MSVC profiles
copy compiler-owned `"Fairy"` storage into a length-five String, expose it, append to length ten,
drop to normalized storage, create a six-byte CString, expose a logical length-five CStr view, drop
it, and validate compiler-owned `c"Fairy"` storage directly. Owned text closes over SystemHeap,
Buffer, and text view, with every component emitted once. FY cannot dereference the returned views,
so exact copied bytes, the terminator byte, interior-NUL/invalid-UTF-8 rollback, self-overlap, empty
CString, allocation failure, and malformed-state negatives remain Linux-probe evidence. This raw
target command does not create safe source resources, source `Result` mapping, or stable record
layouts; ADR 0247 provides the public source wrapper separately.

`examples/owned_text/owned_text.fy` now adds a separate safe source boundary over five opaque
storage bridges: String copy, String append, String drop, CString from text, and CString drop. The
public example resources are move-only and keep their raw pointers private; constructors and append
return source `Result`, callers can read byte lengths and the two typed error status fields, and
safe `string_as_str` plus `cstring_as_cstr` wrappers return owner-bound internal views.
The C bridges move flat storage through `void *` plus `memcpy` rather than dereferencing generated
Fairy resource storage through an incompatible C struct pointer. Constructors require canonical
empty output, append preserves the prior owner on failure, and successful drop stores canonical
empty state back. Both local MSVC profiles execute nonempty and empty construction, append,
interior-NUL status 3, ordinary return cleanup, and implicit cleanup with native result 42. The
wrappers use exact unsafe compiler built-ins that require the shared owner borrow alongside the raw
pointer and extent. The C path constructs no new allocation or copy, normalizes an empty String to
a private non-null `ref str` sentinel, and preserves the existing pointer-only `ref CStr` transport. This
is superseded as the user-facing entry by ADR 0247's SDK module; it remains independent historical
evidence and does not define a public layout.

SDK `0.0.0-stage0.59` retains move-only `String` and `CString` from `std::text`. `copy` and
`cstring_from_text` construct canonical owners fallibly; `append` preserves the existing String on
failure. `string_len`, `string_is_empty`, `cstring_len`, and `cstring_is_empty` expose byte metadata,
with CString length excluding its required terminator. `string_as_str` and `cstring_as_cstr` return
shared views tied to the exact owner borrow. `TextError` preserves separate text and allocation
status fields. ADR 0248 additionally exposes
`validate_utf8(ref u8[]) -> Result<ref str, TextError>`: success borrows the exact input bytes with
no allocation or copy, while invalid input reports text status `2` and allocation status `0`. Raw
storage, capacity, mutable text bytes, implicit conversions, formatting, locale, Unicode
normalization, lossy decoding, and streaming validation remain outside the public surface.

`as_bytes(ref str) -> ref u8[]` reinterprets the same valid UTF-8 storage as immutable bytes in
constant time. It allocates, copies, scans, and calls no runtime symbol. The result retains the
input borrow origin, so a view derived from `string_as_str(ref owner)` keeps that String shared-
borrowed until its last use. Empty text uses the existing private non-null sentinel normalization.
The operation exposes neither capacity nor mutable bytes and performs no lifetime extension.

`from_buffer` converts an owned `std::bytes::Buffer` into a `String` without allocation or copy
when the initialized bytes are valid UTF-8: the same allocation ownership moves into the String.
Invalid UTF-8 returns the original Buffer unchanged as the `Result` err payload, and an empty
Buffer converts to an empty String. The SDK-internal raw storage wrapper rejects null, aliasing,
and nonempty-output calls without touching either caller storage, and commits both storages only
after the existing UTF-8 validation succeeds. `text::copy` remains the copy-required borrowed-text
path. `bytes::empty()` constructs canonical empty storage without allocation. `into_buffer`
transfers a String's exact allocation, capacity, alignment, and initialized length into a mutable
Buffer in constant time without allocation, copy, or redundant UTF-8 validation, then stores
canonical empty state into the consumed String.
Mutation removes the valid-UTF-8 invariant; a later `from_buffer` call validates it again.

`examples/std_text_quickstart` imports only that safe module. Its Windows development and release
builds execute valid nonempty multibyte validation, invalid-byte rejection with zero allocation
status, exact ASCII/multibyte/appended/empty immutable byte views, String copy and append, empty String, nonempty and empty CString, owner-bound views,
interior-NUL rejection, and implicit cleanup with native result 42.

A `c"..."` literal provides allocation-free immutable `ref CStr` storage with one compiler-appended trailing NUL and rejects a decoded interior NUL. The implemented C path emits exact `static const uint8_t[]` bytes, deduplicates equal storage, and permits only an explicit unsafe conversion to immutable `ptr u8` for the direct-literal raw FFI form. It is source syntax, not a runtime constructor.

A `b"..."` literal provides allocation-free immutable `ref u8[]` storage with exact decoded bytes, no UTF-8 promise, no appended terminator, and hidden `Static` origin. MIR retains the byte count as the logical length. The generated-C path uses a private non-null `{ data, length }` value, deduplicated immutable arrays for nonempty values, and a one-byte sentinel with length zero for empty values. The source property `.len` returns that byte count as `usize`, while `.is_empty` returns its exact zero comparison as `bool`; neither scans, allocates, calls, or traps. Matching `usize` values support typed equality, ordering, checked ordinary arithmetic, bitwise operations, and exact-width checked shifts without promotion. Immutable indexing evaluates the receiver and `usize` index once from left to right, returns the exact `u8` byte when in bounds, and traps with `bounds` otherwise. End-exclusive slicing evaluates the receiver and every written `usize` bound once in source order, normalizes an omitted start to zero and an omitted end to the retained receiver length, proves `start <= end && end <= len`, and returns a zero-copy immutable subview with the same borrow origin; invalid bounds trap before pointer derivation. The target-independent `u8 as i32`, `u8 as u32`, and `u8 as usize` conversions may widen an indexed result without a trap or unsafe scope. The checked-in Windows quickstart combines omitted-end slicing, indexing, metadata operations, fixed-array stores, and mutable-view stores. An explicit unsafe conversion may expose a literal or typed shared local's immutable `ptr u8` data address. Separately, a contextual `ref mut u8[]` over a mutable fixed `u8[N]` local aliases the array with the same exclusive origin and exact length, supports checked `u8` reads and stores, and may expose `ptr mut u8`; literals and shared views can never strengthen mutability. Fixed arrays remain value copies across ordinary assignment and calls, while views retain backing identity. Indexed stores check bounds before evaluating the value. Mutable slicing, text slicing, a public slice ABI, and implicit text conversion remain unimplemented.

A `"..."` literal provides allocation-free immutable `ref str` storage after the complete decoded byte sequence passes UTF-8 validation. MIR retains a distinct text operand and byte length rather than reusing byte-string or C-string meaning. The generated-C path uses a private non-null `fy_str { data, length }` value, deduplicated exact immutable UTF-8 arrays for nonempty values, and a one-byte sentinel with length zero for empty values. The source property `.len` returns the UTF-8 byte count as `usize`, and `.is_empty` returns whether that byte count is zero; neither counts scalar values or graphemes. An explicit unsafe conversion may expose a literal or typed local's immutable `ptr u8` data address; mutable extraction, indexing, slicing, a public string ABI, implicit byte-to-text conversion, and implicit C termination remain unimplemented.

## Console byte I/O v0

The compiler-bundled `std::io` module exposes
`write(ref str) -> Result<usize, ConsoleError>` for stdout and
`write_stderr(ref str) -> Result<usize, ConsoleError>` for stderr. It also exposes
`write_bytes(ref u8[]) -> Result<usize, ConsoleError>` and
`write_stderr_bytes(ref u8[]) -> Result<usize, ConsoleError>` for exact binary output. All four
declare `@effects(console)`, append no newline, allocate nothing, and retain no borrow. Text input
writes its exact valid UTF-8 bytes; byte-view input writes its exact bytes without validation,
transcoding, or terminator insertion. Success preserves the exact written byte count. Failure preserves a numeric status
and completed-prefix byte count. The private raw declarations, `@repr(C)` transport, pointer
conversion, and `fy_std_stdout_write_all_v0`/`fy_std_stderr_write_all_v0` symbols remain inside the
SDK and target support units.

`write_i32(i32 value) -> Result<usize, ConsoleError>` uses the same stdout result and
`@effects(console)` contract. It emits canonical locale-independent decimal ASCII with no newline:
zero is `0`, negative values have one leading minus, positive values have no sign, and there are no
leading zeroes. It formats the full `i32` domain into an 11-byte stack buffer and allocates nothing.
The private `fy_std_stdout_write_i32_v0` target support symbol computes negative magnitude in
unsigned arithmetic so `INT32_MIN` is defined, then delegates the exact resulting view to the
existing target-owned write-all loop.

`write_u32(u32 value) -> Result<usize, ConsoleError>` and
`write_usize(usize value) -> Result<usize, ConsoleError>` format canonical locale-independent
decimal ASCII with no sign, no leading zeroes, and no newline: `u32` covers `0` through
`4294967295` in at most 10 bytes, and `usize` covers the full pointer-width unsigned domain (up to
`18446744073709551615` on this LLP64 x64 target) in at most 20 bytes. `write_bool(bool value) ->
Result<usize, ConsoleError>` writes exactly ASCII `true` or `false`. All three declare
`@effects(console)`, allocate nothing, and reuse the same stdout result and completed-prefix
semantics. The private `fy_std_stdout_write_u32_v0` and `fy_std_stdout_write_usize_v0` support
symbols share one back-to-front unsigned-decimal helper per target; `write_bool` reuses the
existing `write`/`fy_std_stdout_write_all_v0` path and adds no raw symbol.

`write_stderr_i32`, `write_stderr_u32`, and `write_stderr_usize` select stderr while retaining the
same canonical stack-only decimal format and `ConsoleError` completed-prefix semantics.
`write_stderr_bool` writes exactly ASCII `true` or `false` through the existing
`write_stderr`/`fy_std_stderr_write_all_v0` path and adds no bool-specific raw symbol. The private
numeric stderr symbols are target-neutral closed standard-library entries implemented for both
current target adapters.

Status 1 is invalid input, 2 is an unavailable selected stream, 3 is an operating-system write
failure, and 4 is impossible zero progress. Empty input succeeds without consulting the selected
stream. The Linux adapter routes both descriptors through one POSIX `write` loop, retries `EINTR`,
and bounds requests by `SSIZE_MAX`. The Windows adapter routes both standard handles through one
`WriteFile` loop in `DWORD`-bounded chunks. Both stop at the first non-retryable failure and return
the completed prefix. The API does not format, append a newline, retry after a reported failure, or
promise terminal-specific encoding behavior. Byte-view calls deliberately make no claim that a
terminal will render arbitrary bytes as readable text.

The same module exposes
`read_stdin(ref mut u8[] buffer) -> Result<usize, ConsoleReadError>`. It declares
`@effects(console)`, allocates nothing, retains no borrow, and performs at most one operating-system
read bounded by `buffer.len`. Success contains the exact initialized prefix length; success with zero
is EOF. A short success is ordinary and callers that need to fill a buffer or consume a stream must
loop. The operation writes only within `buffer[..read]`; the remaining suffix keeps its prior bytes.
It does not validate UTF-8, scan for a delimiter, append a terminator, or retry to fill the buffer.

`ConsoleReadError` has numeric `status` and `read` fields and is intentionally distinct from
output's `ConsoleError`. Raw status 1 is invalid nonempty storage, 2 is unavailable stdin, and 3 is
an operating-system read failure. Empty input succeeds with zero before consulting the stream. The
safe wrapper cannot construct invalid nonempty storage. The private
`fy_std_stdin_read_v0` declaration, mutable pointer conversion, and C transport remain inside the
SDK and target support unit.

The POSIX adapter makes one `read` request bounded by `SSIZE_MAX`, retries `EINTR`, maps `EBADF` to
status 2, and treats a zero result as EOF. The Win32 adapter makes one `ReadFile` request bounded by
`UINT32_MAX`, maps null, invalid, and `ERROR_INVALID_HANDLE` handles to status 2, and treats
`ERROR_BROKEN_PIPE` and `ERROR_HANDLE_EOF` as EOF. Neither adapter buffers, owns the process handle,
or changes terminal mode.

The checked-in `examples/std_io_quickstart` contains no unsafe code or platform symbol. The Windows
smoke runs it through stable `fairyc run --release --locked` with `stdin!\n`, forwards exact
`Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n` to stdout and
`Fairy C warns too\nbytes on stderr\n` to stderr with status zero, builds and directly executes the
retained image with EOF, short `stdin!\n`, and a longer nine-byte payload. The direct inputs produce
respectively no echo, the exact short echo, and only the first eight bytes. A separately built
compiler repeats the locked run with short input plus build and artifact execution. Stable run
attaches input only after successful compilation and image inspection and does not pre-read or retain
it.
Linux Clang/GCC low/optimized executions are configured through the native report v36
`linuxConsoleExecutions` field and exercise stdin, stdout, stderr, and the full-domain `i32` decimal
boundaries, but remain unobserved on this
Windows host.

## Process arguments v0

The compiler-bundled `std::process` module exposes
`arg_count() -> usize` and
`arg(usize index, ref mut u8[] buffer) -> Result<usize, ProcessArgError>`. Both functions are
effect-free and read only the process-startup argument data captured when the program launched;
they do not access the environment, the current directory, console streams, or any mutable host
state. `arg_count` includes the executable name at index 0. `arg` copies one argument's exact UTF-8
bytes into the caller buffer and returns the byte count; an empty argument succeeds with zero and
writes nothing.

`ProcessArgError` has numeric `status` and `needed` fields. Status 1 is index out of range, status 2
is invalid encoding (including an unpaired UTF-16 surrogate, or argv that could not be materialized
as valid UTF-16), and status 3 is buffer too small; status 3's `needed` carries the full UTF-8 byte
length and doubles as the length query, so there is no separate `arg_len` API. Every failure leaves
the caller buffer byte-for-byte unchanged because the C implementation computes the exact byte
length and completes all validation before the second conversion pass writes. The private raw
declarations, `@repr(C)` result transport, and
`fy_std_process_arg_count_v0`/`fy_std_process_arg_v0` symbols remain inside the SDK and the
Windows-only target support unit.

The Windows adapter reads the UCRT wide argv through `__p___argc()` and `__p___wargv()`; the narrow
argv is ANSI-code-page encoded and is never used. Because the `mainCRTStartup` narrow policy under
the unchanged `int main(void)` entry leaves `*__p___wargv()` NULL, the adapter materializes the
wide argv lazily with `_configure_wide_argv(_crt_argv_unexpanded_arguments)` — the same unexpanded
mode the `wmain` startup would use, so no wildcard expansion is introduced. If materialization
fails or the wide argv is still NULL, `arg` fails closed with status 2 and `needed == 0` without
touching the caller buffer; `arg_count` still reports the same startup argument set. UTF-16 to
UTF-8 conversion uses `WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, ...)` in a strict two-pass
form, and an unpaired UTF-16 surrogate reports status 2. No `wmain`, environment side channel,
shell32 dependency, or ANSI fallback exists.

The checked-in `examples/process_args_quickstart` contains no unsafe code or platform symbol. The
Windows smoke builds it at both profiles, executes the retained images directly with a five-argument
set covering empty, flag-looking, space/quote/trailing-backslash, and non-ASCII UTF-8 values, and
runs stable `fairyc run -- <args>` and `fairyc run --release --locked -- <args>`; the program
returns 42 only after verifying argv[0] presence, count, ordering, exact byte round-trips,
out-of-range status 1, and too-small status 3 with exact `needed` and unchanged sentinel bytes. A
host-smoke-only C decode probe at low and optimized profiles proves the narrow-startup NULL wide
argv, successful lazy `_configure_wide_argv` materialization, exact quote, trailing-backslash,
Korean, and empty argument round-trips, and status 2 with an unchanged buffer for an unpaired
surrogate.

### Legacy Windows example symbol

The target-only unsafe symbol `fy_std_windows_stdout_write_all_v0` accepts one immutable byte
pointer and `usize` length and returns a private `@repr(C)` result containing `u32 status` and
`usize written`. Empty input succeeds without consulting the process handle. Nonempty null input
returns status 1 with zero progress. Status 2 means stdout was unavailable, status 3 reports a
`WriteFile` failure, and status 4 reports impossible zero progress. Success is status 0 with
`written == length`.

Its FY raw declaration and the checked safe text wrapper both declare `@effects(console)`. The
wrapper returns `Result<usize, ConsoleError>`: success preserves the exact byte count, while failure
preserves the nonzero raw status and completed-prefix count in an example-local typed record. The
example entry must propagate the effect even though it contains no raw pointer or unsafe block.
This remains executable authority and typed-progress tracking for the historical Windows-only
example. ADR 0205's safe SDK API uses the target-neutral internal symbol; neither result record is a
public tagged ABI.

The implementation obtains `STD_OUTPUT_HANDLE` once and loops in `DWORD`-bounded chunks until all
bytes are written or a failure reports the completed prefix. It does not allocate, append a newline,
transcode, retain the input pointer, retry a failed operation, or hide partial progress. The call is
synchronous: every input-pointer use ends before it returns. The configured Windows smoke verifies
empty and invalid inputs plus exact ASCII bytes through a captured pipe; interactive Unicode console
encoding, terminal modes, stdin, stderr writing through that legacy symbol, colors, formatting, and cancellation remain
unproven or unimplemented.

The legacy raw function is admitted only by the `x86_64-windows-msvc` support resolver and must be called
inside `unsafe`. The checked-in example confines the call to a safe function in an explicitly listed
support module; that function accepts `ref str`, derives the raw length from `.len`, declares
`@effects(console)`, and reports success
only after checking status and completed progress. It remains an example of unsafe containment;
new source should use `std::io::write`.

## Linux `File` v0

The C implementation is explicitly tied to `x86_64-linux-gnu`; the shared source API accepts
`ref CStr`, not a purported portable filesystem-path abstraction.

The minimum operations are fallible open, partial read, partial write, `write_all`, `sync`, and consuming `close`. `write` returns the actual byte count. A `write_all` error reports the number already written.

On the Linux v0 profile, `close(self)` consumes ownership on success and failure and is never retried automatically. A caller requiring durability uses `sync` before `close`. Automatic drop performs best-effort close, returns no error, allocates nothing, and emits no log.

The provisional Linux C helper now implements bounded C-string paths, closed open options, partial
read/write counts, progress-reporting `write_all`, `fsync`, consuming close, and silent best-effort
drop. Close clears ownership before one kernel call, including the failure path. Its native probe
uses three-byte reads and a forced `EBADF` close failure across the configured compiler/profile
matrix.

A separate generated-Fairy source probe crosses exact operation-result, File, and open-option
records into the real C unit. It opens a prepared file for truncating writes, writes all nine
`FairyFile` bytes, synchronizes, consumes close, requires a second close to report the closed status,
reopens for reading, performs three three-byte reads, observes EOF with count zero, closes, and
requires the reconstructed bytes and normalized closed record. The companion C code owns only
bounded inputs and observations; it does not replace the production file operations.

File ownership, source effects, and typed error resources now connect through ADR 0331's seven
target-neutral private symbols. The public create/open family, partial `read`/`write`, `write_all`,
`sync`, consuming `close`, and best-effort drop select this component automatically. Linux validates
a nonempty terminator within 4096 storage bytes, supports exclusive create and kernel append,
creates with mode `0666` subject to umask, and stores descriptor-plus-one in private pointer-width
storage. ADR 0332 adds two common same-owner operations: `position` queries `lseek(SEEK_CUR)` and
`seek_start` uses `lseek(SEEK_SET)` after rejecting offsets above `INT64_MAX`. Both retry `EINTR`,
return the absolute byte position, and never reopen a path. ADR 0333 adds shared-borrow `size`
through `fstat` on the same descriptor, retries `EINTR`, and maps a negative kernel size to
`EOVERFLOW`; every result preserves the cursor and owner. ADR 0334 adds bounded `i32` current- and
end-relative movement through `lseek(SEEK_CUR)` and `lseek(SEEK_END)`, including cursor-only
movement beyond EOF. Ordinary POSIX path lookup remains; containment, no-follow authorization,
path mutation, other metadata, exact reads, Buffer growth, and Unicode normalization stay outside
the Linux public slice. Native report v36 requires four generated product executions returning 42
after observing position and size seven; moving to two, one, five, and ten; preserving logical size
seven beyond EOF; rejecting `usize` maximum without cursor movement; and restoring seven.

## Windows `File` C slice v0

The provisional `x86_64-windows-msvc` component stores one `HANDLE` token plus an explicit open byte
inline and normalizes the closed state to `{ 0, 0 }`. It performs no hidden allocation. Open
accepts a bounded NUL-terminated ASCII byte path and closed read/write/create/truncate options,
converts the path into a stack-owned `WCHAR[4096]`, and calls `CreateFileW`. Empty paths, non-ASCII
bytes, missing termination within the bound, invalid option combinations, and non-closed output
storage fail before ownership changes. This narrow bootstrap boundary is not the future portable or
Unicode-complete path contract.

Read performs one `ReadFile` call and returns its exact partial count, including zero at EOF.
`read_exact` repeatedly uses that same bounded primitive and advances only within the caller's
exclusive mutable view. Complete fill succeeds with the requested length. A successful zero read
before completion returns status 7, OS code 0, and the cumulative initialized prefix; a later OS
failure returns status 1, the exact Win32 code, and the same cumulative prefix. The unread suffix
keeps its prior bytes, and cursor advancement is not rolled back. An empty request on an open File
succeeds with zero before `ReadFile`. The operation allocates nothing and retains no pointer, but on
a pipe or device it may perform additional synchronous reads and block longer than `read`.
`write` exposes one call to the bounded single-write helper, returns the exact successful count,
and leaves retry policy to safe callers. `write_all` loops over the same primitive, preserves the completed prefix on
failure, and reports no progress if a successful write completes zero bytes. Sync calls
`FlushFileBuffers` explicitly.

Positioning uses `SetFilePointerEx` without allocation. `position` observes the absolute byte
offset with `FILE_CURRENT`; `seek_start` moves to one `usize` offset with `FILE_BEGIN` and returns
the resulting offset. Because the admitted target is Windows x86-64, `usize` carries the complete
64-bit component result. Values above `INT64_MAX` reject before an OS call with status 2, OS code
0, completed 0, and an unchanged cursor. ADR 0305 adds `seek_current` and `seek_end` with signed
`i32` deltas and the same absolute `usize` result. A negative resulting position fails with the
exact Windows error and leaves the cursor unchanged. Moving beyond EOF changes only the cursor;
the file length remains unchanged until a later write or `set_len`. Wider signed deltas, sparse-gap
content promises, portable offsets, and Linux public linkage remain outside the slice.

Size observation uses `GetFileSizeEx` through a shared File borrow and does not move the cursor.
It returns the exact byte length as `usize` on the admitted 64-bit target, status 3 for a closed
File, or status 1 plus the exact Win32 code for a kernel failure. Every outcome preserves ownership,
position, and allocation state. Compound metadata records, timestamps, path-based
queries, and external-write snapshot guarantees remain outside the slice.

Read-only metadata observation uses `GetFileInformationByHandleEx(FileBasicInfo)` through the same
shared File handle and maps only `FILE_ATTRIBUTE_READONLY` to a target-neutral `bool`. The private
result is `{ status: u32, os_code: u32, enabled: u32 }`; invalid and closed storage preserve the
existing status mapping and failures return `enabled = 0`. `is_read_only(ref File)` does not move
the cursor, change size, allocate, retain a pointer, reopen a path, or mutate attributes. The value
does not guarantee write success because handle access, ACLs, sharing, filesystem policy, and later
external metadata changes remain independent.

File-kind observation uses `GetFileType` through the same shared File handle and does not move the
cursor or allocate. The private result is `{ status: u32, os_code: u32, kind: u32 }`; the public
`FileKind` enum normalizes target results to exhaustive `.Disk`, `.Character`, `.Pipe`, or `.Unknown`
variants. Compatibility predicates remain available without exposing target constants.
`FILE_TYPE_UNKNOWN` is distinguished from failure by clearing
and then reading the thread last-error value around the call. Closed storage reports status 3;
kernel rejection reports status 1 and the exact Win32 code.

File-identity observation uses `GetFileInformationByHandleEx(FileIdInfo)` through the same shared
File handle and does not reopen a path, move the cursor, allocate, or retain a pointer. Private
caller output contains the 64-bit volume serial and complete 128-bit file identifier as six `u32`
words and changes only after a successful query. Public opaque `FileId` exposes borrowed equality
over every retained bit and no field, ordering, hash, display, integer, or serialization operation.
The value compares bounded observations on one computer; it is not a durable key, authorization
token, cross-machine identity, or promise against filesystem-specific change or reuse over time.
Providers may reject the query, and the current evidence covers disk files rather than pipes or
character devices.

Explicit close first copies the live handle, normalizes the caller-owned state, and then calls
`CloseHandle` exactly once. The state therefore remains consumed even when the kernel rejects the
close, and automatic drop cannot retry it. Repeated explicit close reports the closed status.
Automatic drop similarly normalizes before one best-effort close, discards the error, allocates
nothing, and emits no output.

The I/O raw result record is `{ status: u32, os_code: u32, count: usize }`; positioning returns
`{ status: u32, os_code: u32, position: uint64 }`; size observation and length change return
`{ status: u32, os_code: u32, size: uint64 }`; kind observation returns
`{ status: u32, os_code: u32, kind: uint32 }`; read-only observation returns
`{ status: u32, os_code: u32, enabled: uint32 }`; identity observation writes a private six-`u32`
record only after success and returns the ordinary I/O result. The Windows-specific portion of the
closed target contract
admits only `fy_std_windows_file_open_v0`, `fy_std_windows_file_remove_v0`,
`fy_std_windows_directory_create_v0`, `fy_std_windows_directory_remove_v0`,
`fy_std_windows_directory_rename_v0`, `fy_std_windows_file_rename_v0`,
`fy_std_windows_file_read_v0`,
`fy_std_windows_file_read_exact_v0`, `fy_std_windows_file_write_v0`,
`fy_std_windows_file_write_all_v0`, `fy_std_windows_file_sync_v0`,
`fy_std_windows_file_close_v0`, `fy_std_windows_file_drop_v0`,
`fy_std_windows_file_position_v0`, `fy_std_windows_file_seek_start_v0`,
`fy_std_windows_file_seek_current_v0`, `fy_std_windows_file_seek_end_v0`,
`fy_std_windows_file_size_v0`, `fy_std_windows_file_kind_v0`,
`fy_std_windows_file_read_only_v0`,
`fy_std_windows_file_link_count_v0`,
`fy_std_windows_file_id_v0`, and
`fy_std_windows_file_set_len_v0`. The public resolver separately admits twelve common File symbols:
open, read, position, size, absolute/current/end seek, write, write-all, sync, close, and drop. A separate-translation-
unit probe
executes create/truncate, write-all, cursor-preserving length shrink/extend, sync, normal and repeated close, three-byte reads, EOF, forced
`ERROR_INVALID_HANDLE` close and size failures, absolute/current/end position changes, size 9,
overflow and negative-result rejection with cursor preservation, beyond-EOF movement with unchanged
size, disk and `NUL` character kinds, normal/read-only metadata plus one-to-two hard-link counts
with unchanged cursor and size,
bounded same/distinct disk identities,
closed and invalid-handle kind and identity failures,
drop, and repeated drop at both local MSVC profiles.

The historical example-level `examples/windows_file/windows_file.fy` module connects that raw contract to
an effect-checked move-only FY `File`, typed `FileError`, create/truncate, read, and update opens, write-all,
sync, consuming close, and `read(ref mut u8[])`. A mutable fixed array supplies a contextual exclusive
view with its canonical length and private writable data pointer. Runtime state crosses the private
opaque boundary through bytewise `memcpy` load/store so the C unit does not dereference generated
resource storage through an incompatible struct type. Low and optimized source builds write and
read nine bytes, observe EOF, execute explicit close, and prove implicit drop by reopening a path
whose write handle would otherwise deny sharing. ADR 0264 promotes the same ownership and I/O
contract into SDK `0.0.0-stage0.14` as public `std::fs`; ADR 0269 extends it in SDK
`0.0.0-stage0.15` with `position` and absolute `seek_start`; ADR 0305 later extends it in SDK
`0.0.0-stage0.39` with bounded `seek_current` and `seek_end`; ADR 0306 later extends it in SDK
`0.0.0-stage0.40` with same-handle `is_read_only`; ADR 0307 later extends it in SDK
`0.0.0-stage0.41` with same-handle `hard_link_count`; ADR 0308 later extends it in SDK
`0.0.0-stage0.42` with no-replace `hard_link`; ADR 0309 later extends it in SDK
`0.0.0-stage0.43` with same-handle `allocated_size`; ADR 0310 later extends it in SDK
`0.0.0-stage0.44` with same-handle `set_read_only`; ADR 0311 later extends it in SDK
`0.0.0-stage0.45` with bounded recursive `remove_dir_all`; ADR 0270 extends it in SDK
`0.0.0-stage0.16` with shared `size`; ADR 0271 extends it in SDK `0.0.0-stage0.17` with
existing-file read/write `open_update`; ADR 0272 extends it in SDK `0.0.0-stage0.18` with
cursor-preserving `set_len`; ADR 0274 extends it in SDK `0.0.0-stage0.19` with read/write
`open_or_create`, using `OPEN_ALWAYS` to create an empty missing file or preserve an existing file
at byte zero; ADR 0275 extends it in SDK `0.0.0-stage0.20` with write-only `create_new`, using
`CREATE_NEW` to reject an existing path without changing its bytes; ADR 0276 extends it in SDK
`0.0.0-stage0.21` with write-only `open_append`, using `FILE_APPEND_DATA` and `OPEN_ALWAYS` so each
underlying write begins at the then-current EOF after any seek; ADR 0277 extends it in SDK
`0.0.0-stage0.22` with same-handle target-neutral file-kind observation; ADR 0278 extends it in SDK
`0.0.0-stage0.23` with exact-fill reads and explicit unexpected EOF; ADR 0279 extends it in SDK
`0.0.0-stage0.24` with same-handle 128-bit file identity plus volume comparison; ADR 0281 moves it to
`0.0.0-stage0.25` by promoting `FileKind` to an exhaustive public enum; ADR 0285 extends it to
`0.0.0-stage0.26` with one-call partial `write`; ADR 0288 extends it to `0.0.0-stage0.27` with
one-call `read_append(ref mut File, ref mut bytes::Buffer, usize)` that reserves spare capacity but
commits only successfully initialized bytes. Allocation failure precedes I/O and preserves the
complete Buffer; file failure preserves bytes and length while allowing reusable capacity growth.
ADR 0290 extends it to `0.0.0-stage0.28` with
`read_to_end(ref mut File, ref mut bytes::Buffer)`. The source helper starts at the current cursor,
uses bounded 65,536-byte growth windows, returns only newly appended bytes, and preserves every
completed prefix if a later read or allocation fails. Allocation for each new window precedes its
corresponding read, so allocation failure consumes no additional file byte. It adds no target
symbol or runtime ABI surface.
ADR 0291 extends it to `0.0.0-stage0.29` with `remove_file(ref CStr)`. The one-call Windows target
operation reuses the bounded ASCII path contract, returns exact Win32 failures, treats a missing path
as an error, and never closes or mutates a live File owner. The current non-delete-sharing open
policy makes a live-handle removal fail until the caller explicitly closes that owner.
ADR 0292 extends it to `0.0.0-stage0.30` with
`rename_file(ref CStr, ref CStr)`. The Windows target first admits only an existing non-directory
source and then calls `MoveFileExW` with flags `0`: an existing destination is never replaced and a
cross-volume copy/delete fallback is never requested. It returns exact Win32 failures, performs no
implicit close, and makes no metadata-durability claim. The attribute gate is path-based rather than
handle-pinned, so concurrent external replacement and reparse-point behavior remain target-defined.
ADR 0293 extends it to `0.0.0-stage0.31` with `create_dir(ref CStr)` and
`remove_dir(ref CStr)`. The target operations create only the final path component and remove only
an existing empty directory. They reuse exact path/error transport, do not enumerate or delete
children, do not change the current directory, and make no explicit ACL or durability promise.
ADR 0294 extends it to `0.0.0-stage0.32` with
`rename_dir(ref CStr, ref CStr)`. The Windows target first admits only an existing directory source
and then calls `MoveFileExW` with flags `0`: an existing destination is never replaced and a
cross-volume copy/delete fallback is never requested. A nonempty directory moves with its children;
the adapter does not enumerate, copy, or delete them. It returns exact Win32 failures, performs no
implicit close, and makes no metadata-durability claim. The attribute gate is path-based rather than
handle-pinned, so concurrent external replacement and reparse-point behavior remain target-defined.
ADR 0296 extends it to `0.0.0-stage0.33` with `create_dir_all(ref CStr)`. The target validates the
complete bounded ASCII path before effects, rejects lexical `.`/`..`, repeated interior separators,
drive-relative and device-namespace paths, and then creates each missing ordinary relative,
drive-rooted, current-drive-rooted, or UNC-share descendant prefix. Existing directories succeed;
an existing file preserves `ERROR_ALREADY_EXISTS`. A later target failure returns exactly and keeps
every prefix already created for explicit caller recovery. It does not enumerate or remove children,
roll back partial creation, choose ACLs, or claim metadata durability.
ADR 0297 extends it to `0.0.0-stage0.34` with
`copy_file_bytes(ref CStr, ref CStr)`. The target validates both bounded ASCII paths before effects,
holds one exclusive source handle and one exclusive `CREATE_NEW` destination handle, and copies
through a fixed 65,536-byte stack buffer. Existing destinations are never changed. A later failure
keeps the completed destination prefix and reports its exact byte count; success returns the exact
copied count. It copies no metadata, performs no flush or atomic publication, and retains ordinary
Windows reparse-point resolution.

ADR 0298 extends it to `0.0.0-stage0.35` with exhaustive public `PathKind` variants `.File`,
`.Directory`, and `.Other`, plus `path_kind(ref CStr) -> Result<PathKind, FileError>`. The Windows
target reuses bounded ASCII conversion and one `GetFileAttributesW` observation. Ordinary files and
non-reparse directories map directly; observed device or reparse attributes map conservatively to
`.Other`; failures retain exact status and Windows code with zero completed bytes. The value is a
momentary path-name snapshot, not an opened-object identity, authorization or containment decision,
or race-free precondition for a later operation. Metadata records, reparse-tag details, portable
Unicode and link policy, and Linux public linkage remain separate gates.

ADR 0299 extends the SDK to `0.0.0-stage0.36` with
`std::bytes::append_byte(ref mut Buffer, u8) -> Result<usize, BufferError>`. It delegates to the
existing overlap-safe, failure-atomic Buffer append primitive with an exact count of one, returns
the new initialized length, and preserves the complete owner on failure. It adds no runtime symbol,
allocation policy, implicit growth promise, byte conversion, or `push`-style language convention.
The checked-in quickstart executes the generated-Fairy success path, while low and optimized MSVC
failure probes force growth allocation failure and require unchanged zero length/capacity plus zero
live allocations after automatic cleanup.

ADR 0373 extends the SDK to `0.0.0-stage0.55` with `std::bytes::append_i32`, `append_u32`,
`append_usize`, and `append_bool`. Each operation appends canonical locale-independent ASCII to a
caller-owned Buffer; integer paths use bounded stack storage and commit through one existing bulk
append. Generated-Fairy low and optimized MSVC execution proves exact boundary bytes, forced growth
failure with the initialized prefix and owner preserved, and zero live allocations. No formatting
syntax, parser, C varargs path, reflection hook, runtime symbol, or user-defined display protocol is
introduced.

ADR 0305 extends it to `0.0.0-stage0.39` with
`seek_current(ref mut File, i32)` and `seek_end(ref mut File, i32)`. The target shares one
same-handle seek helper across `FILE_BEGIN`, `FILE_CURRENT`, and `FILE_END`; public results expose
only the resulting absolute `usize`. Negative movement before byte zero fails without moving the
cursor, while movement beyond EOF does not change file length until a later write or `set_len`.
The bounded signed delta adds no wider integer type, allocation, path lookup, or runtime ABI layout.

ADR 0306 extends it to `0.0.0-stage0.40` with
`is_read_only(ref File) -> Result<bool, FileError>`. The target queries `FileBasicInfo` through the
same live handle and exposes only whether `FILE_ATTRIBUTE_READONLY` is present. A dedicated raw
probe and the public generated-Fairy quickstart cover normal/read-only files, closed and invalid
storage, and cursor/size preservation under low and optimized MSVC. Timestamps, atomic metadata,
attribute mutation, writable-access guarantees, ACLs, and Linux public linkage remain separate.

ADR 0307 extends it to `0.0.0-stage0.41` with
`hard_link_count(ref File) -> Result<usize, FileError>`. The target queries `FileStandardInfo`
through the same live handle and maps `NumberOfLinks` into the existing target-neutral scalar
transport. A dedicated raw probe creates a real second link and observes one then two, while the
public generated-Fairy quickstart observes count two through both host-created names. Both paths
preserve cursor and size under low and optimized MSVC. Hard-link mutation, path enumeration,
allocation size, timestamps, and Linux public linkage remain separate.

ADR 0308 extends it to `0.0.0-stage0.42` with
`hard_link(ref CStr, ref CStr) -> Result<i32, FileError>`. The target validates both bounded paths
before one no-replace `CreateHardLinkW` call. Raw and public generated-Fairy paths cover exact
collision and missing-source failures, shared identity/count/bytes, source-name removal, surviving
alias count one, and cleanup under low and optimized MSVC. It is not path containment, directory or
symbolic-link support, cross-volume fallback, or Linux public linkage.

ADR 0309 extends it to `0.0.0-stage0.43` with
`allocated_size(ref File) -> Result<usize, FileError>`. The target maps nonnegative
`FileStandardInfo.AllocationSize` through the same live handle. Raw and public generated-Fairy
paths cover ordinary, read-only, and two-link files, closed and invalid storage, and cursor/logical
size preservation under low and optimized MSVC. The value does not promise physical usage, block
shape, compression, sparse extents, quotas, billing, or concurrent stability.

ADR 0310 extends it to `0.0.0-stage0.44` with
`set_read_only(ref mut File, bool) -> Result<bool, FileError>`. The target reads current
`FileBasicInfo`, changes only `FILE_ATTRIBUTE_READONLY`, preserves zero timestamp fields and
unrelated attributes, and sets through the same handle. Raw and generated-Fairy low/optimized paths
cover true/false transitions, cursor/size preservation, closed and invalid storage, invalid boolean
input, and exact read-only-handle `ERROR_ACCESS_DENIED` with unchanged state. It is not an ACL,
sharing, filesystem-policy, ownership, or future-write guarantee.

ADR 0311 extends it to `0.0.0-stage0.45` with
`remove_dir_all(ref CStr) -> Result<usize, FileError>`. The target validates the bounded root and
every supplied prefix before effects, enumerates ordinary directories, removes observed directory
reparse points as leaves, and returns the exact number of already removed objects on failure. Raw
and generated-Fairy low/optimized paths cover a four-object tree, missing, sharing, and read-only
failures, partial progress, preserved error storage, and retry cleanup. Enumeration order is
filesystem-defined, and this is not an atomic, rollback-capable, race-free containment boundary.

ADR 0317 extends it to `0.0.0-stage0.46` with effect-free `path_kind_is_file` and
`path_kind_is_directory`. Both predicates exhaust all three `PathKind` variants and return false
for `.Other`; they add no C symbol and do not strengthen the underlying snapshot semantics.

ADR 0319 extends it to `0.0.0-stage0.48` with public
`metadata(ref File) -> Result<FileMetadata, FileError>`. The SDK sequentially calls the existing
same-handle size, allocated-size, hard-link-count, kind, and read-only operations, returning the
first error in that order. The result allocates nothing, moves no cursor or owner, adds no target
symbol, and is not an atomic snapshot.

ADR 0331 makes the create/open family, partial read/write, write-all, sync, close, and drop portable
across the two implemented targets. The remaining path, metadata, seek, exact-read, and Buffer APIs
stay specific to `x86_64-windows-msvc`; none of the slice is a portable Unicode path, retained-
pointer contract, or public resource-layout contract.

## Compiler-private contract-state runtime

ADR 0382 adds a stack-only target-neutral C implementation for the first native verified-state
handoff. It owns a single bounded 64-byte cell snapshot and overlay, exact layout/body identity
checks, read/write/delete, commit, and rollback. Low and optimized Windows MSVC probes cover commit,
reject, trap, read-your-write, malformed store and hash rejection, and zero live resources.

These `fycs_native_*_v0` names and record layouts are compiler-private. They are not part of
`fairyc-runtime/0`, the SDK, stable C ABI, production persistence, or target support. Maps,
multi-field stores, graphs, helpers, borrows, durable adapters, concurrency, and Linux emission stay
closed.

## Stage 0 product linkage

The constrained C unit records sorted unique external names used by actual calls. Product builds
resolve only exact `fy_std_*_v0` members through the shared standard-library contract, close their
component dependencies, and combine those C sources with the scalar runtime. Unused foreign
declarations do not select a component. Buffer and Arena require SystemHeap; owned text requires
SystemHeap, Buffer, and text view. Linux console, Linux File, Windows console, and Windows File are
target-specific components. The Windows File-to-Buffer adapter closes over Windows File,
SystemHeap, and Buffer and is ordered after all three; POSIX components are ordered before portable
sources so
their feature-test macros precede system headers.

This is a closed implementation link, not package discovery. Other foreign names receive no source,
object, library, search path, or linker argument from Fairy C and remain ordinary unresolved-link
errors unless the selected driver's fixed environment already provides them.

This is not a generic POSIX contract. POSIX.1-2024 and Linux differ for interrupted close behavior; other targets must define their own consuming-close rule before claiming support.

## Foreign pointer retention

- Call-bounded C use may be wrapped with a safe borrow if the unsafe wrapper proves no retention.
- Ownership transfer consumes `Buffer`; the drop obligation is disarmed only after confirmed transfer, and failure returns ownership.
- Retention until synchronous unregister requires a resource that owns both registration and storage. Unregister must make later foreign access impossible before returning and cannot fail in a way that threatens memory safety.
- A region or stack borrow cannot be passed to a retaining API.
- Retained, asynchronous, cross-thread, or fallibly-unregistering integrations that cannot meet these rules remain raw unsafe APIs.

## Composite resources

Composite resources may contain resources for infallible reverse-order drop. Stage 0 does not define a general fallible close protocol that extracts several owned fields or accumulates multiple close errors. Such APIs keep their leaves separate or remain target- and module-specific until a no-partial-move design is proven.

## `Vec<T>` on Windows

SDK `0.0.0-stage0.59` selects the core-plus-vector bundle when source imports
`std::vec`; other SDK imports keep the six-module core. The bundle's exact digest
and SDK version participate in lock and cache identity. The stable workspace CLI
checks, builds and runs it; tests require `--native` on Windows AMD64 MSVC.

`empty<T>` creates an allocation-free move-only vector. `len<T>`, `is_empty<T>`
and `capacity<T>` inspect it. `reserve<T>(ref mut Vec<T>, usize)` returns
`Result<usize, u32>` with an element capacity or the existing allocation status.
`push<T>(ref mut Vec<T>, T)` returns `Result<usize, T>`: failure preserves the
vector and returns the original element. `pop<T>` returns `Option<T>` and moves
the last element out. `truncate<T>(ref mut Vec<T>, usize new_length)` destroys
only the tail in reverse order, returns the removed count, and retains capacity.
A requested length at or above the current length is a no-op returning zero;
it never grows or initializes elements. `clear<T>` delegates to truncation at zero
and destroys the initialized prefix in reverse order,
returns its old length and keeps capacity. Cleanup destroys remaining elements
and frees the allocation. Neither operation destroys uninitialized spare capacity.

`at<T>(ref Vec<T>, usize)` returns an immutable owner-bound `ref T`; an invalid
index traps with `assertion-failed`. Mutation and movement are rejected while an
element reference remains live. Supported element types are explicit builtin
scalars or directly named non-generic structs/resources with an admitted nonzero
native layout. This does not admit user generic aggregate declarations or nominal
arguments to user generic functions. See ADRs 0390/0393 and `examples/std_vec_quickstart`.

## Explicit exclusions

Hidden allocation, infallible heap allocation, generic allocator vtables, public raw realloc, non-Windows or nested generic vectors, mutable vector element access, vector iteration, resource-aware arenas, safe spare capacity, implicit string allocation, mutable `String` bytes, portable `Path`, sockets, mapped files, async I/O, cross-thread callbacks, drop-time logging or allocation, and embedded or freestanding support claims remain excluded.

## External references

- Microsoft x64 `malloc` alignment:
  <https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/malloc?view=msvc-170>
- POSIX.1-2024 `malloc`: <https://pubs.opengroup.org/onlinepubs/9799919799/functions/malloc.html>
- Linux `write(2)`: <https://man7.org/linux/man-pages/man2/write.2.html>
- Linux `close(2)`: <https://man7.org/linux/man-pages/man2/close.2.html>
- Windows `CreateFileW`: <https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew>
- Windows `MoveFileExW`: <https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw>
- Windows `ReadFile`: <https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-readfile>
- Windows `CloseHandle`: <https://learn.microsoft.com/en-us/windows/win32/api/handleapi/nf-handleapi-closehandle>

These references constrain the initial hosted implementation. Fairy wrappers intentionally normalize implementation-defined allocator and file-descriptor behavior.
