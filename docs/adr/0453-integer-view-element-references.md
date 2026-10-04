# ADR 0453: Integer view element references

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0401, 0415, 0451, 0452

## Decision

Named views of u8, i32, i64, u32 and usize admit `ref values[i]` and
`ref mut values[i]`. The result references the selected stored integer cell,
not a copied indexed value. Grouping the view/index place is allowed. Shared
references can originate in shared or mutable views; mutable references require
an exclusive view. The exact element type must match. Static byte strings admit
shared u8 references through a named view, including offset subviews.

```fairy
fn void increment(ref mut usize counter) { *counter = *counter + 1; }
let ref mut usize[] counts = ref mut counters;
increment(ref mut counts[code]);
```

The direct root must be a named descriptor. Fixed arrays first convert to
named shared/mutable views; direct `ref array[i]`, temporary call/slice descriptor
roots, raw pointers, bool views, owned slices and reference for-in loops stay
outside this admission. Existing nominal element-reference graph validation
and source boundaries remain intact. This supersedes the integer-reference
exclusion in ADR 0451. ADR 0454 subsequently admits mutable integer reference
iteration over named immutable mutable views. ADR 0455 subsequently admits
direct named fixed-array integer element references.

## Lifetime and evaluation

Use the existing ADR 0415 whole-owner loans and child generations. A mutable
child freezes its parent and aliases until last use; separate indices do not
prove disjointness. Shared references can coexist and read, and block referent
writes while retained. Source Flow retains parameter origins across calls,
shared/mutable returns and tagged results. Local-owner escape and owner
structural mutation/moves while a reference is live remain rejected.

Select and snapshot the descriptor before evaluating the index once. A matching
bounds check must dominate the address. Index traps create no reference or
continuation. Effectful indices cannot obtain competing access or mutate the
selected owner. Rebinding the descriptor during index evaluation does not
retarget the address. Named dereference stores now snapshot scalar receivers
as well as aggregate receivers before RHS effects; rebinding that reference
variable does not redirect the store. RHS traps preserve the selected cell.

## Implementation

The semantic checker admits the existing `view-element-borrow` HIR shape for
these integers while field projections retain their nominal-only gate. MIR
`view-element-borrow` uses `IndexedViewElement` and preserves exact reference
kind and type. The verifier checks actual receiver/index local definitions for
byte views as well as other integers. Static byte operands infer only the
existing immutable byte-view type. Operand annotations cannot manufacture
mutability or reinterpret another element width. Bounds dominance and closed
operation metadata still gate codec/backend use; no wire schema changes.

The interpreter adds stored-array scalar references, plus immutable static-byte
references. Subview offsets select the actual backing cell; later reads see its
updates. Mutable static references are rejected. Existing C emission already
produces the exact typed element address; no new C runtime, cast policy, host
flag, SDK export or alias promise is required.

## Verification and limits

The five-type source/execute matrix covers shared/mutable returns, offset
subviews, stored writes, independent value copies, large values, tagged i64
references and static byte reads. Source negatives cover live parent/alias
reads and writes, shared writes, exclusive siblings, later call arguments,
effectful indices, wrong widths, shared-to-exclusive conversion, local escape,
temporary descriptors and direct array roots. Verifier mutations reject missing
bounds, forged shared/mutable or element types, wrong receiver/index and extra
metadata before codec/backend use. Both C profiles emit successfully.

Interpreter boundary tests check every integer backing cell and static subview
offsets. Windows dev/release native checks cover the matrix, end/huge/empty
bounds, index assertion, checked overflow, descriptor rebinding and scalar RHS
rebinding: 16 outcomes. Standalone public CLI check/build/run covers the matrix
and rejects live parent and Vec-owner mutation. The Fairy byte-histogram example
passes fourteen native outcomes (default, empty, repeated, Korean UTF-8, usage,
4096-byte limit and over-limit in both profiles), using independent Python byte
counts as its test oracle.

Evidence: `docs/ops/evidence/integer-element-borrow-20261004.json`. Compiler,
source/SDK closure, fixture, native image and raw result hashes are retained.
Related tests and Go vet pass. SDK stage0.85, language /20, runtime /0 and target
maturity remain unchanged. Linux actual execution is deferred to the owner's
batch; Linux C emission is verified. No benchmark, install, dependency, DB,
CI/runner, push or release change is included.
