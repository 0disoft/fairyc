# ADR 0444: Constant checked-failure reachability in C emission

- Status: implemented; Windows dev/release native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0394, 0412, 0442, 0443

## Problem and oracle

A local i64 initialized to its maximum and incremented before a struct return
passes semantic/MIR verification. MSVC release build rejects generated unreachable
return and caller continuation code with C4702/C2220 (stable CLI exit 6), although
Fairy execution has the defined `integer-overflow` trap. The exact ADR 0443 source
reproduction and prior compiler identity are preserved in the review receipt.
After this correction both dev/release builds must succeed and `run` must exit 5
with `trap: integer-overflow`, preserving effects before the failing operation and
omitting effects after it. Warnings remain enabled and treated as errors.

## Bounded proof

The C backend proves only failing `checked-arithmetic` + / - / * guards for the
existing i32/i64/u8/u32/usize domain and minimum-value checked signed negation.
Both admitted C targets are x86-64, so usize has the existing 64-bit transport.
A block-local map tracks scalar literals through verified assign/initialize copy
operations. A checked-arithmetic marker has no write. Every other operation clears
all local facts; an unknown reassignment drops the destination fact. Incoming
block facts, calls, indirect writes, alias contents, casts, arithmetic results,
merges, division and shifts are not evaluated by this proof. Safe arithmetic is
not folded. The proof's numeric comparisons never evaluate overflowing arithmetic.

The same predicate selects the failure successor in C block reachability and
actual guard output. Existing unwind edges remain conservatively reachable.
Existing non-returning-callee inference then excludes impossible aggregate returns
and caller assignments/continuations. Preceding operations remain in their order.
Original verified MIR, its trap marker and its interpreter observation are unchanged;
this is not a MIR optimizer or new source-level constant evaluation capability.

## Helper demand and ownership

Inspect the entire retained MIR for capability/type errors before pruning output.
After inspection, recompute static i32/i64 helper demand from the actually emitted
reachable blocks and non-pruned guards. A proof must not leave unused static
bitcast/overflow/shift helper definitions for strict consumers. Live signed shifts
retain their helper dependencies. No warning suppression or runtime-symbol change
is introduced. Proof facts are recreated per straight-line block and emission;
there is no persisted analysis/cache format or cross-call assumption.

## Evidence and limits

A fixed independent big.Int test oracle compares mathematical range checks at
signed/unsigned boundaries with the bounded proof. Facts tests cover copy chains,
reassignment, indirect/unknown operations, unknown input, wrong types/arity,
unmodeled division/shift guards and retained unwind edges. Compiler tests assert
MIR preservation, interpreter trap and C omission of impossible continuation and
unused helpers. Both Linux and Windows C emission profiles run in backend tests;
Linux actual GCC/Clang execution is deferred by the owner.

Windows native tests cover aggregate RHS/direct return, dynamic safe input,
mutation through an alias, signed and unsigned add/subtract/multiply failures.
The public CLI uses the exact original baseline source digest and a console-effect
fixture, each with check and dev/release build/run. Source/compiler/SDK digests,
raw outcomes and executable hashes are recorded in
`docs/ops/evidence/constant-checked-c-20261004.json`.

This bounded proof does not promise to eliminate every unreachable-code warning
for arbitrary constant computations or paths beyond its facts. SDK stage0.84,
language /20, runtime /0, grammar, MIR/wire schemas, ABI and target maturity remain
unchanged. Compiler revision/executable identity separates native/cache generations.
No whole-repository test, benchmark, installation, CI or runner change is required.
