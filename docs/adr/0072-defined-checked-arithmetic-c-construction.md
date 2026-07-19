# ADR 0072: Defined checked-arithmetic C construction

- Status: Accepted
- Date: 2026-07-19

## Context

Verified MIR checks signed `i32` negation, addition, subtraction, and multiplication before their
result operations. The constrained C emitter previously repeated the corresponding signed C
operation in the success block. Reached executions were in range, but source constants remained
visible to the C frontend. MSVC therefore diagnosed `2147483647 + 1` as signed constant overflow
while compiling with warnings as errors, before generated control flow could trap.

Using volatile operands for every ordinary arithmetic operation would suppress constant folding but
would add unnecessary memory traffic. The result bits of an in-range checked operation are the same
as the corresponding modulo-2^32 unsigned operation.

## Decision

1. Verified MIR's widened check and trap edge remain the semantic authority.
2. After that edge, checked unary negation constructs zero-minus-value in `uint32_t`. Checked
   addition, subtraction, and multiplication apply the matching operation to `uint32_t` operands.
3. The resulting bits return to `int32_t` through the existing `memcpy`-based bit-preserving helper.
   Generated C never evaluates the signed operation itself.
4. This is private constrained-C transport. It does not change Fairy C overflow semantics, wrapping
   operations, MIR, the runtime ABI, or the public C ABI.
5. Backend source tests reject the old visible signed constant expression. The Windows host smoke
   compiles and executes integer-overflow traps through low and optimized MSVC profiles.

## Consequences

- Strict C frontends cannot reject an unreachable constant signed-overflow expression because none
  is emitted.
- The result construction remains defined even if a future backend-control defect reaches it after
  a missing check; that does not excuse or mask verifier and trap-edge failures, which retain their
  own tests.
- Division and remainder keep the separate checked-divisor materialization from ADR 0071 because
  unsigned division would not preserve signed quotient and remainder semantics.
