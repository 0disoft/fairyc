# Fairy Atlas Result-propagation collector, 2026-11-06

Part of the complete implementation queue. The initial private collection and
subsequent Complete-bound production/command scopes are recorded separately
below. Broader source/command and final acceptance remain open. Native Linux
stays batched.

## Source observation

Owner: `docs/atlas/semantic-snapshot-v0.md`. The Go prerequisite independently
fixes the source fact as the single written question byte, not the complete
operand, an explicit `.err(...)`, or the implicit return produced by lowering.

`project_atlas_return_error_sites` accepts owned IR and lowers it through the
existing verifier to a private CFG. It visits each reachable block once and
observes the Result-tag branch only after its operand completes. The branch's
source/target and expression kind/span must agree. The retained half-open span
is `[expression.end - 1, expression.end)`. Terminal operands and earlier traps
do not reach that branch. Explicit error constructors add no observation.

Each frame retains sorted, duplicate-free sites. Standalone and batch APIs keep
independent caller work/count policies, the existing maximum of 65,536, and
frame ownership. Traversal is O(B+E+S log S) time and O(B+S) auxiliary space for
B CFG blocks, E edges and S raw observations, excluding the ordinary verifier.
Invalid IR/frame/count input returns 9, exhausted work/count returns 3 and
allocation failure returns 2. No graph or supplied observed-site list grants
collection authority. Complete-owned source binding remains the producer's job.

## Windows evidence

`TestAtlasReturnErrorCollectionSources` independently admits nineteen Go sources
in 1.163s. They contain the original fourteen oracle cases plus parenthesized
operands, grouped propagation, logical conditions, concrete generic calls and
a concrete cleanup owner. The native comparator retains all ordinary and
specialized function/cleanup owners, compares exact original Go spans, checks
single/batch agreement and verifies unchanged source trees with empty PATH.

`TestCandidateAtlasReturnErrorSites` passes Windows low/optimized in 406.107s,
including 327.70s/64.02s profile subtests. Nineteen sources and twenty controls
per profile execute 78 unique native compiler processes without selected skips.
Controls cover empty/nonempty zero cap, invalid/past frames, invalid maximum,
and just-below/exact/just-above work and count limits for both APIs.
OOM itself is not injected. These are concurrent local timings, not a benchmark.

The authored unlocked and locked checks pass on 679 source files and 8,479 MIR
functions. Scoped CLI vet, formatting, UTF-8/LF/final-newline and diff checks
pass. Go map/pack/snapshot implementation inputs are unchanged; their previous
allocation-production regression results are reused, not counted as new runs.

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/cli -run '^TestAtlasReturnErrorCollectionSources$'
go test -mod=readonly -count=1 -timeout 10m -failfast -v ./internal/cli -run '^TestCandidateAtlasReturnErrorSites$'
go vet -mod=readonly ./internal/cli
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

Logs under `.cache/`: `atlas-return-error-collector-sources-20261106.log`,
`atlas-return-error-collector-native-20261106.log`,
`atlas-return-error-collector-check-20261106.log`,
`atlas-return-error-collector-locked-20261106.log` and
`atlas-return-error-collector-vet-20261106.log`.

At collector revision `5d662b0c`, only private collection was new and
`returns-error` was still unavailable in produced maps. Borrow collection/production, remaining source/command work and
all original self-build, distribution/clean-machine, same-commit hosted and
Linux gates stay open. No stable API/SDK ABI, DB, dependency, map/cache schema,
default timeout, runner or repository-policy change; no push or dispatch.

## Complete-bound production

`project_atlas_calls` now retains those original question spans in ordinary,
specialized and concrete cleanup summaries. It checks every selected frame
against the collection and writes spans in that frame's original module.
`project_atlas_produce` adds their count to the relation inventory, and
`project_atlas_header` promotes only `returns-error` to exact, removing its
unavailable omission. Existing non-call completeness counters remain zero;
borrows, effects and other uncollected classes remain explicitly unavailable.
Map schemas, checksums and current-source/join authority are unchanged.

Six independent Go source inventories freeze ordinary (two questions), operand
order (one), concrete generics (two), cleanup owner (one), branches/loop (four),
and root/dependency functions (two across the package boundary). The eighteen
version selections contain fifteen admitted combinations and three older-schema
refusals. `TestAtlasReturnErrorProductionSources` passes in 0.844s before reading
any candidate output. Function/specialization comparisons and relation counts
retain all original Go question spans. The pack oracle now retains these sites
instead of replacing their array with `[]`; only uncollected borrows are removed.

| Current production scope | Result | Seconds | Log under `.cache/` |
| --- | --- | ---: | --- |
| Full optimized source/version/current/join corpus | pass; 131 native processes, 213 current maps | 164.975 | `atlas-return-error-production-optimized-20261106.log` |
| Low six new error source families, all versions | pass; 18 native processes, 30 current maps | 156.324 | `atlas-return-error-production-low-20261106.log` |
| Six selected actual optimized CLI groups | pass | 257.117 | `atlas-return-error-cli-optimized-20261106.log` |
| Actual low six new error pack cases | pass; 21 native processes | 275.163 | `atlas-return-error-cli-low-20261106.log` |

There are no selected skips. Combined source production has 149 native processes
and 243 independent current maps; only optimized replays the entire previous
source corpus. The actual-main groups are impact-families, impact-sdk-vector,
catalog-ordinary, pack-ordinary, pack-sources and pack-sdk. Both profiles compare
all six new sources through overview/show/around and older-version refusals:
42 new actual command processes, consisting of 36 successes and six refusals.
Every source tree remains unchanged. These scopes do not attest all 41 full-main
groups or the other low groups. Timings are concurrent local observations.

The real main and dispatcher are retained, PATH is empty during commands and no
companion or Go frontend fallback performs Atlas work. Authenticated C input has
26 body units, a 2,086,644-byte header, a 153,115-byte runtime and a largest body
of 2,096,919 bytes. Product timeouts, warnings and the private stack policy stay
unchanged.

| Profile | C/profile/toolchain source digest | Image SHA-256 |
| --- | --- | --- |
| optimized | `4b6ed620fcf5bce0a3971062d4fe680a5b81530661ccb3cb800f382ab8bbbf84` | `ca25ac778da3ce202eff2ddae359856f5dd7b9f1534245ab9b0ba328f253bc85` |
| low | `d3ea06fa10d6a390fdf14763672bf117ea82a4aa47caa91ccf721f505fd40fe9` | `6765f7d679d7222a7f13214273e41ab366d165963cd3ec8bf854a1953a09dd73` |

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/cli -run '^TestAtlasReturnErrorProductionSources$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^optimized$'
go test -mod=readonly -count=1 -timeout 10m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^low$/^errors-'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-return-error-cli-20261106'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^optimized$/^(impact-families|impact-sdk-vector|catalog-ordinary|pack-ordinary|pack-sources|pack-sdk)$'
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^pack-sources$/^errors-'
go vet -mod=readonly ./internal/cli
```

The authored unlocked/locked checks pass on 679 files / 8,480 MIR functions:
`atlas-return-error-production-check-20261106.log` and
`atlas-return-error-production-locked-20261106.log`. Scoped vet passes in
`atlas-return-error-production-vet-20261106.log`; the Go source inventory log is
`atlas-return-error-production-sources-20261106.log`, all under `.cache/`.
Formatting, UTF-8/LF/final-newline and diff checks pass. Earlier Go core map,
snapshot and pack regressions are reused on their unchanged implementation
inputs. Borrow collection/production, broader source/command and all original
final gates remain open. Native Linux stays batched; no push or dispatch.
