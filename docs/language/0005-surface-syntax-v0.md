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
- A `let` resource may still be consumed. Binding immutability is not ownership immobility.
- A by-value position consumes a resource. Stage 0 has no `move` expression or `move` argument keyword.
- Globals are compile-time `const` only. Mutable globals are excluded.

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

The current executable C-scalar source slice recognizes all sixteen C scalars declared by the
selected TargetSpec, preserves every one as an exact identity distinct from coincidentally sized
Fairy scalars, and permits them only as transported foreign-call results, locals, and arguments.
Their literals, operators, comparisons, casts, and raw pointers remain deferred. Exact C-scalar
fields are admitted only in the current nonempty `@repr(C)` struct slice, which also admits acyclic
nested `@repr(C)` records but not arrays or enum fields.
`c::size_t` uses the ISO C `<stddef.h>` typedef; `c::ssize_t` is available only for the current
`x86_64-linux-gnu` POSIX.1-2008 profile through `<sys/types.h>`. Plain `c::char` follows the selected
target's signed `char` fact rather than inheriting either `c::schar` or `c::uchar`.
Raw `*void` and `*mut void` are opaque C-interoperability pointers; safe borrows to `void`,
dereference, arithmetic, integer conversion, and pointee-changing casts remain unavailable.

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

There is no parallel `extern C {}` block syntax. C-layout types use `@repr(C)`. Pointer casts and raw dereferences require unsafe code. `as` is limited to value-preserving conversions; narrowing uses an explicit fallible conversion.

## Grammar and recovery authority

`spec/grammar/stage0.ebnf` is the complete normative prototype grammar. `docs/language/0006-lexing-parsing-and-recovery-v0.md` owns source-byte handling, literal cooking, the Pratt binding-power table, module identity, recovery, and canonical byte spans. `spec/fixtures/parser/manifest-v0.json` owns positive and negative acceptance evidence.

A recovered syntax tree is retained for diagnostics only. It cannot silently legalize a missing semicolon, delimiter, import alias, invalid escape, chained comparison, or module mismatch.

## Explicit exclusions

Function overloading, default and named arguments, operator overloading, user implicit conversions, C declarators and casts, preprocessing, textual includes, mutable globals, macros, user generics, dynamic dispatch, closures, async, rich patterns, resource-bearing user enums, source-level uninitialized storage, character literals, raw strings, interpolation, octal literals, numeric suffixes, and hexadecimal floats remain outside Stage 0.
