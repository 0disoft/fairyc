# Real-scanner vector-loop execution comparison

Date: 2026-09-24. Fairy C source:
`6e92ebd38ddb0b18e374925aa963ce22e5d56bfa`. Windows AMD64,
MSVC 14.51.36231, optimized C11, `GOMAXPROCS=1`.

The [raw report](evidence/vector-lexer-execution-20260924.json) compares two
versions of the checked-in `examples/lexer_candidate/src/scanner.fy`: explicit
indexed loops and all three compiler-private read-only vector loops. Stable
analysis still rejects the candidate spelling. The private lane passes Flow
and verified MIR. Both optimized images were built once and reused, with the
same reporter output, a Go-derived checksum of 36 per scan, and successful
post-run allocation cleanup. Both EXEs are 205,824 bytes.

Each timed execution performs 10,000 `collect` and `storage` calls on
`let n=12345; /*open`. The two storage loops run on every scan. The token
reporter runs once before timing, so this experiment does **not** measure its
hot-loop cost. QPC timing excludes process startup, input parsing and output.
After one warmup pair, twelve measured pairs alternate lane order. There were
no retries, exclusions or adaptive stopping.

| Lane | Median QPC time | Relative full span | EXE |
| --- | ---: | ---: | ---: |
| Explicit index | 40.02235 ms | 45.32% | 205,824 B |
| Private vector candidate | 44.51685 ms | 46.30% | 205,824 B |

The paired candidate/index median is 1.11951, and the ratio of lane medians
is 1.11230. Individual paired ratios range from 0.89014 to 1.51809; their
relative full span is 56.09%. Every timing distribution exceeds the
predeclared 10% stability heuristic. **No runtime-cost verdict follows from
these samples.** The equal EXE sizes are an artifact observation for this
scanner workload, not a language-wide size result.

One-shot analysis/lowering intervals were 316.41 ms for indexed and 295.72 ms
for candidate; native build intervals were 1,752.86 ms and 2,033.65 ms.
Their sequential order, caches and ambient host load were uncontrolled, so
these are diagnostic observations, not a check/build regression comparison.
This experiment has no independent C full-scanner lane and does not close the
cumulative 8% C-relative budget.

## Predeclared longer window

The [second raw report](evidence/vector-lexer-execution-long-20260924.json)
uses source `c86b3b061ec7f9852d86da985f21a4ee1ae3b487` and increases each
timed run to 250,000 scans, 25 times the first window. It keeps one warmup
pair, twelve alternating measured pairs, the same input and Go checksum, and
the same two real-scanner source variants. Both optimized EXEs remain 205,824
bytes. No pair was retried, dropped or added after seeing the first result.

| Lane | Median QPC time | Relative full span |
| --- | ---: | ---: |
| Explicit index | 861.80065 ms | 49.18% |
| Private vector candidate | 777.03640 ms | 80.89% |

The paired candidate/index median is 0.97072, whereas the ratio of lane
medians is 0.90164. Individual paired ratios range from 0.77959 to 1.30119;
their relative full span is 53.73%. The longer window also fails the 10%
stability heuristic, and its point estimates reverse the first window's
ordering. These observations cannot establish which Fairy form is faster or
whether either meets the C-relative 8% budget. Further local reruns under the
same uncontrolled host conditions would not settle the admission decision;
the next timing gate needs controlled machine load and frequency or another
measurement method with the same output and ownership checks.
