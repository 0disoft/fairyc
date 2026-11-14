# Git snapshot blob staging — 2026-11-14

One refactor: enforce blob admission before collecting its body. Previously
`cat-file --batch` stdout was collected up to 64.5 MiB, then individual 8 MiB
and aggregate 64 MiB limits were checked. Oversized files therefore consumed
their bodies before refusal; valid snapshots also staged all expanded bytes.

The batch is now consumed through a pipe. A 128-byte header reader admits the
object/type/canonical size before reading the body. Reuse a geometrically grown,
8 MiB-capped scratch buffer; copy the separately retained manifest (at most its
existing 1 MiB parser limit). Validate delimiters and final EOF. Rejected input
cancels the child and every started child is waited exactly once. A failed child
still fails preparation even after syntactically valid stdout EOF.

Blob staging changes from O(T) to O(M + H): total expanded payload T, largest
admitted file M, retained manifest H. Other storage, notably the final ZIP and
existing unpack/identity verification, is unchanged. This is not an O(M) claim
for the complete snapshot operation. The four production Git invocations remain
four; no per-file subprocesses were introduced.

## Measurements

Windows/amd64, Ryzen 5 7430U, identical synthetic admitted input: eight 8 MiB
zero-filled blobs, same ZIP writer. The frozen previous consumer and new
consumer each run three iterations, repeated three times:

```text
go test -mod=readonly ./internal/packagesource -run '^$' \
  -bench '^BenchmarkGitSnapshotBlobStaging$' -benchtime=3x -benchmem -count=3
before B/op: 168498362, 168501941, 168498469
after  B/op:   8763309,   8763202,   9122034
median: 168498469 -> 8763309 bytes/op (-94.8%)
```

These are cumulative allocations for the staging/ZIP benchmark, not process
RSS or a full-snapshot peak-memory measurement. Log:
`.cache/git-stream-bench-final.log`. Latency varies and is not a production claim.

## Validation

- Entire `internal/packagesource` passes (9.750 s), including actual temporary
  Git repositories, symlink/gitlink refusal and source identity checks.
- Old/new consumers emit byte-identical ZIPs. A retained manifest survives
  scratch reuse. Malformed/truncated/trailing framing and aggregate excess fail.
- Oversized blob/manifest declarations refuse after at most 128 input bytes,
  without collecting the body. A continuously writing test producer is killed
  and reaped on refusal; exit 7 after valid EOF is propagated.
- Fresh Go CLI build and seven actual CLI processes pass (2.914 s test package,
  process replay 2.78 s), including snapshot ZIP digest, existing-output
  preservation, immutable-reference refusal and offline package consumption.
- `gofmt` and `git diff --check` pass. Logs: `.cache/git-stream-full.log` and
  `.cache/git-stream-cli-final.log`.

No public API/schema, dependency, database, runner/CI or hygiene configuration
changes. The owning ADR 0560 remains unchanged. UI/input tests are excluded by
the user's standing constraint; Linux-native benchmarking was not available on
this Windows host. Only this input-memory issue was changed.
