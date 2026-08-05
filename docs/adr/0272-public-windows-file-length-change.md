# ADR 0272: Change Windows file length without moving the cursor

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0123, ADR 0147, ADR 0264, ADR 0269, ADR 0270, ADR 0271

## Context

The public Windows File surface can open an existing file for update, overwrite bytes, seek, and
observe size, but it cannot remove a stale suffix or deliberately extend the file. A caller cannot
therefore complete a bounded in-place replacement when the new representation has a different
length.

The cursor-based Win32 `SetEndOfFile` operation would make length changes depend on temporary file
position and require a restoration policy. Windows instead exposes `FileEndOfFileInfo` through
`SetFileInformationByHandle`, which accepts the new absolute end independently of the cursor.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.17` to `0.0.0-stage0.18`. Language
   remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. `std::fs` adds `set_len(ref mut File, usize) -> Result<usize, FileError>` with the `filesystem`
   effect. Success returns the requested new byte length. The exclusive borrow reflects mutation of
   the file's durable contents while preserving the owner.
3. The Windows target component adds
   `fy_std_windows_file_set_len_v0(void *, uint64_t)`. It rejects values above `INT64_MAX` with
   status 2 before calling Windows, then passes a `FILE_END_OF_FILE_INFO` to
   `SetFileInformationByHandle` with `FileEndOfFileInfo`.
4. Success truncates or extends the file without changing its current position. Bytes removed by
   truncation are lost. Contents between the old and new ends after extension are unspecified until
   the program writes them; the API does not promise zero filling.
5. A closed File returns status 3. A read-only handle or another kernel failure returns status 1
   with the exact Win32 error. Every outcome retains File ownership and reports completed bytes as
   zero. The operation does not allocate, synchronize, move the cursor, or expose the handle.

## Compatibility

The Fairy surface and closed target-support symbol set are additive, but exact SDK selection is not
lock-compatible. The SDK source digest, package instance, public-interface hash, dependent
identities, generated locks, and whole-package cache keys change. The new C symbol is a pre-1
target-support contract rather than a stable public native ABI.

## Evidence

- The separate-translation-unit C probe rejects `UINT64_MAX` without changing size or position,
  shrinks 9 bytes to 5, extends back to 9, and keeps position 9 throughout. It rewrites the
  unspecified extended region before reading and observes exact `FairyCode` bytes.
- The raw probe also requires status 3 for a closed File, exact `ERROR_ACCESS_DENIED` for a
  read-only handle, and exact `ERROR_INVALID_HANDLE` after externally closing the underlying
  handle.
- The private wrapper and compiler-bundled SDK quickstart execute the same shrink/extend,
  position-preservation, explicit rewrite, oversized rejection, and read-only failure paths.
- Low and optimized generated-Fairy MSVC images and independent C11 probe objects return success
  after object, PE, and unwind inspection.

## Consequences

Windows Stage 0 programs can finish bounded in-place replacements and deliberately resize a file
without using cursor state as a hidden input. Zero-filled extension, sparse-file guarantees,
allocation-size control, mapped-file coordination, portable paths, and Linux public linkage remain
separate contracts.
