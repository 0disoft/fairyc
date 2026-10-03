# ADR 0435: Exclusive u8 and i64 view iteration

- Status: implemented (Windows native execution; Linux execution deferred)
- Date: 2026-10-03
- Related: ADR 0403, 0415, 0432, 0433, 0434

## Decision

Admit the existing `for (let u8 value in view)` and
`for (let i64 value in view)` forms over named immutable descriptor bindings
and parameters of exact `ref mut u8[]` and `ref mut i64[]` types. An immutable
binding fixes the descriptor, not the contents. Each body entry copies one
checked scalar load. Length is observed once; loads observe prior body stores
in source order. Empty views skip the body. Items stay immutable and body-scoped.
Reference items, direct collection values, iterable calls/slice expressions,
mutable descriptor bindings and other element types remain unadmitted.

The loop holds its existing exclusive owner loan through every body exit,
including break, return and Result propagation. This hold does not create a
shared reader or an additional exclusive sibling: stores through the iterated
view, shorter exclusive child views, and shortened shared readers are governed
by existing access rules. Retained readers block mutation; retained children
freeze ancestors and siblings. Nested loops retain their own owners. Owner
reads/borrows, structural changes, replacement and move stay blocked until exit
and any later retained use. After releasing a Vec view, growth and a fresh view
are allowed. No array mutation bypass or iterator allocation is introduced.

Continue advances the existing hidden usize counter exactly once; break skips
its post statement. Defer, resource cleanup and propagation use the existing
scalar-loop edges. Checked numeric and range traps are unchanged.

## Implementation and safety correction

Semantic admission uses `hir.IndexedViewElement` to match the exact item/view
pair. The previous i64 loop-hold helper now covers both u8 and i64, shared and
exclusive. This also closes the shared-u8 early-exit owner-write gap.

A focused rejection test exposed a previous loop-hold gap: passing a borrowed
Vec by value to a consuming function immediately before return was accepted,
because owner-move checking considered later lexical uses but ignored held
loans. Owner-move checking now examines both retained and pending held loans,
with E2105 owner/loan/last-use locations and duplicate-loan suppression. The
correction applies to both shared and exclusive view loops. Resource Flow may
reject some loop-state changes first with E1105; tests preserve that diagnostic.

AST preparation, HIR/MIR scalar loops, checked index operations, codec schemas,
interpreter and C backend need no new instruction. The existing Windows
exclusive-i64 volatile-access workaround from ADR 0433 remains in use; its
optimization cost is unmeasured and it provides no atomic/concurrent guarantee.

Following ADR 0430-0434 additive admissions, language `/20`, SDK stage0.81,
runtime `/0`, grammar, SDK source/public exports and MIR/artifact schemas stay
unchanged. Compiler executable identity separates disposable cached generations;
there is no released compiler version to bump or new foreign ABI.

## Verification and limits

The [Fairy example](../../examples/mut_view_for/src/main.fy) checks actual
contents, copied items, future-element writes, signed extrema, values above i32,
empty/singleton/subrange views, nested shadowing, continue/break/return,
Result propagation and post-loop owner mutation. Verified MIR codec execution
and both Linux/Windows C emission cover the same program. Interpreter and
Windows development/release native cases check u8 and i64 overflow/underflow.
SDK Vec checks cover an empty allocation, element updates, post-loop growth,
new/child view iteration and move after release. Cleanup observers check exact
Defer/drop order on continue/break/return and propagation for both types.

Semantic rejections cover owner writes/borrows including early exits,
parent/sibling conflicts, retained shared readers, mutable descriptors, item
mutation and scope escape. SDK checks reject read, push/reserve/pop/clear,
replacement and consuming moves including early return. Existing shared loops
and other borrow-flow regressions are retained.

[The evidence receipt](../ops/evidence/mut-view-for-20261003.json) records
source/compiler/SDK identities, native image hashes, raw outcomes, stable CLI
check/build/run results and validation commands. Linux C emission is not Linux
execution. Linux runtime checks remain deferred to the user's batch. No
full-repository test suite or performance benchmark is claimed.
