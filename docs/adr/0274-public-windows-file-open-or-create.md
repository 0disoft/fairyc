# ADR 0274: Open or create a Windows file without truncating it

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264, ADR 0271

## Context

The public Windows `std::fs` slice can replace a file with `create_truncate` or update an existing
file with `open_update`, but it cannot atomically obtain a read/write file that may not exist while
preserving bytes when it does exist. A source-level existence check followed by one of those
constructors would introduce a check-use race and still cannot express the desired disposition.

The closed Windows target component already validates and implements the exact raw option
combination with one `CreateFileW` call and `OPEN_ALWAYS`. Exposing that existing path needs no new
C entry point, raw options layout, native record, ownership rule, or error mapping.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.18` to `0.0.0-stage0.19`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add `std::fs::open_or_create(ref CStr) -> Result<File, FileError>` with the `filesystem` effect.
   It returns the existing move-only `File` owner and existing typed `FileError` failure.
3. The wrapper calls `fy_std_windows_file_open_v0` with read, write, and create enabled and truncate
   disabled. The target component maps that combination to one `CreateFileW` call using
   `OPEN_ALWAYS`; it does not precheck existence.
4. A missing path creates an empty file. An existing path preserves its contents. Both outcomes
   start at absolute byte position zero, so this is not append mode. Writes are durable only under
   the existing explicit `sync` contract.
5. Path encoding, path length, sharing, status/Win32-code mapping, automatic drop, consuming close,
   and private resource layout remain unchanged. Exclusive create, append, portable Unicode paths,
   directory operations, and Linux public linkage remain separate contracts.

## Compatibility

The SDK source digest, package instance, public-interface hash, dependent identities, generated
locks, and whole-package cache keys change. No C symbol, raw options layout, language revision,
runtime ABI, manifest schema, lockfile schema, target profile, or public resource layout changes.
Existing `create_truncate`, `open_read`, and `open_update` behavior remains unchanged.

## Evidence

- The checked-in safe quickstart opens one absent path, observes size and position zero, writes four
  exact bytes, closes it, then opens the same path again, observes preserved size and position zero,
  and reads those exact bytes.
- The Windows host smoke removes only the named probe files inside its copied quickstart build
  directory before each execution, then builds, inspects, and executes the generated Fairy program
  under low and optimized MSVC profiles.
- The complete Go suite, Go build/vet/format, Windows host smoke, documentation validation, and diff
  checks cover the synchronized implementation and contract.

## Consequences

Safe Fairy code can initialize or reopen a file without destroying existing bytes and without a
race-prone existence precheck. Callers must still choose deliberately among replacement,
open-or-create, and existing-only update; this API does not infer intent or provide append or
exclusive-create semantics.
