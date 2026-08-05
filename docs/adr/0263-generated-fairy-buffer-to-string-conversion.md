# ADR 0263: Convert owned Buffer into String without copying

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0246, ADR 0247, ADR 0248, ADR 0258

## Context

ADR 0246 added the move-only `std::bytes::Buffer` and ADR 0247 added the move-only
`std::text::String`, but the two owners do not compose: a program holding an owned Buffer with
valid UTF-8 bytes must either copy through `text::copy` or leave `std::text` and hand-build
validation plus raw storage. `text::copy` is the borrowed-text path and intentionally allocates.
There is no public way to adopt an already-allocated Buffer as a String without a second
allocation.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.12` to `0.0.0-stage0.13` because this
   adds a public function to `std::text`. Language remains `fairyc-language/8` and runtime ABI
   remains `fairyc-runtime/0`.
2. `std::text` adds `pub fn Result<String, bytes::Buffer> from_buffer(bytes::Buffer value)`,
   effect-free, implemented with an internal `import std::bytes as bytes;`. On success the exact
   initialized bytes of `value` are valid UTF-8 and the same allocation ownership moves into the
   returned `String` with no allocation or copy. On failure the original Buffer is returned
   unchanged as the `.err` payload with its data, capacity, alignment, and length intact. An empty
   Buffer converts to an empty String. The built-in `Result` err slot carries the Buffer itself;
   no resource-bearing error struct such as `IntoStringError{buffer}` is added.
3. The private raw extern is `fy_std_string_from_buffer_storage_v0(void *input_storage, void
   *output_storage)`. The C storage wrapper rejects null or aliasing pointers and a nonempty
   output storage with `INVALID_ARGUMENT` without touching either caller storage, copies both
   records locally through `memcpy`, reuses the existing probe-only
   `fy_std_string_from_buffer_v0` algorithm, and commits only on success: the input storage is
   zeroed and the output storage is populated. On failure both caller storages remain byte-for-byte
   unchanged. No public C ABI, generic formatter, or new allocator dependency is added.
4. `text::copy` remains the copy-required borrowed-text path. `from_buffer` is the owned
   zero-copy path and neither replaces nor deletes it. The SDK-internal `discard_bytes_buffer`
   helper consumes the moved-from owned Buffer in the success arm so the resource slot is closed
   exactly once; the failure arm discards the empty String output and moves the Buffer into `.err`.

## Compatibility

The Fairy functions are source-additive. Exact Stage 0 SDK selection is intentionally not
lock-compatible: the SDK source digest, package instance, public-interface hash, dependent
identities, generated locks, and whole-package cache keys change because `std::text` now imports
`std::bytes` and exports one new public function. The runtime ABI remains `fairyc-runtime/0`. The
new C symbol is private pre-1 target support compatibility data, not a public C ABI or stable
Fairy native ABI; the probe-only `fy_std_string_from_buffer_v0` remains outside the public symbol
contract.

## Evidence

- Compiler integration tests run the checked-in quickstart through package loading, semantic Flow,
  verified MIR, and entry-reachable C emission and freeze the observed
  `unit.ForeignSymbols`:
  `fy_std_buffer_append_v0`, `fy_std_buffer_drop_v0`, `fy_std_buffer_initialize_v0`,
  `fy_std_string_drop_storage_v0`, and `fy_std_string_from_buffer_storage_v0`. An unused program
  emits no new symbols. Because the C backend emits SDK MIR reachable from the entry, this set is
  observed per program rather than an exact-set contract for the whole module.
- Component-contract tests require `fy_std_string_from_buffer_storage_v0` to select the
  SystemHeap, Buffer, TextView, and OwnedText components and reject a public
  `fy_std_string_from_buffer_v0` symbol.
- The separate-translation-unit raw probe adds layout `_Static_assert`s for the buffer/string
  storage records and separates: nonempty-output rejection with both storages byte-for-byte
  unchanged, invalid UTF-8 with an empty output returning status 2 with both storages unchanged,
  valid transfer with zero new allocations and an exact pointer-identity move plus a single-owner
  drop to zero live allocations, and empty success. Each case has its own return code.
- The Windows smoke builds and executes `examples/std_text_from_buffer_quickstart` at low and
  optimized MSVC profiles: 2 inspected PE32+ images covering the valid `.ok`, the invalid `.err`
  with returned len/capacity/byte preservation, and the empty success arms, all returning 42.
- The generated-Fairy probe renames the generated `main` and runs it under a fail-after-two probe
  schedule with the existing probe-only SystemHeap allocator: 2 forced `.err` executions across 2
  inspected low/optimized images, observing the returned Buffer unchanged, zero live allocations
  after automatic drop, and native result 42. The existing Region 4/2, Buffer 4/4, String append
  2/2, CString 2/2, and String copy 2/2 proofs remain unchanged; the host-smoke receipt for this
  change reports 87 inspected PE32+ images total, including the new 2 quickstart and 2
  generated-probe images.

## Consequences

Programs can adopt an owned validated Buffer as a String without a second allocation and without
losing the Buffer on invalid UTF-8. The raw C probe continues to own byte-level boundaries:
null/alias/nonempty-output negatives, invalid-byte preservation, repeated drop, and exact
alignment/length preservation. Implicit byte-to-text conversion, text indexing or slicing, a
public view ABI, and generic conversions remain absent.
