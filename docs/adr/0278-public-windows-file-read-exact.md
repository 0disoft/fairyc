# ADR 0278: Fill a Windows file read buffer exactly

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264

## Context

Public `std::fs::read` deliberately performs one bounded read and returns any short success, including
zero at end of file. Callers that require a fixed header or record currently need to loop, advance a
mutable destination, preserve completed-prefix failures, and distinguish ordinary EOF from complete
success. Stage 0 does not yet expose mutable slicing, so user code cannot express that loop without
unsafe pointer arithmetic or copying.

Windows synchronous `ReadFile` reports end of file as success with zero completed bytes. Treating
that result as a successful `read_exact` would contradict the operation name and silently leave an
uninitialized or stale suffix in the caller's buffer.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.22` to `0.0.0-stage0.23`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add public
   `read_exact(ref mut File, ref mut u8[] buffer) -> Result<usize, FileError>` with the
   `filesystem` effect. Success means every requested byte was written and returns `buffer.len`.
3. Add stable File status `7` for unexpected EOF. It carries OS code `0` and the exact initialized
   prefix length in `FileError.completed`. Bytes beyond that prefix keep their prior values.
4. Preserve existing status `1` and the exact Win32 code for an operating-system failure, while
   carrying the prefix completed by earlier successful reads. Closed and invalid argument statuses
   retain their existing meanings.
5. An empty request against an open canonical File succeeds with zero without calling `ReadFile`.
   File validation still occurs, so an empty request does not make closed or malformed storage
   successful.
6. Add private `fy_std_windows_file_read_exact_v0`. It repeatedly calls the existing bounded
   single-read primitive on the same File storage and advances only within the caller's buffer. It
   allocates nothing, retains no pointer, and stops on complete fill, zero progress, or failure.
7. The raw separate-translation-unit probe and public generated-Fairy quickstart must execute exact
   nine-byte success, a twelve-byte unexpected-EOF request with a nine-byte preserved prefix and
   untouched suffix, final cursor position nine, and closed/invalid-handle failures under low and
   optimized MSVC. The raw probe additionally covers empty success.

## Consequences

- Fixed-size binary readers gain a safe allocation-free operation without waiting for mutable slice
  syntax.
- On pipes, devices, or other blocking handles, `read_exact` may perform additional synchronous
  reads and therefore may block longer than `read`. It adds no timeout, cancellation, buffering,
  delimiter, decoding, or record framing.
- The buffer and File are mutated through existing exclusive borrows. On failure, the cursor and
  initialized prefix reflect completed reads; the operation does not roll them back.
- `File` and `FileError` layouts, language, and runtime ABI do not change. The new symbol and status
  are additive pre-1 Windows x86-64 MSVC compatibility data.
- Linux public linkage and a future portable implementation remain separate work.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the bounded exact-read loop.
- `internal/cstdlib/stage0_windows_file_probe.c` covers success, unexpected EOF, untouched suffix,
  empty input, cursor advancement, closed state, and invalid handles.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public safe wrapper.
- `examples/std_fs_quickstart/src/main.fy` carries complete and incomplete paths through generated C.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes both profiles.
