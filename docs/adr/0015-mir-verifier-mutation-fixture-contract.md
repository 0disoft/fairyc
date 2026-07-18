# MIR Verifier Mutation Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

The MIR verifier defines twenty-one isolated `Vxxxx` compiler-defect recipes. Source-pass MIR fixtures can prove normalized behavior, but valid source cannot deliberately create a missing resource definition, missing terminator, missing tagged-operation type, uninitialized drop flag, missing or invalid Region reset, incomplete checked-arithmetic edge, unwind successor, or absent unsafe provenance. Serializing arbitrary MIR graphs just to test these failures would accidentally create an internal interchange format before MIR representation exists.

## Decision

1. `spec/fixtures/mir-verifier/manifest-v0.json` owns Stage 0 verifier mutation recipes.
2. The contract uses closed named valid seeds, fixture-local entity labels, exactly one typed mutation, and one expected verifier code.
3. Every seed must verify before mutation. A failing baseline is a harness failure, not a passing negative test.
4. The mutation target and any replacement or check named as a supporting entity must exist, the mutation must change the graph exactly once, and the verifier must return the complete listed code set with no cascade.
5. Entity labels do not expose block numbers, local numbers, source symbols, temporary names, or backend identities.
6. The current twenty-one cases cover every invariant currently listed in `docs/mir/mir-v0.md`, from `V1001` through `V1701`, including missing resource-definition and tagged-operation type identity, three `V1403` Region reset shapes, and two `V1501` checked-operation shapes.
7. The recipe schema is internal test data, not serialized MIR, a plugin ABI, a cache format, or a public compiler API.
8. Arbitrary graphs, multiple mutations, fuzz-corpus encoding, verifier-cascade ordering, crash bundles, and backend-specific post-handoff checks remain deferred.

## Consequences

- Phase 1 can require one negative recipe per verifier invariant without freezing the graph representation.
- A new verifier code requires a seed and typed mutation capable of isolating it, or an explicit schema-versioned extension.
- A verifier crash, extra code, ignored mutation, or invalid baseline fails the harness.
- The current Go runner proves only the listed isolated recipes; it does not prove arbitrary malformed graphs or any backend.

## Revisit conditions

Version the schema when a new invariant cannot be isolated by one mutation, when seed vocabulary changes semantic meaning, when verifier cascades become intentional, or when fuzz minimization needs a richer malformed-graph input. A general serialized MIR format requires a separate decision.
