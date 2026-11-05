# Current Windows authored Atlas CLI replay, 2026-11-05

## Scope and bounded execution

The full `TestCandidateAtlasCLI` retains the authored main/dispatcher and all
nineteen original verifier calls plus help, in their original order, and the new
static-operation impact/MCP verifiers. Each now
has a named subtest, allowing bounded profile/group replays. Existing assertions,
per-process deadlines, C split limit and 16 MiB stack policy stay unchanged.
PATH is empty for every verifier, with no Go child, companion or prebuilt-map
substitution for the Fairy compiler.

The existing retained-image policy rederives current generated C and runtime,
profile, toolchain name/path/version and stack policy before reuse. Both the
source-policy digest and actual image SHA-256 must match the retained receipt;
otherwise the image is rebuilt. Every profile now logs both digests. Selecting a
group attests only that group; a partial profile pass is not a full replay.

Current C input is a 1,976,252-byte header, 153,115-byte runtime and 24 body
parts, largest 2,096,561 bytes. All parts fit the unchanged 2 MiB limit.
Product source is unchanged from `2b1a2d2a`; no compiler, source-admission or
wire change is introduced by named test grouping.

## Optimized full replay

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-cli-20261105'
go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run '^TestCandidateAtlasCLI$/^optimized$'
go vet -mod=readonly ./internal/cli
```

All twenty original optimized groups pass with no selected skips in 248.753s (native
profile 228.26s). The exact expected top-level group set is also compared with
the completed test log; nested Vec fixture results do not inflate that count.
Scoped vet, Go formatting and diff checks pass. Log:
`.cache/atlas-full-cli-optimized-20261105.log`; scoped vet:
`.cache/atlas-full-cli-vet-20261105.log`.

The newly registered `impact-static-operations`/`mcp-static-operations` groups remain pending in this
full authored-main image. Its separate command-leaf acceptance is recorded in
the static-operation receipt and does not count as this full-image replay.

Optimized generated-input/policy SHA-256:
`7ca0718ef3068260576a7d0541f6527b4668ccd345646a6958e5c2fbf1125881`.
Image SHA-256:
`f22e421b6e0d5d29247bbfdb58923f3f0a3cb0bef570b13454ecfdf9a4878c0e`.
The image is 5,642,752 bytes.

## Low replay coverage

Low acceptance remains partial; selected groups must pass on the same current
generated-input/policy and image digests before they are counted. The current
low image is 14,811,648 bytes. Its generated-input/policy SHA-256 is
`928c51f83ee93731750f5519997eb3da2e4ae55ac21de9dc0aa8c0d83fb9702d`, and its
image SHA-256 is
`3e181c6e1be313e7ea87bcfb442b0275b7e14f0fa2a654469fddcd823e3f2b86`.

The mixed impact/MCP/help replay passes all three selected groups without skips
in 628.749s (native profile 608.92s), with both digests matching the image above:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^(impact-sdk-aggregates|mcp-sdk-aggregates|help)$'
```

The same integration/artifact environment shown above was set. Log:
`.cache/atlas-full-cli-low-mixed-query-20261105.log`. An earlier mixed editor
attempt was explicitly interrupted during a read-only status request and is not
counted. Its completed retry passes in 441.443s (native profile 422.31s), again
with the same source/image digests:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^edit-sdk-aggregates$'
```

Completed retry log: `.cache/atlas-full-cli-low-mixed-edit-20261105.log`. All four
mixed/help groups pass.

The seven core query groups also pass without selected skips in 1042.132s
(native profile 1023.38s), with the same source/image digests:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^(impact-ordinary|impact-families|mcp-ordinary|mcp-aggregates|unsupported|impact-generics|impact-aggregates)$'
```

Log: `.cache/atlas-full-cli-low-core-query-20261105.log`. Eleven distinct low
groups now pass; the nine remaining original groups and the new static-operation
groups remain pending.

| Group | Optimized | Low |
| --- | --- | --- |
| impact-ordinary | Pass | Pass |
| impact-families | Pass | Pass |
| mcp-ordinary | Pass | Pass |
| mcp-aggregates | Pass | Pass |
| mcp-static-operations | Pending | Pending |
| edit-ordinary | Pass | Pending |
| edit-publication | Pass | Pending |
| edit-aggregates | Pass | Pending |
| unsupported | Pass | Pass |
| impact-generics | Pass | Pass |
| impact-static-operations | Pending | Pending |
| impact-aggregates | Pass | Pass |
| impact-sdk | Pass | Pending |
| impact-sdk-vector | Pass | Pending |
| impact-sdk-aggregates | Pass | Pass |
| mcp-sdk | Pass | Pending |
| mcp-sdk-vector | Pass | Pending |
| mcp-sdk-aggregates | Pass | Pass |
| edit-sdk | Pass | Pending |
| edit-sdk-vector | Pass | Pending |
| edit-sdk-aggregates | Pass | Pass |
| help | Pass | Pass |

The complete low replay, broader Atlas source families, final whole-compiler
Go/Fairy comparisons, consecutive self-build and Windows distribution/clean-machine
acceptance remain open. Native Linux stays batched. Hosted same-commit CI requires
an authorized push/dispatch. No grammar, public CLI, SDK/runtime ABI, map wire,
DB, dependency, timeout, target, repository hygiene or runner change. No push,
dispatch, release or deployment.
