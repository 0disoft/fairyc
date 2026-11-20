# ADR 0561: Conditional expressions in Go inline bodies

- Status: Accepted for bounded Go bootstrap prototype
- Date: 2026-11-21
- Extends: ADRs 0493, 0509, 0548 and 0557

Admit value-producing `if` expressions in the existing public, safe, pure
`i32` inline family. Conditions are bool literals or the six comparisons
between admitted `i32` expressions. Each arm supplies an `i32` tail expression
without statements, or another conditional in an `else if` chain. Initializers,
assignments and returns share the existing 64-node limit, which also counts
conditions, arm blocks and groups. Statement branches, branch-local bindings,
loops and compound boolean tests remain outside this admission.

Canonical `fairyc.inline-function/v5` encodes the condition, then arm and else
arm in that order. Both arms participate in identity even if the condition is
constant. Parameter names, group syntax and spans remain inert. Bodies without
conditionals retain their exact v1–v4 encodings. Existing package inline class
and cache keys consume the resulting marker.

Ordinary MIR lowering evaluates the condition once and executes only its selected
arm. The existing acyclic CFG and operation verification, checked arithmetic,
trap source spans and private C linkage apply. This extends the Go bootstrap;
the authored Fairy candidate still requires its own source and detached verifier
support before it admits this body family.
