# Direct-return Result arm versus C

Date: 2026-09-24. Compiler source: `d4fb67953bea62095d53cb0a9ca0b7a1b1d9a91a`
plus the benchmark fixture and harness in this change. Windows AMD64,
Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11
`/W4 /WX /MT /O2 /INCREMENTAL:NO`.

## Method

The Fairy fixture classifies each byte as `Result<u32, u32>`, then uses a
value-producing `.ok` arm and a direct-return `.err` arm. Independently written
C uses an equivalent conditional branch. Both kernels accumulate the same
decoded value over 50 million iterations, read the same seed and round count,
and use the same QPC timing and output support. A separate Go calculation
checks each result. Both lanes passed five boundary inputs before timing.

After one build/run warmup per lane, five alternating C/Fairy pairs were
recorded. The [raw evidence](evidence/c-budget-result-return-20260924.json)
contains ten samples, ten boundary executions, fixture hashes, emitted source
hashes, build times, and PE sizes. Kernel timing excludes input parsing,
process startup, and printing. Build timing includes source load, Fairy checking
and MIR/C emission where applicable, MSVC compile/link, artifact reads, and
final EXE write. Common warmed toolchain discovery, harness startup, and
workspace setup are excluded. Ambient OS caches were not controlled.

## Results

| Metric | C median | Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Kernel | 233.756 ms | 233.821 ms | 1.00028 |
| EXE | 167,936 B | 168,960 B | 1.00610 |
| Build | 534.512 ms | 520.057 ms | 0.97296 |

The median paired runtime ratio is 1.00028; individual pairs range from
0.98969 to 1.00958. Fairy's EXE is 1,024 B larger (+0.610%). This workload's
runtime and EXE ratios fall within the cumulative 1.08 budget. The observed
build ratio also falls within it, but the five build pairs fluctuate and do not
establish a reliable compiler-speed improvement. No samples were discarded.

The C fixture has no `Result` tag and the Fairy fixture does, while Fairy also
retains its usual checked arithmetic and full required product support. The
measurement therefore covers the complete current paths rather than isolating
the return-arm syntax. This single Windows optimized scalar workload does not
establish a language-wide 8% result, parser-scale performance, Linux behavior,
cold CLI time, or metadata size. The earlier
[vector-slot comparison](c-budget-vector-slot-20260924.md) still exceeds the
runtime budget and remains an independent optimization target.
