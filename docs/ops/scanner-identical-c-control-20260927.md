# Identical C scanner timing control

Date: 2026-09-27. Child HEAD: `1c4bdf98`. Windows AMD64, MSVC
14.51.36231, Go 1.26.4, optimized C11 host profile. The
[raw QPC receipt](evidence/scanner-identical-c-control-20260927.json)
retains both complete runs. A temporary focused Go test and mustflow intent
collected them and were removed afterward.

The existing independent-C `fc_workload_round` scanned the fixed
`let n=12345; /*open` input for 250,000 rounds per call. The same function
ran twice in one optimized image per pair, alternating first position across
24 pairs. One call warmed the scanner before each run. Every call returned
the checked 9,000,000 work checksum; QPC excluded process launch and output.
Neither run removed or retried a pair. The second full run followed a fix to
the receipt's warmup field name; its timing procedure was unchanged. Both
runs are retained, including the unfavorable one.

| Run | Median B/A QPC ratio | Relative full span | 10% stability limit |
| --- | ---: | ---: | --- |
| 1 | 1.00026 | 7.65% | Pass |
| 2 | 1.00602 | 14.22% | Fail |

The first run is stable under the predeclared threshold, but the second run
shows that even **identical C scanner calls** can cross it on this host. The
[fixed arithmetic control](scanner-host-noise-control-20260927.md) had only
2.96% spread; it does not model scanner allocation, cache state, or input work.
These controls cannot assign the variance to the Fairy loop or establish the
cumulative 8% Fairy/C runtime gate. Repeating this host measurement until a
favorable run appears would bias the decision. Runtime admission needs a
controlled environment or a revised, predeclared scanner measurement protocol;
the compiler-private vector iteration remains unadmitted.
