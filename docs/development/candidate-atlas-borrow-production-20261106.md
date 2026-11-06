# Complete-bound Atlas borrow publication, 2026-11-06

Continues the [semantic collector](candidate-atlas-borrow-sites-20261106.md) and
[Complete/current-source restoration](candidate-atlas-borrow-observations-20261106.md).
The broader implementation queue and final acceptance gates remain separate.

## Production ownership

`project_atlas_produce::package_version` takes borrow observations from the actual
checked Complete after existing current-source/schema admission. Function and
concrete generic/cleanup summaries serialize every retained observation with its
original source span, shared/mutable kind and escape flag. Canonical order and
multiplicity are retained. Pure SDK inline leaves have no executable borrow sites.
Copied closure projections use their separate checked helper frame.

Summary and package relation counts include borrows exactly once. Source-backed
headers/completeness mark only the newly observed `borrows` relation exact and
remove its previous omission. Uncollected effects/reads/writes/test-calls and
uninstantiated generic-body coverage remain unavailable; generation stays partial.

Existing private raw-IR package and summary signatures remain. They use an absent
observation table and retain unavailable borrow completeness. An empty observed
table is distinct from absent evidence. Observed summary construction rejects an
absent table or a frame count different from its executable IR. Existing header
ports also preserve their previous unobserved behavior through explicit wrappers.
No syntax scan or second source analysis is introduced to fabricate observations.

The source-backed ordinary/generic/SDK test harnesses now enter the Complete
factory. Independent Go expectations compare original Borrow arrays and count
them directly. Pack expectations no longer replace those original arrays with
`[]`; their original spans/kinds/escape flags must survive actual command output.

## Windows verification scope

`TestCandidateAtlasBorrowCompleteness` compares all fourteen relation classes
before/after explicit observation authority; only borrow completeness may change.
`TestCandidateAtlasSourceVersion` adds twenty-seven original borrow families, an
imported returned-parameter case and a copied-closure owner case across v1/v2/v3.
The existing SDK vector/generic matrices retain full Go summary and count checks.
Source-version execution also validates current-map and workspace-join admission.

Seven source pack cases exercise shared/mutable locals, returned parameters,
retained call results, multiple origins, generic functions and concrete cleanup
owners through actual overview/show/around and old-version refusal commands.
Original command verifiers remain in place. Native runtime PATH is empty; source
workspace preservation remains required.

Verified Windows commands and timings:

| Scope | Result | Seconds | Log under `.cache/` |
| --- | --- | ---: | --- |
| Absent/observed completeness, both profiles | pass, two native processes | 15.10 | `atlas-borrow-production-windows-final-20261106.log` |
| Generic production, both profiles | preceding root passes; enclosing command later times out | 133.53 | same log |
| Complete source/version/current/join corpus, optimized | pass, 218 native processes / 381 independent maps | 192.907 | `atlas-borrow-source-versions-optimized-20261106.log` |
| Six actual optimized CLI groups | pass | 258.586 | `atlas-borrow-cli-optimized-connected-20261106.log` |
| Seven actual low borrow Pack families | pass, 24 native processes | 335.439 | `atlas-borrow-cli-low-20261106.log` |
| SDK unused/used/pop sources, both profiles | pass, six complete native map comparisons | 445.216 | `atlas-borrow-sdk-sources-20261106.log` |
| Five SDK budget/coordinate controls, optimized | pass | 88.267 | `atlas-borrow-sdk-controls-optimized-20261106.log` |

The six full-main groups are impact-families, impact-sdk-vector, catalog-ordinary,
pack-ordinary, pack-sources and pack-sdk. All seven new Pack families pass both
profiles: 48 new processes, consisting of 42 successes and six old-version
refusals. These selections do not attest every original full-main group or the
complete low source-version corpus. The preceding collector/Complete receipt
retains the separate twenty-eight-family both-profile span/kind/escape proof.

| Profile | C/profile/toolchain source digest | Image SHA-256 |
| --- | --- | --- |
| low | `3325631b738f8edffeb06c53d878a68cad69eb1225874805afd0d1ec39c038bd` | `820830cb2df0a689769d1684e4af39d4f70848e40d9d7da1492785237524ab75` |
| optimized | `d58de9b9c85512a851b408d98af7b466c765a5fff4f211fb783b5ac2ae499c58` | `dd418f1b7f7ea26149cd414825216d9104e4c086bbfa40c2dffbfd32167a3bd8` |

Authored check and locked check pass 684 files / 8,548 MIR functions; logs are
`atlas-borrow-production-{source,locked}-connected-20261106.log`. Scoped CLI and
C-backend vet, complete atlasmap/contextpack/semanticsnapshot suites, formatting,
UTF-8/LF/final-newline checks and diff checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/^optimized$'
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestCandidateAtlasSDKVectorProduction$/^(low|optimized)$/^(unused|used|pop)$/^source$'
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestCandidateAtlasSDKVectorProduction$/^optimized$/^unused$/^(source-budget|export-budget|validation-budget|join-budget|missing-coordinate)$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-borrow-cli-20261106'
go test -mod=readonly -count=1 -timeout 20m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^optimized$/^(catalog-ordinary|impact-families|pack-ordinary|pack-sources|pack-sdk|impact-sdk-vector)$'
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^low$/^pack-sources$/^borrows-'
```

## Attempts and limits

The first authored source check exposed a missing explicit `usize` on a new
zero-or-site-count expression; it was corrected without changing typing rules.
An initial completeness test compared JSON `[]` with Go nil using DeepEqual;
the assertion now checks exact scalar fields and an empty reason list. The first
full-CLI invocation omitted its opt-in environment flag and skipped; it supplies
no acceptance evidence. The explicitly enabled invocation is recorded separately.
The initial bounded edit script stopped on mismatched occurrence counts before
publishing each affected file; subsequent source checks cover the final files.

Actual full-main integration also reproduced a 2,099,763-byte shared C header,
2,611 bytes above the existing native-driver limit. The separate
[shared-prototype correction](shared-c-prototype-budget-20261106.md) removes only
unused declaration parameter names, reducing it to 2,096,365 bytes. A cleanup-only
intermediate reduction remained 683 bytes over and is retained as a failed run.
No driver/header budget is raised; the final full-CLI replay uses the correction.

The first compiled full-main impact case also exposed one remaining old raw-IR
producer call in `project_atlas_source`. It retained unavailable borrows despite
the new Complete-bound producer. That actual query path now calls
`package_version` with its current Complete, as catalog/calls/pack already do.
The independent expected omissions are preserved; they were not changed back
to accept the missed connection. The final input after this connection has a
2,096,156-byte header, 26 body units with largest 2,096,431 bytes, and the same
153,115-byte runtime.

The original combined four-root native command reached the unchanged 15-minute
Go package timeout during SDK optimized compilation, before SourceVersion began.
Its passed completeness/generic roots are retained as their bounded evidence,
but the enclosing command and unfinished SDK/version roots are not passes.
SDK positive sources and budget/coordinate controls now have named subtests,
preserving default execution while allowing bounded selective replay. They do
not skip checks or extend product timeouts. The separate six-source SDK replay
passes in 445.216s: low 347.95s and optimized 83.00s including native builds.
Full low SDK fault-control replay remains separate from its positive-source proof.

Native Linux remains batched. Full-source family parity, all full-main groups,
consecutive self-build, distribution/clean-machine and same-commit hosted gates
remain open. No public CLI/API, SDK/runtime ABI, DB, dependencies, target policy,
repository hygiene or runner changes. No push, CI dispatch, release or deployment.
