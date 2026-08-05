# ADR 0281: Promote Windows FileKind to a target-neutral public enum

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0277, ADR 0280

## Context

ADR 0277 introduced same-handle file-kind observation as an opaque public struct plus four predicate
functions because the C backend could not carry a user enum inside built-in `Result`. ADR 0280 closes
that backend gap with target-owned private enum and outer Result layout plus native execution.

The opaque value prevents callers from expressing the most useful property of this classification:
the four target-neutral cases are closed and can be switched exhaustively. Exposing raw Win32
`FILE_TYPE_*` values would instead couple source code to one target and would mishandle successful
unknown classifications.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.24` to `0.0.0-stage0.25`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Change public `std::fs::FileKind` from a private-field struct to an inert enum with exactly
   `.Unknown`, `.Disk`, `.Character`, and `.Pipe` variants.
3. Keep `kind(ref File) -> Result<FileKind, FileError>` and its `filesystem` effect unchanged. Map
   private raw kind 1 to `.Disk`, 2 to `.Character`, 3 to `.Pipe`, and every other successful value
   to `.Unknown`. Failure remains `.err(FileError)`.
4. Retain `kind_is_disk`, `kind_is_character`, `kind_is_pipe`, and `kind_is_unknown`. Implement each
   as an exhaustive enum switch so supported predicate-based callers remain source-compatible.
5. Do not change `fy_std_windows_file_kind_v0`, its fixed raw C result, Win32 handling, File layout,
   FileError layout, effects, allocation behavior, cursor behavior, or ownership behavior.
6. Update the generated-Fairy `std::fs` quickstart to switch directly over disk and character
   results while also exercising the retained predicates. Execute it under low and optimized MSVC.

## Compatibility

- The SDK source digest, public-interface hash, package instance, dependent identities, generated
  locks, and whole-package cache keys change, so the SDK version advances.
- Supported callers that only receive `FileKind` from `kind` and pass it to the retained predicates
  remain source-compatible. The previous private field could not be named or constructed by an
  external module.
- Callers may now switch exhaustively. The four variants are therefore a public closed source
  contract; adding another variant requires a later compatibility decision.
- No public native layout is promised before 1.0, and the private Windows C contract does not change.

## Consequences

- Callers can express complete control flow without a chain of predicates or target constants.
- Successful unknown classification remains distinct from target failure.
- FileKind remains target-neutral even though the current public `std::fs` linkage is Windows-only.
- Direct enum pattern matching raises the cost of adding a future category, which is intentional for
  a closed classification rather than an open metadata record.

## Evidence

- `internal/sdkstdlib/sources/std/fs.fy` owns the enum, raw normalization, and retained predicates.
- `examples/std_fs_quickstart/src/main.fy` switches exhaustively over disk and `NUL` character-device
  observations and checks the compatibility predicates.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes the generated-Fairy quickstart
  under low and optimized MSVC.
- `internal/sdkstdlib/stdlib.go` owns SDK `0.0.0-stage0.25`.
