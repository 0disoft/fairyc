# Fairy Atlas observed relations, 2026-11-06

Part of the complete implementation queue. Current source production collects
calls, drops, trap categories, unsafe sites, moves and Region allocations and declares other families unavailable. A
checked collector alone must not promote that production/completeness scope or
finish the broader queue.
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
- [x] Shared verified drop/trap batch with one whole-IR proof and one CFG
  traversal, retaining both standalone APIs and independent count limits.
- [x] Complete-bound trap production, summary/header accounting, completeness
  and independent v1/v2/v3 source/current/join comparisons.
- [x] Finish the six selected actual authored-main trap command groups under
  both Windows profiles at `53728a07`. All twelve profile/group combinations
  pass without selected skips; see `docs/ops/windows-candidate-atlas-traps-cli-20261106.md`.
  A private collector or map comparison alone is not command evidence.
- [x] Verified unsafe-site collector with independent ordinary, specialized,
  concrete cleanup and pinned SDK helper observations and exact budget controls.
- [x] Complete-bound unsafe production with independent source/current/join
  comparisons under both profiles.
- [ ] Complete passing both-profile selections for the six actual unsafe CLI
  groups; four low groups retain assertions only from failed aggregate parents.
- [x] Verified move-site collector with independent original Go operands,
  scope/conditional/generic/cleanup comparisons and exact work/count controls.
- [x] Complete-bound move summaries/counts/completeness with both-profile
  original-source/current/join evidence.
- [ ] Complete the six selected actual move CLI groups at both profiles; all
  optimized groups and the two new low move-pack source cases pass.
- [x] Verified Region allocation-site collector with independent ordinary,
  concrete generic/cleanup Go facts and exact frame/work/count refusals.
- [x] Complete-bound allocation production, summaries/counts/completeness and
  independent source/current/join and actual-command comparisons.
- [ ] Remaining error-propagation and borrow collectors, each with independent
  verified Go facts and production evidence.
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
one-short/exact/one-more work/count limits. At this collector-only snapshot, the
trap class was still unavailable in production. The separate connection below
requires its own source/current/query/CLI comparisons. Native Linux and all
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

## Shared verified collection

`project_atlas_drop_sites::collect_observed_all` owns one `lower_all` result
and one private traversal per frame for both canonical drop sites and trap
category masks. The immutable IR proof runs once per batch, rather than once
for each separately invoked collector. Graph verification remains per frame.
`project_atlas_trap_kinds` contains only pure names/mask arithmetic and grants
no graph or source authority. `project_atlas_traps` retains its previous API
through explicit trap-only entries in the shared owner.

Standalone drop collection never charges trap extraction or applies a trap
count limit. Standalone trap collection never gathers or limits drop sites.
The combined factory retains separate 0..65,536 drop and 0..6 category limits,
the shared caller work meter and the existing 65,536-site batch ceiling.
Only private helpers receive graphs or partially filled buffers; public
factories publish their owned results only after the complete batch succeeds.
No public graph argument or verification-bypass flag is introduced.

Existing single/batch Go-oracle comparisons now also compare combined ranges,
spans and masks. Twelve new native controls exercise combined work limits,
category limits, empty/nonempty zero-drop limits, exact drop count and invalid
caps. The combined Windows collector command passes in 769.619s with no selected
skips: drop collection 296.92s (low 228.82s, optimized 53.86s) and trap collection
472.55s (low 399.67s, optimized 59.91s). Each profile executes fifteen drop and
twenty-eight trap source fixtures, eighteen standalone controls per collector
and twelve combined controls: 182 native probe processes total. These probes
compare compiler facts; they do not execute generated user programs.

Command: `go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli
-run '^TestCandidateAtlas(DropSites|Traps)$'`. Log:
`.cache/atlas-observed-shared-native-owned-20261106.log`. Authored `check` and
`check --locked` pass for this shared-source snapshot (673 files, 8,344 MIR
functions); scoped vet, Go format and diff checks pass. The production connection
below is separate and is not attested by this collector receipt.

Earlier shared-source attempts rejected a nested mutable borrow, a dereference
assignment and a branch ownership gap. Their check logs are respectively
`atlas-observed-shared-check-20261106.log`,
`atlas-observed-shared-check-assignment-20261106.log` and
`atlas-observed-shared-check-return-mask-20261106.log`; none is a pass. The final
owned-buffer route keeps publication after success and retains the native
controls above. Native Linux and the broader/final gates remain open. No public
CLI/grammar, SDK/runtime ABI, cache wire, dependencies, DB, timeout, runner or
repository-policy change.

## Complete-bound trap production

`project_atlas_calls` now writes sorted unique categories from the combined
verified collection into ordinary, concrete generic and concrete cleanup
summaries. Source-authenticated intrinsic leaves without executable frames keep
their witnessed empty summary; the existing missing-frame refusals remain.
Each package relation inventory now counts calls, drops and trap categories.
Only `may-trap` is newly exact and loses its unavailable omission. The remaining
observed relation families stay explicitly unavailable, so overall generation
continues to be partial. Existing wire schemas and current-source authority do
not change.

The independent Go function oracle now compares categories as well as calls and
drop spans. Counts are derived from original Go summaries, never candidate
headers. New source-version cases cover all six categories in an ordinary body,
signed/unsigned generic division, concrete resource cleanup and implicit checks
from an empty byte iteration. Original declarations/specializations, map decoding,
current-source checks and workspace joins remain part of the comparison.

`TestCandidateAtlasSourceVersion` passes in 676.152s, low 576.27s and optimized
84.26s: 100 native compiler probes and 130 independently checked current maps,
including existing ordinary/generic/aggregate/SDK cases and schema refusals.
Log: `.cache/atlas-trap-production-versions-20261106.log`. The separate ordinary,
generic and SDK Vec production roots pass both profiles in 754.834s: ordinary
81.39s, generic 136.04s and SDK Vec 537.24s (low 448.02s, optimized 75.44s).
Original maps, header/completeness, source/export/validation/join budget and
missing-coordinate refusals remain checked. Log:
`.cache/atlas-trap-production-roots-20261106.log`.
Authored check/locked pass (673 files, 8,345 MIR functions). Scoped vet, complete
Go atlasmap/contextpack regressions, format and diff checks pass.

The actual full authored-main optimized replay passes all six selected groups
in 158.813s (body 158.67s, profile 140.27s), without a Go fallback or selected
skip: impact-families, impact-sdk-vector, catalog-ordinary, pack-ordinary,
pack-sources and pack-sdk. The unchanged independent Go command oracles retain
categories and count only the explicitly collected classes. Log:
`.cache/atlas-trap-production-cli-optimized-20261106.log`.
Generated-C/profile/toolchain source digest:
`f233e33f51913e0e8d94ef24b88a6b8286883a353d30b38636e444344a10ead4`;
image SHA-256:
`6cac6540b97a0968582d20b25f6d92ac96cee10291fed4222ae446b5f2e83239`.

Low command replay also passes all six groups in bounded parent selections using
the existing image digest checks and unchanged product timeouts. The separate
receipt `docs/ops/windows-candidate-atlas-traps-cli-20261106.md` records every
selector, timing, source/image digest and local log. Final binary digests were
recomputed and agree with both retained receipts. The other 35 full-CLI groups,
remaining collectors, broader source/consumer work, consecutive self-build,
distribution, same-commit hosted CI and native Linux remain open.
No stable CLI flags, source grammar, SDK/runtime ABI, wire schema, dependency,
DB, target, timeout, runner or repository-policy change; no push or dispatch.

## Unsafe-site collector

`project_atlas_unsafe_sites::collect` and `collect_all` construct verified CFGs
from immutable body/cleanup IR. The batch entry proves the whole IR once.
No public factory accepts a graph, permission flag or detached observations as
authority. Source-relative spans acquire package/source identity only at the
separate Complete-bound publication boundary.

Reachable raw-address casts, raw-view constructors, SDK storage operations and
foreign C calls contribute sites. Foreign targets come from verified callable
contracts and use a sorted module/declaration index. Ordinary calls do not
become unsafe merely because they occur in an unsafe function or block; empty
unsafe blocks, safe borrows, scalar casts and null pointers add no sites.
Direct literal-false assertions stop traversal as in Go MIR; ordinary branches
retain both successors. Canonical spans sort by start/end and deduplicate.

The pinned Vec cleanup folds its safe `take_slot<T>` call to storage-take IR.
That caller span is not an unsafe site. A verified folded invocation plus the
program's authenticated SDK bytes supplies the original helper's unique
`storage_take<T>(slot)` span through an owned leaf record. Concrete helpers with
ordinary frames retain normal collection; frame-less implementations use that
record only with the existing checked specialization witness. Names or plan
nodes alone grant no leaf authority. A changed/missing/duplicate pinned source
pattern rejects instead of guessing a span or dropping the helper's facts.

Apart from existing separately bounded proof/CFG construction, traversal costs
O(B + E + N + C log(F + 1) + U log(U + 1)) for blocks, edges, nodes, calls,
foreign targets and raw occurrences. The target index costs O(F log(F + 1))
once per collection; the bounded pinned-source scan adds O(S) work for S SDK
bytes. Site and work limits are independent, with at most 65,536
raw sites per frame and 65,536 batch output sites. Invalid frame/count input
returns 9, work/count exhaustion returns 3 and allocation failure returns 2.
Failed factories publish no partial result; existing product timeouts remain.

`TestCandidateAtlasUnsafeSites` compares every emitted ordinary, specialized
and concrete cleanup owner with original independent Go `UnsafeSites` facts.
Single-frame and batched results agree, with separate witnessed frame-less
helper rows. SDK Vec unused/used/pop cases use the
batch path to avoid repeated whole-program proof. The fixture corpus retains
the safe-call overcount regression instead of altering Go expectations.
Native controls cover zero/past-end frames, exhausted and one-short/exact/
one-more work/count limits, over-ceiling counts and empty/nonempty zero caps.
Child PATH is empty and workspace bytes must remain unchanged.

This collector alone does not attest unsafe map/CLI production. The separate
connection below owns its independent map/current/join/command comparisons.
Native Linux and every broader/final implementation gate remain open.

### Windows unsafe collector validation

Both bounded native selections pass without selected skips:

| Exact selection after `^TestCandidateAtlasUnsafeSites$` | Package time | Profile body | Native processes |
| --- | ---: | ---: | ---: |
| `/low` | 479.689s | 463.55s | 49 |
| `/optimized` | 84.173s | 69.52s | 49 |

Command prefix: `go test -mod=readonly -count=1 -failfast -timeout 15m -v
./internal/cli -run`. Logs: `.cache/atlas-unsafe-native-leaf-low-20261106.log`
and `atlas-unsafe-native-leaf-optimized-20261106.log`. Each profile executes
29 source fixtures, eighteen frame/work/count controls and two zero-count
controls. All ordinary, specialized and concrete cleanup owners, including
frame-less pinned SDK helpers, retain original Go expectations.

Authored check and locked check pass (674 files, 8,382 MIR functions) in
`atlas-unsafe-check-leaf-20261106.log` and `atlas-unsafe-locked-leaf-20261106.log`.
Scoped CLI vet, Go formatting, UTF-8/LF/final-newline audit and diff checks pass.
Earlier syntax/probe failures, safe-call overcount and folded-SDK overcount are
not passes; their logs retain the native attempts. The interrupted
`atlas-unsafe-native-foreign-20261106.log` run is likewise not a completed pass.
The collector does not execute generated user programs or establish complete
maps/CLI, Linux, hosted CI, self-build or distribution acceptance. No stable
CLI/grammar, SDK/runtime ABI, cache wire, dependencies, DB, target, timeout,
runner or repository-policy change; no push or dispatch.

## Complete-bound unsafe production

`project_atlas_calls` now collects one verified unsafe batch for the complete
project and retains its canonical sites in ordinary, generic and concrete
cleanup summaries. The authenticated frame-less SDK helper uses its original
source leaf; caller summaries do not duplicate a folded storage operation.
`project_atlas_produce` counts unsafe sites with calls, drops and trap categories.
Only `uses-unsafe` becomes newly exact and loses its unavailable omission;
every remaining uncollected family stays unavailable and generation stays partial.

The source/current/join oracle retains independent original Go `UnsafeSites`;
pack expectations retain those sites and count them instead of clearing them.
The versioned source corpus adds raw-view/foreign-call, generic, cleanup and
safe-call-in-unsafe-scope controls. Source-backed pack commands also exercise
unsafe ordinary and cleanup declarations. No expected fact comes from the
candidate result, and no product work/count or process timeout is increased.

An earlier SDK Vec current-map join exhausted the shared JSON work budget
(`join:9/16777152`). The separate decoded-key index fix in `b9c99000` preserves
the 16,777,216 limit and exact decoded duplicate behavior; this production
selection uses that final input. Earlier failed/partial diagnostic attempts
are not acceptance evidence.

The versioned source selections pass in 789.366s (low) and 97.914s (optimized),
with 62 native processes and 83 independent current maps per profile. Logs:
`.cache/atlas-unsafe-production-versions-indexed-low-20261106.log` and
`atlas-unsafe-production-versions-indexed-optimized-20261106.log`.
Both authored checks pass (674 files, 8,384 MIR functions), with logs
`atlas-unsafe-production-indexed-check-20261106.log` and
`atlas-unsafe-production-indexed-locked-20261106.log` under `.cache`.
The refreshed ordinary/generic/SDK Vec production-root selection also passes
both profiles in 929.227s, including independent maps, current admission, joins
and the existing refusal controls. The exact selection is
`^TestCandidateAtlas(Production|GenericProduction|SDKVectorProduction)$` with
`go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run`.
Log: `.cache/atlas-unsafe-production-roots-20261106.log`. Both initial low CLI
parents hit their aggregate 20-minute test alarm, after completing catalog/
ordinary pack or the two impact groups. They remain failed commands. Source
pack and SDK pack pass separate 15-minute replays in 312.971s and 449.408s,
with identical authenticated inputs and unchanged product policies. The four
other low groups still lack separate zero-exit parents. Complete low CLI
acceptance stays pending; the unsafe command receipt distinguishes retained
assertions, failed parents and completed isolated selections.

## Move-site Go prerequisite

`TestAtlasMoveSitesOracle` fixes original source spans from fourteen independent
Go source fixtures, with v3 encode/decode before comparison. It distinguishes
copy/borrow/discard/temporary uses from binding, return, call, owning field,
Option/Result payload and switch-scrutinee transfers. The two argument-order
controls prove that a first argument's move survives a later direct assertion
trap, while a later argument after an earlier trap has no move observation.
Resource-bearing `?` transfers an obligation internally but contributes no
Go `MoveSites`; error propagation has its own observation family.

This changes the collector implementation plan: derive consumption at each
evaluated operand, including partial parent evaluation, rather than scanning
only completed calls/constructors or treating every ownership state change as
an Atlas move. Source-relative identity is `main.fy` under the manifest's source
root. Empty decoded site lists are compared semantically, not by nil/slice shape.

The Windows Go selection passes fourteen cases without skips in 1.123s:
`go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run
'^TestAtlasMoveSitesOracle$'`. Log: `.cache/atlas-move-sites-oracle-20261106.log`.
Initial probe compile/shape/source-path/syntax errors were corrected before the
passing selection and are not production defects. No Fairy move collector,
map completeness promotion, generated-program run, Linux or hosted result is
claimed by this Go-only prerequisite. The full implementation queue stays open.

## Move-site collector

`project_atlas_move_sites.fy` derives consumed complete-root paths from verified
body/cleanup IR and a private executable CFG. It marks operand evaluation, not
only completed parent calls, so a first consumed argument remains observed before
a later argument traps, while an unevaluated later argument stays absent.
Copy, explicit borrow, fresh temporary, direct discard and internal Result `?`
transfer do not become invented MoveSites. Binding, reassignment, return, owned
fields, tagged payloads, consuming receivers and switch results retain original
Go operand spans. Generic functions and concrete cleanup frames use the same walk.

Block-tail capture uses the verified scope's local obligations and cleanup exits.
The then arm of an if expression is stored as a statement block, unlike its else
block expression. The initial expanded native selection reproduced a missing
then-tail site; applying the same block-scope rule to that arm fixes it without
marking every owning conditional as a move. Nested else-if, external tail paths
with/without a scoped live obligation, and ignored/propagated blocks are compared
to independent original Go observations.

Per-frame and batched APIs agree. The batch proves the whole IR once; source
paths and spans remain private observations until a Complete-owned producer
binds them to a package. Reachability has one visited ledger per frame. Root and
type peeling keep the existing 64-level bounds; merge sorting canonicalizes
scope/site spans, with at most 65,536 retained sites and the existing 16,777,216
work limit. Invalid frame/count arguments, zero capacity on empty/nonempty input,
and one-short/exact/one-more work/count limits preserve typed refusals. No policy
limit or warning is relaxed.

The native corpus has 34 admitted sources, two explicit source-factory gap
controls and twenty frame/work/count controls per Windows profile (56 executions).
The original Go-supported implicit shared/exclusive owner method forms still
reject at the current Fairy factory with `4/1006`; explicit ref/ref-mut receiver
bindings are positive controls. These two gaps remain broader source-family work,
not empty successful move lists or a claim of full Go/Fairy parity.

Windows validation:

- Original Go sites and expanded source admission pass, 48 selected cases,
  in 3.471s (`.cache/atlas-move-oracles-expanded-20261106.log`).
- Optimized native selection passes in 71.76s, including all 56 executions
  (`.cache/atlas-move-native-final-optimized-20261106.log`).
- Low native selection passes in 368.35s, including all 56 executions
  (`.cache/atlas-move-native-final-low-20261106.log`). Both profiles complete
  112 native executions without selected skips.
- Authored `check` and `check --locked` pass for two instances, 675 source files
  and 8,417 MIR functions. Scoped `go vet ./internal/cli`, Go formatting and
  `git diff --check` pass.

Complete-bound move summary/header/count/completeness production and actual CLI
comparisons remain a separate required step. Allocation/error/borrow collectors,
unsafe low replays, broader source families and all original full-comparison,
self-build/distribution/hosted/Linux gates remain open. Native Linux stays batched;
no push or dispatch.

## Complete-bound move production

The Complete-owned summary path now writes original move spans into ordinary,
specialized and concrete cleanup functions. Relation counts include moves beside
calls/drops/traps/unsafe. Only `moves` becomes newly exact and loses its unavailable
omission; allocation/error/borrow/effects and other uncollected classes stay
unavailable, and generation stays partial. Folded SDK scalar helper leaves keep
empty move observations rather than gaining fabricated caller sites.

A first production comparison reproduced an existing call-reachability defect:
`moves-before-later-trap/v1` had five relations instead of the independent Go
four, because checking the parent call was mistaken for executing it. The failed
command is retained in `.cache/atlas-move-production-first-optimized-20261106.log`.
`project_atlas_reachability.fy` now builds one verified whole-program CFG batch
and a private per-frame completed-expression ledger. Calls are included only at
their evaluated node; deferred calls are included at their invocation node, not
capture time. Original checked-call/target guards remain, as do folded-helper
identity checks. A consumed first operand and any completed earlier calls stay
observed when a later argument traps. The ledger/traversal costs O(sum(N+B+E)) time beyond
IR/CFG construction, O(sum(N+B)) temporary/retained storage, and is metered by
the existing work budget. No caller-supplied graph can grant this authority.

Independent source/current/join comparisons cover the original nineteen source
fixtures, nine move fixtures and three call-reachability fixtures, each requested
at v1/v2/v3. Named missing source families keep their existing explicit refusals.
Go function comparisons retain the original MoveSites, and command pack oracles
now preserve moves instead of stripping them. No candidate output teaches those
expectations. Ordinary/cleanup move pack fixtures are registered in the actual
full authored CLI verifier.

Windows move production validation:

- The corrected first optimized move selection passes in 79.88s: 27 native
  processes and 46 independently prepared current maps.
- Final full optimized source selection passes in 110.65s: 98 native processes
  and 147 independent current maps; `.cache/atlas-move-production-final-optimized-20261106.log`.
- Final low selections pass in three bounded parents: moves/reachability in
  303.16s (36 native processes, 64 current maps), SDK in 466.28s (9/15), and
  ordinary in 350.16s (53/68). Logs are
  `.cache/atlas-move-production-final-low-{moves,sdk,ordinary}-20261106.log`.
  Their disjoint union matches all 93 optimized source/version selections.
  Both profiles complete 196 native processes and 294 independently prepared
  current maps, with no selected skips. Every parent retains its twelve-minute
  Go test limit; product limits and verifier assertions are unchanged.
- Authored check/locked pass (676 files, 8,423 MIR functions). Go atlasmap and
  contextpack regressions pass in 3.717s/0.257s; scoped vet/format/diff pass.
- Actual full-main pack-sources/move-sites and cleanup-move-sites pass under
  optimized in 115.39s and low in 118.44s: eight native invocations per profile.
  Logs: `.cache/atlas-move-production-cli-first-{optimized,low}-20261106.log`.
- All six selected actual optimized CLI groups pass in 93.43s using the same
  authenticated generated-C/profile/toolchain/image digest:
  catalog-ordinary, pack-ordinary, pack-sources, pack-sdk, impact-families and
  impact-sdk-vector. Log: `.cache/atlas-move-production-cli-final-optimized-20261106.log`.
  Complete low command replay remains a separate gate.

The full low six-group command replay remains open; two selected low source
cases do not attest the complete pack group. Other full CLI groups, unsafe low
replays, remaining collectors/source families and every original final acceptance
gate remain open.
Native Linux remains batched; no push, dispatch, release or deployment.

## Region allocation Go prerequisite

`TestAtlasAllocationSitesOracle` fixes original complete call spans from fourteen
Go source/codec controls: empty, single/repeated placement, branches, loops,
direct lexical ownership, zeroed/copy bytes, both partial argument orders and
three trapped allocation operands. SystemHeap/Arena creation is not reported as
a Region allocation. This selects Region placement operations, not all foreign
or SDK allocation calls.

The first lexical probe used a borrowed Arena alias, outside the current Go MIR
source profile. Its failure remains in
`.cache/atlas-allocation-go-oracle-first-20261106.log`; the positive lexical case
uses the already admitted direct ref-mut owned local. This does not add alias
or computed-Arena execution support.

The corrected source corpus reproduced a separate real lowering defect:
`r.place({assert(false);7})` returned internal compiler failure instead of
preserving its already terminated operand. `lowerRegionPlace` now uses the same
post-operand termination guard as ordinary calls, before requiring an executable
value or creating a placement. The shared lowering also covers zeroed-byte count
and copied-byte view operands. Nonterminated invalid operands still reject.
Failure log: `.cache/atlas-allocation-go-oracle-direct-20261106.log`.

All fourteen Go controls pass in 2.15s. Three actual Windows programs per
low/optimized profile compile and execute: all six retain empty output and the
original assertion-failed receipt/status. These are genuine generated SDK/Arena
executions, not snapshot-only checks. Combined log:
`.cache/atlas-allocation-go-oracle-native-20261106.log` (11.640s parent).
The complete Go MIR package passes in 1.208s; scoped vet, format and diff pass.
Native Linux stays batched. No API/ABI, grammar, SDK/version, cache format,
dependency, default timeout, target, runner or repository-policy change.

This prerequisite does not implement the Fairy allocation collector, promote
allocation completeness or complete map/command parity. Those steps and every
original wider/final gate remain required; no push or dispatch.

## Region allocation-site collector

`project_atlas_allocation_sites` lowers independently verified detached IR into
private CFGs and visits executable blocks from each entry. Only reached typed
Region place/zeroed-byte/copy-byte operations create observations. Arena or
SystemHeap construction creates none. Terminal operands and earlier call
arguments do not invent a later placement; a completed earlier placement stays
observed when a later argument traps. Complete-bound source/package authority
and map publication remain the production caller's separate responsibility.

Both standalone and batch APIs canonicalize full original call spans with a
bounded merge sort. For B blocks/E edges/A raw sites, traversal uses O(B+E)
work/space and canonicalization O(A log A) work and O(A) space, excluding the
separately bounded immutable-IR proof and CFG construction. Standalone lowering
proves the program; batch lowering proves it once and independently verifies
every resulting graph. No public collector accepts a caller-supplied CFG or
detached observation list. Raw sites and total retained batch sites are bounded
at 65,536; the caller's per-frame unique maximum is independent. Frame/count
malformation returns 9, caller work/count exhaustion 3, allocation failure 2.
No partial collection is published on failure.

`TestCandidateAtlasAllocationSites` compares all ordinary/concrete function and
cleanup owners to independent original Go spans, including every selected SDK
frame. Standalone/batch comparison is repeated for root-source frames; retained
SDK frame comparisons do not redundantly rerun whole-program proof for every
unrelated helper. Twenty actual source cases include zeroed/copy bytes, both
partial argument orders, all three terminal operands, after-direct-trap,
nested placement, Result propagation, loop exits, generics and concrete cleanup.
Twenty separate controls retain empty/nonempty zero-count behavior, bad frame
IDs, one-short/exact/one-more work and unique counts, and over-ceiling counts.
PATH is empty in every generated compiler process and workspace bytes stay
unchanged. There is no Go fallback.

The initial low parents failed after 60.01s in two source cases, with no output:
`.cache/atlas-allocation-collector-low-corpus-a-20261106.log` and
`atlas-allocation-collector-low-corpus-b-20261106.log`. Isolated `two-places`
passed in 51.93s (76.724s parent). The SDK-heavy compiler check was incorrectly
using the package-fetch helper's one-minute process deadline. The collector
now uses the existing three-minute Atlas SDK process helper, including explicit
timeout diagnostics. Product/compiler limits and warning levels are unchanged.
Four bounded fifteen-minute parent commands preserve all source and control
assertions; together they pass all forty unique low observations without skips:

| Selected low group | Parent seconds | Log under `.cache/` |
| --- | ---: | --- |
| first ten source cases | 619.314 | `atlas-allocation-low-corpus-a-final-20261106.log` |
| remaining ten source cases | 646.482 | `atlas-allocation-low-corpus-b-final-20261106.log` |
| standalone and zero-count controls | 671.889 | `atlas-allocation-low-controls-a-final-20261106.log` |
| remaining batch controls | 449.834 | `atlas-allocation-low-controls-b-final-20261106.log` |

The matching forty optimized observations pass in 157.031s, log
`atlas-allocation-collector-final-optimized-20261106.log`. These are concurrent
correctness timings, not isolated compiler-speed benchmarks. The source corpus
also passes the Go preparation oracle; subsequent Go backend fixes are checked
in the current-bootstrap renewal below rather than silently attributed to these
earlier compiler images.

After the separate Go Result/C-backend corrections, a fresh optimized compiler
passes all forty source/control observations in 159.122s. A fresh low compiler
passes the trapped-byte-view source and standalone/batch exact-work controls
in 194.469s (three selected observations). Logs:
`atlas-allocation-current-bootstrap-optimized-20261106.log` and
`atlas-allocation-current-bootstrap-low-20261106.log`. These renewals are
explicitly separate from the earlier full low matrix: they do not claim that
all forty low cases were repeated with the changed Go bootstrap. A log audit
confirms the full low and renewed optimized selected case sets are identical,
with twenty source cases and twenty controls each and no duplicate or missing
observations. The forty-three renewed executions are not new source coverage.

The final original Go allocation oracles pass in 5.307s, and scoped CLI vet,
formatting, UTF-8/LF/final-newline and diff checks pass. Final authored unlocked
and locked checks pass with two instances, 678 source files and 8,453 MIR
functions; the lock is updated and then verified. Logs:
`atlas-allocation-collector-commit-check-20261106.log` and
`atlas-allocation-collector-commit-locked-20261106.log`.

No schema/API/ABI, grammar, SDK/version, cache format, dependency, DB, product
timeout, target-maturity, runner or repository-policy change is made.
This closes the private collector only: allocation summaries, header inventory,
completeness, current/join/command production and all original broader/final
gates remain open. Native Linux stays batched; no push or workflow dispatch.

## Result propagation Go prerequisite

`TestAtlasReturnErrorSitesOracle` fixes thirteen independent source/codec
observations before the Fairy collector is derived. A propagation site is the
single original `?` byte, not the full operand or its implicit error return.
Explicit `.err` construction creates no propagation site. The corpus retains
local/call operands, repeated/nested propagation, branches/loops, both partial
argument orders, an earlier direct trap and terminal trap/return operands.

The trapped operand reproduced exit 4, `internal compiler invariant failed`,
because Go MIR demanded a result value after its operand had already terminated.
`lowerPropagate` now preserves that termination before demanding a live value
or creating any result switch, cleanup or observation. The related Result
construction guard likewise preserves an already terminated payload. Invalid
operands on live paths still reject. Failed attempts remain in
`.cache/atlas-return-error-trapped-operand-before-20261106.log` and
`atlas-return-error-oracle-native-20261106.log`.

The native partial-argument case exposed a second required backend correction:
MSVC optimized emission inferred the callee's constant Result tag, then reported
C4702 on its caller's valid MIR continuation. `result_switch.go` now snapshots
the tag once through a private volatile write/read under `_MSC_VER` before
selecting the unchanged success/error edges. Other C compilers retain the
ordinary tag reads. This has an explicit two-access volatile cost per Result
switch on MSVC; it changes neither source/ABI/IR/cache contracts nor warning
policy. The failed native attempt and inspected generated unit remain in
`.cache/atlas-return-error-operand-guard-20261106.log` and
`result-payload-debug.c`. The temporary diagnostic Go test was removed.

The thirteen source/codec controls and six actual Windows low/optimized native
executions pass together in 9.775s. The native programs retain assertion trap
receipts, empty stdout/stderr and the explicit early error-return exit 3.
Command: `go test -mod=readonly -count=1 -failfast -v ./internal/cli -run
'^Test(AtlasReturnErrorSitesOracle|WindowsResultPropagationOperandExit)$'`.
Log: `.cache/atlas-return-error-operand-native-pass-20261106.log`.

A separately observed cleanup/defer-plus-propagation source initially produced
duplicate Go Atlas calls and failed canonical encoding. The failed observation
is retained in `.cache/atlas-return-error-oracle-native-final-20261106.log`;
the independent correction and its expanded receipt follow below.
Fairy error-site collection/map/CLI work, allocation production and all broader
and final queue gates remain open. Native Linux stays batched; no push/dispatch.

## Deferred source-call canonicalization

One deferred source invocation can appear in several mutually exclusive MIR
exit blocks, including normal return and Result propagation. Go snapshot
construction previously emitted each copy and incremented its completeness
counter before canonical sorting. That duplicated source calls and made valid
cleanup/defer programs fail strict `.fymap` encoding. Reproduction:
`.cache/atlas-deferred-result-calls-before-20261106.log`.

After the existing source-key sort, `buildFunctionSummaries` now compacts only
fully identical owned Call records and counts the resulting source inventory.
It preserves different written sites and every target/specialization/ABI/
certainty field. Conflicting records remain distinct and subject to the
unchanged strict codec; no arbitrary winner is selected. The added pass is
O(C) time and O(1) auxiliary space over the already-owned sorted C-call slice.
Executable MIR and input snapshots are not changed.

`TestAtlasDeferredResultCalls` independently requires three source calls with
resource cleanup, four with two distinct written defers, and six across two
generic specializations. It checks ordinary/concrete owners, exact source-call
and question inventories, canonical codec round trips and noninflated global
completeness. The original error-site corpus now includes the cleanup/defer
case: all fourteen source/codec controls and three new call cases pass in 1.134s.
Command: `go test -mod=readonly -count=1 -failfast -v ./internal/cli -run
'^TestAtlas(ReturnErrorSitesOracle|DeferredResultCalls)$'`.
Log: `.cache/atlas-deferred-result-calls-final-20261106.log`.

Full Go snapshot/map/pack suites pass (0.208s/2.325s/0.226s), as does scoped vet.
The preceding Result-operand correction separately passes the complete Go MIR
and C-backend suites (0.710s/74.346s) and its scoped vet. Formatting,
UTF-8/LF/final-newline and diff checks pass. Source-call multiplicity is clarified
in the snapshot contract; no schema/API/ABI, cache format, dependency, DB,
timeout, runner or repository-policy change. Fairy collector/production and
every broader/final gate remain open; native Linux stays batched, no push/dispatch.

## Complete-bound allocation production

`project_atlas_calls` collects verified Region allocation sites for the complete
program and retains each frame's canonical range in ordinary, specialized and
concrete cleanup summaries. Source coordinates use the frame owner's original
module and the full written Region call expression, including zero-byte and
byte-copy calls. A completed allocation operand survives a later trapping
argument; an allocation after an earlier terminal operand does not appear.
Arena/SystemHeap creation does not become a Region allocation observation.

`project_atlas_produce` includes these sites in header relation counts.
`project_atlas_header` marks only `allocates` newly exact and removes only its
unavailable omission. Non-call completeness counters remain zero under the
existing map contract; error propagation, borrows and other uncollected classes
remain explicitly unavailable. There is no map schema or checksum change.

The independent Go source inventory has five families: ordinary (four sites),
partial operand order (one), concrete generic (two), concrete cleanup (three),
and loop/Result (two). The first, second and fifth admit v1/v2/v3; generic needs
v2 and cleanup needs v3. `TestAtlasAllocationProductionSources` freezes all
fifteen admission/count observations before any Fairy map is read. The source
version producer then compares exact original Go spans, declarations, symbols,
specializations and relation counts, checks current-source and join admission,
and rejects unsupported older schemas without changing source bytes.

The pack oracle retains original Go allocation sites rather than replacing them
with empty arrays or learning counts from candidate output. Five new SDK pack
fixtures select ordinary, partially evaluated, generic, cleanup and loop/Result
owners through the real authored main. Each compares overview/show/around
against independently scoped Go encoders and checks unchanged workspace bytes.

The standalone Go source inventory passes in 1.375s. Full optimized source/version
production passes in 147.252s: 113 native processes and 183 independently checked
current maps, including all five allocation families. The six selected actual
optimized CLI groups pass in 267.342s, including all fifteen new allocation pack
commands. Logs: `.cache/atlas-allocation-production-sources-20261106.log`,
`atlas-allocation-production-optimized-20261106.log` and
`atlas-allocation-cli-optimized-20261106.log` in the same cache directory.

The same five allocation source families pass every v1/v2/v3 selection under
low, with no selected skips: bounded parents finish in 692.323s and 711.491s,
fifteen native processes and 36 current maps. Combined source production proves
128 native processes and 219 current maps; only the optimized parent includes
the full pre-existing source corpus. The five actual allocation pack cases also
pass low in bounded 697.183s and 1079.871s parents: thirty new actual-main pack
processes across both profiles without selected skips. The implementation source
revision is `04894e8f`. Exact commands, scope, image digests and interrupted-run accounting:
`docs/ops/windows-candidate-atlas-allocations-20261106.md`.

Full Go atlasmap/contextpack/semanticsnapshot suites pass (2.284s/0.253s/0.210s),
and scoped CLI vet passes. Both-profile source and command scope is recorded in
the allocation command receipt. Broader source/CLI/generated-consumer work,
remaining collectors, consecutive self-build, distribution/clean-machine,
same-commit hosted and native Linux acceptance remain separate open gates.
Native Linux stays batched; no push or dispatch.
