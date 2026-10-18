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

- Authority order and the document contract families: `docs/README.md`
- Topical index of every contract, implementation, and fixture owner: `docs/development/source-of-truth.md`
- Product scope: `docs/product/02-spec.md`; accepted ADRs constrain that scope
- Normative Stage 0 grammar: `spec/grammar/stage0.ebnf`; fixtures own acceptance
- Validation names: `VALIDATION.md`
- Agent routing: `.agents/context-map.md`

The topical index was moved out of this file so it stays readable. It is the same content, one path
per line, and it carries no authority by itself: authority stays with the documents it names.
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
