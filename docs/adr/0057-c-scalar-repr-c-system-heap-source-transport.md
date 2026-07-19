# C-Scalar `@repr(C)` Fields and SystemHeap Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0056 made every TargetSpec C scalar available as an exact foreign result, local, and argument,
but kept aggregate fields closed. The provisional SystemHeap block already has one stable internal C
shape: `void *data`, `size_t size`, and `size_t alignment`. Describing that record with Fairy
`usize` or `*mut u8` would happen to match the first target's width while lying about its C type.
Without an exact record, the existing SystemHeap C probe remains disconnected from generated Fairy
callers.

## Decision

1. The constrained C backend accepts all sixteen exact TargetSpec C scalars as fields of a nonempty,
   flat `@repr(C)` struct. It does not add them to private inert-struct transport, resources, enum
   payloads, nested aggregates, arrays, or by-value foreign aggregate calls.
2. A C-scalar field uses the same ordered source-to-TargetSpec-to-C table as a foreign scalar. Its
   exact C spelling, provider header, feature, target size, alignment, and signedness assertions are
   therefore shared rather than duplicated.
3. Raw `*void` and `*mut void` are admitted as opaque foreign and flat `@repr(C)` pointer values.
   They emit `const void *` and `void *`. Dereference, arithmetic, integer conversion, pointee casts,
   and raw pointers to C scalars remain closed.
4. The internal Fairy SystemHeap transport record is exactly `*mut void`, `c::size_t`, and
   `c::size_t`. Generated C asserts its 24-byte LP64 layout and field offsets in addition to the
   independent `size_t` target assertions.
5. A generated Fairy caller initializes the block, calls the actual
   `fy_std_system_heap_allocate_v0`, observes the mutated record through a bounded C test observer,
   calls the actual `fy_std_system_heap_deallocate_v0`, and observes the normalized empty record.
   Tiny C producers supply exact `size_t` zero and eight because C-scalar literals remain closed.
6. Only the two exercised major-suffixed SystemHeap symbols join the closed internal-standard-library
   allowlist. The `fy_std_` prefix does not become generally callable.
7. `fairyc.native-c-report/v21` adds `systemHeapSourceCallExecutions`, expected to be four after both
   compiler directions and both profiles execute successfully.
8. This is an internal source-to-C transport probe. It does not create a public `std::alloc` module,
   a safe `SystemHeap` constructor, allocation effects, automatic standard-library linkage in
   `experimental build`, deterministic OOM injection, or target conformance.

## Consequences

- Fairy source can now describe and mutate the real provisional SystemHeap block without
  width-based ABI aliases.
- Buffer, Arena, owned text, and File records can reuse the same exact flat-field mechanism in later
  source crossings, but nested resource wrappers and public APIs still need their own contracts.
- Native execution remains configured but unobserved on this Windows host.

ADR 0058 later admits acyclic nested `@repr(C)` fields and uses the exact SystemHeap block as the
first field of an internal Fairy Buffer record. Public resource wrappers remain outside both
decisions.
