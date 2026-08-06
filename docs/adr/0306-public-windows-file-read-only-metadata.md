# ADR 0306: Add same-handle Windows file read-only metadata

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0270, ADR 0277

## Context

Public Windows `std::fs::File` can already observe size, target-neutral handle kind, and bounded
file identity without reopening a path. It cannot answer the common portable question of whether
the opened object currently carries read-only metadata. Callers would otherwise need an unsafe
Win32 declaration or a separate path query whose result is not pinned to the live File owner.

Modification times are also useful, but Fairy C has not admitted a portable timestamp or signed
64-bit source representation. A broad metadata record would therefore either expose Windows
`FILETIME`, duplicate existing size and kind calls, or prematurely settle the future time model.
One explicit boolean query closes a demonstrated metadata gap without those commitments.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.39` to `0.0.0-stage0.40`. Language remains
   `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.
2. Add allocation-free public
   `is_read_only(ref File) -> Result<bool, FileError>`. It declares `filesystem`, borrows the live
   owner, and returns only target-neutral boolean meaning.
3. Add private Windows symbol `fy_std_windows_file_read_only_v0`. It calls
   `GetFileInformationByHandleEx(FileBasicInfo)` on the same stored HANDLE and maps
   `FILE_ATTRIBUTE_READONLY` to `0` or `1` in a private
   `{ status: u32, os_code: u32, enabled: u32 }` result.
4. Invalid and closed storage retain the existing status mapping. Kernel rejection returns status
   `1` plus the exact Win32 error; every failure returns `enabled = 0`.
5. The query does not move the cursor, change length, allocate, retain a pointer, reopen a path,
   modify attributes, or promise a stable snapshot against concurrent external metadata changes.
6. Keep timestamps, writable-access guarantees, permission mutation, ACLs, DOS attribute records,
   Linux public linkage, paths, sharing, close/drop behavior, and every non-SDK version unchanged.

## Evidence

- SDK and closed-symbol tests require the public function, exact private symbol, Windows-only
  linkage, component closure, and SDK `0.0.0-stage0.40`.
- `internal/cstdlib/stage0_windows_file_metadata.c` owns the attribute result and same-handle
  adapter while reusing the main Windows File source's storage validation in one combined C unit.
- A dedicated raw C probe runs low and optimized MSVC and observes normal then read-only metadata
  on one live handle, unchanged cursor and size, and exact closed plus invalid-storage results.
- The public quickstart receives one host-created read-only file and one ordinary file under each
  low/optimized profile. Generated Fairy C opens both, observes `true` and `false`, verifies
  position and size preservation, closes both owners, and returns 42.
- Metadata proof lives in separate C and Fairy source files; the existing near-threshold probe,
  implementation, and quickstart files remain below 40 KiB.

## Consequences

Windows Stage 0 programs can branch on opened-object read-only metadata without unsafe Win32 code
or a path reopen. The function describes the observed attribute, not whether a particular write
will succeed: handle access, ACLs, sharing, filesystem policy, and later external changes remain
independent. Portable timestamps, a compound metadata record, link counts, attribute mutation,
permission models, and Linux linkage remain separate gates.
