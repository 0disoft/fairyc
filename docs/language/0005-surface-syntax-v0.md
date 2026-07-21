# Surface Syntax v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0005-stage-0-surface-and-module-boundary.md`
- Grammar decision: `docs/adr/0009-stage-0-grammar-and-parser-contract.md`
- Normative grammar: `spec/grammar/stage0.ebnf`

## Lexical and declaration rules

- Source files use `.fy` and UTF-8. Stage 0 identifiers are ASCII letters, digits, and underscore and cannot start with a digit.
- Statements end with explicit semicolons. There is no automatic insertion.
- A declaration is written `name: Type`; C declarators and `typedef`-dependent parsing do not exist.
- Every function return type is explicit. No return value is `void`.
- Source-level locals have initializers. `let` prevents reassignment; `var` permits reassignment and mutable borrowing.
- An ordinary internal by-value parameter may use `mut name: Type`. The modifier makes only the
  callee-local place assignable and mutably borrowable; foreign parameters and explicit receivers
  do not use this spelling in Stage 0.
- A `let` resource may still be consumed. Binding immutability is not ownership immobility.
- A by-value position consumes a resource. Stage 0 has no `move` expression or `move` argument keyword.
- Globals are compile-time `const` only. Mutable globals are excluded.
- An ordinary or raw foreign function may declare one closed external-capability set with
  `@effects(clock, console, environment, filesystem, network, process, random)`. Arguments use only
  the effects actually required; omission is the empty set. Exact semantics are owned by
  `0008-effects-and-capabilities-v0.md`.

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

## Types

Stage 0 type syntax includes:

```text
bool, void
i8, u8, i16, u16, i32, u32, i64, u64, isize, usize
c::bool, c::char, c::double, c::float, c::int, c::long, c::longlong, c::schar
c::short, c::size_t, c::ssize_t, c::uchar, c::uint, c::ulong, c::ulonglong, c::ushort
f32, f64
Path::Type
&T, &mut T
*T, *mut T
[T], [T; N]
Option<T>, Result<T, E>
Region
unsafe extern C fn(...) -> T
```

`str` and `CStr` are unsized compiler-known view types and appear as `&str` and `&CStr`. User-defined generic declarations are excluded even though built-in `Option`, `Result`, slices, and arrays accept type arguments.

A safe borrow whose element type is `str` or `[u8]` exposes the read-only property `.len` with
result type `usize`. The property reports the view's byte count, including embedded NUL bytes and
multi-byte UTF-8 encodings. It reads existing view metadata without scanning, allocation, a call, or
a trap. The same views expose `.is_empty` with result type `bool`, true exactly when the retained
byte count is zero and with the same no-scan and no-call guarantees. These properties do not define
capacity, Unicode scalar or grapheme counting, `CStr` metadata, or a public view representation.

An immutable `&[u8]` supports the read-only expression `bytes[index]`. The index has type `usize`;
an unsuffixed integer literal receives that contextual type. The result has type `u8`. The receiver
and index evaluate once from left to right, and an index greater than or equal to `.len` traps with
category `bounds` before any data access.

A mutable fixed scalar-array place supports `values[index] = value` when the receiver is a `var`
local or `mut` by-value parameter. A value of type `&mut [u8]` supports both the same checked `u8`
read and `bytes[index] = value`; the borrow binding itself may remain `let` because the borrow type,
not binding reassignment, grants referent mutation. A shared view, immutable array place, literal,
call result, or other non-place target is rejected. The receiver and `usize` index evaluate once from
left to right, the complete `bounds` check runs before the right-hand side, and only the success path
evaluates the value once and stores it. Mutable slicing, nested projections, resource elements,
compound assignment, and user-defined indexing remain outside Stage 0.

The same immutable view supports end-exclusive `bytes[start..end]`, `bytes[..end]`,
`bytes[start..]`, and `bytes[..]`. Written bounds have type `usize`; an omitted start is zero and an
omitted end is the retained length of the already-evaluated receiver. The receiver and every
written bound evaluate once from left to right. The result is `&[u8]` with the receiver's borrow
origin when `start <= end && end <= bytes.len`; otherwise evaluation traps with `bounds` before
deriving a pointer. This does not admit inclusive bounds, `str` indexing or slicing, indexed
assignment, mutable-view indexing or slicing, or unchecked access.

The first executable fixed-array subset accepts `[T; N]` only when `T` is `bool`, `i32`, `u8`,
`u32`, or `usize` and `1 <= N <= 4096`. The ceiling is a provisional Stage 0 implementation guard
for the initial by-value representation, not a stable language or ABI limit. A nonempty list
`[a, b, ...]` constructs an inert copy value and evaluates its elements exactly once from left to
right. `array[index]` requires `usize`, returns `T`, and traps with `bounds` when the index is at
least `N`. `array.len` evaluates the receiver once and returns exact canonical `N` as `usize`
without a scan, allocation, runtime call, memory read, or trap. Empty and repeated literals remain parser-valid but unsupported until zero-size
representation and expansion-budget contracts exist. Array fields, nesting, resource elements,
mutable indexing, slicing, properties other than `.len`, broad array coercions, and foreign
signatures remain closed. One direct shared borrow of an admitted `[u8; N]` place contextually
becomes `&[u8]` when that view type is already expected, as in
`let bytes: &[u8] = &array` or an internal `&[u8]` call argument. A direct exclusive borrow of a
mutable admitted `[u8; N]` place likewise becomes `&mut [u8]` only where that exact type is expected.
Both views retain the array's origin and exact `N`; other element types, bare values, temporary
sources, inferred conversions, and escaping local views remain closed. The mutable view may cross
internal calls and expose `.len`, but it adds no safe indexing, assignment, slicing, or resizing.
A `var` safe-borrow local may be
reassigned in straight-line, branch, or `while` control flow; reachable branch, header,
`continue`, and `break` origins are joined and returning a multi-origin result is rejected.

Internal function parameters may name an exact `&[T; N]` or `&mut [T; N]` for an admitted fixed
array. This is a safe borrow of the whole array, not a slice, and does not add resizing, mutable
indexing, array fields, foreign-reference parameters, or a public ABI. Contextual `&[u8; N]` to
`&[u8]` and `&mut [u8; N]` to `&mut [u8]` conversion preserve their respective shared or exclusive
loan; neither converts between mutabilities.

Ordinary internal functions may accept `&str` and `&[u8]` parameters. Calls preserve the exact safe
borrow and its caller-proven origin; they do not convert the argument to a raw pointer or transfer
ownership. A safe function may contain an explicit lexical `unsafe` block and expose a checked
result to safe callers, but foreign declarations themselves still reject safe-borrow parameters and
the internal view transport is not a public ABI.

Unsafe code may call exactly `str::from_raw_parts(owner, data, length)` or
`CStr::from_raw_parts(owner, data, storage_length)`. The first argument is a shared safe borrow and
defines the returned view's origin; the second is a raw `u8` or opaque `void` pointer; the third is
`usize`. These are compiler-known operations, not user functions or general constructors. Safe
wrappers may return their result, but raw-pointer-only construction, mutable text views, arbitrary
view element types, and a public raw-parts protocol remain unavailable.

`assert(condition);` is a statement with mandatory parentheses and semicolon. Its condition is
exactly `bool`; true continues and false traps as `assertion-failed`. It remains active in every
build profile and is neither a C macro nor an unsafe optimizer assumption.

The current executable C-scalar source slice recognizes all sixteen C scalars declared by the
selected TargetSpec, preserves every one as an exact identity distinct from coincidentally sized
Fairy scalars, and permits them only as transported foreign-call results, locals, and arguments.
Their literals, operators, comparisons, casts, and raw pointers remain deferred. Exact C-scalar
fields are admitted only in the current nonempty `@repr(C)` struct slice, which also admits acyclic
nested `@repr(C)` records but not arrays or enum fields. Those admitted records may be foreign-call
results by value. Foreign aggregate parameters remain deferred.
`c::size_t` uses the ISO C `<stddef.h>` typedef; `c::ssize_t` is available only for the current
`x86_64-linux-gnu` POSIX.1-2008 profile through `<sys/types.h>`. Plain `c::char` follows the selected
target's signed `char` fact rather than inheriting either `c::schar` or `c::uchar`.
Raw `*void` and `*mut void` are opaque C-interoperability pointers. Inside `unsafe`, a direct local
`&T` may become `*void`, and `&mut T` may become `*void` or `*mut void`; the conversion preserves
the concrete owner type in verified MIR and never strengthens mutability. Exact typed
borrow-to-pointer casts and immutable `&CStr` parameter data extraction use the same explicit unsafe
boundary. An existing safe resource or fixed-array borrow may likewise erase only its referent type
to `void`; it forwards the referent pointer and never takes the address of the borrow carrier. Raw
dereference, arithmetic, integer conversion, and every other pointee-changing cast remain
unavailable.
The executable foreign slice additionally admits one pointer-to-opaque-pointer output shape such as
`*mut *mut void`; both layers must be mutable, and pointer nesting does not generalize to C scalars,
arbitrary pointees, or the other mutability combinations.

## Structs, resources, and enums

An inert struct contains only inert non-borrow fields:

```fy
struct Point {
    x: f32;
    y: f32;
}
```

A resource may own cleanup and defines at most one infallible drop body. Ordinary methods live in an `impl` block:

```fy
resource File {
    handle: c::int;

    drop fn(self: &mut Self) {
        unsafe {
            os_close_best_effort(self.handle);
        }
    }
}

impl File {
    pub fn close(self: Self) -> Result<void, CloseError> {
        // target-specific consuming close
    }
}
```

Receivers are exactly `self: &Self`, `self: &mut Self`, or `self: Self`. Resource fields are private outside the defining module. General field extraction from a resource remains forbidden.

Restricted user enums provide typed inert errors:

```fy
enum IoError {
    NotFound;
    PermissionDenied { os_code: c::int; }
}
```

User-enum payload fields must be inert and copyable. Resource payloads remain special to built-in `Option` and `Result`, whose active-variant move and drop behavior is compiler-defined.
Payloadless construction uses `IoError::NotFound`; named payload construction uses
`IoError::PermissionDenied { os_code: value }`. A shallow user-enum match lists every declared
variant exactly once and binds every named field in the active payload.

## Expressions and control flow

`if` is an expression so source code does not need deferred initialization:

```fy
let file =
    if use_backup {
        fs::File::open(backup)?
    } else {
        fs::File::open(primary)?
    };
```

`Result` construction uses `Result::Ok(expression)` and `Result::Err(expression)` only where one
contextual `Result<T, E>` type is known. A shallow Result match uses exactly one
`Result::Ok(binding)` arm and one `Result::Err(binding)` arm. Restricted user-enum patterns now use
shallow named inert fields and exact exhaustiveness. Wildcards, nested destructuring, guards,
or-patterns, slice patterns, `ref` modifiers, and resource-field extraction are excluded.

Assignment is a statement. Comma expressions, `++`, `--`, C ternary syntax, and integer truthiness are excluded. Ordinary arithmetic traps as defined by the product specification; wrapping, saturating, and checked behavior uses methods rather than additional operator tokens.

The fixed precedence and associativity table is in `docs/language/0006-lexing-parsing-and-recovery-v0.md`. Comparisons do not chain. Parentheses remain explicit group nodes through AST lowering.

`Result` propagation uses postfix `?`. Explicit discard is:

```fy
discard logger.flush();
```

Discarding a built-in tagged value runs drop for its active resource payload.

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
unsafe extern C fn c_read(
    fd: c::int,
    data: *mut u8,
    size: c::size_t,
) -> c::ssize_t;
```

Without the attribute, the Fairy declaration name is also the external C identifier. With it, the
Fairy name remains the source lookup name and the one nonempty cooked string becomes the external
identifier. Missing, non-string, empty, multiple, or duplicate arguments are `E5106`; the backend
still rejects names outside its C11 and private-namespace policy.

There is no parallel `extern C {}` block syntax. C-layout types use `@repr(C)`. Pointer casts and raw
dereferences require unsafe code. Safe integer `as` currently admits exactly `u8` to `i32`, `u32`, or
`usize`, plus `u32` to `usize`; all four preserve every source value on every admitted target.
Narrowing, signedness-changing, C-scalar, pointer/integer, and other numeric conversions remain
rejected and require a later explicit fallible conversion contract.

Binary `+`, `-`, and `*` accept exactly matching `i32`, `u8`, `u32`, or `usize` operands and return
that same type. Their ordinary source meaning is checked: overflow and unsigned underflow trap as
`integer-overflow`; they never inherit C unsigned wrapping. Matching `i32`, `u8`, `u32`, or `usize`
operands also support `/` and `%`; zero traps before the operation and only the signed type has the
minimum-by-negative-one overflow case. Unary `~` accepts any one of those types; the same matching
types support binary `&`, `^`, `|`, `<<`, and `>>`. Bitwise operations preserve the exact type without trapping. Right shift uses arithmetic behavior
for `i32` and logical behavior for unsigned types. Both shifts trap on an amount outside the exact
operand width; left shift additionally traps when the mathematical result is not representable.
Mixed types and implicit promotions remain rejected.

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

Function overloading, default and named arguments, operator overloading, user implicit conversions, C declarators and casts, preprocessing, textual includes, mutable globals, macros, user generics, dynamic dispatch, closures, async, rich patterns, resource-bearing user enums, source-level uninitialized storage, character literals, raw strings, interpolation, octal literals, numeric suffixes, and hexadecimal floats remain outside Stage 0.
