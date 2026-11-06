# Fairy Atlas declared effects, 2026-11-06

## Collection boundary

The Go snapshot already defines exact `effects` as declaration authority, not
transitive runtime behavior. Fairy's real package factory already emits every
declaration through `project_atlas_schema::declaration_named`: ordinary/foreign
functions take their checked signature bits; original generic function headers
read their declared effects; resource declarations retain a separate sorted
`drop_effects` set. Function effects on resource declarations remain empty.
This includes unused foreign declarations and unused generic headers. It does
not assert that every possible generic body or external C behavior was checked.

`project_atlas_produce::encode` now selects the explicit collected-effects
header/completeness factories. There is no new transport version and no effect
row added to executable relation counts. Every successful Package constructor
uses the schema path; the old scalar-only `project_atlas_declarations::declaration`
writer is not used there. Legacy header/version/borrow-observation factories
still lack that evidence and retain unavailable effects. Borrow authority is
independent. Calls with FFI remain partial, generic-template-bodies remain
unavailable, and reads/writes/test-calls retain their previous unavailable scope.

## Independent checks

`TestCandidateAtlasEffectCompleteness` compares twelve pairs across v1/v2/v3,
absent/observed borrows and zero/nonzero external calls. Only effects may change;
the legacy state remains unavailable and the collected state is exact with no
invented resolved/external/unknown counts.

Three source-version families exercise all twelve authorities in scrambled
spelling, pure declarations, associated methods, cross-package clock authority,
used and unused foreign declarations, used/unused generic function headers and
used/unused generic resource cleanup declarations. Complete declarations,
symbols, functions, specializations, relation counts, current-map admission,
workspace joins and catalog output are compared with original Go source facts.
Unsupported older generic schema selections must refuse without output. These
tests preserve workspace bytes and execute the generated Fairy image with an
empty PATH.

## Windows evidence

Logs are local artifacts under `.cache/`, not design authority.

| Scope | Result | Seconds | Log |
| --- | --- | ---: | --- |
| Legacy/collected effects, both profiles | pass, 24 pair comparisons across two native processes | 14.608 | `atlas-effects-header-replay-20261106.log` |
| Legacy/observed borrows, both profiles | root passes; enclosing first header command fails on the new test fixture | 13.06 | `atlas-effects-header-20261106.log` |
| New effects source/version/current/join families, both profiles | pass, 18 native processes / 28 independent maps, including four old-version refusals | 161.621 | `atlas-effects-source-replay-20261106.log` |
| Authored check / locked check | pass, 684 files / 8,550 MIR functions | — | `atlas-effects-authored-{check,locked}-20261106.log` |
| Atlas map / Context Pack / semantic snapshot suites | pass | 5.737 | `atlas-effects-map-regressions-20261106.log` |
| Generic used/unused and SDK Vec unused/used/pop production, optimized | pass, original declarations/summaries and source/current/join checks | 163.119 | `atlas-effects-generic-sdk-20261106.log` |
| CLI vet | pass | — | `atlas-effects-vet-20261106.log` |
| Six actual authored-main CLI groups, optimized | pass, no selected skips; authenticated image reused | 169.970 | `atlas-effects-cli-optimized-replay-20261106.log` |

The actual authored CLI C input stays within existing budgets: common header
2,096,201 bytes; runtime 153,115 bytes; 26 body units, maximum 2,096,312 bytes.
The 2MiB limits, warning levels, build/runtime timeouts and stack policy are
unchanged. The six selected full-main groups are impact-families, impact-sdk-vector,
catalog-ordinary, pack-ordinary, pack-sources and pack-sdk. The optimized image
has source/profile/toolchain digest
`c678cef5d9dd04056625a45d399846499a597dd5ed6377dc148273ffb8cb7b7a`
and SHA-256
`889f6d6817b8b491c4852961007bf080654386687c23a6ffb1f687c5da16c654`.
Formatting, UTF-8/LF/final-newline and diff checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestCandidateAtlas(EffectCompleteness|BorrowCompleteness)$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/(low|optimized)/^effects-'
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlas(GenericProduction|SDKVectorProduction)$/^optimized$/(used|unused|pop)/^source$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
# Set FAIRYC_ATLAS_CLI_ARTIFACTS to an absolute local directory for image reuse.
go test -mod=readonly -count=1 -timeout 10m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/^optimized$/^(impact-families|catalog-ordinary|pack-ordinary|pack-sources|pack-sdk|impact-sdk-vector)$'
go test -mod=readonly -count=1 ./internal/atlasmap ./internal/contextpack ./internal/semanticsnapshot
go vet -mod=readonly ./internal/cli
# In examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

## Failed attempts and remaining scope

The first header command failed because the new native fixture inferred i32 for
two conditional counters passed as usize. Explicit usize bindings fix that
fixture; the final effects root passes both profiles. The first new source
command failed at its grouped method receiver, stage/code `3/9`; the source now
uses the already admitted named shared receiver and passes both profiles. This
does not claim that grouped method receiver support was added.

The first full CLI command built its image successfully but could not start it:
the invoking shell supplied a relative retained-artifact directory, then the
verifier changed its process working directory to a fixture. The replay supplies
the actual absolute artifact path and reauthenticates source/profile/toolchain
and image digests. The failed enclosing commands are not passes.

Native Linux remains batched. No full Go suite, full original source-version
corpus, complete all-group low/optimized CLI acceptance, consecutive self-build,
clean-machine Windows distribution or same-commit hosted CI is claimed here.
The broader implementation queue remains active. No dependency, public CLI/SDK
ABI, database, repository management, runner or release policy changes; no push,
dispatch, release or deployment.
