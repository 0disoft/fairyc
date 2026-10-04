# Nominal struct vector views

SDK stage0.85 exposes the shared `std::vec::view<T>(ref Vec<T>) -> ref T[]` in
`std::vec`. It borrows only the initialized element prefix of the vector without
allocating or copying, and length/bounds count elements, so empty zero-backed
vectors and empty subranges are valid.

`T` may be one of the five existing integer elements (`u8`, `i32`, `i64`, `u32`,
`usize`), which aliases the per-type integer view, or an actual nonempty nominal
struct with plain language scalar leaves (`bool`, `u8`, `i32`, `i64`,
`u32`, `usize`), including acyclic nonempty nested structs under ADR 0441. For a nominal element the view cannot reinterpret a `Token` view
as a same-layout `Other` view. Cycles, missing definitions and array, resource,
borrow, raw-pointer, enum, Option and Result fields at any depth are rejected. This example uses the shared form. ADR 0448 separately adds
`vec::view_mut<T>` with checked whole-element replacement; indexed field
stores/borrows and fixed-array struct conversion remain closed. Named immutable shared struct
views support copied-value `for-in` under ADR 0440.

An index copies one whole struct value out of the view, and `view[i].field` reads
are allowed. Element and field stores are rejected, while a `var ref Token[]`
binding may still be reassigned to another view. A copied mutable local is
independent of the owner; the native fixture holds one copy while the owner is
cleared and reads it back unchanged. A live view keeps the whole owner
shared-borrowed, so move, replacement, reallocation and structural mutation
(including no-op `reserve`/`truncate`) wait for the last retained use, and index or
slice bounds cannot mutate the owner. ADR 0415 sibling/ancestor rules still apply.

## Example

`src/tokens.fy` is a bounded ASCII lexer. Its `Token` contains a nested `Span`
with byte offsets `[start,end)`, validated during copied-value iteration. It recognizes identifiers (letters,
digits, underscore), decimal integers, `+`, `=` and `;`, and ignores ASCII
whitespace. The nonnegative i64 literal and the running sum are checked for
overflow. `src/main.fy` builds a `Vec<Token>`, takes a shared view and a
prefix-through-first-semicolon subrange, and reports token/identifier/number counts
and the summed numeric values. With no argument it analyzes the built-in source and
prints sum `4000000007`.

An optional first argument is the ASCII source to analyze. It is limited to 4095
UTF-8 bytes and 128 tokens; other bytes, overflow, exceeding a limit or an
allocation failure exit 2 with a diagnostic. The program performs lexing and
arithmetic statistics only: it has no grammar and does not evaluate expressions.

## Commands

From this directory:

```sh
fairyc check
fairyc build
fairyc build --release --locked
fairyc run --locked
fairyc run --release --locked
fairyc run --locked -- "price = 4000000000 + fee; other = 7;"
```

Portable verification (Python standard library only) takes the compiler
executable path and runs `check`, locked development/release builds, and the
native default/singleton/empty/invalid/overflow/limit checks; it writes the raw
result to the ignored `build/verify-results.json`:

```sh
python verify.py /absolute/path/to/fairyc
python verify.py C:\path\to\fairyc.exe
```

Focused Go tests use the portable harness (actual Linux execution is deferred):

```sh
bun tools/run-go.ts test ./internal/cli -run '^TestStruct(For|VecView|Token)' -count=1
```

Windows amd64 MSVC native development and release each run eight struct-view cases
(three success, five checked `bounds` traps) and twelve token-analyzer cases.
ADR 0440 adds seven struct-iteration cases per profile, including normal and
error exits, nested/mixed loops and checked sum overflow.
The analyzer and first-semicolon discovery use `for (let Token token in view)`
(with a qualified item type in the main module). The owner stays borrowed through
every loop exit; `break` and `continue` follow the usual C-shaped loop behavior.
Mutable view bindings, reference items and expression-valued iterables are rejected.

Raw receipts are under `.cache/struct-vec-view-native` and
`.cache/struct-token-native`; the checked receipt is
`docs/ops/evidence/struct-views-20261004.json`. See ADR 0439.

ADR 0441 also exercises nested copies, repeated child types, empty views and
bounds traps in `TestStructNestedNative`, plus forbidden nested fields, source
loan checks, private MIR mutations/roundtrip and a nested warm-cache hit.
The current review receipt is `docs/ops/evidence/struct-nested-20261004.json`.

ADR 0442 allows `copy.position.start = ...` and checked integer compound stores
on a mutable copied token or an existing `ref mut` token. The analyzer now rebases
a copied token's span to its segment's first byte through `rebase(ref mut Token, usize)`;
the borrowed vector and output format are unchanged. Whole-root loans still
protect shared views. At that revision aggregate-field replacement and mutable
field borrows remained outside the extension. Evidence:
`docs/ops/evidence/nested-field-store-20261004.json`.

At ADR 0443, the example used `token.position = Span { ... }` inside `rebase` to replace the
whole copied span. Direct and nested aggregate stores support the same exact-type
plain graph; at that revision mutable field borrows remained closed. Shared
struct-view element stores remain closed.
Run the existing portable `verify.py` for actual CLI dev/release execution, or
`bun tools/run-go.ts test ./internal/cli -run '^TestPlainStructFieldStore(Native|Cache)$' -count=1`.
Current evidence: `docs/ops/evidence/plain-struct-field-store-20261004.json`.

ADR 0445 now rebases through a mutable field reference in the copied local token:

```fairy
let ref mut Span position = ref mut local.position;
rebase(position, origin);
```

`rebase(ref mut Span, usize)` edits the stored start/end fields. The child holds
the entire copied token until its last use; the following local assertion is
allowed once `rebase` returns. The shared vector remains unchanged, and output
and input limits are preserved. Resources/nonplain graphs and simultaneous
mutable sibling borrows remain outside the admitted projection. Run the same
portable `verify.py`; no additional harness is needed. Current public CLI and
dev/release receipt: `docs/ops/evidence/mutable-field-example-20261004.json`.

ADR 0446 additionally allows explicit shared field projection from the named
mutable plain struct reference. `rebase` now reads its original span width from
two shared scalar field references before adjusting the mutable span:

```fairy
let ref usize start = ref position.start;
let ref usize end = ref position.end;
let usize width = *end - *start;
position.start -= origin;
position.end -= origin;
assert(position.end - position.start == width);
```

The last shared use ends at the width calculation, so the following writes are
allowed. A retained shared child/alias would block writes through its parent,
parent aliases or any sibling. Shared reads and shared siblings may coexist;
reference descriptor rebinding preserves the earlier child's original referent.
The borrowed vector and report format stay unchanged. No new validation harness
is required. Current dev/release CLI evidence:
`docs/ops/evidence/shared-field-example-20261004.json`.
ADR 0447 fixes the prior MSVC C4702 assertion/overflow reproduction.
The mutable-view source fixture is `internal/compiler/testdata/mutable_struct_vec_view.fy`;
this analyzer continues using shared views and copied local tokens.
