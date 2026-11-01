# Windows Go/Fairy candidate cache parity, 2026-11-02

The later [inline completion receipt](windows-candidate-inline-20261102.md)
supersedes this record's pending inline-admission status. The results below
remain the historical encoder/no-inline baseline for their named source commits.

- Source: `3d916675` (v5 encoder and key producer), `07b5dad3` (source-derived
  key and inline-boundary regression probes).
- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231; native C11 low and optimized.
- Owner: [whole-package cache v5](../package/whole-package-cache-v5.md) and
  [ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md).
- Scope: the authored Fairy candidate's current no-inline source subset. This
  receipt does not promote the candidate to the entire Go language surface.

## Implemented and executed

The ordinary candidate encoder previously used the v4 domain and omitted owner
and dependency inline identities. It now emits the same v5 canonical bytes and
`fycache5-sha256-` digest namespace as Go. The generic envelope includes the same
base bytes. The complete source-derived producer supplies the exact zero inline
class for its admitted no-inline sources, rather than authorizing v4 reuse.

| Probe | Windows evidence |
| --- | --- |
| Independent encoder oracles | 24 ordinary and 11 generic input cases agree byte-for-byte and digest-for-digest at both profiles. Owner and dependency inline identities mutate independently. Malformed inputs, work budgets and v4 text rejection also pass. |
| Actual source-derived keys | `Plain` and `Aliases` resolve two multi-package source graphs, check candidate IR and derive ordinary and generic keys for six package instances per profile. The authored fixture compares every canonical byte against Go-produced templates, replacing only the explicit executing-image identity. The Go host independently checks both SHA-256 key digests and exact executing-PE binding in the 16,157-byte ledger. Both profiles pass. |
| Closed inline boundary | Two bounded `@inline` function bodies pass Go checking and reject through the actual Fairy signature authority at the attribute span with code 3. The ordinary function control passes. Both profiles execute successfully. This records the remaining gap instead of silently discarding inline annotations. |
| Full candidate workspace | The isolated source workspace passes `check`, then `check --locked`: two instances, 574 source files and 7,232 MIR functions. |

Commands executed successfully from the repository root:

```powershell
go test -mod=readonly ./internal/cli -run '^(TestCandidateCacheKeyV5Parity|TestCandidateInlineAdmissionBoundary|TestCandidateCompleteCacheKeysV5)$' -count=1 -timeout 4m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
go test -mod=readonly ./internal/cli -run '^TestCandidateCompleteCacheKeysV5$' -count=1 -timeout 10m -v
Remove-Item Env:FAIRYC_RUN_PARITY_INTEGRATION
go vet -mod=readonly ./internal/cli
git diff --check
```

The first command passes the encoder and inline-boundary probes and deliberately
skips the opt-in complete-key probe. The second command explicitly runs that
probe and passes in 168.71 seconds. Its low/optimized executions take about
6.21/0.145 seconds; those numbers exclude source checking and C compilation and
are not a compiler throughput comparison.

The full workspace check used the ignored isolated source copy
`.cache/parity-check-3d916675` and its freshly built `fairyc-bootstrap.exe`:

```powershell
Push-Location .cache/parity-check-3d916675
./fairyc-bootstrap.exe check
./fairyc-bootstrap.exe check --locked
Pop-Location
```

The complete-key probe uses the existing private compiler-host API, 90 C body
parts, a 1,200,098-byte shared header and an explicit 16 MiB stack reservation.
Each C invocation retains a 60-second allowance; execution has a private
two-minute allowance. A monolithic optimized C compile exceeded its allowance
during development, so this probe uses the existing partitioned emission path.
Public `run` linkage does not admit this private host-symbol closure and was not
relaxed. No product timeout, default stack or warning policy changed. The heavy
probe requires an explicit environment opt-in and adds no workflow or runner.

## Deferred and next slices

Native Linux remains in the [combined queue](../development/ordered-work-20261031.md).
The encoder and inline-boundary probes have a guarded Linux path; it has not run.
The complete-key private host replay is currently Windows-only and still needs
its Linux companion path before native Linux validation can be claimed.

Next is actual candidate `@inline` admission: derive nonzero inline classes,
preserve typed body identity, validate restored artifacts and connect backend
emission. After that, separately close the broader scalar generics, static
interfaces and copied local closures introduced by the Go prototypes. Those
language families are not established by this cache-key receipt. Remote-package,
Atlas and state developer tools retain their separately documented Go-only scope.

Full `go test ./...`, a rebuilt standalone whole-candidate CLI acceptance run,
hosted CI, clean-machine installation and Linux execution were not performed.
No push, release or deployment occurred. No public API/SDK ABI, DB, dependency,
repository hygiene, CI or runner settings changed; the cache v5 contract gained
the bounded candidate parity record, and the ordered queue/index link this receipt.
