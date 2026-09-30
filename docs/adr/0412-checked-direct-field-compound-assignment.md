# ADR 0412: Checked direct field compound assignment

- Status: Accepted
- Date: 2026-09-30
- Owner: `docs/language/0002-borrows-and-return-origins.md`

The calculator workload had to spell `state.position = state.position + 1`
because compound assignment was limited to locals/indexed views. Complete the
existing assignment semantics for direct integer fields of mutable local
structs/resources and existing mutable aggregate references. Support the same
checked arithmetic, division/remainder, shift and bitwise operators as locals.

Select the named receiver once, capture the old field value, evaluate the RHS
once, apply the existing checked operator, then store only on success. Reuse
ordinary field-read, checked arithmetic and field-store MIR; no new MIR opcode,
backend ABI, grammar, language revision or SDK surface is introduced. This is a
bounded executable-subset extension like ADR 0394. Ordinary assignment and
whole-root loan/privacy/moved-owner checks remain intact.

Nested field targets, shared receivers, boolean arithmetic and aggregate compound
assignment remain unsupported. No mutable field address or disjoint loan is added.
Tests cover interpreter/native low and optimized behavior, value-copy isolation,
mutable/resource receivers, RHS ordering and checked arithmetic traps. MIR artifact
round-trip execution ensures the existing transport retains these operations.
