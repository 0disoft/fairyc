# Mutable vector build-phase diagnostic

Date: 2026-09-24. Exact Fairy C source: `d1455505e4060843cedf0a5e1d91a32e4e580ac0`. Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [direct-address vector comparison](c-budget-vector-address-20260924.md) put runtime and EXE size inside the cumulative 1.08 C budget, but its ordinary build median was 1.40358 times C. This diagnostic uses the same `vector_slot.fy`, `vector_slot.c` and `support.c` bytes and the same five alternating C/Fairy pairs. Only opt-in wall-clock subdivisions were added to the test harness. All ten boundary executions passed. [Raw evidence](evidence/c-budget-vector-phases-20260924.json) retains every sample, exact compiler commit, source hashes and phase durations; no samples were discarded.

| Phase | C median | Fairy median | Boundary |
| --- | ---: | ---: | --- |
| Source and workspace load | below timer resolution | 17.702 ms | Read support/source; Fairy loads the workspace |
| Verified workspace check | — | 114.329 ms | Semantic input, SDK generic preparation, analysis, MIR lowering and verification |
| C emission and support assembly | — | 29.987 ms | SDK authority check, C emission and support-source selection |
| MSVC build | 553.191 ms | 610.728 ms | C compile, link and artifact read |
| Final EXE write | 0.896 ms | 0.545 ms | Write artifact bytes |
| Whole build | 553.748 ms | 779.009 ms | Source read through EXE write |

The 114ms `Check` phase is the largest Fairy-only subdivision. The MSVC phase adds about 58ms over C on this workload, while Fairy emits 35,902 bytes of generated C versus 2,170 bytes of hand-written C. These are wall-time observations, not an attribution of every millisecond to Go CPU or to C text length. The C source-read phase is sometimes recorded as zero at this timer resolution. Phase medians are calculated independently and need not sum to the median whole-build time. Warm toolchain discovery, harness startup, workspace setup and program execution remain outside the build timer; OS caches were not controlled.

The next diagnostic should separate SDK generic preparation, semantic analysis, MIR work and C emission within the verified check. Any optimization must preserve SDK-only storage authority, deterministic diagnostics and the existing verifier. A tiny vector program is a useful stress case for fixed overhead but does not prove the build ratio for a representative multi-file application.

The [SDK authority-reference follow-up](c-budget-vector-sdk-reference-cache-20260924.md) removes one repeated parse while keeping current workspace checks; it records the after-change phase samples separately.
