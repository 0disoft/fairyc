# ADR 0291: Remove one existing Windows file path

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0275, ADR 0290

## Context

Public `std::fs` can create, open, update, append, truncate, synchronize, and close a Windows file,
but safe Fairy code cannot remove a file after its owner is closed. Examples and applications must
therefore leave temporary files behind or cross an unrelated unsafe foreign boundary. Removal is a
path mutation rather than an operation on one `File` owner, so it needs exact path, sharing, missing,
and error semantics before becoming public.

The existing Windows component already owns bounded ASCII-to-UTF-16 path conversion and the stable
`FileError` status mapping. One target operation can close this gap without adding allocation,
directory traversal, recursive behavior, or implicit owner changes.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.28` to `0.0.0-stage0.29`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public
   `std::fs::remove_file(ref CStr) -> Result<i32, FileError>` with the existing `filesystem`
   effect. Success removes one existing file path and returns `.ok(0)`.
3. Preserve the current Stage 0 path boundary: the path is nonempty, NUL-terminated ASCII with a
   maximum scanned storage of 4,096 code units. Invalid, non-ASCII, or unterminated paths retain the
   existing stable path statuses and carry OS code zero.
4. Add private Windows symbol `fy_std_windows_file_remove_v0`. It calls `DeleteFileW` exactly once.
   A target failure returns file status `1`, the exact Win32 code, and completed count zero. A
   missing path is an error rather than idempotent success.
5. Do not close, consume, or mutate an open `File` owner. The current open policy grants
   `FILE_SHARE_READ` but not delete sharing, so removal of a path with one such live handle fails
   with `ERROR_SHARING_VIOLATION`; the caller must close the owner and retry explicitly.
6. Prove raw null and non-ASCII rejection, open-sharing failure, exact successful removal, and
   second-removal `ERROR_FILE_NOT_FOUND` under low and optimized MSVC. Prove the same public sharing,
   close, success, reopen-missing, and repeated-missing behavior through generated Fairy at both
   profiles.

## Consequences

- Safe programs can explicitly clean up ordinary files without unsafe FFI or hidden owner effects.
- The operation is not recursive and does not remove directories. Directory removal, rename,
  replacement, trash/recovery, permission changes, portable Unicode paths, and Linux public linkage
  remain separate contracts.
- Reparse-point and symlink behavior is not promoted to a portable Fairy guarantee by this Windows
  slice.
- The additive SDK and private target-symbol change updates source digest, public-interface hash,
  package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the one-call target operation.
- `internal/cstdlib/stage0_windows_file_probe.c` executes raw path, sharing, success, and missing
  cases.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public safe wrapper.
- `examples/std_fs_quickstart/src/main.fy` executes the public generated-Fairy lifecycle.
