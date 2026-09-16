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
