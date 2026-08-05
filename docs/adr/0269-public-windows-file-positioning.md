# ADR 0269: Add absolute Windows file positioning

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0124, ADR 0147, ADR 0264

## Context

ADR 0264 publishes safe owned Windows file byte I/O, but a caller can only advance the file cursor
implicitly through reads and writes. It cannot inspect the current offset or return to an earlier
absolute byte position. That blocks small binary readers, header rereads, and deterministic
multi-pass processing even though the underlying target already owns a live synchronous `HANDLE`.

Adding relative and end-based seek together would also require signed-offset policy, end-of-file
reasoning, and sparse-file promises. Stage 0 does not need those larger contracts to make the
existing byte-oriented File surface materially more useful.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.14` to `0.0.0-stage0.15`. Language
   remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. `std::fs` adds `position(ref mut File) -> Result<usize, FileError>` and
   `seek_start(ref mut File, usize) -> Result<usize, FileError>`. Both declare `filesystem` and
   return the resulting absolute byte offset on success.
3. The surface is specific to the current 64-bit Windows MSVC target. Its `usize` therefore carries
   the complete 64-bit file offset admitted by the target component without adding unsupported
   Stage 0 `u64` source types.
4. The target component uses `SetFilePointerEx` with `FILE_CURRENT` for observation and
   `FILE_BEGIN` for absolute movement. It adds versioned symbols
   `fy_std_windows_file_position_v0` and `fy_std_windows_file_seek_start_v0` returning the fixed
   record `{ status: u32, os_code: u32, position: uint64 }`.
5. An absolute offset above `INT64_MAX` is rejected before an OS call with status 2, OS code 0,
   completed 0, and an unchanged file position. Closed or invalid handles retain the ordinary
   target error mapping. Neither operation allocates, transfers ownership, or changes close/drop
   behavior.
6. Relative seek, end-relative seek, truncation, sparse-gap content, metadata, portable path or
   offset types, and Linux public linkage remain outside this decision.

## Compatibility

The Fairy surface and closed Windows target-support symbol set are additive, but exact SDK
selection is intentionally not lock-compatible. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
The two new C symbols are pre-1 target-support contracts rather than a stable public native ABI.

## Evidence

- Go contract tests admit exactly the two new Windows-only symbols, require their Win32 primitive
  and origin constants, and keep target-neutral or other-target selection closed.
- The hand-authored separate-translation-unit C probe observes write position 9, rejects
  `UINT64_MAX` without moving the cursor, seeks to 5 and 0, reads the `File` suffix, and exercises
  closed-handle failures under low and optimized MSVC.
- The private source wrapper and compiler-bundled public quickstart execute the same absolute
  positioning lifecycle through generated Fairy C under both profiles. The public path also proves
  exact status 2, OS code 0, completed 0, preserved position, success result 42, and zero retained
  standard-library probe allocations.

## Consequences

Windows Stage 0 programs can now inspect and reset an owned File cursor without unsafe declarations
or hidden allocation. The API deliberately exposes only absolute start-based positioning; richer
random-access, file-sizing, truncation, sparse-file, portable offset, and cross-target contracts
remain separately reviewable work.
