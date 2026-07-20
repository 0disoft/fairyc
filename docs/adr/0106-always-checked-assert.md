# ADR 0106: Always-checked assert

- Status: Accepted
- Date: 2026-07-20
- Language surface: `assert(condition);`

## Context

The trap taxonomy already reserves `assertion-failed`, and the language contract says assertions
must not disappear in optimized builds. The Stage 0 source surface did not yet expose that rule, so
programs could only spell equivalent branching manually. Reusing C's build-mode-sensitive assert
macro would violate Fairy C's optimization-independent safe semantics.

## Decision

1. `assert(condition);` is a statement. Parentheses, one condition, and the terminating semicolon
   are mandatory. The condition has type `bool` and evaluates exactly once.
2. A true condition continues normally. A false condition traps with the exact category
   `assertion-failed`; it does not return a value or unwind resource cleanup.
3. Assertions execute in every compiler profile. They are not debug-only instrumentation and are
   not optimizer assumptions. The compiler may remove one only after ordinary semantics prove the
   condition true without adding evaluation or changing effects.
4. Typed HIR retains a distinct `assert` statement. MIR lowers it to a typed `assert` switch whose
   first successor is continuation and whose second successor is a non-returning
   `assertion-failed` trap.
5. The MIR verifier requires one exact `bool` operand, distinct success and failure edges, and the
   canonical failure trap. A malformed shape is `V1001`; a missing or redirected failure edge is
   `V1501`.
6. Boolean immediates retain `bool` identity in MIR instead of being silently reclassified as
   `i32`. The interpreter and C emitter reject incompatible condition metadata.
7. Generated C compares the normalized condition with zero and calls the existing trap runtime on
   failure. It does not include or depend on the C library `assert` macro.
8. Assertion messages, debug-only assertions, unsafe assumptions, contracts, and compile-time
   assertions remain separate decisions.
9. The interpreter fixture contract grows from 82 to 84 cases and C emission from 28 to 29. The
   derived Linux matrix plans 116 compiler/profile executions and 58 sanitizer executions. The
   Windows smoke proves success and failure at low and optimized profiles without WSL.

## Consequences

- FY programs can state executable invariants with one backend-independent trap identity.
- Release optimization cannot silently erase a safety check.
- This is a control-flow primitive, not a formatting or diagnostic-message facility.

## References

- `docs/language/0004-results-traps-and-control-flow.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
- `docs/compatibility/versioning-v0.md`
