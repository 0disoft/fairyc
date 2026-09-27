# Scanner locked-thread cycle null result

Date: 2026-09-27. The [protocol](scanner-thread-cycles-protocol-20260927.md)
and harness were committed at `1fd71ed4` before this first measurement. The
[raw receipt](evidence/scanner-thread-cycles-null-20260927.json) retains all
208 builds: one warmup and 12 measured 16-build pairs. No build or pair was
excluded or retried. The configured run completed; the predeclared null
criterion failed.

Both lanes used the same indexed scanner SHA-256
`bfbc305c0bf69063812f019e7f3e9cca505a8afb22744e0d50f51c7f72692c2f`
and compiler path. All builds emitted the same 369,776-byte C unit hash, 80
functions and 21 specializations. `GOMAXPROCS=1`, the process affinity mask
was `0x1`, the measuring Go goroutine was locked to one OS thread, and no
automatic Go GC occurred inside a timed build. The host used Go 1.26.4 and
the Windows AMD64 MSVC target.

| Sorted clone/index thread-cycle ratio | Value |
| --- | ---: |
| 4th, required at least 0.98 | 0.98942 |
| 9th, required at most 1.02 | 1.03861 |
| Median of 12 pairs | 1.01399 |
| Minimum / maximum | 0.97864 / 1.05776 |
| Full relative span | 7.80% |

The 9th ratio exceeds the fixed 1.02 limit. Pinning one measuring thread and
aggregating eight builds per lane narrowed the full span compared with the
earlier process-wide cycle null, but the identical-source control is still
too variable for a candidate verdict. This observation does not identify a
compiler regression: both lanes execute identical code. Thread cycles are
not elapsed time and omit work on other threads and off-CPU waits. No private
vector candidate run followed. Compiler-cost parity, ordinary check/build
latency and the C-relative runtime gate remain unproven; stable vector-loop
source admission remains closed.
