# AGENTS.md

## Repository Scope

Scope: general

This repository owns the Fairy C product contracts, language and architecture specifications,
acceptance fixtures, and the Stage 0 compiler implementation.

The current implementation is a project-owned Go frontend slice. Scaffold metadata does not own,
regenerate, or override compiler source.

## Repository Shape

- Primary repository type: cli-tool
- Addons: monorepo

- cli-tool: This repository type owns command behavior, arguments, flags, config loading, exit codes, terminal output, JSON output, runtime compatibility, and shell integration contracts.
- monorepo: This repository type owns workspace boundaries, package ownership, dependency policy, and change coordination.


## Source of Truth

- Product scope: docs/product/02-spec.md
- Architecture decisions: docs/adr/*.md
- Stage 0 language semantics: docs/language/*.md
- Normative Stage 0 grammar: spec/grammar/stage0.ebnf
- Parser acceptance fixtures: spec/fixtures/parser/manifest-v0.json
- Stage 0 frontend and MIR implementation: internal/source/, internal/lexer/, internal/syntax/, internal/ast/, internal/frontend/, internal/semantic/, internal/hir/, internal/mir/
- Experimental compiler CLI implementation: cmd/fairyc/, internal/cli/
- Parser fixture runner: internal/parserfixture/
- Semantic acceptance fixtures: spec/fixtures/semantic/manifest-v0.json
- Semantic fixture runner: internal/semanticfixture/
- Resource Flow fixtures: spec/fixtures/resource/manifest-v0.json
- Resource Flow fixture runner: internal/resourcefixture/
- Region fixtures: spec/fixtures/region/manifest-v1.json
- Region fixture runner: internal/regionfixture/
- Result fixtures: spec/fixtures/result/manifest-v0.json
- Result fixture runner: internal/resultfixture/
- MIR normalization fixtures: spec/fixtures/mir/manifest-v0.json
- MIR normalization fixture runner: internal/mirfixture/
- MIR verifier mutation fixtures: spec/fixtures/mir-verifier/manifest-v0.json
- MIR verifier mutation fixture runner: internal/mirverifyfixture/
- MIR interpreter fixtures: spec/fixtures/interpreter/manifest-v0.json
- MIR interpreter and fixture runner: internal/interpreter/, internal/interpreterfixture/
- Constrained C emitter: internal/cbackend/
- Constrained C emission fixtures: spec/fixtures/c-emission/manifest-v0.json
- Constrained C emission fixture runner: internal/cbackendfixture/
- Shared constrained-C driver and scalar runtime: internal/cdriver/, internal/cruntime/
- Provisional target standard-library C sources and native probes: internal/cstdlib/
- Closed standard-library C symbol and component contract: internal/cstdlibcontract/
- Linux scalar C differential runner: internal/cbackendnative/, cmd/fairyc-native-fixtures/
- Executable target description: internal/target/
- Experimental target JSON schema: spec/cli/experimental-target-spec-v0.schema.json
- Target layout engine: internal/layout/
- Target layout fixtures: spec/fixtures/layout/manifest-v0.json and spec/fixtures/layout/manifest-windows-msvc-v0.json
- Target layout fixture runner: internal/layoutfixture/
- Shared Windows MSVC C probe type emitter: internal/msvccprobe/
- Windows MSVC native layout probe generator: internal/windowslayoutprobe/
- Target C ABI classifiers: internal/cabi/
- C ABI classification fixtures: spec/fixtures/c-abi/manifest-v0.json and spec/fixtures/c-abi/manifest-windows-msvc-v0.json
- C ABI fixture runner: internal/cabifixture/
- Windows MSVC native ABI probe generator: internal/windowsabiprobe/
- Windows AMD64 PE, COFF object, and x64 unwind structure inspector: internal/pecoff/
- Resolved package graph fixtures: spec/fixtures/package/manifest-v0.json
- Executable MIR semantics: docs/mir/*.md
- Package identity and visibility: docs/package/*.md
- Target and C ABI: docs/abi/*.md
- Runtime and standard library: docs/runtime/*.md
- Compatibility and versioning: docs/compatibility/*.md
- Diagnostic meanings: docs/diagnostics/catalog-v0.md
- Atlas map and context contracts: docs/atlas/*.md
- Validation: VALIDATION.md
- Agent routing: .agents/context-map.md
- Repository hygiene: .editorconfig, .gitattributes, .gitignore

## Hard Rules

- Do not infer language behavior from scaffold prose or implementation accidents. Compiler source
  must implement the owning grammar, language, diagnostic, and fixture contracts.
- Do not invent technology choices. Use UNDECIDED when a decision is not known.
- Do not create fake credentials, tokens, secrets, or private values.
- Do not rely on generated, cache, or build output as source truth.

## Repository Hygiene

- .editorconfig sets line ending, encoding, and final newline policy.
- .gitattributes sets Git text normalization and binary diff policy.
- .gitignore excludes local, secret, build, and cache artifacts.
- Generated, cache, and build output must not be used as design-document evidence.
- Do not create large diffs that only change line endings.

## Before Editing

- Read this file, VALIDATION.md, CHECKLIST.md, and .agents/context-map.md.
- Read the skill and checklist named by the context map.
- Confirm source-of-truth documents before changing contracts.

## Out of Scope

- Unrelated application or framework scaffolding.
- Runtime infrastructure such as Docker, Kubernetes, Terraform, or framework apps.
- Project-specific credentials or deployment secrets.

## Final Response Requirements

- List executed validations, passed validations, skipped validations, skip reasons, and remaining risk.
- Name any source-of-truth documents changed.
- Call out API, DB, repository hygiene, and runner changes explicitly.
