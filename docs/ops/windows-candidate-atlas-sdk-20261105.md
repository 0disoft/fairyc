# Windows Atlas SDK work, 2026-11-05

This receipt records prerequisites, not completion of SDK source production or
the full implementation queue. Native Linux remains batched. No push, hosted
CI dispatch, release or distribution is performed here.

## Go SDK vector oracle prerequisite

Preparing a v3 Atlas workspace using the authenticated `std::vec` reproduces
`E0101` at `@sdk/stdlib/std/vec.fy:35:56`, bytes 1415..1430: `storage_size<T>`
loses its SDK source admission. V3 had called the generic compiler directly,
instead of binding `SDKStorageOwners` from the resolved workspace.

`workspacecompiler.CheckGeneric[Context]` now shares ordinary checking's source
and SDK policy path, retaining the separate compiler-private aggregate route.
The verified compiler result preserves its MIR admission for snapshot
reverification. Public ordinary snapshots retain their zero-capability verifier;
no transported map or cache grants storage authority. The internal result field
is excluded from JSON. External CLI/API, SDK ABI, DB, dependencies, default
limits, repository hygiene and runners are unchanged.

Windows checks:

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestAtlasPrepareSDKVectorWorkspace$'
go test -mod=readonly -count=1 -timeout 10m ./internal/compiler ./internal/semanticsnapshot ./internal/workspacecompiler -run 'TestPolicy(ScalarCompatibility|StorageAdmission)$|TestAggregateSnapshotJoinsTypesCleanupCallsAndDocumentation$|TestCompilerSnapshot|TestCheckResolvedWorkspace$|Test.*Cancel'
```

Both commands pass. The first covers actual SDK workspace preparation,
v3 map codec/declaration retention and removal of the MIR admission: the latter
must reject with `storage-layout`, rather than treating `Verified` as sufficient.
The selected policy, generic/aggregate/ordinary snapshot, resolved-workspace and
cancellation checks also pass. Scoped vet, formatting and diff checks are
recorded with the local commit.

The Fairy SDK producer and actual impact/MCP/edit commands are separate work.
Full Go suite, complete current CLI replay, Linux, self-build, distribution and
same-commit hosted acceptance are not established by these checks.
