# ADR 0310: Public Windows File read-only mutation

- Status: Accepted
- Date: 2026-08-06

## Context

ADR 0306 exposes same-handle read-only observation, but Fairy C cannot change that attribute through
the supported File owner. Path-based mutation would reopen an object and introduce a path race;
same-handle mutation keeps the identity boundary explicit. The operation must preserve unrelated
attributes and timestamps and must expose insufficient handle access exactly.

## Decision

SDK `0.0.0-stage0.44` adds:

```fairyc
std::fs::set_read_only(ref mut File, bool) -> Result<bool, FileError>
```

The private Windows adapter `fy_std_windows_file_set_readonly_v0` loads the existing handle,
rejects invalid booleans and closed storage, and queries `FileBasicInfo`. It changes only
`FILE_ATTRIBUTE_READONLY`, removes `FILE_ATTRIBUTE_NORMAL` before combining another flag, and uses
`FILE_ATTRIBUTE_NORMAL` when clearing the final remaining flag. It passes zero timestamp fields to
`SetFileInformationByHandle(FileBasicInfo)`, preserving existing timestamps. The handle must already
have `FILE_WRITE_ATTRIBUTES`; no path is reopened and access is not silently elevated.

Success returns the requested boolean. Failure preserves exact Win32 error storage and zero
completed progress. The operation does not promise that read-only metadata equals ACL, share-mode,
filesystem-policy, or later write authorization.

## Evidence

- SDK and closed-symbol tests require the public function, exact private symbol, component closure,
  and SDK `0.0.0-stage0.44`.
- The raw metadata probe toggles true then false through a write-capable handle, re-observes each
  state, preserves cursor and logical size, and covers closed, invalid, invalid-boolean, and exact
  read-only-handle `ERROR_ACCESS_DENIED` paths under low and optimized MSVC.
- The checked-in generated-Fairy quickstart performs the same true/false lifecycle, verifies zero
  cursor and size, then opens the file read-only and proves exact access denial plus unchanged
  metadata under low and optimized MSVC.

## Consequences

Fairy C gains one explicit C-friendly metadata mutation without path reopening, allocation, or a
compound permissions model. SDK identity, locks, and dependent cache keys change; language and
runtime ABI do not. General attribute editing, ACLs, ownership, mode bits, directory guarantees,
portable Unicode paths, and Linux public linkage remain separate gates.
