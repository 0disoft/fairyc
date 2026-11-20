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
declarations share the function's 16-local limit. Value arms retain the
immutable-binding restriction.

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

Both compilers also admit void `if` statements with optional `else` or
`else if`, lexical `i32` declarations, early returns and plain assignments to
visible mutable `i32` locals. The root must return a value on every path.
Each statement, expression and arm block counts toward the shared 64-node
budget. The 16-local limit and the MIR operation limit still apply.

Statement bodies use `fairyc.inline-function/v6`. Ordered `block` and
`if-statement` nodes preserve evaluation order. `let` and `var` declare lexical
slots, `assign` names its destination slot, and `return` records the returned
expression. Slots depend on active declaration depth, starting at 64, so
assignment changes the slot's value without changing its identity at a branch
join. An all-returning branch may leave an empty MIR join with no incoming edge.
Existing v1–v5 encodings are preserved. The Fairy candidate independently
checks statement coverage, lexical slots and return paths before accepting a
restored v6 marker.

Both compilers also admit `while` and classic `for` with the same pure
conditions and lexical assignment rules. `break` and `continue` use the nearest
enclosing loop. A classic `for` keeps its initializer in scope through the
condition, body and post clause. A loop by itself does not satisfy the root's
return requirement. Iteration count and trap behavior follow ordinary loops.

Loop bodies use the `fairyc.inline-function/v7` encoding and
`fyinlinefn2-sha256-` carrier. It records the loop kind, condition, body, loop
control statements, and every present or omitted `for` clause. The Go MIR verifier
admits reachable `goto` back edges only when the target dominates the source.
Removing those edges must leave an acyclic graph. The same block, operation and
acyclic call-graph limits apply. The previous carrier still requires acyclic
MIR. Before accepting restored bodies, the Fairy candidate recomputes the v7
marker from typed statements and independently checks the loop header records,
lexical slots, and nesting of loop controls. Its package cache keys use the same
loop carrier as the Go bootstrap.
