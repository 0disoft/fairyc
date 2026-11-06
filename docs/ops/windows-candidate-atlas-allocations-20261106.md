# Windows Fairy Atlas allocation production, 2026-11-06

Scope: existing Complete-owned source/version maps and actual authored-main CLI.
The private collector was accepted separately at `95e26d5e`. This receipt does
not promote unrelated source families, every CLI verifier, native Linux,
consecutive self-build, distribution or same-commit hosted CI.

Implementation source revision: `04894e8f`. The source/image digests below bind
this receipt to that allocation implementation, before later collector work.

## Source and command checks

`TestAtlasAllocationProductionSources` fixes five independent Go inventories:
ordinary, partial operand order, generic, cleanup and loop/Result. Twelve
source/version combinations are admitted and three older-schema combinations
reject. Source/version production compares original Go declarations, symbols,
allocation spans, specializations, counts and current-source expectations;
candidate output never supplies the expected inventory.

| Scope | Result | Seconds | Log under `.cache/` |
| --- | --- | ---: | --- |
| Independent Go source/version inventory | pass | 1.375 | `atlas-allocation-production-sources-20261106.log` |
| Full optimized source/version/current/join corpus | pass; 113 native processes, 183 current maps | 147.252 | `atlas-allocation-production-optimized-20261106.log` |
| Low allocation ordinary and operand-order corpus | pass; 6 native processes, 18 current maps | 692.323 | `atlas-allocation-production-low-a-20261106.log` |
| Low allocation generic, cleanup and loop/Result corpus | pass; 9 native processes, 18 current maps | 711.491 | `atlas-allocation-production-low-b-20261106.log` |
| Six selected actual optimized CLI groups | pass | 267.342 | `atlas-allocation-cli-optimized-20261106.log` |
| Actual low allocation ordinary/operand-order pack commands | pass; 6 native processes | 697.183 | `atlas-allocation-cli-low-a-20261106.log` |
| Actual low allocation generic/cleanup/loop pack commands | pass; 9 native processes | 1079.871 | `atlas-allocation-cli-low-b-20261106.log` |

The optimized full-main groups are impact-families, impact-sdk-vector,
catalog-ordinary, pack-ordinary, pack-sources and pack-sdk. The latter includes
five new allocation owners through overview/show/around: fifteen new processes
per compiler profile, exact independent Go pack bytes and unchanged workspaces.
Low replay selects these five new cases separately; it does not attest the
other low command groups or every pre-existing SDK pack fixture.

All five new allocation cases pass both profiles without selected skips: thirty
fresh actual-main pack processes in total. Each bounded low parent exits zero.
The recorded times are concurrent local observations, not a controlled benchmark.

The full compiler image retains the real main/dispatcher and has empty PATH
during every command. No companion or Go frontend fallback performs Atlas work.
The input has 26 C body units, a 2,079,347-byte shared header, a 153,115-byte
runtime and a largest unit of 2,097,044 bytes. The ordinary product compiler
timeouts, strict MSVC warnings and private 16-MiB stack policy are unchanged.

| Compiler profile | Authenticated C/profile/toolchain source digest | Image SHA-256 |
| --- | --- | --- |
| optimized | `ea3064dd45dfa09d6348c7f26536b6eabc956c1169fb7880dcbc574218269245` | `33cea21e7051216600e0d1da4ba996dc27fac50e5095e14490951e99caab942f` |
| low | `b44d9208370443c2bdb06e32b9a804efe8849cf99d4ff61977b57aba4f7920e5` | `a262edfbc595b555a44475b93eac336fd54ffaee5febfb55bcb84807340db8ac` |

Both separately retained low images have identical source and image digests.
Independent directories avoid concurrent writes to the retained image receipt.

## Reproduction

```powershell
go test -mod=readonly -count=1 -timeout 3m -v ./internal/cli -run '^TestAtlasAllocationProductionSources$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^optimized$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^low$/^sdk-allocations-(ordinary|operand-order)$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^low$/^sdk-allocations-(generic|cleanup|loop-and-result)$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-allocation-cli-20261106'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^optimized$/^(impact-families|impact-sdk-vector|catalog-ordinary|pack-ordinary|pack-sources|pack-sdk)$'
go test -mod=readonly -count=1 -timeout 20m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^pack-sdk$/^allocations-(ordinary|operand-order)$'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-allocation-cli-low-b-20261106'
go test -mod=readonly -count=1 -timeout 20m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^pack-sdk$/^allocations-(generic|cleanup|loop-and-result)$'
```

## Other checks and boundaries

Full Go atlasmap/contextpack/semanticsnapshot suites pass in 2.284s/0.253s/0.210s;
scoped CLI vet also passes. Logs are `atlas-allocation-production-go-regression-20261106.log`
and `atlas-allocation-production-vet-20261106.log` under `.cache/`.
The authored check and locked check pass on the same Fairy inputs: 678 source
files and 8,454 MIR functions. Logs are `atlas-allocation-production-check-20261106.log`
and `atlas-allocation-production-locked-20261106.log`.

The earlier combined low source/version run was interrupted after observed SDK
costs required smaller parents. Its log is
`atlas-allocation-production-low-20261106.log`; interruption is not a pass.
An initial second low full-main parent was interrupted before image production
to separate the retained directories; its log is
`atlas-allocation-cli-low-b-interrupted-20261106.log`, also not a pass.

Error propagation and borrow collection/production, broader source/CLI/generated
consumer coverage and every original final gate remain open. Native Linux stays
batched. No stable API/SDK ABI, DB, dependency, map/cache schema, default timeout,
runner or repository-policy change; no push, dispatch, release or deployment.
