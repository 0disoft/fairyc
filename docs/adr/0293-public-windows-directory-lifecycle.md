# ADR 0293: Create and remove one Windows directory

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0291, ADR 0292

## Context

Public `std::fs` can create, remove, and rename file paths but cannot establish or clean up a
directory used by an application. A broad directory API would prematurely combine parent creation,
recursive traversal, child deletion, link handling, access-control policy, and durability. Windows
already exposes narrow final-component creation and empty-directory removal operations whose exact
failures fit the existing bounded path and `FileError` contracts.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.30` to `0.0.0-stage0.31`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public `std::fs::create_dir(ref CStr) -> Result<i32, FileError>` and
   `std::fs::remove_dir(ref CStr) -> Result<i32, FileError>` with the existing `filesystem` effect.
   Each successful operation returns `.ok(0)`.
3. Apply the existing nonempty, NUL-terminated, bounded ASCII path conversion. A conversion failure
   carries its stable status, OS code zero, and completed zero.
4. Add private Windows symbols `fy_std_windows_directory_create_v0` and
   `fy_std_windows_directory_remove_v0` to the existing Windows File component.
   `create_dir` calls `CreateDirectoryW(path, NULL)`: it creates only the final directory, inherits
   the parent security descriptor through the Windows default, and reports exact collision or
   missing-parent failures. It does not create parents.
5. `remove_dir` calls `RemoveDirectoryW(path)` once. It removes only an existing empty directory and
   reports exact missing, non-empty, access, or sharing failures. It does not enumerate, unlink
   children, retry, or recurse.
6. Neither operation opens, closes, consumes, or mutates a `File` owner, changes the process current
   directory, or claims a metadata flush or power-loss durability barrier.
7. Junction, reparse-point, symlink, inherited ACL, compression, and encryption behavior remains the
   Windows target behavior. None is promoted to a portable Fairy guarantee or configurable policy.
8. Prove invalid paths, missing parents, file/directory collisions, non-empty removal failure, child
   cleanup, exact empty removal, repeated missing removal, and cleanup under low and optimized raw
   MSVC execution. Prove the public lifecycle through generated Fairy at both profiles.

## Consequences

- Safe programs can establish and clean up one explicit directory without unsafe FFI.
- Parent creation, recursive deletion, directory rename, directory iteration, portable Unicode
  paths, explicit permissions, durability barriers, and Linux public linkage remain separate gates.
- The additive SDK and private target-symbol change updates the source digest, public-interface hash,
  package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the two one-call Windows adapters.
- `internal/cstdlib/stage0_windows_file_directory_probe.c` executes raw failures and cleanup through
  the combined Windows File probe translation unit.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public wrappers.
- `examples/std_fs_quickstart/src/path_ops.fy` executes the generated-Fairy lifecycle.
