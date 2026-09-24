# Scanner keyword-length grouping comparison

Date: 2026-09-25. Baseline: the [symbol dispatch comparison](symbol-dispatch-four-image-20260925.md) at `ccf43d4583412db05ec7c1b7acc3aac1f57fb9df`. The candidate changes only `scanner::keyword` so an identifier scans the reserved words of its own byte length. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw report](evidence/keyword-length-four-image-20260925.json) retains image hashes, sizes, sample hash, execution order and every QPC sample.

Both reports use `let n=12345; /*open`, 250,000 scanner rounds per execution, one warmup and twelve measured four-image rotations. Runs check the independent C reporter against the Go lexer, both Fairy reporters against each other, completed rounds and checksum. No samples were retried or excluded. The predeclared stability rule is `(max - min) / median <= 0.10`.

| Optimized paired ratio | Before median | After median | After relative full span |
| --- | ---: | ---: | ---: |
| Explicit-index Fairy / up-front C | 1.28799 | 1.04061 | 12.03% |
| Vector-loop Fairy / up-front C | 1.29292 | 1.02194 | 17.12% |
| Vector-loop / explicit-index Fairy | 1.00516 | 0.98236 | 10.33% |

Every after-change paired series fails the stability rule. These medians are diagnostic point estimates, not evidence that the runtime meets 1.08 or that this change has a measured speedup. The source does less reserved-word work per identifier; its general runtime effect remains unproven.

| Image | Before generated C bytes | After generated C bytes | Before EXE bytes | After EXE bytes |
| --- | ---: | ---: | ---: | ---: |
| Explicit-index Fairy | 399,177 | 401,663 | 205,824 | 206,336 |
| Vector-loop Fairy | 400,005 | 402,491 | 205,824 | 206,336 |

Generated C grows by 2,486 bytes (about 0.62%) and each optimized Fairy EXE by 512 bytes. The up-front C image stays 184,320 bytes; the Fairy images are now 11.94% larger, above the cumulative 8% size gate. The lexical corpus adds all 34 reserved spellings and passes against the independent Go lexer for both C storage modes and both Fairy source forms under low and optimized MSVC. The four-image 1,000-round smoke also passes. C reference completeness, object size, check/build time, unused-feature overhead, other inputs and whole-application performance remain open. The vector source form stays private.
