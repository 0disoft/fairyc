# Windows Fairy Atlas drop command replay, 2026-11-06

Source: `1e4b9248`, before the separate trap collector. This verifies the actual
authored main and dispatcher after batched CFG drop production. It does not
attest all 41 registered CLI groups, future sources or any broader queue gate.
No Go fallback or build companion is installed; child PATH is empty.

## Final-source results

| Authored-main group | Windows low | Windows optimized |
| --- | ---: | ---: |
| impact-families | PASS, 540.15s | PASS, 8.43s |
| impact-sdk-vector | PASS, 347.08s | PASS, 15.73s |
| catalog-ordinary | PASS, 559.25s | PASS, 7.39s |
| pack-ordinary | PASS, 571.61s | PASS, 7.96s |
| pack-sources | PASS, 227.54s | PASS, 4.27s |
| pack-sdk | PASS, 372.12s | PASS, 16.21s |

All twelve selected profile/group combinations pass without selected skips.
The catalog, ordinary pack, source pack and SDK pack log respectively 44, 48,
25 and six native invocations per profile. Source packs include generic,
unused-template, cleanup-specialization, owner-namespace and cleanup-local-drop
cases. SDK packs cover ordinary SDK and used Vec; vector impact includes
unused, used and pop fixtures. Source/current/query/pack expectations come from
independent original Go facts and retain their drop spans and completeness.

The optimized command passes in 166.725s. Low commands pass separately in
580.838s (catalog), 593.346s (ordinary pack), 906.874s (impact selections) and
621.889s (source/SDK packs). Concurrent commands share CPU, so these are
execution observations rather than isolated performance measurements.

Retained images are authenticated by generated-C/profile/toolchain source
digests and actual binary digests on every run:

| Profile | Source digest | Image digest |
| --- | --- | --- |
| low | `551ce8c728bda16ea1f7b65e8d1bf2f09b2c58d3f433cdeca5299f5021fca0f9` | `f1dfebee8e7143781818465baa9b7aaeff71c6995a4e1299316765fe5433a574` |
| optimized | `a67d38623f62d9dac25b078f207e8277bf9334bdeed0d7d76eeba52fc23faf5d` | `45786b726fb1926a864530f677f4a4fd5dbacf0722064b6c89d57bfd77895685` |

## Commands and logs

Set `FAIRYC_RUN_PARITY_INTEGRATION=1` and an absolute
`FAIRYC_ATLAS_CLI_ARTIFACTS` pointing to `.cache/atlas-drop-production-cli`.
The selectors below reproduce the groups with
`go test -mod=readonly -count=1 -failfast -v ./internal/cli`. Use a bounded
parent timeout (for example `-timeout 20m` for a low group):

| Exact `-run` selection | Local log |
| --- | --- |
| `^TestCandidateAtlasCLI$/optimized/(pack-ordinary\|pack-sources\|pack-sdk\|impact-families\|impact-sdk-vector\|catalog-ordinary)$` | `.cache/atlas-drop-batch-cli-optimized-20261106.log` |
| `^TestCandidateAtlasCLI$/low/catalog-ordinary$` | `.cache/atlas-drop-batch-cli-low-catalog-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(impact-families\|impact-sdk-vector)$` | `.cache/atlas-drop-batch-cli-low-impact-20261106.log` |
| `^TestCandidateAtlasCLI$/low/pack-ordinary$` | `.cache/atlas-drop-batch-cli-low-pack-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(pack-sources\|pack-sdk)$` | `.cache/atlas-drop-batch-cli-low-source-sdk-20261106.log` |

Only parent Go-test grouping limits differ. Product C compilation stays at
60 seconds and native helper process limits are unchanged. The earlier broad
20-minute parent selection timed out (`atlas-drop-batch-cli-20261106.log`);
it is not counted as a pass. Splitting the selections retains every assertion.
The pre-batching optimized result also remains historical, not final evidence.

## Remaining scope

This completes the six selected drop command comparisons only. Other full-CLI
registrations, remaining observed-relation collectors and source families,
generated consumers, complete Go/Fairy comparison, consecutive self-build,
Windows distribution/clean-machine execution, native Linux and same-commit
hosted CI remain open in the implementation queue. No product source, public
API, ABI, DB, dependencies, timeout policy, runners or hygiene rules change in
this documentation receipt. No push or dispatch.
