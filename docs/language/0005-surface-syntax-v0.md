# Surface Syntax v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0005-stage-0-surface-and-module-boundary.md`

## Lexical and declaration rules

- Source encoding is UTF-8. Stage 0 identifiers are ASCII letters, digits, and underscore and cannot start with a digit.
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

The module path must match the package-relative file mapping defined by the future package manifest. Every import has an alias and names remain qualified through that alias. Wildcard imports, selective imports, re-exports, implicit prelude imports, and import cycles are excluded.

`pub` grants source-module visibility only. It does not export a stable binary symbol, layout, mangling, or calling convention.

## Types

Stage 0 type syntax includes:

```text
bool, void
i8, u8, i16, u16, i32, u32, i64, u64, isize, usize
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

`match` supports shallow built-in or user-enum variants, a single bound payload, shallow named inert fields, and `_`. Nested destructuring, guards, or-patterns, slice patterns, `ref` modifiers, and resource-field extraction are excluded.

Assignment is a statement. Comma expressions, `++`, `--`, C ternary syntax, and integer truthiness are excluded. Ordinary arithmetic traps as defined by the product specification; wrapping, saturating, and checked behavior uses methods rather than additional operator tokens.

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

There is no parallel `extern C {}` block syntax. C-layout types use `@repr(C)`. Pointer casts and raw dereferences require unsafe code. `as` is limited to value-preserving conversions; narrowing uses an explicit fallible conversion.

## Compact grammar skeleton

```ebnf
source      ::= module_decl import_decl* item* EOF ;
module_decl ::= "module" path ";" ;
import_decl ::= "import" path "as" IDENT ";" ;
path        ::= IDENT ("::" IDENT)* ;

item ::= attribute* "pub"?
         (const_decl | struct_decl | resource_decl | enum_decl |
          impl_decl | function_decl) ;

binding ::= ("let" | "var") IDENT (":" type)? "=" expression ";" ;
function_decl ::= "unsafe"? ("extern" "C")? "fn" IDENT
                  "(" parameters? ")" "->" type (block | ";") ;

type ::= primitive | path type_args? | "&" "mut"? type |
         "*" "mut"? type | "[" type "]" |
         "[" type ";" const_expression "]" |
         "unsafe"? "extern" "C" "fn" "(" type_list? ")" "->" type ;

statement ::= binding | assignment | return_stmt | discard_stmt |
              while_stmt | break_stmt | continue_stmt | region_stmt |
              expression_stmt ;
expression ::= if_expression | match_expression | logical_expression ;
```

Exact operator precedence, literal escapes, package-to-file mapping, and recovery productions must be fixed with parser fixtures before implementation. They cannot introduce a second spelling for an accepted Stage 0 operation.

## Explicit exclusions

Function overloading, default and named arguments, operator overloading, user implicit conversions, C declarators and casts, preprocessing, textual includes, mutable globals, macros, user generics, dynamic dispatch, closures, async, rich patterns, resource-bearing user enums, and source-level uninitialized storage remain outside Stage 0.
