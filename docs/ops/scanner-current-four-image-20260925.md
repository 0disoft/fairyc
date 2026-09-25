# Current four-image scanner runtime and size

Date: 2026-09-25. Child source HEAD: `607201fc`. Windows AMD64,
MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`.
The [raw receipt](evidence/scanner-current-four-image-20260925.json) records
image hashes, the fixed sample hash, all QPC readings, and execution order.

Both C storage modes and both Fairy loop forms ran `let n=12345; /*open` for
250,000 rounds per execution. One warmup preceded twelve four-lane rotations.
Each execution checked the reporter, completed rounds, and the 9,000,000 work
checksum. QPC excludes process startup and final output. There were no retries
or excluded samples. The sample omits token reporting inside the timed loop.

| Optimized lane | Median | Relative full span | EXE bytes |
| --- | ---: | ---: | ---: |
| Independent C, up-front storage | 375.77 ms | 5.43% | 184,832 |
| Geometric C | 425.42 ms | 4.92% | 185,344 |
| Explicit-index Fairy | 401.87 ms | 9.80% | 201,728 |
| Private vector-loop Fairy | 402.47 ms | 9.26% | 201,728 |

The predeclared stability rule is `(maximum - minimum) / median <= 0.10`.
The paired explicit-index/up-front-C median is 1.06282 with a 9.56% span,
so this fixed-workload ratio is stable and below 1.08. The paired private-vector/
up-front-C median is 1.07110 with a 12.34% span; the paired private-vector/
explicit-index median is 1.01277 with a 13.84% span. Both fail the stability
rule. Their point estimates do not establish either a runtime pass or a vector
regression. The up-front C lane is the faster and smaller C reference in this
run.

Both Fairy images are 201,728 bytes, 1.09141 times the 184,832-byte C image.
The whole-byte 1.08 ceiling is 199,618, leaving 2,110 bytes to remove.
Both Fairy images retain 21 specializations; their generated C sources are
395,860 and 396,700 bytes respectively. No size or specialization cost is
attributed to the vector source form in these images.

The independent C reporter does not yet reproduce every accepted input,
combined limit and later allocation-failure path. This single fixed input is
therefore a diagnostic scanner-workload comparison, not whole-application
admission. The cumulative EXE gate fails here, and the private vector runtime
gate is inconclusive. Read-only SDK vector iteration stays compiler-private.
