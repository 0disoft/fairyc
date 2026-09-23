# Argument-label AST layout experiment

Date: 2026-09-23. Compiler HEAD: `bd0085b0dcf7b159268ea75debad9652afb2d336` plus the opt-in measurement harness. Host: Windows AMD64, Go 1.26.4, `GOMAXPROCS=1`.

## Question and method

The current `ast.Expression` is 136 bytes because every expression carries a `CallLabels` pointer. Without that field it is 128 bytes. The [earlier scanner count](argument-label-cost-experiment.md#follow-up-unused-expression-object-space) found 9,026 reachable unlabeled expressions and calculated 144,416 bytes of direct object-class exposure. That calculation was not an end-to-end allocation or check-time measurement.

The configured `fairyc_measure_argument_ast_layout` intent builds two Go test binaries. The current lane is unmodified. A test-only Go overlay removes the label field, label lowering and checking, cloning of label metadata, and label-specific test inspections in the compact lane. This lane **does not support labeled calls and is not a candidate implementation**. Both binaries process exactly the same unlabeled scanner workspace and selected SDK sources. Each isolated process gets one untimed warmup, then four fresh load/parse/check/verified-MIR iterations. Six process pairs alternate lane order; `runtime.GC()` runs before each timed section, and ordinary GC remains active. Go `TotalAlloc` and `Mallocs` differences are divided by four. Build time, process startup, source setup, MIR JSON hashing, native compilation, and persistent cache work are excluded.

Every pair passed source-hash equality for all eight files, verified without diagnostics, and produced the same MIR JSON hash and 212 functions. The overlay and binary hashes, full per-pair observations, and method text are in [raw evidence](evidence/argument-label-ast-layout-20260923.json).

## Observations

| Pair | First lane | Current check ms | Compact check ms | Compact/current | Compact minus current allocation bytes |
| ---: | --- | ---: | ---: | ---: | ---: |
| 1 | current | 518.18 | 456.88 | 0.882 | -445,140 |
| 2 | compact | 536.44 | 460.00 | 0.858 | -451,338 |
| 3 | current | 425.70 | 407.31 | 0.957 | -444,692 |
| 4 | compact | 399.59 | 477.43 | 1.195 | -442,284 |
| 5 | current | 458.87 | 521.31 | 1.136 | -453,934 |
| 6 | compact | 452.84 | 417.37 | 0.922 | -452,894 |

The paired median check-time ratio is 0.939, but individual ratios range from 0.858 to 1.195, with two pairs favoring the current layout. This run does **not** establish a check-time improvement. The compact lane allocated 442,284–453,934 fewer Go bytes per iteration, with a paired median difference of -448,239 bytes. That is roughly 0.26% of the current lane's approximately 169.4 MB of cumulative allocation per iteration. The paired median `Mallocs` difference is zero; neither number is retained heap, peak RSS, or GC pause time.

## Decision

Keep the production AST unchanged. The allocation difference is consistent with a cost from the larger expression layout, but the overlay also removes label checking, so this experiment cannot assign the entire difference to the field alone or preserve labeled-call behavior. The noisy elapsed times and small fraction of total allocation do not justify a production side table or metadata union yet. A future representation proposal would need to retain labeled-call diagnostics, generic cloning, source spans, generated C, and cache behavior, then compare labeled and unlabeled real workloads on the same source. The cumulative C-relative 8% runtime/EXE policy does not apply to this Go frontend experiment.
