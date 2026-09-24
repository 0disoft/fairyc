# Scanner symbol dispatch comparison

Date: 2026-09-25. Baseline: the [plain-decimal fast path](decimal-fastpath-four-image-20260925.md) at `2b323effebbb9f9537e7f2db5e46901a0a0e7b7b`. The candidate replaces only `scanner::symbol_width`'s full symbol-list scan with first-byte dispatch; subsequent commits before this run added failure probes but did not change the scanner workload or compiler. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw report](evidence/symbol-dispatch-four-image-20260925.json) retains image hashes, sizes, sample hash, execution order and all QPC samples.

Both reports use `let n=12345; /*open`, 250,000 scanner rounds per execution, one warmup and twelve measured four-image rotations. Each run checks the independent C reporter against the Go lexer and the two Fairy reporters against each other, plus completed rounds and checksum. No samples were retried or excluded. The predeclared stability rule is `(max - min) / median <= 0.10`.

| Optimized paired ratio | Before median | After median | After relative full span |
| --- | ---: | ---: | ---: |
| Explicit-index Fairy / up-front C | 1.57333 | 1.28799 | 4.72% |
| Vector-loop Fairy / up-front C | 1.57821 | 1.29292 | 6.58% |
| Vector-loop / explicit-index Fairy | 1.00001 | 1.00516 | 8.03% |

The Fairy/C medians are lower by 18.14% and 18.08% respectively against the earlier run. All after-change ratio spans pass the stability rule. This is a same-input improvement, not a whole-lexer speed claim; the independent C reference still lacks full limit and failure-path parity. Both Fairy forms remain above the cumulative 1.08 runtime gate.

| Image | Before generated C bytes | After generated C bytes | Before/after EXE bytes |
| --- | ---: | ---: | ---: |
| Explicit-index Fairy | 388,167 | 399,177 | 205,824 |
| Vector-loop Fairy | 388,967 | 400,005 | 205,824 |

Generated C grows by about 2.84%; the optimized EXEs stay equal to their earlier byte counts and remain 11.67% larger than the up-front C image of 184,320 bytes. The independent C images and source byte counts are unchanged. The checked-in lexical corpus now includes all listed symbol spellings and passes across Go, both C storage modes and both Fairy forms under low and optimized MSVC; the four-image 1,000-round smoke also passes. Object size, check/build time, unused-feature overhead, other inputs and whole-application performance remain unmeasured. The vector source form stays private.
