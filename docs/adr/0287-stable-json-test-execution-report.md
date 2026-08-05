# ADR 0287: Stabilize JSON test execution reports

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Schema source: `spec/cli/stable-test-report-v1.schema.json`
- Depends on: ADR 0181, ADR 0265, ADR 0267, ADR 0273

## Context

Stable test identity discovery already has `fairyc.test-list/v1`, but semantic and native execution
still produce only a human text report. CI, editors, and LLM tooling must therefore parse prose to
distinguish a passing test, a language trap, the selected engine, and lock state. The runner already
owns all of those facts after the complete selected suite finishes, so a machine report does not
require new test semantics or native harness behavior.

Exposing partial JSON while tests are running would make writer failure, process failure, and trap
continuation ambiguous. The first contract should preserve the existing atomic buffered report
boundary.

## Decision

1. Stabilize `fairyc test --json [--native [--release]] [--exact <test-identity>]... [--locked]`.
   `fairyc test --list --json [--locked]` retains its existing `fairyc.test-list/v1` output.
2. Successful completion of the selected execution set emits one compact newline-terminated
   `fairyc.test-report/v1` object. The report is written only after every selected test has a
   classified outcome; compilation, capability, native-build, or execution-infrastructure failure
   emits no JSON report.
3. Required top-level fields are `schemaVersion`, `engine`, `target`, `profile`, `tests`, `count`,
   `passed`, `failed`, `status`, and `lock`. Interpreter reports use engine `interpreter` with null
   target and profile. Native reports use engine `native`, target `x86_64-windows-msvc`, and profile
   `development` or `release`.
4. Each canonical-order test entry contains required `identity`, `outcome`, and `trapCategory`.
   Outcome is `passed` with null trap category or `failed` with the exact validated language-trap
   category. No path, source text, timing, program output, host inventory, private harness identity,
   or temporary artifact enters the report.
5. `count` equals the outcome-array length and equals `passed + failed`. Status is `ok` exactly when
   failed is zero, otherwise `failed`. Empty selections produce an empty array and zero counts.
6. JSON preserves the existing process codes: all passing is 0, one or more classified language
   traps is 5 after the complete JSON report, and invocation/source/internal/native/package failures
   retain codes 2/3/4/6/7. Writer failure may accept a prefix and returns 4.
7. Preserve the human execution report byte-for-byte when `--json` is absent. Exact selection,
   canonical order, complete-overlay admission, trap continuation, native isolation, output
   rejection, lock behavior, and artifact cleanup remain unchanged.
8. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.26`, runtime ABI `fairyc-runtime/0`, manifest,
   lockfile, package identity, MIR, and generated native code unchanged.

## Consequences

- Automation can consume semantic and native results without parsing prose or guessing which engine
  ran.
- Previously invalid execution-mode `--json` becomes valid; every invocation without it remains
  compatible.
- Schema v1 deliberately omits durations, retries, stdout/stderr attachments, source locations,
  ignored/skipped states, workspace-member results, host/toolchain inventory, and flaky-test policy.
- Future portable native targets require a new admitted target value and corresponding evidence;
  callers must not infer portability from the generic `native` engine name.

## Evidence

- `internal/cli/workspace_tests.go` owns option parsing, canonical outcome collection, and atomic JSON
  encoding.
- `spec/cli/stable-test-report-v1.schema.json` owns the machine-readable field and value contract.
- `internal/cli/workspace_tests_test.go` proves semantic trap continuation, empty reports, writer
  failure, native identity/profile encoding, and unchanged text behavior.
- `internal/cli/schema_test.go` guards top-level and per-outcome schema/Go field agreement.
- `cmd/fairyc-windows-host-smoke/main.go` runs the separately built compiler through semantic JSON
  execution and an optimized isolated native JSON trap-plus-pass execution.
