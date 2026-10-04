# ADR 0461: Bool views and scalar iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0435, 0451, 0455, 0457, 0458, 0460

## Decision

Extend the existing scalar view path to bool. Fixed bool arrays already support
value initialization, repeated literals, indexing and stores. They now also
support `ref bool[]` and `ref mut bool[]` views and direct checked element
references (`ref flags[index]`, `ref mut flags[index]`). The element remains
exactly bool; i32 values/views cannot stand in for bool despite the private
backend's matching scalar storage width.

Views provide `len`, `is_empty`, checked indexing/stores and half-open slicing.
An empty subview at the end is valid; indexing it traps. Borrowed parameters may
return a view or element reference subject to the existing origin rules. Local
and by-value array storage cannot escape. An exclusive child freezes its parent
while live; disjoint mutable siblings are still unsupported.

```fairy
var bool[8] occupied = [false; 8];
let ref mut bool[] slots = ref mut occupied;
let ref mut bool[] middle = slots[2..5];
for (let ref mut bool flag in middle) { *flag = true; }
```

Admit copied (`let bool`), shared (`let ref bool`) and exclusive (`let ref mut
bool`) items over named immutable view bindings and named fixed arrays. The
exclusive item requires a mutable view or mutable array. Copied array loops
retain a shared whole-array loan; copied mutable-view loops permit descriptor
writes, and later iterations observe current storage. Shared-reference loops
retain shared protection through loop exit, even after the current item's last
use. Exclusive items reference the stored cell and obey ordinary child-loan
last-use rules. Hidden index traversal, `continue`, `break`, return and Result
propagation reuse the existing loop cleanup path.

Bounds checks use usize indices. The existing array guard remains 1..=4096.
Owned unsized slices, computed loop roots, mutable descriptor bindings and
fixed struct arrays are not admitted. This decision adds no numeric-to-bool
cast or bool arithmetic and does not extend tagged mutable-scalar payloads.

## Implementation

HIR recognizes `[bool]` via `TypeBoolSlice` and the canonical scalar-slice
classifier `ScalarViewElement`, replacing the narrower integer-only classifier.
Semantic type admission, public interface type normalization, snapshots, function
identities and storage specialization all use that same exact scalar vocabulary.
Generic SDK `Vec<bool>` views reuse the existing authenticated `view<bool>` and
`view_mut<bool>` paths; no SDK source/export or version change is needed.

Existing array-view, view-slice, view-index, element-borrow and store operations
carry the new exact types through HIR/MIR verification and the codec. The C
emitter declares separate `fy_bool_slice` and `fy_mut_bool_slice` descriptors
using `const int32_t *` / `int32_t *`, matching existing Fairy bool scalar/array
storage. Using C `_Bool *` would mismatch that backing storage. These are
compiler-private pointer/length descriptors, with no bit packing or backing
allocation. The existing interpreter scalar-cell logic handles bool values.

## Verification and compatibility

Semantic tests exercise origin escape, exact bool/i32 mismatch, shared stores,
owner mutation, exclusive parent access, parent/child conflicts, siblings,
unsupported descriptor/loop roots and array limits. Compiler tests check repeated
analysis, identical MIR encoding, codec round-trip, interpreter execution and
both Windows MSVC and Linux C profiles. Malformed array, load, store, reference,
slice and missing-bounds metadata are rejected by the verifier, codec and both
emitters. Historical bool-rejection fixtures now reflect the admitted surface.

The Fairy matrix covers repeated initialization, scalar element references,
copied/shared/exclusive iteration, offset/empty views, copy independence,
future-element mutation, nesting/shadowing, early return and Result propagation.
Windows dev/release execution also covers generic SDK Vec<bool> storage/views
and six bounds traps. A source session verifies cold lowering and warm all-hit
reuse of the bool SDK fixture. A separate compiler executable checks four public
diagnostic classes and executes the matrix, two bounds traps and the two-module
occupancy example with exact exit/stdout/stderr checks. Source, SDK, compiler,
EXE hashes and raw execution receipts are under
`docs/ops/evidence/bool-view-20261005.json` and companions.

The expanded scanner lane exposes two pre-existing stale specialization-count
assertions (`ScannerCanonicalCallParity` and `UnusedNativeFootprint`): expected
21, actual 31. A Go overlay restoring every changed production Go file from
base commit 4f048530 reproduces both failures before this bool admission.
Those assertions are not changed here; the broad lane is not reported as an
all-pass result. The bool, array/reference and cache checks pass independently.

Language /20, SDK stage0.86, runtime /0, MIR/wire schemas, ABI and target maturity
are unchanged. Compiler executable identity separates cached generations.
Linux actual execution is deferred to the owner's consolidated batch. No CI,
benchmark, installation, push or release is included.
