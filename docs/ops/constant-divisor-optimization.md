# Constant divisor optimization

Date: 2026-09-22. Host: Windows AMD64, Go 1.26.4, GOMAXPROCS 1, MSVC 14.51.36231, optimized `/O2`.
Before compiler: `69b526fc5e6d3dfed1c526c0f9e4d694f1fd275d`. After: same source plus constant-divisor patch.
Scope is implementation-only: no language or SDK contract change, so no version bump. Cache keys use executable build identity.

## Limitation first

Ambient drift between phases is large, so absolute before-after speedup is not claimed and sustained 8% certification is not claimed.
C checksum median moved 326.556 ms before to 164.573 ms after; C classifier median moved 128.025 ms before to 238.897 ms after.
Only the alternating sequential C/Fairy comparison inside each phase is interpretable, and this report judges the after phase. No whole-language claim is made: 2 synthetic kernels on Windows O2 only.

## Optimization scope

- Emits direct unsigned nonzero literal divisors for `u8`, `u32`, and `usize` only.
- Signed, zero, and dynamic divisors are unchanged; all MIR trap checks are unchanged.
- Before Fairy classifier generated code kept two volatile checked-divisor slot uses with two `div` instructions in disassembly.
- After: zero `div` ops in both Fairy kernels; hand-authored C had zero `div` ops in both phases. Trap checks remain.

## Method

- Same two-kernel harness as the C budget baseline: checksum and classifier, Fairy versus hand-written C, same flags, stdin input, shared QPC/output support.
- Five alternating paired samples after one build/run warmup per workload and lane; 24 builds with main executions plus 20 boundary executions, 44 native executions per phase, all matched to the Go reference.
- Correctness: targeted C emitter tests PASSED, plus 6 cases x 2 profiles native PASSED covering unsigned `u8`/`u32`/`usize` max, 1, 128, and max divisor, plus unsigned literal zero, remainder zero, dynamic zero, and signed `MIN`/`-1` quotient/remainder behavior.
- Evidence: [before](evidence/constant-divisors-before-20260922.json), [after](evidence/constant-divisors-after-20260922.json).
- Artifacts inspected read-only: `.cache/constant-divisors/{before,after}/{checksum,classifier}/{c,fairy}/unit.{c,asm}`.

## Results

After-phase alternating sequential ratios are the only judged comparison. Ratio of separate medians and median of paired ratios are both shown.

| workload | C median | Fairy median | ratio of medians | median of paired ratios | pair spread |
|---|---|---:|---:|---:|---:|
| checksum after | 164.573 ms | 152.628 ms | 0.927416 | 0.927243 | 0.59410..1.42046 |
| classifier after | 238.897 ms | 240.363 ms | 1.006137 | 0.995351 | 0.98384..1.05416 |

Observed after medians sit inside 8% on both workloads under both estimators, but checksum has high variance with rep3 Fairy and rep4 C spikes, so sustained budget compliance is inconclusive from timing alone. Before-phase numbers are preserved but not compared absolutely: checksum ratio of medians 1.040858 with paired median 1.03530, classifier ratio of medians 1.770264 with paired median 1.68247.

| workload | C EXE | Fairy EXE | EXE delta | C unit | Fairy unit before -> after |
|---|---|---:|---:|---:|---:|
| checksum | 167936 | 168960 | +0.60976% | 817 | 4299 -> 4151 |
| classifier | 168448 | 168960 | +0.30395% | 1020 | 6181 -> 6033 |

EXE bytes were identical on every repetition per workload and lane in both phases. Build times are noisy observation only, so no general build claim is made. After-phase ratios of separate build medians are 1.043214 for checksum and 0.963922 for classifier, observation only; after-phase ambient and per-rep build movement is large in both lanes.

## Decision

- After-phase observed runtimes sit inside 8% on both workloads under both estimators, but checksum timing variance makes sustained budget compliance inconclusive; this is a two-kernel observation, not a language-wide certification.
- EXE sizes are unchanged by the patch and stay inside the 8% EXE budget on both workloads (+0.60976% checksum, +0.30395% classifier).
- No API, ABI, database, or version change was made.
