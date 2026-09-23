# `Result.is_ok` no-cache check timing

Date: 2026-09-23. Compiler source: `ff20349b` plus the measurement test only.
Host: Windows AMD64, Go 1.26.4, `GOMAXPROCS=12`. This is a local diagnostic, not a CI gate or a compiler-wide build-time claim.

## Compared work

The `lexer_candidate` parser workspace uses the same files and language pin in both lanes. Only the two `Vec::push` tests in `src/body_state.fy` differ: the baseline uses the previous exhaustive `switch`, and the candidate uses `.is_ok`. The source hashes emitted by the test are `9b4121e0b91fb0c540e56f440fe535c7d255eb65ccaf9d20b31df81fb5e8f1d3` (switch) and `ad76ffa0dbb4fa3e630aff5aa730b2e30e39914143acf10ccd7e33f5c1d924d7` (`.is_ok`). Both produce 407 MIR functions.

The configured `fairyc_measure_result_is_ok_check_timing` intent runs `TestResultBranchReviewIsOkCheckTiming` with its dedicated test flag. Routine Result tests and the full Go suite skip this timing diagnostic. It loads a fresh workspace before each sample, then times only `workspacecompiler.Check` with Go's monotonic elapsed clock. This includes semantic analysis and verified MIR construction, but excludes source loading, C emission, MSVC compilation, linking, process startup, and persistent cache behavior. Each lane has one untimed warmup. Eight measured pairs alternate which lane runs first; there are no sample removals or timing retries after the valid run. Every check must verify successfully with no diagnostics.

## Raw measured check durations

All durations are milliseconds. The pair index joins sequential observations; first lane alternates.

| Pair | First lane | Switch | `.is_ok` | Paired ratio |
|---:|---|---:|---:|---:|
| 1 | switch | 675.7585 | 661.2634 | 0.978550 |
| 2 | `.is_ok` | 628.1292 | 751.9864 | 1.197184 |
| 3 | switch | 662.4074 | 703.8396 | 1.062548 |
| 4 | `.is_ok` | 751.1138 | 768.6585 | 1.023358 |
| 5 | switch | 773.0149 | 807.1509 | 1.044160 |
| 6 | `.is_ok` | 747.5808 | 805.7285 | 1.077781 |
| 7 | switch | 686.4642 | 735.8899 | 1.072000 |
| 8 | `.is_ok` | 699.5986 | 684.6847 | 0.978682 |

Switch median: 693.0314 ms. `.is_ok` median: 743.93815 ms. Ratio of medians: 1.073455; median of paired ratios: 1.053354. The relative full span is about 20.9% for switch, 19.6% for `.is_ok`, and 20.8% for paired ratios. The two median estimators disagree by about two percentage points, and one pair ranges as high as 1.197184.

## Decision and limits

The measured `.is_ok` checks were slower in this local run, but the spread prevents a reliable regression conclusion. No compiler-time pass or fail threshold was defined, and the C-relative 8% runtime and EXE budget does not apply to this check-time comparison. The existing native size and cleanup checks passed in the same configured test intent. No language, SDK, backend, or version change follows from this observation.
