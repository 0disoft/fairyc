# ADR 0249: Retain the Arena owner in lexical Region MIR

- Status: Implemented local compiler and verifier gate
- Date: 2026-07-31
- Language source: `docs/language/0003-regions-and-arenas.md`
- MIR source: `internal/mir/`
- Interpreter source: `internal/interpreter/`
- Verifier fixtures: `internal/mirverifyfixture/`
- Depends on: ADR 0020, ADR 0021, ADR 0024

## Context

Lexical Region lowering previously emitted a `region-enter` containing only the compiler-owned
Region symbol. Semantic Flow had already proved the direct `ref mut Arena` source, but MIR discarded
the Arena local before verification. The interpreter could model abstract placement and reset, yet a
native backend had no verified operand from which to derive the real Arena address. Emitting a C
begin call from source or HIR data would bypass the verified-MIR boundary.

## Decision

1. `region-enter` retains exactly one local operand naming the Arena owner. It records the owner
   resource type, exact `ref mut` result type, and `&mut` operator.
2. Lowering still accepts only the existing direct local source. It does not evaluate a
   side-effecting borrow expression, invent a second borrow temporary, or expose an Arena address.
3. `V1403` requires one live local operand, a mutable borrow whose element equals the recorded owner
   type, the `&mut` operator, no destination, and one program-level resource definition.
4. After a malformed enter, verification retains the symbol's conservative active-state fact when
   possible. This prevents one shape defect from cascading into false inactive-place and
   inactive-reset failures; the original enter defect remains reported.
5. The interpreter evaluates the owner operand and requires the exact resource identity before
   creating its abstract Region state. It retains that identity until reset without exposing
   storage, offsets, addresses, capacity, or allocation policy.
6. This changes compiler-private MIR only. `fairyc-language/7`, SDK `0.0.0-stage0.9`, and
   `fairyc-runtime/0` remain unchanged.

## Evidence and limits

A source-to-MIR regression compiles a real `std::alloc::Arena` lexical Region, requires the exact
owner operand and types, and mutates only that operand away to obtain the isolated `V1403` failure.
The complete MIR, verifier-mutation, interpreter-fixture, and Go suites pass with the new shape.

This is a prerequisite, not public Arena promotion. The constrained C backend still rejects Region
operations, the SDK Arena remains a placeholder with no constructor, and no opaque storage bridge,
fallible begin/reset result, placement allocation, native execution, public layout, or ABI is added.
