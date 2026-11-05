# Fairy Atlas observed relations, 2026-11-06

Part of the complete implementation queue. Existing source production collects
calls and declares other families unavailable. A checked collector alone must
not promote that production/completeness scope or finish the broader queue.
Native Linux remains batched; no push, dispatch or release is authorized here.

## Required sequence

- [x] Verified body/cleanup IR and CFG drop-site collection, canonical ordering,
  source ownership, independent Go comparisons and exact work/count refusals.
- [ ] Complete-bound production of those sites in v1/v2/v3 function and concrete
  cleanup summaries, correct record counts and relation-specific completeness.
  Refresh source/current/join/pack/command comparisons without learning expected
  facts from candidate output.
- [ ] Remaining trap, unsafe, move, Region allocation, error-propagation and borrow
  collectors, each with independent verified Go facts and production evidence.
- [ ] Remaining source families and final full-CLI/generated-consumer/self-build/
  distribution/hosted/Linux gates from the complete implementation queue.

Owners: `docs/atlas/semantic-snapshot-v0.md`, `fymap-logical-plan-v1.md`,
`fymap-query-v1.md` and their generic/aggregate refinements. Independent Go
owner: `internal/semanticsnapshot/build.go` and `generic_cleanup.go`.

## Drop-site collector

`project_atlas_drop_sites::collect` constructs a verified CFG from its immutable
body IR and visits it from the entry with a bounded owned queue. It never accepts
a caller-supplied graph or detached list as executable authority. Both edges of
ordinary conditions remain represented, matching Go MIR facts rather than
constant-folding source behavior. Direct literal-false assertion is the Go
MIR-owned terminal in `internal/mir/lower_checks.go`; grouped assertions retain
their ordinary continuation in this metadata model.

Only visited drop blocks supply cleanup operands. A cleanup request's start/end
describe the exit that schedules it; the operand's start/end describe the owned
binding or temporary, which is the Atlas site. Separate owned buffers merge-sort
by start/end and deduplicate equal spans. No package identity or source bytes are
invented: the future Complete-bound production caller must qualify each site
with its exact retained source and ordinary/specialized/cleanup owner.

Traversal uses O(B + E) work and space for B CFG blocks and E edges; sorting D
selected cleanup occurrences uses O(D log D) work and O(D) space. This excludes
the existing separately bounded IR proof/CFG lowering and graph verification.
The caller ledger charges collector entry, CFG construction work, queue setup,
edge scheduling, traversal, sorting and output selection. CFG proof retains its
existing private bounds. At most 65,536 raw sites and explicit 0..65,536 unique
output sites are accepted. Count exhaustion and caller work exhaustion return
3 without publishing partial output; malformed frame/count input returns 9.
Allocation failures return 2. No runtime, source grammar or cache wire changes.

`TestCandidateAtlasDropSites` compiles actual ordinary and aggregate Complete
programs, then compares every ordinary function and concrete function/cleanup
owner with the independent Go snapshot. The corpus includes empty, lexical and
nested scopes, branch returns, discard, ownership transfer, loop exits, direct/
grouped/conditional assertion traps, nested owning resources, owning tags,
consuming methods, generic functions and a generic cleanup body. Separate modes
exercise zero/past-end frames, exhausted work, one-short/exact/one-more work and
unique counts, and over-ceiling counts. Child PATH is empty and workspace bytes
must remain unchanged.

## Windows validation

`go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli
-run '^TestCandidateAtlasDropSites$'` passes in 252.399s (test body 252.19s;
low 179.07s, optimized 59.93s). Each profile executes fifteen source fixtures
and ten explicit frame/work/count controls: fifty native processes total.
Log: `.cache/atlas-drops-native-cfg-20261106.log`.

The earlier native attempts are not passes: the first omitted a fixture module,
the second exposed exit spans instead of operand spans, and the third exposed
continuation after a direct assertion trap. Their logs remain respectively
`atlas-drops-native-20261106.log`, `atlas-drops-native-owner-20261106.log` and
`atlas-drops-native-operand-20261106.log`. The final source corpus retains those
regressions rather than narrowing the expected Go observations.

Authored `check` and `check --locked` pass with two instances, 671 source files
and 8,317 MIR functions. Scoped `go vet -mod=readonly ./internal/cli`, Go format,
UTF-8/LF/final-newline audit and `git diff --check` pass. Native Linux, full
authored CLI, generated consumers, consecutive self-build, distribution and
same-commit hosted CI remain unexecuted for this slice. No stable CLI, source
grammar, SDK/runtime ABI, dependency, DB, runner or repository-policy change.

This collector is not connected to map production, existing header completeness
or actual commands. The rest of the sequence and complete implementation queue
remain open.
