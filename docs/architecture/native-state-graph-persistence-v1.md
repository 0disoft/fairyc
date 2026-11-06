# Native state graph persistence v1

Status: compiler-private one-attempt graph execution and CAS persistence
implemented on Windows. This completes the existing graph plan's native storage
step. The later [source/developer step](source-state-graph-v1.md) adds explicit
source graph admission and native developer selection for five source codecs.
Source helpers and root borrow/view/copy are now connected under ADRs 0504/0505.
Dynamic byte/text schemas, Complete/artifact/cache/ordinary CLI and final
acceptance remain required.
Native Linux remains batched. No push, dispatch, release or deployment.

Owners: [graph execution](native-state-graph-v1.md),
[graph transport](native-state-graph-transport-v1.md),
[native integration](native-state-persistence-v1.md), and
[local persistence ADR 0500](../adr/0500-local-contract-state-execution-and-persistence.md).
The original literal overlay and single-cell executors retain their own sealed
admission and unchanged entry points.

## One attempted transaction

`ExecuteGraphPersistent` requires an opaque built `GraphExecutable`. Invalid
owners, nil context/storage/execution and invalid limits refuse before storage
observation. One call then performs exactly:

1. One expected-layout snapshot, copied into the graph request under its full
   layout/body/program/version/limits/snapshot identity.
2. One context-bound C process execution. Nonzero process exit, stderr, trap
   channel, malformed/foreign witness, runtime rejection and cancellation fail
   before publication. The graph decoder supplies the actual received bytes.
3. For completed commit only, one atomic layout/version `CompareAndPublish` of
   the complete sorted overlay. A conflict, layout change or backend failure
   never retries the snapshot, process or CAS.

The complete path cost, successful read observations and write-byte cost remain
the C result admitted by the sealed decoder. Host execution does not interpret
MIR or generate replacement values. Reject returns its exact reads/cost without
CAS; trap propagates the existing typed `contractstatemir.Trap` panic without
publication. No proposed error/rollback overlay reaches storage.

`GraphRuntimeError` preserves the native status and unwraps existing closed,
value/count/byte/map-key and borrow-invalidation causes. Native value-arena or
framing failures are not misreported as a caller's transaction policy. Process,
snapshot and publication wrappers preserve original causes and simultaneous
cancellation. The synchronous persistence port retains its existing contract:
cancellation is checked before and after snapshot and before publication; this
does not make a synchronous provider operation itself cancellable.

## Version and acknowledgement

An empty commit still performs version/layout CAS. It preserves the version
and cannot succeed over another writer's change. Versions zero, UINT64_MAX-1 and
UINT64_MAX remain valid for no-op commits. A nonempty overlay at UINT64_MAX
refuses before publication instead of wrapping.

A successful CAS acknowledgement is retained even if cancellation arrived
during that call. Inconsistent acknowledged layout/version/write count returns
the existing `PublicationReceiptError` together with the committed outcome and
actual receipt. State may already be published; this error must be inspected
rather than automatically replayed. The local store's process-restart contract
is unchanged and does not claim physical power-loss durability.

## Windows execution evidence, 2026-11-07

The real local-store matrix executes nineteen original v0-v4 fixtures, fourteen
dynamic seven-codec graphs and an empty commit under both Windows consumer
profiles, twice each with close/reopen between attempts: 136 paired native/Go
outcomes and full snapshot comparisons. Initial snapshots contain an unmentioned
physical map entry; it survives every commit, rejection, trap, limit rollback
and reopen. Coverage includes graph branches/joins, dynamic keys/values,
transitive helpers, nested borrow/view/copy, deletion and exact costs.

Two separately opened local-store handles take the same real snapshot before
either process may publish. The C executions run concurrently. Publication is
scheduled to avoid the provider's separate nonblocking busy policy, then both
attempt the real OS-locked CAS: each profile observes one winner and one
version conflict, exactly two snapshots/CAS calls, with no native replay.
Read-only execution independently verifies a competing publication through a
second opened handle: the empty CAS refuses the stale version and preserves
the winner's bytes.

Actual C executions cover each transaction policy failure, injected backend
refusal, real local value-capacity refusal, corrupt image launch and cancellation
after snapshot/process. They preserve the pre-attempt durable bytes. Cancellation
during successful publication preserves the exact Go result and reopened state.
Binary 65,540-byte read-only commits run under a one-byte write-value policy and
retain version/file bytes across repeated reopen.

Six additional real-store executions cover 64MiB-minus-one, exact 64MiB and one
byte beyond the complete graph response in both profiles. The valid boundaries
perform one no-op CAS; the oversized witness retains no completed outcome or
publication authority. Snapshot count is one and the durable file/version is
unchanged in all six cases. Six repeated large reads reach these evidence sizes
while keeping the base64-encoded local state below its unchanged 16MiB ceiling.

Controlled-port tests cover 29 failure boundaries, all eleven runtime statuses,
typed trap/reject, exact helper cost/operation order, acknowledged cancellation,
receipt mismatch and no-op version/conflict boundaries. Every case asserts the
snapshot/process/CAS call counts rather than treating a successful callback as
proof of native execution; the separate tests above execute actual generated C.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/contractstatenative -run '^TestGraphPersistence'
go test -mod=readonly -count=1 -timeout 3m ./internal/contractstatenative -run '^(TestGraphTransport|TestOverlayPersistence(BoundaryFailures|Acknowledgement|FailuresNeverPublish|NoopVersionBoundary))'
go test -mod=readonly -count=1 -timeout 3m ./internal/contractstate ./internal/contractstatemir
go vet -mod=readonly ./internal/contractstatenative
git diff --check
```

## Attempts and limits

The first combined matrix/writer command failed because the two independent
snapshot file leases competed before either graph started. Its 136-comparison
matrix root passed, but the enclosing command is not counted as passing. The
test now schedules snapshot reads serially and holds both attempts until both
snapshots exist, preserving concurrent native execution and the real storage
CAS. The corrected writer replay passes in 1.599s.

The first ceiling fixture requested a forbidden 32MiB local limit; the unchanged
provider refused before execution. A second fixture's five repeated reads left
its base64-encoded file over 16MiB and was also refused. Six repeated reads now
produce the exact same response boundaries from a smaller valid snapshot. No
provider, transaction, capture or timeout limit was increased. The corrected
ceiling replay passes in 7.189s; failure/cancellation/read-only controls pass in
5.524s. The new and existing transport/acknowledgement controls pass in 0.841s;
complete contract-state/state-MIR regressions pass in 1.153s/0.265s. Scoped vet,
formatting and diff checks pass.

After both test controls were corrected, the complete selected graph-persistence
replay passes in 67.394s with no selected skips, including all 136 paired native/
Go comparisons and the actual store/cancellation/conflict/ceiling cases:
`.cache/native-graph-cas-final-20261107.log`. This final successful command
supersedes neither the recorded failed attempts nor the outstanding full-repo
and platform gates.

## Remaining integration

Source/developer invocation, exact current-source/artifact/cache admission and
ordinary CLI selection for the broader graphs/helpers/borrows remain required.
The earlier single-cell `experimental run-state` command does not acquire these
features by calling this internal API. Full current-source compiler comparisons,
consecutive self-build, clean-machine distribution, same-commit hosted and native
Linux acceptance remain open. Public source/CLI/API, SDK/runtime ABI, DB/provider,
dependencies, default limits, CI/runners and repository policy are unchanged.
