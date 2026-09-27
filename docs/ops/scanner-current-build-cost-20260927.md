# Current scanner source-form build cost

Date: 2026-09-27. Child source HEAD: `fb24e0c5`. The
[raw receipt](evidence/scanner-current-build-cost-20260927.json) compares the
explicit-index scanner with all three eligible loops rewritten to the
compiler-private read-only `Vec<T>` form. Their scanner source hashes match
the two images in the [robust runtime receipt](evidence/scanner-robust-same-image-c-triads-20260927.json).
The benchmark main differs from that timing probe. This measurement uses one
warmup pair and five measured pairs in alternating order, `GOMAXPROCS=1`, Go
1.26.4, optimized MSVC 14.51.36231, fresh temporary package files and no
persistent-cache hit. Both forms have 21 specializations.

| Form | Median check | Median phased total | Generated C | Optimized EXE |
| --- | ---: | ---: | ---: | ---: |
| Explicit index | 341.11 ms | 1,863.27 ms | 369,719 bytes | 197,120 bytes |
| Private vector loop | 339.16 ms | 1,860.12 ms | 370,559 bytes | 197,120 bytes |

The median of the five paired vector/index total-build ratios is **0.99896**.
Their relative full span is **16.26%**, above the existing 10% stability
heuristic. The last pair measured 1,929.42 ms for the vector form followed by
2,243.15 ms for the indexed form; the indexed check, C-emission and native
phases were all slower than in its other measured pairs. The pair remains in
the receipt. No retry or exclusion was used to turn the central ratio into a
pass. This observation does not establish a source-form build-speed advantage,
disadvantage, or equivalence.

The vector form adds 840 generated-C bytes here while the linked EXE sizes
match. The phased total includes package load, metadata, semantic analysis,
verified-MIR lowering, C emission, runtime-source assembly, MSVC compile/link
and artifact validation. Source preparation, toolchain resolution, stable CLI
startup, persistent-cache behavior and an independent C build are outside the
timer. The [local robust runtime screen](scanner-robust-triad-result-20260927.md)
and separate standalone EXE-size pass do not settle this compiler-time gate.
The source form remains compiler-private.
