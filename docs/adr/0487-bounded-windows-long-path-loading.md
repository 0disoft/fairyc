# ADR 0487: Bounded Windows long-path normalization for file open and directory loading

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [Runtime standard library v0](../runtime/runtime-stdlib-v0.md)
- Extends: [ADR 0407](0407-windows-file-utf8-paths.md) UTF-8 file paths and
  [ADR 0410](0410-public-windows-directory-enumeration.md) directory enumeration

## Context

A long source filename made the candidate package loader fail even though Go read the same file.
Windows can enable long paths through an application manifest together with the registry setting;
this compiler path does not opt in, so it uses internal extended-length prefixes for normalized
absolute paths of 248 WCHARs or more. File open already held the decoded path in a stack
`WCHAR[4096]`; directory open called `GetFullPathNameW` without extending a long result.

## Decision

File open, including create through the open options, and `read_dir_utf8` share one stack-only
normalization helper in `internal/cstdlib/stage0_windows_path.inc`. The owning entry point still
validates and strictly decodes UTF-8 first. The helper then calls `GetFullPathNameW`. Normalized
absolute paths shorter than 248 WCHARs keep ordinary Win32 handling; longer drive paths use
`\\?\` and longer UNC paths use `\\?\UNC\`. The result must fit in 4096 WCHARs. Normalized or
expanded output beyond that buffer returns status 6 with `ERROR_FILENAME_EXCED_RANGE`. Existing
input-length validation is unchanged: `open_utf8` rejects a 4,096-byte input with status 2 before
decoding, while `read_dir_utf8` returns status 6 above 4,095 bytes. Normalization adds no heap
allocation and no external dependency.

File open keeps its existing explicit namespace handling. File remove, rename and other path
mutation APIs are unchanged and make no long-path claim. Directory enumeration still rejects
user-supplied device or extended namespace, wildcard, NUL, malformed UTF-8, non-drive colon and
reparse-point directory prefixes. It checks space for the `\*` suffix after expansion.
`create_dir_all` keeps its existing lexical validation.

## Evidence

`TestWindowsLongPathNative` passed on Windows AMD64 with MSVC `/Od` and `/O2`. It created and
read a file through a 346-byte UTF-8 path with a Korean filename, covered relative, absolute, dot,
parent and slash variants, enumerated the directory, rejected a junction ancestor, denied device
and extended namespace, wildcard, NUL and invalid UTF-8 inputs, and checked UNC lexical conversion
and an expanded-output over-cap failure.

The [candidate evidence](../ops/evidence/candidate-windows-long-path-20261023.json) records 234 exact
runtime C byte/admission comparisons through Go-native and development/release candidate test
binaries. The completed trap run passes all 24 comparisons: 12 cases with the development command
compiler in development and release program profiles. `long-source-key` loads and emits only the trap
category because its source key exceeds 256 bytes. Dependency and replaced-source cases retain
locations from compiled snapshots.

No live UNC share was tested, and Linux behavior is unchanged. A whole-CLI build from the latest
clean source, release command-compiler coverage, full CLI parity and fixed-point refresh remain pending.

## References

- Microsoft, [Maximum path length limitation](https://learn.microsoft.com/en-us/windows/win32/fileio/maximum-file-path-limitation)
- Microsoft, [`GetFullPathNameW`](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfullpathnamew)
