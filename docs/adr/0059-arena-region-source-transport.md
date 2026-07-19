# Arena and Region Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

The provisional Arena implementation cannot be represented honestly as only a pointer and a count.
Its Arena and Region records contain pointers to a private self-linked node, while each node contains
the exact nested SystemHeap block. Replacing node pointers with `void *` would preserve width but
make the caller and callee C struct definitions incompatible across translation units.

Zeroed allocation also returns storage through `void **`. The backend already transports opaque
`void *`, but a single pointer cannot express the address of a mutable pointer output slot.

## Decision

1. The internal source probe declares exact `@repr(C)` records for the SystemHeap block, the
   self-linked Arena node, Arena, and Region. A pointer to the containing node is not a by-value
   recursive dependency; the nested SystemHeap block remains a value dependency.
2. On `x86_64-linux-gnu`, the source records use `c::ulong` for the C implementation's `uint64_t`
   generation fields. The companion C unit requires with `_Generic` and `_Static_assert` that
   `uint64_t` is exactly `unsigned long` before any crossing can compile.
3. The constrained foreign-pointer slice admits one additional shape: exactly mutable pointer to
   mutable opaque `void` pointer. This allows an exact direct-local
   `&mut *mut void -> *mut *mut void` output-slot address. It does not admit pointer nesting over C
   scalars, arbitrary pointees, the other three mutability combinations, dereference, arithmetic,
   or pointer conversion.
4. Generated C asserts the 24-byte SystemHeap block, 32-byte node, 24-byte Arena, and 32-byte Region
   layouts plus their natural alignments and field offsets.
5. A generated Fairy caller initializes exact Arena and Region records, calls the real
   `fy_std_arena_initialize_v0` and `fy_std_region_begin_v0`, obtains eight zeroed bytes through the
   real `fy_std_region_allocate_zeroed_v0`, and passes the returned pointer to a bounded C observer.
6. The same caller invokes the real `fy_std_region_reset_v0` and `fy_std_arena_drop_v0`. C observers
   require released nodes, generation transitions from one to two to three, cleared active flags,
   and an invalidated Region record.
7. Only those five exercised major-suffixed Arena/Region symbols join the internal standard-library
   allowlist. The `fy_std_` namespace remains closed by default.
8. `fairyc.native-c-report/v23` adds `arenaSourceCallExecutions`, expected to be four after both
   compiler directions and both profiles execute successfully.
9. This probe does not expose the internal records as the public Arena representation, connect
   lexical `region` syntax to these helpers, add safe allocation methods, automatically link the
   standard library in product builds, or establish target conformance.

## Consequences

- A real generated Fairy caller now crosses Arena initialization, Region attachment, zeroed
  allocation, reset, and final drop without substituting mock C storage.
- The node record is transport evidence only. Node allocation grouping and public Arena layout stay
  private and may change with a runtime ABI-major change.
- The existing C-only probe remains responsible for second-begin rejection, malformed state,
  unsupported-alignment rollback, empty allocation, stale use, and active-drop rejection.
- Native execution remains configured but unobserved on this Windows host.
