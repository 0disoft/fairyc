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

The first executable fixed-array subset accepts `T[N]` only when `T` is `bool`, `i32`, `u8`,
`u32`, or `usize` and `1 <= N <= 4096`. The ceiling is a provisional Stage 0 implementation guard
for the initial by-value representation, not a stable language or ABI limit. A nonempty list
`[a, b, ...]` constructs an inert copy value and evaluates its elements exactly once from left to
right. `array[index]` requires `usize`, returns `T`, and traps with `bounds` when the index is at
least `N`. `array.len` evaluates the receiver once and returns exact canonical `N` as `usize`
without a scan, allocation, runtime call, memory read, or trap. Empty and repeated literals remain parser-valid but unsupported until zero-size
representation and expansion-budget contracts exist. Array fields, nesting, resource elements,
mutable indexing, slicing, properties other than `.len`, broad array coercions, and foreign
signatures remain closed. One direct shared borrow of an admitted `u8[N]` place contextually
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

Assignment remains a statement, never an expression. Mutable `i32`, `u8`, `u32`, and `usize`
locals and admitted mutable indexed scalar places support `+=`, `-=`, `*=`, `/=`, `%=`, `<<=`,
`>>=`, `&=`, `^=`, and `|=`. A compound assignment evaluates its target place once, captures the
old scalar value, evaluates the right-hand side once, applies the corresponding existing checked
binary operation, and stores only on success. Bounds, overflow, division, remainder, and shift
traps therefore leave the target unchanged. The right-hand side must have exactly the target type;
there are no promotions, wrapping shortcuts, resource compound assignments, field-target support,
or returned assignment value.

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

## Explicit exclusions

Function overloading, default and named arguments, operator overloading, user implicit conversions, C declarators and casts, preprocessing, textual includes, mutable globals, macros, user-defined generic aggregates/methods and inferred instantiation, dynamic dispatch, closures, async, rich patterns, type-wide must-use contracts, resource-bearing user enums, source-level uninitialized storage, character literals, raw strings, interpolation, octal literals, numeric suffixes, and hexadecimal floats remain outside Stage 0.
