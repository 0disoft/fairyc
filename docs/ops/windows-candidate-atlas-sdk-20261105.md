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

## Core SDK source producer and current/join APIs

The private producer now handles the seven authenticated ordinary SDK modules:
alloc, bytes, fs, io, num, process and text. The source probe imports all seven
and checks Buffer/Arena ownership, numeric parsing and SDK calls. Independent
Go v3 partitions validate all produced declarations, visible/full SymbolIds,
ordinary/generic interface hashes and ordered calls in all three packages.
The Go decoder accepts every emitted partition. Native execution here runs the
Fairy compiler/projection probe; it is not an execution claim about its target
program or the complete authored CLI.

Two preparation defects are fixed:

- Generic interface hashing gets its existing 1 MiB operation allowance, then
  charges actual work into the enclosing 16 MiB source/projection ledger on
  success or failure. Prior source work no longer incorrectly exhausts the hash.
- Declaration identity parsing retains only the exact current package UID and
  nominal coordinate references consumed by SymbolId derivation. Full lock,
  source-file and interface expectations still enter current-map admission;
  unrelated file metadata is no longer copied and parsed once per declaration.

Current admission retains the validated declaration-to-symbol token table.
Cross-package joins search canonical declarations and function summaries in
O(log D) and O(log F), with missing rows distinct from exhausted work. The
declaration table replaces the earlier duplicate bitmap with bounded token slots;
it retains O(D) storage and the existing 4,096-row ceiling. Cross-tree lookups
compare all four key components, never JSON-local token ordinals.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run '^TestCandidateAtlasSDKProduction$'
go test -mod=readonly -count=1 -timeout 15m ./internal/cli -run '^TestCandidateAtlas(OrderedKeys|Validation|GenericValidation|GenericJoin|Impact)$'
```

Both commands pass on the final implementation: SDK APIs 318.186s (low core
probe 105.34s, optimized core probe 3.98s), lookup/invalid-map/join/impact
regressions 272.804s. Each profile also verifies exact Go hash output and equal
measured operation cost with the enclosing counter at 1 MiB-1, 1 MiB and
1 MiB+1; a full 16 MiB ledger rejects. All native probes preserve workspace
bytes. The SDK API root uses ten fresh native processes across both profiles.

Ordinary and function-generic source production also passed in 268.621s after
the hash/identity preparation fixes, before the subsequent join-index change.
That result is not substituted for the final join/impact command above. Scoped
vet, Go formatting and diff checks pass.

### Acceptance boundary and failed attempts

The SDK API probe uses one bounded 16 MiB source/projection ledger and a separate
bounded 16 MiB current-admission/join ledger. The extra map export for the Go
oracle has its own bounded artifact allowance. These explicit API operations do
not establish admission under the actual command's existing shared ledger:
`project_atlas_source::query` still refuses SDK workspaces. SDK command ledger
integration and impact/MCP/edit acceptance remain required. Vec source production
and SDK-plus-user-aggregate combinations are also not established here.

Earlier full SDK probes reproduced the false hash limit, then reached the
shared-counter ceiling (`16777215` used) while Go accepted the produced map.
They are failures, not passes. A 60-second test-process ceiling also interrupted
the low full-map export; the dedicated API probe permits three minutes per SDK
process. Product/default build timeouts and CLI policy were not changed, and
no 60-second full SDK command success is claimed. A temporary Result-binding
test preparation error was corrected before final execution. One obsolete
pre-fix native test was intentionally stopped after recording its failures;
the final complete command above is a new execution.

External API/SDK ABI, map wire schema, DB, dependencies, repository hygiene and
runner settings are unchanged. New lookup helpers/retained slots are compiler
private. Linux, full current CLI, self-build, distribution and hosted acceptance
remain outstanding.
