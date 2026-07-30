# ADR 0224: Defer unproven surface shorthands

- Status: Accepted deferral
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Surface source: `docs/adr/0136-c-shaped-source-surface.md`
- Evidence scope: compiler-bundled SDK and checked-in examples at language revision `/6`
- Depends on: ADR 0005, ADR 0136, ADR 0213, ADR 0218

## Context

Range iteration, checked narrowing conversions, and expression-bodied functions are individually
small features in other languages. In Fairy C each still adds a second semantic or syntactic path:
a range loop needs iterator ownership and mutation rules, narrowing needs an exact failure type and
target-independent conversion matrix, and an expression body gives every function two body forms.
Their small appearance is not evidence that their complete compiler and teaching cost is small.

The audited 29-file non-fixture Fairy C corpus contains exactly one length-indexed byte-view loop,
in `examples/windows_quickstart/main.fy`. It contains no checked or manually guarded narrowing
conversion; the only safe numeric casts are the already lossless `u8 as i32` cases. About twenty
functions are a block containing one explicit return, mostly tiny examples, accessors, and
ownership probes. Replacing those blocks would save lines but would not improve type information in
the function header, ownership visibility, Atlas identity, or generated code.

## Decision

1. Do not add range-`for`, checked narrowing syntax, or expression-bodied function syntax at
   language revision `/6`. No spelling, diagnostic, HIR or MIR form, ABI rule, or compatibility
   behavior is reserved.
2. Keep classic scalar `for` and explicit `while` as the complete loop surface. Reconsider a range
   form only after at least three independent non-fixture loops over one supported view or
   collection shape demonstrate repeated indexing boilerplate and the proposal specifies mutation,
   borrow origin, early exits, evaluation count, and resource-element behavior.
3. Keep the four implemented safe widening casts only. Reconsider narrowing only after real source
   needs at least two destination widths and the proposal chooses explicit checked failure behavior,
   signedness rules, diagnostics, compile-time evaluation, MIR traps or `Result`, and C transport.
   A C-style silent truncation is not an admissible shortcut.
4. Keep one mandatory braced function body. Reconsider an expression form only if a representative
   project demonstrates material readability or maintenance benefit beyond deleting `return` and
   braces. Any later form must preserve explicit return type, effects, attributes, receiver, source
   spans, declaration documentation, and deterministic formatter output.
5. Generic iterators remain behind the user-generics gate in ADR 0223. Provider-qualified imports
   and formatter behavior remain separate package and tooling decisions; this ADR does not close or
   admit them.

## Consequences

Fairy C retains one obvious spelling for control-flow bodies and avoids importing hidden iterator
or conversion policy. The cost is one manual byte-view loop and several three-line trivial
functions in the current corpus. Those costs are visible, grep-friendly, and small enough to keep
as future evidence rather than prepaying permanent language complexity.

Language `/6`, public-interface v3, MIR v0, C11 transport, caches, runtime ABI, and target schemas
remain unchanged.
