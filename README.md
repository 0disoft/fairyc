# Fairy C

- Status: Design draft
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

Fairy C is an experimental systems programming language for new native software that needs C-like control, predictable costs, explicit C interoperability, and a substantially safer default resource model without a garbage collector or a heavy runtime.

The project is not trying to beat every hand-tuned C program or make every C ownership graph safe. Its intended advantage is that ordinary Fairy C code carries enough ownership, aliasing, bounds, and effect information for the compiler to reject common failures and generate competitive machine code without relying on build-mode-dependent undefined behavior.

## Current phase

The repository is in the design and Stage 0 planning phase. No compiler implementation, standard library, runtime, package manager, or stable language revision exists yet.

## Design anchors

- Product contract: `docs/product/02-spec.md`
- Compiler and repository boundary: `docs/architecture/00-system-boundary.md`
- Language concepts: `docs/architecture/01-domain-model.md`
- Compilation flow: `docs/architecture/02-runtime-flow.md`
- Quality targets: `docs/architecture/03-quality-attributes.md`
- Initial compiler decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Contract ownership decision: `docs/adr/0002-contract-source-of-truth.md`
- Stage 0 safety boundary: `docs/adr/0003-stage-0-safety-boundary.md`
- Language semantics: `docs/language/`
- Normative Stage 0 grammar: `spec/grammar/stage0.ebnf`
- Parser acceptance fixtures: `spec/fixtures/parser/manifest-v0.json`
- Semantic acceptance fixtures: `spec/fixtures/semantic/manifest-v0.json`
- Resource Flow fixtures: `spec/fixtures/resource/manifest-v0.json`
- Region fixtures: `spec/fixtures/region/manifest-v0.json`
- Result fixtures: `spec/fixtures/result/manifest-v0.json`
- MIR normalization fixtures: `spec/fixtures/mir/manifest-v0.json`
- MIR verifier mutation fixtures: `spec/fixtures/mir-verifier/manifest-v0.json`
- Resolved package graph fixtures: `spec/fixtures/package/manifest-v0.json`
- MIR and C backend contracts: `docs/mir/`
- Package identity and visibility: `docs/package/package-graph-v0.md`
- Target and C ABI: `docs/abi/c-abi-v0.md`
- Runtime and standard library: `docs/runtime/runtime-stdlib-v0.md`
- Compatibility and versioning: `docs/compatibility/versioning-v0.md`
- Diagnostic catalog: `docs/diagnostics/catalog-v0.md`
- Atlas schemas: `docs/atlas/`
- CLI contract: `docs/cli/command-contract.md`
- Workspace ownership: `docs/monorepo/package-ownership.md`

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not committed
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
