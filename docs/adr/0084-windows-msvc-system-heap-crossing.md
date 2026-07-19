# ADR 0084: Windows MSVC SystemHeap crossing

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Symbols: `fy_std_system_heap_allocate_v0`, `fy_std_system_heap_deallocate_v0`

## Context

ADR 0083 proved one allocation-free call into project-owned C. The next runtime boundary is real
allocation and release through the platform CRT. The shared SystemHeap source used
`_Alignof(max_align_t)` as its maximum fundamental alignment, but the MSVC C headers used by the
local toolchain do not define `max_align_t` in `<stddef.h>`. Inventing a replacement typedef would
risk disagreeing with the allocator and SDK.

Microsoft documents that x64 `malloc` returns storage aligned to 16 bytes and requires a separate
aligned allocator above that boundary. That target fact can drive the existing SystemHeap policy
without duplicating its allocation state machine.

## Decision

1. The shared SystemHeap source obtains its maximum fundamental alignment through
   `FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0`. Its default remains `_Alignof(max_align_t)` for the Linux
   C11 path.
2. When and only when Windows target linkage selects SystemHeap, the support source defines that
   ceiling as 16 bytes before the shared component. This is a target contract, not host probing or
   a guessed typedef.
3. The Windows allowlist adds exactly `fy_std_system_heap_allocate_v0` and
   `fy_std_system_heap_deallocate_v0`. Resize and every other standard-library or foreign symbol
   remain rejected before toolchain discovery.
4. The checked-in FY example declares a private 24-byte `@repr(C)` record containing `*mut void`,
   `usize`, and `usize`. It requests eight bytes at alignment eight, requires the returned record to
   report both values, deallocates it, and requires size/alignment to return to zero.
5. Low and optimized `build-target` executions also retain the ADR 0083 UTF-8 call. Each generated
   caller and selected support unit must pass MSVC compilation, intermediate COFF inspection,
   linked PE/unwind inspection, native execution, and result 42.
6. This does not admit over-aligned allocation, resize, forced OOM evidence, Buffer/Arena/owned-text,
   a public `std::alloc` API, arbitrary libraries, CRT redistribution, static CRT policy, installer
   packaging, hosted execution, conformance, or target support.

## Consequences

- Windows target execution now reaches real UCRT allocation and release through project-owned
  SystemHeap code instead of only allocation-free C.
- Target-specific allocator alignment is explicit and testable while the state machine remains one
  shared C implementation.
- The Windows standard-library allowlist grows by two exact symbols without becoming a generic
  native linker interface.

## References

- Microsoft `malloc` alignment contract:
  <https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/malloc?view=msvc-170>
- Microsoft x64 alignment convention:
  <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- `docs/adr/0040-provisional-system-heap-c-slice.md`
- `docs/adr/0057-c-scalar-repr-c-system-heap-source-transport.md`
- `docs/adr/0083-windows-msvc-closed-stdlib-crossing.md`
