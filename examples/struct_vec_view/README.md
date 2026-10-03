# Nominal struct vector views

SDK stage0.84 exposes the shared `std::vec::view<T>(ref Vec<T>) -> ref T[]` in
`std::vec`. It borrows only the initialized element prefix of the vector without
allocating or copying, and length/bounds count elements, so empty zero-backed
vectors and empty subranges are valid.

`T` may be one of the five existing integer elements (`u8`, `i32`, `i64`, `u32`,
`usize`), which aliases the per-type integer view, or an actual nonempty nominal
struct with plain language scalar leaves (`bool`, `u8`, `i32`, `i64`,
`u32`, `usize`), including acyclic nonempty nested structs under ADR 0441. For a nominal element the view cannot reinterpret a `Token` view
as a same-layout `Other` view. Cycles, missing definitions and array, resource,
borrow, raw-pointer, enum, Option and Result fields at any depth are rejected. The view is shared-only: there is no
`ref mut` struct slice or fixed-array conversion. Named immutable shared struct
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

ADR 0443 now uses `token.position = Span { ... }` inside `rebase` to replace the
whole copied span. Direct and nested aggregate stores support the same exact-type
plain graph; mutable field borrows and struct-view element stores remain closed.
Run the existing portable `verify.py` for actual CLI dev/release execution, or
`bun tools/run-go.ts test ./internal/cli -run '^TestPlainStructFieldStore(Native|Cache)$' -count=1`.
Current evidence: `docs/ops/evidence/plain-struct-field-store-20261004.json`.
