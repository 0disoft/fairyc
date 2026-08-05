# ADR 0271: Open existing Windows files for update

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0123, ADR 0147, ADR 0264, ADR 0269, ADR 0270

## Context

The public Windows File surface can create a truncated write handle or open an existing read-only
handle. It can also seek and observe size, but neither constructor gives one owner both read and
write access. A program therefore cannot inspect and patch an existing file in place even though
the private raw open contract already supports the required access flags.

Creating missing files or truncating an existing file would add separate data-loss and creation
policies. They are not required for the common bounded update case and would make this constructor
less predictable.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.16` to `0.0.0-stage0.17`. Language
   remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. `std::fs` adds `open_update(ref CStr) -> Result<File, FileError>` with the `filesystem` effect.
   It opens an existing file with read and write access, does not create or truncate it, and starts
   with the cursor at byte zero.
3. The wrapper reuses `fy_std_windows_file_open_v0` with exact options `{ read: 1, write: 1,
   create: 0, truncate: 0 }`. No C symbol, runtime ABI, language rule, resource layout, allocation,
   or output contract changes.
4. Open failure keeps the existing typed mapping: status 1, exact Win32 error, and completed byte
   count 0. A missing simple filename therefore returns `ERROR_FILE_NOT_FOUND` and remains absent.
5. Successful open preserves existing bytes and length. Only later explicit write operations may
   change data; the constructor itself does not synchronize, resize, or take a portable path.

## Compatibility

The Fairy surface is additive, but exact SDK selection is not lock-compatible. The SDK source
digest, package instance, public-interface hash, dependent identities, generated locks, and
whole-package cache keys change. The existing closed Windows target-support symbol set does not.

## Evidence

- The private wrapper and compiler-bundled SDK quickstart first create `FairyFile`, open it for
  update, seek to byte 5, write `Code`, and observe size 9.
- Both paths reset and read through the same read/write handle, then close and reopen read-only to
  observe the exact final bytes `FairyCode`.
- Both paths require a missing update-open to return status 1, Win32 error 2
  (`ERROR_FILE_NOT_FOUND`), completed 0, and no successful owner.
- Low and optimized generated-Fairy MSVC images return 42 after object, PE, and unwind inspection.

## Consequences

Windows Stage 0 programs can perform bounded in-place edits with one safe move-only File owner.
Create-without-truncate, explicit truncation or resizing, append policy, portable Unicode paths,
Linux public linkage, file locking, and concurrency guarantees remain separate contracts.
