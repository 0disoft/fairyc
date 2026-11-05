# Windows Go SDK/user aggregate source composition, 2026-11-05

## Scope

SDK Vec previously selected a source policy that rejected every client generic
aggregate header with E4501, including the compiler-private Go workspace path.
Core SDK plus a user scalar struct already passed. The explicit
`admission.Policy.ScalarAggregates` opt-in now composes existing scalar user
struct/resource admission with authenticated SDK templates. Only
`workspacecompiler.CheckGeneric` selects it; stable checks and generic cache
profiles retain their previous admission.

The flag requires SDK-template capabilities and a valid explicit storage-owner
scope. Only the current source-authenticated SDK owns storage operations. Client
generic functions keep their existing scalar constraints; newly admitted client
aggregate parameters are scalar-constrained too. Direct nominal aggregate
arguments fail with E4502 at the written request rather than escaping as a
planner error. Forwarded requests remain constrained by the planner.

## Windows evidence

`TestAtlasSDKUserAggregateOracle` covers core SDK/struct, SDK Vec/struct and
SDK Vec/resource. Each v3 preparation retains both SDK and user instance facts,
concrete fields and nonempty resource cleanup, preserves canonical wire bytes
through decode/re-encode and passes the independent current workspace join.
Each family also requires stable-source rejection, a root-client storage
intrinsic rejection at main.fy and a nominal aggregate argument rejection.
These are three positive source families, three stable refusals and six targeted
negative requests. They do not execute generated Fairy code.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 4m -v ./internal/cli -run '^TestAtlasSDKUserAggregateOracle$'
go test -mod=readonly -count=1 -timeout 5m ./internal/admission ./internal/genericsource
go test -mod=readonly -count=1 -timeout 6m ./internal/compiler -run '^Test(Policy|GenericAggregatePackage|GenericSignatureAdmission)'
go test -mod=readonly -count=1 -timeout 6m ./internal/workspacecompiler -run '^Test(SDKCapabilitySessionReauthenticatesBeforeLowering|GenericAggregateCompilerCacheAndStableAdmission|GenericPersistentSessionBoundsAndPolicy)$'
go vet -mod=readonly ./internal/admission ./internal/genericsource ./internal/compiler ./internal/workspacecompiler ./internal/cli
```

The source/map/join command passes in 2.060s. Admission and complete generic-source
packages pass in 0.105s/0.688s; focused compiler/workspace regressions pass in
1.144s/3.078s. Scoped vet, Go formatting and diff checks pass. Cache results only
attest unchanged default profiles and ownership, not a new mixed-source cache hit.
Logs use `.cache/atlas-sdk-aggregate-{oracle,policy,compiler,workspace}-20261105.log`.

Development failures remain separate: the initial failure-diagnostic helper used
a policy-free compiler and obscured the actual E4501; the corrected helper uses
the authenticated workspace owner. A first nominal negative exposed a raw
planner constraint error and prompted the explicit direct-argument guard. An
empty cleanup body has no executable Go cleanup summary; the final resource
fixture uses a real assertion body rather than claiming an absent summary.

Fairy Complete/producer/current/join, impact/MCP/edit and full authored-main
acceptance remain next steps in the composition plan. Native Linux stays batched.
Final comparisons, consecutive self-build, distribution and hosted CI stay open.
No grammar, public CLI shape, SDK/runtime ABI, map schema, DB, dependency,
product timeout, target, repository hygiene or CI runner change. No push or dispatch.
