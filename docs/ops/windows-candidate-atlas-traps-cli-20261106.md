# Windows Fairy Atlas trap command replay, 2026-11-06

Source: `53728a07d47092e05c582c59d4a0ecc827ace312`, with shared verified
collection from `37e57d98`. The actual authored main and dispatcher publish
ordinary/generic/cleanup trap categories and count them with calls and drops.
Only may-trap is newly exact; other uncollected classes remain unavailable.
No Go fallback or build companion is installed; child PATH is empty.

## Source results

| Authored-main group | Windows low | Windows optimized |
| --- | ---: | ---: |
| impact-families | PASS, 466.99s | PASS, 7.92s |
| impact-sdk-vector | PASS, 344.21s | PASS, 12.70s |
| catalog-ordinary | PASS, 489.63s | PASS, 6.88s |
| pack-ordinary | PASS, 498.54s | PASS, 7.24s |
| pack-sources | PASS, 237.66s | PASS, 4.88s |
| pack-sdk | PASS, 398.11s | PASS, 15.85s |

All twelve selected profile/group combinations pass without selected skips.
The catalog, ordinary pack, source pack and SDK pack log
respectively 44, 48, 25 and six native command invocations per profile. Source
packs retain generic, unused-template, cleanup-specialization, owner-namespace
and cleanup-local-drop coverage. SDK packs include ordinary SDK and used Vec;
vector impact includes unused, used and pop cases. Expected declarations,
calls, drops, trap categories, counts, completeness, query/pack bytes and
checksums are derived from independent original Go facts. Candidate output is
never the source of expected facts.

The optimized parent passes in 158.813s. The completed low parents pass in
527.017s (catalog), 518.184s (ordinary pack), 655.367s (source/SDK packs),
487.345s (impact families) and 364.277s (SDK vector impact).
Commands share CPU, so timings are observations rather than isolated
performance measurements.

Every invocation authenticates the retained image against the complete
generated-C/profile/toolchain input and actual binary digests. Parents started
before and after the source commit retain the same pair for each profile:

| Profile | Source digest | Image digest |
| --- | --- | --- |
| low | `8a602e851c57820b3c623ccd42596f53a33177bc6c36486647ff133211f5a380` | `ddb35e0a2828b6a4840dcfd67cb8bffcc0a321fb090e49f259120c5b7edb0c1b` |
| optimized | `f233e33f51913e0e8d94ef24b88a6b8286883a353d30b38636e444344a10ead4` | `6cac6540b97a0968582d20b25f6d92ac96cee10291fed4222ae446b5f2e83239` |

The input has a 2,049,259-byte header, 153,115-byte support runtime and 25 body
units; the largest body is 2,097,059 bytes. Existing split limits, 60-second C
build limit and native process policies are unchanged.

## Commands and local logs

Set `FAIRYC_RUN_PARITY_INTEGRATION=1` and an absolute
`FAIRYC_ATLAS_CLI_ARTIFACTS` pointing to `.cache/atlas-trap-production-cli`.
Use `go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli`
with the exact selectors below. Separate bounded parents retain every selected
verifier assertion; they do not attest unselected registrations.

| Exact `-run` selection | Local log |
| --- | --- |
| `^TestCandidateAtlasCLI$/optimized/(catalog-ordinary\|pack-ordinary\|pack-sources\|pack-sdk\|impact-families\|impact-sdk-vector)$` | `.cache/atlas-trap-production-cli-optimized-20261106.log` |
| `^TestCandidateAtlasCLI$/low/catalog-ordinary$` | `.cache/atlas-trap-production-cli-low-catalog-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(pack-ordinary)$` | `.cache/atlas-trap-production-cli-low-pack-ordinary-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(pack-sources\|pack-sdk)$` | `.cache/atlas-trap-production-cli-low-pack-sources-sdk-20261106.log` |
| `^TestCandidateAtlasCLI$/low/impact-families$` | `.cache/atlas-trap-production-cli-low-impact-families-20261106.log` |
| `^TestCandidateAtlasCLI$/low/impact-sdk-vector$` | `.cache/atlas-trap-production-cli-low-impact-sdk-vector-20261106.log` |

## Remaining scope

This closes only the six selected trap command groups. The other 35 registered
CLI groups, unsafe/move/
allocation/error/borrow collection, broader source/editor coverage, generated
consumers, complete Go/Fairy comparison, consecutive self-build, Windows
distribution/clean-machine execution, native Linux and same-commit hosted CI
stay open in the implementation queue. This documentation adds no product
source, public API/ABI, DB, dependency, timeout, runner or hygiene-policy change.
No push or dispatch.
