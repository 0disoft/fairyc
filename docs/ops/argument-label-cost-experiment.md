# Argument label cost experiment

Date: 2026-09-22. Host: Windows AMD64, Go 1.26.4, MSVC 14.51.36231, GOMAXPROCS=1.
Baseline: `2a5d2dd7dd05114154522fe22818ff299f60abda`.
Current: `d5dc062b44e413f8e308bd402052513e034e7875`.
Design reference: [ADR 0396](../adr/0396-position-preserving-argument-labels.md). This document does not change ADR 0396.

## Method

- Six alternating process pairs (repetitions 0-5 x baseline/current, 12 samples).
- Each process: one warmup per corpus, then eight uncached load/parse/check/verified-MIR iterations per corpus.
- Forced GC once before timing; normal GC stays active during timing.
- `check_ns` is the per-iteration mean of reload plus parse, check, and verified MIR lowering. Package cache is not used in this timing.
- `go_alloc_bytes` and `go_mallocs` are the cumulative difference over the timed section divided by 8 (per-iteration mean), not peak, RSS, or child-process memory.
- Setup, initial lock creation, process startup, Go compilation, and cache serialization are excluded.
- Unlabeled `.fy` bytes are identical across lanes; manifests use each compiler exact language pin.
- Scanner plus SDK control is frontend-only.
- The synthetic 64-call corpus additionally observes ordinary C and native builds in both profiles, artifact bytes, and in-memory package-cache serialized payload bytes (not disk cache or RSS) with verified warm hits.
- Separate existing private `/Z7` and `/DEBUG:FULL` probes capture PDB.
- Ambient OS and tool caches are uncontrolled; source spans and random MSVC temporary paths may affect metadata.
- Sizes are observations, not filtered by equality. Generated C identity is enforced as a gate; other sizes are recorded as-is.
- No runtime-speed, Linux, full-application, or fymap conclusion is made.
- Persistent disk cache: baseline-vs-current comparison not measured; current plain-vs-labels only.
- Disk check uses the current compiler only on the same 64-call synthetic corpus: 3 fresh directories per variant (6 samples), cold store of 1 package, session close, reopen with verified all-hit. File bytes are the sum of actual regular file lengths, not disk allocation blocks or RSS.
- Harness: [measure-argument-labels.ts](../../tools/measure-argument-labels.ts), [cli cost test](../../internal/cli/argument_labels_cost_test.go), [cdriver debug test](../../internal/cdriver/argument_labels_cost_test.go), [disk cost test](../../internal/cli/argument_labels_disk_cost_test.go).
- Evidence: [raw measurements](evidence/argument-label-cost-20260922.json), [disk measurements](evidence/argument-label-disk-cost-20260922.json).

## Results

Frontend check medians:

| corpus | baseline median | current median | labels median | median delta |
|---|---|---:|---:|---:|
| plain synthetic | 16.466 ms | 19.401 ms | 17.428 ms | baseline to current plain +17.818% |
| scanner | 393.885 ms | 401.504 ms | - | baseline to current scanner +1.934% |

Ranges overlap across repetitions and order effects remain possible, so neither a speedup nor no-regression is proven. No statistical significance and no general compile-time regression are claimed.

Go cumulative allocation medians:

| corpus | baseline bytes | current bytes | baseline mallocs | current mallocs |
|---|---|---:|---:|---:|
| plain synthetic | 5451851 | 5462337 | 13348.5 | 13413 |
| scanner | 169224475 | 169673170.5 | 445717.5 | 447153.5 |
| current labels | - | 5732642.5 | - | 14122 |

Allocation counts increase in every paired repetition and byte medians increase, but one plain byte pair is lower on current, so allocation growth is not equated with execution-speed degradation.

Cache and artifacts:

| observation | plain | labels |
|---|---|---:|
| in-memory package-cache serialized payload bytes (not disk cache or RSS) | 339228 | 339644 (+416) |
| persistent disk cache actual file bytes, current compiler only (not allocation blocks or RSS) | 461346 | 461902 (+556) |
| generated C bytes and SHA | 34567, identical | 34567, identical |
| low EXE bytes | 116736 | 116736 |
| optimized EXE bytes | 107520 | 107520 |
| low `unit.obj` bytes | 10970 | 10970 |
| optimized `unit.obj` bytes | 1216 | 1216 |

Disk file bytes were identical on every run per variant (3 of 3 each). The disk check plain source hash matches the primary raw JSON plain source hash, confirming the same synthetic corpus. Low `runtime.obj` varies as 1041 versus 1045 bytes across lanes and variants, so complete object identity is not claimed. Debug PDB low values also vary by 8192 bytes within the same lane, so a label effect must not be inferred from PDB size; temporary-path noise remains possible.

## Execution verification

- Before measurement, formatting and a scoped check were run; the opt-in tests are the measurement itself.
- All 12 measurement samples completed with the expected `LABEL_COST_REPORT` output.
- All 36 ordinary native executions in the synthetic corpus exited 0 with empty stdout and stderr.
- All 12 separate PDB probe executions exited 0 with empty stdout and stderr.
- All 6 disk samples stored 1 package cold, closed the session, reopened it, and verified all-hit.
- Timed check iterations total 240 (96 baseline + 144 current).
- Generated C was byte-identical across lanes and variants; unlabeled source identity also held.
- The full suite was not repeated; the earlier implementation turn hit a 180s timeout, which is not a failure of this measurement. The new opt-in measurement itself fully succeeded.
- No API, runtime, database, or production code change was made; this change is measurement tests, tool and evidence documentation only.
- No product change such as feature removal or public-scope expansion was made in this step.

## Decision

- The strict original condition cannot be claimed as satisfied.
- Cache cost proof fails: +416 bytes in memory and +556 bytes in actual persistent files were observed for the labels corpus.
- Compile-time no-regression is not proven: small-corpus medians moved upward, but ranges overlap and the workload is synthetic and small, so neither a universal regression nor significance is claimed.
- Cause candidates only, without proof: the existing AST pointer addition may enlarge node memory, and cache growth may include longer source-location metadata. Neither cause is proven and no optimization completion is claimed.
- Labels input is longer than plain input, and the synthetic small workload cannot be generalized.
- Recommended next work is to separate unused-code cost causes from metadata-representation cost, improve the metadata representation, and not set an arbitrary cache allowance.
