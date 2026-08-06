# ADR 0307: Public Windows File hard-link count

- Status: Accepted
- Date: 2026-08-06

## Context

The public Windows `std::fs::File` surface can already observe byte length, target-neutral kind,
read-only metadata, and bounded identity through one live handle. It cannot report how many directory
entries currently name that same filesystem object. Callers that manage deduplicated files, safe
replacement workflows, or diagnostics therefore cannot distinguish an ordinary single-name file
from one with additional hard links without leaving the supported Fairy C surface.

Allocation size and timestamps were considered alongside link count. Allocation size has
filesystem-specific sparse, compression, and block-rounding semantics that would be easy to confuse
with byte length. Timestamps require a portable signed time representation and epoch contract that
Stage 0 does not yet have. Windows already exposes a bounded unsigned link count on the same handle,
and the current SDK can transport it as `usize` without adding a compound public metadata record.

## Decision

SDK `0.0.0-stage0.41` adds:

```fairyc
std::fs::hard_link_count(ref File) -> Result<usize, FileError>
```

The Windows adapter `fy_std_windows_file_link_count_v0`:

- validates and loads the existing inline `File` storage;
- rejects canonical closed storage with the existing closed status;
- calls `GetFileInformationByHandleEx` with `FileStandardInfo` on the same live `HANDLE`;
- maps `FILE_STANDARD_INFO.NumberOfLinks` into target-neutral `usize`;
- preserves the exact Win32 error on query failure; and
- does not allocate, reopen a path, retain a pointer, move the cursor, or change file length.

The result is a momentary observation. Another actor may create or remove a link immediately after
the call. The count is not an authorization decision, persistent identity, path enumeration, or
promise that every filesystem exposes identical link behavior. `FileId` remains the bounded equality
surface for comparing two live observations.

The existing private `raw_file_size_result` transport is reused because it already carries status,
OS code, and one pointer-width unsigned scalar. This is private SDK-to-target transport, not a public
C ABI record.

## Evidence

- SDK and closed-symbol tests require the public function, exact private target symbol, Windows-only
  linkage, component closure, and SDK `0.0.0-stage0.41`.
- `internal/cstdlib/stage0_windows_file_metadata.c` owns the query beside read-only metadata while
  reusing the main Windows File source's storage validation in one combined C translation unit.
- The dedicated raw metadata probe observes one link, creates a real second hard link with
  `CreateHardLinkW`, observes two links through the original live handle, rechecks cursor and size,
  and covers exact closed and invalid-storage results under low and optimized MSVC.
- The public quickstart receives a host-created source/alias hard-link pair. Generated Fairy opens
  both names, observes count two from each handle, and proves cursor and size preservation under low
  and optimized MSVC.

## Consequences

Fairy C gains a small C-friendly metadata primitive with deterministic native evidence and no new
resource, allocation, generic, or compound-record machinery. The SDK version and source identity
change, so locks and whole-package cache keys change; the language and runtime ABI do not.

This decision does not add hard-link creation or removal APIs, path-based link counts, link-name
enumeration, symbolic-link semantics, directory-link promises, allocation size, timestamps,
attribute mutation, portable Unicode paths, or Linux public `std::fs` linkage.
