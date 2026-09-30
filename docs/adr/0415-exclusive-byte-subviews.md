# ADR 0415: Exclusive byte-view subviews

- Status: implemented
- Date: 2026-10-01
- Owner: [Borrow and return origins](../language/0002-borrows-and-return-origins.md)

Slicing an existing `ref mut u8[]` produces an exclusive subview of the same
storage. Shared byte slicing keeps its existing shared result. Admit explicit
and omitted bounds with `0 <= start <= end <= len`, including an empty range
at the end. Evaluate the receiver, explicit start, and explicit end once in
that order. Omitted start is zero; omitted end reads the selected descriptor's
length. Competing mutation is rejected while explicit bounds are evaluated;
a pure length read such as `view[0..view.len]` is allowed.

The resulting descriptor preserves the receiver's owner, origin and original
root loans and adds a subview loan generation. While that generation is live,
reading or modifying its parent, a copied parent descriptor, or original owner
is E2101. The child may be read or modified, passed to a shared reader, copied
with the same generation, or sliced again. A descendant freezes its ancestors
until its last use. Copies retain the generation and do not create independent
mutable access; overlapping arguments remain rejected in either order.

Freeze the whole original view conservatively. Creating two retained siblings
is rejected even if their numeric ranges are disjoint; there is no interval
alias proof, split-at API or disjoint-field loan in this admission. Empty
subviews use the same lifetime rules. Last-use analysis, reachable branch joins
and loop fixed points govern release, rather than lexical scope alone.

Internal return summaries retain whether a reachable return creates an
exclusive subview. Callers retain a corresponding generation rooted in the
selected actual argument. This proof follows aliases and the single safe-borrow
leaf in Option/Result. Branches with the same origin conservatively retain the
subview restriction if any reachable return requires it. Parameter-derived
returns are allowed; local-owner escape and ambiguous-origin returns keep
E2102/E2103. Bounds failure keeps the existing non-returning `bounds` trap.

HIR `view-slice` and the existing MIR `view-slice-check`, checked switch and
`view-slice` continuation now admit matching shared or exclusive byte-view
owner/result types. Mutable source types are derived from local definitions
rather than trusted operand annotations. Shared-to-mutable laundering and
missing/reordered bounds checks fail verification. The interpreter preserves
the backing store, offset and mutable access. C transport emits the existing
`fy_mut_byte_slice` descriptor for exclusive results and `fy_byte_slice` for
shared results, with no unchecked native memory access added.

This extends a previously unsupported executable subset without new source
spelling, MIR fields, public SDK symbols or runtime representations. Language
/20, MIR version 1, SDK stage0.73 and runtime ABI /0 remain unchanged.

Semantic regressions, artifact round-trip execution, MIR mutations, ordered
bound side effects and interpreter bounds traps pass. The executable
[example](../../examples/exclusive_subview/README.md) passes Windows MSVC
low/optimized execution together with reversed, past-end and omitted-bound
failure cases. See the [native evidence](../ops/evidence/exclusive-subview-20261001.json).

MSVC 14.51.36231 `/O2` execution initially lost an inlined write through a
copied nested subview: the parent read retained 7 after `fill` wrote 9.
The same generated source passed at `/Od` and with the writer marked
`noinline`. Windows C transport now emits byte-view index stores through a
volatile-qualified byte access. This preserves the observable write without
changing descriptor layout or disabling general optimization. Shared reads,
fixed-array stores and Linux transport are unchanged. This workaround is not
a public atomicity, concurrency or volatile contract; its optimization cost
has not been measured. The exact upstream optimizer mechanism is not proven.
Physical power-loss experiments remain deferred and unrelated to this change.
