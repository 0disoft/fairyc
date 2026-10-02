# ADR 0411: Public Windows file replacement and interrupted saves

- Status: Accepted
- Date: 2026-09-30

## Context

The task-manager workload needs to publish a prepared file without a separate
rename-to-backup step that temporarily removes the live destination. Disk-full,
process termination and power loss require distinct contracts and evidence.

## Decision

SDK `0.0.0-stage0.72` adds the Windows AMD64 filesystem operation
`replace_file(ref CStr replacement, ref CStr replaced) -> Result<i32, FileError>`.
The runtime validates both paths with the existing bounded UTF-8 CStr policy,
rejects a source directory, and calls `MoveFileExW` with `REPLACE_EXISTING` and
`WRITE_THROUGH`. There is no destination deletion, backup dance, copy fallback,
cross-volume copy, implicit file-data flush, or automatic recovery. A missing
destination is allowed; success consumes the replacement name. Windows errors
retain their OS code, including disk-full and sharing violations. Existing
`rename_file` keeps its no-replace behavior. Target metadata/ACLs are not preserved
as a separate merge; the replacement file supplies the new object. Callers must
use distinct names, their own writer exclusion and trusted directories.

## Save and recovery protocol

1. Exclusively create a unique sibling temporary file on the same volume.
2. Write the complete encoded record, checking `write_all` and its completed count.
3. Check `sync` and explicit `close`; do not publish after either fails.
4. Replace the live name only after all preparation succeeds.
5. On failure retain the live file and report the error. An incomplete temporary
   file is not a committed record. Keep or remove it according to application policy.
6. On restart validate the live record format, length and checksum/version before
   accepting it. An orphan temporary file is not proof that a save committed.
   If recovery from corrupted media is required, retain a separately validated
   previous generation/journal before publication and define explicit recovery
   ordering. This primitive does not create that journal or choose a generation.

Abrupt process termination bypasses language cleanup. Under the tested local
Windows filesystem, termination during preparation or before replacement leaves
the old live bytes; termination after replacement leaves the complete new bytes.
These checkpoint tests do not prove every possible interruption inside the OS
rename operation or atomicity on every remote/filesystem provider.

`sync` requests `FlushFileBuffers`; write-through requests OS move completion.
Neither is a hardware-independent power-loss durability guarantee. Directory
metadata flush, storage-controller caches, torn writes and filesystem recovery
are outside this public contract. Physical power-cut/reboot testing is not performed.
Synthetic flush/write errors test application-visible failure handling, not actual
post-power-loss disk state. Applications needing stronger durability must implement
validated generations/journaling and test on their supported storage stack.

## Validation

Low and optimized Windows native tests cover Fairy public API linkage, replacement,
missing source, invalid/directory paths, and preserved no-replace rename semantics.
Production runtime fault probes inject first-write disk-full, partial-write disk-full
(exact completed prefix), flush failure and rename disk-full using real temporary
files. A real incompatible share handle checks failure preserves both files.
Bounded child processes are forcibly killed during writing and at pre/post-publication
checkpoints; byte-for-byte assertions check live and temporary state. Injection is
test-only: no host disk filling, mount changes or device simulation. Cross-volume
copy is disabled by flags; execution on a second volume is not claimed.

## Consequences

SDK source identity and dependent locks change. Language revision and runtime ABI
stay unchanged. Linux replacement was not admitted by this decision; ADR 0424
subsequently adds it through target-owned neutral symbols. Hardware durability, automatic orphan cleanup,
multiwriter transactions, and metadata-preserving replacement are not admitted.

Sources: [MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw),
[FlushFileBuffers](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-flushfilebuffers).
