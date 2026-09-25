# Unused vector-loop source check cost

Date: 2026-09-25. Child source baseline: `1ac04eda`.
The [raw receipt](evidence/vector-scanner-unused-check-cost-20260925.json)
compares the stable SDK storage analysis with the compiler-private vector-loop
candidate analysis on **the exact same indexed scanner source** and benchmark
main. That source never spells the proposed `Vec<T>` loop. Both lanes use fresh
temporary package files, one warmup pair and eight measured pairs with
alternating order under `GOMAXPROCS=1`. Every invocation reaches verified MIR
with 21 specializations. The existing artifact preflight also passed in this
run: canonical MIR, generated C bytes, function count, foreign symbols and
specialization count are equal for the two lanes.

`check_ns` adds package loading, metadata extraction, semantic analysis and
verified-MIR lowering. Source preparation, C emission, MSVC build, CLI startup
and persistent-cache hits are outside the timer. Ambient filesystem caching,
process load and Go GC were not controlled.

| Check cost | Stable lane | Private candidate lane |
| --- | ---: | ---: |
| Median | 447.98 ms | 453.75 ms |
| Relative full span | 33.4% | 31.3% |

The ratio of lane medians is 1.01289. The median of eight paired
candidate/stable ratios is 0.97276, but the paired series has a 20.3%
relative full span. All three spreads exceed the predeclared 10% stability
heuristic. Neither the apparent lane-median increase nor the paired-median
decrease supports a check-speed claim. Repeating the same uncontrolled
measurement until a favorable sample appears would not close this gate.

This receipt proves identical compiler artifacts for this unused-source
workload and leaves check-time overhead inconclusive. It does not measure
native object or EXE bytes, a future integrated stable path, or the
independent-C cumulative 8% runtime and EXE budget. The vector source form
remains compiler-private.
