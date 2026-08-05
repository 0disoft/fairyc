# ADR 0275: Create a new Windows file without clobbering

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264, ADR 0274

## Context

The public Windows `std::fs` slice can replace, open, or open-or-create a file, but none of those
constructors can safely claim a new path without changing an existing file. A source-level
existence check followed by `create_truncate` has a check-use race and can destroy bytes created by
another actor between the two operations.

Windows already provides the required atomic disposition through `CreateFileW` with `CREATE_NEW`.
The existing target-support open function can select it through one private option without adding
another native symbol or changing the public `File` and `FileError` layouts.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.19` to `0.0.0-stage0.20`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add `std::fs::create_new(ref CStr) -> Result<File, FileError>` with the `filesystem` effect. It
   returns a write-only move-owned `File` for a newly created empty file at byte position zero.
3. Extend the compiler-private open options with one `exclusive` byte. The target component accepts
   that flag only with write and create enabled and truncate disabled, then makes one
   `CreateFileW` call with `CREATE_NEW`. There is no existence precheck.
4. If the path already exists, creation fails without opening or changing it. Windows reports the
   existing `FileError` values status `1`, OS code `ERROR_FILE_EXISTS` (`80`), and completed bytes
   `0`.
5. Path encoding, path length, sharing, consuming close, automatic drop, sync, and durability keep
   their existing contracts. This is not temporary-file creation, append, directory creation, or a
   portable cross-platform API.

## Compatibility

The SDK source digest, package instance, public-interface hash, dependent identities, generated
locks, and whole-package cache keys change. The private source-to-target open-options record grows
from four bytes to five; compiler-bundled source and target support are built together, and this
pre-1 record is not a stable public native ABI. No C symbol, public resource layout, language
revision, runtime ABI, manifest schema, lockfile schema, or target profile changes.

## Evidence

- The raw Windows C probe creates a distinct absent path, writes exact bytes, closes it, requires a
  second exclusive create to return `ERROR_FILE_EXISTS` into canonical closed output storage, then
  reopens and reads the unchanged bytes.
- The checked-in safe quickstart proves the same create/fail/reopen behavior through generated
  Fairy under both low and optimized MSVC profiles. The host removes only the named quickstart
  probe file before each profile.
- The complete Go suite, Go build/vet/format, Windows host smoke, documentation validation, and diff
  checks cover the synchronized implementation and contract.

## Consequences

Safe Fairy code can publish a new file without silently clobbering an existing path or introducing
an existence-check race. Callers still own collision policy and cleanup; this operation does not
invent a unique name or make a multi-file publication atomic.
