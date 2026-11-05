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

## Remaining boundary

Actual impact/MCP/editor commands and full authored-main registration/replay
remain separate steps. This result does not broaden single-operation language
admission, implement multi-operation interfaces or close other Atlas families.
Native Linux, full compiler comparison, consecutive self-build, distribution,
clean-machine and hosted same-commit acceptance remain open. No source grammar,
public CLI, SDK/runtime ABI, map wire, DB, dependency, timeout, target, repository
hygiene or runner changes. No push, dispatch, release or deployment.
