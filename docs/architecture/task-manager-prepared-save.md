# Task-manager prepared-save evidence

2026-10-01. The application migration is implemented in
[task_manager_cli](../../examples/task_manager_cli/README.md), version 0.1.1.
It consumes the existing SDK stage0.72 public `replace_file` operation under
[ADR 0411](../adr/0411-public-windows-file-replacement.md). There is no compiler,
language, SDK or runtime ABI change in this migration.

The authoritative file is the live FYTASK1 TSV. The application validates its
header/version, bounded total length, canonical ordered IDs, statuses, UTF-8
titles and record terminators before accepting it. It prepares all new data and
the response in memory. An exclusive cooperative `.lock` owns the exclusive
sibling `.tmp` slot. Complete write count, `sync` and consuming `close` must all
succeed before one replacement of the live name. Success acknowledgement follows
lock removal. A failed preparation never attempts publication. Replacement
failure preserves live and temp; post-publication interruption may leave a valid
new live file and stale lock without a success acknowledgement.

## Validation

- Preserved regression corpus: **118/118** exact CLI output and stored-byte cases,
  59 each in development and release. Fresh product EXEs were built once per
  profile using compiler revision `0bafe327`, SHA256
  `d0ebc6dc9f807fc2118f816214b906bc22fa7f26759891c5a6adfcb5d7850648`.
- Only help and the replacement failure expectation changed. `MoveFileExW` returned
  Windows access denied `5` for the incompatible share handle on this host,
  whereas the old no-replace rename oracle expected sharing violation `32`.
  The first regression attempt stopped at that assertion. After correcting the
  expected diagnostic, the new development image's compiler/source/hash receipt
  was checked and reused; release was built once. No historical EXE was reused.
- **20/20 native storage scenarios**, ten each under low and optimized MSVC:
  success; first-write disk full; two-byte partial write followed by disk full;
  sync error; consuming-close error; replacement disk full; forced termination
  during partial writing; after data sync; immediately before replacement;
  immediately after successful replacement.
- Every scenario asserts live bytes, temp contents/existence, absent `.bak`,
  lock residue, exit status, stdout and stderr. Each restarts twice, asserts no
  orphan promotion/deletion, and checks normal reading after explicit operator
  cleanup in the test-owned directory. The post-replacement kill retains the
  complete new TSV; all earlier kills retain the exact old TSV.
- The fault test compiles the exact Fairy modules to verified MIR and C through
  the existing compiler. Test-only Win32 wrappers inject errors or signal a
  controlled checkpoint before a real child kill. Production file I/O and all
  Fairy validation/save/cleanup/response logic run. C fixtures do not implement
  replacement application logic. Fault selectors exist only in instrumented
  test images, never the public SDK or shipped application.
- The nine historical downloaded projects were compared without following
  reparse links: **678 file/link observations, zero changes**. This snapshot is
  narrower than the earlier round-two 692-observation inventory; counts refer
  to their respective recorded inventories, not interchangeable coverage.

## Evidence

Local artifacts are preserved under the downloaded workload root at
`_revalidation/task-save-0bafe327/`. The application subdirectory contains Fairy
sources, its regenerated SDK lock, fixtures, raw execution records, transcript,
source hashes and new EXEs. `fault-receipt.json` contains per-case bytes/diagnostics,
generated-C and production/instrumented-support hashes, and separate test EXE
hashes. `original-before.json` and `original-integrity.json` record preservation.

Product EXE SHA256:

| Profile | SHA256 |
| --- | --- |
| development | `4c56b8046306b958eb10cbed466acd1d6616112feb1abd869fbfcd3313f06500` |
| release | `6560b57c7aa2e68ebb96f9346b2b29b2fce9d76032bff147aea204b8ca2f6030` |

## Remaining boundary

The storage target is the current local Windows AMD64/MSVC filesystem, with a
trusted directory and cooperating writers using the identical path spelling.
There is no automatic recovery, multiwriter fencing across aliases, metadata/ACL
merge, Linux replacement admission or exhaustive interruption inside the OS move.
After a stale lock, an operator must establish that all owners exited, preserve
copies, validate the live file and quarantine reviewed orphans before resuming.

Disk-full, sync and close errors are synthetic Win32 return values against real
files; no host disk was filled. Real quota/full-volume tests, hardware torn writes,
controller cache loss, physical power-cut/reboot tests and stronger post-power-loss
recovery were not run. FYTASK1 has no checksum, retained validated previous
generation or recovery journal. A valid-looking corruption can escape its format
checks. Those design and lab steps remain open; `sync` and write-through success
are not a power-loss durability guarantee. This commit does not declare that
remaining capability complete.
