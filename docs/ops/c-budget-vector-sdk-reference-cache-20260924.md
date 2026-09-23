# Cached SDK authority reference build comparison

Date: 2026-09-24. Before: `d1455505e4060843cedf0a5e1d91a32e4e580ac0`; after: `413e0b41357217b8585c42983dece45cc2daba12`. Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [phase baseline](c-budget-vector-build-phases-20260924.md) found repeated verified workspace checks to be the largest Fairy-only build stage. `SDKStorageOwners` used to parse the immutable embedded vector SDK catalog afresh on every authority check. It now keeps that expected parsed catalog private and prepares it once per process. Every call still compares the current workspace's SDK identity, digest, bytes, module catalog and AST against the expected catalog. The existing forged-byte, AST, identity and order tests passed, as did low and optimized native mutable-vector execution.

The same `vector_slot.fy`, `vector_slot.c` and `support.c` fixture hashes, generated Fairy C hash and support C hash were observed before and after. Five alternating C/Fairy pairs, each with one build/run warmup, passed all ten boundary executions. [Raw after-change evidence](evidence/c-budget-vector-sdk-reference-cache-20260924.json) retains every sample and phase duration; [raw before-change evidence](evidence/c-budget-vector-phases-20260924.json) is kept separately. No samples were discarded.

| Phase | Before Fairy median | After Fairy median |
| --- | ---: | ---: |
| Workspace load | 17.702 ms | 18.491 ms |
| Verified check | 114.329 ms | 99.400 ms |
| C emission/support assembly | 29.987 ms | 17.145 ms |
| MSVC build | 610.728 ms | 582.399 ms |
| Whole build | 779.009 ms | 718.306 ms |

The C whole-build medians were 553.748ms before and 537.367ms after, yielding Fairy/C ratios of 1.40679 and 1.33672. The check and support-assembly reductions are consistent with removing repeated embedded-catalog parsing. MSVC time also moved despite identical generated C and support bytes, so the 60.703ms whole-build difference cannot all be attributed to this change. These are separate local runs with uncontrolled OS caches, not a paired A/B experiment. The post-change ordinary build remains above the cumulative 1.08 C observation target; runtime and EXE size were unchanged in code. More build-cost work is required.
