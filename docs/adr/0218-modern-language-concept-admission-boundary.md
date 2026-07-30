# ADR 0218: Bound adoption of modern language concepts

- Status: Accepted
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Roadmap source: `docs/product/01-roadmap.md`
- Depends on: ADR 0001, ADR 0005, ADR 0136, ADR 0217

## Context

Modern C++, Java after 17, and Go contain useful answers to recurring programming problems, but
their complete object, template, runtime, reflection, concurrency, and compatibility models do not
fit Fairy C. Copying a feature because another language standardized it would gradually turn the
project into a second C++ or a runtime-dependent Java subset and would undermine readable costs,
fast builds, constrained generated C, and the C-shaped source surface.

Fairy C already adopts several problem-level ideas without importing their source-language
machinery: `Result<T, E>` instead of exceptions, bounded views instead of pointer-only ranges,
move-only resources and lexical cleanup, exhaustive tagged switching, explicit receivers, modules,
and bounded lexical `defer`. Future borrowing needs one stable admission rule and an explicit queue;
otherwise a roadmap mention can be mistaken for accepted syntax or implemented behavior.

## Decision

1. A borrowed concept is eligible only when all of these conditions hold:
   - it removes a demonstrated class of unsafe behavior or repeated user code;
   - safe source exposes allocation, blocking, cleanup, control-flow, and ownership costs;
   - it has deterministic diagnostics and a bounded implementation and build-cost model;
   - it lowers without requiring exceptions, garbage collection, reflection, a scheduler, or an
     unstable C++ ABI;
   - its C ABI and constrained-C transport boundary can remain explicit;
   - a library solution is materially worse than the language or compiler feature;
   - positive, negative, semantic, MIR, backend, and compatibility evidence can be named before
     implementation.
2. Admission has three distinct states: `candidate`, `accepted`, and `implemented`. A candidate has
   no grammar, keyword, diagnostic, compatibility, or availability promise. Acceptance requires a
   separate ADR with exact syntax and semantics. Implementation requires the owning fixtures and
   validation evidence. Product documentation must not collapse these states.
3. The first recommended candidates, in order, are:
   - a narrow `@must_use` contract for explicitly annotated inert return types or functions,
     extending the existing mandatory handling of `Result` and resources without making every
     return value mandatory;
   - bounded compile-time evaluation consisting of a deliberately restricted `const fn` subset
     and `static_assert`, with pure admitted operations, deterministic step and recursion limits,
     no heap, I/O, raw-pointer access, foreign call, resource, or target-dependent host execution;
   - one caller-source-location compiler intrinsic for diagnostics and logging, without opening
     general default arguments or runtime stack inspection;
   - continued construction through ordinary explicit functions returning `T` or `Result<T, E>`,
     with validation before ownership is published and no special constructor, copy-constructor,
     conversion-constructor, or implicit initialization protocol.
4. A static interface or constraint system is a later evidence-gated candidate only after
   user-defined generics have demonstrated real duplication that ordinary functions cannot remove.
   Its first admissible form must favor static dispatch, explicit implementation, bounded
   monomorphization, deterministic name resolution, and no inheritance. Dynamic dispatch requires
   a separate object-layout, lifetime, ABI, and allocation decision.
5. The following remain non-goals unless a later ADR first overturns this boundary with measured
   need and a complete cost model: class inheritance, implicit virtual dispatch, exceptions and
   stack unwinding, user-defined implicit conversions, function or operator overloading, unrestricted
   template metaprogramming, runtime reflection, annotation-driven framework behavior, a universal
   nullable reference, language-owned async/coroutines/virtual threads, hidden executors, and a
   garbage-collected object root.
6. Design-pattern names never justify a language feature by themselves. Construction uses ordinary
   functions, cleanup uses resources or bounded defer, closed visitors use exhaustive `switch`,
   dependency injection uses explicit parameters and imports, and strategy uses an explicit
   function value plus context when that function-value contract exists.
7. This decision changes no Stage 0 grammar or implementation. It reserves no spelling, does not
   increment `fairyc-language/3`, and does not authorize implementation without a feature-specific
   ADR and fixtures.

## External concept references

These are design inputs rather than authorities over Fairy C:

- OpenJDK JEP 441 and JEP 440 describe exhaustive pattern switching and record deconstruction.
- OpenJDK JEP 444 demonstrates that virtual threads depend on a substantial runtime scheduler and
  therefore do not fit the core-language boundary.
- WG21 P0323 describes `std::expected`; Fairy C already owns its separate `Result` semantics.
- WG21 P0847 describes explicit object parameters; Fairy C keeps explicit typed receivers without
  C++ overload resolution or value-category deduction.
- WG21 P1208 describes caller source location; Fairy C considers only a bounded intrinsic.
- WG21 P2131 summarizes concepts, modules, and span in C++20; Fairy C adopts explicit modules and
  bounded views while deferring generic constraints.

## Consequences

The project has a controlled way to learn from newer languages without inheriting their complete
complexity. The deliberate cost is that attractive features remain candidates until their failure
modes, compile-time budget, lowering, diagnostics, and compatibility cut are executable. This keeps
Stage 0 small and makes later expansion evidence-driven rather than fashion-driven.
