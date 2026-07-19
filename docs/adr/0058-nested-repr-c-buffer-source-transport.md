# Nested `@repr(C)` Records and Buffer Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0057 connected one flat `@repr(C)` SystemHeap block to the real provisional allocation C unit.
The provisional Buffer record is not flat: it contains that block by value plus one exact `size_t`
length. Replacing the nested block with repeated pointer, size, and alignment fields would create a
different C record and hide the composition that the target standard-library implementation
actually uses.

Emitting source declarations alphabetically is also insufficient. C requires a complete nested
record definition before a containing record can use it by value, while Fairy source declaration
order is not an ABI dependency order.

## Decision

1. The constrained C backend accepts an acyclic, nonempty `@repr(C)` struct as a field of another
   nonempty `@repr(C)` struct. A non-`@repr(C)` nested struct remains outside C-layout transport.
2. Before layout or C emission, the backend computes a deterministic dependency order from nested
   struct fields. Dependencies are emitted before their users; otherwise independent definitions
   retain canonical-type ordering.
3. A by-value recursive struct dependency fails closed before layout. The backend does not emit an
   incomplete C declaration or attempt to assign a finite layout to an infinite value type.
4. A nested field reuses the already computed child layout as one field type. The selected target
   therefore owns nested alignment, internal padding, containing offsets, and trailing padding.
5. The internal Fairy Buffer transport record exactly mirrors the provisional C record: a nested
   `fy_std_system_heap_block_v0` followed by `c::size_t`. Generated C asserts the child 24-byte
   layout and the containing 32-byte layout with Buffer length at offset 24.
6. A generated Fairy caller constructs the exact empty nested record, calls the real
   `fy_std_buffer_initialize_v0`, grows its initialized prefix through the real
   `fy_std_buffer_resize_zeroed_v0`, observes capacity, alignment, length, and zeroed bytes through
   a bounded C observer, calls the real `fy_std_buffer_drop_v0`, and observes the normalized empty
   record.
7. Tiny C producers supply exact `size_t` zero, four, and eight because C-scalar literals remain
   closed. Only the three exercised major-suffixed Buffer symbols join the internal standard-library
   allowlist; the `fy_std_` prefix remains closed by default.
8. `fairyc.native-c-report/v22` adds `bufferSourceCallExecutions`, expected to be four after both
   compiler directions and both profiles execute successfully.
9. This is an internal source-to-C transport probe. It does not create a public
   `std::bytes::Buffer` type, safe methods, automatic standard-library linkage in
   `experimental build`, by-value aggregate FFI, deterministic OOM injection, or target
   conformance.

## Consequences

- Source records can now mirror nested C storage records without flattening or width-based aliases.
- Definition ordering is derived from value dependencies rather than source or public-tag spelling.
- Arrays, explicit C enums, callbacks, exports, by-value aggregate calls and returns, and public
  Buffer ownership semantics remain separate gates.
- Native execution remains configured but unobserved on this Windows host.
