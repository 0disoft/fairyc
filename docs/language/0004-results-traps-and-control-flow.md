# Results, Traps, and Control Flow

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`
- Result fixtures: `spec/fixtures/result/manifest-v0.json`

## Evaluation order

Fairy C evaluates expression operands, aggregate fields, and call arguments from left to right. The frontend normalizes side-effecting subexpressions into ordered temporaries before flow analysis and MIR lowering. A backend cannot rely on its host language's expression order.

## Option and Result

`Option<T>` and `Result<T, E>` are tagged values. Their copyability and cleanup obligations follow their active payloads. Reading a resource payload by value moves it.

`Result` is must-use. It must be switched over, returned, propagated with `?`, handled by `.is_ok`, or discarded with
`discard expression;`. `?` is not defined for `Option` in Stage 0.

The explicit discard spelling is `discard expression;`. Discarding `Option` or `Result` drops only the active payload. A restricted user enum cannot contain a resource payload in Stage 0.

Every evaluated Result carries one must-use obligation. Storing it transfers that obligation to the
destination local, and passing it by value transfers the obligation to the callee's Result
parameter. Result parameters therefore begin pending. Return, `discard`, exhaustive Result switch,
`.is_ok`, and `?` discharge the obligation; assignment cannot silently discharge the previous destination.

`value.is_ok` evaluates one owned `Result<T, E>` once and yields `bool`: true for `.ok`, false for
`.err`. It handles that Result exactly like `switch (value) { case .ok(_): { true } case .err(_): { false } }`.
The active payload is dropped if it owns resources; both possible payload types must allow implicit
drop and the caller must declare their required cleanup effects. A resource-bearing local moves into
the test, while copyable locals retain the ordinary copy rules. Borrowed Results, payload extraction,
and an `Option.is_ok` property are not admitted.

Any pending obligation at a normal scope exit, function exit, earlier return edge, or overwrite is
`E4101`. A branch join is pending when any reachable incoming path is pending. Loop analysis covers
zero iterations and repeated back edges, so a value produced by one iteration cannot be overwritten
by the next. These rules do not make every Result move-only: copyability still follows the payload,
but every new Result evaluation must occupy a handling or transfer position.

The error alternative of `?` is an earlier return edge for these checks. Its
operand is handled first, then other pending obligations must already have been
handled or transferred; a `discard` after `?` on the success path is too late.

Stage 0 constructs Option variants with contextual leading-dot forms:

```fy
.some(value)
.none
```

Both forms require one contextual `Option<T>` type. `.some` carries exactly one payload;
`.none` is a bare variant and is not callable. A shallow Option switch contains exactly one
`case .some(binding)` and one bare `case .none`. The `.none` case creates no binding and no
payload extraction. Wildcards, guards, nested patterns, and reference-style payload bindings remain
outside this slice. Scalar, inert, and resource-bearing active payloads execute through verified MIR.
A case block with no tail whose final statement is literal `assert(false);` does not
contribute a value to an Option switch join; one other case must continue with a value.

Stage 0 constructs Result variants with contextual leading-dot forms:

```fy
.ok(value)
.err(error)
```

These forms require one contextual `Result<T, E>` type and do not perform ordinary function
lookup. A shallow switch uses exactly one `case .ok(binding)` and one
`case .err(binding)`. The binding is local to its case, the scrutinee is evaluated once, and
both continuing cases have one compatible result type. A case block with no tail and a final
direct `return` contributes no value to the join; its return value must still match the
enclosing function's result type. A case block with no tail whose final statement
is literal `assert(false);` also contributes no value, and one other case must
continue with a value. Neither form adds a general `never` type. Wildcard Result cases, guards, nested patterns, and
ref-style payload bindings are outside this slice.

## Propagation

`expr?` evaluates `expr` once, branches on the result tag, moves the success payload into the continuation, and moves the error payload into the function return path. Normal cleanup for every exited scope runs before that return. No backend receives a `?` operator.

The Result fixtures classify expressions as returned, discarded, propagated, constructed, or
switched, record canonical success and error types, and assert exactly-once evaluation. Their
parameter-driven cases remain valid evidence for handling independent of construction, while the
contextual constructor and switch cases execute through typed HIR. Scalar Result and Option
variants execute through verified MIR. Resource-bearing construction, propagation, active-variant
cleanup, and deterministic drop traces are covered by the MIR interpreter fixtures.

## Normal exits

`return`, `break`, `continue`, and fallthrough clean only the lexical scopes they exit. Cleanup is reverse declaration and reverse field-initialization order. Cleanup obligations are elaborated before backend selection.

`break` and `continue` target the nearest enclosing `while` or classic `for`; either form outside a
loop is `E1012`. Both terminate the reachable portion of their block. In classic `for`, `continue`
performs exited-scope cleanup and then executes the post clause before the next condition; `break`
performs its cleanup and skips the post clause. Resource Flow requires every normal body, break,
and continue edge to preserve the loop-header state of resources declared outside the loop. Result
Flow requires pending locals, including Result-valued pattern captures, in exited scopes to be handled, and continue edges participate in
repeated-iteration overwrite checks. Labelled and value-producing loop exits are not Stage 0.

Compound assignment reuses the same trap categories as its underlying binary operator. It captures
the old scalar value and completes the checked operation before the final store, so any bounds,
overflow, division, remainder, or shift trap leaves the destination unchanged and performs no
language-level cleanup.

`defer call(...);` registers one bounded lexical cleanup call. Its copy-scalar arguments evaluate
immediately from left to right and are retained by value. Calls registered in the same block execute
in reverse order when that block falls through or is crossed by `return`, Result propagation,
`break`, or `continue`. Return-value evaluation completes first; deferred calls then run before
implicit resource drops and Region resets on the same edge. A trap never runs deferred calls, and a
trap inside a deferred call stops the remaining cleanup chain because Stage 0 has no unwinding.

## Traps

A trap is a non-returning, non-unwinding termination. It performs no language-level drop or region reset. Trap categories are stable semantic outcomes even when the platform-specific termination mechanism differs.

Stage 0 traps include at least bounds failure, ordinary integer overflow, division by zero, signed minimum divided by negative one, invalid shift amount, failed assertion, and an explicitly invoked unreachable condition. Ordinary `i32` negation and matching-type `i32`, `u8`, `u32`, or `usize` addition, subtraction, and multiplication trap when their mathematical result is not representable in that exact type. Unsigned subtraction therefore traps below zero rather than wrapping. Matching-type division and remainder accept the same four types and trap on zero; signed minimum with negative one is the additional `i32` overflow case. Matching-type shifts accept the same four types. `i32` right shift is arithmetic, unsigned right shift is logical, and any negative `i32` amount or amount at or above the exact 32-, 8-, 32-, or 64-bit width traps as `invalid-shift`. Left shift uses the same width rule, then traps as `integer-overflow` when `value * 2^amount` is not representable in the exact type. Invalid amount is checked before overflow. Typed unary `~` and matching-type `&`, `^`, and `|` preserve the exact integer type without trapping.

Wrapping, saturating, and checked arithmetic remain distinct operations rather than build profiles.

## Foreign control flow

Foreign code may return normally or terminate the process. It cannot unwind, throw, or `longjmp` across a Fairy C frame with live obligations. Integrations that use those mechanisms require a C or C++ barrier shim that catches or translates the event before returning a normal status value to Fairy C.

Stage 0 foreign callbacks are raw `unsafe extern C` function pointers plus explicit user data. A safe callback abstraction is deferred until lifetime, reentrancy, threading, and nonlocal-control rules are specified.

## Assertions and optimizer assumptions

`assert(condition);` requires one `bool`, evaluates it once, continues when true, and traps as
`assertion-failed` when false. It is always checked at every optimization profile. A debug-only
assertion, if later introduced, cannot become an optimizer fact. An assumption that permits
undefined behavior when false belongs to an explicit unsafe operation and must remain visible in
MIR provenance. Messages and compile-time assertions are not part of this statement.

## Required diagnostics

The compiler must distinguish an unused `Result`, `?` on a non-`Result`, `?` in an incompatible return context, loop control without an enclosing loop (`E1012`), and forbidden foreign nonlocal control flow.
