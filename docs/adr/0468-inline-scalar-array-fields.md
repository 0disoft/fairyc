# ADR 0468: Inline scalar array fields

- Status: implemented; Windows dev/release native and public CLI validated; Linux C emission only, actual execution deferred to the owner's batch; the direct element-store and `ref` field-view subset is extended by ADR 0469
- Date: 2026-10-05
- Related: ADRs 0416, 0461, 0462, 0463, 0464, 0467, 0469

## Decision

Admit a bounded inline scalar fixed-array field on a non-`@repr(C)` struct:
`struct Packet { u8[4] bytes; usize length; }`. The element set is exactly the existing
copy-scalar leaf set `bool`, `u8`, `i32`, `i64`, `u32` and `usize`, and the length is
`1..=4096`. Fields receive no implicit initialization, so every struct literal supplies each
array field explicitly with a list (`[1, 2, 3, 4]`) or repeated (`[7; 4]`) initializer.

```fairy
struct Packet { u8[4] bytes; usize length; }
struct Envelope { Packet packet; i32[2] codes; }
var Packet packet = Packet { .bytes = [1, 2, 3, 4], .length = 4 };
let u8[4] extracted = packet.bytes;      // whole-field read copies
packet.bytes = [8, 9, 10, 11];           // replacement through a mutable owner
var Envelope envelope = Envelope { .packet = packet, .codes = [0; 2] };
envelope.packet.bytes = [6; 4];          // nested plain-struct owner path
```

Whole-array field reads and copies produce independent values, and replacement stores the whole
right-hand-side array through a named mutable owner or a `ref mut` owner, including nested plain
structs. By-value structs, arrays of structs and `Option` payloads copy array contents
independently, so the copy cost scales with the field contents rather than sharing storage.

Nested arrays (`u8[2][2]`), arrays of nominal structs, resource elements, raw pointers,
borrowed, tagged and array-of-array fields stay closed, and `@repr(C)` structs keep rejecting
array fields. Direct array-field element stores (`packet.bytes[0] = 1`) and `ref` field
views/subviews are admitted by ADR 0469 over this same bounded field; the
whole-field read, copy and replacement rules above are unchanged.

## Evaluation and lowering

`internal/hir/views.go` widens `PlainViewField` to admit `ScalarArrayField`, a fixed array
whose element is a copy scalar and whose shape is `ExecutableFixedArrayShape` (`1..=4096`).
`internal/semantic/resource_types.go` admits the array field only for a non-`@repr(C)` struct
and otherwise reports the existing unsupported subset, and `internal/semantic/typecheck.go`
admits nested (`depth > 1`) whole-field assignment targets through `PlainViewField`.

Lowering reuses the existing `field-store` operation for a whole array field value:
`internal/mir/field_store.go` and `internal/mir/plain_struct_field_store.go` extend the
admitted target shape and `internal/interpreter/field_store.go` validates the stored array length
and element kinds. `internal/mir/verify.go` rejects inline array fields on `@repr(C)` structs
or on non-admitted shapes (`V1001`). C emission resolves the array layout for non-`@repr(C)`
fields in `internal/cbackend/aggregate_types.go` and records struct-declared array shapes in
`internal/cbackend/arrays.go` so the nominal array wrappers are declared before their containing
struct definitions. No new grammar production, MIR operation, wire field, SDK export or runtime helper is
added, because the existing array value transport carries the field value.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or host
flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, public ABI and target maturity
stay unchanged, and this is an additive admission. Exact compiler revision/executable identity
separates cached generations.

## Verification and limits

Semantic tests reject the immutable whole-field source (`E1008`), element width mismatch
(`E1006`), a `@repr(C)` array field, zero and oversize lengths, an array of nominal structs, a
nested array, a resource element and a live shared owner write (`E2101`). The HIR graph test
admits the bounded scalar array leaf and rejects zero-length and nested array shapes, and a MIR
mutation test rejects replacing the array leaf with an invalid shape. Compiler round-trip tests exercise the
MIR codec, interpreter execution and both the Windows MSVC and Linux C emission profiles, and reject
MIR mutations for zero, oversize, nested and `@repr(C)` array fields.

The related suite passes 195 tests with no failures or skips, the Windows native matrix produced 4
outcomes across the matrix and past-end cases in both dev and release profiles, and the standalone
public CLI `check`/`build`/`run` recording passed 2 normal outcomes for dev and release with the
five rejected sources rejecting as expected (immutable, width, `@repr(C)`, owner loan and nominal
element). Compiler build, `go vet` and formatting pass. Both the Windows MSVC and Linux C11
profiles emit; Linux actual execution remains deferred to the owner's batch, so only C emission is
verified for Linux. Evidence is recorded at
`docs/ops/evidence/struct-array-storage-20261005.json` with `.cli.json` and `.native.json`
companions.
