# Indexed and vector-loop scanner build-phase cost

Date: 2026-09-25. Child source HEAD: `3c445535`.
The [raw receipt](evidence/vector-scanner-build-cost-20260925.json) compares
the checked-in explicit-index scanner with the compiler-private form using all
three eligible read-only SDK vector loops. Both use the same package graph,
benchmark main, runtime C support and optimized Windows AMD64 MSVC profile.
The test prepares fresh temporary package files for every image, warms up one
pair, then builds five measured pairs with alternating lane order under
`GOMAXPROCS=1`. Every image has 21 specializations.

`check_ns` covers package loading, metadata extraction, semantic analysis and
verified-MIR lowering. `emit_ns` covers generated C emission. `support_ns`
assembles the runtime C source; readings of zero are below the timer's observed
resolution, not proof of no work. `native_ns` covers MSVC compile/link and
artifact validation and reads. The phased total adds these four durations;
source-file preparation and toolchain resolution are outside it. There is no
persistent-cache hit or stable CLI startup in this measurement. Ambient
filesystem caching, process load and Go GC remain uncontrolled.

| Measured phase | Indexed median | Vector-loop median | Indexed / vector full span |
| --- | ---: | ---: | ---: |
| Check | 472.80 ms | 474.87 ms | 3.5% / 13.0% |
| C emission | 265.89 ms | 252.88 ms | 13.9% / 12.3% |
| Native build and artifact reads | 1,488.72 ms | 1,467.81 ms | 3.5% / 3.6% |
| Phased total | 2,240.62 ms | 2,186.76 ms | 4.2% / 0.8% |

The vector/index ratio of the lane medians for the phased total is 0.97596.
The five paired total ratios are 1.01337, 0.97041, 0.96901, 0.98627 and
0.97548; their median is 0.97548 and their relative full span is 4.55%.
This run is inside the existing 10% stability heuristic for total build cost.
The C-emission and vector check series exceed that heuristic, so this receipt
does not identify either phase as a reliable speed improvement or regression.
The generated C sources are 395,860 and 396,700 bytes respectively; both
optimized EXEs are 201,728 bytes on every measured image.

This is a local comparison of two Fairy source forms, not a comparison with an
independent C build. The 8% build marker is observation-only; the cumulative
runtime and EXE budget against equivalent C, stable runtime, and unused-feature
overhead remain separate admission gates. The vector source form stays private.
