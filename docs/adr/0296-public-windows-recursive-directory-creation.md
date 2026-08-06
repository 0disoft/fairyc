# ADR 0296: Create missing Windows directory parents

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0293, ADR 0294

## Context

Public `std::fs::create_dir` intentionally creates only one final component. Applications that own
a nested output, cache, or project path must therefore precompute and create every parent outside
safe Fairy code. A recursive creation operation is useful, but it must not inherit ambiguous
drive-relative paths, traverse lexical parent components, pretend rollback is atomic, or silently
accept a file where a directory is required.

The Windows File component already owns bounded ASCII-to-UTF-16 conversion and exact `FileError`
transport. It can validate the complete path before effects, then create one prefix at a time with
no allocation or new library dependency.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.32` to `0.0.0-stage0.33`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public `std::fs::create_dir_all(ref CStr) -> Result<i32, FileError>` with the existing
   `filesystem` effect. Success returns `.ok(0)` when the final directory was created or already
   existed as a directory.
3. Apply the existing nonempty, NUL-terminated, bounded ASCII conversion. Before any filesystem
   effect, reject `.` or `..` components, repeated interior separators, drive-relative `C:path`,
   Windows device namespaces such as `\\?\` and `\\.\`, and incomplete UNC roots with stable
   invalid-argument status `2`, OS code zero, and completed zero. Trailing separators are ignored
   down to the admitted root.
4. Admit ordinary relative paths, current-drive rooted paths, drive-rooted paths, and complete UNC
   share roots. Drive and UNC roots are observations, not creation targets. A root-only input
   succeeds only when the observed root exists as a directory.
5. Add private Windows symbol `fy_std_windows_directory_ensure_v0`. Walk the validated components
   from root to leaf and call `CreateDirectoryW` for each prefix. `ERROR_ALREADY_EXISTS` continues
   only when `GetFileAttributesW` confirms that prefix is a directory; a file collision retains the
   exact Windows error.
6. Return the first other `CreateDirectoryW` or attribute error exactly. Do not roll back prefixes
   created before a later OS failure. Those directories remain caller-owned recovery state and the
   operation may be retried after the failure is repaired.
7. Do not enumerate children, remove paths, change the process current directory, choose explicit
   ACLs, follow a portable link policy, allocate, retry externally changing paths, or claim a
   metadata durability barrier.
8. Prove raw null and lexical-parent rejection, nested creation, exact later invalid-name failure,
   and retained-prefix cleanup under low and optimized MSVC. Prove public nested creation,
   existing-directory success, file collision, exact partial-prefix preservation, and cleanup
   through generated Fairy at both profiles.

## Consequences

- Safe programs can establish a complete nested directory path without unsafe FFI or repeated
  source-level parent calls.
- Failure is explicit but not transactional: a successfully created prefix is an observable effect.
- Recursive deletion, directory iteration, replacement, atomic exchange, portable Unicode paths,
  device namespaces, drive-relative paths, durability barriers, and Linux public linkage remain
  separate contracts.
- The additive SDK and private target-symbol change updates the source digest, public-interface
  hash, package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns validation and prefix creation.
- `internal/cstdlib/stage0_windows_file_probe.c` executes raw validation, success, failure, and
  partial-prefix cleanup.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public wrapper.
- `examples/std_fs_quickstart/src/path_ops.fy` executes the generated-Fairy lifecycle.
