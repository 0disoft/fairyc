# Source of Truth

- Status: Draft
- Scope: general
- Repository Type: cli-tool
- Addons: monorepo

This is the topical source index for the repository, moved out of `AGENTS.md` so that file
stays short. The authority rules stay in `AGENTS.md` and `docs/README.md`.

Authority does not change by this move. Product scope is set by `docs/product/02-spec.md`,
accepted ADRs constrain that scope, `spec/grammar/stage0.ebnf` owns exact spelling and
precedence, and the fixture manifests named in the detail pages own acceptance. Generated
output, compiler caches, benchmark reports, and emitted maps are evidence only and cannot
override those sources.

## Detail pages

- [Language, Frontend, and Backend](source-of-truth-language-and-backend.md): Product, architecture, and authority; Language contract and admission policy; Frontend and compiler implementation; Acceptance fixtures and runners; C backend, targets, layout, and ABI; Windows object, unwind, and link proofs
- [Packages, CLI, and Standard Library](source-of-truth-packages-cli-and-sdk.md): Package graph, interface identity, and digests; Cache, artifacts, and compilation reuse; Build measurement, manifest, scaffold, and workspace loading; Stable CLI and workspace commands; Standard library and SDK surface
- [Capabilities, State, and Atlas](source-of-truth-capabilities-contracts-and-atlas.md): C transport, aggregates, and native proofs; Language capability admission and releases; Contract state; Generics, contract IDL, contract profile, and chain primitives; Contract document families; Atlas compiler-fact surfaces; Validation, routing, and repository hygiene

- [Additional implementation routes](source-of-truth-additional-routes.md) retained from the agent context map.
- [Design references](design-references.md) moved from the root README.
