# ADR 0561: Conditional expressions in inline bodies

- Status: Accepted for bounded Go bootstrap and Fairy candidate prototypes
- Date: 2026-11-21
- Extends: ADRs 0493, 0509, 0548 and 0557

Admit value-producing `if` expressions in the existing public, safe, pure
`i32` inline family. Conditions are bool literals or the six comparisons
between admitted `i32` expressions, composed with `!`, `&&` and `||`. Each arm
supplies an `i32` tail expression without statements, or another conditional in
an `else if` chain. Both compilers also admit immutable `i32` bindings in
value arms. Every initializer, assignment and return shares the existing
64-node limit, which also counts conditions, arm blocks and groups. Arm
declarations share the function's 16-local limit. Statement branches,
assignment within arms and loops remain outside this admission.

Canonical `fairyc.inline-function/v5` encodes the condition, then arm and else
arm in that order. Both arms participate in identity even if the condition is
constant. Logical operators encode their spelling and children in source order,
including operands skipped at runtime. The `mutable-bindings` root distinguishes
conditional bodies containing mutable locals from those with only immutable bindings.
Parameter names, group syntax and spans remain inert. Bodies without
conditionals retain their exact v1–v4 encodings. Existing package inline class
and cache keys consume the resulting marker.

An arm with bindings encodes an `arm-bindings` node with its initializers in
source order followed by the tail. Local path positions start at 64 plus the
number of active arm bindings. This keeps them distinct from positions 0–63
for root steps and allows sibling blocks to reuse positions. Each binding
becomes visible after its initializer and goes out of scope at the end of its
arm. Arms without bindings retain their original encoding.

Ordinary MIR lowering short-circuits logical operands and executes only the
selected arm. The existing acyclic CFG and operation verification, checked arithmetic,
trap source spans and private C linkage apply. The Fairy candidate computes the
identity during source admission. Its detached verifier checks bool conditions,
`i32` results and each arm's binding records and tail. It rebuilds lexical
visibility before recomputing the marker and accepting a restored body.
