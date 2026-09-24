# Current-image three-way scanner C budget triplets

Date: 2026-09-25. Fairy C compiler source: `52e6e9b6a294235cc6048613ce2f08254ab538d0` plus this measurement intent. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw report](evidence/vector-lexer-c-budget-20260925.json) retains image hashes, sizes, QPC samples, sample hash, and execution order.

The current independent C scanner, explicit-index Fairy scanner, and compiler-private vector-loop Fairy scanner each scanned `let n=12345; /*open` for 250,000 rounds per execution. One warmup triplet preceded twelve measured triplets, rotating all images through every position. Every execution retained the reporter, completed-round count, and independent Go checksum. QPC covered scanner work and cleanup, excluding process startup and final output. No sample was removed or retried.

| Optimized series | Median | Relative full span |
| --- | ---: | ---: |
| Independent C | 390.304 ms | 10.26% |
| Explicit-index Fairy | 694.190 ms | 7.08% |
| Vector-loop Fairy | 690.919 ms | 5.69% |
| Paired index/C | 1.79253 | 14.11% |
| Paired vector/C | 1.75740 | 13.26% |
| Paired vector/index | 0.98705 | 8.88% |

The predeclared stability rule is `(max - min) / median <= 0.10`. The C lane and both C-relative ratios fail it, so this run gives no stable 8% runtime verdict. The vector/index ratio passes this heuristic and shows no measured runtime penalty for the private spelling on this sample; it does not establish a general speed advantage. These are medians of paired ratios, not ratios of lane medians.

The C executable is 184,320 bytes. Both Fairy executables are 205,824 bytes, 21,504 bytes or 11.67% larger than this restricted C reference. They have different hashes but identical byte counts and 21 specializations each. Generated C is 384,636 bytes for the indexed source and 385,464 bytes for the private candidate. The C reference still lacks complete accepted-input and failure-path parity, so its current executable size cannot close the representative whole-scanner 8% gate. The measured C lane also uses up-front token and diagnostic arrays; the separate geometric C lane was not part of these triplets.

This report supersedes the earlier image sizes and QPC observations only for the current source revision. It does not prove the full scanner budget, whole-application performance, build/check time, or unused-feature overhead. Source admission remains closed.
