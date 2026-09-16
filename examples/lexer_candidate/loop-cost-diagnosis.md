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
host contention and only three samples limit cross-row precision. No CPU, heap,
allocation or individual semantic-pass profiler was collected.

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
global minimality is not claimed. A follow-up optimization should first attribute
semantic-pass work, then preserve convergence and diagnostic behavior. Caching,
fixed-point changes, constant-false pruning and admission-limit changes remain
outside this change.
