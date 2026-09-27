# GC-controlled scanner process-CPU null result

Date: 2026-09-27. The [protocol](scanner-process-cpu-gc-protocol-20260927.md)
and opt-in harness were committed at `4d95d467` before this first run. The
[raw receipt](evidence/scanner-process-cpu-gc-null-20260927.json) retains all
200 builds: one warmup and 24 measured eight-build pairs, with four builds per
lane. No row or pair was removed or retried. Go was 1.26.4 with
`GOMAXPROCS=1`; no MSVC child process ran.

Both lanes used the same indexed scanner SHA-256
`2a61a99092b59c1c30cacd1503945f32c9f09ff0467a5f8436f180089fb9ef95`
and the same compiler path. Every build emitted the same 369,719-byte C unit
with the same hash, 80 functions and 21 specializations. The harness forced
GC before each timed build, disabled automatic GC, and rejected any change in
`NumGC` during the interval.

| Sorted clone/index process-CPU ratio | Value |
| --- | ---: |
| 7th, required at least 0.98 | 0.97273 |
| 18th, required at most 1.02 | 1.00952 |
| Median of 24 pairs | 0.99554 |
| Minimum / maximum | 0.93636 / 1.07477 |
| Full relative span | 13.90% |

The lower order statistic fails the predeclared `[0.98, 1.02]` null band.
All 200 process-CPU readings are still multiples of 15.625 ms, even though
each measured ratio sums four builds per lane. Individual CPU readings range
from 359.375 to 546.875 ms. The remaining pair variation cannot be attributed
to automatic GC during the timed interval, but this receipt does not identify
its source. A controlled host or different predeclared, validated clock and
measurement boundary is needed. No vector candidate comparison followed.
This result neither changes the separate runtime/EXE findings nor establishes
compiler-cost parity or normal-GC build latency.
