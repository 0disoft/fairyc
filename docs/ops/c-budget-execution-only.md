# C budget execution-only diagnostic

Date: 2026-09-22. Host: Windows AMD64, Go 1.26.4, GOMAXPROCS 1, logical CPUs 12, MSVC 14.51.36231.
Compiler source: `8ae04669791ef793ece02fc4f8652eaef8baf8e7`. Test executable identity: `fairyc.compiler-executable-sha256/v1:93242604fcaea0697160d651dde72914be2723843c49a591708d5e2dff364380`.
Flags: `/std:c11 /TC /W4 /WX /MT /O2 /INCREMENTAL:NO`. Seed 12345, rounds 50000000. Runtime budget ratio 1.08, unchanged.

## Purpose

- Diagnostic stability guard for execution timing, separate from the 8% budget. Not a CI certification and not a language-wide claim.
- The 10% relative-full-span heuristic was fixed before results for each lane and for paired ratios; exceeding it defers the verdict even when medians look inside budget. Order-group or median-estimator disagreement at 1.08 also defers.

## Method

- Build all four images exactly once before any native execution: checksum C/Fairy and classifier C/Fairy. No timer or execution runs during the build phase.
- Reuse identical artifact bytes for 3 warmup pairs and 12 measured pairs per kernel, alternating first lane. The native driver materializes the same bytes in a fresh temporary executable path per invocation; QPC kernel time excludes that write, process startup, input parsing, and output.
- Every result is checked against the Go reference. No rebuilds, sample removal, retries, affinity or power changes, or adaptive stopping.
- Counts: 4 builds, 80 native executions = 48 measured + 12 warmup + 20 boundary. Artifact bytes verified unchanged during measurement.
- Build-time comparisons are intentionally absent, so no compile-time conclusion is made. Ambient host load and frequency are uncontrolled.
- Harness: [c_budget_execution_test.go](../../internal/cli/c_budget_execution_test.go), [c_budget_execution_stats_test.go](../../internal/cli/c_budget_execution_stats_test.go).
- Evidence: [raw measurements](evidence/c-budget-execution-only-20260922.json).

## Provenance

Full source and harness hashes live in the JSON `source_sha256` and `harness_sha256` fields. The JSON `builds` array carries full EXE, unit, and support SHA256 values with byte sizes, and each sample links its `exe_sha256`.

## Results

Ratio of medians and median of paired ratios are different estimators and are labeled separately. Medians alone do not declare a pass.

| workload | C median | Fairy median | ratio of medians | median of paired ratios | relative full span C / Fairy / paired | order groups C-first / Fairy-first | status |
|---|---|---:|---:|---:|---:|---:|---:|
| checksum | 357.07705 ms | 329.10855 ms | 0.9216737676084196 | 0.9286804626942678 | 0.192213 / 0.143765 / 0.114088 | 0.9334081943306652 / 0.923374978935539 | inconclusive |
| classifier | 280.29265 ms | 290.55590 ms | 1.0366161938245615 | 1.016578969139755 | 0.146503 / 0.186468 / 0.173233 | 1.0021568393256992 / 1.016578969139755 | inconclusive |

Exact JSON statuses copied: both `status` values are `inconclusive` with `reasons` `["relative full span exceeds predeclared 10% stability limit"]`. All 80 native executions passed with Go-result checks, and the summary unit tests passed; the verdict is still deferred because every lane and paired span exceeds the predeclared 10% limit.

## Decision

- No within-budget or exceeds-budget label is assigned from medians; both workloads are inconclusive under the predeclared stability rule.
- No repeat measurement is planned. All original historical raw files are untouched.
- No compiler, product, or version change was made.
