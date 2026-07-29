# ADR 0217: Add bounded lexical defer

- Status: Accepted
- Date: 2026-07-29
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Cleanup source: `docs/language/0004-results-traps-and-control-flow.md`
- Depends on: ADR 0024, ADR 0116, ADR 0121, ADR 0216

## Context

Fairy C needs a small, readable way to schedule ordinary cleanup or final reporting without adding
callbacks, closures, exceptions, or C++-style destructors to inert values. Go's `defer` is useful,
but importing its unrestricted call and capture surface would immediately expose resource moves,
borrow lifetimes, foreign calls, and hidden heap-backed environments.

The first form therefore has to remain directly lowerable to existing verified calls and explicit
cleanup edges. Adding the reserved word and statement is a language change, so it cannot be folded
into `fairyc-language/2` as an implementation patch.

## Decision

1. The exact product language pin becomes `fairyc-language/3`.
2. The source form is `defer call_expression;`. It accepts only a resolved internal function call
   returning `void`. Foreign calls, constructors, non-call expressions, and calls returning values
   are rejected.
3. Explicit arguments must be copy scalars from the closed `bool`, `i32`, `u8`, `u32`, and `usize`
   set. Resource, aggregate, pointer, safe-borrow, view, and method-receiver capture remain absent.
4. Argument expressions evaluate immediately at the defer statement, once from left to right. Each
   resulting scalar is copied into a compiler temporary. Later mutation of the source local does not
   change the captured value. Evaluation may call or trap according to ordinary expression rules.
5. Registered calls execute in reverse registration order when their lexical block exits by
   fallthrough, `return`, Result `?` propagation, `break`, or `continue`. A return value is fully
   evaluated before deferred calls begin.
6. Deferred calls run before implicit resource drops and Region resets for the same exiting edge.
   They are ordinary non-unwinding calls. If one traps, remaining deferred calls and cleanup do not
   execute because Stage 0 traps never unwind.
7. A trap at the registration site or elsewhere skips deferred calls. There is no panic recovery,
   cancellation hook, exception path, process-exit hook, or guaranteed execution after abnormal
   termination.
8. CST, AST, and typed HIR retain the defer statement. MIR eliminates it into capture assignments
   plus existing call terminators on explicit cleanup edges; the interpreter and C backend gain no
   new executable operation kind.
9. The immediately preceding `/2` parser rejects `defer`. Valid `/2` source remains valid `/3`
   source, so the migration tool validates source bytes unchanged while exact manifest pins still
   require an explicit update.

## Compatibility

The source grammar and reserved-word set change, requiring `/3`. Runtime ABI, target schemas, C11
transport, MIR version, diagnostic catalog, SDK source bytes, and SDK version remain unchanged.
Package, lock, and cache identities change because they include the exact language revision.

## Evidence

- Lexer and parser tests prove current-only reservation, lossless reconstruction, and `/2` rejection.
- AST and semantic tests retain the call and reject value-returning or borrow-capturing forms.
- MIR tests prove immediate scalar capture, ordinary call elimination, cleanup facts, and verifier
  success.
- Interpreter fixtures prove nested LIFO order, immediate capture, return execution, and trap skip.
- C-emission fixtures prove the eliminated form remains ordinary checked C11 call transport.
- The Windows host smoke compiles and executes immediate-capture defer at low and optimized MSVC
  profiles, with an assertion inside the deferred call guarding the captured value.

## Consequences

Common final reporting and scalar cleanup calls become concise without adding closures or hidden
ownership. The deliberate cost is a narrow first surface: resource-aware deferred close, arbitrary
blocks, foreign calls, method receivers, and borrowed captures need separate lifetime and failure
contracts before admission.
