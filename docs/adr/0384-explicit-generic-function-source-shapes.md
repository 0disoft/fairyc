# ADR 0384: Retain explicit generic function source shapes

- Status: Accepted frontend preparation and compiler-private local specialization
- Date: 2026-09-05

## Decision

Use C-shaped function declarations with explicit ordered type parameters:

```text
fn T identity<T>(T value) { return value; }
fn i32 main() { return identity<i32>(42); }
```

The parser retains parameter names in a distinct type-parameter node and recognizes a type-argument
list only after a path when its closing angle is followed by a call. Bounded lookahead of at most
256 significant tokens separates this shape from ordinary comparisons. The AST owns ordered
`Function.TypeParameters` and `Expression.TypeArguments`; type arguments are never ordinary value
arguments and must never be erased before specialization.

## Admission sequence

This frontend commit deliberately rejects generic functions in semantic analysis and retains
unsupported specialization expressions. Even an unused type parameter on an otherwise concrete
function cannot bypass the gate. It does not release a new language/SDK version or claim runnable
generics.

Source admission must connect these AST facts to ADR 0339's bounded planner, clone and
substitute concrete types before ordinary HIR publication, check all admitted instantiations, and
reject implicit inference and unsupported ability or recursive expansion. Package-owned identity,
public-interface/cache transport, and native evidence are required before closing issues 26 and 27.

## Compiler-private source execution

`compiler.CheckGenericFunctions` now connects one source module to the existing planner through
`internal/genericsource`. It admits private, unqualified, module-local generic functions with
explicit builtin scalar arguments. Type references are cloned and structurally substituted through
parameters, results, bindings, casts, compound types, blocks, and loop regions. Generic-to-generic
calls become planner dependencies; concrete requests are deduplicated before function generation.
Every generated function then passes the ordinary semantic, typed-HIR, Flow, MIR lowering and
verification pipeline. No runtime dispatch or type-variable MIR representation is introduced.

The planner retains its catalog, arity, cycle, depth, per-declaration and total-expansion limits.
Source declarations carry complete AST digests; dependent specialization IDs change transitively.
Build-local package ownership participates in identity. These identities are deliberately not
stable package/cache identities: public generic declarations, qualified calls, nominal type
arguments, source abilities, inference, and workspace/interface transport remain unadmitted.
Bindings cannot shadow generic functions in this initial private path, and reserved generated names
cannot be supplied by source. Template removal must not hide module declaration collisions.

The ordinary compiler entry points still reject generic source. This private preparation changes
no released language, SDK, runtime ABI, or CLI contract. Stable admission still requires versioned
compile fixtures, diagnostic integration, public-interface/cache invalidation evidence and a
language-version decision. It does not complete issues 26 or 27.

The compiler tests execute the same source through the MIR interpreter and low/optimized Windows
MSVC C transport, expecting 42. They also cover repeated instantiations, transitive calls, input AST
preservation, package-owner separation, template-body invalidation, negative admission and budgets.

## Evidence

- `internal/syntax/generic.go`
- `internal/ast/generic_test.go`
- `internal/compiler/generic_syntax_test.go`
- `internal/compiler/generic_source_test.go`
- `internal/compiler/generic_native_test.go`
- `spec/grammar/stage0.ebnf`
