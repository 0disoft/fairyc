# C performance budget baseline

Date: 2026-09-22. Host: Windows AMD64, Go 1.26.4, GOMAXPROCS 1, MSVC 14.51.36231.
Compiler source: `7983ab0db07fd83d4fe28f7c4b9d88ab9cf3cc78`. Profile: optimized.
Flags: `/std:c11 /TC /W4 /WX /MT /O2 /INCREMENTAL:NO`. Seed 12345, rounds 50000000.

## Budget policy

- Runtime budget: 8% cumulative against equivalent C per workload, never a per-feature allowance.
- EXE budget: 8% cumulative against equivalent C per workload, never a per-feature allowance.
- Build time: 8% is an observation-only provisional marker per workload; cold CLI startup was not measured.
- Cache and debug sizes are tracked separately and are not part of the runtime/EXE budget.
- This 8% policy replaces the previous strict-zero and interim 5% policies by user agreement. The original run recorded 5%; the same preserved numbers are re-evaluated here under 8% without a new measurement. The raw evidence JSON is left unchanged. The earlier cache +0.12% alone is therefore not blocking.

## Method

- Two independently authored scalar kernels: checksum and classifier, each in Fairy and hand-written C (not generated C against itself).
- Same MSVC optimized flags, same stdin input, same shared QPC/output support harness.
- Stdin input prevents constant folding. Kernel time excludes input parsing, process startup, and printing; the result is consumed and checked against an independent Go reference.
- Build time is full in-process translate plus compile/link/write EXE: source read/load, frontend analysis/verified MIR, C emission, C compile/link, artifact reads, and final EXE write. Common warmed toolchain discovery, harness startup, and workspace scaffolding are excluded.
- C does not link the Fairy runtime; Fairy keeps ordinary checked arithmetic and full required product support.
- Both lanes guard `seed < 65521` and `rounds <= 50000000`. The C sources carry math-proof comments that admitted arithmetic stays in range; no UB is permitted. Error-path behavior is not compared for binary equivalence.
- Five alternating paired samples after one build/run warmup per workload and lane. Boundary inputs `(0,0), (65520,0), (0,1), (65520,1), (12345,257)` add 5 inputs x 2 workloads x 2 lanes = 20 extra executions.
- 2 kernels x 2 lanes x (1 warmup + 5 reps) = 24 builds with main executions, plus 20 boundary executions = 44 native executions, all matched to the Go reference.
- No persistent package cache; ambient OS caches uncontrolled. No PDB, Linux, real parser/container workload, or cold CLI startup claim.
- Harness: [c_budget_baseline_test.go](../../internal/cli/c_budget_baseline_test.go), [checksum.fy](../../tools/benchmarks/c-budget/checksum.fy), [checksum.c](../../tools/benchmarks/c-budget/checksum.c), [classifier.fy](../../tools/benchmarks/c-budget/classifier.fy), [classifier.c](../../tools/benchmarks/c-budget/classifier.c), [support.c](../../tools/benchmarks/c-budget/support.c).
- Evidence: [raw measurements](evidence/c-budget-baseline-20260922.json).

## Results

| workload | C runtime median | Fairy runtime median | delta | paired Fairy/C |
|---|---|---:|---:|---:|
| checksum | 327.6545 ms | 343.1993 ms | +4.7443% | 1.05836, 1.05374, 1.03340, 1.04993, 1.06223 |
| classifier | 236.4271 ms | 306.4154 ms | +29.6025% | 1.2659..1.3184, all 5 exceed |

Do not confuse the ratio of separate medians with the median of paired ratios. The table deltas +4.7443% and +29.6025% are ratios of separate medians and are kept as observations. The medians of the paired runtime ratios are +5.374% for checksum and +29.852% for classifier. Both estimators are shown together so that neither favorable statistic is selected. Under the current 8% policy, both checksum estimators sit inside the observed 8%.

| workload | C EXE | Fairy EXE | delta |
|---|---|---:|---:|
| checksum | 167936 | 168960 | +0.60976% |
| classifier | 168448 | 168960 | +0.30395% |

EXE bytes were identical on every repetition per workload and lane. Build medians are observation only: checksum 547.4054 ms to 531.9355 ms (-2.826%), classifier 534.6068 ms to 548.3339 ms (+2.568%). The medians of the paired build ratios are -0.828% for checksum and +1.711% for classifier, with wide pair spread (checksum about -5.8% to +8.0%, classifier about -0.5% to +18.6% with rep4 near +18.6%), so no general build claim is made.

## Decision

- Checksum is within the observed 8%: the ratio of medians is +4.7443% while the paired-ratio median is +5.374%. This is a first-pilot observation only, and no hard rule is created that every noisy sample must fall inside a fixed 8%.
- Classifier still exceeds on this workload: the ratio of medians is +29.6025%, the paired-ratio median is +29.852%, and all 5 pairs exceed.
- EXE sizes meet 8% on both workloads.
- No whole-language no-regression is claimed and no whole-language certification is made: this baseline covers 2 synthetic kernels on Windows O2 only and does not certify the full language.
- This result is not judged as label cost: the prior measurement kept generated C identical between plain and labels.
- Recommended next work is classifier emitted-C inspection, range-proof review, and branch-lowering cause separation first; the execution bottleneck outranks small AST allocation work.
- No development-code API, ABI, database, or version change was made.
