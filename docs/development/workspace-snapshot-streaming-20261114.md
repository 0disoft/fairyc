# Workspace snapshot streaming comparison — 2026-11-14

One refactor: remove whole-file temporary byte slices from unchanged-source
freshness checks. `FileSnapshot.Matches` is called by language-server analysis
reuse and Atlas MCP workspace reuse. Previously every non-overlay source was
read with `io.ReadAll` before `bytes.Equal`, allocating source-sized slices on
each check even when analysis could be reused.

Use one lazily allocated 32 KiB buffer per Matches call, reused across all files
and packages. It is not stored on the reusable snapshot, so concurrent checks
do not share mutable scratch. Overlay comparison and fingerprints are unchanged.
Opened-file identity/kind/size checks now have one private owner shared by the
existing read API and the new comparer. Disk bytes are still compared exactly;
size/timestamps are never substituted for freshness.

Continue reading after a mismatch to preserve later I/O/size error precedence.
Read at most the original per-file guard plus one overflow byte; check
cancellation between reads. Extra empty files, deletion/truncation and changes
beyond the first buffer still invalidate the snapshot.

## Allocation evidence

Benchmark the real Matches method with eight immutable 8 MiB source inputs
(64 MiB total), no overlays, absent graph lock. Workspace/fixture creation and
parsing are excluded. The sources use module/comment text; AST construction is
not part of a freshness check. Windows/amd64, Ryzen 5 7430U:

```text
go test -mod=readonly ./internal/workspaceloader -run '^$' \
  -bench '^BenchmarkFileSnapshotUnchanged64MiB$' -benchtime=2x -benchmem -count=3
before B/op: 136613832, 136615352, 136615344
after  B/op:    100260,    132952,    100060
median: 136615344 -> 100260 bytes/op (-99.9266%)
```

The before run uses a Go build overlay restoring `file_snapshot.go` from
`50d40fa7`, with the same benchmark and unchanged opened-file checks. Its
generated local mapping is `.cache/snapshot_before_overlay.json`. Logs:
`.cache/snapshot-stream-before-final.log`, `.cache/snapshot-stream-after-final.log`.

This measures cumulative allocations for Matches, not peak process RSS or an
end-to-end language-server speedup. Read-byte work remains O(T). The additional
comparison byte buffer changes from O(M), largest file M, to a fixed 32 KiB;
cumulative source-copy allocation O(T) is removed. Existing retained sources,
directory/manifest metadata, overlay hashing and lock handling remain unchanged.
Read-call counts can differ with fixed buffering; no I/O-count reduction claimed.

## Validation

- Focused snapshot/stream tests pass: tail mutation beyond two buffers with
  preserved size/mtime, unchanged sources, truncate/delete/add-empty, EOF,
  short/long contents, max+1 guard, cancellation and I/O errors after mismatch.
- Full workspaceloader: pass, 17.313 s. Full languageserver: pass, 23.952 s,
  including unchanged-analysis reuse and invalidation of modified disk inputs.
- Atlas MCP CLI/protocol tests: pass, 19.721 s, including preparation/index
  reuse, source/manifest/lock changes and recovery between protocol requests.
- `gofmt` and scoped `git diff --check`: pass.
- Logs: `.cache/snapshot-stream-tests.log`, `.cache/snapshot-stream-owner-tests.log`,
  `.cache/snapshot-stream-atlas-tests.log`.

No public API, language/SDK/schema, DB, dependency, runner/CI or hygiene changes.
No owning normative contract changed; this is an implementation/evidence receipt.
No screen/keyboard operations. Linux-native measurements were not performed on
this Windows host. Unrelated documentation-site work is preserved separately.
