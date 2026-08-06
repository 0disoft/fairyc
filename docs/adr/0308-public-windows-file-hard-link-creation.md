# ADR 0308: Public Windows File hard-link creation

- Status: Accepted
- Date: 2026-08-06

## Context

ADR 0307 lets a Fairy C program observe the current hard-link count through an open Windows File,
but the public SDK cannot create a second name for an existing file. Callers that intentionally
deduplicate immutable content or preserve one file under an additional name must leave the supported
surface and write unsafe Win32 code.

The existing path APIs already establish bounded `ref CStr` input, exact `FileError`, no-replace
rename/copy behavior, and Windows-only bootstrap scope. A direct hard-link operation fits those
contracts without adding allocation, generic path types, file enumeration, or a compound metadata
record.

## Decision

SDK `0.0.0-stage0.42` adds:

```fairyc
std::fs::hard_link(ref CStr source, ref CStr destination) -> Result<i32, FileError>
```

The Windows adapter `fy_std_windows_file_hard_link_v0`:

- validates and converts both bounded paths before performing a filesystem effect;
- calls `CreateHardLinkW(destination, source, NULL)` exactly once;
- never replaces an existing destination;
- copies no file bytes or metadata and opens no persistent handle;
- returns zero on success and exact existing `FileError` status/OS-code storage on failure; and
- allocates no heap storage and performs no existence precheck.

The Win32 call is the no-replace creation decision; Fairy C does not implement a check-then-create
sequence. The source and destination must satisfy the active filesystem's hard-link rules, including
same-volume and provider capability constraints. The operation does not promise directory hard
links, symbolic-link behavior, cross-volume fallback, source immutability, durability, or metadata
isolation.

This API is not a sandbox or containment primitive. Source-name resolution, destination-parent
resolution, reparse points, permissions, sharing, external races, and later mutation of the shared
file remain Windows/filesystem policy. Both names refer to the same underlying file; removing one
name does not remove the object while another link remains.

## Evidence

- SDK and closed-symbol tests require the public function, exact private Windows symbol, component
  closure, and SDK `0.0.0-stage0.42`.
- `internal/cstdlib/stage0_windows_file_link.c` owns the target adapter while reusing the main
  Windows File source's bounded path conversion and result construction in one combined C unit.
- A dedicated raw probe covers null and invalid-encoding paths, missing-source no-create behavior,
  real creation, exact destination collision, equal `FileId`, source-name removal, surviving alias
  bytes, and cleanup under low and optimized MSVC.
- `examples/std_fs_quickstart/src/link_ops.fy` executes the same public workflow: success, collision,
  missing source with no destination, count two through both names, equal identity, exact bytes,
  source removal, surviving alias count one, and final cleanup under low and optimized MSVC.

## Consequences

Fairy C gains a small C-friendly hard-link workflow paired with ADR 0307's observation API. The SDK
version and source identity change, so generated locks and whole-package cache keys change; language
revision `fairyc-language/10` and runtime ABI `fairyc-runtime/0` do not.

This decision does not add path-based link-count queries, hard-link enumeration, symbolic-link APIs,
directory-link support, cross-volume copying, portable Unicode paths, Linux public `std::fs`
linkage, or a trusted-root containment abstraction.
