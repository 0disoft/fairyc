# Windows local state execution receipt, 2026-11-02

- Source: `e8d66e67` (one-attempt execution), `495242cb` (local storage),
  `bef5a646` (experimental source developer command).
- Owner: [ADR 0500](../adr/0500-local-contract-state-execution-and-persistence.md),
  extending [the existing port](../adr/0376-contract-state-persistence-port.md)
  only for the named local prototype.
- Host: Windows amd64, Go 1.27.1. These are Go execution and filesystem checks;
  no generated-C durable runtime or on-chain execution is implied.

## Executed and passed

```powershell
go test -mod=readonly ./internal/contractstate ./internal/contractstatemir ./internal/contractstatefixture ./internal/contractstateconflictfixture ./internal/contractstateretryfixture ./tools/dev/fairyc-state -count=1
go vet -mod=readonly ./internal/contractstate ./internal/contractstatemir ./internal/contractstatefixture ./internal/contractstateconflictfixture ./internal/contractstateretryfixture ./tools/dev/fairyc-state
go build -mod=readonly -o .cache/local-state-bef5a646/fairyc-state.exe ./tools/dev/fairyc-state
git diff --check
```

All six package suites and scoped vet pass. The command suite and vet were rerun after
the final source-I/O exit classification change; the final built-executable receipt below
is from exact code `bef5a646`. Earlier development repetitions are not added to its six
executions. All changed Go files were formatted with gofmt.

| Slice | Evidence |
| --- | --- |
| Verified execution boundary | The state-MIR verifier rejects before snapshot access. A commit takes one snapshot and submits one overlay. Reject and trap submit none; a competing writer leaves exactly one snapshot/submit attempt and an explicit conflict. |
| Existing oracle compatibility | Existing typed kernel, MIR, state fixtures, overlapping conflict fixtures and caller-visible retry fixtures all pass. Managed bodies cannot seize commit/rollback; escaped transactions are closed after backend failure. Value limits reject before publication. |
| Local restart and schema | Reopen preserves values/version; deletion and append-only layout publication increment once. No-op write/layout publication preserves version. Old layout and old versions reject, including stale empty writes. Missing state never initializes a replacement. |
| File damage and bounds | Truncation, bad checksum, inconsistent layout hash, duplicate entries and oversized files reject without repair. Duplicate/deleted-with-value/oversized mutations, mutation-count excess and version exhaustion reject. |
| Failure boundary | Injected pre-rename failure leaves exact old bytes and no temporary file. Separate child processes exit abruptly immediately before publication and after successful publication. The parent observes complete old state and then complete new state, respectively. Crash-left staging is ignored, not auto-published. |
| Concurrency | A child cannot open the store while a parent holds the lock; it receives busy. Two opened stores attempting version zero produce exactly one winner; the loser receives busy or stale-version conflict. |
| Root/file policy | Windows state-file and lock-file symlink rejection controls execute and pass without a skip; external target bytes remain unchanged. Store identity is pinned through an opened directory. Trusted ancestors/ownership remain a precondition. |
| Source integration | Existing `CheckExperimentalState` parses and verifies one `.fy` source transaction. Commit/write/read succeeds; altered reject/trap bodies preserve the prior committed file bytes. An invalid storage field rejects without state mutation. Snapshot does not execute the body. |

## Standalone Windows execution

A native Go-built Windows executable runs six independent processes against one explicit
store: init, commit, snapshot, reject, trap, final snapshot. The input declares one u64
cell and writes/reads 42 before its selected exit. Return codes are `0,0,0,6,8,0`.
Both snapshots observe version 1 and one canonical u64 value 42; reject/trap preserve
the exact committed file SHA-256. These processes exercise the real source compiler,
verified-MIR executor, file lock, canonical persistence record and rooted publication.

The ignored receipt is `.cache/local-state-bef5a646/receipt.json`; it binds exact source
commit, executable hash, committed-state hash and all six JSON outputs. It is local
evidence, not a hosted-CI artifact or distributable binary.

To replay with an existing empty directory and a `transaction.fy` file:

```fairy
module transaction;
@contract_uid("example.local-state")
state { cell u64 value; }
@effects(storage_read, storage_write)
fn void transaction() {
    storage::write("value", 42);
    storage::read("value");
    storage::commit();
}
```

```powershell
go run -mod=readonly ./tools/dev/fairyc-state init .cache/my-state transaction.fy
go run -mod=readonly ./tools/dev/fairyc-state execute .cache/my-state transaction.fy
go run -mod=readonly ./tools/dev/fairyc-state snapshot .cache/my-state transaction.fy
```

The directory must already exist. Replacing `storage::commit()` with
`storage::reject()` or `storage::trap("overflow")` selects the rollback controls.
`go run` can wrap a nonzero child status as its own error; use the built executable
to inspect exact 6/8 process codes. Source and store paths are explicit; the tool does
not rewrite source, initialize a workspace or select a chain target.

## Skipped and remaining

Native Linux is deliberately deferred to the [combined queue](../development/ordered-work-20261031.md).
No WSL discovery, provisioning or Linux execution was attempted. Full `go test ./...`,
hosted CI, clean-machine transfer, generated-C persistence and chain execution were not
run. This work neither reopens old Linux baseline acceptance nor extends it to new code.

The prototype rewrites complete bounded state under an exclusive lock. Its checksummed
files detect corruption, not rollback or forgery by the directory owner. Interrupted
creation and crash-left temporary files require explicit operator inspection. Power-loss
and physical-media retention, network filesystems, authenticated recovery, reentrant
external calls, chain SDK/RPC integration, signing and deployment remain outside scope.
The source runner makes no production VM or stable runtime/ABI support claim.

Source-of-truth changed: ADR 0500 and its developer command contract. The ordered queue
and topical index link that admission and this receipt. A new private local file format
and experimental developer command were added; existing public API/SDK ABI, existing DB
schemas, dependencies, repository hygiene, CI/workflows and runner settings were not
changed. No push, deployment, publication or hosted workflow dispatch was performed.
