# ADR 0085: Windows MSVC Buffer dependency closure

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Component: `buffer` requiring `system-heap`

## Context

ADR 0084 proved direct SystemHeap allocation and release on Windows. A real standard-library build
also needs dependency closure: selecting a higher component must pull in its lower component once,
in deterministic source order, without exposing user linker inputs. Buffer is the smallest stateful
consumer of SystemHeap and already has a shared C implementation.

The Fairy source subset cannot safely dereference the private allocation pointer, so this slice can
observe record state and lifecycle but cannot independently verify zeroed byte contents. Existing
Linux C and source probes retain that byte-level responsibility.

## Decision

1. The Windows allowlist adds exactly `fy_std_buffer_initialize_v0`,
   `fy_std_buffer_resize_zeroed_v0`, and `fy_std_buffer_drop_v0`.
2. Selecting any admitted Buffer symbol closes over SystemHeap. The support resolver emits
   SystemHeap before Buffer, each at most once, applies the Windows x64 16-byte heap-alignment
   contract once, and then appends independently selected text view and scalar runtime.
3. The checked-in FY example declares a private nested 32-byte `@repr(C)` Buffer record over the
   24-byte SystemHeap block. It initializes capacity eight, resizes to length four, requires capacity
   eight, alignment one, and length four, drops the Buffer, and requires all scalar state to return
   to zero.
4. Low and optimized Windows target builds keep the direct SystemHeap and UTF-8 observations. Both
   images must compile through MSVC, pass intermediate COFF and linked PE/unwind inspection, execute
   without output, and return 42.
5. This decision does not claim that the Windows path observes zeroed contents. It does not admit
   reserve, append, truncate, clear, Arena, owned text, a public `std::bytes` API, arbitrary native
   inputs, hosted execution, conformance, or support.

## Consequences

- The Windows product path now proves target-specific dependency-closed standard-library linkage,
  not only independent leaf components.
- Buffer lifecycle metadata crosses nested generated/source-owned records at both compiler profiles.
- The allowed native surface remains exact and auditable; requesting the next component still fails
  before MSVC discovery.

## References

- `docs/adr/0041-failure-atomic-overlap-safe-buffer-c-slice.md`
- `docs/adr/0058-nested-repr-c-buffer-source-transport.md`
- `docs/adr/0084-windows-msvc-system-heap-crossing.md`
- `docs/runtime/runtime-stdlib-v0.md`
