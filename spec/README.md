# Fairy C Specification Data

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Authority decision: `docs/adr/0002-contract-source-of-truth.md`

This directory contains versioned normative data that future compiler tests can consume directly. It is not generated output and it does not contain a compiler implementation.

## Current contents

- `grammar/stage0.ebnf`: normative Stage 0 prototype grammar;
- `cli/experimental-parse-v0.schema.json`: machine contract for the implemented experimental parse report;
- `fixtures/parser/schema-v0.json`: parser-manifest schema;
- `fixtures/parser/manifest-v0.json`: parser case index and expectations;
- `fixtures/parser/cases/`: source-root-aware positive and negative mini-packages.
- `fixtures/semantic/schema-v0.json`: resolver, type, and Flow expectation schema;
- `fixtures/semantic/manifest-v0.json`: semantic case index and canonical assertions;
- `fixtures/semantic/cases/`: parser-valid multi-file compile-pass and compile-fail mini-packages.
- `fixtures/resource/schema-v0.json`: whole-root resource-state expectation schema;
- `fixtures/resource/manifest-v0.json`: move, cleanup-obligation, join, and loop case index;
- `fixtures/resource/cases/`: parser-, resolver-, and type-valid resource Flow mini-packages.
- `fixtures/region/schema-v0.json` and `manifest-v0.json`: historical parameter-origin Region contract;
- `fixtures/region/schema-v1.json`: Region expectations with an optional resolved-package-graph reference;
- `fixtures/region/manifest-v1.json`: current parameter and local origins, `E3101` escape, and Region restriction cases;
- `fixtures/result/schema-v0.json`: Result handling and propagation expectation schema;
- `fixtures/result/manifest-v0.json`: return, discard, `?`, and must-use cases plus explicit deferrals.
- `fixtures/mir/schema-v0.json`: source-linked MIR normalization expectation schema;
- `fixtures/mir/manifest-v0.json`: ordered evaluation, moves, cleanup, Result branching, calls, and trap-path cases plus explicit deferrals.
- `fixtures/mir-verifier/schema-v0.json`: named valid-seed and one-mutation verifier expectation schema;
- `fixtures/mir-verifier/manifest-v0.json`: one isolated negative recipe for every current `Vxxxx` invariant.
- `fixtures/package/schema-v0.json`: already-resolved package graph expectation schema;
- `fixtures/package/manifest-v1.json`: historical v0 package identity/visibility cases plus post-deduplication dependency-cycle rejection.
- `fixtures/package-identity/manifest-v1.json`: canonical resolved-instance bytes, full SHA-256 IDs, component boundaries, and identity validation cases.
- `fixtures/module-interface/manifest-v3.json`: current coordinate-based source-visible interface digest with public must-use invalidation and private-module-constant stability; historical v1 and v2 remain frozen.
- `fixtures/package-source/manifest-v1.json`: exact canonical path/source-byte digest and invalidation cases.
- `fixtures/package-cache/manifest-v4.json`: current coordinate-based dependency compatibility key with fyif3 and exact language `/5` inputs; historical v1 through v3 remain frozen.
- `fixtures/project-manifest/manifest-v1.json`: complete strict `fairy.toml` local-exact parser cases.
- `fixtures/workspace-loader/manifest-v1.json`: complete local workspace trees for filesystem-safe graph construction.

Explanatory language documents may summarize these files but cannot silently change spelling, grouping, module identity, semantic phase ownership, canonical facts, spans, recovery acceptance, or verified MIR observations. A contract change updates the normative data, ADR, diagnostics when applicable, and affected fixtures together.
