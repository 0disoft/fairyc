# ADR 0277: Observe a Windows file handle kind

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264, ADR 0270

## Context

The public Windows `std::fs` slice can observe a file's size and position through its open handle,
but callers cannot distinguish an ordinary disk file from a character device or pipe. Reopening a
path to learn that fact would weaken the existing same-handle ownership boundary and introduce a
check/use race. A full metadata record would prematurely combine kind, timestamps, attributes,
links, directories, and target-specific values.

Win32 `GetFileType` already classifies the object behind one `HANDLE` as disk, character, pipe, or
unknown. Its `FILE_TYPE_UNKNOWN` return is ambiguous: it may be a successful unknown
classification or a failure, distinguished by the thread last-error value.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.21` to `0.0.0-stage0.22`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add opaque inert public `std::fs::FileKind` and
   `kind(ref File) -> Result<FileKind, FileError>` with the `filesystem` effect. Add
   `kind_is_disk`, `kind_is_character`, `kind_is_pipe`, and `kind_is_unknown`; no target numeric
   constant becomes public API.
3. Query only the already-open shared File handle. The operation allocates nothing, performs no
   path lookup, does not move the cursor, and does not transfer ownership.
4. Add private `fy_std_windows_file_kind_v0(const void *)` returning the fixed C record
   `{ status: u32, os_code: u32, kind: u32 }`. It validates canonical File storage, reports status
   3 for closed state, clears last error immediately before `GetFileType`, and maps a nonzero error
   after `FILE_TYPE_UNKNOWN` to status 1 with the exact Win32 code.
5. Normalize every successful raw value other than disk, character, or pipe to public unknown.
   `FILE_TYPE_REMOTE` remains unused target detail rather than a public category.
6. The raw separate-translation-unit probe must cover a disk file, `NUL` as a character device,
   closed storage, and a forcibly invalidated handle. The public generated-Fairy quickstart must
   execute disk and `NUL` classification under low and optimized MSVC and return 42.

## Consequences

- Callers gain a small target-neutral capability query without a second path lookup or raw Win32
  constants.
- `File`, `FileError`, language, and runtime ABI layouts do not change. The new private result and
  symbol remain pre-1 Windows x86-64 MSVC implementation compatibility data.
- `FileKind` is an opaque value plus predicates instead of a public enum because the current C
  backend does not yet support a user enum carried inside a built-in `Result` payload. This keeps
  the slice executable without broadening the language/backend in the same change.
- ADR 0279 later adds bounded same-handle file identity. Timestamps, attributes, directories,
  link counts, path-based metadata, portable Unicode paths, Linux public linkage, and snapshot
  guarantees remain separate work.
- ADR 0280 later closes the cited C-backend `Result` payload limitation. `FileKind` remains opaque
  because changing an already published SDK type is a separate source-compatibility decision.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns the same-handle adapter and error disambiguation.
- `internal/cstdlib/stage0_windows_file_probe.c` executes disk, character, closed, and invalid
  handle branches against the separately compiled component.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public opaque value and predicates.
- `examples/std_fs_quickstart/src/main.fy` carries disk and `NUL` observations through generated C.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes the raw and generated-Fairy
  paths under low and optimized MSVC.
