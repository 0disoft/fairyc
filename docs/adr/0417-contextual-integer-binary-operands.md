# ADR 0417: Contextual integer binary operands

- Status: implemented
- Date: 2026-10-01
- Owner: [Expression typing](../language/0007-name-resolution-and-typing-v0.md)

Admit `1048576 / rows` when `rows` has a supported integer type, without
requiring a temporary typed literal. An integer-literal-only tree is flexible:
integer tokens, groups, unary `+`, `-`, `~`, and trees of integer arithmetic,
bitwise or shift operations. Names, including folded constants, calls, casts
and blocks are not flexible trees. A fixed integer peer selects the flexible
tree's type on either side of an integer operation or comparison. Without a
fixed peer, the expected result type selects integer-valued operations; without
either context, integer literals still default to `i32`. A comparison's `bool`
result context never selects its integer operand type.

The admitted integer types remain `u8`, `i32`, `i64`, `u32` and `usize`.
Literal selection is not value conversion. Variables, constants, casts and
call results retain their types. Mixed-width operands and a result that does
not match its destination remain E1006. A fixed peer takes precedence over
the destination: `let u32 out = 1 + byte` still fails when `byte` is `u8`.
Each literal is range-checked in the selected type (E1013); an expression that
would fit only after cancellation or division does not evade those checks.
Unary plus/minus remain signed-only, including the existing direct-token
signed-minimum exception. Bitwise complement receives the existing integer
context. No new unsigned negation or narrowing conversion is introduced.

Type checking may inspect the fixed peer first, but HIR and MIR keep source
child order. Runtime operands execute left to right, exactly once. Arithmetic
remains checked in the selected type; division by zero, overflow and invalid
shift amounts still trap. Short-circuit boolean operators are unchanged.
The existing compile-time scalar subset uses the same peer selection: its
flexible tree has no names, calls or effects, so the fixed peer can be evaluated
once first. Compile-time range/trap failures remain E4303 and mixed types E1006;
its node budget, call depth and supported types (which exclude `usize`) remain.
When both constant operands fail, no left-first diagnostic-span guarantee is
added for the new peer-selection path.

The [example](../../examples/contextual_integer/README.md) and related semantic,
artifact/interpreter and Windows low/optimized checks cover destination and
peer selection, groups, comparisons, typed constants, evaluation order, and
division/overflow/shift traps. [Native evidence](../ops/evidence/contextual-integer-20261001.json)
records executable hashes. Grammar, language `/20`, MIR v1, SDK stage0.73 and
runtime ABI `/0` remain unchanged; this adds no public SDK API or MIR opcode.
