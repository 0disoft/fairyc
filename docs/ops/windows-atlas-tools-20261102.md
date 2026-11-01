# Windows bounded Atlas tools receipt, 2026-11-02

- Impact source: `33e06c0e`; MCP source: `ec3ab9f9`, negotiation/ID fix: `fab77332`.
- Structural edit and final protocol/output gates: `567e4db4`.
- Owner: [ADR 0499](../adr/0499-bounded-atlas-impact-mcp-and-body-edits.md) and
  [the consumer contract](../atlas/impact-mcp-edit-v1.md).
- Host: Windows amd64; Go 1.27.1; MSVC 14.51.36231, native low/optimized.

## Executed and passed

| Slice | Windows evidence |
| --- | --- |
| Reverse impact | Complete exact-current quickstart join reaches the cross-package caller. Visible/full selectors yield identical results. Depth/node cutoffs return partial traversal, controlled unknown map coverage stays partial, stale expectations reject, and old locks cannot produce current CLI results. The subject/context identities remain present; no host path or source body leaks. |
| Read-only MCP | Initialize/ready/list/call, version negotiation, ping, pre-initialization rejection, explicit capability admission, unknown write tools, extra/root arguments, duplicate keys, frame/message bounds, oversized IDs and current-lock failures pass. The CLI returns output failure as code 4. No source write tool is advertised. |
| Actual stdio process | A Go-built Windows compiler is launched by an independent Python subprocess client. Initialize, initialized, tools/list and ping produce exactly three valid JSON-RPC responses and no stderr. Only atlas_impact is advertised with read-only annotations. No persistent server or client configuration is installed. |
| Structural preview | Exact current private helper/body selection, raw before/after hashes, AST body range and verified status pass. Preview preserves source and graph lock. Type errors, invalid syntax and early body closing/sibling injection fail without changing the file. |
| Structural application | The one-file replacement preserves every byte outside the intended body and leaves fairy.lock unchanged. Old locked compilation then fails until ordinary check. A changed dependency between planning and applying rejects; nonportable/escape/link inputs reject. The Windows symlink control runs without a skip. |
| Edited native programs | After application and ordinary check, two actual MSVC executions, development and release, return the expected changed result 43 with empty streams. No source behavior-equivalence claim is made. |
| Related checks | Full atlasmap and atlasmcp package tests, focused CLI/help tests, scoped vet, gofmt and Git diff checks pass. Development repetitions are not additional native coverage. |

The standalone process smoke covers protocol startup/discovery/ping. Real impact tools/call is
verified in the Go CLI integration test against a current locked workspace; no external MCP SDK
or client-product compatibility matrix was run. The local process transcript is ignored output
at `.cache/atlas-mcp-process-smoke.json`, not a contract owner.

## Replay

```powershell
go test -mod=readonly ./internal/atlasmap ./internal/atlasmcp -count=1
go test -mod=readonly ./internal/cli -run '^(TestAtlas(Impact|MCP|Edit)|TestStableHelpInventoryOwnsTopLevelCommands)' -count=1 -v
go vet -mod=readonly ./internal/atlasmap ./internal/atlasmcp ./internal/sourceedit ./internal/cli
go build -mod=readonly -o .cache/fairyc-atlas-tools.exe ./cmd/fairyc
gofmt -l internal/atlasmap/impact.go internal/atlasmcp internal/sourceedit internal/cli/atlas_impact.go internal/cli/atlas_mcp.go internal/cli/atlas_edit.go
git diff --check
```

All listed completed commands returned 0. No focused test skipped. The report-range addition was
rechecked through preview/application tests; accepted native inputs and lowering were unchanged.
Exact validated symbol/declaration selection supplies the edit target; no relation-filter heuristic is used.

## Deferred and excluded

Native Linux remains unexecuted at maintainer request. The queue records the shared package/CLI
commands. The authored native edit helper selects GCC/Clang low/optimized on Linux, but neither
those executions nor Linux rooted rename behavior are established by this Windows receipt.
No WSL provisioning, hosted workflow dispatch, push, release or deployment occurred.

Full repository tests, broad semantic diff/points-to/type-dependency impact, inferred rename,
multi-file/public/generic refactoring, MCP writes, every optional MCP feature, arbitrary external
client compatibility, hostile concurrent writers, crash/power-loss tests and Fairy-authored
candidate parity remain outside this bounded receipt. Impact is bounded call evidence, not safe
change. Edit validation proves admitted source/MIR, not equivalent behavior. Output or lock-cleanup
failure can follow a committed source replacement; interrupted stages/locks need inspection.

The source-of-truth change is ADR 0499 and its Atlas/CLI contract, with map-consumer/index/queue
references updated. New experimental Go CLI/internal APIs are admitted. Language, SDK/runtime,
map/graph-lock schemas, DB, dependency versions, repository hygiene and CI/runner settings remain
unchanged. The existing ordinary-interface/effects diff remains a separate bounded review tool.
