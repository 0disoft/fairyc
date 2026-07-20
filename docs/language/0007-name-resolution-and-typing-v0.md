# Name Resolution and Typing v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0010-stage-0-semantic-fixture-contract.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Analysis order

An error-free AST enters three ordered frontend phases:

1. `resolve` builds module scopes, import aliases, item declarations, parameter scopes, and lexical local scopes;
2. `type` assigns canonical Fairy C types and validates calls, operators, conditions, assignments, and returns;
3. `flow` validates initialization, ownership, loans, borrow origins, regions, must-use results, and cleanup facts.

A failed phase does not manufacture trustworthy facts for later phases. The implementation may continue locally to improve diagnostics, but errors derived only from a poisoned name or type are cascades and are not independent fixture results.

## Module and lexical names

Each file is one module and each import introduces its explicit alias into that module. Module items and import aliases must be unique within the module declaration scope. Function parameters must be unique, and a lexical block cannot declare the same local name twice in that block.

Qualified `alias::name` first resolves the alias, then resolves a source-visible item in the imported module. Unqualified names resolve from the innermost lexical block through parameters and enclosing blocks, then the current module. There is no implicit prelude, wildcard import, overload set, or fallback search through imported modules.

When the driver supplies a resolved package graph, import-module lookup follows `docs/package/package-graph-v0.md`: the current package and direct dependencies are visible, transitive dependencies are not, and equal module paths from multiple visible package instances are ambiguous. Public manifest parsing and dependency solving remain outside this frontend phase.

Fixture coordinates identify a declaration by module path, declaration name, declaration kind, and its source-root-relative declaration span. They deliberately exclude build-local symbol IDs and absolute filesystem paths.

## Visibility

An item is visible to another module only when declared `pub`. Private fields of a resource remain private outside their defining module. `pub` affects source lookup only; it does not imply native symbol export or ABI stability.

## Canonical types

Fixture types use the source spelling normalized without trivia:

- primitives and nominal paths use names such as `i32`, `bool`, and `math::Point`;
- the compiler-known C scalars keep the sixteen exact canonical identities listed in
  `0005-surface-syntax-v0.md`, distinct from Fairy fixed-width, pointer-width, boolean, and floating
  types and from a user module named `c`;
- references use `&T` or `&mut T`;
- raw pointers use `*T` or `*mut T`;
- arrays and slices use `[T; N]` and `[T]`;
- built-ins use `Option<T>` and `Result<T, E>`.

Nominal `struct`, `resource`, and `enum` declarations are not structurally interchangeable. Stage 0 has no user-defined implicit conversion or overload resolution.

## Literal and expression typing

Integer and floating literal text remains exact until an expected type is available. An unconstrained integer defaults to `i32`; an unconstrained float defaults to `f64`. Decimal, hexadecimal, and binary integer values are range-checked after type selection and do not wrap; an out-of-range implemented integer literal is `E1013` before MIR lowering. Direct unary minus may fold the exact magnitude `2147483648` to `i32` minimum in any admitted radix. The positive token is not separately representable, and parentheses or unary plus do not receive that exception. Semantic magnitude comparison is digit-bounded and does not allocate a second input-sized arbitrary-precision integer.

The implemented safe integer `as` subset is closed to `u8 -> i32`, `u8 -> u32`, `u8 -> usize`, and
`u32 -> usize`. Each target contains the complete mathematical source range, so no value changes or
traps. Other integer pairs are not inferred, truncated, wrapped, or sign-reinterpreted; they remain
unsupported until an explicit fallible conversion contract exists.

Binary `+`, `-`, and `*` require two matching operands from `i32`, `u8`, `u32`, or `usize` and
produce that same type. Each operation has checked ordinary semantics and traps on a result outside
the selected type, including unsigned subtraction below zero. There is no arithmetic-time promotion,
coercion, wrapping, or mixed signedness. `/` and `%` admit the same matching types, return that type,
and trap on a zero divisor. Unary `~` admits one value from the same integer set. Binary `&`, `^`,
`|`, and `>>` admit matching values from that set and return the exact operand type. Bitwise
operations do not trap. Right shift keeps signed arithmetic behavior for `i32`, uses
logical behavior for unsigned types, and traps before execution when the amount is outside the
exact operand width. Left shift accepts the same matching types, checks that width first, and then
traps when the mathematical result is not representable in the exact type.

`==` and `!=` require two matching operands from `bool`, `i32`, `u8`, `u32`, or `usize` and produce
`bool`. `<`, `<=`, `>`, and `>=` require two matching operands from `i32`, `u8`, `u32`, or `usize`
and produce `bool`. Two matching raw-pointer types may use equality for null or identity checks.
There is no comparison-time promotion or coercion. Aggregate, borrow, and C-scalar equality plus
pointer ordering and boolean ordering remain unsupported.

The current source C-scalar slice deliberately has no contextual literals. Its only value origin is
a typed foreign-call result, which may be relayed through locals and later matching foreign-call
arguments. Arithmetic, comparison, conversion, boolean normalization, and floating-point behavior
remain unsupported rather than inheriting Fairy scalar rules or C promotion rules.

Boolean conditions are exact: `if`, `while`, `&&`, `||`, and `!` require `bool`. Integer truthiness is rejected. Both branches of an `if` expression must have one compatible type. A function tail expression or explicit return must match the declared return type.

Calls require an exact arity and each argument must satisfy its parameter type. Assignment requires a mutable place and a value compatible with that place. Borrow and raw-pointer conversions follow their explicit safety rules; `null` never inhabits a safe borrow.

## Borrow facts exposed to fixtures

Semantic pass fixtures may assert:

- `shared` or `mutable` borrow kind;
- hidden origin `Static`, `Param(i)`, `RegionParam(i)`, `Local(name)`, or `RawUnknown`;
- the last source use that ends a local loan when one is known;
- whether the borrow is returned from the function.

These are language facts, not a promise about Flow block numbering or HIR layout. Whole-root overlap, call-bounded mutable reborrows, and single-origin return rules remain owned by `docs/language/0002-borrows-and-return-origins.md`.

## Diagnostic determinism

Semantic diagnostics use zero-based, half-open byte spans over the original source. The primary span names the rejected use or expression. Related spans use stable roles such as `declaration`, `previous-declaration`, `expected-type`, `loan`, `origin`, `candidate-origin`, `conflict`, and `last-use`.

Within one case, diagnostics sort by source path using unsigned UTF-8 byte order, then primary start byte, end byte, and code. Related spans sort by role, source path, start byte, and end byte. Full English prose and rendering layout are not fixture assertions.

## Initial diagnostic rules

- `E0101` unresolved name or path;
- `E0102` duplicate declaration in one owning scope;
- `E0103` access to an item that is not source-visible;
- `E1006` incompatible expected and actual types;
- `E1007` non-boolean condition or logical operand;
- `E1008` assignment target is not a mutable place;
- `E1009` return expression does not match the declared return type;
- `E1010` call arity does not match the declaration.
- `E1013` integer literal does not fit its selected implemented integer type.

Ownership and borrow diagnostics retain their `E11xx` and `E21xx` meanings from `docs/diagnostics/catalog-v0.md`.

## Deferred rules

Overloading, user generics, type classes or interfaces, implicit numeric promotion, user conversion protocols, trait lookup, extension methods, closure capture, separate type/value namespaces, public manifest-driven dependency resolution, and stable public symbol encoding remain outside this v0 contract.
