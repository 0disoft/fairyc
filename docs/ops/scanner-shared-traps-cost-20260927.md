# Scanner cost after fixed-category trap sharing

Date: 2026-09-27. Child source HEAD: `84f1fecd`. Windows AMD64,
MSVC 14.51.36231, Go 1.26.4, `GOMAXPROCS=1`, optimized profile. The
[runtime receipt](evidence/scanner-shared-traps-four-image-20260927.json)
and [build receipt](evidence/scanner-shared-traps-build-cost-20260927.json)
retain image/source sizes, execution order and raw timings.

The four standalone images scanned `let n=12345; /*open` for 250,000 rounds
per execution. One warmup preceded twelve four-lane rotations. Every run
verified the reporter, completed rounds and 9,000,000 work checksum; no
sample was retried or excluded. QPC excludes process startup and final
output, and token reporting is outside the timed loop.

| Paired QPC ratio | Median | Relative full span |
| --- | ---: | ---: |
| Explicit-index Fairy / independent C | 1.05864 | 11.86% |
| Private vector-loop Fairy / independent C | 1.06880 | 24.84% |
| Private vector-loop Fairy / explicit-index Fairy | 1.00751 | 16.73% |

All three spans exceed the predeclared 10% stability limit. The point
estimates cannot establish an 8% runtime pass or a stable vector-loop
regression. The independent C, geometric C, indexed Fairy and vector-loop
Fairy EXEs measure 184,320, 185,344, 197,120 and 197,120 bytes. The
smaller C image sets a 199,065-byte 8% ceiling, leaving 1,945 bytes of
size margin for this workload.

The separate build-cost run used one warmup and five balanced indexed/vector
pairs. Each measured a fresh source check, generated-C emission, support
assembly and optimized MSVC build. Both forms retained 21 specializations.

| Fairy source form | Median total build | Full span | Generated C | EXE |
| --- | ---: | ---: | ---: | ---: |
| Explicit index | 2.007 s | 9.27% | 369,719 bytes | 197,120 bytes |
| Private vector loop | 1.930 s | 7.15% | 370,559 bytes | 197,120 bytes |

The paired vector/indexed total-build ratio has median 0.98489 and 5.19%
relative span. This same-program measurement does not isolate the compiler
time effect of trap sharing against the previous backend, compare Fairy
builds with independent C, or establish unused-feature overhead. Runtime
stability, broader C-equivalent inputs and the remaining admission gates
keep read-only SDK vector iteration compiler-private.
