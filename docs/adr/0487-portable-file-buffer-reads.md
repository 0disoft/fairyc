# ADR 0487: Portable file-to-Buffer reads

- Status: Accepted for prototype; runtime and public SDK linkage implemented
- Date: 2026-10-24
- Extends: ADR 0288 and the target-neutral File core

## Decision

Provide `fy_std_file_read_append_v0` on the Linux GNU and Windows MSVC targets.
Keep one reserve/read/commit implementation in `stage0_file_buffer.c`, with small
target adapters. The Windows-only legacy symbol and raw result remain available.
The public `read_append` and `read_to_end` signatures need no new parameter,
allocation policy or effect. SDK `0.0.0-stage0.87` selects the portable symbol.
The SDK version changes because its bundled source and foreign-link identity
change; the language and runtime ABI revisions remain unchanged.

The operation validates File before Buffer. A zero maximum validates both
owners without allocating or making an OS read. Overflow and allocation failure
occur before I/O, preserving cursor, allocation, capacity, length and bytes.
After successful reserve, a read failure preserves the initialized prefix and
length; increased capacity remains owned by the Buffer. A successful short read
publishes exactly its returned bytes. EOF returns zero without changing length.
The operation retains no pointer and adds no runtime loop beyond the File
adapter's existing interrupted-read handling. `read_to_end` keeps its SDK loop.

The result retains three `u32` fields and a `usize` count: file status, OS code,
allocation status, count. Linux records errno in the OS-code field; Windows
records the Win32 error. File and allocation failures remain separate. No status
number, runtime ABI revision, language revision or public File storage changes.

Each target's component closure includes File, SystemHeap and Buffer before the
adapter. The shared C source is also included before the Windows adapter in the
authored compiler's authenticated runtime pack. The existing asset IDs retain
their meanings; the shared source is appended as a new asset.

## Verification and remaining work

`TestFileBufferNative` uses production C with test-only allocation and OS-read
observers. It covers prefix preservation, short reads, EOF, overflow, zero
maximum, allocation failure before reads, write-only File failure, closed File
precedence and zero live allocations after cleanup. Windows also exercises the
legacy symbol; Linux injects EINTR before a successful read. Both MSVC profiles
pass. Linux GCC/Clang profiles are registered in the same test for an available
Linux runner.

The Linux probe cross-compiles and links at `-O0` and `-O2` with Zig's Linux GNU
2.39 target and warnings as errors. This is not Linux execution evidence.
`TestSDKFileBufferEmission` verifies both target selections from public Fairy
source. `TestSDKFileBufferExecution` passes MSVC low/optimized builds for binary
bytes, an initialized prefix, zero maximum, short reads, EOF, write-only errors,
and a 65,539-byte `read_to_end` crossing its 65,536-byte read window. Files are
closed and removed in an isolated directory. The same program is registered for
GCC/Clang low/optimized execution on Linux.

`TestFileBufferSDKAllocationFailuresNative` reuses the existing generated-Fairy
failure probes at both MSVC profiles: failed initial growth and failed second
read window preserve cursor and committed bytes, with zero retained probe
allocations after cleanup. The SDK embedding check and generated-Fairy SDK
pipeline also pass. The authored SDK bundle's two native tests authenticate the
exact core/vector sources and reject changed, incomplete and over-budget input.
The source check covers 539 files and 6,905 MIR functions. Linux native execution
remains outstanding.

The candidate's SDK source pin, literal parts and bundle version are updated.
An older candidate correctly rejects the changed SDK as code 6801. Refreshing
that bootstrap seed and the complete candidate/fixed-point records remains
separate work. No target maturity or distribution claim changes.
