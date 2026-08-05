# ADR 0279: Compare bounded Windows file identities

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264, ADR 0277

## Context

Paths are names, not stable object identities. Two open paths can name the same file through a
rename, hard link, or another alias, while equal-looking path text can be reopened after the target
changes. ADR 0277 already classifies the object behind one open handle, but callers cannot compare
two live observations without exposing raw handles or reopening a path after a check.

Windows documents [`FILE_ID_INFO`](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_info)
as a 64-bit volume serial number plus a 128-bit file identifier.
Comparing both values determines whether two open handles represent the same file on one computer.
The older 64-bit index in `BY_HANDLE_FILE_INFORMATION` is not guaranteed unique on ReFS, so it is
not sufficient for the public contract.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.23` to `0.0.0-stage0.24`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add opaque inert public `std::fs::FileId`,
   `file_id(ref File) -> Result<FileId, FileError>` with the `filesystem` effect, and effect-free
   `file_id_equal(ref FileId, ref FileId) -> bool`.
3. `FileId` retains the complete volume serial and 128-bit target identifier but exposes no fields,
   integer conversion, ordering, hashing, display, serialization, or persistence contract.
4. Add private `fy_std_windows_file_id_v0`. It validates the canonical File storage and non-null
   output, calls `GetFileInformationByHandleEx` with `FileIdInfo` on the same open handle, and copies
   the result to caller storage only after success. Failure preserves caller output and uses the
   existing File status and exact Win32 code.
5. Equality compares every retained bit. It is meaningful only for bounded observations on one
   computer. A FileId is not an authorization token, durable database key, cross-machine identity,
   or guarantee against identifier reuse or filesystem-specific change over time.
6. The raw separate-translation-unit probe must execute same-file reopen equality, distinct
   simultaneously live disk-file inequality, null-output rejection, closed and invalid-handle
   failure, and unchanged failure output under low and optimized MSVC. The public generated-Fairy
   quickstart must execute equality across two simultaneously open handles to the same disk file and
   inequality against a second live disk file under both profiles.

## Consequences

- Callers gain a same-handle, allocation-free identity observation without exposing Windows handles,
  paths, or target constants.
- The operation avoids a path check/use race for the observation itself, but later path-based rename,
  delete, reopen, or authorization remains a separate operation with its own race and policy.
- Filesystems and network providers may reject the query or provide behavior with narrower guarantees.
  Pipe and character-device identity are not promised by this disk-file evidence.
- `File`, `FileError`, language, and runtime ABI layouts do not change. The private 24-byte transport
  and one target symbol are additive pre-1 Windows x86-64 MSVC compatibility data.
- Timestamps, attributes, link counts, directories, portable Unicode paths, and Linux public linkage
  remain separate work.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns same-handle `FILE_ID_INFO` observation.
- `internal/cstdlib/stage0_windows_file_probe.c` covers raw success, comparison, and failure integrity.
- `internal/sdkstdlib/sources/std/fs.fy` owns the opaque public value and equality operation.
- `examples/std_fs_quickstart/src/main.fy` carries same-file and distinct-file comparisons through
  generated C.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes both profiles.
