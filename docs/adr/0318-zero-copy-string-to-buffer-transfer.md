# ADR 0318: Zero-copy String-to-Buffer transfer

- Status: Accepted
- Date: 2026-08-06
- SDK: `0.0.0-stage0.47`
- Depends on: ADR 0246, ADR 0247, ADR 0263

## Decision

1. Add effect-free `std::bytes::empty() -> Buffer` as the canonical allocation-free empty owner.
2. Add effect-free `std::text::into_buffer(String) -> bytes::Buffer`. It transfers the exact
   allocation, capacity, alignment, and initialized byte length in constant time without allocation,
   copying, or a redundant UTF-8 scan.
3. Successful transfer stores canonical empty state into the consumed String before normal cleanup.
   The returned Buffer is mutable and no longer promises that later bytes remain valid UTF-8.
4. Admit only the private storage bridge `fy_std_string_into_buffer_storage_v0`; the typed C helper
   remains probe-only. Preserve language `fairyc-language/10` and runtime ABI `fairyc-runtime/0`.
5. Advance the compiler-bundled SDK to `0.0.0-stage0.47` because both public functions are additive.

## Evidence

The closed C probe forces every later allocation to fail, transfers a nonempty String back to Buffer,
checks pointer/capacity/length identity and canonical source invalidation, then releases the one live
allocation. It also covers null, aliasing, and empty storage. The checked-in generated-Fairy
quickstart performs Buffer-to-String-to-Buffer round trips, mutates returned bytes, preserves invalid
UTF-8 ownership, covers empty storage, and executes under low and optimized MSVC with zero live
allocations after return.

## Consequences

Callers can validate or build text and then hand its allocation back to byte-oriented code without a
copy. Mutation belongs only to the returned Buffer; converting it to String again must revalidate
UTF-8 through `from_buffer`. This does not expose resource fields, allocator identity, raw pointers,
or an unchecked mutable String view.
