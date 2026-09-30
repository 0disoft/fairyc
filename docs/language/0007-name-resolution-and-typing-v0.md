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

Within the type phase, ADR 0220 first validates private scalar constant and `@const_eval`
signatures, rejects dependency cycles, evaluates constants and `static_assert`, then folds constant
uses to typed HIR scalar literals before ordinary Flow. Fixed-array lengths remain literal-only and
therefore do not depend on this evaluator.

A failed phase does not manufacture trustworthy facts for later phases. The implementation may continue locally to improve diagnostics, but errors derived only from a poisoned name or type are cascades and are not independent fixture results.

## Module and lexical names

Each file is one module and each import introduces its explicit alias into that module. Module items and import aliases must be unique within the module declaration scope. Function parameters must be unique, and a lexical block cannot declare the same local name twice in that block.

Private module constants share the module item namespace. Their initializers may refer forward to
other constants or eligible `@const_eval` functions in the same module, but never to an imported,
public, ordinary runtime, method, or foreign callable. The combined dependency graph is acyclic.

Qualified `alias::name` first resolves the alias, then resolves a source-visible item in the imported module. Unqualified names resolve from the innermost lexical block through parameters and enclosing blocks, then the current module. There is no implicit prelude, wildcard import, overload set, or fallback search through imported modules.

When the driver supplies a resolved package graph, import-module lookup follows `docs/package/package-graph-v0.md`: the current package and direct dependencies are visible, transitive dependencies are not, and equal module paths from multiple visible package instances are ambiguous. Public manifest parsing and dependency solving remain outside this frontend phase.

Fixture coordinates identify a declaration by module path, declaration name, declaration kind, and its source-root-relative declaration span. They deliberately exclude build-local symbol IDs and absolute filesystem paths.

## Visibility

An item is visible to another module only when declared `pub`. Private fields of a resource remain
private outside their defining module. Consequently, a resource record literal outside the
declaring module is `E0103` even when the resource type itself is public; external construction goes
through a source-visible ordinary function returning that resource or `Result<T, E>`. This does not
change public inert-struct record literals. `pub` affects source lookup only; it does not imply
native symbol export or ABI stability.

## Canonical types

Fixture types use the source spelling normalized without trivia:

- primitives and nominal paths use names such as `i32`, `bool`, and `math::Point`;
- the compiler-known C scalars keep the sixteen exact canonical identities listed in
  `0005-surface-syntax-v0.md`, distinct from Fairy fixed-width, pointer-width, boolean, and floating
  types and from a user module named `c`;
- references use `ref T` or `ref mut T`;
- raw pointers use `ptr T` or `ptr mut T`;
- arrays and slices use `T[N]` and `T[]`;
- built-ins use `Option<T>` and `Result<T, E>`.

Nominal `struct`, `resource`, and `enum` declarations are not structurally interchangeable. Stage 0 has no user-defined implicit conversion or overload resolution.

## Literal and expression typing

ADR 0413 adds executable `i64` to the scalar sets below. Its contextual literals,
constants, internal parameters/results, aggregate fields, fixed arrays,
references and Option/Result payloads retain all 64 signed bits. Unconstrained
integer literals still default to i32. A direct unary minus in an i64 context
admits magnitude 9223372036854775808 as its minimum value. Widening casts from
u8, i32 and u32 to i64 are explicit and value preserving; usize to i64 is rejected.
Arithmetic is checked, right shift is arithmetic, and shift counts must be
0 through 63. Minimum divided or remaindered by -1 traps. Unary minus of the
minimum traps. Mixed-width binary operations are rejected.

Integer and floating literal text remains exact until an expected type is available. An unconstrained integer defaults to `i32`; an unconstrained float defaults to `f64`. Decimal, hexadecimal, and binary integer values are range-checked after type selection and do not wrap; an out-of-range implemented integer literal is `E1013` before MIR lowering. Direct unary minus may fold the exact magnitude `2147483648` to `i32` minimum in any admitted radix. The positive token is not separately representable, and parentheses or unary plus do not receive that exception. Semantic magnitude comparison is digit-bounded and does not allocate a second input-sized arbitrary-precision integer.

A nonempty fixed-array list receives its element type from an expected `T[N]` when present;
otherwise its first element selects `T` and later elements must match exactly. The resulting length
is the written element count, so an annotation with another `N` is `E1006`. The executable subset
admits only the five copy-scalar element types named by ADR 0110 and currently rejects `N` outside
`1..=4096` as unsupported. That ceiling guards the provisional by-value implementation rather than
defining a stable language or ABI limit. An array index is contextually `usize` and returns the
array element type. The exact `.len` property returns `usize`; no other fixed-array property is in
the executable subset. When `ref u8[]` is the expected type, a direct `ref` of an admitted `u8[N]`
place contextually produces that immutable view. When `ref mut u8[]` is expected, a direct `ref mut`
of a mutable admitted `u8[N]` place contextually produces the exclusive view. Neither rule applies to
a bare array value, another element type, a temporary, or mismatched mutability. Exact whole-array
safe-borrow types such as `ref u8[N]` and `ref mut u8[N]` may also appear on internal parameters;
they remain distinct from the contextual unsized views. The mutable view currently admits internal
transport and `.len`, not safe element mutation, slicing, resizing, foreign references, or public
ABI use.

Language revision `/9` additionally admits one bounded scalar fixed array as a direct payload of
compiler-known `Option` or either payload of compiler-known `Result`. Construction, parameter and
return transfer, exhaustive switching, and compatible Result `?` propagation retain ordinary
fixed-array value-copy semantics. This does not admit nested fixed arrays, arrays of resources or
aggregates, array fields, raw C arrays, public tagged layout, or foreign tagged ABI.

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

The current source C-scalar slice deliberately has no contextual literals. A typed foreign-call
result may be relayed through locals and later matching foreign-call arguments. The exact signed
32-bit `c::int` may also convert explicitly in either direction with `i32`; semantic analysis
admits no other C-scalar cast and never infers this conversion. C-scalar arithmetic, comparison,
boolean normalization, floating-point behavior, and C promotion rules remain unsupported.

Boolean conditions are exact: `if`, `while`, `&&`, `||`, and `!` require `bool`. Integer truthiness
is rejected. Both branches of an `if` expression must have one compatible type. A non-`void`
function uses explicit `return`, whose expression must match the declared return type.

Calls require an exact arity and each argument must satisfy its parameter type. An ordinary
parameter is an immutable callee-local place. Code that needs reassignment first copies or moves it
into an explicit `var` local. Assignment and `ref mut` require a mutable local place and a compatible
value. Inside `unsafe`, a direct borrow may erase its pointee to `ptr void` only without strengthening mutability; verified MIR retains
the original owner type. Other borrow and raw-pointer conversions follow their explicit safety
rules, and `null` never inhabits a safe borrow.

`source_location()` is resolved before ordinary callable lookup and accepts no arguments. Its bare
name, a declaration, parameter, local, import alias, pattern binding, or Region binding named
`source_location` is `E4401`. With valid retained source geometry, typing replaces the call with an
ordinary static `ref str` HIR literal whose bytes encode canonical relative path, one-based line,
one-based UTF-8 byte column, and zero-based identifier start byte. Invalid path or line geometry is
`E4402`; no host-path or empty fallback is synthesized. The intrinsic remains outside
`@const_eval` and creates no new MIR operation or runtime effect.

Before checking bodies, the type phase canonicalizes every ordinary and raw foreign
`@effects(...)` declaration in the resolved module set. A direct call additionally requires the
enclosing function's declared effects to contain the callee's set. These facts are source-order
independent and are exposed through typed HIR and semantic fixtures. The complete vocabulary and
compatibility rules are owned by `0008-effects-and-capabilities-v0.md`.

The compile-time evaluator uses the same checked `bool`, `u8`, `i32`, and `u32` operator meanings
as runtime source. It evaluates only the selected branch of `if`, `&&`, and `||`, limits calls to 32
frames, and limits evaluated expression nodes to 65,536 per package. Overflow, invalid division,
remainder, shift, or cast is `E4303`; excluded expressions are `E4302`; cycles or exhausted budgets
are `E4304`; and a false module assertion is `E4305`. `E4301` owns malformed, public, misplaced, or
ineligible constant declarations and `@const_eval` attributes. Semantic fixtures may assert the
constant's module, name, declaration span, exact scalar type, canonical value, and full source span.

For an assignment to a safe-borrow local, Flow evaluates the right-hand side before
replacing the target origin and retained-loan set. Reachable `if` and `switch` exits join origin
facts; different origins remain a disjunction and cannot be returned as if one branch had won.
Loan conflicts use branch-reachable later uses. `while` additionally computes a header fixed point
from entry, fallthrough, and `continue`, then joins header and `break` exits after the loop. The
admitted classic-`for` header subset cannot borrow or change ownership state, so Flow uses the same
fixed point while resolution keeps the initializer binding scoped through condition, body, and post.

Before Flow applies those rules at call sites, it infers a fixed-point return-origin summary for
each internal borrow-returning function. The summary maps one formal borrow or region parameter, or
one static source, to the corresponding actual call argument. Unknown recursive results remain
fail-closed and summaries never become part of source-visible function types.

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
- `E1013` integer literal does not fit its selected implemented integer type;
- `E4301` malformed or ineligible compile-time declaration;
- `E4302` expression outside the bounded compile-time subset;
- `E4303` checked compile-time arithmetic, division, shift, or cast trap;
- `E4304` compile-time dependency cycle or evaluation-budget exhaustion;
- `E4305` false module-level `static_assert`;
- `E4401` reserved `source_location` name declaration, binding, or non-call use;
- `E4402` unavailable or inconsistent canonical source-location geometry;
- `E7101` malformed, duplicate, empty, or unknown effect declaration;
- `E7102` call requires an effect absent from the enclosing function declaration.

Ownership, borrow, and effect diagnostics retain their `E11xx`, `E21xx`, and `E71xx` meanings from `docs/diagnostics/catalog-v0.md`.

## Deferred rules

Overloading, user generics, type classes or interfaces, implicit numeric promotion, user conversion protocols, trait lookup, extension methods, closure capture, separate type/value namespaces, public manifest-driven dependency resolution, and stable public symbol encoding remain outside this v0 contract.
