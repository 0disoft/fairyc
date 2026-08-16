# ADR 0333: Make public File size observation target-neutral

- Status: Accepted
- Date: 2026-08-16
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0270, ADR 0331, ADR 0332

## Context

ADR 0270 added a shared-borrow `std::fs::size` operation, but its private symbol and native
implementation remained Windows-specific. After ADRs 0331 and 0332, an ordinary Linux product can
own a File, transfer bytes, and control its absolute position, yet it still cannot inspect the
length without reading to EOF or leaving the public SDK.

The already-open Linux descriptor exposes the same bounded observation without reopening a path.
Promoting only logical byte length avoids prematurely standardizing timestamps, allocation size,
file identity, permissions, or a compound metadata snapshot.

## Decision

SDK `0.0.0-stage0.51` maps the existing public `size(ref File)` function to
`fy_std_file_size_v0`. The closed source resolver selects exactly one target-owned File component
for Linux GNU or Windows MSVC.

The Linux adapter validates the private File storage, reports closed state before a system call,
and calls `fstat` on the already-open descriptor. It retries `EINTR`, maps a kernel failure to
status 1 with the exact `errno`, and maps an impossible negative `st_size` to status 1 with
`EOVERFLOW`. Success returns the nonnegative byte length as `usize`. Every outcome preserves the
cursor, owner, descriptor, allocation state, and output streams. This is a momentary observation;
concurrent external mutation is not made atomic.

The Windows adapter delegates the common symbol to the existing same-HANDLE
`fy_std_windows_file_size_v0` implementation and retains its established status mapping.

Host-independent contract tests require the common symbol on both implemented targets while
selecting only the target-owned File component. Generated-C tests require the common call and
reject retention of the Windows-specific size name.

Native report v35 extends the four `linuxPublicFileExecutions`. The ordinary generated-Fairy
program writes seven bytes, observes size seven at positions seven and two, proves both size
queries preserve the cursor, completes the existing absolute-seek rejection and restore sequence,
reopens and verifies the exact bytes, emits no output, and returns 42 under Clang and GCC at low
and optimized profiles. The Windows generated quickstart continues to cross the same public
function through the common adapter.

## Compatibility

The SDK advances to `0.0.0-stage0.51`, changing its source digest and dependent package/cache
identities. The public function signature, File layout, language `fairyc-language/10`, runtime ABI
`fairyc-runtime/0`, target identities, MIR, lockfile schema, and public ABI policy remain unchanged.

## Consequences

Linux and Windows programs can inspect an open File's logical byte length without moving the
cursor or exposing target handles. Relative/end seek, exact and Buffer-backed reads, other
metadata, path mutation, portable Unicode paths, containment, and external-write snapshot
guarantees remain separate contracts.
