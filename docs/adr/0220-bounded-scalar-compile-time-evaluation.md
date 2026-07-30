# ADR 0220: Add bounded scalar compile-time evaluation

- Status: Accepted, not implemented
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Diagnostics source: `docs/diagnostics/catalog-v0.md`
- Depends on: ADR 0008, ADR 0136, ADR 0218, ADR 0219

## Context

The grammar already accepts C-shaped module constants as `const Type NAME = expression;`, but the
current AST boundary does not retain them and the semantic frontend therefore cannot implement or
reliably reject them. That silent loss must be closed before Fairy C claims compile-time functions.

Compile-time evaluation is useful for protocol constants and invariant checks. Unrestricted
evaluation would instead create a second execution environment with hidden
I/O, target dependence, runaway recursion, allocator behavior, and cache invalidation resembling
the C++ template and build-cost problems Fairy C is intended to avoid. The surface should also stay
C-shaped rather than importing Rust's `const fn` spelling or Zig's general `comptime` model.

## Decision

1. The first implementation closes one feature set containing private module constants, private
   compile-time-callable functions, and module-level static assertions. Partial implementation must
   fail closed; a parsed constant may never disappear silently.
2. A module constant keeps the existing spelling `const Type NAME = expression;`. The first subset
   is private and admits only the already executable scalars `bool`, `u8`, `i32`, and `u32`.
   `pub`, inference, every other written integer width, floats, C ABI scalars, `usize`, arrays,
   aggregates, enums, pointers, borrows, resources, `Option`, and `Result` are rejected in this
   revision. Constants are value expressions only; fixed-array lengths remain literal-only.
3. A private ordinary free function becomes compile-time-callable through one zero-argument
   `@const_eval` attribute:

   ```fy
   @const_eval
   fn i32 square(i32 value) {
       return value * value;
   }

   const i32 ANSWER = square(6) + 6;
   static_assert(ANSWER == 42);
   ```

   This reuses the existing attribute boundary and retains the established `fn Result name(...)`
   header. `const fn`, `constexpr`, `consteval`, and general `comptime` become no alternate syntax.
4. An annotated function must be safe, private, module-local, effect-free, non-method, and
   non-foreign. Its parameters and result use the same fixed scalar set as constants. Its body is
   exactly one explicit `return expression;` statement. Bindings, mutation, assignment, loops,
   `defer`, `discard`, runtime `assert`, blocks with side effects, unsafe scopes, Region operations,
   propagation, resource operations, foreign calls, methods, and address-bearing values are closed.
5. Constant expressions admit literals, constant parameters, private module constants, grouping,
   scalar prefix and binary operations, short-circuit boolean operations, conditional expressions,
   current admitted casts among `u8`, `i32`, and `u32`, and calls to other eligible `@const_eval`
   functions. A call may
   also execute normally at runtime; the attribute does not force inlining or change its ABI.
6. Evaluation uses Fairy C's declared-width checked integer semantics. Overflow, invalid signed
   division, division or remainder by zero, invalid shifts, and rejected casts are compile-time
   diagnostics, never host-language wraparound. No target layout, host integer width, clock,
   environment, filesystem, random source, process, network, C compiler, MIR interpreter, or
   generated executable participates.
7. The compiler rejects cycles in the combined constant and const-eval call dependency graph before
   evaluation. It also enforces a deterministic maximum call depth of 32 and 65,536 evaluated
   expression nodes per package. Short-circuited and unselected conditional branches cost their
   controlling expression but are not evaluated. A later increase is a compiler limit change, not
   permission for unbounded source execution.
8. `static_assert(expression);` is a module item with exactly one expression and no custom message.
   `static_assert` is contextual at module-item start rather than a globally reserved lexer keyword,
   so existing non-item identifiers keep their spelling. The expression must evaluate to `bool` and
   must be true. It emits no HIR executable statement, MIR, C declaration, object data, or runtime
   check.
9. `E4301` owns malformed, duplicate, misplaced, public, or otherwise ineligible `@const_eval` and
   module-constant declarations. `E4302` owns an expression that is well typed but not in the
   admitted constant subset. `E4303` owns a compile-time arithmetic, shift, division, or cast trap.
   `E4304` owns dependency cycles and evaluation-budget exhaustion. `E4305` owns a false
   `static_assert`; ordinary type diagnostics still own a non-boolean condition.
10. The frontend owns evaluation after name and type resolution and before Flow or MIR lowering.
    The semantic result retains private constant declaration identity, declared type, canonical
    scalar value, and source span. Constants do not gain declaration-documentation attachment; ADR
    0199 continues to exclude them. Runtime references enter typed HIR as ordinary scalar literals;
    `@const_eval` functions remain ordinary HIR functions. MIR and generated C retain no evaluator
    metadata.
11. Semantic Snapshot v0, SymbolId v1, `.fymap` v1, and current Atlas queries keep their closed five-
    kind declaration vocabulary and do not index module constants. Their owning documentation must
    state this omission explicitly before implementation is called complete; they must not silently
    relabel a constant as a local or function. Adding constant symbols to Atlas requires a separate
    versioned transport decision.
12. Public constants and public `@const_eval` functions are deliberately excluded. Supporting them
    would require a stable cross-package value or evaluator-body transport, public-interface
    invalidation rules, and downstream budget accounting. Public-interface v3 therefore remains
    unchanged in this revision.
13. Implementation advances the exact product language pin from `fairyc-language/4` to
    `fairyc-language/5`. The previous-revision parser already preserves generic `@const_eval`
    syntax and existing module constants, so migration preserves bytes and requires a manual exact-
    pin update. New `static_assert` source has no `/4` equivalent and the migrator must reject it as
    non-previous source rather than invent a rewrite.

## Required evidence

- CST and AST retention for attributes, module constants, and `static_assert`, including malformed
  arity and the current silent-constant-loss regression.
- Name resolution for forward constants, const-eval calls, duplicates, private visibility, and
  deterministic cycle paths.
- Type and evaluator fixtures covering every admitted scalar family, short-circuiting, conditional
  selection, casts, nested calls, forward references, and canonical values.
- Stable `E4301` through `E4305` fixtures for placement, types, excluded operations, traps, cycles,
  depth, step budget, and false assertions without diagnostic cascades.
- Semantic-fact evidence for retained private constant identity and value plus HIR evidence that
  runtime references are folded literals and annotated functions remain ordinary runtime callables.
- Semantic Snapshot and Atlas contract tests proving their five declaration kinds remain unchanged
  and their documented module-constant omission is explicit rather than guessed as completeness.
- MIR, interpreter, and C-emission identity evidence proving assertions disappear and constant
  references become ordinary scalar operands without evaluator metadata.
- `/4` migration and embedded-source checks plus current `/5` manifest, SDK, lock, scaffold,
  example, cache-vector, and documentation pins.
- Build-cost fixtures whose declared node and depth limits complete deterministically without heap,
  file, process, network, or target-toolchain access.

## Consequences

Fairy C gains useful compile-time checks and reusable scalar calculations without creating a second
general-purpose language. The deliberately sharp cost is that the first version cannot publish
compile-time APIs, construct tables or aggregates, loop, recurse, allocate, inspect the target, or
format custom assertion messages. Those exclusions keep evaluation cheap, portable, cache-honest,
and removable before MIR.
