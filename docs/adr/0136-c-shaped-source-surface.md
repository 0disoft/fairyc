# ADR 0136: Adopt a C-shaped source surface without C declarators

- Status: Accepted for the next pre-1 language revision
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Current surface source: `docs/language/0005-surface-syntax-v0.md`
- Current grammar source: `spec/grammar/stage0.ebnf`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Context

The implemented Stage 0 surface uses `fn name(parameters) -> Result`, `name: Type`, `&mut`,
`impl`, `match`, qualified built-in variants, and `drop fn(self: &mut Self)`. Those spellings are
parseable and already carry useful ownership semantics, but together they present Fairy C as a
small Rust-shaped language rather than as a repaired language for C programmers.

Copying C declarators is not an acceptable remedy. C declarations split one type around its name,
make array and function binding depend on punctuation placement, and can require typedef-aware
parsing. Fairy C still needs a complete type tree that can be parsed without name resolution, one
declared name per declaration, mandatory initialization, no array-to-pointer decay, and no implicit
integer truthiness or conversion.

Several proposed surface changes are independent language features rather than necessary syntax
replacements. Classic and range `for`, lexical `defer`, generic iterator protocols, checked
narrowing conversions, and provider-qualified package imports therefore must not be smuggled into
this cutover without their own semantic and fixture decisions.

## Decision

### Compatibility boundary

1. This ADR defines the target surface of the next pre-1 language revision. It does not change the
   grammar accepted by the current compiler.
2. The cutover is one breaking language revision. The product parser will not accept both the old
   and new spellings, auto-detect a revision, or rewrite source while compiling.
3. The immediately preceding revision may receive a separate lossless-CST migration tool. Its CLI
   spelling remains undecided until a configured command contract and executable implementation
   exist.
4. An implemented cutover increments the language revision and requires at least a compiler minor
   release. This design-only ADR does not itself change a release version, C ABI, runtime ABI, target
   schema, or MIR version.

### Declaration spine

The source surface uses the visual order of C99/C11 while retaining explicit declaration
introducers and complete prefix type trees:

```fy
fn i32 add(i32 left, i32 right) {
    return left + right;
}

let i32 fixed = 42;
var usize cursor = 0;
let inferred = add(20, 22);
const usize PAGE_SIZE = 4096;

struct Pair {
    i32 left;
    i32 right;
}

let Pair pair = Pair {
    .left = 20,
    .right = 22,
};
```

The exact rules are:

- functions begin with `fn`, followed by result type, name, parameters, and body;
- locals begin with `let` or `var`, followed by either one name for inferred typing or one complete
  written type and one name, then an initializer and semicolon;
- `let` forbids binding reassignment but does not forbid moving a resource;
- `var` permits binding reassignment and mutable borrowing under the normal Flow rules;
- `let name = value;` and `var name = value;` are the only inferred forms; `auto` is not a source
  type or declaration marker;
- module constants use `const Type NAME = value;`; `const` does not replace `let`, qualify a binding,
  or participate in C-style specifier combinations;
- fields and parameters use `Type name`; one declaration never declares multiple names;
- record construction uses C99-shaped designators, `Type { .field = value, ... }`; each selected
  field is named explicitly, and `field: value` is not a construction form;
- foreign declarations use `unsafe extern C fn Result name(Parameters);`;
- `type Name = Type;` is the only alias form; `typedef` is absent.

The following remain forbidden:

```text
bare local declarations that start with an unresolved type name
name-splitting declarators such as int *a, b or int (*callback)(int)
postfix array declarations such as int values[16]
multiple declarators, implicit initialization, implicit int, and array decay
```

### Control headers and returns

`if`, `while`, and `switch` require parentheses around their condition or scrutinee. A future
classic `for` proposal must use the same parenthesized control-header shape, but no `for` form is
accepted by this ADR.

```fy
if (ready) {
    start();
}

while (running) {
    update();
}
```

An ordinary function body is not a value-producing block. A non-`void` function returns only
through an explicit `return expression;`; falling through such a function is an error. A `void`
function may use `return;` or fall through. Blocks owned by value-producing `if` branches and
`switch` cases may still end in one unsemicolonized result expression because their syntactic
context already demands a value.

Expression-bodied function shorthand such as `fn i32 add(...) => expression;` is not part of the
cutover. It may be reconsidered later if real source demonstrates enough small leaf functions to
justify a second body form. A parenthesized expression between the signature and a block is never a
return clause or executable epilogue.

### Type and access spelling

Binding reassignment, lifetime and alias guarantees, and access through a handle are separate axes.
The next surface uses these four first-class type constructors:

| Type | Lifetime and alias guarantee | Nullable | Write through this handle |
| --- | --- | --- | --- |
| `ref T` | checked shared borrow | no | no |
| `ref mut T` | checked exclusive borrow | no | yes |
| `ptr T` | none | yes | no |
| `ptr mut T` | none | yes | yes |

Read-only access is the unqualified and shortest form. The explicit `mut` capability grants write
access through a handle; omitting a word can therefore remove authority but never add it. `let` and
`var` independently own binding reassignment, while `const` is reserved for module constants.

Types remain standalone prefix trees:

```fy
ref T
ref mut T
ptr T
ptr mut T
T[N]
T[]
Option<T>
Result<T, E>
fn R(A, B)
unsafe extern C fn R(A, B)
```

`T[]` remains unsized and cannot appear by value. Safe views are `ref T[]` and `ref mut T[]`.
Function values are non-null; nullable callbacks use `Option<fn R(A)>`. Borrow-returning functions
remain ineligible for first-class function conversion until return origin becomes part of the
function type.

The array and slice brackets are part of the complete type and always precede the declared name.
Prefix constructors consume the complete type to their right: `ref u8[]` is a reference to a byte
slice and `ptr u8[16]` is a pointer to a fixed array. Parentheses select the opposite nesting, so
`(ptr u8)[16]` is a fixed array of raw pointers. This binding is syntax-only and never depends on a
typedef or resolved identifier.

Borrow expressions change with their types:

```fy
ref value
ref mut value
```

The first creates a shared borrow and the second an exclusive borrow. Place validity remains a
semantic check. Raw-pointer construction, dereference, arithmetic, and representation casts remain
inside explicit unsafe contracts; this ADR does not create an implicit safe-reference-to-pointer
conversion.

### Functions, associated functions, and methods

`impl` and `Self` are removed. A type-qualified function names its owner with `::`. Its first
parameter is an ordinary, fully typed parameter; the contextual name `this` makes it a method.

```fy
fn Result<File, OpenError> File::open(ref CStr path) {
    // associated function: no receiver
}

fn usize File::len(ref File this) {
    return this.length;
}

fn Result<usize, IoError> File::read(ref mut File this, ref mut u8[] output) {
    // exclusive receiver
}

fn Result<void, CloseError> File::close(File this) {
    // consuming receiver
}
```

The owner qualifier and canonical receiver owner must match. Methods may be declared only in the
owner type's module. Extension methods, receiver-only overloads, implicit raw-pointer dereference,
multi-step autoderef, and mutable borrowing of a temporary are forbidden. Names such as `data`,
`data_mut`, and `into_bytes` must expose materially different ownership behavior instead of relying
on overload resolution.

Method-call syntax may create at most one receiver borrow:

```fy
file.len();       // shared borrow for this call
file.read(out);  // exclusive borrow for this call
file.close();    // value consumption
```

The receiver and remaining arguments evaluate exactly once from left to right. A consuming
receiver is moved before later arguments are evaluated, so a later use of the same value is an
error.

### Resources and cleanup

`resource` remains the declaration that introduces affine ownership. The function-shaped drop hook
is replaced by a non-callable `cleanup` block:

```fy
resource File {
    c::int handle;

    @effects(filesystem)
    cleanup {
        unsafe {
            os::close_best_effort(this.handle);
        }
    }
}
```

The compiler supplies `this` as an exclusive execution view of the fully initialized resource.
`cleanup` has no parameter list, result type, callable symbol, or `?` propagation. It cannot move or
return `this`, recursively invoke cleanup, or report a recoverable failure. Field cleanup follows
the body in reverse declaration order. A resource may omit `cleanup`; it remains move-only but has
no user cleanup body. Observable fallible close remains a separate consuming function returning
`Result`.

### Variants, switch, and Result propagation

`match` is replaced by an exhaustive expression-shaped `switch`. Contextual leading-dot variants
replace qualified built-in constructors and patterns:

```fy
fn Result<i32, Error> increment(Result<i32, Error> input) {
    return switch (input) {
        case .ok(value):  { .ok(value + 1) }
        case .err(error): { .err(error) }
    };
}
```

The construction forms `.ok(value)`, `.err(error)`, `.some(value)`, and `.none` require an expected
type from a declaration, argument, return, case scrutinee, or other typed context. User-enum variants
retain their declared spelling after the leading dot. Global best-fit variant lookup is forbidden.

Every case owns one mandatory block. Cases never fall through and have no switch-targeted `break`.
`default`, whole-value `case _`, guards, or-patterns, and nested patterns are absent from the first
revision. Every declared variant appears exactly once. `_` may discard only a selected payload.

The scrutinee evaluates exactly once. An owned non-Copy scrutinee moves into a hidden owner before
tag inspection; an owned payload moves atomically into its case binding; a borrowed scrutinee
produces a borrow of the selected payload with the same access and origin. Inactive payloads are
never loaded, moved, or dropped. Case-local cleanup and outer Flow joins reuse the existing verified
MIR obligations.

Postfix `?` remains the sole shorthand for propagating `Result<T, E>` into an enclosing
`Result<U, E>` with the exact same error type. It does not invoke conversion protocols, exceptions,
or unwinding. Error conversion continues to require an explicit switch.

### Modules and intentionally separate proposals

`module`, explicit aliased `import`, and `::` qualification remain. `::` owns module and type
qualification; `.` owns runtime member access. Wildcard imports, namespace reopening, textual
includes, and implicit transitive visibility remain forbidden.

Provider-qualified dependency imports are a promising package design, but their exact source
spelling depends on a public manifest and lock contract that does not yet exist. They are not fixed
by this ADR. Classic and range `for`, lexical `defer`, checked narrowing conversions, generic
iterators, and formatter behavior are also separate follow-up decisions rather than cutover
requirements.

## Implementation prerequisites

1. Replace AST type strings and semantic string reparsing with structural prefix, postfix, and
   grouped type syntax plus interned semantic type identities before accepting the new compound
   spellings.
2. Preserve owner-qualified functions, explicit receiver types, variant nodes, switch cases, and
   their spans structurally through AST and typed HIR.
3. Reuse existing ownership, Result, switch, cleanup, and verifier semantics in MIR rather than
   encoding source spelling in backend logic.
4. Freeze the previous lossless grammar for the migration tool, but remove its parse path from the
   product compiler after cutover.
5. Convert syntactically valid sources with the migrator; rewrite intentionally malformed parser
   fixtures against the new recovery contract instead of pretending they are migratable programs.

## Consequences

- C programmers see result types, parameter types, fields, and local types before their names
  without inheriting C declarator ambiguity.
- Safe and raw handles are visibly distinct, read-only access is the shortest form, and access
  permission is orthogonal to binding reassignment.
- `impl`, `Self`, `&mut`, qualified built-in variants, `match`, and function-shaped drop syntax no
  longer dominate the language's visual identity.
- Parenthesized control headers, explicit function returns, postfix array types, inferred locals
  without `auto`, and C99-shaped field designators make the ordinary source silhouette C-shaped
  without reintroducing name-splitting declarators.
- The cutover is intentionally source-incompatible and requires a coordinated grammar, parser,
  fixture, example, documentation, and diagnostic migration.
- No current source example or parser behavior changes until the implementation gate completes.

## Completion gate

The ADR may be marked implemented only when the normative grammar, lexer, CST, structural AST type
tree, semantic type identities, typed HIR, parser and semantic fixtures, all valid `.fy` examples,
diagnostic catalog, migration evidence, and documented compatibility vector agree on the new
revision. The old grammar must be unreachable from the product compile path. The complete Go suite
and every applicable native host lane must pass after the cutover.
