# Scanner identical-source process-CPU control result

Date: 2026-09-27. The [protocol](scanner-process-cpu-cost-protocol-20260927.md)
and Windows harness were committed at `640817db` before this first process-CPU
run. The [raw receipt](evidence/scanner-process-cpu-null-20260927.json) keeps
all 100 fresh scanner preparations, checks, verified-MIR lowerings and C
emissions: one warmup and 24 measured `A-B-B-A` / `B-A-A-B` pairs. Both lanes
used the identical explicit-index source and stable compiler path. Go was
1.26.4 with `GOMAXPROCS=1`; no MSVC child process ran.

| Sorted clone/index process-CPU ratio | Value |
| --- | ---: |
| 7th (null lower bound) | 0.95946 |
| 18th (null upper bound) | 1.03488 |
| Median of 24 pairs | 1.00676 |
| Full relative span | 31.90% |

The predeclared null requires both order statistics inside `[0.98, 1.02]`.
Both fail. Every build emitted the same 369,719-byte C unit with the same
SHA-256, 80 functions and 21 specializations. The indexed scanner and
benchmark-main source hashes match the earlier build-cost receipt. No row or
pair was removed, and no candidate comparison was run.

All 100 recorded process-CPU durations are multiples of **15.625 ms** even
though `GetProcessTimes` expresses them in 100 ns units. This observed
quantization and the 31.90% ratio span make the process-CPU clock inadequate
for this 2% screen on this host; Go GC and other process work were also not
isolated. The result gives no vector-versus-indexed compiler-cost verdict and
does not supersede the failed wall-time controls. A different controlled
environment or a separately specified, higher-resolution compiler-work
measurement is needed before the cost gate can be decided.
