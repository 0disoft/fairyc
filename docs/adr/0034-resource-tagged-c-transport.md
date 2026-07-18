# ADR 0034: Transport resource-bearing Option and Result through constrained C

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- MIR source: `docs/mir/mir-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`
- Implementation source: `internal/mir/` and `internal/cbackend/`

## Context

ADR 0033 lets a direct nonempty all-`i32` resource with an empty drop body cross the constrained C
boundary, but `Option<Token>` and `Result<Token, i32>` still stopped at the interpreter. MIR already
marks active-payload moves and drops, so keeping these values interpreter-only left a type-shaped gap
instead of a missing ownership rule.

The first typed backend attempt also exposed a hidden propagation defect. On the error path of
`Result<Token, i32>?` inside a function returning `Result<i32, i32>`, lowering returned the input
container instead of extracting its error and rebuilding the declared return type. The interpreter's
runtime tag has no static container type, so final values agreed while the MIR type contract did not.

## Decision

1. The constrained C backend accepts `Option<R>` and `Result<S, E>` when `R`, `S`, and `E` are each
   either exactly `i32` or a direct resource admitted by ADR 0033, and at least one payload is such a
   resource. Nested `Option` or `Result` payloads remain rejected.
2. Concrete resource-bearing tag types are collected from verified MIR, sorted by canonical Fairy
   type, and assigned private numeric C names such as `fy_tagged_0`. Source type and binding names do
   not become generated symbols.
3. A private tagged C struct contains `uint32_t tag` followed by one explicit payload field for
   `Option` or two explicit payload fields for `Result`. A constructor initializes the active field;
   omitted inactive fields are C aggregate-zero-initialized. Stage 0 deliberately does not use a C
   union or expose this larger private shape as Fairy or C ABI.
4. `internal/layout` computes the tagged struct's size, alignment, and declaration-order offsets
   from `x86_64-linux-gnu` target facts. Generated C11 asserts `sizeof`, `_Alignof`, and each
   `offsetof` before any native artifact can be produced.
5. Construction, switch, and payload extraction use the concrete MIR container type. A payload
   extraction must carry `MovePayload` exactly when the selected payload needs drop. Whole-container
   moves and normal drops remain explicit verified MIR events and emit no cleanup call while every
   reachable user drop body is empty.
6. Scalar `Option<i32>` and `Result<i32, i32>` retain their existing private compact structs. The
   new registry applies only to resource-bearing tag types and does not silently generalize nested
   or arbitrary payloads.
7. `?` lowering always extracts the error payload and constructs a new `Result::Err` with the
   enclosing function's declared Result type before normal-exit cleanup. Error types must match.
   Returning the operand container directly is forbidden even when the interpreter would observe
   the same variant and payload.
8. The C-emission v0 manifest grows from eleven to seventeen cases, adding resource Option Some and
   None, resource Result Ok and Err, and resource Result propagation success and error. The Linux
   matrix therefore plans 68 process-status comparisons across Clang/GCC and low/optimized profiles.
   Its four product-build smokes now combine resource-bearing Option, Result, propagation, and field
   access in one source.
9. `fairyc.target-spec/v0` reports
   `scalar-and-flat-i32-resource-tagged-c11`; `emittable`, `conformant`, and `supported` remain false.
10. Process status still cannot observe cleanup order or drop traces. This slice is emission and
    planned differential evidence, not native cleanup, public layout, ABI, sanitizer, or user
    drop-body conformance.

## Consequences

- The interpreter and C backend now share the implemented resource-bearing built-in tag subset.
- Typed C emission catches MIR container mismatches that an untyped runtime value oracle can hide.
- Explicit inactive fields cost private translation-unit space but avoid an unimplemented union
  layout and inactive-member access contract.
- Nested tags, broader resource fields, partial initialization, user drop effects, and native hosted
  evidence remain separate gates.

## Rejected alternatives

- Bitcast one Result container into another on propagation: hides malformed MIR and depends on a
  private layout coincidence.
- Reuse one untyped tagged struct for every payload: erases concrete storage types and lets invalid
  field access compile.
- Use a C union immediately: requires a target union-layout and active-member access contract that
  the current layout engine and fixtures do not own.
- Treat interpreter agreement as sufficient: the interpreter shares MIR lowering and does not carry
  the static container identity that exposed this defect.
