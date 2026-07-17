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

Explanatory language documents may summarize these files but cannot silently change spelling, grouping, module identity, spans, or recovery acceptance. A contract change updates the normative data, ADR, diagnostics, and affected fixtures together.
