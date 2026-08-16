# ADR 0334: Make bounded relative and end-based File seek target-neutral

- Status: Accepted
- Date: 2026-08-16
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0305, ADR 0331, ADR 0332, ADR 0333

## Context

ADR 0305 added bounded `seek_current` and `seek_end` operations, but their private symbols and
native implementation remained Windows-specific. Linux products can now own a public File, inspect
its logical size, and seek absolutely, yet common local backtracking and footer access still require
manual position arithmetic.

The existing public `i32` delta is intentionally smaller than the target offset domain and already
defines the portable source contract. Both implemented targets can apply that exact bounded delta
to the same open object without exposing a general seek-origin enum or a new integer type.

## Decision

SDK `0.0.0-stage0.52` maps the existing public `seek_current(ref mut File, i32)` and
`seek_end(ref mut File, i32)` functions to `fy_std_file_seek_current_v0` and
`fy_std_file_seek_end_v0`. The closed source resolver selects exactly one target-owned File
component for Linux GNU or Windows MSVC.

The Linux adapter validates private File storage, reports closed state before a system call, and
calls `lseek` on the already-open descriptor with `SEEK_CUR` or `SEEK_END`. It retries `EINTR`,
returns the resulting nonnegative absolute position as `usize`, and maps a kernel failure to status
1 with the exact `errno`. A failed movement leaves the descriptor position unchanged under the
target contract. Seeking beyond EOF moves only the cursor; it does not extend the file.

The Windows adapter delegates both common symbols to the established same-HANDLE
`SetFilePointerEx` operations and retains their status mapping. Both targets preserve File ownership,
perform no allocation or output, and keep the signed movement bounded to `i32`.

Host-independent contract tests require both common symbols on both implemented targets while
selecting only the target-owned File component. Generated-C tests require the common calls and
reject retention of the Windows-specific seek names.

Native report v36 extends the four `linuxPublicFileExecutions`. After the existing write, size, and
absolute-position checks, the generated-Fairy product seeks from position two to one, seeks from end
to five, then seeks three bytes beyond end to position ten. It proves logical size remains seven and
position remains ten, resets to two for the existing invalid-absolute-offset check, emits no output,
and returns 42 under Clang and GCC at low and optimized profiles. The Windows generated quickstart
continues to cross both public functions through the common adapter.

## Compatibility

The SDK advances to `0.0.0-stage0.52`, changing its source digest and dependent package/cache
identities. Public function signatures, File layout, language `fairyc-language/10`, runtime ABI
`fairyc-runtime/0`, target identities, MIR, lockfile schema, and public ABI policy remain unchanged.

## Consequences

Linux and Windows programs can perform bounded local backtracking and footer-relative movement
without manual arithmetic or target names. Wider signed offsets, exact and Buffer-backed reads,
other metadata, path mutation, portable Unicode paths, containment, asynchronous I/O, sparse-file
policy, and external-write snapshot guarantees remain separate contracts.
