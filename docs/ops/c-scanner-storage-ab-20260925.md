# Independent C scanner storage A/B

Status: measured; timing inconclusive. This compares two builds of the same independent
C scanner, not Fairy against C. The baseline allocates token and diagnostic
arrays for `length + 1` entries. The alternate starts at eight entries and
doubles capacity, matching the private Fairy `Vec<T>` storage strategy.

Both optimized MSVC images scan `let n=12345; /*open` for 250,000 rounds per
execution. Each execution must match the Go lexer reporter, completed rounds,
and checksum. QPC covers scan, storage and cleanup inside the repeated loop;
process startup and final output are excluded. One warmup pair precedes twelve
measured pairs. Execution order alternates between the two C images. There are
no retries, exclusions, adaptive stops or input changes. The opt-in smoke uses
one pair of 1,000 rounds and makes no timing claim.

The [raw evidence](evidence/c-scanner-storage-ab-20260925.json) records every
QPC sample, frequency, order, source and executable digest, EXE byte count,
Go 1.26.4 and MSVC 14.51.36231. Before interpreting a timing difference,
require `(max - min) / median <= 0.10` for both lane times and the paired
geometric/preallocated ratio. Even a stable A/B only identifies storage cost
for this short scanner sample. It cannot establish the full-scanner 8% Fairy/C
runtime or EXE verdict because the C reference still has missing lexical and
budget behavior, and Fairy diagnostic payload ownership differs.

| Optimized C series | Median | Minimum | Maximum | Relative full span |
| --- | ---: | ---: | ---: | ---: |
| Preallocated | 563.535 ms | 476.270 ms | 704.659 ms | 40.53% |
| Geometric | 593.495 ms | 458.965 ms | 839.155 ms | 64.06% |
| Paired geometric/preallocated | 1.12506 | 0.85043 | 1.20662 | 31.66% |

Each lane took the first position six times. All 26 executions matched the
Go reporter, completed 250,000 rounds, and returned the same 9,000,000 work
checksum; the first pair was warmup and excluded from the table. The ratio of
lane medians is 1.05316, distinct from the median paired ratio. All three
series fail the predeclared 10% stability rule, so these timings do not
identify a faster C storage strategy. No samples were removed or rerun.

The preallocated executable is 181,760 bytes; the geometric executable is
181,248 bytes, 512 bytes smaller (0.28%) in this optimized build. Their
different hashes are in the raw evidence. This size observation is limited to
the current short-sample C images and does not decide the full scanner 8%
Fairy/C runtime or executable-size gate.
