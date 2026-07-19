# ADR 0086: Windows MSVC Arena and Region lifecycle

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Component: `arena` requiring `system-heap`

## Context

ADR 0085 proved that a Windows target build can close one stateful Buffer component over
SystemHeap. Arena is the next dependency-closed component and exercises a harder boundary: a
self-linked node, one active Region, a `void **` output slot, generation invalidation, and cleanup of
all nodes allocated since a private mark.

The current FY subset can compare raw pointers but cannot safely dereference the returned allocation.
The Windows lane can therefore prove component selection and lifecycle state, but not zeroed payload
bytes. The existing Linux C-only and generated-source probes retain the byte-level and failure-matrix
responsibility.

## Decision

1. The Windows allowlist adds exactly `fy_std_arena_initialize_v0`, `fy_std_region_begin_v0`,
   `fy_std_region_allocate_zeroed_v0`, `fy_std_region_reset_v0`, and `fy_std_arena_drop_v0`.
2. Selecting any admitted Arena or Region symbol closes over SystemHeap. The support resolver emits
   SystemHeap before Arena, each at most once, and preserves independent Buffer and text-view
   selection in deterministic component order.
3. The checked-in FY example declares private 32-byte node, 24-byte Arena, and 32-byte Region
   `@repr(C)` records. Windows generation fields use target-width `usize`; for the selected x64 MSVC
   profile it is layout-compatible with the C implementation's eight-byte `uint64_t` storage. This
   is a private transport choice, not type identity or a public representation promise.
4. Low and optimized target builds require generation 1 after initialization; one active Region;
   non-null output and node state after an eight-byte, alignment-eight allocation; generation 2 and
   cleared Region state after reset; and generation 3 after Arena drop. Both images retain the
   existing direct SystemHeap, Buffer, UTF-8, LLP64, COFF, PE, and unwind observations and must
   return 42 without output.
5. This decision does not claim zeroed-byte observation, copied placement, simultaneous-Region or
   stale-use rejection, active-drop failure, forced allocation failure, safe lexical `region`
   lowering, a stable Arena representation, a public `std::alloc` API, hosted execution,
   conformance, or support.

## Consequences

- Windows target emission now executes one complete Arena/Region success lifecycle through the real
  project-owned C source at both compiler profiles.
- Arena and Buffer may be independently requested while their shared SystemHeap dependency is
  emitted once.
- The exact Windows standard-library surface remains fail-closed; owned text and every other
  unproven symbol still fail before MSVC discovery.

## References

- `docs/adr/0042-single-active-region-arena-c-slice.md`
- `docs/adr/0059-arena-region-source-transport.md`
- `docs/adr/0064-used-symbol-standard-library-auto-link.md`
- `docs/adr/0084-windows-msvc-system-heap-crossing.md`
- `docs/adr/0085-windows-msvc-buffer-dependency-closure.md`
- `docs/runtime/runtime-stdlib-v0.md`
