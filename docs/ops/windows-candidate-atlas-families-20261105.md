# Windows Fairy Atlas source-family production, 2026-11-05

The actual impact producer now consumes `project_atlas_types` and
`project_atlas_schema` after the complete current-source verification gate. It
publishes ordinary/foreign functions, structs, resources and enums rather than
inferring declarations from executable frames. Unused C declarations, explicit
link names, aggregate fields/variants, zero-field resources and cleanup effects
are preserved. Constants and static assertions remain outside the map inventory.

Function counts and summaries exclude aggregate and foreign declarations.
Foreign calls retain their exact source span, C ABI and link name, and make call
coverage partial with `external-ffi`. Only resolved calls consume the workspace
call budget. Other summaries explicitly remain `candidate-atlas-not-collected`;
the output does not claim Go's richer non-call collection or generic parity.
File-path order and module-name order are sorted separately: `a/b.fy < a0.fy`
does not imply `a::b < a0`.

## Validation

`go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run
'^TestCandidateAtlasProduction$'` passes Windows MSVC low/optimized (77.39s).
Go's `DecodeV3` accepts the fresh in-memory Fairy partitions. Every produced
declaration, function key/provenance and ordered call record matches the
independent Go source oracle, including source spans, certainty, C ABI and link
name. Duplicate or unexpected partitions reject in the test. Native PATH is
empty and the workspace tree remains unchanged.

The complete candidate CLI passes Windows MSVC low/optimized (1,008.38s;
low 888.04s, optimized 95.98s), using:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/atlas-families-cli-20261105'
go test -mod=readonly -count=1 -timeout 20m -v ./internal/cli -run '^TestCandidateAtlasCLI$'
```

This executes both `verifyCandidateAtlasCommand` and the new source-family
verification through the complete authored main/dispatcher, with PATH empty and
no companion. Existing MCP, structural preview/application, old-lock refusal,
ordinary-check lock regeneration and help/unsupported-source checks also pass.
The generated C input has a 1,925,454-byte header and 23 body units, largest
2,097,103 bytes, within the unchanged 2 MiB partition allowance. Scoped
`go vet -mod=readonly ./internal/cli`, `gofmt -l` and `git diff --check` pass.

The family fixture selects 17 declarations by both visible and full IDs and
checks the complete identity vector, graph nodes and call completeness against
Go. It independently asserts the candidate's explicit non-call omissions and
generic-unavailable boundary, rather than requiring Go's richer fact coverage.
Two resolved and two foreign calls exercise acceptance at `--max-calls 2` and
rejection at `1`; every query must preserve the entire workspace tree.

Earlier command attempts failed first on independently canonical file ordering,
then on an invalid test assumption of identical Go/Fairy completeness records
and an empty-versus-null reason-list expectation.
The standalone command was not rerun after the final reason-list correction;
its same verification helpers pass in the complete CLI above. Earlier failures
are not passing observations. Identity/call comparisons remain
exact; the final assertions also require every documented candidate omission.

SDK and generic/interface/closure Atlas production, native Linux, consecutive
self-build equality, clean-machine distribution and same-commit hosted CI remain
open. Candidate source admission changes as described; no stable public API/SDK
ABI, DB, dependency, repository hygiene or runner changes.
No push, release or deployment.
