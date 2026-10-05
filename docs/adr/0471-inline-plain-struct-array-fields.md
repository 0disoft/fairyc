# ADR 0471: Inline plain-struct array fields

- Status: implemented; Windows dev/release native and MIR codec/interpreter verified; Linux C emission only, actual execution deferred to the owner's batch
- Date: 2026-10-05
- Related: ADRs 0416, 0462, 0463, 0464, 0468, 0469, 0470

## Decision

Admit a bounded inline fixed-array field on a non-`@repr(C)` struct whose element is an admitted
plain struct (ADR 0462): `struct APage { ZToken[4] tokens; usize used; }`. The element must be an
actual, nonempty, acyclic struct whose leaves are copy scalars in the existing set `bool`, `u8`,
`i32`, `i64`, `u32` and `usize`. Nested ordinary structs are allowed, but no array field may
appear anywhere in the element's field graph, so the element graph stays array-free. The length
bound is `1..=4096`, and nominal identity stays exact even when two element structs have equal
layouts.

Fields receive no implicit initialization, so every struct literal supplies each array field
explicitly with a list (`[t0, t1, t2, t3]`) or repeated (`[t; 4]`) initializer of that element
struct. Whole-field reads and copies produce independent values, and replacement stores the whole
right-hand-side array through a named mutable owner or a `ref mut` owner, including a nested plain
struct parent path. By-value structs, calls and returns, arrays of structs and `Option` payloads
carry independent copies, so the copy cost scales with the field contents rather than sharing
storage.

Declaration order does not matter: the element struct may be declared after its container.

```fairy
struct APage { ZToken[4] tokens; usize used; }
struct ASection { APage page; }

struct ZToken { u32 kind; u8 flags; }   // declared after its container

var APage page = APage { .tokens = [ZToken { .kind = 1, .flags = 0 }; 4], .used = 4 };
let ZToken[4] copied = page.tokens;                        // whole-field read copies
let ZToken first = page.tokens[0];                         // value-read copy, indexed immediately
page.tokens = [ZToken { .kind = 2, .flags = 1 }; 4];       // replacement through a mutable owner

var ASection section = ASection {
    .page = APage { .tokens = [ZToken { .kind = 0, .flags = 0 }; 4], .used = 0 },
};
section.page.tokens = [ZToken { .kind = 9, .flags = 2 }; 4];   // nested plain parent path

var ZToken[4] local = section.page.tokens;                 // copied-out local nominal array
local[0].kind = 5;                                         // existing nominal-array operation
```

A whole-field read is an ordinary value copy, so it can be indexed immediately:
`let ZToken first = page.tokens[0];` reads the field value and then indexes the copied value, with
no field place or reference and no element-level borrow of the field storage. A local nominal array
copied out of the field likewise keeps its existing fixed-array operations (ADR 0462, ADR 0463), so
`local[0].kind = 5` above is the ordinary array path and not a field path. Direct field mutation
through an element store (`page.tokens[0] = ...`), element references (`ref page.tokens[0]`), field
subviews and `for` over the field remain closed and are reserved for the next. Nominal element
mismatch, a missing initializer field and a length mismatch are rejected, and resource, raw-pointer,
borrowed, tagged and array-of-array elements, nested arrays, cyclic element graphs and any array
field on a `@repr(C)` struct are rejected.

## Evaluation and lowering

The admission extends the same ADR 0468 field position from a copy scalar to an admitted plain
struct element. Semantic admission and the MIR verifier check the actual element field graph, as
they already do for the ADR 0462 array element, so a forged annotation cannot relabel a scalar or
an unrelated array. Lowering reuses the existing whole-field `field-store` and array value
transport, and the element-graph and nominal checks run against the actual local definitions, so
nominal mismatch, a missing field, a length mismatch, an array-bearing element graph and a
`@repr(C)` field are rejected with the existing type and mutability diagnostics (`E1006`,
`E1008`).

The C emitter must order the dependency graph element struct -> nominal array wrapper ->
containing struct: the element struct definition first, the nominal array wrapper that names it
second, and the containing struct last, so source declaration order never changes the emitted
order. The nominal wrapper shares the existing declaration site of the ADR 0462 array wrapper.

No new grammar production, MIR operation, wire field, artifact schema, SDK export or runtime helper
is added, because the existing array value transport already carries a struct-element field value.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or
host flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, public ABI and target maturity
stay unchanged, and this is an additive admission over ADR 0468. Exact compiler
revision/executable identity separates cached generations.

## Verification and limits

Verification is implemented and passing for the Windows dev and release native profiles together
with the MIR codec round-trip and interpreter execution; Linux is limited to C emission, and actual
Linux execution stays deferred to the owner's batch. The positive source fixture is
`internal/compiler/testdata/inline_nominal_array_fields.fy`, driven by the
`TestInlineNominalArrayField*` tests across the semantic, compiler and CLI layers, including a warm
all-hit cache pass. Negative cases cover a nominal element mismatch, a missing initializer field, a
length mismatch, an array-bearing element graph, a nested array, a cyclic element graph,
resource/pointer/borrow/tagged elements and a `@repr(C)` array field. MIR mutations that place an
inline field on `@repr(C)` or on a non-admitted shape are rejected by the exact `V1001` inline-field
verifier guard. Source order is exercised independently of declaration order, and the `1` and `4096`
length bounds are both covered.

The regression sweep additionally found two non-plain tests that still used `i32[2]` arrays, a shape
ADR 0468 had already admitted, so those cases no longer proved rejection; the implementing unit
replaced those shapes with an `Option<i32>` field to keep the original rejection intent. This record
does not assert exact case totals or a final full-suite pass.
