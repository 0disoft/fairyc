# Surface Syntax v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Original boundary decision: `docs/adr/0005-stage-0-surface-and-module-boundary.md`
- Current surface decision: `docs/adr/0136-c-shaped-source-surface.md`
- Classic loop decision: `docs/adr/0213-classic-scalar-for-loop.md`
- Explicit scalar generic decision: `docs/adr/0385-stable-explicit-scalar-generics.md`
- Grammar decision: `docs/adr/0009-stage-0-grammar-and-parser-contract.md`
- Normative grammar: `spec/grammar/stage0.ebnf`

> This document and the normative grammar describe the C-shaped surface accepted by the current
> compiler. The immediately preceding Rust-shaped source revision is migration input only and is not
> an alternate product grammar.

## Lexical and declaration rules

- ADR 0478 admits `@repr(c::uint) enum Status { Ready; Busy; }`, using any admitted C integer
  base. These payload-free enums use zero-based declaration ordinals and may appear in repr(C)
  record fields. This reuses the existing attribute grammar; explicit discriminant values and
  payloads are excluded. See [ADR 0478](../adr/0478-explicit-c-enum-fields.md) for the bounded
  source and transport rules.

- Source files use `.fy` and UTF-8. Stage 0 identifiers are ASCII letters, digits, and underscore and cannot start with a digit.
- Simple statements end with explicit semicolons. Compound `if`, standalone `{ ... }`, and
  standalone `unsafe { ... }` block statements do not; there is no automatic insertion.
- Written declarations put one complete `Type` before one name. C name-splitting declarators,
  multiple declarators, and `typedef`-dependent parsing do not exist.
- Functions use `fn Result name(Parameters)`. Every result type is explicit; `void` explicitly marks
  the absence of a returned value.
- Language 14 additionally permits ordinary generic functions as `fn T identity<T>(T value)` and
  explicit calls as `identity<i32>(42)`. Specialization precedes ordinary typed HIR/MIR; see the
  bounded generic function contract below.
- Source-level locals have initializers. `let` prevents reassignment; `var` permits reassignment and mutable borrowing.
- Language 15 admits `.field` in record initializers as exactly `.field = field`, using
  the ordinary lexical value of that name. Explicit and shorthand fields may be mixed.
  Field completeness, duplicate checks, visibility, evaluation order and resource moves
  are unchanged. This adds no defaults, spreads or implicit copies (ADR 0395).
- A written local is `let Type name = value;` or `var Type name = value;`. The inferred forms are
  exactly `let name = value;` and `var name = value;`; `auto` is not a source declaration marker.
- Parameters use `Type name`. A callee that needs a reassignable local explicitly initializes a
  `var` local from its parameter; parameters have no separate `mut name` spelling.
- A `let` resource may still be consumed. Binding immutability is not ownership immobility.
- A by-value position consumes a resource. Stage 0 has no `move` expression or `move` argument keyword.
- Globals are private compile-time `const` only. The first executable set is `bool`, `u8`, `i32`,
  and `u32`; public constants, mutable globals, inferred types, aggregate values, and constant array
  lengths are excluded.
- Classic counter loops use `for (initializer; condition; post) { ... }`. Each clause is optional;
  an omitted condition is `true`. The initializer owns a lexical scope covering the condition,
  body, and post clause. The admitted v3 header subset is one scalar declaration or assignment,
  one pure boolean condition, and one direct scalar assignment. The post and direct-assignment
  initializer may use ordinary `=` or an admitted compound assignment. Calls, borrows, indexing, blocks,
  propagation, resource movement, `++`, `--`, and comma expressions are not header forms.
- Language 19 also permits `for (let u8 byte in view) { ... }` when `view` is a named
  immutable local or parameter of exact type `ref u8[]`. The view length is
  captured once; every iteration copies one checked byte. `continue` advances
  the hidden index, and exits use ordinary scalar-loop cleanup (ADR 0403).
- ADR 0457 also admits direct named fixed-array iteration for `u8`, `i32`, `i64`,
  `u32` and `usize`: `for (let T item in array)` copies each element;
  `for (let ref mut T item in array)` lends each stored element and requires
  a mutable array binding. The whole original array is borrowed through loop
  exit, without copying it into a temporary. Copied loops permit shared owner
  reads but block mutation; exclusive loops block separate owner access even
  after an item's last use. `continue` advances the hidden index; break,
  return and propagation release the loop-held loan at exit. Current arrays
  retain the `1..=4096` size guard. Struct arrays, array fields and
  literal/call iterable roots are not admitted.
- ADR 0458 admits `for (let ref T item in source)` for the same five integer
  types over named fixed arrays and immutable named `ref T[]`/`ref mut T[]`
  bindings. Items reference original storage and cannot be written through.
  The shared loop loan blocks owner writes and exclusive reborrows through
  exit. Items may be returned from borrowed parameters or retained in
  `Option<ref T>` under existing lifetime rules; local/by-value array storage
  cannot escape. Shared array loops may nest; length and checked ascending
  traversal are unchanged.
- ADR 0459 also admits `for (let ref Token item in items)` over immutable
  named shared or mutable plain struct view bindings. Items reference stored
  elements, including nested scalar-field structs, and can be returned or
  retained under existing lifetime rules. The whole loop blocks owner writes
  and exclusive reborrows; live returned references retain that protection.
  Mutable items require mutable views. Resource/pointer/borrowed/array/tagged
  field graphs, computed
  iterable roots and fixed struct arrays were deferred there and are admitted by ADR 0462.
- ADR 0460 admits copied plain struct items over immutable named mutable view
  bindings as well as shared views: `for (let Token item in items)`. Each item
  copies the current stored element before the body runs. Writes through the
  mutable descriptor are allowed; later iterations observe earlier changes to
  not-yet-visited elements. Retained value copies survive owner cleanup after
  loop exit. The loop still holds the owner against structural mutation/moves;
  existing live element-reference loans still restrict competing writes.
- ADR 0461 extends the scalar view/iteration path to `bool`. Existing `bool[N]`
  arrays can yield `ref bool[]`/`ref mut bool[]`, checked element references,
  and copied/shared/exclusive loop items. Named view bindings must remain
  immutable; exclusive array items require a mutable array. Bool subviews
  preserve exact element type, bounds checks, owner lifetime and parent-freezing
  rules. Shared-reference loops block competing writes through loop exit;
  copied mutable-view loops allow writes and see current later elements.
  Bool storage is not bit-packed, and no numeric-to-bool conversion is added.
- ADR 0462 admits fixed arrays of nonempty plain structs with bool/integer
  leaves, including nested plain structs. Arrays support value copies, list and
  repeated initialization, checked element loads/replacement, element references,
  shared/exclusive views and copied/shared/exclusive named-array loops. A repeated
  initializer is evaluated once and its elements are independent whole values.
  Direct indexed field stores use a mutable view or an explicit element reference
  in this slice. Resources, pointers, borrows, tagged fields and array fields stay
  excluded; exact nominal identity and whole-owner borrow rules still apply.
- ADR 0463 admits direct indexed field stores `array[index].field = value`,
  nested scalar and nested plain-struct field paths, checked scalar compound
  assignment and whole plain-struct field replacement on named mutable fixed
  arrays of admitted plain structs. The place root is the array; the index
  evaluates once under the whole-owner checks, then bounds, the compound read and
  the right-hand side run before one field store. Immutable, temporary and shared
  roots, competing index or right-hand-side owner access and live shared/
  exclusive references stay rejected. Resources, pointers, borrows, tagged fields
  and array fields remain excluded.
- ADR 0464 admits direct struct array field references `ref array[index].field`
  and nested paths such as `ref mut array[index].inner.field` on named fixed
  arrays of admitted plain structs. The reference rebases the existing field
  projection on a checked element borrow of the original array storage, never a
  copied element. Scalar and whole nested plain-struct fields are admitted;
  shared references also come from immutable local or by-value array bindings
  and exclusive references require a mutable array root. The whole array stays
  the loan owner, the index runs once under bounds dominance and fields or
  indices never prove disjointness. Local/by-value array references cannot
  escape and temporary or call roots stay rejected. Resources, pointers,
  borrowed struct fields, tagged fields and array fields remain excluded.
- ADR 0465 admits direct slice and subview iteration `for (let T item in
  data[start..end])` over a bare named fixed array or an immutable named
  `ref T[]` / `ref mut T[]` view binding, with explicit or omitted bounds and
  the same `bool`/integer/plain-struct item set. The root stays a bare name;
  computed, call, field and nested-slice roots and mutable (`var`) descriptor
  bindings remain closed. Bounds evaluate once before the loop and the runtime
  checks `start <= end <= len` before the body; omitted bounds and empty
  subviews are valid. Arrays are borrowed implicitly as shared for copied/shared
  items and require a mutable array for exclusive items; slicing a view keeps the
  parent's mutability even for copied/shared items, and a mutable subview
  generation freezes the parent descriptor and its aliases for the loop duration
  under ADR 0415. Owner loan, break/continue/return and escape rules are reused.
- ADR 0467 admits explicit `ref`/`ref mut` borrows of a slice range over a bare
  named local or by-value parameter fixed array: `let ref Point[] middle =
  ref points[1..3]` and `adjust(ref mut points[1..3])`, with grouping around the
  base and explicit, half-open or omitted bounds. Element types follow the
  admitted fixed-array scalar and plain-struct set. Bounds evaluate once, start
  then end, and the runtime checks `start <= end <= len`; empty ranges are valid
  and out-of-range slices trap `bounds`. The root stays a bare name, so computed,
  call, temporary, field and nested-slice roots and ordinary slicing outside this
  explicit borrow remain closed. Exclusive borrows require a mutable `var` owner
  (`E1008`). Shared reads and borrows coexist, and only conflicting access is
  `E2101`: an owner write or exclusive borrow conflicts with a live shared subview,
  and an exclusive subview conflicts with owner reads and writes and every
  competing borrow. Local or by-value views cannot escape (`E2102`).
- ADR 0468 admits bounded inline scalar fixed-array fields on non-`@repr(C)`
  structs: `struct Packet { u8[4] bytes; usize length; }`, with element types
  `bool`, `u8`, `i32`, `i64`, `u32` and `usize` and length `1..=4096`.
  Fields receive no implicit initialization, so every struct literal supplies
  each array field explicitly with a list or repeated initializer. Whole-field
  reads and copies (`let u8[4] bytes = packet.bytes;`) and replacement through a
  named mutable owner or `ref mut` owner (`packet.bytes = [8, 9, 10, 11];`,
  `envelope.packet.bytes = [6; 4];`) reuse the existing array value transport,
  so by-value structs, arrays of structs and `Option` payloads carry independent
  copies and the copy cost scales with the array contents. Nested arrays, arrays
  of nominal structs, resources, raw pointers, borrowed, tagged and array-of-array
  fields stay closed and `@repr(C)` keeps rejecting array fields. Direct
  array-field element stores and `ref` field views/subviews are admitted by
  ADR 0469 over this same bounded field, and the whole-field position is
  extended to an admitted plain-struct element by ADR 0471.
- ADR 0469 admits direct element access and explicit `ref`/`ref mut` views over
  the bounded inline scalar array fields of ADR 0468 on plain non-`@repr(C)`
  structs: element stores and checked compound stores (`packet.bytes[0] = 1;`,
  `packet.bytes[0] += 1;`), element references (`ref packet.bytes[0]`,
  `ref mut packet.bytes[0]`), whole-field views (`let ref u8[] all =
  ref packet.bytes;`) and field subviews with explicit, half-open, omitted or
  empty bounds (`fill(ref mut envelope.packet.bytes[1..3]);`). The root must
  resolve to a named local or a by-value/`ref`/`ref mut` parameter through
  field and group projections, so call, temporary, raw-pointer, tagged,
  indexed-aggregate and nested-slice roots stay closed; ordinary array-field slicing stays closed
  outside the explicit `ref`/`ref mut` borrow and the ADR 0470 direct `for` source, so a view is
  no longer required first. Element access and borrowing reuse the existing array-view,
  view-index and view-element-borrow operations on the resolved field address with
  no array copy or allocation. The whole struct is the single loan owner with no
  field or range disjointness: shared views coexist with shared owner reads, an
  owner write or exclusive borrow while a shared view is live is `E2101`, an
  exclusive view blocks owner reads and writes and every competing borrow, an
  exclusive view needs a mutable owner (`E1008`), width and non-`usize` bound
  mismatches are `E1006`, and local or by-value escapes are `E2102`. Indices
  and bounds evaluate once with the original storage selected first, bounds trap
  before element access and an early index or bound return skips the rest. Nested
  arrays and arrays of nominal structs stay closed.
- ADR 0470 admits direct `for` iteration over the bounded inline scalar array fields of
  ADR 0468: `for (let T item in packet.bytes)`, `for (let ref T item in
  packet.bytes[lo..hi])` and `for (let ref mut T item in envelope.packet.bytes[..])`. The
  place root must resolve to a named plain non-`@repr(C)` struct local or a
  by-value/`ref`/`ref mut` parameter through field and group projections, so call,
  temporary, indexed-aggregate, tagged, raw-pointer and nested-slice roots stay closed. Ordinary
  array-field slicing stays closed outside the direct `for` source and the explicit
  `ref`/`ref mut` field view of ADR 0469. The item set is
  exactly the field's scalar element type (`bool`, `u8`, `i32`, `i64`, `u32`,
  `usize`). The loop reuses the implicit descriptor of the named-array and named-view loops
  over the explicit field view of ADR 0469, so no array copy or allocation is added. The whole
  struct is the loan owner with no field or range disjointness: a shared loop blocks owner
  writes and exclusive borrows through exit, an exclusive loop blocks owner reads and writes and
  every competing borrow and freezes the parent descriptor and aliases under ADR 0415, and
  exclusive items require a mutable owner (`E1008`). Bounds evaluate once left to right before
  the length snapshot and the body, and the runtime checks `start <= end <= len`, so omitted
  bounds and empty subviews are valid and out-of-range slices trap `bounds`. Existing
  `break`/`continue`/`return` cleanup and escape rules are reused.
- ADR 0471 admits a bounded inline fixed-array field whose element is an admitted
  plain struct (ADR 0462) on a non-`@repr(C)` struct:
  `struct APage { ZToken[4] tokens; usize used; }`, where `ZToken` is an actual
  nonempty acyclic struct with `bool`/`u8`/`i32`/`i64`/`u32`/`usize` leaves and
  nested ordinary structs but no array field anywhere in its field graph. Length
  is `1..=4096` and nominal identity stays exact. Fields receive no implicit
  initialization, so every struct literal supplies each array field with a list
  or repeated initializer of that element struct. Whole-field reads and copies,
  and replacement through a named mutable owner or `ref mut` owner including a
  nested plain parent path, reuse the existing array value transport, so by-value
  structs, calls, returns, arrays of structs and `Option` payloads carry
  independent copies and the cost scales with the contents. Declaration order is
  free, so the element struct may follow its container. A direct element read
  (`let ZToken first = page.tokens[0];`) selects the original field descriptor and
  copies only the selected element value, so it does not read or copy the whole
  field, and a local nominal array copied out of the field keeps its existing
  fixed-array operations. Direct field
  mutation through an element store, whole-element replacement, element references,
  explicit views, field subviews and `for` over the field are admitted by ADR 0472
  over this same field position. Nominal element mismatch, a missing initializer field
  and a length mismatch are rejected, and resource, raw-pointer, borrowed, tagged
  and array-of-array elements, nested arrays, cyclic element graphs and any array
  field on a `@repr(C)` struct are rejected.
- ADR 0472 admits direct element access, explicit `ref`/`ref mut` views and subviews,
  and `for` iteration over the ADR 0471 inline plain-struct array field on plain
  non-`@repr(C)` structs: scalar field stores (`page.tokens[0].kind = 2;`),
  whole-element replacement (`page.tokens[0] = ZToken { .kind = 3, .flags = 1 };`),
  shared/exclusive whole-field views (`let ref ZToken[] all = ref page.tokens;`,
  `ref mut ZToken[]`), field subviews (`ref page.tokens[0..]`), shared element or
  scalar-field references, and copied/shared/exclusive `for` items over the same
  element. The place root resolves to a named plain-struct local or a
  by-value/`ref`/`ref mut` parameter through field and group projections, so call,
  temporary, raw-pointer, tagged, indexed-aggregate and nested-slice roots stay
  closed; the whole-field borrow must be the unsized `ref T[]`/`ref mut T[]`, so a
  sized `ref T[N]` field binding stays closed. Element access, views, subviews and
  iteration select the original field storage and reuse the existing array-view,
  view-index, view-element-borrow and view-slice operations, so no whole-array copy
  or allocation is added; a direct element value read or a copied `for` item still
  copies that one element, and whole-field value reads and copies keep independent
  values (ADR 0471). The whole struct is the single loan owner with no field, index
  or range disjointness: a shared view or element reference coexists with shared
  reads, an owner write or exclusive borrow while one is live is `E2101`, an
  exclusive view, element reference or loop item needs named mutable storage (`var`
  local, `ref mut` parameter or a local/parameter holding a `ref mut` owner; `E1008`),
  bounds and element mismatches are `E1006`, and local or by-value escapes are
  `E2102`. Indices and bounds evaluate once in written order with the original
  storage selected first, bounds trap before element access and an early index or
  bound return skips the rest.
- Lexical cleanup uses `defer direct_void_call(copy_scalar_arguments...);`. Arguments evaluate
  immediately and the retained calls execute LIFO when their block exits normally or through
  `return`, `?`, `break`, or `continue`. Traps do not unwind through defers.
- An ordinary or raw foreign function may declare one closed external-capability set with
  `@effects(clock, console, environment, filesystem, network, process, random)`. Arguments use only
  the effects actually required; omission is the empty set. Exact semantics are owned by
  `0008-effects-and-capabilities-v0.md`.

- A private safe effect-free free function may use one zero-argument `@const_eval` attribute when
  all parameters and its result are `bool`, `u8`, `i32`, or `u32` and its body is exactly one
  explicit `return expression;`. Calls outside a constant context remain ordinary runtime calls.
- Module-level `static_assert(expression);` is contextual item syntax rather than a globally
  reserved identifier. It requires one compile-time `bool` value and emits no runtime operation.
- Compile-time expressions are closed to scalar literals, parameters, private module constants,
  admitted scalar operators and widening casts, conditionals, and eligible `@const_eval` calls.
  Loops, mutation, recursion, effects, unsafe, foreign calls, resources, borrows, pointers,
  target-width values, allocation, I/O, and custom assertion messages are excluded.

## Modules and imports

Each source file declares exactly one module:

```fy
module app::reader;

import std::fs as fs;
import std::text as text;
```

The driver supplies one source root for each resolved package instance. Removing `.fy` from a source-root-relative path and joining its valid identifier segments with `::` produces the required module declaration. `src/net/http.fy` therefore declares `module net::http;`. Package coordinates and instance identities do not participate in this path.

Every import has an alias and names remain qualified through that alias. A module path may resolve in the current package or exactly one direct dependency instance. Transitive dependency modules are not visible; zero providers is missing and multiple visible providers is ambiguous. The standard library follows the same explicit dependency rule and creates no implicit prelude. Wildcard imports, selective imports, re-exports, and import cycles are excluded.

`pub` grants source-module visibility only. It does not export a stable binary symbol, layout, mangling, or calling convention.

## Package-owned contract state

Language 13 admits one explicit persistent-state schema per package:

```fy
@contract_uid("example.token")
state {
    cell u64 total_supply;
    map bytes20 u64 balances;
}
```

`state`, `cell`, and `map` are contextual language-13 declaration words, so existing ordinary names
remain source-compatible. The required `contract_uid` string owns the
stable storage namespace; source order owns field indexes. The closed storage-codec vocabulary is
`bool`, `u32`, `u64`, `bytes20`, and `bytes32`. These codec spellings are not general user-defined
generic types or new ordinary arithmetic types. A package cannot declare more than one state block,
and a state block cannot be `pub`. State access expressions and native persistence remain outside
this declaration-only slice.

## Types

Stage 0 type syntax includes:

```text
bool, void
i8, u8, i16, u16, i32, u32, i64, u64, isize, usize
c::bool, c::char, c::double, c::float, c::int, c::long, c::longlong, c::schar
c::short, c::size_t, c::ssize_t, c::uchar, c::uint, c::ulong, c::ulonglong, c::ushort
f32, f64
Path::Type
ref T, ref mut T
ptr T, ptr mut T
T[], T[N]
Option<T>, Result<T, E>
Region
unsafe extern C fn T(...)
```

`str` and `CStr` are unsized compiler-known view types and appear as `ref str` and `ref CStr`.
User-defined generic aggregate declarations remain excluded. Ordinary generic functions use the
bounded contract below; built-in `Option`, `Result`, slices, and arrays retain their existing type
argument rules. `T[]` is unsized and cannot appear by value; safe views use `ref T[]`
or `ref mut T[]`.

Bounded generic functions follow this contract:

ADRs 0384/0385 admit explicit ordered type parameters on ordinary module functions, including
public functions reached through exact package imports. The executable ordinary builtin-scalar
argument set is `bool`, `u8`, `i32`, `u32`, and `usize`; a generic call may also forward one of its
own type parameters. Syntax recognition of the larger scalar vocabulary above does not admit
additional typed-HIR types. In particular, planner recognition alone cannot make `i8`, `i16`,
`i64`, `u16`, `u64`, `isize`, `f32`, or `f64` executable. C scalars and nominal/composite generic
arguments are not admitted to user function templates by the source expansion contract.

ADR 0479 separately admits distinct `f32`/`f64` types for repr(C) record fields and local/internal
value transport from C-produced records. This does not widen the generic scalar argument set.
Floating literals, operators, comparisons and conversions remain unsupported; see
[ADR 0479](../adr/0479-repr-c-floating-field-transport.md) for the exact transport boundary.

ADR 0390 adds a controlled SDK exception: `std::vec::Vec<T>` and its SDK-owned
operations accept supported scalars and directly named non-generic structs/resources
through stable Windows workspace compilation. The SDK is authenticated by its
complete source bundle. This does not authorize user generic aggregate declarations,
nominal arguments to user functions, nested generic types or private storage intrinsics.

Every reached instantiation is substituted to concrete types before ordinary semantic/Flow and MIR
verification. Identical requests are deduplicated by exact defining-package identity; repeated
source calls still execute. Inference, generic methods, user generic resource/struct/enum declarations,
static interfaces, recursive specialization and unbounded expansion remain excluded. Unused source
templates are retained for identity and documentation but are not proof of executable body checking.
The stable workspace and explicit-file CLI entry points perform specialization; raw semantic-only
entry points reject generic syntax instead of silently erasing it.

A safe borrow whose referent type is `str` or `u8[]` exposes the read-only property `.len` with
result type `usize`. The property reports the view's byte count, including embedded NUL bytes and
multi-byte UTF-8 encodings. It reads existing view metadata without scanning, allocation, a call, or
a trap. The same views expose `.is_empty` with result type `bool`, true exactly when the retained
byte count is zero and with the same no-scan and no-call guarantees. These properties do not define
capacity, Unicode scalar or grapheme counting, `CStr` metadata, or a public view representation.

An owned `Result<T, E>` exposes `.is_ok` as a consuming boolean test. It handles the Result through
the existing exhaustive two-variant switch semantics, including active-payload cleanup. The
receiver is evaluated once; borrowed Results and `Option` do not expose this property.

An immutable `ref u8[]` supports the read-only expression `bytes[index]`. The index has type `usize`;
an unsuffixed integer literal receives that contextual type. The result has type `u8`. The receiver
and index evaluate once from left to right, and an index greater than or equal to `.len` traps with
category `bounds` before any data access.

A mutable fixed scalar-array place supports `values[index] = value` and checked integer compound
assignment when the receiver is a `var`
local. A value of type `ref mut u8[]` supports both the same checked `u8`
read and `bytes[index] = value`; the borrow binding itself may remain `let` because the borrow type,
not binding reassignment, grants referent mutation. A shared view, immutable array place, literal,
call result, or other non-place target is rejected. The receiver and `usize` index evaluate once from
left to right, the complete `bounds` check and old-value read run before the right-hand side, and only
the success path evaluates the value once and stores it. Mutable slicing, nested projections,
resource elements, and user-defined indexing remain outside Stage 0.

The same immutable view supports end-exclusive `bytes[start..end]`, `bytes[..end]`,
`bytes[start..]`, and `bytes[..]`. Written bounds have type `usize`; an omitted start is zero and an
omitted end is the retained length of the already-evaluated receiver. The receiver and every
written bound evaluate once from left to right. The result is `ref u8[]` with the receiver's borrow
origin when `start <= end && end <= bytes.len`; otherwise evaluation traps with `bounds` before
deriving a pointer. This does not admit inclusive bounds, `str` indexing or slicing, indexed
assignment, mutable-view indexing or slicing, or unchecked access.

The executable fixed-array subset accepts `T[N]` when `T` is `bool`, `i32`, `i64`, `u8`,
`u32`, `usize`, or an admitted plain scalar-field struct (ADR 0462), and `1 <= N <= 4096`. The ceiling is a provisional Stage 0 implementation guard
for the initial by-value representation, not a stable language or ABI limit. A nonempty list
`[a, b, ...]` constructs an inert copy value and evaluates its elements exactly once from left to
right. `array[index]` requires `usize`, returns `T`, and traps with `bounds` when the index is at
least `N`. `array.len` evaluates the receiver once and returns exact canonical `N` as `usize`
without a scan, allocation, runtime call, memory read, or trap. `[value; count]` evaluates `value`
once and copies that scalar or plain struct into each element; the count is an integer literal or evaluated global
integer constant in `1..=4096`. Empty and zero-count arrays remain unsupported until zero-size
representation is defined. See [ADR 0416](../adr/0416-repeated-scalar-array-initialization.md).
Named mutable arrays support checked whole-element replacement. Arrays can produce
shared/exclusive typed views; slicing is performed through those views. Array fields are limited
to the bounded scalar array fields admitted by ADR 0468 and the bounded plain-struct-element array
fields admitted by ADR 0471; nested arrays, resource elements, properties other than `.len`, broad
array coercions, and foreign signatures remain
closed. One direct shared borrow of an admitted `u8[N]` place contextually
becomes `ref u8[]` when that view type is already expected, as in
`let ref u8[] bytes = ref array` or an internal `ref u8[]` call argument. A direct exclusive borrow
of a mutable admitted `u8[N]` place likewise becomes `ref mut u8[]` only where that exact type is
expected.
Both views retain the array's origin and exact `N`; other element types, bare values, temporary
sources, inferred conversions, and escaping local views remain closed. The mutable view may cross
internal calls and expose `.len`, but it adds no safe indexing, assignment, slicing, or resizing.
A `var` safe-borrow local may be
reassigned in straight-line, branch, or `while` control flow; reachable branch, header,
`continue`, and `break` origins are joined and returning a multi-origin result is rejected.

Internal function parameters may name an exact `ref T[N]` or `ref mut T[N]` for an admitted fixed
array. This is a safe borrow of the whole array, not a slice, and does not add resizing, mutable
indexing, array fields, foreign-reference parameters, or a public ABI. Contextual `ref u8[N]` to
`ref u8[]` and `ref mut u8[N]` to `ref mut u8[]` conversion preserve their respective shared or exclusive
loan; neither converts between mutabilities.

Ordinary internal functions may accept `ref str` and `ref u8[]` parameters. Calls preserve the exact safe
borrow and its caller-proven origin; they do not convert the argument to a raw pointer or transfer
ownership. A safe function may contain an explicit lexical `unsafe` block and expose a checked
result to safe callers, but foreign declarations themselves still reject safe-borrow parameters and
the internal view transport is not a public ABI.

Unsafe code may call exactly `str::from_raw_parts(owner, data, length)`,
`CStr::from_raw_parts(owner, data, storage_length)`,
`byte_slice::from_raw_parts(owner, data, length)`, or
`byte_slice::from_raw_parts_mut(owner, data, length)`. The text and shared-byte forms require a
shared safe owner borrow. The mutable-byte form requires an exclusive safe owner borrow and a
mutable raw `u8` or opaque `void` pointer. The owner defines the returned view's origin; the length
is `usize`. These are compiler-known operations, not user functions or general constructors. Safe
wrappers may return their result, but raw-pointer-only construction, mutable text views, arbitrary
view element types, and a generic public raw-parts protocol remain unavailable.

`assert(condition);` is a statement with mandatory parentheses and semicolon. Its condition is
exactly `bool`; true continues and false traps as `assertion-failed`. It remains active in every
build profile and is neither a C macro nor an unsafe optimizer assumption.

The current executable C-scalar source slice recognizes all sixteen C scalars declared by the
selected TargetSpec, preserves every one as an exact identity distinct from coincidentally sized
Fairy scalars, and permits them as transported foreign-call results, locals, and arguments. The
exact signed 32-bit `c::int` additionally converts explicitly in both directions with `i32` on
targets whose declared `c.int` range matches `i32`. Other C-scalar literals, operators,
comparisons, casts, and raw pointers remain deferred. Exact C-scalar
fields are admitted only in the current nonempty `@repr(C)` struct slice, which also admits acyclic
nested `@repr(C)` records but not arrays or enum fields. Those admitted records may be foreign-call
results by value. Foreign aggregate parameters remain deferred.
`c::size_t` uses the ISO C `<stddef.h>` typedef; `c::ssize_t` is available only for the current
`x86_64-linux-gnu` POSIX.1-2008 profile through `<sys/types.h>`. Plain `c::char` follows the selected
target's signed `char` fact rather than inheriting either `c::schar` or `c::uchar`.
Raw `ptr void` and `ptr mut void` are opaque C-interoperability pointers. Inside `unsafe`, a direct
local `ref T` may become `ptr void`, and `ref mut T` may become `ptr void` or `ptr mut void`; the conversion preserves
the concrete owner type in verified MIR and never strengthens mutability. Exact typed
borrow-to-pointer casts and immutable `ref CStr` parameter data extraction use the same explicit unsafe
boundary. An existing safe resource or fixed-array borrow may likewise erase only its referent type
to `void`; it forwards the referent pointer and never takes the address of the borrow carrier. Raw
dereference, arithmetic, integer conversion, and every other pointee-changing cast remain
unavailable.
The executable foreign slice additionally admits one pointer-to-opaque-pointer output shape such as
`ptr mut ptr mut void`; both layers must be mutable, and pointer nesting does not generalize to C scalars,
arbitrary pointees, or the other mutability combinations.

## Structs, resources, and enums

An inert struct contains only inert non-borrow fields:

```fy
struct Point {
    f32 x;
    f32 y;
}
```

A resource may own cleanup and defines at most one infallible, non-callable `cleanup` block.
Associated functions and methods use an owner-qualified function name:

```fy
resource File {
    c::int handle;

    cleanup {
        unsafe {
            os_close_best_effort(this.handle);
        }
    }
}

pub fn Result<void, CloseError> File::close(File this) {
    // target-specific consuming close
}
```

The first parameter named `this` is the receiver. Its exact forms are `ref File this`,
`ref mut File this`, or `File this`, and its owner type must match `File::`. A qualified function
without `this` is associated but is not a method. Resource fields are private outside the defining
module. A resource record literal is therefore valid only in that defining module; making the
resource type `pub` does not publish its fields or let another module bypass its ordinary factory
functions. Public inert structs keep their existing field-by-field record-literal behavior. General
field extraction from a resource remains forbidden.

Restricted user enums provide typed inert errors:

```fy
enum IoError {
    NotFound;
    PermissionDenied { c::int os_code; }
}
```

User-enum payload fields must be inert and copyable. Resource payloads remain special to built-in `Option` and `Result`, whose active-variant move and drop behavior is compiler-defined.
Payloadless construction uses contextual `.NotFound`; named payload construction uses
`.PermissionDenied { .os_code = value }`. An exhaustive shallow `switch` lists every declared
variant exactly once and binds every named field in the active payload. The expected enum type,
never a global best-fit search, resolves a leading-dot variant.

## Expressions and control flow

`if` is an expression so source code does not need deferred initialization:

```fy
let file =
    if (use_backup) {
        fs::File::open(backup)?
    } else {
        fs::File::open(primary)?
    };
```

`Result` construction uses `.ok(expression)` and `.err(expression)` only where one contextual
`Result<T, E>` type is known. A shallow Result switch uses exactly one `case .ok(binding): { ... }`
and one `case .err(binding): { ... }`. `Option` analogously uses `.some(expression)` and `.none`.
Every case owns a mandatory block, never falls through, and cannot target `break`. Restricted
user-enum patterns use shallow named inert fields and exact exhaustiveness. Whole-value wildcards,
nested destructuring, guards, or-patterns, slice patterns, and resource-field extraction are
excluded; `_` may discard only a selected payload.
In a Result switch, an arm block with no tail and a final direct `return` does not
contribute a value to the join. Other arms still need matching value types when the switch
produces a value. This does not add a source `never` type or change Option and user-enum arms.

`if`, `while`, classic `for`, and `switch` require parenthesized headers, conditions, or scrutinees.
In classic `for`, `continue` executes the post clause before the next condition and `break` skips
the post clause. Function bodies do not
return their final expression: a non-`void` function returns only through `return expression;`,
while a `void` function may use `return;` or fall through. Value-producing `if` branches and
`switch` case blocks may retain one unsemicolonized tail expression.

Assignment is a statement. Comma expressions, `++`, `--`, C ternary syntax, and integer truthiness are excluded. Ordinary arithmetic traps as defined by the product specification; wrapping, saturating, and checked behavior uses methods rather than additional operator tokens.

The fixed precedence and associativity table is in `docs/language/0006-lexing-parsing-and-recovery-v0.md`. Comparisons do not chain. Parentheses remain explicit group nodes through AST lowering.

`Result` propagation uses postfix `?`. Explicit discard is:

```fy
discard logger.flush();
```

Discarding a built-in tagged value runs drop for its active resource payload.

An ordinary internal, associated, method, or raw foreign function may carry one zero-argument
`@must_use` attribute when its result is an eligible non-`void` inert value:

```fy
@must_use
fn usize bytes_written(Writer writer);
```

The attribute is a call-site contract, not a second ownership system. A resolved annotated call
must feed a binding, assignment, return, tail return, argument, receiver, operator, condition, cast,
index, slice, aggregate, or switch, or it must be acknowledged with `discard`. A transparent group,
`unsafe` block, ordinary block, `if`, or `switch` cannot hide an ignored tail call. Storing the value
ends this obligation; general unused-local analysis is separate. `Result`, resources, safe borrows,
`Region`, and `void` use their existing rules and cannot be annotated. Malformed, duplicate,
misplaced, or ineligible declarations are `E4201`; ignored calls are `E4202`. Type-wide annotations,
custom messages, and function-value transport remain absent.

The exact zero-argument contextual intrinsic `source_location()` returns one static `ref str` in
the canonical form `<relative-path>:<line>:<byte-column>@<start-byte>`. Line and column are
one-based; the final byte offset is zero-based. The path is source-root-relative with `/`
separators, and column counts original UTF-8 bytes rather than characters or display cells. The
caller passes this value explicitly to logging or diagnostic helpers. `source_location` is reserved
in value namespaces; hidden default arguments, callee propagation, runtime stack inspection,
absolute paths, and a structured location ABI do not exist.

Construction uses the same ordinary call syntax as every other function. Infallible named factories
return `T`; validation, allocation, handle acquisition, I/O, and other fallible construction return
`Result<T, E>`. Policy stays visible in names such as `copy`, `from_text`, `open_read`, or
`create_truncate`. There is no `new` expression, constructor declaration, initializer list,
constructor overload, implicit conversion constructor, or special copy/move member protocol.

## Region syntax

A lexical region is opened through a compiler-recognized statement:

```fy
region scratch in arena {
    let bytes = scratch.copy_bytes(input)?;
    parse(bytes)?;
}
```

`Region` values cannot be constructed with a literal. A function may accept a `Region` parameter and return its single-origin borrow.

## Unsafe and C syntax

Raw operations occur in `unsafe {}` or an `unsafe fn`. Each C declaration is independent:

```fy
@link_name("read")
unsafe extern C fn c::ssize_t c_read(
    c::int fd,
    ptr mut u8 data,
    c::size_t size,
);
```

Without the attribute, the Fairy declaration name is also the external C identifier. With it, the
Fairy name remains the source lookup name and the one nonempty cooked string becomes the external
identifier. Missing, non-string, empty, multiple, or duplicate arguments are `E5106`; the backend
still rejects names outside its C11 and private-namespace policy.

There is no parallel `extern C {}` block syntax. C-layout types use `@repr(C)`. Pointer casts and raw
dereferences require unsafe code. Safe integer `as` currently admits exactly `u8` to `i32`, `u32`, or
`usize`, plus `u32` to `usize`; all four preserve every source value on every admitted target. It
also admits the exact representation-preserving pairs `i32 -> c::int` and `c::int -> i32` when the
target declares `c.int` as signed 32-bit. Narrowing, signedness-changing, other C-scalar,
pointer/integer, and other numeric conversions remain rejected and require a later explicit
fallible conversion contract.

Binary `+`, `-`, and `*` accept exactly matching `i32`, `u8`, `u32`, or `usize` operands and return
that same type. Their ordinary source meaning is checked: overflow and unsigned underflow trap as
`integer-overflow`; they never inherit C unsigned wrapping. Matching `i32`, `u8`, `u32`, or `usize`
operands also support `/` and `%`; zero traps before the operation and only the signed type has the
minimum-by-negative-one overflow case. Unary `~` accepts any one of those types; the same matching
types support binary `&`, `^`, `|`, `<<`, and `>>`. Bitwise operations preserve the exact type without trapping. Right shift uses arithmetic behavior
for `i32` and logical behavior for unsigned types. Both shifts trap on an amount outside the exact
operand width; left shift additionally traps when the mathematical result is not representable.
Mixed types and implicit promotions remain rejected.

Assignment remains a statement, never an expression. Mutable `i32`, `i64`, `u8`, `u32`, and
`usize` locals, admitted mutable indexed scalar places, admitted plain-struct scalar field places,
and named mutable integer references (`*name` or the grouped `*(name)`) support `+=`, `-=`, `*=`,
`/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and `|=`. A compound assignment evaluates its target place
once, captures the old scalar value, evaluates the right-hand side once, applies the corresponding
existing checked binary operation, and stores only on success. Through a named `ref mut` target the
selected referent address and old value are captured before the right-hand side, so a right-hand
side that rebinds the reference still stores to the original address. Bounds, overflow, division,
remainder, and shift traps therefore skip the final compound write; the right-hand side's own
effects persist, so a destination the right-hand side wrote keeps that value. The right-hand side
must have exactly the target type; there are no promotions, wrapping shortcuts, resource compound
assignments, `bool`, raw-pointer, or aggregate compound assignments, or returned assignment value.

The first `defer` surface accepts only resolved internal calls returning `void`. Explicit arguments
are limited to `bool`, `i32`, `u8`, `u32`, and `usize`; each is evaluated once at registration and
copied into compiler-owned storage. Foreign calls, returned values, resources, aggregates, pointers,
views, safe borrows, and method receivers are rejected. Deferred calls run before implicit resource
drops and Region resets for the same lexical exit. Arbitrary deferred blocks and abnormal-exit
guarantees do not exist.

Equality operators `==` and `!=` accept matching operands of `bool`, `i32`, `u8`, `u32`, or
`usize`. Ordered comparison operators `<`, `<=`, `>`, and `>=` accept matching operands of `i32`,
`u8`, `u32`, or `usize`. Equality also accepts two matching raw-pointer types for null and identity
checks; it does not dereference either operand and needs no additional unsafe scope. Comparisons do
not promote, widen, or mix types. Aggregate equality, pointer ordering, and boolean ordering remain
rejected.

## Grammar and recovery authority

`spec/grammar/stage0.ebnf` is the complete normative prototype grammar. `docs/language/0006-lexing-parsing-and-recovery-v0.md` owns source-byte handling, literal cooking, the Pratt binding-power table, module identity, recovery, and canonical byte spans. `spec/fixtures/parser/manifest-v0.json` owns positive and negative acceptance evidence.

A recovered syntax tree is retained for diagnostics only. It cannot silently legalize a missing semicolon, delimiter, import alias, invalid escape, chained comparison, or module mismatch.

## Position-preserving argument labels

An argument may repeat its positional parameter name: `f(value, .minimum = 1, .depth = 0, .allow_record = true)`. Labels do not reorder, omit, duplicate, rename, or supply defaults; each label must exactly name the parameter at that position. Positional arguments may appear before or after labeled ones. Labels are allowed only on same-module private non-generic ordinary direct calls, never on public, external, method, indirect, intrinsic, contextual-constructor, or explicitly generic calls. Evaluation order, ownership, effects, and cleanup retain their existing rules.

## Explicit exclusions

Function overloading, default arguments and argument reordering, operator overloading, user implicit conversions, C declarators and casts, preprocessing, textual includes, mutable globals, macros, user-defined generic aggregates/methods and inferred instantiation, dynamic dispatch, closures, async, rich patterns, type-wide must-use contracts, resource-bearing user enums, source-level uninitialized storage, character literals, raw strings, interpolation, octal literals, numeric suffixes, and hexadecimal floats remain outside Stage 0.
