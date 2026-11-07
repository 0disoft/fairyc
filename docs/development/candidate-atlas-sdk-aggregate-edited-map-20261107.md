# Edited SDK/aggregate whole maps, 2026-11-07

Status: the specific edited-source whole-map observation left open by
`candidate-atlas-sdk-aggregate-commands-20261107.md` passes Windows low/optimized.
This does not close the broader Atlas implementation/final-acceptance queue.

## Exact source and independent expectations

The new test uses `atlasSDKAggregateCleanupEditedSource()`, the exact source used
by the existing actual editor/impact acceptance. Its SHA-256 is
`6bc8280da93a19b09c1ea12501c05efe3d018ead5854e28bb4d9a3098de9e130`.
This test writes those edited bytes into a fresh fixture and independently runs
current Go Atlas v3 preparation. It does not run the editor again or substitute
Go maps, a checked Complete or a mode selector into Fairy production.

The existing independent oracle checks the new `Cell<bool>`/`cell<bool>` and
`Check<bool>` facts, three cleanup-to-SDK calls, all fourteen unchanged concrete
SDK implementations and pinned SDK declarations/source/interface identities.
Fairy's actual `project_atlas_source::load` chooses aggregate mode, checks and
builds Complete, produces maps, admits them against current sources and joins
the packages. Native child PATH is empty.

The native records are decoded by strict Go v3 readers and compared against
fresh Go expectations. The comparison includes running executable hash,
file/module records, source/dependency/package-reference identities, declared
symbols/types/spans, interfaces, ordinary function facts, concrete aggregate
fields, specializations and cleanup summaries. Each map checks all fourteen
completeness rows against the admitted collector contract and independently
observed Go counts. Uncollected reads/writes/test-calls and unspecialized template
bodies retain unavailable status; the maps are not relabelled fully exact.

| Package | Declarations | Functions | Implementations | Relations |
| --- | ---: | ---: | ---: | ---: |
| user app | 5 | 2 | 3 | 8 |
| local answer | 1 | 1 | 0 | 0 |
| SDK | 278 | 156 | 14 | 516 |

Both profiles produce all three maps without duplicates. Global resolved calls
are 128, including three cross-package cleanup calls; concrete implementations
total seventeen. The edited map is current under new source and refuses the
saved pre-edit user expectation with `ErrStaleMap`. All workspace files and the
exact edited source remain unchanged through native production.

## Windows verification

The Sol High worker owns the new test only; the coordinator adds explicit
file/module and running-image checks and owns integration/commit. No production
source changes are needed by this verification slice.

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly -count=1 -failfast -timeout 12m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateEditedMap$'
go vet -mod=readonly ./internal/cli
```

The final strengthened test passes in 205.26s (package 205.413s), low 122.11s /
optimized 66.11s, with no selected skips. Log:
`.cache/atlas-sdk-aggregate-edited-map-final-20261107.log`. The earlier worker
matrix passes in 240.445s before the additional file/module/image assertions;
its log is `.cache/atlas-sdk-aggregate-edited-map-20261107.log`. An initial stale-
source test assumption and an integration comparison of string/typed identity
fail before the final corrections; those attempts remain failed evidence.
Warning levels, product bounds and timeouts are unchanged. Gofmt, scoped vet
and diff checks pass.

Only the new Go test and this record change the verification behavior. Source/
lock checks from the unchanged enum-integration source are reused rather than
repeated for a test-only change. The exact current source/lock inventory still
matches that successful snapshot. No language/schema contract, public API/SDK
ABI, DB, dependency, repository-policy or runner change. Linux remains batched;
full authored-main groups and refreshed final gates remain pending. No push,
hosted dispatch, release, deployment or publication.
