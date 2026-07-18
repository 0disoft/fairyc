# ADR 0037: Run generated C under a closed sanitizer mode

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`
- Implementation source: `internal/cdriver/` and `internal/cbackendnative/`

## Context

The native differential lane compares interpreter outcomes, process status, and compiler-owned drop
order across Clang/GCC and low/optimized builds. Those oracles can still miss executed C undefined
behavior or memory errors when the process happens to return the expected status. Conversely, a
sanitizer result cannot define Fairy C semantics: it sees only executed generated-C paths, inherits
toolchain instrumentation limits, and cannot prove that a value or cleanup sequence is correct.

Making sanitizers a product optimization profile would also be the wrong boundary. It would expand
the experimental CLI, alter ordinary artifacts, and let validation-only runtime policy become a
user-facing compiler contract before the build-profile design is settled.

## Decision

1. `cdriver.Build` continues to accept exactly the product `low` and `optimized` profiles. The
   experimental CLI continues to reject every other profile name.
2. A separate `cdriver.BuildSanitized` entry point is available only to internal native
   conformance code. Its closed identity is `address-undefined-o1`; it compiles and links with
   `-O1`, `-fsanitize=address,undefined`, `-fno-omit-frame-pointer`, and
   `-fno-sanitize-recover=all`.
3. The sanitizer lane consumes the production, uninstrumented C unit and scalar trap runtime. It
   never links the drop observer, so stderr remains sanitizer diagnostics rather than a mixed
   binary/text protocol.
4. Every one of the seventeen admitted C-emission cases is compiled and executed once with Clang
   and once with GCC under the sanitizer mode. A clean configured report therefore records 34
   sanitizer executions in addition to the existing 68 low/optimized drop-trace executions.
5. The runner removes ambient `ASAN_OPTIONS`, `LSAN_OPTIONS`, and `UBSAN_OPTIONS` and supplies
   deterministic fail-closed values. Address and undefined-behavior findings abort; leak detection
   is enabled; symbolization and color are disabled; any sanitizer stderr, timeout, excess output,
   unexpected stdout, or process-status mismatch fails the lane.
6. The native report advances from `fairyc.native-c-report/v3` to
   `fairyc.native-c-report/v4`, adding exact `sanitizer` identity and
   `sanitizerExecutions` fields. The existing `cases` list is the sanitizer case identity set; a
   second duplicate list is intentionally omitted.
7. Sanitizer success is backend evidence only. The interpreter and versioned fixtures remain the
   semantic oracle, while the ordinary low/optimized differential remains the optimization-
   independence oracle.
8. No hosted Linux success receipt has been observed. Unit tests prove the closed flags,
   environment override, report shape, and CLI exclusion on the current Windows host; they do not
   pretend to execute ASan or UBSan.

## Consequences

- Executed generated-C memory and undefined-behavior faults have a fail-closed detection path under
  both required compiler families.
- Product C text, product artifacts, CLI options, and ordinary optimization flags remain unchanged.
- The current scalar runtime allocates no heap memory, and trap paths terminate through `_Exit`, so
  leak detection adds little evidence today and does not run an exit sweep on those trap paths.
- Sanitizers do not inspect inactive branches, prove public layout or calling convention, execute
  user drop-body effects, replace bidirectional ABI fixtures, or establish target conformance.

## Rejected alternatives

- Add `--profile sanitized` to `experimental build`: turns an internal validation policy into a
  premature user-facing contract.
- Sanitize only Clang: loses the independent GCC generated-C path and can hide compiler-specific
  instrumentation or lowering behavior.
- Mix sanitizer diagnostics with drop-trace stderr: makes both protocols ambiguous and lets binary
  trace bytes corrupt actionable sanitizer output.
- Treat a clean sanitizer run as proof of language safety: confuses absence of a finding on executed
  paths with semantic correctness or universal memory safety.
