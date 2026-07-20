# ADR 0119: Flat copy-resource C transport

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- MIR source: `docs/mir/mir-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`

## Context

ADR 0033 admitted only nonempty all-`i32` resources with empty user drop bodies. That was enough to
prove move-only aggregate transport but cannot represent a native file handle on 64-bit Windows.
Using an `i32` would truncate `HANDLE`; using a copyable inert struct would erase ownership. A real
safe File therefore needs a move-only resource carrying a pointer-width value before nonempty drop
execution and the File API can be connected.

The C-emission manifest had also outgrown its JSON Schema feature enum. The runner parsed only the
schema header, so multiple valid manifest feature names were not schema-admitted even though tests
were green.

## Decision

1. The constrained C backend admits nonempty empty-drop resource fields of `bool`, `u8`, `i32`,
   `u32`, `usize`, or a raw pointer. Field types remain exact in MIR; canonical C storage does not
   permit a differently typed operand.
2. Resource size, alignment, and field offsets come from the selected target layout engine. C field
   declarations use the same scalar and raw-pointer type mapping as other private generated values.
3. An all-`i32` resource retains the existing `flat-i32-resource` feature identity. Every other
   admitted resource uses `flat-copy-resource`, keeping old fixture and report meaning stable.
4. A raw pointer field is a non-owning copy value during interpreter cleanup. Dropping the enclosing
   resource does not dereference or free it. The resource container remains move-only, and later
   nonempty drop execution must own native-handle release explicitly.
5. Nested structs, arrays, C scalars, borrows, Regions, nested resources, and nonempty user drop
   bodies remain outside this transport. This ADR does not by itself make a raw-pointer resource a
   safe File.
6. The Linux target evidence label becomes
   `scalar-and-flat-copy-resource-tagged-c11`. Target maturity booleans remain false.
7. The C-emission schema lists every feature currently used by the manifest, and a test rejects any
   future manifest feature absent from that schema.

## Consequences

- A future Windows File resource can carry an exact pointer-width handle without becoming copyable
  or pretending that `i32` is pointer-sized.
- Empty-drop raw-pointer resources are suitable only as transport evidence. Exposing one as a safe
  owning API before nonempty drop execution would leak the handle and remains forbidden.
- Schema header validity is no longer confused with schema/manifest agreement.

## Verification

- C-backend tests execute a `resource` containing `*mut void` and `u32`, assert its x64 private C
  layout, and retain nested-struct rejection;
- the 33-case C-emission manifest adds the same resource and the schema-feature drift guard;
- the checked-in Windows target program constructs and reads a move-only pointer-width resource,
  then the WSL-free host smoke compiles generated C with local x64 MSVC and requires result 42.
