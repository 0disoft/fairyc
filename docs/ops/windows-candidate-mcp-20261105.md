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

## Streaming input and invocation policy

`project_mcp_stream` owns a 4 KiB fixed read window and constructs one bounded
line at a time. It preserves CRLF and final EOF handling, rejects the first
over-limit frame, and checks complete writes. Native tests leave stdin open
while reading two separate responses, proving that processing does not wait
for EOF. No owning resource field borrow or per-byte external read is required.

`project_mcp_options` requires every invocation capability and numeric limit,
rejects aliases/duplicates/noncanonical numbers, and returns the six exact
configured values. Its native acceptance and field values are compared with
the Go command, including zero-depth and all minimum/maximum boundaries.

Both Windows profiles pass:

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/cli -run '^TestCandidateMCPStream$'
go test -mod=readonly -count=1 -timeout 3m ./internal/cli -run '^TestCandidateMCPOptions$'
```

## Current-source command and full CLI

`project_mcp_cli` connects the explicit policy, bounded streaming input and exact
response writes. Each tool call freshly runs `project_atlas_source::query` against
the invocation root and its current lock. It does not retain stale source facts,
open a caller-selected root or ask a Go child to supply a map. The same dispatch
is reachable as `experimental atlas-mcp` in the full Fairy candidate.

Both Windows profiles pass the actual command tests with an empty PATH. Impact
results are compared with the independent Go workspace call index, including
source/interface/compiler identities and completeness. Tests check workspace
immutability, stale-source refusal followed by recovery after restoration,
unavailable selectors, invocation permission and limits, the message ceiling,
interactive stdin with no EOF, help and a broken stdout pipe.

Executed and passed:

```powershell
go test -mod=readonly -count=1 -timeout 10m ./internal/cli -run '^TestCandidateMCPCommand$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/mcp-cli-20261105'
go test -mod=readonly -count=1 -timeout 20m -v ./internal/cli -run '^TestCandidateAtlasCLI$'
go vet -mod=readonly ./internal/cli ./internal/atlasmcp
git diff --check
```

The focused command run completed in 209.544 seconds; the full CLI run in 494.945
seconds, with both low and optimized subtests passing. These are suite durations,
not throughput measurements. Full CLI acceptance also replays the existing Atlas
queries, unsupported-source controls and experimental help routes.

Linux GCC/Clang execution remains in the combined queue; no Windows skip counts
as Linux evidence. Full repository tests, hosted CI, clean-machine execution,
push, release and deployment were not performed. Linux experimental dispatch and
broader Atlas source families are separate implementation obligations.

The Go response-bound correction preserves ordinary IDs and emits a bounded
`-32603`/`id:null` fallback only when escaped IDs cannot fit. No public CLI flag,
MIR/schema, SDK ABI, dependency, DB, repository hygiene or runner setting changes.
