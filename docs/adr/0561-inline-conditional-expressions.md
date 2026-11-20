# ADR 0561: Conditional expressions in inline bodies

- Status: Accepted for bounded Go bootstrap and Fairy candidate prototypes
- Date: 2026-11-21
- Extends: ADRs 0493, 0509, 0548 and 0557

Admit value-producing `if` expressions in the existing public, safe, pure
`i32` inline family. Conditions are bool literals or the six comparisons
between admitted `i32` expressions, composed with `!`, `&&` and `||`. Each arm
supplies an `i32` tail expression without statements, or another conditional in
an `else if` chain. Initializers,
assignments and returns share the existing 64-node limit, which also counts
conditions, arm blocks and groups. Statement branches, branch-local bindings
and loops remain outside this admission.

Canonical `fairyc.inline-function/v5` encodes the condition, then arm and else
arm in that order. Both arms participate in identity even if the condition is
constant. Logical operators encode their spelling and children in source order,
including operands skipped at runtime. The `mutable-bindings` root distinguishes
conditional bodies containing mutable locals from those with only immutable bindings.
Parameter names, group syntax and spans remain inert. Bodies without
conditionals retain their exact v1–v4 encodings. Existing package inline class
and cache keys consume the resulting marker.

Ordinary MIR lowering short-circuits logical operands and executes only the
selected arm. The existing acyclic CFG and operation verification, checked arithmetic,
trap source spans and private C linkage apply. The Fairy candidate computes the
identity during source admission. Its detached verifier checks bool conditions,
`i32` results and that each arm block has a tail and no statements, then
recomputes the marker before accepting a restored body.
