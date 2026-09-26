# Scanner runtime after fixed-category trap-edge lowering

Date: 2026-09-27. Child source HEAD: `b09adf19`. Windows AMD64,
MSVC 14.51.36231, Go 1.26.4, `GOMAXPROCS=1`, optimized profile. The
[raw receipt](evidence/scanner-post-trap-edge-four-image-20260927.json)
contains image hashes, the fixed input hash, execution order and every QPC
reading.

Both independent C storage modes and both Fairy loop forms ran the same
`let n=12345; /*open` scanner input for 250,000 rounds per execution. One
warmup preceded twelve four-lane rotations. Each execution verified the
reporter, completed rounds and 9,000,000 work checksum. QPC excludes process
startup and final output; token reporting is outside the timed loop. No sample
was retried or excluded.

| Paired runtime ratio | Median | Relative full span |
| --- | ---: | ---: |
| Explicit-index Fairy / independent C | 1.07026 | 37.63% |
| Private vector-loop Fairy / independent C | 1.08193 | 25.41% |
| Private vector-loop Fairy / explicit-index Fairy | 1.01742 | 24.22% |

The predeclared stability rule requires `(maximum - minimum) / median <= 0.10`.
All three ratios fail it. Their point estimates establish neither an 8%
runtime pass nor a stable private-vector regression. This run cannot isolate
the runtime effect of the trap-edge lowering from run-to-run variation.

The independent C, geometric C, explicit-index Fairy and private vector-loop
Fairy images are 184,320, 185,344, 200,192 and 200,192 bytes respectively.
The smaller C image sets a 199,065-byte whole-file 8% ceiling; both Fairy
images exceed it by 1,127 bytes. The two Fairy forms retain 21 specializations
each and generated C sizes of 381,760 and 382,600 bytes. This one fixed input
does not prove whole-application runtime parity or close the remaining size,
build-time and source-admission gates.
