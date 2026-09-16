# Bounded nested-loop cost diagnosis

This is diagnostic evidence, not an optimization, release gate or runtime loop
execution test. Baseline compiler sources: `facfa8ba6f016e5215802703cf01977c80e3c90a`.
No compiler, parser, semantic algorithm or input admission limit changed.

## Reproduction and measurement boundary

The configured `fairyc_go_test_loop_cost` intent runs
`TestWindowsFairyParserLoopCost` independently of the ordinary canonical corpus.
It reuses the same native candidate compilation helper, once per low/optimized
profile, and measures depths 0, 4, 8, 12 and 16, with up to three samples each.
The source is exactly this template with `N` opening/closing loop pairs:

```text
module main;fn i32 f(){<while(false){ repeated N times>let i32 x=1;<} repeated N times>return 0;}
```

At depth 16 this is only 269 bytes. A concrete shallow example is:

```fairy
module main;fn i32 f(){while(false){while(false){let i32 x=1;}}return 0;}
```

Measured intervals are native process launch + input + parse/write + output,
private adapter conversion, Go frontend parsing/lowering, semantic analysis of
the adapted AST, and semantic analysis of the reference Go AST. Compilation,
source construction, equality checks and log formatting are outside the intervals.
Each pair of semantic results must match completely and contain no diagnostics
or unsupported features. The input functions themselves are never executed.

Escalation stops after any semantic sample reaches 500ms; this is a diagnostic
sampling stop, not a latency assertion or an interruption of an in-flight call.
Maximum depth is always 16. The dedicated Go test process has a 60-second timeout
and the command wrapper a 90-second timeout. The existing 120-second regression
intent and parser/adapter limits are unchanged.

## Observed local result (2026-09-16)

Windows AMD64, Go 1.26.4 as reported by the running test. Each row is the median
of three samples, in milliseconds. Values are observations, not portable budgets.

| Native profile | Depth | Source bytes | Native process | Adapted AST semantics | Go AST semantics |
| --- | ---: | ---: | ---: | ---: | ---: |
| low | 8 | 157 | 41.576 | 1.105 | 0.618 |
| low | 12 | 213 | 44.823 | 13.899 | 16.782 |
| low | 16 | 269 | 44.091 | 228.884 | 192.815 |
| optimized | 8 | 157 | 33.048 | 0.556 | 0.553 |
| optimized | 12 | 213 | 37.870 | 8.351 | 10.436 |
| optimized | 16 | 269 | 38.154 | 168.294 | 169.249 |

Adapter and Go frontend samples were at most 0.652ms and 0.559ms respectively.
Some short intervals were reported as zero: this is timer resolution/observation
granularity, not evidence of zero work. The full diagnostic test passed in 12.747s.
Native profiles change the candidate executable, not the Go semantic analyzer;
differences between their Go timings must not be attributed to native optimization.
Adapted analysis is always measured before reference analysis. Cache/GC effects,
host contention and only three samples limit cross-row precision. This initial
timing run collected no CPU, heap or allocation profile. The later CPU attribution
run below is separate evidence, with profiling overhead.

## Localization and source evidence

- Native parsing/process cost did not show the depth-dependent increase seen in
  semantics; adapter conversion and Go parsing remained below the timing scale
  of the deeper semantic samples.
- Both independently built ASTs show the same increase. This rules out the
  private adapter as a necessary trigger for the measured slowdown.
- The depth-12 to depth-16 reference-semantic medians grew approximately 11.5x
  and 16.2x while source size grew only 1.26x. This identifies a semantic-analysis
  growth problem in the sampled range, not a minimum failing depth or a measured
  timing prediction for depth 30.
- In [borrow_flow.go](../../internal/semantic/borrow_flow.go), `evaluateWhile`
  evaluates the body during convergence probing and again after convergence.
  Nested loops repeat both traversals even when the outer analyzer is probing;
  `probeDepth` suppresses observable facts rather than eliminating nested work.
  For the one-probe case this structure admits a recurrence of at least
  `T(d) = 2*T(d-1) + local work`. This is source-level mechanism evidence, not a
  measured call count or proof that this pass is the only/dominant contributor.

The previous depth-30 corpus attempt exceeded its 120-second command timeout.
This diagnosis deliberately does not rerun that input or increase the timeout.
The smallest retained growth witness here is the bounded 8/12/16 depth series;
global minimality is not claimed. The CPU attribution below narrows the next
optimization target; any change must preserve convergence and diagnostic behavior. Caching,
fixed-point changes, constant-false pruning and admission-limit changes remain
outside this change.

## CPU attribution (2026-09-16)

Compiler baseline: `c85e4076c60573669c34c0bb9eb2ca93f3a19904`. Only the diagnostic
test adds `pprof.Do` labels: `phase=semantic`, `depth`, and `ast=adapted/reference`.
Production semantic code is unchanged. The same bounded test and sampling stop
remain in place. The new `fairyc_go_profile_loop_cost` intent retains the existing
60-second Go / 90-second wrapper timeouts and writes only ignored cache artifacts:

- `.cache/loop-cost-20260916.cpu.pprof`, SHA-256
  `3bcccfe2b71781ccfb1231886f1b8bc771d99578252a76522d6728417810c009`
- `.cache/loop-cost-20260916.test.exe`, SHA-256
  `9d2f877322e2856a3ea071dbaffef4ae927aab8ba06c6b4b9fe5fd66f20f2d1b`

These are local observations, not committed release artifacts. Rerunning replaces
them, so match the hashes before comparing this recorded result.
`fairyc_go_read_loop_profile` reads cumulative functions filtered by
`phase=semantic`. `fairyc_go_read_loop_profile_detail` reads labels, relative
flat/cumulative costs and annotated sources for the two observed loop paths.
No external symbols or hosted profiler is required.

The profiled test passed in 10.795s. CPU sampling duration was 10.69s, with
18.05 CPU-seconds across all sampled goroutines. Only 1.98 CPU-seconds carried the
semantic label. Do not use percentages of the unfiltered total as semantic shares.

| Semantic-label attribution | CPU samples | Share of 1.98s |
| --- | ---: | ---: |
| `borrowFlowAnalyzer.evaluateWhile` call path (cumulative) | 1.31s | 66.16% |
| `analyzer.resultFlowWhile` call path (cumulative) | 0.66s | 33.33% |
| Depth 16 (label) | 1.86s | 93.94% |
| Depth 12 (label) | 0.11s | 5.56% |
| Depth 8 (label) | 0.01s | 0.51% |

Adapted-AST semantic samples totalled 1.03s; reference-AST samples totalled 0.95s.
The runtime allocation path `mallocgc` had 0.75s cumulative attributed cost;
`cloneBorrowEnvironment` had 0.60s and `cloneResultEnvironment` 0.24s. These are
nested costs and must not be added to the enclosing loop-path totals.

Annotated source locates two body visits at `borrow_flow.go:818` and `:846`
(convergence probe and observable pass), and at `result_flow.go:296` and `:320`
(initial body evaluation and back-edge evaluation). Both recursive paths appear
in measured CPU samples. The second result-flow visit is not the borrow-analysis
fixed-point loop; optimizing only the borrow path would leave substantial work
in result-use analysis on this fixture.

Conclusion: repeated nested-loop analysis in **both borrow flow and result-use
flow** dominates the labelled CPU samples. This strengthens the earlier static
mechanism evidence; it is not proof of an exact invocation count. Recursive
source-line cumulative entries can count the same sampled stack at multiple call
sites, and deep stacks may omit outer frames; use the function view above rather
than summing annotated-line totals. Unlabelled background GC and compiler work
are excluded, so these shares describe labelled samples, not all process CPU.
Shallow depths have too few samples for stable percentages. No heap/allocation
profile, 30-level rerun, algorithm change, memoization or timeout increase was made.

## Closed scalar borrow-probe optimization (2026-09-16)

Compared with `429245166cc20156f3d905395918693746611a45`, production borrow flow
now omits convergence probing only for a closed scalar loop with empty incoming
borrow environment, no prior loans, no pending call borrows, and no borrow-return
summary analysis. The ordinary observable condition/body pass still runs.

The classifier accepts typed `i32`/`bool` leaves, grouping, unary `+`/`-`, scalar
arithmetic/comparisons, scalar bindings, simple path assignment, blocks and nested
`while`. References, calls, other types, compound assignments, conditionals,
`for`, early returns and loop exits retain the previous path. This subset cannot
create borrow state or capture an exit, so the empty header is already a fixed
point. An analyzer-local AST classification memo avoids repeatedly classifying
nested subtrees; it does not cache environments, diagnostics or flow results.
Unknown forms and recursive cycles fail closed. The optimization does not rely
on `while(false)` constant folding and also admits scalar variable conditions.

Tests force the old probe path and compare typed results, diagnostics and borrow
facts against the optimized path. Separate guards cover active borrow state,
summary mode, references, control exits and compound assignment. The complete
semantic unit suite and native adapted/reference corpus remain the regression
checks; no language, API, ABI, SDK or diagnostic contract is changed. This is
compiler patch-level performance work, not a new compatibility-vector revision.

The unchanged labelled diagnostic ran on Windows/AMD64 with Go 1.26.4, three
samples per depth and native profile. The following are semantic wall-time
medians in milliseconds (adapted / reference AST), using the previous **profiled**
run above as the baseline, not the earlier unprofiled measurements:

| Native profile | Depth | Before | After |
| --- | ---: | ---: | ---: |
| low | 8 | 1.174 / 1.090 | 0.548 / 0.000 |
| low | 12 | 10.052 / 9.920 | 3.319 / 3.320 |
| low | 16 | 189.161 / 189.677 | 48.188 / 53.539 |
| optimized | 8 | 0.591 / 0.608 | 0.553 / 0.000 |
| optimized | 12 | 8.551 / 10.098 | 2.829 / 2.689 |
| optimized | 16 | 153.505 / 142.509 | 53.825 / 53.134 |

Zero is timer granularity, not zero work. Depth 16 improved by about 2.7-3.9x
on this fixture; three local samples are not a general performance guarantee.
Native profile flags do not change the Go analyzer binary. Both AST origins
still produce equal complete semantic results. No input function is executed.

The profiled test passed in 10.154s. Of 18.58 total sampled CPU-seconds, 0.63s
carried the semantic label (previously 1.98s). `resultFlowWhile` accounts for
0.62s cumulative, 98.41% of labelled samples; borrow loop probing no longer
appears in sampled attribution, which does not mean literally zero cost.
Depth 16 accounts for 0.57s, depth 12 for 0.05s. Result-use analysis still visits
the nested body twice and remains the next separately scoped optimization target.
Neither this change nor these samples establish linear overall semantic cost.

New ignored artifacts preserve the baseline pair:

- `.cache/loop-cost-20260916-scalar.cpu.pprof`, SHA-256
  `83f562228bcc22fc881ff540b542d59ce46b442b0cc74622fc6e8663d1ca3fc2`
- `.cache/loop-cost-20260916-scalar.test.exe`, SHA-256
  `0c982aa517713432d69670575329abbf0727772e8a45d6ae4efa170bbf2f7e1a`

The profile intents now use these names. No timeout/admission-limit increase,
depth-30 retry, result-use optimization or full release validation is included.

Validation passed: Go formatting, `internal/genericsource` and
`internal/semantic` tests, both profile readers, and the native body-tree suite
(53.951s). Each native profile retained 115 semantic comparisons, 58 unsupported
shapes, 50 malformed transports, seven producer-budget rejections and 112
canonical bodies. Full release validation is deferred because this increment
changes only borrow-probe selection and no release artifact is being published.

## Closed scalar Result back-edge optimization (2026-09-16)

Baseline: `b7241dfcb0f78f2ddaf6c4adc639bbeece585fad`, including the borrow-probe
optimization above. `resultFlowWhile` now retains the first condition/body visit
but skips the second body visit for the same closed typed scalar subset when
the incoming Result environment is empty. The common structural classifier lives
in `internal/semantic/scalar_loop.go`; borrow and Result analyses own separate
classification memos. The borrow guard and admitted scalar operators are unchanged.

The subset cannot produce, transfer or handle Result obligations, emit handling
facts, or capture loop exits. With an empty incoming environment, every branch
state remains empty; the omitted second visit therefore adds neither state nor
diagnostics. References, calls, Result values, conditional control flow and exits
fall back to the existing path. Even handled entries in a nonempty environment
retain that path. Only source-typed while nodes qualify: synthesized `for`
wrappers are explicitly excluded. No constant-false pruning is involved.

Tests compare the flow outputs with the second visit forcibly restored, including
pending Result parameters, overwrite, break/continue, early return, handling and
propagation fixtures. Tests also check fast-path admission and conservative guards.
The existing borrow comparison tests exercise the shared-classifier extraction.

The unchanged Windows/AMD64 diagnostic used three samples per native profile and
depth, with the same Go runtime and timeouts as the previous run. Semantic wall-time
medians in milliseconds (adapted / reference AST) were:

| Native profile | Depth | Before | After |
| --- | ---: | ---: | ---: |
| low | 8 | 0.548 / 0.000 | 0.000 / 0.537 |
| low | 12 | 3.319 / 3.320 | 0.000 / 0.000 |
| low | 16 | 48.188 / 53.539 | 0.000 / 0.556 |
| optimized | 8 | 0.553 / 0.000 | 0.000 / 0.000 |
| optimized | 12 | 2.829 / 2.689 | 0.000 / 0.000 |
| optimized | 16 | 53.825 / 53.134 | 0.000 / 0.568 |

Zero denotes timer granularity, not zero work. Every depth-16 sample was between
0 and 1.009ms. This shows removal of the observed large nested-loop cost on this
fixture, but cannot support an accurate speedup ratio or general scalability claim.
Full semantic results still match between adapted and reference ASTs.

The profiled test passed in 8.786s. Only 0.01 CPU-seconds carried the semantic
label, compared with 0.63s before this change. Neither optimized loop path appears
in that sample. One sample is insufficient to rank remaining hot paths, including
the resource-flow frame it happened to observe. The detailed annotated-loop
profile read was skipped because there were no samples for those loop functions.
Native process startup and compilation remain outside semantic timings.

Ignored profile artifacts use separate names to preserve both earlier baselines:

- `.cache/loop-cost-20260916-result.cpu.pprof`, SHA-256
  `1c0b597fc1e1193d07fdeae1cc9d0d131269fb50a1f28d92634a813c46cd1b0d`
- `.cache/loop-cost-20260916-result.test.exe`, SHA-256
  `f537b7329c8df9e87801be4d9974a3f4d01358ddd480621ffd768269fd7eeb72`

This is compiler patch-level performance work, with no language, API, DB, ABI,
SDK or compatibility-vector change. Admission limits and timeout budgets are
unchanged. Depth 30 and full release validation remain outside this local
increment; general Result-bearing loops still use the previous algorithm.

Validation passed: Go formatting, generic-source and semantic unit tests, the
profile diagnostic and cumulative reader, and the native body-tree suite
(43.798s). Both native profiles retained 115 semantic comparisons, 58 unsupported
shapes, 50 malformed transports, seven producer-budget rejections and 112
canonical bodies. Full release validation was not run because this increment
does not publish a release or change its contracts.

## Thirty-loop regression restored (2026-09-16)

On compiler baseline `5be39d0679849e1d03adc20137570f1194e78b90`, the ordinary
native scalar corpus again uses 30 nested `while(false)` bodies around
`let i32 x=1;`. This replaces the temporary one-loop/58-prefix depth witness.
The initializer remains at wire depth 63. Wrapping the restored input in a 31st
loop still triggers producer-budget rejection; no adapter or producer limit
was raised.

The existing body-tree intent passed in 50.484s under its unchanged 120-second
wrapper timeout. Both low and optimized native profiles compared the restored
input's AST fields/summaries and complete semantic results against the Go parser,
with no diagnostics or unsupported results. Each profile retained 115 semantic
comparisons, 58 unsupported shapes, 50 malformed transports, seven producer-budget
rejections and 112 canonical bodies. The duration is for the entire suite,
including compilation and native process startup, not an individual loop timing.

This closes the original timed-out scalar regression with local evidence. It does
not establish performance for Result-bearing loops or unrestricted control flow.
Only test input and this report changed in the compiler repository: no production
code, grammar, API, ABI, SDK or version revision changes. The bounded 8/12/16
profiling test is unchanged and was not rerun; full release validation is omitted
for this test-only increment.
