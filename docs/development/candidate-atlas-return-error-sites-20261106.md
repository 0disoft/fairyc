# Fairy Atlas Result-propagation collector, 2026-11-06

Part of the complete implementation queue. This closes private collection only;
Complete-bound map production, counts, completeness and actual command admission
remain separate required work. Native Linux stays batched.

## Source observation

Owner: `docs/atlas/semantic-snapshot-v0.md`. The Go prerequisite independently
fixes the source fact as the single written question byte, not the complete
operand, an explicit `.err(...)`, or the implicit return produced by lowering.

`project_atlas_return_error_sites` accepts owned IR and lowers it through the
existing verifier to a private CFG. It visits each reachable block once and
observes the Result-tag branch only after its operand completes. The branch's
source/target and expression kind/span must agree. The retained half-open span
is `[expression.end - 1, expression.end)`. Terminal operands and earlier traps
do not reach that branch. Explicit error constructors add no observation.

Each frame retains sorted, duplicate-free sites. Standalone and batch APIs keep
independent caller work/count policies, the existing maximum of 65,536, and
frame ownership. Traversal is O(B+E+S log S) time and O(B+S) auxiliary space for
B CFG blocks, E edges and S raw observations, excluding the ordinary verifier.
Invalid IR/frame/count input returns 9, exhausted work/count returns 3 and
allocation failure returns 2. No graph or supplied observed-site list grants
collection authority. Complete-owned source binding remains the producer's job.

## Windows evidence

`TestAtlasReturnErrorCollectionSources` independently admits nineteen Go sources
in 1.163s. They contain the original fourteen oracle cases plus parenthesized
operands, grouped propagation, logical conditions, concrete generic calls and
a concrete cleanup owner. The native comparator retains all ordinary and
specialized function/cleanup owners, compares exact original Go spans, checks
single/batch agreement and verifies unchanged source trees with empty PATH.

`TestCandidateAtlasReturnErrorSites` passes Windows low/optimized in 406.107s,
including 327.70s/64.02s profile subtests. Nineteen sources and twenty controls
per profile execute 78 unique native compiler processes without selected skips.
Controls cover empty/nonempty zero cap, invalid/past frames, invalid maximum,
and just-below/exact/just-above work and count limits for both APIs.
OOM itself is not injected. These are concurrent local timings, not a benchmark.

The authored unlocked and locked checks pass on 679 source files and 8,479 MIR
functions. Scoped CLI vet, formatting, UTF-8/LF/final-newline and diff checks
pass. Go map/pack/snapshot implementation inputs are unchanged; their previous
allocation-production regression results are reused, not counted as new runs.

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/cli -run '^TestAtlasReturnErrorCollectionSources$'
go test -mod=readonly -count=1 -timeout 10m -failfast -v ./internal/cli -run '^TestCandidateAtlasReturnErrorSites$'
go vet -mod=readonly ./internal/cli
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

Logs under `.cache/`: `atlas-return-error-collector-sources-20261106.log`,
`atlas-return-error-collector-native-20261106.log`,
`atlas-return-error-collector-check-20261106.log`,
`atlas-return-error-collector-locked-20261106.log` and
`atlas-return-error-collector-vet-20261106.log`.

Only the private collector is new: `returns-error` remains unavailable in
produced maps. Borrow collection/production, remaining source/command work and
all original self-build, distribution/clean-machine, same-commit hosted and
Linux gates stay open. No stable API/SDK ABI, DB, dependency, map/cache schema,
default timeout, runner or repository-policy change; no push or dispatch.
