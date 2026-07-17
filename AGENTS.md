# AGENTS.md

## Repository Scope

Scope: general

This repository owns product, architecture, ADR, engineering, and operational design scaffolds.

This scaffold does not generate implementation source code; any existing source remains project-owned.

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
- Semantic acceptance fixtures: spec/fixtures/semantic/manifest-v0.json
- Resource Flow fixtures: spec/fixtures/resource/manifest-v0.json
- Executable MIR semantics: docs/mir/*.md
- Target and C ABI: docs/abi/*.md
- Runtime and standard library: docs/runtime/*.md
- Compatibility and versioning: docs/compatibility/*.md
- Diagnostic meanings: docs/diagnostics/catalog-v0.md
- Atlas map and context contracts: docs/atlas/*.md
- Validation: VALIDATION.md
- Agent routing: .agents/context-map.md
- Repository hygiene: .editorconfig, .gitattributes, .gitignore

## Hard Rules

- Do not generate or infer application source code from this scaffold.
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

- Application source scaffolding.
- Runtime infrastructure such as Docker, Kubernetes, Terraform, or framework apps.
- Project-specific credentials or deployment secrets.

## Final Response Requirements

- List executed validations, passed validations, skipped validations, skip reasons, and remaining risk.
- Name any source-of-truth documents changed.
- Call out API, DB, repository hygiene, and runner changes explicitly.
