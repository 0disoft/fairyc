# ADR 0298: Observe one Windows filesystem path kind

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0277, ADR 0281

## Context

Public `std::fs` can classify an already-open `File`, but safe programs cannot distinguish an
ordinary file path from a directory path without attempting an ownership-producing open or a
mutating operation. The existing `FileKind` enum describes the target type behind one open handle:
`Disk`, `Character`, `Pipe`, or `Unknown`. Reusing it for path attributes would collapse a directory
into `Disk` and confuse two different observation contracts.

The Windows File component already owns bounded ASCII `CStr` conversion, exact target errors, and a
private fixed `{ status, os_code, kind }` result. A read-only path query can reuse those boundaries
without adding a public metadata record or promising that the observed name remains bound to the
same object after the call.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.34` to `0.0.0-stage0.35`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public exhaustive `std::fs::PathKind` variants `.File`, `.Directory`, and `.Other`, plus
   `path_kind(ref CStr) -> Result<PathKind, FileError>` with the existing `filesystem` effect.
   Keep `FileKind` and `kind(ref File)` unchanged.
3. Add private Windows-only `fy_std_windows_file_path_kind_v0(const uint8_t *)`, reusing the fixed
   raw kind-result record and the existing nonempty, NUL-terminated, bounded ASCII path conversion.
   Invalid representation fails before a filesystem lookup.
4. Observe the path once with `GetFileAttributesW`. Missing or inaccessible paths return stable
   file status `1`, the exact Windows error, `completed == 0`, and no successful `PathKind`.
5. Map an ordinary non-directory object to `.File`, a directory without reparse attributes to
   `.Directory`, and device or reparse-point attributes to `.Other`. The query does not expose
   Win32 attribute bits, follow a portable link policy, or claim that `.Other` is exhaustive across
   all providers.
6. Treat the result as a momentary path-name observation only. It is not a handle-pinned identity,
   containment proof, authorization decision, safe precondition for a later open, or defense
   against concurrent replacement. Callers needing stable identity retain `file_id(ref File)` on
   one already-open owner.
7. Add no allocation, retry, caching, mutation, enumeration, recursion, metadata persistence, or
   durability behavior. Portable Unicode paths and Linux public linkage remain separate gates.
8. Prove raw null/non-ASCII rejection, ordinary file and directory classification, exact missing
   error, and cleanup under low and optimized MSVC. Prove public file/directory classification,
   exhaustive `PathKind` switching, exact missing error, and cleanup through generated Fairy under
   both profiles. Keep reparse/device mapping under structural source checks because portable,
   privilege-independent reparse creation is not part of the current smoke environment.

## Consequences

- Safe programs can branch on an ordinary path's current broad kind without creating a `File`
  owner or deliberately invoking a failing mutation.
- `PathKind` stays semantically separate from handle-oriented `FileKind`, avoiding target constant
  leakage and misleading disk-versus-directory equivalence.
- `.Other` fails closed for observed reparse and device attributes, but the entire API remains a
  path snapshot with ordinary Windows name-lookup races.
- General metadata, timestamps, permissions, link counts, reparse-tag details, portable link
  policy, Unicode paths, Linux public linkage, and handle-relative authorization remain deferred.
- The additive SDK and private target-symbol change updates the source digest, public-interface
  hash, package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns bounded path conversion and attribute mapping.
- `internal/cstdlib/stage0_windows_file_path_kind_probe.c` owns raw success and failure execution.
- `internal/cstdlibcontract/` owns Windows-only symbol admission and component closure.
- `internal/sdkstdlib/sources/std/fs.fy` owns `PathKind` and the public wrapper.
- `examples/std_fs_quickstart/src/path_ops.fy` executes the public generated-Fairy lifecycle.
- `cmd/fairyc-windows-host-smoke/` compiles, inspects, and executes both evidence paths.
