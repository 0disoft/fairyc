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

## Propagation

`expr?` evaluates `expr` once, branches on the result tag, moves the success payload into the continuation, and moves the error payload into the function return path. Normal cleanup for every exited scope runs before that return. No backend receives a `?` operator.

Result fixtures classify an expression as returned, discarded, or propagated, record canonical success and error types, and assert exactly-once evaluation. They accept typed `Result` parameters so no fixture invents `Ok` or `Err` constructor syntax. Active resource-payload moves and cleanup order remain MIR and runtime assertions.

## Normal exits

`return`, `break`, `continue`, and fallthrough clean only the lexical scopes they exit. Cleanup is reverse declaration and reverse field-initialization order. Cleanup obligations are elaborated before backend selection.

## Traps

A trap is a non-returning, non-unwinding termination. It performs no language-level drop or region reset. Trap categories are stable semantic outcomes even when the platform-specific termination mechanism differs.

Stage 0 traps include at least bounds failure, ordinary integer overflow, division by zero, signed minimum divided by negative one, invalid shift amount, failed assertion, and an explicitly invoked unreachable condition.

Signed left shift is deferred in Stage 0 because its overflow and sign behavior would otherwise need an additional source-level rule. Wrapping, saturating, and checked arithmetic are distinct operations rather than build profiles.

## Foreign control flow

Foreign code may return normally or terminate the process. It cannot unwind, throw, or `longjmp` across a Fairy C frame with live obligations. Integrations that use those mechanisms require a C or C++ barrier shim that catches or translates the event before returning a normal status value to Fairy C.

Stage 0 foreign callbacks are raw `unsafe extern C` function pointers plus explicit user data. A safe callback abstraction is deferred until lifetime, reentrancy, threading, and nonlocal-control rules are specified.

## Assertions and optimizer assumptions

`assert` is always checked. A debug-only assertion, if later introduced, cannot become an optimizer fact. An assumption that permits undefined behavior when false belongs to an explicit unsafe operation and must remain visible in MIR provenance.

## Required diagnostics

The compiler must distinguish an unused `Result`, `?` on a non-`Result`, `?` in an incompatible return context, unsupported signed left shift, and forbidden foreign nonlocal control flow.
