# Nominal module JSON reuse experiment

Date: 2026-09-24. Before: `413e0b41357217b8585c42983dece45cc2daba12`; after: `b3e4394510efc3bca9bf55d636addc4394d47ccc`. Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The source change reused each module's JSON encoding between closed-graph nominal identity and local nominal type registration. A compatibility test proved that the closed-graph digest and module JSON bytes match the previous encoding, including escaped source text. Generic frontend, nominal binding and vector SDK candidate tests passed. The broad vector-storage intent stopped at its configured stdout limit; it did not report a test assertion failure.

The same checked vector fixture, generated C and support C hashes were observed before and after. Five alternating C/Fairy pairs ran after one build/run warmup per lane, and all ten boundary executions passed. [Raw before-change evidence](evidence/c-budget-vector-sdk-reference-cache-20260924.json) and [raw after-change evidence](evidence/c-budget-vector-nominal-encoding-20260924.json) retain every sample; no samples were discarded.

| Phase | Before Fairy median | After Fairy median |
| --- | ---: | ---: |
| Workspace load | 18.491 ms | 20.489 ms |
| Verified check | 99.400 ms | 100.065 ms |
| C emission/support assembly | 17.145 ms | 17.914 ms |
| MSVC build | 582.399 ms | 603.914 ms |
| Whole build | 718.306 ms | 734.503 ms |

C whole-build medians were 537.367ms before and 540.518ms after. The two runs have uncontrolled OS caches and should not be treated as a precise A/B estimate. There is no measured check-time gain, while the implementation adds a retained JSON buffer and a second representation in the closed-graph encoder. Decision: revert this source change and keep the negative measurement, rather than claim an optimization. The remaining ordinary build gap is elsewhere in the compiler path and MSVC input.
