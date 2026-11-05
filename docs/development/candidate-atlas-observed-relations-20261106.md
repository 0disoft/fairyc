# Fairy Atlas observed relations, 2026-11-06

Part of the complete implementation queue. Existing source production collects
calls and drops and declares other families unavailable. A checked collector
alone must not promote that production/completeness scope or finish the broader queue.
Native Linux remains batched; no push, dispatch or release is authorized here.

## Required sequence

- [x] Verified body/cleanup IR and CFG drop-site collection, canonical ordering,
  source ownership, independent Go comparisons and exact work/count refusals.
- [x] Complete-bound production of those sites in v1/v2/v3 function and concrete
  cleanup summaries, correct record counts and relation-specific completeness.
  Refresh source/current/join comparisons without learning expected facts from
  candidate output.
- [x] Finish the six selected authored-main pack/catalog/impact command groups
  against independently retained drops under both Windows profiles at `1e4b9248`;
  see `docs/ops/windows-candidate-atlas-drops-cli-20261106.md`.
- [x] Verified CFG trap-category collection with independent Go function and
  concrete cleanup facts, canonical sets and exact work/count refusal controls.
- [ ] Complete-bound trap production, summary/header accounting, completeness
  and actual command comparisons. A private collector is not production evidence.
- [ ] Remaining unsafe, move, Region allocation, error-propagation and borrow
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

The collector receipt above predates the production connection below; it alone
does not attest map/command coverage or finish the implementation queue.

## Complete-bound drop production

`project_atlas_calls` now writes the collected canonical sites into ordinary,
concrete generic and concrete resource-cleanup summaries. The retained source
module qualifies every operand span. Checked intrinsic leaf implementations
without executable frames remain empty; this is the existing witnessed scalar
SDK take-slot leaf, not permission to invent a missing body. Other missing
executable owners still reject.

Each package-summary build obtains one owned `collect_all` result from the
immutable IR. Its existing `lower_all` port proves the whole IR once and checks
every resulting CFG; only locally constructed graphs reach the private
traversal. For F summarized frames this reduces whole-IR proofs from F to one
per package-summary build. Flat owned site/range vectors retain each frame's
canonical spans with a 65,536-site batch ceiling and the shared work ledger.
No public entry accepts a caller-supplied CFG or detached observation authority.
The collector corpus compares every emitted owner with both its single-frame
and batched result, including eight extra batch budget/count controls per
profile.

Summary accounting retains separate call and drop counts. Header relation
inventory includes both, including specialization and cleanup summaries; the
existing call-only traversal budget is unchanged. All source-selected map
versions report `drops` exact and remove only its unavailable omission. As in
the independent Go snapshot, non-call completeness resolved/external/unknown
counters stay zero: the actual occurrence inventory belongs to summaries and
the header relation count. Other uncollected families remain unavailable.

Source-version comparisons retain original Go drop sites and independently
count calls plus drops. Added ordinary/conditional-trap and generic-cleanup
fixtures exercise concrete spans and counts across all admitted versions. Pack
oracles likewise retain original drops while clearing only other uncollected
families; no expected body fact or inventory is learned from candidate output.
The actual pack-source CLI corpus includes a cleanup body with a local resource,
and SDK completeness comparisons now require the exact drop class.

## Final-source native production validation

`go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli
-run '^TestCandidateAtlas(DropSites|SourceVersion)$'` passes in 844.528s.
Log: `.cache/atlas-drop-batch-native-20261106.log`.

| Root | Windows low | Windows optimized | Scope |
| --- | --- | --- | --- |
| DropSites, 290.63s body | PASS, 226.00s | PASS, 52.47s | 66 native processes; fifteen source fixtures and eighteen controls per profile; single/batch and independent Go observations |
| SourceVersion, 553.76s body | PASS, 471.37s | PASS, 69.12s | 76 native processes, 94 independent current maps; eleven families, v1/v2/v3 admission, exact drop spans/inventory/completeness, current queries and joins |

The previously timed-out low SDK-unused/v3 case passes in 59.88s with the same
three-minute per-process limit; used SDK Vec passes in 61.19s. Optimized cases
pass in 2.41s/2.44s respectively. Concurrent selections were running, so these
are execution observations, not isolated performance benchmarks.

Authored check/locked pass with two instances, 671 files and 8,330 MIR functions.
Scoped CLI vet, Go formatting, UTF-8/LF/final-newline and diff audits pass.
Selected existing Go query/pack command regressions pass: atlasmap 0.287s,
contextpack 0.172s, CLI 4.975s (`atlas-drop-go-query-regression-20261106.log`).
The experimental CLI contract and source index now describe collected drops;
stable CLI flags, grammar, SDK/runtime ABI, dependency, DB, runner and hygiene
policies are unchanged.

The separate final-source command receipt now passes the six selected
authored-main pack/catalog/impact groups under both profiles at `1e4b9248`;
see `docs/ops/windows-candidate-atlas-drops-cli-20261106.md`. This factory
receipt alone does not attest those commands. The earlier broad parent timeout
in `.cache/atlas-drop-batch-cli-20261106.log` remains a failed observation.
Full CLI/generated consumers/self-build/
distribution/hosted/Linux gates and the other observed relation collectors
remain open.

The initial per-frame production attempt failed the unchanged three-minute
SDK low limit (`atlas-drop-production-versions-20261106.log`), after 24 current
maps had passed. Its repeated whole-IR proof was replaced by the existing
verified batch port, not by widening the timeout. The initial optimized actual
CLI selection passed in 318.892s (`atlas-drop-production-cli-optimized-20261106.log`)
but predates that correction and does not attest the final source.

## Trap-category collector

`project_atlas_traps::collect` and `collect_all` construct their own verified
CFGs from immutable body/cleanup IR. The batch entry performs the existing
whole-IR proof once. No public entry accepts a graph, a source-authenticated
summary or a caller-provided category set as executable authority.

Reachable assertion blocks, checked scalar unary/binary and compound operations,
and array/view index/slice evaluations contribute the six Go MIR categories:
`assertion-failed`, `bounds`, `division-by-zero`, `integer-overflow`,
`invalid-shift`, `signed-division-overflow`. Division overflow applies only to
i32/i64. The normalized minimum integer literal is not unary arithmetic.
Ordinary branches retain both CFG edges; a direct literal-false assertion stops
the continuation exactly as Go lowering does. Callee categories are not merged
into callers. Category names form a sorted unique set, not a list of source sites.
The iteration CFG step also retains Go's implicit bounds check and checked
usize increment, including empty views and bodies that terminate with a trap.

Each frame uses a six-bit mask. Traversal costs O(B + E) work and O(B) temporary
space for B blocks/E edges, apart from existing separately bounded verification
and CFG lowering. Batch storage adds one mask per frame. Independent caller
work and per-frame 0..6 distinct-category limits reject without partial results.
Invalid frame/count inputs return 9, exhausted work/count returns 3, and
allocation failure returns 2. Existing compiler/process timeouts are unchanged.

`TestCandidateAtlasTraps` uses actual source-owned ordinary, generic and concrete
cleanup bodies against independent Go snapshots. It compares single and batch
collections, checks owner coverage and unchanged workspace bytes, and exercises
one-short/exact/one-more work/count limits. Native Windows replay passes below;
the trap class remains unavailable in production until the separate connection
and its source/current/query/CLI comparisons are complete. Native Linux and all
broader/final implementation gates stay open.

### Final trap collector validation

`go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli
-run '^TestCandidateAtlasTraps$'` passes in 367.888s (body 367.76s;
low 301.80s, optimized 53.17s). Each profile runs 28 source fixtures and eighteen
frame/work/count controls: 92 native collector processes, no selected skips.
Log: `.cache/atlas-traps-native-iteration-20261106.log`.

Coverage includes all six categories in one function, signed/unsigned division,
remainder, shifts, unary/binary/compound arithmetic, the normalized i32 minimum,
ordinary/grouped/constant assertion terminals, conditional and short-circuit
edges, array reads/stores, slicing, borrows without traps, loop exits, ordinary/
empty/trapping iteration, defer captures, generics and concrete cleanup.

Initial native attempts are not passes. The first rejected an ambiguous static
borrow in category-name selection (`atlas-traps-native-20261106.log`); one
static string with selected slices fixes its origin. The next exposed a test
JSON package-key mismatch (`atlas-traps-native-category-owner-20261106.log`).
The subsequent independent comparison reproduced missing implicit iteration
bounds/increment categories (`atlas-traps-native-owner-key-20261106.log`).
Go's owning expansion is `internal/ast/byte_for_lower.go`; the final collector
and the two added empty/trapping loop controls retain those facts.

Final authored `check` and `check --locked` pass with two instances, 672 source
files and 8,339 MIR functions (`atlas-traps-check-iteration-20261106.log`,
`atlas-traps-locked-iteration-20261106.log`). Scoped CLI vet, Go formatting,
UTF-8/LF/final-newline and diff checks pass. These tests execute the collector
inside a native compiler probe; they do not execute generated user programs or
establish trap map/CLI production. Public CLI/grammar, SDK/runtime ABI, cache
wire, dependencies, DB, target, timeout, runner and hygiene policy are unchanged.
The production connection and every remaining implementation/final gate stay open.
