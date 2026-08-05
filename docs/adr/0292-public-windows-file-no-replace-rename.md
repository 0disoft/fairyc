# ADR 0292: Rename one Windows file without replacement or cross-volume copy

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0279, ADR 0291

## Context

Public `std::fs` can create and remove files but cannot change one file's path. A conventional
rename name alone would leave important behavior ambiguous: whether an existing destination is
replaced, whether a cross-volume operation silently becomes copy-and-delete, whether directories
are admitted, whether live File owners are closed, and whether success implies metadata durability.

The current Windows component already owns bounded path conversion and exact `FileError` mapping.
Windows `MoveFileExW` exposes replacement and cross-volume copy as opt-in flags, so omitting those
flags provides a narrow no-replace, no-copy boundary. Because that target operation can also move a
directory, a source attribute gate is needed before publishing a file-only API.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.29` to `0.0.0-stage0.30`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public
   `std::fs::rename_file(ref CStr, ref CStr) -> Result<i32, FileError>` with the existing
   `filesystem` effect. The arguments are source then destination; success returns `.ok(0)`.
3. Apply the existing nonempty, NUL-terminated, bounded ASCII path conversion independently to both
   paths. A path conversion failure carries its stable status, OS code zero, and completed zero.
4. Add private Windows symbol `fy_std_windows_file_rename_v0`. It observes source attributes first.
   Missing or inaccessible source metadata returns exact Win32 failure. A source carrying
   `FILE_ATTRIBUTE_DIRECTORY` is rejected with stable invalid-argument status `2` and OS code zero.
5. For an admitted file source, call `MoveFileExW(source, destination, 0)`. Do not set
   `MOVEFILE_REPLACE_EXISTING`, `MOVEFILE_COPY_ALLOWED`, or `MOVEFILE_WRITE_THROUGH`. An existing
   destination fails unchanged, and a cross-volume request is not simulated with copy-and-delete.
   The API makes no final-directory flush or power-loss durability claim.
6. Do not close, consume, or mutate a live `File` owner. The current non-delete-sharing open policy
   makes the controlled live-source rename fail with `ERROR_SHARING_VIOLATION`; callers explicitly
   close and retry.
7. Treat concurrent external path replacement between attribute observation and rename as
   target-defined interference. The attribute gate prevents ordinary directory use but is not a
   handle-pinned authorization or race-free identity check.
8. Prove raw invalid source/destination paths, directory rejection, missing source, destination
   collision with both files preserved, live-source sharing failure, final success, old-path
   disappearance, exact final bytes, and cleanup under low and optimized MSVC. Prove the public
   collision, preservation, sharing, success, and final-byte lifecycle through generated Fairy at
   both profiles.

## Consequences

- Safe programs can perform an explicit no-clobber file rename without unsafe FFI.
- Replacement rename, directory rename, cross-volume move, copy, hard-linking, durability barriers,
  portable Unicode paths, and Linux public linkage remain separate contracts.
- Reparse-point and symlink behavior is not promoted to a portable Fairy guarantee.
- The additive SDK and private target-symbol change updates source digest, public-interface hash,
  package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the attribute gate and target rename.
- `internal/cstdlib/stage0_windows_file_probe.c` executes raw rejection, preservation, sharing, and
  success behavior.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public wrapper.
- `examples/std_fs_quickstart/src/main.fy` executes the generated-Fairy lifecycle.
