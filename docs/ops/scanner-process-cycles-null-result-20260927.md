# Scanner process-cycle null result

Date: 2026-09-27. The [protocol](scanner-process-cycles-protocol-20260927.md)
and opt-in harness were committed at `d4736aa6` before this first run. The
[raw receipt](evidence/scanner-process-cycles-null-20260927.json) retains all
200 builds: one warmup and 24 measured eight-build pairs. No pair or build was
removed or retried. The configured measurement completed successfully; the
predeclared null criterion did not.

Both lanes used the same indexed scanner SHA-256
`2a61a99092b59c1c30cacd1503945f32c9f09ff0467a5f8436f180089fb9ef95`
and the same compiler path. Every build emitted the same 369,719-byte C unit
with one hash, 80 functions and 21 specializations. Go GC was forced before
each timed build and did not run within it.

| Sorted clone/index process-cycle ratio | Value |
| --- | ---: |
| 7th, required at least 0.98 | 0.97556 |
| 18th, required at most 1.02 | 1.01725 |
| Median of 24 pairs | 0.99519 |
| Minimum / maximum | 0.92371 / 1.07185 |
| Full relative span | 14.89% |

The lower order statistic fails the fixed `[0.98, 1.02]` band. The separate
process-CPU readings still have 15.625 ms steps, while cycle deltas range from
897,937,343 to 1,611,991,072 per build. The cycle counter avoids that CPU
clock's coarse integer steps but does not eliminate pair variation or identify
its cause. Cycles are not elapsed time, and the observation includes all Go
process threads. No vector candidate run followed. Compiler-cost parity and
ordinary check/build latency remain unproven; the private source form stays
closed.
