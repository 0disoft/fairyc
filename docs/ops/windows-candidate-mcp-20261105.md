# Windows Fairy MCP implementation, 2026-11-05

## Protocol core

The authored `project_mcp_protocol`, `project_mcp_json` and
`project_mcp_messages` modules provide owned per-frame parsing and pure session
transitions. A transition cannot open a workspace or execute a tool. Pending
queries carry copied UID/symbol/ID bytes, and the host finishes that selection
before the next transition. Only `atlas_impact` is advertised and selectable.

The existing Go server supplies an independent JSON response oracle. The native
test-only host supplies a fixed document; this is protocol unit evidence, not a
current-workspace query or an integrated Fairy CLI acceptance claim.

Windows amd64/MSVC C11 low and optimized executions pass lifecycle/ready-phase,
notifications, repeated initialization, version negotiation, exact tool arguments,
capability escalation rejection, missing selectors, malformed/trailing JSON,
duplicate decoded keys, nesting, integer/string/null ID boundaries, HTML escaping,
invalid host documents and 65,535/65,536/65,537-byte frame boundaries. CRLF and
EOF without a final newline are covered by the test host. Every response is
compared as JSON with the Go server; payloads obey the configured output bound.

Executed and passed:

```powershell
go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateMCPProtocol$'
go test -mod=readonly -count=1 ./internal/atlasmcp
go vet -mod=readonly ./internal/cli ./internal/atlasmcp
git diff --check
```

The final native protocol run completed in 11.607 seconds. This is a test duration,
not compiler or application throughput. Parser work retains the shared bounded
JSON reader's resource policy; full source queries and the real streaming stdio
host are not established by this protocol fixture.

## Remaining acceptance

Connect explicit command limits, actual streaming stdin/stdout, fresh locked
source verification and full Fairy CLI dispatch. Then replay unavailable/stale
workspaces, continued requests, message/output limits and stream failures through
the real host. Linux GCC/Clang execution remains in the combined queue; no Windows
skip counts as Linux evidence. Full repository tests, hosted CI, clean-machine
execution, push, release and deployment were not performed.

The Go response-bound correction preserves ordinary IDs and emits a bounded
`-32603`/`id:null` fallback only when escaped IDs cannot fit. No public CLI flag,
MIR/schema, SDK ABI, dependency, DB, repository hygiene or runner setting changes.
