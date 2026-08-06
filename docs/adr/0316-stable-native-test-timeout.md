# ADR 0316: Stable native-test process timeout

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0265, ADR 0315

## Decision

1. Add native-only `--timeout-seconds <1..60>` with canonical decimal parsing. Omission preserves 60 seconds.
2. Apply the selected duration only to each test process after compilation, linking, and PE inspection.
3. Timeout is execution infrastructure failure: code 6, no completed report, no later test execution, and no retained artifact.
4. Preserve report schemas and SDK, language, runtime ABI, MIR, target, package, and generated-code versions.

## Evidence

Go tests cover range, mode, delimiter, default, and duration propagation. Windows host smoke runs a
runtime-dependent nonterminating generated-Fairy test at low and optimized MSVC, proving exact
one-second termination, code 6, and an empty report.

## Consequences

Callers can fail fast on stuck native tests. Per-test timeout outcomes, continuation, cancellation
receipts, and semantic-interpreter limits remain separate contracts.
