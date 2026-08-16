# ADR 0332: Target-neutral public File positioning

- Status: Accepted
- Date: 2026-08-16
- Depends on: ADR 0269, ADR 0331

## Context

ADR 0331 made the public File ownership and byte-I/O core available on Linux and Windows, but
`std::fs::position` and `std::fs::seek_start` still selected Windows-named private symbols. A
generated Fairy program could therefore create, read, and write a Linux file but could not observe
or deliberately change the current byte position through the public SDK.

## Decision

SDK `0.0.0-stage0.50` maps the existing public `position(ref mut File)` and
`seek_start(ref mut File, usize)` functions to `fy_std_file_position_v0` and
`fy_std_file_seek_start_v0`. The closed source resolver selects exactly one target-owned File
component for either symbol.

The Linux adapter operates on the descriptor already stored in the File owner. `position` calls
`lseek` with `SEEK_CUR` and a zero displacement. `seek_start` rejects an offset above `INT64_MAX`
before calling the kernel, then calls `lseek` with `SEEK_SET`. Both operations retry `EINTR`, return
the resulting absolute byte position, preserve the established invalid/closed/I/O status classes,
and never reopen a path. Seeking beyond end of file does not itself change file length.

The Windows adapter delegates to the existing same-HANDLE position and absolute-seek
implementation. Relative/end seek, size, metadata, exact-read, Buffer-read, path mutation, Unicode
path policy, containment, and no-follow authorization remain independently gated.

## Evidence

Host-independent contract tests require both common symbols to be admitted for Linux GNU and
Windows MSVC while selecting only the target-owned File component. Generated-C tests require both
common calls and reject retention of the corresponding Windows names.

Native report v34 extends the four `linuxPublicFileExecutions`. The ordinary generated-Fairy
program writes seven bytes, observes position seven, seeks to and re-observes position two, rejects
`usize` maximum with status 2 and unchanged position two, restores position seven, synchronizes,
consumes the owner, reopens and reads the exact bytes, consumes the second owner, emits no output,
and returns 42 under Clang and GCC at low and optimized profiles.
The existing low and optimized Windows generated-Fairy filesystem executions cross the same two
common symbols through the Windows adapter.

## Compatibility

The SDK advances to `0.0.0-stage0.50`, changing its source digest and dependent package/cache
identities. Public function signatures, File layout, language `fairyc-language/10`, runtime ABI
`fairyc-runtime/0`, target identities, MIR, lockfile schema, and public ABI policy remain unchanged.
