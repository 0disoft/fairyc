# ADR 0448: Mutable plain struct vector views

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0414, 0415, 0433, 0439, 0441, 0447

## Decision

SDK stage0.85 adds `std::vec::view_mut<T>(ref mut Vec<T>) -> ref mut T[]`.
It borrows only the initialized prefix without allocation or element copies.
`T` is one of the five existing integer elements (u8/i32/i64/u32/usize), or an
actual nonempty nominal struct in ADR 0441's acyclic plain scalar-leaf graph.
Existing per-integer helpers remain available; shared `view<T>` is unchanged.

```fairy
let ref mut Token[] items = vec::view_mut<Token>(ref mut owner);
let Token old = items[0];
items[0] = Token { .kind = 2, .position = Span { .start = 3, .end = 8 },
                   .value = 5000000000, .active = true };
```

An index reads a whole independent value. `items[i] = exact_nominal_value`
replaces the whole stored element, copying every nested field. Its old value
has no destruction obligation under the admitted plain graph. Changing the
source replacement or a loaded copy cannot change the stored element.
Length/empty predicates, checked subviews, returned and tagged mutable views,
short shared-call reborrows and reference descriptor rebinding use existing
view rules. Zero-backed empty vectors and zero-length ranges are valid.

## Loans and closed operations

A live exclusive view or descendant holds the whole owner. Direct owner reads,
move, replacement, reallocation and structural mutation (even no-op reserve or
truncate) are rejected. Bounds and replacement-value evaluation keep the target
borrowed. ADR 0415 generations freeze parent/aliases while a mutable child is
retained; disjoint mutable siblings remain rejected. Last use releases the hold.
Mutable storage descriptors retain a child generation even when created from
a reference parameter with no local root loan; function returns and Option
payloads preserve that generation. Shared-call returned children also block
competing writes. Local and multiple-
origin escapes remain subject to the existing return-origin rules.

Element field stores, element/field address borrows through an index, mutable
struct-view for-in, fixed-array struct conversion and owned slices remain closed
in this first unit. Cycles, missing/empty definitions and array/resource/borrow/
raw-pointer/enum/Option/Result fields remain rejected at every depth. Same-layout
nominals are not interchangeable. No implicit shared-to-exclusive conversion or
new raw public constructor/foreign ABI is admitted.

## Lowering and execution

The existing private SDK `storage-borrow-mut` carries the owner, mutable void
address and usize count. Semantic analysis and MIR separately validate the actual
plain graph. Actual local definitions, owner/address mutability, exact store
value type, bounds-check dominance and unsafe provenance remain enforced.
Existing operations and codecs represent the descriptor, subview and whole
store; no new MIR opcode/schema is needed.

The C descriptor is `fy_mut_struct_slice { void *data; uintptr_t length; }`.
Loads/stores use the exact concrete struct pointer cast; subviews advance in
elements and avoid null-pointer arithmetic for empty zero-start ranges. Windows
uses the existing ADR 0433 volatile store policy and volatile nominal loads when
mutable nominal descriptors are present; Linux C emission uses ordinary typed
loads/stores. No runtime C function, warning suppression or optimization flag is
added. Interpreter array-backed nominal views copy both load and store values;
this does not add host-storage/Vec allocator interpretation.

Indexed assignment evaluates receiver and index, checks bounds, then evaluates
its RHS. A discovered existing lowerer failure for a terminating RHS is corrected:
a trap produces no stored value or continuation. Native regressions distinguish
out-of-range `bounds` before RHS from an in-range RHS `assertion-failed`.

## Verification and compatibility

The ordinary Fairy fixture is
`internal/compiler/testdata/mutable_struct_vec_view.fy`. Windows dev/release
exercise whole replacement, nested copied-value independence, view-to-view copy,
shared reads after writes, child last use, empty/growth, returned/tagged origins,
five generic integer aliases, end/empty/huge bounds, reversed/past-end ranges and
RHS trap order. Source rejection tests cover active owner/alias conflicts and the
closed operations. Private MIR mutations reject forged mutability/nominals/local
annotations, missing checks/definitions and unsafe fields, including codec
rejection. Both C emission profiles, mutable snapshot nominal identity,
interpreter copy/bounds tests and cold/warm all-hit cache pass.

Public CLI check and locked dev/release build/run confirm success and bounds,
and reject live owner mutation. Related stage tests, existing shared struct native
cases and Go vet pass. Raw argv/cwd/outcomes, source/SDK/compiler digests and image
hashes are in `docs/ops/evidence/mutable-struct-views-20261004.json`.

Language /20, runtime /0, grammar, public foreign ABI, MIR/interface/artifact
schemas and target maturity remain unchanged; SDK export/source/version identity
advances to stage0.85. Exact compiler identity separates cache generations.
Linux actual execution remains deferred to the owner's batch. No broad repository
suite, benchmark, installation, CI/runner change, push or release is included.
