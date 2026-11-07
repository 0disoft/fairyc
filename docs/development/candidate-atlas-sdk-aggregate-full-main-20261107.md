# SDK aggregate cleanup through the full main, 2026-11-07

The two remaining SDK/aggregate cleanup groups now pass the actual authored
`file_cli` main and dispatcher, at Windows low and optimized. This closes the
full-main follow-up to `candidate-atlas-sdk-aggregate-commands-20261107.md`;
the edited whole-map comparison has its separate receipt. Unselected Atlas
groups and later compiler changes are not attested by this selection.

## Harness correction and source boundary

The first run refuses before native command execution: the full compiler emits
a 2,137,711-byte header, above ordinary project admission. The test now selects
the existing explicit compiler-host builder from ADR 0510, and includes that
role in the reusable generated-C/profile/toolchain/image digest. No header
limit, body limit, deadline, stack reserve or warning policy is enlarged.

There are 26 C body parts, largest 2,097,087 bytes, and 160,657 runtime bytes.
The default one-minute native compile-unit bound and 16 MiB compiler-host stack
remain unchanged. The complete source was copied from the `0bb199e7` state
before the parallel flat-Option implementation. These images must not be
relabelled as acceptance of that later compiler change or current self-build.

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-fullmain-current-20261107'
go test -mod=readonly -count=1 -failfast -timeout 30m -v ./internal/cli -run '^TestCandidateAtlasCLI$/^(low|optimized)$/^(mcp-sdk-aggregate-cleanup|edit-sdk-aggregate-cleanup)$'
```

Final package result: **PASS, 1166.271s**, no selected skips.

| Group | low seconds | optimized seconds |
| --- | ---: | ---: |
| MCP SDK aggregate cleanup | 253.69 | 9.53 |
| Edit SDK aggregate cleanup | 717.79 | 28.06 |
| Complete profile including native construction | 995.02 | 141.81 |

The existing verifiers retain real stdio/protocol, locked source refresh,
stale-source refusal and recovery, edit preview/publication/refusals, current
user and SDK queries, independent Go inventories, and actual running compiler
identity. Both profiles observe 128 resolved calls, three cross-package cleanup
calls, and the edited three-package inventory including all 14 SDK instances.
Native child PATH is empty; no companion or Go frontend handles these commands.

## Retained identities

| Profile | Generated C/profile/toolchain digest | Complete image SHA-256 |
| --- | --- | --- |
| low | `2fd3cd81c73ecd3c5ec8c302dc8056492c1327f4bc3b6d1ea455feab562c8d6a` | `32d02a78c962fdc891fb62f5c442ce03d4684274a693d4b3d5455c2a94aac0f7` |
| optimized | `883cff205de8dc80441aef0af578c40d69a18af922a8481ca24515baf58c84ce` | `43d16c2b6b814646bedeef52e2cc476f7399bc39abf60e1e98146dea5a84ced1` |

Images and digest records are retained under the artifact directory above.
Final log: `.cache/atlas-fullmain-sdk-aggregate-cleanup-host-20261107.log`,
SHA-256 `801211dc5e91e9c35f38053d165ae479c5f1c604f74210655f0feeed425ee20a`.
The initial refused run remains separately in
`.cache/atlas-fullmain-sdk-aggregate-cleanup-20261107.log`.

This test-only change alters no source-language/Atlas contract, public API,
SDK/runtime ABI, database, dependency, runner or repository policy. The full
Atlas group set, later compiler-source acceptance, self-build/fixed-point,
distribution, actual clean machine and hosted CI remain separate. Linux stays
batched; no push, dispatch, release, deployment or publication.
