# ADR 0314: Stable native-test output attachments

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Schema: `spec/cli/stable-test-report-v4.schema.json`
- Depends on: ADR 0207, ADR 0265, ADR 0287, ADR 0300, ADR 0313

## Context

Stable native tests capture bounded process stdout and stderr internally but reject any non-trap
program output so text and v1-v3 JSON reports remain unambiguous. That prevents generated-Fairy
tests from retaining diagnostic bytes emitted before a pass or trap. Mixing arbitrary binary bytes
into the text report would break its grammar and UTF-8 assumptions. Treating Go strings as JSON text
would also replace invalid UTF-8 and lose exact process output.

## Decision

1. Advance stable workspace test behavior to v13 with opt-in
   `fairyc test --native --json --capture-output ...`. The flag requires both `--native` and
   `--json`; semantic execution, text execution, and `--list` reject it. A literal
   `--capture-output` after the first program-argument delimiter remains an ordinary process
   argument.
2. Emit `fairyc.test-report/v4`. Every outcome adds required `stdoutBase64` and `stderrBase64`
   strings using canonical padded RFC 4648 base64. Empty streams are empty strings. `source` remains
   present only when `--locations` is requested.
3. Preserve exact captured bytes without UTF-8 decoding. For a recognized language trap, remove only
   the exact private trap receipt suffix before encoding stderr; retain all preceding program bytes.
4. Preserve canonical test order, pass/trap continuation, target/profile identity, totals, status,
   lock state, exit codes, argument and stdin behavior, temporary-directory isolation, and artifact
   non-retention. Existing bounded process capture and its shared 64 KiB ceiling remain fail-closed;
   cross-stream ordering is not represented.
5. Without `--capture-output`, retain exact text and v1-v3 JSON bytes and continue rejecting native
   program output as code 2. Compilation, toolchain, inspection, execution-infrastructure, capture-
   limit, and package failures still emit no JSON report.
6. Keep SDK `0.0.0-stage0.45`, language `fairyc-language/10`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, interface, cache, and generated-code versions unchanged.

## Evidence

- Go tests cover flag-domain validation, v4 schema/struct field closure, exact invalid-UTF-8-safe
  base64 encoding, empty streams, default output rejection, trap-receipt stripping, and continuation.
- The checked-in `examples/native_test_output_quickstart` writes binary stdout from a passing test
  and binary stderr before a runtime-dependent trap in a later test.
- The configured Windows host smoke runs both tests in development and release, verifies exact v4
  JSON and lock states, and structurally inspects all four generated PE images.

## Consequences

Machines can retain exact native-test logs without parsing text output or corrupting binary bytes.
Human text rendering, live/no-capture streaming, per-stream timestamps, cross-stream ordering,
per-test capture limits, truncation reports, portable native targets, parallel execution, and retries
remain separate contracts.
