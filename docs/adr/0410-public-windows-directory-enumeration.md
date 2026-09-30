# ADR 0410: Public Windows directory enumeration

- Status: Accepted
- Date: 2026-09-30

## Context

The private directory-size-analyzer workload cannot discover files using the current
public SDK. File metadata exists, but directory child enumeration is missing.

## Decision

SDK `0.0.0-stage0.71` adds the following Windows AMD64 operations to `std::fs`:

```fairyc
read_dir_utf8(ref str) -> Result<ReadDir, FileError>
next_dir(ref mut ReadDir, ref mut u8[]) -> Result<DirEntry, FileError>
close_dir(ReadDir) -> Result<i32, FileError>
```

`ReadDir` owns a search handle and one fixed-size native state allocation. Explicit
close and cleanup consume the state before attempting `FindClose` and free it once.
Open failure leaves no owned state. Ordinary File or console users do not select
the new `windows-directory` runtime component.

`DirEntry` contains `has_entry`, `name_length`, `kind: PathKind`, `size: usize` and
`is_reparse`. The entry name is copied as UTF-8 into the supplied buffer without a
NUL terminator. The caller owns that buffer; no stored borrow or implicit clone is
introduced. Size means observed logical file bytes, and is meaningful for ordinary
files. `has_entry=false` means stable EOF, distinct from an enumeration error.

Enumeration skips `.` and `..`, includes hidden entries, and preserves filesystem
order. Reparse entries are reported as `.Other` with `is_reparse=true`; callers can
skip them explicitly. A short name buffer fails with file status 6 and Windows
`ERROR_INSUFFICIENT_BUFFER`, stores required bytes in `FileError.completed`, and
keeps the entry pending for retry. No partial name is returned. Other errors retain
the Win32 code; automatic cleanup is best effort and emits no output.

Open validates nonempty UTF-8 paths of at most 4095 bytes, rejects embedded NUL,
wildcards and device namespaces, and resolves an absolute path within a 4096-wide-
character buffer including the search suffix. It rejects observed reparse points
in the root and its directory prefixes. UNC shares and normal drive roots use the
same bounded policy. Concurrent changes can race between attribute observation
and enumeration; this is not a snapshot or an adversarial containment guarantee.

## Validation

The Windows native lifecycle fixture checks UTF-8 names, logical sizes, ordinary
directories, skipped dot entries, short-buffer retry, empty/repeated EOF, explicit
close, early-return cleanup, and missing/file/invalid paths in low and optimized
profiles. Child reparse classification and root/prefix symbolic-link rejection
are exercised when the host permits creating a link; an unavailable privilege
is reported separately. Component tests
check Windows-only admission and absence from unrelated File linkage.

## Consequences

The SDK version, source digest and dependent locks change. Language revision and
runtime ABI remain unchanged. Linux linkage, deterministic ordering, recursive
walks, race-free handle-relative access, and high-level directory collections are
not added. The analyzer implements its own recursion, grouping and sorting.
