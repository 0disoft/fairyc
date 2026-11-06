# Windows Fairy Atlas unsafe command replay, 2026-11-06

Product source: `f8fc5354`, with collector `29bb4e58` and the unchanged-limit
JSON key index from `b9c99000`.

The authored main and dispatcher retain original Go unsafe observations in
ordinary, specialized and concrete cleanup summaries. The separately witnessed
pinned SDK leaf retains the helper's source span, without inventing a caller
site for its folded storage operation. Only uses-unsafe is newly exact; all
uncollected relation classes remain unavailable and map generation stays partial.
No Go fallback or companion is installed, and child PATH is empty.

## Source results

| Authored-main group | Windows low | Windows optimized |
| --- | --- | --- |
| impact-families | pending | PASS, 8.18s |
| impact-sdk-vector | pending | PASS, 15.23s |
| catalog-ordinary | pending | PASS, 7.37s |
| pack-ordinary | pending | PASS, 8.52s |
| pack-sources | pending | PASS, 5.50s |
| pack-sdk | pending | PASS, 19.58s |

The completed optimized parent passes in 179.891s. Its catalog and ordinary,
source and SDK packs retain respectively 44, 48, 33 and six native invocations.
The source pack corpus adds unsafe ordinary and concrete cleanup controls;
SDK packs retain ordinary SDK and used Vec, while vector impact retains
unused/used/pop. Every expected declaration, site, trap, count, omission,
specialization, query/pack byte and checksum comes from original independent
Go facts, with only still-uncollected classes explicitly projected away.

Each selected test parent authenticates the retained image against the complete
generated-C/profile/toolchain input and actual binary digests:

| Profile | Source digest | Image digest |
| --- | --- | --- |
| low | `5bd2b49a6e662dc1f8c0b988b7062951c9b8a88614a6995087f6bcac8b32f89f` | `267ffc08aba65465decf36c0ff489cf9e510d52ec7d6262a47594d652494e7f7` |
| optimized | `20f5d708424170a1fb136239f7270ea759c2af27805ec743eaa366f791e5f652` | `85a40bdc476ae274cb3626740583708fcbc7fa19900016d37e0732438310aa77` |

The input has a 2,061,188-byte header, 153,115-byte support runtime and 25 body
units; the largest body is 2,097,059 bytes. Existing split limits, 60-second C
build limit and native process policies are unchanged. Runs share CPU; elapsed
times are observations, not isolated performance measurements.

## Commands and local logs

Set `FAIRYC_RUN_PARITY_INTEGRATION=1` and an absolute
`FAIRYC_ATLAS_CLI_ARTIFACTS` pointing to `.cache/atlas-unsafe-production-cli`.
Use `go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli`
with these exact `-run` selections. Bounded parents preserve every selected
verifier assertion; unselected registrations are not attested.

| Exact selection | Local log |
| --- | --- |
| `^TestCandidateAtlasCLI$/optimized/(catalog-ordinary\|pack-ordinary\|pack-sources\|pack-sdk\|impact-families\|impact-sdk-vector)$` | `.cache/atlas-unsafe-production-cli-optimized-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(catalog-ordinary\|pack-ordinary\|pack-sources)$` | `.cache/atlas-unsafe-production-cli-low-pack-catalog-20261106.log` |
| `^TestCandidateAtlasCLI$/low/(pack-sdk\|impact-families\|impact-sdk-vector)$` | `.cache/atlas-unsafe-production-cli-low-impact-sdk-20261106.log` |

## Remaining scope

Low replay is still running and is not counted as a pass here. The other 35
registered CLI groups, remaining move/allocation/error/borrow collection,
broader source/editor coverage, generated consumers, complete Go/Fairy
comparison, consecutive self-build, Windows distribution/clean-machine
execution, native Linux and same-commit hosted CI stay open in the complete
implementation queue. This receipt changes no stable API/ABI, DB, dependency,
timeout, runner or repository policy. No push or dispatch.
