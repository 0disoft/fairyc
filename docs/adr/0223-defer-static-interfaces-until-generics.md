# ADR 0223: Defer static interfaces until generic evidence exists

- Status: Accepted deferral
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Evidence scope: compiler-bundled SDK and checked-in examples at language revision `/6`
- Depends on: ADR 0001, ADR 0005, ADR 0218

## Context

Static interfaces, C++ concepts, Java interfaces, Go interfaces, and Rust traits solve overlapping
but materially different problems. A Fairy C feature chosen before its actual generic call sites
exist would prematurely decide method lookup, implementation coherence, constraint satisfaction,
code generation, diagnostics, public-interface encoding, and cross-package compatibility. It would
also create pressure for dynamic values, implicit conformance, default methods, associated types,
inheritance, specialization, or runtime tables that the language does not otherwise need.

The audited non-fixture corpus contains 29 Fairy C files under `internal/sdkstdlib/` and `examples/`
and 82 function declarations. Sixteen repeated declarations are independent `i32 main()` entry
points. After excluding those, exact normalized declaration shapes repeat only for two small
`consume(Token)` resource demonstrations; they use the same nominal type and do not represent one
algorithm duplicated across different production types. The corpus therefore contains no measured
generic algorithm whose valid operations need an interface constraint.

The grammar permits type arguments on a type use so compiler-known `Option<T>` and `Result<T, E>`
can be written, but it has no type-parameter declaration, generic function, generic aggregate,
instantiation budget, or user-defined generic lowering. An interface admitted now would have no
user-generic consumer and no representative code-growth workload.

## Decision

1. Fairy C does not admit static-interface, concept, trait, protocol, or dynamic-interface syntax
   at language revision `/6`. No keyword, attribute, declaration form, diagnostic namespace, HIR or
   MIR node, runtime table, ABI rule, or compatibility promise is reserved.
2. User-defined generics must receive a separate accepted and implemented contract before a static
   constraint system can return to the admission queue. Built-in `Option` and `Result` type
   applications do not satisfy that prerequisite.
3. Reopening the candidate requires checked-in evidence of all of the following:
   - at least three independent non-fixture owner modules duplicate one algorithm over different
     nominal types;
   - ordinary functions, modules, generated private helpers, or a concrete shared type cannot
     remove the duplication without losing type safety or exposing representation;
   - the implemented user-generic subset has deterministic instantiation identity, recursion and
     expansion limits, package-boundary invalidation, and measured clean-build and output-size data;
   - the candidate names its positive, ambiguity, missing-operation, conflicting-implementation,
     ownership, effect, MIR, C-backend, cache, and compatibility evidence before syntax is accepted.
4. The first admissible proposal must be static-only and explicitly implemented. It must not make
   interface values, boxes, existential storage, virtual tables, reflection objects, hidden
   allocation, or runtime type tests.
5. Implicit structural conformance, inheritance, default method bodies, associated types, blanket
   implementations, specialization, negative implementations, interface composition, and dynamic
   dispatch remain outside that first proposal. Each would need separate demonstrated demand and a
   separate decision even after static constraints exist.
6. Until the gate reopens, concrete ordinary functions and explicit modules remain the solution for
   reusable code. Closed variant behavior remains an exhaustive `switch`; resource policy remains
   an explicit owner function with an explicit receiver or value parameter.

## Consequences

The current compiler avoids a second callable-resolution system and keeps language revision `/6`,
public-interface v3, MIR v0, C11 transport, cache keys, runtime ABI, and target schemas unchanged.
The cost is deliberate duplication if a real generic algorithm appears before the prerequisite is
implemented. That duplication is useful evidence: it can be measured and reviewed instead of being
replaced by a speculative object model.

This decision closes the current post-Stage 0 modern-concept queue. It does not reject user-defined
generics forever; it requires generics to establish their own bounded value before constraints are
designed around them.
