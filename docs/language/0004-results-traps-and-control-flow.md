# Results, Traps, and Control Flow

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`
- Result fixtures: `spec/fixtures/result/manifest-v0.json`

## Evaluation order

Fairy C evaluates expression operands, aggregate fields, and call arguments from left to right. The frontend normalizes side-effecting subexpressions into ordered temporaries before flow analysis and MIR lowering. A backend cannot rely on its host language's expression order.

## Option and Result

`Option<T>` and `Result<T, E>` are tagged values. Their copyability and cleanup obligations follow their active payloads. Reading a resource payload by value moves it.

`Result` is must-use. It must be matched, returned, propagated with `?`, or discarded with `discard expression;`. `?` is not defined for `Option` in Stage 0.

The explicit discard spelling is `discard expression;`. Discarding `Option` or `Result` drops only the active payload. A restricted user enum cannot contain a resource payload in Stage 0.

Every evaluated Result carries one must-use obligation. Storing it transfers that obligation to the
destination local, and passing it by value transfers the obligation to the callee's Result
parameter. Result parameters therefore begin pending. Return, `discard`, exhaustive Result match,
and `?` discharge the obligation; assignment cannot silently discharge the previous destination.

Any pending obligation at a normal scope exit, function exit, earlier return edge, or overwrite is
`E4101`. A branch join is pending when any reachable incoming path is pending. Loop analysis covers
zero iterations and repeated back edges, so a value produced by one iteration cannot be overwritten
by the next. These rules do not make every Result move-only: copyability still follows the payload,
but every new Result evaluation must occupy a handling or transfer position.

Stage 0 constructs Option variants with compiler-known qualified forms:

```fy
Option::Some(value)
Option::None
```

Both forms require one contextual `Option<T>` type. `Option::Some` carries exactly one payload;
`Option::None` is a bare variant and is not callable. A shallow Option match contains exactly one
`Option::Some(binding)` arm and one bare `Option::None` arm. The None arm creates no binding and no
payload extraction. Wildcards, guards, nested patterns, and reference-style payload bindings remain
outside this slice. Scalar, inert, and resource-bearing active payloads execute through verified MIR.

Stage 0 constructs Result variants with compiler-known qualified paths:

```fy
Result::Ok(value)
Result::Err(error)
```

These forms require one contextual `Result<T, E>` type and do not perform ordinary function
lookup. Shallow matching uses exactly one `Result::Ok(binding)` arm and one
`Result::Err(binding)` arm. The binding is local to its arm, the scrutinee is evaluated once, and
both arms have one compatible result type. Wildcard Result arms, guards, nested patterns, and
ref-style payload bindings are outside this slice.

## Propagation

`expr?` evaluates `expr` once, branches on the result tag, moves the success payload into the continuation, and moves the error payload into the function return path. Normal cleanup for every exited scope runs before that return. No backend receives a `?` operator.

The Result fixtures classify expressions as returned, discarded, propagated, constructed, or
matched, record canonical success and error types, and assert exactly-once evaluation. Their
parameter-driven cases remain valid evidence for handling independent of construction, while the
qualified ADR 0017 constructor and match cases execute through typed HIR. Scalar Result and Option
variants execute through verified MIR. Resource-bearing construction, propagation, active-variant
cleanup, and deterministic drop traces are covered by the MIR interpreter fixtures.

## Normal exits

`return`, `break`, `continue`, and fallthrough clean only the lexical scopes they exit. Cleanup is reverse declaration and reverse field-initialization order. Cleanup obligations are elaborated before backend selection.

`break` and `continue` target the nearest enclosing `while`; either form outside a loop is `E1012`.
Both terminate the reachable portion of their block. Resource Flow requires every normal body,
break, and continue edge to preserve the loop-header state of resources declared outside the loop.
Result Flow requires pending locals in exited scopes to be handled, and continue edges participate
in repeated-iteration overwrite checks. Labelled and value-producing loop exits are not Stage 0.

## Traps

A trap is a non-returning, non-unwinding termination. It performs no language-level drop or region reset. Trap categories are stable semantic outcomes even when the platform-specific termination mechanism differs.

Stage 0 traps include at least bounds failure, ordinary integer overflow, division by zero, signed minimum divided by negative one, invalid shift amount, failed assertion, and an explicitly invoked unreachable condition. Ordinary `i32` negation and matching-type `i32`, `u8`, `u32`, or `usize` addition, subtraction, and multiplication trap when their mathematical result is not representable in that exact type. Unsigned subtraction therefore traps below zero rather than wrapping. Matching-type division and remainder accept the same four types and trap on zero; signed minimum with negative one is the additional `i32` overflow case. Matching-type right shift accepts the same four types. `i32` uses arithmetic right shift and traps when its amount is negative or at least 32; `u8`, `u32`, and `usize` use logical right shift and trap when their amount is at least 8, 32, or 64 respectively on the currently admitted targets. Typed unary `~` and matching-type `&`, `^`, and `|` preserve the exact integer type without trapping.

Left shift is deferred in Stage 0 because its overflow and sign behavior would otherwise need an additional source-level rule. Wrapping, saturating, and checked arithmetic are distinct operations rather than build profiles.

## Foreign control flow

Foreign code may return normally or terminate the process. It cannot unwind, throw, or `longjmp` across a Fairy C frame with live obligations. Integrations that use those mechanisms require a C or C++ barrier shim that catches or translates the event before returning a normal status value to Fairy C.

Stage 0 foreign callbacks are raw `unsafe extern C` function pointers plus explicit user data. A safe callback abstraction is deferred until lifetime, reentrancy, threading, and nonlocal-control rules are specified.

## Assertions and optimizer assumptions

`assert` is always checked. A debug-only assertion, if later introduced, cannot become an optimizer fact. An assumption that permits undefined behavior when false belongs to an explicit unsafe operation and must remain visible in MIR provenance.

## Required diagnostics

The compiler must distinguish an unused `Result`, `?` on a non-`Result`, `?` in an incompatible return context, unsupported left shift (`E1011`), loop control without an enclosing `while` (`E1012`), and forbidden foreign nonlocal control flow.
