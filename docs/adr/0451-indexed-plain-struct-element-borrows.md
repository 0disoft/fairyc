# ADR 0451: Indexed plain struct element borrows

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0441, 0448, 0449, 0450

## Decision

Named plain struct views admit `ref items[i]` and `ref mut items[i]`. The shared
form accepts a shared or mutable descriptor; the exclusive form requires a
mutable descriptor. The referent is the exact stored nominal element, never a
copied indexed value. Grouping preserves place identity. Mutable descriptor
bindings are allowed: the selected descriptor is snapshotted before the index.

```fairy
let ref mut Token[] items = vec::view_mut<Token>(ref mut owner);
let ref mut Token token = ref mut items[3];
edit(token);
let ref Token selected = ref items[3];
assert(selected.value == 5000000000);
```

Direct call arguments, bindings, parameter-origin returns and Option wrapping
use the same element reference. Nested field projections continue through
existing shared/mutable field-reference rules. `*token` and ordinary `items[i]`
reads copy the plain aggregate; changing a copied value does not modify storage.
Whole nominal `*token = value` is subsequently admitted by ADR 0452, while
indexed whole replacement retains ADR 0448.

The element graph remains actual nonempty acyclic structs with scalar leaves;
resource, pointer, borrowed, array and tagged fields remain excluded. Same-layout
nominals are not interchangeable. Integer element references are
subsequently admitted by ADR 0453. Temporary call/slice-expression descriptors,
raw-pointer roots, fixed-array struct views and owning slices remain outside
this admission.

## Lifetime and evaluation

An exclusive element introduces the existing ADR 0415 child generation and
freezes the whole parent and its aliases while live. Different fields/indices
do not prove separation; simultaneous exclusive siblings remain rejected.
Shared elements may coexist and read; their retained loans block referent writes.
Last use releases the child hold. Structural owner mutation and moves remain
blocked while a retained element can reach storage. Returned and tagged elements
preserve origin and loan generations; references into a local owner cannot
escape. Earlier call arguments remain held while later arguments are evaluated.

Receiver selection precedes the index, which executes exactly once. A checked
bounds branch dominates the address operation. Index traps create no element
reference or continuation. Effectful indices cannot structurally mutate the
selected owner or acquire competing access. Rebinding a descriptor during index
evaluation does not change the selected receiver, its length, or its referent.

## Implementation

Source admission reuses `view-element-borrow` from ADR 0449 and the existing
indexed-place Flow analysis. Grouped HIR/source indices retain the named-root
boundary. The ADR 0450 checker-only synthetic initializer capability is removed:
ordinary source references and generated mutable iteration now share one path.
The loop's separate shape gate and whole-owner hold remain intact.

MIR, codec, interpreter and C backend operation contracts are unchanged. Actual
receiver/index definitions, exact nominal/result types, plain graph and bounds
check dominance continue to be verified. No backend exception or warning
suppression is introduced. Source-to-MIR tests assert that whole-element
borrows do not introduce aggregate `view-index` copies, then round-trip the
private codec and emit both Windows and Linux C profiles.

## Verification and compatibility

`internal/compiler/testdata/indexed_struct_borrow.fy` uses grouped/effectful
indices, direct call arguments, shared siblings, mutable/shared returns, Option
payloads, nested field children, values above i32, copy independence and owner
release. Windows dev/release native execution verifies it and end/huge/empty
bounds, index assertion, checked overflow and descriptor rebinding into an empty
independent owner: 14 outcomes. Public CLI check/build/run verifies the fixture
in both profiles and rejects live-owner mutation.

Source rejection tests cover parent/alias reads and writes, competing siblings,
shared/exclusive conflicts, later call arguments, effectful indices, returned/
tagged holds, local escape, exact nominal matching and closed source forms.
Existing indexed-field MIR verifier mutations and array-backed interpreter
reference/copy checks pass. Mutable iteration regressions, cold/warm workspace
cache reuse and related Go vet pass.

The token example reads its first-token origin through `ref segment[0]` instead
of copying a whole Token. The shared child ends before mutable iteration.
Its existing portable verifier passes check, locked dev/release build/run and
14 native normal/error inputs without changing CLI output or limits.

Evidence: `docs/ops/evidence/indexed-struct-borrow-20261004.json`, including raw
argv/cwd/outcomes, compiler/SDK/source digests and EXE hashes. SDK stage0.85,
language /20, runtime /0, wire fields and target maturity remain unchanged.
Linux actual execution is deferred to the owner's batch. No benchmark,
installation, SDK export, DB, CI/runner change, push or release is included.
