# ADR 0305: Add bounded relative and end-based Windows file seek

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0269, ADR 0270

## Context

Public Windows `std::fs::File` can observe its cursor and seek to an absolute `usize` offset. A
binary reader still has to combine `position`, checked arithmetic, and `seek_start` to move a small
distance backward, while reading a footer requires a separate size query and the same arithmetic.
Those reconstructions are noisy and can fail before the target's existing signed seek primitive is
used.

Fairy C does not expose `i64` as a public Stage 0 type. Promoting a new integer width solely for file
offsets would be wider than the demonstrated need, while an origin enum would duplicate the already
public, C-friendly `seek_start` name. Two explicit operations with an `i32` delta cover common
header, footer, and local backtracking work without changing the language type system.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.38` to `0.0.0-stage0.39`. Language remains
   `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.
2. Add allocation-free public operations
   `seek_current(ref mut File, i32) -> Result<usize, FileError>` and
   `seek_end(ref mut File, i32) -> Result<usize, FileError>`. Both declare `filesystem`, preserve
   File ownership, and return the resulting absolute position.
3. Keep the signed delta intentionally bounded to the existing `i32` source domain. Larger signed
   relative movement remains unavailable until Fairy C admits a wider signed integer for broader
   language reasons.
4. Add private target symbols `fy_std_windows_file_seek_current_v0` and
   `fy_std_windows_file_seek_end_v0`. They share the existing same-handle `SetFilePointerEx` adapter
   with `FILE_CURRENT` and `FILE_END`, respectively, and retain the existing fixed position-result
   record.
5. A movement whose resulting position would be before byte zero returns the exact Windows failure,
   completed zero, and leaves the cursor unchanged. Closed and invalid handles retain the existing
   status and OS-code mapping.
6. Seeking beyond EOF changes only the cursor. It does not extend the file; a later write or
   `set_len` owns any length and gap-content consequences. Neither operation implies sparse-file,
   allocation-size, durability, or concurrent external-write guarantees.
7. Keep paths, reparse behavior, sharing, close/drop behavior, report layouts, generated-C dialect,
   and every non-SDK version unchanged.

## Evidence

- SDK and closed-symbol tests require both public functions, exact private symbols, Windows-only
  linkage, component closure, and SDK `0.0.0-stage0.39`.
- A dedicated raw C probe runs low and optimized MSVC and covers negative current movement,
  start-preserving failure, negative end movement, a position beyond EOF with unchanged size,
  reset from EOF, and closed storage.
- The private Windows File wrapper and compiler-bundled public quickstart execute relative and
  end-based movement through generated Fairy C under low and optimized MSVC. They verify exact
  positions, failure storage, unchanged size, result 42, and ordinary automatic cleanup.
- The public quickstart moves positioning proof into its own source module so no modified source
  file crosses the repository's 40 KiB refactoring threshold.

## Consequences

Windows Stage 0 programs can perform bounded local backtracking and footer-relative reads without
manual position arithmetic or unsafe declarations. The explicit names remain visually C-friendly
and avoid a speculative general seek-origin abstraction. Wider signed offsets, Linux public
linkage, portable offset types, asynchronous I/O, and sparse-gap guarantees remain separate gates.
