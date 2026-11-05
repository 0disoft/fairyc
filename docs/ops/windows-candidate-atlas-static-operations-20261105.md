# Windows Fairy Atlas static-operation production, 2026-11-05

Owner: [implementation plan](../development/candidate-atlas-static-operations-20261105.md),
[ADR 0495](../adr/0495-explicit-static-operation-interfaces.md) and the existing
[Atlas contract](../atlas/impact-mcp-edit-v1.md).

## Actual producer acceptance

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasStaticOperationProduction$'
```

Windows MSVC low and optimized pass all five two-package sources in 165.744s
(low 75.00s; optimized 72.21s, including native-image preparation). The producer
uses the actual Fairy generic Complete/schema/instance/current-reader/join
owners. PATH is empty during execution; no Go child or prebuilt map supplies
facts to the Fairy compiler. The existing bounded C partition/stack/toolchain
policy is unchanged.

Five sources cover used and unused operations, selected and unused provider
body edits and an unrelated private body edit. Each profile produces ten
package maps. All twenty maps pass strict Go v3 decoding, exact source
declaration/symbol/interface and concrete implementation/call comparisons, and
independent explicit completeness checks. The existing candidate non-call
omissions remain unavailable rather than becoming invented exact facts.

Used source preserves the two cross-package algorithm calls, matching operation
implementations and exact ordinary provider endpoints. Original generic headers
remain declarations rather than synthetic executable wrappers. Unused templates
have no concrete implementations. Four additional native refusal invocations
cover exhausted work and missing coordinates, giving fourteen fresh native
processes across both profiles. Workspace bytes remain unchanged throughout.

This source family already works through the existing generic producer, so no
product source correction or new source-admission mode was introduced. The Go
oracle fixture matrix is shared verbatim with the native probe; expectations
still come independently from Go source verification and canonical v3 decoding.

Logs: `.cache/atlas-static-operation-production-20261105.log`,
`.cache/atlas-static-operation-shared-oracle-20261105.log` and
`.cache/atlas-static-operation-production-vet-20261105.log`.
The shared five-fixture Go oracle passes again in 0.450s after fixture extraction;
scoped `go vet -mod=readonly ./internal/cli`, Go formatting and diff checks pass.

## Actual impact command

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasStaticOperationCommand$'
```

Windows low/optimized pass in 175.514s (100.11s/61.58s including preparation).
The actual Fairy argument/query entry accepts visible and full selectors for
the operation, algorithm, selected i32 provider and unused bool provider. All
eight reports match independent Go subjects/nodes, exact current source/generic
interface contexts and explicit candidate omissions. The exact six-call bound
passes; five calls, one package or one output byte reject without publication.
A provider-body edit rejects the stale lock, then an independently refreshed
query retains the current implementation identities. All thirteen native
invocations per profile preserve workspace bytes. Log:
`.cache/atlas-static-operation-command-20261105.log`.
Scoped vet, Go formatting and diff checks pass; vet log:
`.cache/atlas-static-operation-command-vet-20261105.log`.

The verifier is registered as `impact-static-operations` in full authored-main
acceptance; registration alone does not attest that group. The original twenty
optimized groups and completed low groups retain their separate receipt.

## Actual read-only MCP

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateMCPStaticOperationCommand$'
```

Windows low/optimized pass in 100.776s (25.89s/62.39s including preparation).
Each real stdio process negotiates the existing protocol, lists exactly the
read-only impact tool, queries the imported operation against the independent
six-call Go workspace, rejects changed unlocked source and recovers in the same
session after explicit independent relocking. Structured source/interface/
implementation contexts match Go; queries preserve both root and provider bytes.
Protocol-only stdout, existing response/message bounds and child/pipe cleanup
assertions remain active. The existing SDK helper delegates its unchanged body
to a workspace helper, allowing the actual custom provider source without
substituting a map or Go child. Log:
`.cache/atlas-static-operation-mcp-20261105.log`.
Scoped vet, Go formatting and diff checks pass; vet log:
`.cache/atlas-static-operation-mcp-vet-20261105.log`.

The full authored-main `mcp-static-operations` verifier is registered, not replayed.

## Remaining boundary

Actual editor commands and the two new full authored-main group replays remain
separate steps. This result does not broaden single-operation language
admission, implement multi-operation interfaces or close other Atlas families.
Native Linux, full compiler comparison, consecutive self-build, distribution,
clean-machine and hosted same-commit acceptance remain open. No source grammar,
public CLI, SDK/runtime ABI, map wire, DB, dependency, timeout, target, repository
hygiene or runner changes. No push, dispatch, release or deployment.
