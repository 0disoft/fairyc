# Contextual integer operands

`1048576 / rows` takes the integer type of `rows`. Literal-only arithmetic
also receives an expected integer result type, so `let u32 value = 2147483647 + 1`
uses checked u32 arithmetic. Typed values are never implicitly converted.
See [ADR 0417](../../docs/adr/0417-contextual-integer-binary-operands.md).

This example checks groups, comparisons, constants, returns, function arguments,
repeated-array elements, single call evaluation and left-to-right runtime order.
Related semantic and artifact checks use `fairyc_literal_check`; Windows normal,
division-by-zero, overflow and invalid-shift execution on low/optimized profiles
uses `fairyc_literal_native`. [Evidence](../../docs/ops/evidence/contextual-integer-20261001.json)
records source and executable hashes. Original mini-project sources are unchanged.
