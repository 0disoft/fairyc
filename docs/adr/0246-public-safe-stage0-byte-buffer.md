# ADR 0246: Public safe Stage 0 byte Buffer

- Status: Implemented local compiler, SDK, and native-smoke gate
- Date: 2026-07-31
- SDK source: `internal/sdkstdlib/sources/std/bytes.fy`
- Runtime source: `internal/cstdlib/stage0_buffer.c`
- Symbol contract: `internal/cstdlibcontract/`
- Example: `examples/std_bytes_quickstart/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0041, ADR 0058, ADR 0085, ADR 0134, ADR 0135, ADR 0147

## Context

The Stage 0 runtime already had a failure-atomic, overlap-safe byte Buffer with allocation-failure
probes, but Fairy source could reach only three raw symbols and had no public owned byte container.
Publishing allocation and mutation without a safe view would create a container whose contents
could not be inspected without exposing its raw pointer. Text-only raw-parts built-ins could not
construct an owner-bound `ref u8[]` or `ref mut u8[]`.

## Decision

1. SDK `0.0.0-stage0.7` adds `std::bytes`. Its move-only `Buffer` owns the exact private runtime
   storage and calls `fy_std_buffer_drop_v0` from infallible cleanup. `BufferError` preserves the
   runtime's stable `u32` allocation status.
2. The public safe functions are `with_capacity`, `copy`, `len`, `capacity`, `is_empty`, `view`,
   `view_mut`, `reserve`, `append`, `resize_zeroed`, `truncate`, `clear`, and `error_status`.
   Allocation and mutation remain explicit `Result` operations. No method syntax, hidden growth,
   implicit text conversion, spare-capacity access, allocator parameter, or general `Vec<T>` is
   introduced.
3. Language `fairyc-language/7` adds exactly two unsafe compiler-known operations:
   `byte_slice::from_raw_parts(owner, data, length)` and
   `byte_slice::from_raw_parts_mut(owner, data, length)`. The shared form requires a shared safe
   owner borrow. The mutable form requires both an exclusive safe owner borrow and mutable raw
   data. Both return the owner's exact borrow origin; neither allocates, copies, or changes length.
4. The SDK uses those raw-parts operations to expose the initialized prefix. Safe callers
   cannot obtain capacity bytes, use an immutable owner for a mutable view, move or mutate the
   Buffer while a derived view remains live, or retain a view after cleanup.
5. The closed target symbol contract now admits all seven Buffer operations and preserves the
   existing Buffer-to-SystemHeap dependency. The runtime ABI stays `fairyc-runtime/0`; the SDK and
   language pins change because source-visible API and compiler-known meaning changed.

## Evidence and limits

Semantic tests reject mutable raw-parts construction with a shared owner or immutable data. HIR,
verified MIR, and constrained-C tests freeze both operations, unsafe provenance, result types, and
owner mutability. The complete Go suite passes.

The checked-in quickstart contains no `unsafe` or raw pointer. On Windows it executes copy,
reserve, append, zeroed growth, checked mutable-view stores, checked shared-view reads, truncate,
clear, and implicit drop in both development and release builds. The configured MSVC 14.51 smoke
inspects and executes both resulting PE images and finishes with 57 structurally inspected images.

This does not stabilize the private four-field layout as a foreign ABI, expose uninitialized spare
capacity, permit resizing while a view is live, add generic element storage, define iterator or
formatting APIs, prove hosted same-commit CI, or promote another target. Allocation remains outside
the effect system under the existing Stage 0 policy.
