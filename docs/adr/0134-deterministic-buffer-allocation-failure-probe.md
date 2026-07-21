# ADR 0134: Add deterministic Buffer allocation-failure probes

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Related decisions: `docs/adr/0041-failure-atomic-overlap-safe-buffer-c-slice.md`,
  `docs/adr/0085-windows-msvc-buffer-dependency-closure.md`,
  `docs/adr/0132-deterministic-owned-text-allocation-failure-probe.md`

## Context

Buffer promises that failed construction does not publish partial storage and that failed reserve,
append growth, or zeroed resize preserves the original pointer, capacity, alignment, initialized
length, and bytes. Overflow and malformed-state probes covered adjacent failures, but ordinary host
allocation could not reliably reach the exact out-of-memory branches. The Windows Fairy source
example observes successful Buffer metadata but cannot dereference its returned storage.

The shared probe-only SystemHeap adapter already provides deterministic one-shot allocation failure
and live-allocation counts without entering product linkage. Reusing it in the raw Buffer caller
proves the ownership boundary without adding a public allocator hook or widening the three admitted
Buffer symbols.

## Decision

1. `BufferProbeImplementationSource` prepends the shared SystemHeap probe adapter to the portable
   SystemHeap plus Buffer implementation. `WindowsMSVCBufferProbeImplementationSource` does the same
   for the target-specific 16-byte-alignment closure.
2. Product `StorageSource`, source-call fixtures, and target builds remain uninstrumented and contain
   no probe control symbols.
3. The raw probe forces nonempty initialization failure and requires the caller's sentinel output to
   remain unchanged with zero live allocations.
4. After normal initialization and a six-byte append, the probe separately forces reserve growth,
   relocating self-overlap append growth, and zeroed-resize growth failures. Each must preserve the
   exact pointer, capacity, alignment, length, six initialized bytes, and one live allocation.
5. A following unforced self-overlap append must produce `abcdefbcde`; zeroed resize must preserve
   that prefix and expose six zero bytes. Replacement allocation keeps one live block, and final drop
   returns the live count to zero.
6. The native Linux runner consumes the instrumented portable implementation when its glibc,
   Clang, and GCC lane runs. The native Windows smoke compiles the raw caller and target-specific
   implementation separately and executes low and optimized x64 MSVC profiles without WSL.
7. `fairyc.native-c-report/v31` is not bumped. The existing `bufferExecutions` success condition is
   strengthened without adding or changing a report field or counter.

## Consequences

- Buffer construction and all three allocation-bearing growth paths now have deterministic
  preservation and cleanup evidence on local Windows MSVC.
- The live-allocation oracle proves bounded cleanup for this closed probe, not general heap safety,
  thread-safe fault injection, allocator substitution, or a public Buffer API.
- Capacity growth remains private. The probe checks only the documented minimum capacity and exact
  initialized prefix, not a stable geometric growth factor or spare bytes.

## Verification

- Go source-contract tests require deterministic portable and Windows probe implementations,
  adapter-before-fallback ordering, Windows alignment injection, and absence of probe controls from
  product source;
- the raw C caller verifies unchanged output and storage across all four forced failures, recovery
  bytes, zero-filled growth, replacement ownership, and final live-allocation cleanup;
- the Windows host smoke inspects caller and callee COFF objects plus both linked PE/unwind images and
  executes low and optimized profiles with native x64 MSVC;
- the configured Linux native lane performs the same probe when a glibc Clang/GCC environment is
  available; it is not executed on this Windows-only development host.
