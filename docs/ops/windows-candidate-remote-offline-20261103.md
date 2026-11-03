# Windows Fairy offline package restoration, 2026-11-03

Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231. Linux remains in the batched queue;
no WSL, remote runner, push or CI dispatch was used.

## Implementation

- `f107b648`: authored raw DEFLATE decoding with stored/fixed/dynamic Huffman blocks,
  validated trees, overlapping backreferences and bounded output. No external ZIP tool,
  Go helper, compression DLL or new dependency is required by the resulting program.
- `743e8b42`: authored ZIP directory/path/type/CRC/identity validation and an owned bundle.
- `094abb1d`: private Windows offline restore entry, retained-handle filesystem traversal,
  exclusive fetch lock, rehash, staged publication and exact-tree reuse.
- The final follow-up adds small-value ZIP64 EOCD/entry metadata, descriptor CRC validation,
  bounded path-index work and failure/restart/junction/multiple-package regressions.

`project_remote_offline::restore(workspace)` parses and binds both metadata files before
locking. Every cache use rereads and hashes the exact `<sha256>.zip`. Metadata validation
finishes before creating a vendor stage. New files are written and synced beneath retained
directory handles; a directory rename publishes without replacement. Existing packages are
compared completely, including unexpected entries, rather than overwritten or merged.

The native C additions only open/create deletion-capable directory handles. Fairy owns
source policy, limits, sequencing, CRC/DEFLATE, identity, comparison, cleanup and result counts.
Two private ABI symbols are admitted by the Go/Fairy runtime inventories. The old ensure-open
directory behavior and 4,096-row retention scan stay intact; package scans separately allow
8,192 rows. The matching runtime asset pin was regenerated from authoritative C sources:
22 sources, 151,871 bytes, SHA-256
`f6d9518aa8f81b7c783d588753116545e535b44fd8ce9ea7ceeb75504b6b18fd`.

## Executed and passed

```powershell
go test -mod=readonly -count=1 -timeout 4m ./internal/cli -run '^TestCandidateInflate$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateRemote(Archive|Offline)$'
go test -mod=readonly -count=1 ./internal/candidatehost -run '^TestPrivateWindows'
go test -mod=readonly -count=1 ./internal/packagesource
go vet -mod=readonly ./internal/cli ./internal/packagesource ./internal/candidatehost
python -B tools/package-candidate-runtime-assets.py
git diff --check
```

Go formatting is clean. Raw DEFLATE passed 30 inputs per MSVC profile in 8.234 seconds:
five Go compression levels with empty/text/binary/long-repeating payloads, plus malformed,
truncated, wrong-size and output-limit cases. The final ZIP/offline run passed in 31.855 seconds.

ZIP verification runs 23 inputs in each profile. Successful ledgers compare every file name
and content SHA-256 with Go's ZIP reader. Rejections cover traversal, absolute/backslash/device
and VCS paths, duplicate/case/type conflicts, symlinks, wrong package identity/language/version,
nested workspace, payload/descriptor CRC and the per-file size bound. Legal ZIP64 and ignored
empty directories succeed.

Offline verification runs 26 cases in each profile. Twenty-five compare acceptance and the
entire resulting tree against Go `Fetcher.Fetch(..., true)`; the deterministic stale-stage
collision is a candidate-only preservation control because Go uses random stage names.
The cases cover stored/deflated/ZIP64 archives, exact reuse, extra empty directories,
missing/corrupt/wrong-case caches, warm-cache corruption, missing cache directories,
actual NTFS member/cache junctions, differing/extra/incomplete existing files, destination
files, held locks, member case, malformed source archives and identity mismatch.

A late Windows filename failure occurs after earlier staged writes; both implementations
remove the owned stage/lock and leave the same resulting tree. Two-package success and a
second-package cache failure confirm bounded aggregate accounting and preservation of the
first complete package. Every successful case is restored again by a fresh native process
and reports reuse with zero installations. HTTP spies on the Go oracle receive zero calls;
the authored import/foreign-symbol closure has no HTTP transport. Source pins, root manifests,
graph locks, cached archives and junction targets are preserved. Ordinary error reports have
zero cleanup failures. Snapshots treat junction/special entries as metadata without following
or attempting to read their contents.

## Limits and pending work

This is private API execution, not acceptance of a Fairy `experimental fetch-packages` command.
Candidate command dispatch, online transport, distribution/clean-machine acceptance and native
Linux adapter selection/execution remain pending. Pure archive modules are portable source;
the retained-handle restoration facade currently selects Windows. Linux tests were not run.

The existing 16 MiB/archive, 8 MiB/file, 64 MiB compressed/expanded operation, 4,096 file,
8,192 path-entry and 32-component bounds remain. The private implementation additionally
limits directory scans to 8,192 rows and path-index work to 268,435,456 units. Its two-minute
deadline is cooperative around entries/packages/publication; it does not cancel native I/O
or interrupt one decoder entry. Arbitrary ZIP-tool extensions/layouts and every possible
Go ZIP compatibility spelling are not certified by this fixture set.

Stale digest-derived stage names require inspection; they are never deleted by a new attempt.
The fetch lock coordinates cooperating writers. Hostile concurrent workspace replacement,
physical-media durability and global rollback are outside the existing source contract.
No real 120-second deadline wait, power-loss, process-kill or cleanup-error injection was run.

Public CLI/SDK API, DB, dependency versions, runner configuration and repository hygiene are
unchanged. Private host ABI/inventory and the runtime source-pack pin changed as described above.
Contract/index/ordered-work documents record this slice; no target support is promoted.
