# ADR 0398: Result branch local continuation

- Status: Accepted (existing-syntax refactor)
- Date: 2026-09-22
- HEAD: `9c14543b`

## Decision

Refactor `add` in [diagnostics.fy](../../examples/lexer_candidate/src/diagnostics.fy) to existing syntax only: `return switch (bytes::copy(token))` with `case .ok(owned)` constructing `Entry` plus nested `switch (v::push(...))`, and `case .err(_)` returning `3`. No new `let-else`, `never`, or fallthrough type; no compiler, API, SDK, or language-version change.

## Context

`checkResultMatch` in [typecheck.go](../../internal/semantic/typecheck.go) requires all arm types to match. `checkBlock` types a return-only arm with no tail as void, so extracting a resource with `.err` returning `3` is rejected with `E1006`. `lowerBuiltinMatch` in [lower.go](../../internal/mir/lower.go) already skips terminated blocks, but that alone does not admit the typing.

## Options

 1. Branch-local continuation (selected): each arm yields an i32 tail value; the enclosing return exits the function (no new return-arm semantics); fits this case with no grammar change.
2. Diverging-arm typing (deferred): admit diverging arms producing an outer value; revisit only when a real consumer needs the value outside and cannot reasonably restructure.
3. `let-else` (deferred): adds grammar, scope, and ownership coverage; not justified for this single refactor.

## Preserved behavior

Single evaluation moves `owned` exactly once. Copy failure returns `3` with no `Entry` or push. Push failure returns `3` with `Entry` cleanup. Success and limit statuses `0`/`4`, special code `10` at length `257`, and `add_limited` budget `5` are unchanged.

## Evidence

At this ADR's date, the resource-initializer `.err` return form was rejected with `E1006`. [ADR 0402](0402-result-return-only-match-arm.md) later admitted that exact direct-return arm; [TestResultBranchReviewReturnOnlyArm](../../internal/cli/result_branch_review_test.go) now verifies acceptance. `TestResultBranchReviewDiagnosticsNative` passed 4 scenarios x 2 MSVC profiles (8 executions): success status `0` alloc 2 free 2; copy-failure status `3` alloc 0 free 0 failures 1; push-failure status `3` alloc 1 free 1 failures 1; limits status `0` alloc 529 free 529. All live bytes and blocks `0`. Limits: 256 normal adds `0`; `add_limited` at count 256 limit 256 returns `5` leaving length 256; 257th normal code 1 returns `4`; code 10 at 257 returns `0` and at 258 returns `4`. No broader lexer or perf-benchmark tests were run for this narrow change; no perf claim.

## Scope

ADR only; implementation is the small example refactor. Product scope stays in [02-spec.md](../product/02-spec.md); index in [README.md](README.md). Prior loop-label deferral is separate: [ADR 0397](0397-defer-named-loops-pending-workload-evidence.md).
