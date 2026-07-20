# ADR 0116: Stage 0 source effects

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0008-effects-and-capabilities-v0.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Context

The runtime already has raw internal transports for selected file and console operations, but
opening safe public wrappers before source-visible authority exists would make I/O indistinguishable
from an ordinary deterministic call. Conversely, importing a research-sized effect algebra would
delay the small compiler without improving its first File or console API.

The parser and AST already preserve generic item attributes, so effects do not need a keyword or a
second function grammar. The missing contract is a closed vocabulary, canonical representation,
call rule, diagnostics, and an explicit relationship with `unsafe`.

## Decision

1. A callable declares external authority with at most one `@effects(...)` item attribute. Omission
   is the empty set; there is no `pure` keyword.
2. The closed Stage 0 vocabulary is `clock`, `console`, `environment`, `filesystem`, `network`,
   `process`, and `random`. Typed HIR stores the set in lexical order without duplicates.
3. Empty, duplicated, malformed, literal, or unknown declarations are `E7101` in the new `E71xx`
   effects namespace.
4. A resolved direct call is valid only when the enclosing ordinary function declares a superset
   of the callee's effects. `E7102` points at the callee path and relates the callee effect
   declaration. Preparation happens for the complete resolved module set before bodies are typed,
   so source order and import direction do not change the result.
5. Ordinary and `unsafe extern C` declarations use the same effect metadata. Foreign-call unsafe
   provenance remains a separate `E51xx` obligation; an unsafe block neither grants nor hides an
   effect.
6. `main` has no ambient authority. Deliberate over-declaration is accepted, allowing an API to
   preserve a stable upper bound while its implementation changes.
7. `unsafe`, explicit allocation, Region operations, language traps, and resource cleanup are not
   members of this external-capability vocabulary.
8. Function-type effects, inference, polymorphism, handlers, and user-defined effects remain
   deferred. This slice does not make real host I/O executable and does not unlock `fairyc check`.

## Consequences

- Safe File and console work can now name authority before the API and runtime are exposed.
- Callers cannot accidentally launder an effect through an ordinary wrapper.
- Foreign declarations that omit effects remain source-checker claims of an empty set, not proof
  that the external implementation is pure; wrapper authors own that contract.
- Adding a declared effect can invalidate source callers and therefore participates in the language
  compatibility vector.

## Verification

- HIR unit tests cover the closed vocabulary, canonical ordering, ordinary calls, and foreign calls;
- semantic unit tests cover malformed, unknown, duplicate, missing, and unsafe-independent cases;
- the semantic fixture contract grows from 29 to 33 cases with canonical effect facts and exact
  `E7101` and `E7102` spans.
