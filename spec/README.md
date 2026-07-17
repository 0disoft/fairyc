# Fairy C Specification Data

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Authority decision: `docs/adr/0002-contract-source-of-truth.md`

This directory contains versioned normative data that future compiler tests can consume directly. It is not generated output and it does not contain a compiler implementation.

## Current contents

- `grammar/stage0.ebnf`: normative Stage 0 prototype grammar;
- `fixtures/parser/schema-v0.json`: parser-manifest schema;
- `fixtures/parser/manifest-v0.json`: parser case index and expectations;
- `fixtures/parser/cases/`: source-root-aware positive and negative mini-packages.
- `fixtures/semantic/schema-v0.json`: resolver, type, and Flow expectation schema;
- `fixtures/semantic/manifest-v0.json`: semantic case index and canonical assertions;
- `fixtures/semantic/cases/`: parser-valid multi-file compile-pass and compile-fail mini-packages.
- `fixtures/resource/schema-v0.json`: whole-root resource-state expectation schema;
- `fixtures/resource/manifest-v0.json`: move, cleanup-obligation, join, and loop case index;
- `fixtures/resource/cases/`: parser-, resolver-, and type-valid resource Flow mini-packages.
- `fixtures/region/schema-v0.json`: Region origin and capability expectation schema;
- `fixtures/region/manifest-v0.json`: parameter-origin and Region restriction cases plus explicit deferrals;
- `fixtures/result/schema-v0.json`: Result handling and propagation expectation schema;
- `fixtures/result/manifest-v0.json`: return, discard, `?`, and must-use cases plus explicit deferrals.
- `fixtures/mir/schema-v0.json`: source-linked MIR normalization expectation schema;
- `fixtures/mir/manifest-v0.json`: ordered evaluation, moves, cleanup, Result branching, calls, and trap-path cases plus explicit deferrals.
- `fixtures/mir-verifier/schema-v0.json`: named valid-seed and one-mutation verifier expectation schema;
- `fixtures/mir-verifier/manifest-v0.json`: one isolated negative recipe for every current `Vxxxx` invariant.
- `fixtures/package/schema-v0.json`: already-resolved package graph expectation schema;
- `fixtures/package/manifest-v0.json`: package-instance equality, direct visibility, module ownership, and explicit standard-library injection cases.

Explanatory language documents may summarize these files but cannot silently change spelling, grouping, module identity, semantic phase ownership, canonical facts, spans, recovery acceptance, or verified MIR observations. A contract change updates the normative data, ADR, diagnostics when applicable, and affected fixtures together.
