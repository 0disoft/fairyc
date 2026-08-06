# ADR 0311: Public Windows recursive directory removal

- Status: Accepted
- Date: 2026-08-06

## Context

Fairy C can create a directory tree recursively but can remove only one empty directory at a time.
Callers otherwise need their own target-specific traversal and cannot observe partial cleanup. A
recursive deletion API must reject filesystem roots, avoid intentionally following observed reparse
points, and report completed destructive effects precisely.

## Decision

SDK `0.0.0-stage0.45` adds:

```fairyc
std::fs::remove_dir_all(ref CStr) -> Result<usize, FileError>
```

Success returns the number of removed filesystem objects, including files, reparse links,
directories, and the requested root. Failure stores the number already removed in
`FileError.completed`; removal is neither atomic nor rolled back.

The private Windows adapter `fy_std_windows_dir_remove_all_v0`:

- reuses bounded ASCII conversion and the existing directory root/component parser;
- rejects filesystem roots, trailing separators, `.` and `..` components, and extended namespace
  forms before deletion;
- queries every supplied path prefix and rejects an intermediate non-directory or reparse point;
- treats an observed final or child directory reparse point as a leaf and calls `RemoveDirectoryW`
  without enumerating its target;
- enumerates ordinary directories with `FindFirstFileW` and `FindNextFileW`, skips only `.` and
  `..`, deletes files and file links with `DeleteFileW`, closes every search handle with
  `FindClose`, then removes each emptied directory;
- fails with path-too-long before appending a child beyond the fixed 4096-wide-character buffer;
  and
- preserves the exact Win32 error and completed count on enumeration, deletion, sharing,
  read-only, path-length, or close failure.

Enumeration order is filesystem-defined. The completed count is exact for the order actually
executed, not a deterministic choice of which sibling disappears first. Concurrent external
replacement can race between attribute observation and the following path operation; this API is
not a containment or adversarial-race boundary. It does not follow a reparse point that remains
observed as such, but cannot promise race-free identity without a future handle-relative traversal.

## Evidence

- SDK and closed-symbol tests require the public API, target symbol, component closure, dedicated
  implementation/probe owners, and SDK `0.0.0-stage0.45`.
- The raw probe covers null, encoding, root, trailing-separator, dot-segment, missing, file-input,
  four-object success, deterministic one-object partial progress before an open-root sharing
  failure, read-only-child failure with zero progress, preservation, and cleanup under low and
  optimized MSVC.
- The generated-Fairy quickstart covers root/trailing/file rejection, exact four-object success,
  exact missing failure, one-open-child sharing failure with zero progress, close, retry, and exact
  two-object cleanup under low and optimized MSVC.

## Consequences

Fairy C gains a C-friendly recursive cleanup primitive with explicit destructive progress and no
allocation, exception, iterator, or generic callback surface. SDK identity, locks, and dependent
cache keys change; language and runtime ABI do not. Atomic trash moves, rollback, durability,
permission escalation, read-only clearing, handle-relative race freedom, portable Unicode paths,
and Linux public linkage remain separate gates.
