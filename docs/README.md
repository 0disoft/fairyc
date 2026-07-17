# Documentation

- Status: Draft

## Source of Truth

- Product scope source: docs/product/02-spec.md
- Architecture decisions source: docs/adr/*.md
- Stage 0 language details: docs/language/*.md
- Normative Stage 0 grammar: spec/grammar/stage0.ebnf
- Parser fixture manifest: spec/fixtures/parser/manifest-v0.json
- Semantic fixture manifest: spec/fixtures/semantic/manifest-v0.json
- Resource Flow fixture manifest: spec/fixtures/resource/manifest-v0.json
- Region fixture manifest: spec/fixtures/region/manifest-v0.json
- Result fixture manifest: spec/fixtures/result/manifest-v0.json
- MIR fixture manifest: spec/fixtures/mir/manifest-v0.json
- MIR verifier mutation manifest: spec/fixtures/mir-verifier/manifest-v0.json
- Resolved package graph fixture manifest: spec/fixtures/package/manifest-v0.json
- Executable semantics and C transport: docs/mir/*.md
- Package identity and visibility: docs/package/*.md
- Target and C ABI contract: docs/abi/*.md
- Runtime and standard-library contract: docs/runtime/*.md
- Compatibility and versioning contract: docs/compatibility/*.md
- Diagnostic meanings: docs/diagnostics/catalog-v0.md
- Atlas logical schemas: docs/atlas/*.md
- Operational standard source: docs/ops/00-operational-contract.md
- Validation source: VALIDATION.md
- Agent routing source: .agents/context-map.md
- Repository hygiene source: .editorconfig, .gitattributes, .gitignore
- CLI command contract source: docs/cli/command-contract.md
- CLI config source: docs/cli/configuration.md
- CLI output and exit-code source: docs/cli/output-and-exit-codes.md
- Monorepo source: docs/monorepo/README.md

## Authority order

Product scope is set by `docs/product/02-spec.md`. Accepted ADRs constrain that scope. `spec/grammar/stage0.ebnf` owns exact prototype spelling and precedence; parser fixtures own syntax acceptance, semantic fixtures own stable name, type, and borrow assertions, resource fixtures own whole-root state transitions, Region fixtures own capability origins, Result fixtures own handling and propagation, MIR fixtures own source-linked normalization events after verification, MIR verifier fixtures own isolated `Vxxxx` mutation recipes, and package fixtures own resolved instance and direct-visibility facts. Detailed language, MIR, package, ABI, runtime, compatibility, diagnostics, and Atlas documents refine those decisions. Generated output, compiler caches, benchmark reports, external AI responses, and emitted maps are evidence only and cannot override these sources.
