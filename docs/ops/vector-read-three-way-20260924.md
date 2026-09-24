# Prebuilt three-way vector-read execution comparison

Date: 2026-09-24. Exact Fairy C source:
`d5330b2111e0b7df113626ebe902ea5565fa5f9e` with SDK
`0.0.0-stage0.67`. Windows AMD64, Go 1.26.4, MSVC 14.51.36231,
`GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [raw report](evidence/vector-read-three-way-20260924.json) records one
independently authored checked C image, one stable indexed Fairy image and one
compiler-private vector-iteration image. The three images were built once,
then reused unchanged. The candidate source is still rejected by the stable
workspace check; its authenticated SDK source passes Flow and verified MIR in
the private lane. All three lanes passed five boundary inputs and matched the
independent Go checksum. After three warmup triads, twelve measured triads
rotated through all six possible lane orders twice. There were 60 native
executions in total. No measurement was retried, removed or stopped early.

| Kernel | Median | Relative full span | EXE |
| --- | ---: | ---: | ---: |
| Independent C | 1,244.6239 ms | 7.22% | 168,448 B |
| Indexed Fairy | 1,340.2058 ms | 7.60% | 171,520 B |
| Private candidate Fairy | 1,333.9615 ms | 4.03% | 171,520 B |

| Paired comparison | Median | Ratio of lane medians | Relative full span |
| --- | ---: | ---: | ---: |
| Indexed/C | 1.08404 | 1.07680 | 9.98% |
| Candidate/C | 1.07226 | 1.07178 | 9.18% |
| Candidate/indexed | 0.99563 | 0.99534 | 7.86% |

The predeclared 10% relative full-span stability heuristic is met by each
lane and ratio series. The indexed/C estimators disagree across the cumulative
1.08 runtime line, so that comparison is **inconclusive**. Candidate/C has
both medians under 1.08; its candidate-first, middle and last position-group
ratio medians are 1.06625, 1.07771 and 1.07879. This is an **observed
within-budget result for this microkernel**, not a confidence interval or
language-wide certification. Individual candidate/C ratios range from
1.00894 to 1.10735. Candidate/indexed is close to parity, with ratios on
both sides of one. Both Fairy EXEs are 1.01824 times the C size.

The earlier [build-and-run comparison](c-budget-vector-read-address-20260924.md)
rebuilt for each pair and observed paired medians of 1.05726 and 1.04905;
this report keeps build cost outside the timed repeated executions. Its
artifact hashes and source hashes establish that each lane reused identical
bytes; the independent C and source fixtures remain unchanged. QPC kernel
timing excludes executable materialization, process startup, input parsing
and output. Ambient host load and CPU frequency were uncontrolled.

The result supports near-parity of the two Fairy source forms on one
four-element read kernel. It does not close the three real lexer loops'
runtime, failure and cleanup cost, cold check/build time, unused-feature
overhead, Linux behavior or stable source admission gates in ADR 0404.
