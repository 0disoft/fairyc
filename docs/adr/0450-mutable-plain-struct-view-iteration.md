# ADR 0450: Mutable plain struct-view iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0435, 0440, 0441, 0448, 0449

## Decision

Admit `for (let ref mut Token item in items)` over a named immutable
`ref mut Token[]` descriptor or parameter. The item referent must exactly match
the actual nonempty acyclic plain struct graph with scalar leaves. Qualified
nominal names resolve normally. The descriptor cannot be reassigned by the body.

```fairy
let ref mut Token[] items = vec::view_mut<Token>(ref mut owner);
for (let ref mut Token item in items) {
    item.value += 4000000000;
    let ref mut Span position = ref mut item.position;
    rebase(position, origin);
}
```

Each item points to the stored element. Scalar/nested field stores, checked
compound assignment, nested nominal field replacement and shared/mutable field
children use existing reference rules. Ordinary indexed reads and explicit
value copies still copy the whole aggregate. Whole nominal `*item = value`
assignment is subsequently admitted by ADR 0452; whole indexed replacement
retains ADR 0448.

The item introduces an ADR 0415 exclusive child generation. While it or a
retained descendant is live, parent descriptor, aliases and competing siblings
cannot be accessed. Fields and indices do not establish disjointness. Last use
releases the child hold; no unconditional lexical freeze is introduced.
Independent owners may be iterated in nested loops.

The whole owner stays held through every body exit, including break, continue,
return and Result propagation. Structural owner mutation is rejected even after
the current item's last use. The loop's hold ends at the written loop exit.
Returned/Option-wrapped item references preserve origin and child generations;
references into a local owner cannot escape. A previous iteration's item cannot
remain live when the next iteration acquires a competing item. Overwriting an
unused previous reference follows existing last-use rules.

Shared struct views retain copied-value iteration. Copied items from mutable
struct views, shared reference items from struct slices, mutable descriptor
bindings, direct Vec iteration with mutable items, temporary call/slice
iterables, direct source `ref items[i]`/`ref mut items[i]`, fixed-array struct
conversion and owning slices remain closed. Integer copied-item loops and
read-only authenticated Vec reference iteration are unchanged.

## Lowering and ownership

AST preserves the written loop and prepares the existing length-snapshot,
checked-index and classic-loop control path. The synthetic initializer is a
mutable indexed element borrow. Only that exact initializer receives semantic
admission after the source shape, descriptor and nominal identity are checked;
written indexed element borrows in the body remain rejected. The capability is
restored around nested loop checking and does not survive outside the loop.

HIR/MIR reuse ADR 0449 `view-element-borrow`, including dominating successful
bounds checks and exact receiver/index/result types. Continue executes the
hidden post increment; break bypasses it. No new AST syntax, MIR operation, wire
field, codec envelope, SDK export, runtime symbol or optimization flag is added.
Existing Flow child-generation, loop fixed-point, origin summary and owner-hold
analysis apply to the synthetic reference without a second borrowing model.
ADR 0451 subsequently admits direct indexed element borrows and removes the
checker-only initializer capability; source and synthetic references share a path.

## Verification and compatibility

Stage tests cover written AST identity, exact source admission, parent/alias/
sibling conflicts, nested children, last-use release, item scope, checked MIR
lowering, private codec round-trip and rejection of missing bounds checks.
Both Windows and Linux C emission profiles consume the verified loop. Private
array-backed interpreter execution confirms stored mutation, empty iteration,
continue and break; host Vec storage interpretation remains unavailable.

`internal/compiler/testdata/mutable_struct_for.fy` checks ordered stored edits,
nested field references, values above i32, continue/break, early Option return,
Result propagation, empty subviews and owner release. Windows dev/release
executables also cover independent nested owners, checked overflow and an
asserting RHS. Public CLI check/build/run verifies success and overflow in both
profiles plus a rejected live-owner mutation. Cold/warm workspace cache tests,
related stage regressions and Go vet pass.

Evidence: `docs/ops/evidence/mutable-struct-for-20261004.json`, with compiler and
SDK source digests, raw execution records, toolchain identity and EXE hashes.
SDK stage0.85, language /20, runtime /0 and target maturity stay unchanged.
Linux actual execution remains deferred to the owner's batch. No benchmark,
installation, CI/runner change, push or release is included.
