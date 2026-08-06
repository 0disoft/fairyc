# ADR 0294: Rename one Windows directory without replacement

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0292, ADR 0293

## Context

Public `std::fs` can create and remove one directory but cannot change its path without unsafe FFI.
The existing file-only rename intentionally rejects directories, while a broad move API would mix
file and directory policy, replacement, cross-volume copying, link handling, and durability. A
separate directory operation can reuse the bounded path and exact `FileError` transport while
keeping those larger policies excluded.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.31` to `0.0.0-stage0.32`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public `std::fs::rename_dir(ref CStr, ref CStr) -> Result<i32, FileError>` with the existing
   `filesystem` effect. Success returns `.ok(0)`.
3. Apply the existing nonempty, NUL-terminated, bounded ASCII conversion independently to source
   and destination. Conversion failures carry their stable status, OS code zero, and completed zero.
4. Add private Windows symbol `fy_std_windows_directory_rename_v0` to the existing Windows File
   component. Observe the source attributes and reject a non-directory with the stable invalid-
   argument status before requesting a move.
5. Invoke `MoveFileExW(source, destination, 0)`. An existing destination is not replaced, a
   cross-volume copy/delete fallback is not requested, and write-through is not requested. Exact
   Win32 failures are preserved.
6. A nonempty source directory moves with its children. The operation does not enumerate, open,
   close, copy, or delete those children and does not consume or mutate a `File` owner.
7. The attribute observation is path-based rather than handle-pinned. Concurrent external path
   replacement, junctions, reparse points, symlinks, inherited ACLs, compression, and encryption
   remain Windows target behavior rather than portable Fairy guarantees.
8. Prove invalid and non-directory sources, encoded-path rejection, missing source, destination
   collision, successful nonempty rename, old-path disappearance, moved-child access, cleanup, and
   repeated missing failure under low and optimized raw MSVC execution. Prove collision, success,
   old-path disappearance, child preservation, and cleanup through generated Fairy at both profiles.

## Consequences

- Safe programs can rename one directory tree within the admitted Windows operation without unsafe
  FFI or a recursive user-space copy.
- Destination replacement, cross-volume moves, atomic-swap semantics, directory iteration,
  recursive deletion, portable Unicode paths, durability barriers, and Linux public linkage remain
  separate gates.
- The additive SDK and private target-symbol change updates the source digest, public-interface
  hash, package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the one-call Windows adapter.
- `internal/cstdlib/stage0_windows_file_directory_probe.c` executes raw kind, collision, preservation, and
  cleanup evidence.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public wrapper.
- `examples/std_fs_quickstart/src/path_ops.fy` executes the generated-Fairy path lifecycle.
