# Identical-source scanner build control result

Date: 2026-09-27. Child source HEAD: `ce3416ac`. The
[predeclared protocol](scanner-build-null-protocol-20260927.md) was committed
before this single run. The [raw receipt](evidence/scanner-build-null-20260927.json)
retains the warmup and all five measured pairs. Both lanes compiled the same
explicit-index scanner hash with the same benchmark main, 21 specializations,
369,719 generated-C bytes and 197,120-byte optimized EXEs. Go was 1.26.4,
`GOMAXPROCS=1`, and MSVC was 14.51.36231.

| Measured pair | Clone/index total-build ratio |
| --- | ---: |
| 1 | 1.06317 |
| 2 | 0.96853 |
| 3 | 1.00975 |
| 4 | 1.11321 |
| 5 | 0.99965 |

The paired median is **1.00975**, inside the predeclared `[0.98, 1.02]`
null band, but `(max - min) / median` is **14.33%**, above the 10% stability
limit. The null control therefore **fails**. In pair 4, the clone's native
MSVC phase took 1,596.30 ms versus 1,394.32 ms for the identical indexed
lane; the approximately 202 ms native-phase difference accounts for most of
that pair's total gap. Other pairs vary in both directions. No pair was
discarded or rerun.

The earlier indexed/vector [current-source comparison](scanner-current-build-cost-20260927.md)
also had a wide full span. This identical-source control shows that the
current host and harness can produce a wider-than-allowed build ratio without
a source-form difference. It does not estimate the private vector-loop cost.
A future admission measurement needs a controlled build environment or a
separately validated phase-specific comparator; repeating this run until a
favorable span appears would not answer the question.
