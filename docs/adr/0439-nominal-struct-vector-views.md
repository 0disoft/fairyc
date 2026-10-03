# ADR 0439: Nominal struct vector views

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0431, 0436, 0438

## Decision

SDK stage0.84 adds one shared generic view export,
`std::vec::view<T>(ref Vec<T>) -> ref T[]`, covering two element families. `T`
may be one of the five already-supported executable integers (`u8`, `i32`, `i64`,
`u32`, `usize`), which resolves to that existing per-type integer view, or an
actual nonempty nominal struct whose every field is a plain language scalar
(`bool`, `u8`, `i32`, `i64`, `u32`, `usize`). The descriptor borrows only the
initialized prefix;
lengths and checked bounds count elements, capacity is excluded, and empty
zero-backed vectors and subranges are valid. For nominal `T`, the element must
name a real struct definition in the current package graph: the view cannot
reinterpret a `Token` view as a same-layout `Other` view. Nested struct, array,
resource, borrow, raw pointer and enum fields are rejected, so an element carries
no ownership or retained loan. The mutable helpers are unchanged and stay exact
per-type, so this adds no `ref mut` struct slice, no fixed-array conversion and no
for-in over a struct view.

Indexing copies one whole scalar-field struct value out of the view. Field reads
through an index (`view[i].field`) are allowed, while element and field stores
are rejected. A copied mutable local is independent of the owner: the struct-view
native fixture holds one copy while the owner is cleared and reads it back
unchanged. Live views and returned/tagged derivatives keep
the ADR 0415 loan rules: owner move, replacement, reallocation and structural
mutation (including no-op `reserve`/`truncate`) are blocked while a view is
live, and index or slice bounds cannot mutate the owner. Short shared readers and
subviews use the existing reborrow rules.

## Implementation and identities

The C descriptor is one shared record, `fy_struct_slice { const void *data;
uintptr_t length; }`. Descriptor construction allocates nothing and copies no
element; an index load copies one whole struct value through the exact cast
`((const Token *)(view).data)[i]`, and a slice adjusts the same pointer through
the struct type. Because the view is shared-only there is no `ref mut` struct
slice and no new runtime C symbol; integer byte/i64/i32/u32/usize descriptors are
unchanged. The five integer arguments reuse the existing private storage path
unchanged; only the shared generic spelling and the nominal descriptor are new.
No host-storage interpretation, public raw builtin or foreign ABI is introduced.

The SDK-owned private transfer path admits the shared nominal descriptor with a
`usize` element count through the existing three-operand storage-borrow form.
MIR transfer verification checks the actual owner, address, count, result type
and unsafe provenance instead of trusting declared annotations, and eight
mutation cases are rejected: same-layout different nominal, mutable result,
empty struct, pointer field, resource field, missing definition, wrong index load
element and a receiver relabeled as a byte view. Ordinary and query-only paths
still reject these transfers.

Source/SDK export identity, public interfaces and function/body hashes preserve
the nominal slice: an exact stable type such as `pkg[coordinate]::Token[]`
rebinds to a fresh local package ID in a new build. In the semantic snapshot a
slice type reference contains exactly one child, either a `nominal` reference
carrying the package coordinate or a `parameter` binder index, and is wrapped by
a separate `borrow` reference; integer slices keep their existing builtin
identity and are not modeled as this slice kind. The snapshot, generic-header and
Atlas V3 roundtrip/rebind tests cover ordinary `ref Token[]` and generic
`ref T[]` declaration headers only. SDK-private executable MIR is not an Atlas
admission claim.

## Verification and limits

The example scans a bounded ASCII source, builds a `Vec<Token>` and reads it
through the shared view with whole/partial/empty subviews and an indexed struct
copy; the separate native fixture covers the copy held across an owner `clear`.
Windows native development and release each run eight struct-view cases (three
success, five checked `bounds` traps) and twelve token-analyzer cases, so sixteen
struct-view and twenty-four token native tests pass, with receipts under
`.cache/struct-vec-view-native` and `.cache/struct-token-native`. The public CLI
harness passes all seventeen invocations (one `check`, locked development and
release `build`/`run`, and the native outcomes), the focused
`TestStruct(VecView|Token)` CLI test (including
`TestStructVecViewGenericIntegerAliases` for the five integer aliases) and all
four serialization packages pass, and `go vet` passes. The checked evidence
receipt is captured at `docs/ops/evidence/struct-views-20261004.json` with
compiler, SDK, source and EXE hashes and raw invocation results. The focused
command is
`bun tools/run-go.ts test ./internal/cli -run '^TestStruct(VecView|Token)' -count=1`.

This is not a released-compiler, target-maturity or performance claim. The
compiler used for the evidence is a dirty working tree based on `1aeb475b`.
Language `/20`, runtime ABI `/0`, grammar, MIR/artifact/interface schemas and C
runtime symbols are unchanged; only the SDK version and dependent identities
move. Actual Linux execution is deferred to the owner's batch, so emitted Linux C
is not runtime proof. No broad tests, benchmarks, dependency installation, CI
runner, PowerShell runner or push is added. No general arbitrary-element slice,
mutable struct or owned slice, or user-defined generic aggregate promotion is admitted.
