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

## Bounded ordered-table validation prerequisite

SDK source preparation exposed repeated linear declaration lookup and pairwise
symbol duplicate checks. `project_atlas_model::find_key` now searches only tables
whose strict canonical key order was already validated. All callers use either
the validated declaration table or the fully validated function table. Malformed
and unsorted tables still reject before these lookups. `key_position` returns a
table-relative slot and an explicit missing/exhausted sentinel.

Symbol validation retains full-ID order and checks visible-prefix collisions
against the adjacent full-ID row, after digest matching proves the prefix rule.
A bounded local bitmap detects repeated declaration ownership. No borrowed input
is mutated. For D declarations, F functions and C local calls, these table
lookups change from O((D+F+C)D) to O((D+F+C) log D); symbol duplicate checking
changes from D(D-1)/2 comparisons to D bitmap operations and adjacent checks,
using O(D) extra space within the existing 4,096-row bound.

```powershell
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestCandidateAtlas(OrderedKeys|Validation|GenericValidation)$'
```

The combined Windows command passes in 41.644s. Both low and optimized profiles
pass existing ordinary/generic/aggregate invalid-map controls and the new lookup
boundaries: 0/1/2/257 rows, first/middle/last, missing keys, and exhausted work.
For the supplied complete four-part keys, 257 rows require at most nine
comparisons and 3,150 charged units. Existing source/map/depth/work ceilings and
wire schemas remain unchanged. This establishes the lookup prerequisite, not
full SDK command admission or complete CLI acceptance.
