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
std::bytes  Buffer
std::text   String, CStr, CString; arbitrary byte-to-text validation remains planned
std::fs     Linux File and open options
std::io     exact stdout/stderr byte writes, decimal i32 output, and one bounded stdin byte read
std::c      C scalar types and raw wrapper helpers
```

The standard library is supplied as an explicit package instance under `docs/package/package-graph-v0.md`. ADR 0147 embeds the current Stage 0 source package in the compiler and injects it only for actual `std` or `std::*` imports. SDK `0.0.0-stage0.8` exports opaque `std::alloc::Arena`, ADR 0246's public safe `std::bytes::Buffer`, ADR 0247's public safe `std::text::String` and `CString`, ADR 0205's exact `std::io::write` surface, ADR 0209's matching `write_stderr` surface, ADR 0210's matching immutable byte-view output operations, ADR 0211's bounded mutable-byte-view stdin read, and ADR 0215's allocation-free decimal `i32` output; the rest of the list remains a planned boundary backed in parts by provisional target C components and examples. The `std::*` module prefix does not create an implicit prelude or authorize host installation discovery.

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

The minimum operations are construction with capacity, copy from a byte slice, length and capacity inspection, immutable and mutable initialized-byte views, reserve, append, zero-initialized resize, truncate, and clear. Safe code cannot access spare capacity or raw parts. `Vec<T>` is deferred until user generics and element-drop behavior exist.

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
profiles. The four Linux executions cannot run on this Windows host, but ADR 0214 records their
first hosted success. Product linkage contains no probe controls.

## `Arena` and `Region`

`Arena` is created explicitly from `SystemHeap`. Region syntax exposes only:

- `alloc_zeroed_bytes(size)`;
- `copy_bytes(source)`;
- `place(value)` for a fully initialized inert value inferred from the expression.

There is no safe uninitialized byte allocation, generic `alloc<T>` syntax, resource placement, resource-aware reset, or fallible reset.

The bounded Stage 0 fixture surface exports `std::alloc::Arena` as an opaque resource. `region name in expression` accepts an expression of type `ref mut std::alloc::Arena` and holds that exclusive borrow for the lexical block. The fixture defines no constructor, capacity query, growth policy, ordinary allocation method, or representation guarantee; those remain public-library decisions.

The provisional C Arena slice now models one active Region per Arena. Begin captures a private mark
and generation; allocation publishes a tracking node only after both byte storage and node storage
succeed; reset frees allocations back to the mark, advances the generation, and invalidates the
Region. A second simultaneous Region, stale post-reset use, or Arena drop during an active Region
fails closed. Zero-size placement validates alignment without allocating. Representation, node
granularity, and growth policy remain private.

One internal generated Fairy caller now declares exact target-specific node, Arena, and Region
records, initializes and attaches them through the real helpers, obtains eight initialized zero
bytes through an exact `void **` output slot, then resets the Region and drops the Arena. Bounded C
observers check the allocation, generation transitions, active flags, released node chain, and
invalidated Region. This does not expose those records as the public representation, connect lexical
`region` lowering, or add safe allocation methods. Product builds now select Arena plus SystemHeap
for actual admitted calls only.

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
Safe lexical `region` lowering and public `std::alloc` construction remain outside this Windows slice.

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
with zero. Those raw fixtures are not themselves the public safe `std::text` API and do not
implement a borrowed `str` result from arbitrary bytes, nonempty input storage, Unicode normalization,
grapheme segmentation, case conversion, or locale behavior.

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

SDK `0.0.0-stage0.8` exposes move-only `String` and `CString` from `std::text`. `copy` and
`cstring_from_text` construct canonical owners fallibly; `append` preserves the existing String on
failure. `string_len`, `string_is_empty`, `cstring_len`, and `cstring_is_empty` expose byte metadata,
with CString length excluding its required terminator. `string_as_str` and `cstring_as_cstr` return
shared views tied to the exact owner borrow. `TextError` preserves separate text and allocation
status fields. Raw storage, capacity, mutable text bytes, implicit conversions, formatting, locale,
Unicode normalization, and general byte-to-text validation remain outside the public surface.

`examples/std_text_quickstart` imports only that safe module. Its Windows development and release
builds execute String copy and append, empty String, nonempty and empty CString, owner-bound views,
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
Linux Clang/GCC low/optimized executions are configured through the native report v32
`linuxConsoleExecutions` field and exercise stdin, stdout, stderr, and the full-domain `i32` decimal
boundaries, but remain unobserved on this
Windows host.

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

The initial API is explicitly tied to the `x86_64-linux-gnu` target and accepts `ref CStr`, not a purported portable UTF-8 path.

The minimum operations are fallible open, partial read, partial write, `write_all`, `sync`, and consuming `close`. `write` returns the actual byte count. A `write_all` error reports the number already written.

On the Linux v0 profile, `close(self)` consumes ownership on success and failure and is never retried automatically. A caller requiring durability uses `sync` before `close`. Automatic drop performs best-effort close, returns no error, allocates nothing, and emits no log.

The provisional Linux C helper now implements bounded C-string paths, closed open options, partial
read/write counts, progress-reporting `write_all`, `fsync`, consuming close, and silent best-effort
drop. Close clears ownership before one kernel call, including the failure path. Its native probe
uses three-byte reads and a forced `EBADF` close failure; four compiler/profile runs are configured
but unobserved.

A separate generated-Fairy source probe crosses exact operation-result, File, and open-option
records into the real C unit. It opens a prepared file for truncating writes, writes all nine
`FairyFile` bytes, synchronizes, consumes close, requires a second close to report the closed status,
reopens for reading, performs three three-byte reads, observes EOF with count zero, closes, and
requires the reconstructed bytes and normalized closed record. The companion C code owns only
bounded inputs and observations; it does not replace the production file operations. Public safe
File ownership, source effects, typed error resources, and hosted evidence remain disconnected.
Product build linkage is now automatic only for the five admitted raw File calls; safe source
construction and effect/error semantics remain disconnected.

## Windows `File` C slice v0

The provisional `x86_64-windows-msvc` component stores one `HANDLE` plus an explicit open byte
inline and normalizes the closed state to `{ NULL, 0 }`. It performs no hidden allocation. Open
accepts a bounded NUL-terminated ASCII byte path and closed read/write/create/truncate options,
converts the path into a stack-owned `WCHAR[4096]`, and calls `CreateFileW`. Empty paths, non-ASCII
bytes, missing termination within the bound, invalid option combinations, and non-closed output
storage fail before ownership changes. This narrow bootstrap boundary is not the future portable or
Unicode-complete path contract.

Read performs one `ReadFile` call and returns its exact partial count, including zero at EOF.
`write_all` loops over a component-private single-write helper, preserves the completed prefix on
failure, and reports no progress if a successful write completes zero bytes. Sync calls
`FlushFileBuffers` explicitly.

Explicit close first copies the live handle, normalizes the caller-owned state, and then calls
`CloseHandle` exactly once. The state therefore remains consumed even when the kernel rejects the
close, and automatic drop cannot retry it. Repeated explicit close reports the closed status.
Automatic drop similarly normalizes before one best-effort close, discards the error, allocates
nothing, and emits no output.

The raw result record is `{ status: u32, os_code: u32, count: usize }`. The closed target contract
admits only `fy_std_windows_file_open_v0`, `fy_std_windows_file_read_v0`,
`fy_std_windows_file_write_all_v0`, `fy_std_windows_file_sync_v0`,
`fy_std_windows_file_close_v0`, and `fy_std_windows_file_drop_v0`. A separate-translation-unit probe
executes create/truncate, write-all, sync, normal and repeated close, three-byte reads, EOF, forced
`ERROR_INVALID_HANDLE` close failure, drop, and repeated drop at both local MSVC profiles.

The example-level `examples/windows_file/windows_file.fy` module now connects that raw contract to
an effect-checked move-only FY `File`, typed `FileError`, create/truncate and read opens, write-all,
sync, consuming close, and `read(ref mut u8[])`. A mutable fixed array supplies a contextual exclusive
view with its canonical length and private writable data pointer. Runtime state crosses the private
opaque boundary through bytewise `memcpy` load/store so the C unit does not dereference generated
resource storage through an incompatible struct type. Low and optimized source builds write and
read nine bytes, observe EOF, execute explicit close, and prove implicit drop by reopening a path
whose write handle would otherwise deny sharing. This remains a target-specific example boundary,
not a stable `std::fs`, portable path, general mutable-slice operation set, retained-pointer
contract, or public resource-layout contract.

## Stage 0 product linkage

The constrained C unit records sorted unique external names used by actual calls. Product builds
resolve only exact `fy_std_*_v0` members through the shared standard-library contract, close their
component dependencies, and combine those C sources with the scalar runtime. Unused foreign
declarations do not select a component. Buffer and Arena require SystemHeap; owned text requires
SystemHeap, Buffer, and text view. Linux console, Linux File, Windows console, and Windows File are
target-specific independent components; POSIX components are ordered before portable sources so
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

## Explicit exclusions

Hidden allocation, infallible heap allocation, generic allocator vtables, public raw realloc, `Vec<T>`, resource-aware arenas, safe spare capacity, implicit string allocation, mutable `String` bytes, portable `Path`, sockets, mapped files, async I/O, cross-thread callbacks, drop-time logging or allocation, and embedded or freestanding support claims remain excluded.

## External references

- Microsoft x64 `malloc` alignment:
  <https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/malloc?view=msvc-170>
- POSIX.1-2024 `malloc`: <https://pubs.opengroup.org/onlinepubs/9799919799/functions/malloc.html>
- Linux `write(2)`: <https://man7.org/linux/man-pages/man2/write.2.html>
- Linux `close(2)`: <https://man7.org/linux/man-pages/man2/close.2.html>
- Windows `CreateFileW`: <https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew>
- Windows `ReadFile`: <https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-readfile>
- Windows `CloseHandle`: <https://learn.microsoft.com/en-us/windows/win32/api/handleapi/nf-handleapi-closehandle>

These references constrain the initial hosted implementation. Fairy wrappers intentionally normalize implementation-defined allocator and file-descriptor behavior.
