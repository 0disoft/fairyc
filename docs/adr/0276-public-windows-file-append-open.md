# ADR 0276: Open a Windows file for append-only writes

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0147, ADR 0264, ADR 0269, ADR 0274

## Context

The public Windows `std::fs` slice can replace, update, create exclusively, or open-or-create a
file, but callers cannot request append semantics. Opening for update and seeking to the observed
end is not equivalent: another operation can change the file between those steps, and a later seek
can redirect the write.

Windows defines `FILE_APPEND_DATA` as append-only access. A handle with that write access sends each
write to the then-current end of file instead of honoring a caller-selected byte position. The
existing target-support open and write symbols can carry that access without adding a public native
symbol.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.20` to `0.0.0-stage0.21`. Language remains
   `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. Add `std::fs::open_append(ref CStr) -> Result<File, FileError>` with the `filesystem` effect.
   Like C binary append mode, it returns a write-only owner, creates an empty missing file, and
   preserves an existing file.
3. Extend the compiler-private open options with one `append` byte. Append requires write access,
   forbids truncate and exclusive-create combinations, maps write access to `FILE_APPEND_DATA`, and
   uses the existing `OPEN_ALWAYS` disposition. There is no existence precheck or seek-to-end step.
4. `seek_start` may still move the handle's observable file position, but a later append-handle
   write ignores that selected position and begins at the then-current EOF. The current
   `FILE_SHARE_READ` policy does not admit a second concurrent writer.
5. `write_all` can issue more than one underlying `WriteFile` for a sufficiently large input. Each
   call is append-positioned; the whole operation is not promised as one transactional or
   indivisible application record. Durability still requires explicit `sync`.

## Compatibility

The SDK source digest, package instance, public-interface hash, dependent identities, generated
locks, and whole-package cache keys change. The private source-to-target open-options record grows
from five bytes to six; compiler-bundled source, historical wrappers, and target support are built
together, and this pre-1 record is not a stable public native ABI. No C symbol, public resource
layout, language revision, runtime ABI, manifest schema, lockfile schema, or target profile changes.

## Evidence

- The raw Windows C probe opens a missing path for append, writes exact bytes, seeks back to zero,
  writes again, reopens, selects another non-EOF position, writes once more, and finally reads the
  exact concatenation through a read handle.
- The checked-in safe quickstart performs the same missing-create and repeated seek-resistant append
  sequence through generated Fairy under low and optimized MSVC profiles. The host removes only the
  named quickstart probe file before each profile.
- The complete Go suite, Go build/vet/format, Windows host smoke, documentation validation, and diff
  checks cover the synchronized implementation and contract.

## Consequences

Safe Fairy code can append without a race-prone explicit end lookup and without allowing a later
seek to overwrite existing bytes. This API does not add concurrent-writer sharing, record framing,
multi-call transactionality, portable paths, or Linux public linkage.
