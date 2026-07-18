# Runtime and Standard Library v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0007-runtime-stdlib-and-allocator-boundary.md`
- Target source: `docs/abi/c-abi-v0.md`

## Three layers

### Compiler-known operations

The compiler and MIR directly own checked, wrapping, saturating, and checked-result arithmetic; division and shift validation; bounds checks and traps; layout queries; bit copy and bit cast; built-in `Option` and `Result` tags and active-payload movement; resource move and drop glue; borrow and region markers; raw memory operations; and slice construction and indexing.

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
process statuses with `_Exit`, performs no cleanup, and emits no output. Product builds currently
bind base `128`; differential fixtures bind base `192` so their small ordinary-return oracle stays
disjoint. The complete product runtime, bounded source-location payload, runtime ABI version suffix,
and distribution bundle remain unimplemented.

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
std::text   UTF-8 validation, String, CStr, CString
std::fs     Linux File and open options
std::c      C scalar types and raw wrapper helpers
```

The standard library is supplied as an explicit package instance under `docs/package/package-graph-v0.md`. The `std::*` module prefix does not create an implicit prelude or authorize host installation discovery.

Built-in `Option`, `Result`, slices, `str`, and `CStr` get compiler-known view semantics; library modules provide operations over them.

## Allocation v0

There is no generic allocator interface, borrowed allocator context, or hidden default allocator. `SystemHeap` is an inert zero-size handle whose concrete implementation is selected by the target standard library. `Arena` is a resource and lexical `Region` is the only safe bulk-allocation capability.

Allocation is fallible and reports at least out-of-memory, size overflow, unsupported alignment, and capacity exhaustion. Stage 0 does not trap on OOM by default.

Zero-size requests do not call the platform allocator. They produce length and capacity zero; deallocation is a no-op; empty raw pointer identity is unspecified. Public raw `realloc` is excluded. `Buffer.reserve` checks size overflow, leaves the old allocation unchanged on failure, and cannot run with a live borrow into the buffer.

The first provisional `x86_64-linux-gnu` SystemHeap C slice lives in `internal/cstdlib`, not the thin
runtime. It admits only nonzero power-of-two alignments at or below `_Alignof(max_align_t)`, which
covers all currently admitted scalar layouts; over-aligned requests fail closed. Zero-size requests
still validate alignment but never call `malloc`. Growth and shrink use allocate-copy-release rather
than raw `realloc`, so validation and allocation failures leave the original block unchanged.

This helper is not yet part of generated C, `experimental build`, or a source-visible `std::alloc`
module. Its closed Linux probe checks empty normalization, alignment, prefix preservation, failure
rollback, malformed-block rejection, shrink, and release under Clang/GCC at low and optimized profiles. It does not
deterministically force host OOM, and no hosted success receipt has been observed.

## `Buffer`

`Buffer` is a resource owning initialized bytes from `SystemHeap`. It starts with length zero, exposes only `[0, len)`, and may grow only through fallible methods.

The minimum operations are construction with capacity, copy from a byte slice, length and capacity inspection, immutable and mutable initialized-byte views, reserve, append, zero-initialized resize, truncate, and clear. Safe code cannot access spare capacity or raw parts. `Vec<T>` is deferred until user generics and element-drop behavior exist.

## `Arena` and `Region`

`Arena` is created explicitly from `SystemHeap`. Region syntax exposes only:

- `alloc_zeroed_bytes(size)`;
- `copy_bytes(source)`;
- `place(value)` for a fully initialized inert value inferred from the expression.

There is no safe uninitialized byte allocation, generic `alloc<T>` syntax, resource placement, resource-aware reset, or fallible reset.

The bounded Stage 0 fixture surface exports `std::alloc::Arena` as an opaque resource. `region name in expression` accepts an expression of type `&mut std::alloc::Arena` and holds that exclusive borrow for the lexical block. The fixture defines no constructor, capacity query, growth policy, ordinary allocation method, or representation guarantee; those remain public-library decisions.

## Text types

`str` is an unsized, immutable, valid-UTF-8 language type used as `&str`. UTF-8 validation of `&[u8]` returns a borrow with the input's origin and performs no copy or allocation. Byte slices never convert implicitly to text.

`String` is a resource owning a UTF-8 `Buffer`. It can copy from `&str`, consume a `Buffer` after validation, expose `&str`, and append fallibly. A failed consuming UTF-8 conversion returns ownership of the original `Buffer` in its error resource. Mutable raw bytes are not exposed.

`CStr` is an unsized borrowed NUL-terminated byte sequence with no interior NUL and no UTF-8 promise. `CString` is the owned resource form. Conversion from `&str` copies bytes, rejects interior NUL, appends one terminator, and performs no locale conversion.

A `c"..."` literal provides allocation-free immutable `&CStr` storage with one compiler-appended trailing NUL and rejects a decoded interior NUL. It is source syntax, not a runtime constructor. `b"..."` similarly provides immutable static bytes without claiming UTF-8.

## Linux `File` v0

The initial API is explicitly tied to the `x86_64-linux-gnu` target and accepts `&CStr`, not a purported portable UTF-8 path.

The minimum operations are fallible open, partial read, partial write, `write_all`, `sync`, and consuming `close`. `write` returns the actual byte count. A `write_all` error reports the number already written.

On the Linux v0 profile, `close(self)` consumes ownership on success and failure and is never retried automatically. A caller requiring durability uses `sync` before `close`. Automatic drop performs best-effort close, returns no error, allocates nothing, and emits no log.

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

- POSIX.1-2024 `malloc`: <https://pubs.opengroup.org/onlinepubs/9799919799/functions/malloc.html>
- Linux `write(2)`: <https://man7.org/linux/man-pages/man2/write.2.html>
- Linux `close(2)`: <https://man7.org/linux/man-pages/man2/close.2.html>

These references constrain the initial hosted implementation. Fairy wrappers intentionally normalize implementation-defined allocator and file-descriptor behavior.
