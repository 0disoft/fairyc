# Four-image scanner C budget comparison

Date: 2026-09-25. Fairy C compiler source: `4ce3536453d3279ccf04852766b0962e81477033` plus this measurement intent. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw report](evidence/vector-lexer-four-image-c-budget-20260925.json) retains all image hashes, sizes, QPC samples, sample hash, and execution order.

One independent C scanner uses up-front token and diagnostic arrays; the other doubles their capacities from eight slots. The explicit-index and compiler-private vector-loop Fairy scanners use the same checked-in workload. All four images process `let n=12345; /*open` for 250,000 rounds per execution. One warmup precedes twelve rotations, placing each image three times in each execution position. Both C reporters match the independent Go observation, and the Fairy reporters match each other. All executions complete 250,000 rounds with the 9,000,000 work checksum. QPC includes scanner work and cleanup, excluding startup and final output. No executions were retried or excluded.

| Optimized series | Median | Relative full span |
| --- | ---: | ---: |
| Up-front C | 386.356 ms | 3.66% |
| Geometric C | 434.398 ms | 6.63% |
| Explicit-index Fairy | 685.675 ms | 3.46% |
| Vector-loop Fairy | 689.213 ms | 5.02% |
| Paired geometric/up-front C | 1.12481 | 9.28% |
| Paired index/up-front C | 1.77502 | 3.28% |
| Paired vector/up-front C | 1.79340 | 6.81% |
| Paired vector/index | 1.01354 | 5.84% |

The predeclared stability rule is `(max - min) / median <= 0.10`. Every series shown passes it. The separate index/geometric C paired series has a 10.09% full span and does not pass; its median is not used to select the C runtime baseline. The stable geometric/up-front comparison makes up-front C the faster lane for this fixed sample. The vector/index result indicates a small observed candidate cost here, but it does not establish a general regression or explain the much larger Fairy/C gap.

| Image | EXE bytes | Generated or authored C source bytes | Specializations |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 52,164 | 0 |
| Geometric C | 185,344 | 52,245 | 0 |
| Explicit-index Fairy | 205,824 | 384,636 | 21 |
| Vector-loop Fairy | 205,824 | 385,464 | 21 |

Up-front C is also the smaller C image in this build. Both Fairy executables are 21,504 bytes, or 11.67%, larger than it and equal to each other. On this common input, the faster/smaller C lane is therefore unambiguous and both Fairy forms exceed the 1.08 runtime and EXE screen. The independent C scanner still lacks complete accepted-input, limit, and failure-path parity with the Fairy scanner, so this is a bounded workload result rather than the representative whole-scanner admission verdict. Build/check time, object size, unused-feature overhead, and whole-application performance are not established here. The vector source form remains private.
