# Private Linux candidate retention v1

`internal/candidatehost/linux_retention.c`, updated `internal/candidatehost/linux_rooted_file.c`, and
`examples/lexer_candidate/src/linux_retention.fy` define the private retention slice. This note adds
no CLI or public SDK surface.

## Generation set

Retention admits only regular files named exactly `fyproject0-<64 lowercase hex>-<64 lowercase hex>.fyc`.
The name is 144 bytes: an 11-byte prefix, two 64-byte lowercase-hex fields, one hyphen, and `.fyc`.
`linux_directory` metadata supplies the regular-file check. Every other name, including `.stage`,
partial names, and differently cased names, is ignored. Linux staging is anonymous `O_TMPFILE`, so
a `.stage` pathname is not part of this protocol.
The scan is `linux_directory.scan(root, 4096, work)`. If the directory has more than 4096 non-dot
rows, that scan fails; the sweep returns `Problem.kind=2` with `evicted=0` before any deletion. It
never selects from a truncated inventory; ignored names still consume rows and work.

## Limits, sweep, and results

`Limits` is `{ entries; bytes; record }` with entries `1..4096`, bytes `8..1_073_741_824`, record
`8..268_435_456`, and `record <= bytes`. `preserve` is empty or one exact generation name; it is
included in the observed retained count and bytes and is never an eviction candidate.
Preflight checks a preserved generation over `record` and fails with `Problem.kind=7`,
`evicted=0` before mutation. It builds one index vector of regular generation files, counts only
non-oversize records in `retained`/`size` (at most 4096 × 256 MiB, fitting amd64 `usize`), and heapsorts the
indices by `(modified_seconds, modified_nanos, name bytes, scan index)` in O(n log n) time and O(n)
space. Oversize removals are not added to the retained totals; they are counted in `evicted`.

The shared logical `work` counter is capped at 16,777,216. The sweep charges `name length + 1` per
scanned row, `name length + 41` before an erase, one unit per comparison, and one unit per compared
name byte, all before that operation. Crossing the cap returns `Problem.kind=5`; before any delete
`evicted=0`, and after previous successful evictions it reports their actual count. A work failure
never rolls back an earlier deletion.
One sorted pass drives eviction. Every eligible oversize record is erased regardless of the current
bounds; ordinary records are erased only while non-oversize `retained`/`size` exceed `entries` or
`bytes`. `preserve` is skipped. The pass stops at the first stale or missing snapshot or erase
failure; stale means device, inode, size, mtime, or mode changed, or the name vanished. No deletion
is retried.

`sweep` returns `Result<Stats, Problem>`: `Stats { entries; bytes; evicted }`, `Problem { kind;
os_code; close_code; evicted }`. `Problem.kind` is 1 input, 2 scan (including >4096 rows), 3
erase/close, 4 stale snapshot, 5 work, 6 allocation, or 7 protected oversize or unprovable limits.
`erase_if` returns `Result<bool, EraseProblem>`: `EraseProblem { kind; os_code; close_code; removed }`.
`kind` is 1 input, 2 allocation, 3 native, 4 close, or 5 ABI. `removed` is true when the mutation was
applied. A low-level close error (`kind=4`) therefore retains `removed=true` when the unlink already
succeeded; `sweep` maps that to `Problem.kind=3` with `os_code=0`, nonzero `close_code`, and counts
the applied mutation in `evicted`. A stale or missing candidate is `Problem.kind=4`; a native erase
failure is kind 3 and carries the close code. Preflight/inventory failures before deletion report
`evicted=0`; a failure after deletions reports the actual completed prefix, with no rollback.

## Native conditional erase

```c
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root, const uint8_t *leaf,
        size_t size, const void *expected, uint32_t *removed, uint32_t *close_code);
```

`expected` is 40-byte `rooted::Metadata` (device, identity, size, modified seconds, modified nanos,
mode). The adapter requires non-null outputs, a valid leaf, regular expected mode, nonzero identity,
and nanos below 1,000,000,000; it zeroes `removed` and `close_code` first.
The erase takes the same nonblocking directory lock as the updated `publish_new`: a fresh
`openat(root, ".", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC)` open description, then
`flock(..., LOCK_EX | LOCK_NB)`. Each call has an independent open description even with the same
retained Directory. There is no lock file or waiting, the expected local Linux filesystem is required,
and there is no unsafe fallback. Lock failure returns its errno with `removed == 0`; the lock
descriptor is closed once, and Linux `EINTR` releases it, so close is not retried.

Under the lock, `fstatat(lock_fd, leaf, &actual, AT_SYMLINK_NOFOLLOW)` retries only `EINTR`.
`ENOENT` returns success with `removed == 0`; other stat errors return errno. The actual object must
be regular with `st_nlink == 1` and nonnegative size; actual nanos must be in range, and device,
inode, size, `st_mtim.tv_sec`, `st_mtim.tv_nsec`, and mode must equal `expected` exactly. A mismatch
returns success with `removed == 0`.
Only an exact match reaches `unlinkat(lock_fd, leaf, 0)` once with flags 0. `EINTR` and every other
failure return errno with `removed == 0`; success sets `removed = 1`. A later `ENOENT` returns
`removed = 0` and errno 0. `close_code` records a close failure whether the main operation succeeded
or failed. A successful unlink with a close error returns main code 0, `removed == 1`, and nonzero
`close_code`; the mutation is not undone.
`publish_new` uses the same lock around its exclusive link. If the link succeeds and lock close
fails, publication still returns success so the Fairy wrapper freezes the staged file. Lock failure
returns before any link.

## Guarantee boundary

The directory lock is `flock(2)` advice, not an atomic compare-inode-unlink primitive. It serializes
cooperating publishers and sweeps using this adapter. A noncooperating client can replace the name
between `fstatat` and `unlinkat`, so the unlink may remove a different object. Same-user arbitrary
edits, old publisher versions, and in-place writes through another hard link are outside the
guarantee. `st_nlink == 1` refuses hard-linked generations; such a candidate stops the sweep.
The sweep is not an instantaneous global bound. It scans first and locks each delete independently,
so count and bytes are the observed scan set and can change between steps. Completed deletions are
not rolled back when a later step fails. The result reports the evictions actually completed,
including a close error after a successful mutation.
There is no directory `fsync` or physical power-loss guarantee. The C layer does not sync the parent
after unlink. The sweep is not wired into a CLI, and no public SDK contract changes.

## Verification

`TestLinuxRetentionPolicy` passes both-target emission and Windows low/optimized: oldest by seconds,
nanos, and name; oversize; preserved names; unknown, symlink, and `.stage` names; busy/stale; partial
evictions; close error after mutation; exact cumulative work and one-unit-short; a 4096-row heapsort
with 4095 evictions and every victim order; and duplicate preserved names that cannot false-pass
limits. `TestLinuxRetentionAdapterBoundaries` passes Windows low/optimized with POSIX substitutes: changed device, inode,
size, mtime seconds, mtime nanos, and mode; symlink/hardlink rejection; busy lock; unlink `EINTR`
with no retry; cleanup close error; and successful publish despite lock close error.
`TestLinuxRetentionAdapterNative` is authored for GNU/Linux and skips on Windows. Final candidate
evidence is complete: check and `check --locked` pass with 546 sources and 6983 MIR; all related C
and Fairy rooted/directory/retention low and optimized tests pass; whole `go vet ./...`,
diff-check, and format pass. No native Linux execution is claimed.

References: [flock(2)](https://man7.org/linux/man-pages/man2/flock.2.html) and
[unlink(2)](https://man7.org/linux/man-pages/man2/unlink.2.html).
