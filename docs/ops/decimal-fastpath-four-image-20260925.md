# Scanner plain-decimal fast path

Date: 2026-09-25. Baseline Fairy C source: `fd76b4e980c297fdb38d14df2984f1c82a41ba20`; the candidate differs only in `examples/lexer_candidate/src/numbers.fy`. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw after-change report](evidence/decimal-fastpath-four-image-20260925.json) and [previous four-image report](evidence/vector-lexer-four-image-c-budget-20260925.json) retain image hashes, sizes, QPC samples, sample hash, and execution order.

The integer scanner validates all digits and underscores as before. An ordinary decimal spelling without a leading zero or underscore now appends the validated source bytes to the cooked Buffer in one operation. Zero, leading-zero decimals, underscore spellings, binary and hexadecimal integers retain the existing accumulator. The append failure still returns status 3; no partial token stream escapes.

Both reports use the same `let n=12345; /*open` input, 250,000 rounds per execution, one warmup and twelve four-image rotations. Each run checks C reporter parity with Go, equality between the Fairy reporters, completed rounds and checksum; no samples were removed or retried. The predeclared stability rule is `(max - min) / median <= 0.10`.

| Optimized paired ratio | Before median | After median | After relative full span |
| --- | ---: | ---: | ---: |
| Explicit-index Fairy / up-front C | 1.77502 | 1.57333 | 4.90% |
| Vector-loop Fairy / up-front C | 1.79340 | 1.57821 | 6.78% |
| Vector-loop / explicit-index Fairy | 1.01354 | 1.00001 | 6.19% |

The after-change Fairy/C ratios pass the stability rule and are lower than the earlier ratios by 11.36% and 12.00% respectively. The independent C source and executable byte counts are unchanged between the reports. This is an observed improvement for this numeric sample, not a general lexer speedup or proof that the whole-scanner 8% budget passes. Up-front C remains the faster and smaller comparison lane in the earlier storage comparison.

| Image | Before generated C bytes | After generated C bytes | Before/after EXE bytes |
| --- | ---: | ---: | ---: |
| Explicit-index Fairy | 384,636 | 388,167 | 205,824 |
| Vector-loop Fairy | 385,464 | 388,967 | 205,824 |

The generated C is about 0.9% larger, while the optimized EXE byte count stays fixed. Both Fairy images remain 11.67% larger than the up-front C reference. The cross-language lexical corpus, four-image 1,000-round smoke, and forced first-allocation failure for both Fairy variants pass. C reference completeness, object size, check/build time, unused-feature overhead, other inputs, and whole-application performance remain open. The vector source form stays private.
