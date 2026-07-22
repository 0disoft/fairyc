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
- Region fixture manifest: spec/fixtures/region/manifest-v1.json
- Result fixture manifest: spec/fixtures/result/manifest-v0.json
- MIR fixture manifest: spec/fixtures/mir/manifest-v0.json
- MIR verifier mutation manifest: spec/fixtures/mir-verifier/manifest-v0.json
- Constrained C emission manifest: spec/fixtures/c-emission/manifest-v0.json
- Shared constrained-C driver: internal/cdriver/
- Embedded scalar C runtime: internal/cruntime/
- Provisional target standard-library C sources: internal/cstdlib/
- Linux scalar C differential runner: internal/cbackendnative/
- Resolved package graph fixture manifest: spec/fixtures/package/manifest-v1.json, including historical v0 cases
- Resolved package graph engine and fixture runner: internal/packagegraph/, internal/packagefixture/
- PackageInstanceId v1 vectors and runner: spec/fixtures/package-identity/manifest-v1.json, internal/packageidentityfixture/
- Public-interface hash v1 fixtures and runner: spec/fixtures/module-interface/manifest-v1.json, internal/moduleinterface/, internal/moduleinterfacefixture/
- Package-source digest v1 fixtures and runner: spec/fixtures/package-source/manifest-v1.json, internal/packagecontent/, internal/packagecontentfixture/
- Whole-package cache-key v1 fixtures and bounded store runner: spec/fixtures/package-cache/manifest-v1.json, internal/packagecache/, internal/packagecachefixture/
- Package-stable function IDs, process-local MIR slice reassembly, process-bound codec, and all-hit reuse coordinator: internal/packageartifact/, internal/workspacecompiler/, docs/package/package-function-id-v1.md, docs/package/process-local-mir-artifact-v0.md
- Bounded workspace check-session CLI and NDJSON schema: internal/cli/workspace_session.go, spec/cli/experimental-workspace-check-session-v0.schema.json, docs/adr/0159-bounded-workspace-check-session.md
- Path-free no-cache and process-local cache measurement schemas: internal/buildmeasure/, docs/adr/0150-no-cache-windows-build-calibration.md, docs/adr/0157-process-cache-measurement-receipt.md, docs/adr/0158-process-cache-edit-distributions.md, docs/adr/0160-editor-shaped-session-process-measurement.md
- Project manifest v1 fixtures and strict TOML parser: spec/fixtures/project-manifest/manifest-v1.json, internal/projectmanifest/, internal/projectmanifestfixture/
- Local workspace loader v1 fixtures and implementation: spec/fixtures/workspace-loader/manifest-v1.json, internal/workspaceloader/, internal/workspaceloaderfixture/
- Compiler-bundled Stage 0 standard-library source package: internal/sdkstdlib/
- Generated lockfile v1 implementation: internal/lockfile/
- Package-aware Windows quickstart: examples/workspace_quickstart/
- Windows MSVC layout fixture manifest: spec/fixtures/layout/manifest-windows-msvc-v0.json
- Windows MSVC C ABI fixture manifest: spec/fixtures/c-abi/manifest-windows-msvc-v0.json
- Executable semantics and C transport: docs/mir/*.md
- Package identity and visibility: docs/package/*.md
- Manifest/lockfile readiness and local-exact recommendation: docs/package/manifest-lockfile-readiness.md
- Human project manifest v1: docs/package/project-manifest-v1.md
- Local workspace loader v1: docs/package/workspace-loader-v1.md
- Generated lockfile v1: docs/package/lockfile-v1.md
- Target and C ABI contract: docs/abi/*.md
- Runtime and standard-library contract: docs/runtime/*.md
- Compatibility and versioning contract: docs/compatibility/*.md
- Planned C-shaped pre-1 source cutover: docs/adr/0136-c-shaped-source-surface.md
- Constrained C11 transport baseline: docs/adr/0137-c11-transport-baseline.md
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

Product scope is set by `docs/product/02-spec.md`. Accepted ADRs constrain that scope. `spec/grammar/stage0.ebnf` owns exact prototype spelling and precedence; parser fixtures own syntax acceptance, semantic fixtures own stable name, type, and borrow assertions, resource fixtures own whole-root state transitions, Region fixtures own capability origins, Result fixtures own handling and propagation, MIR fixtures own source-linked normalization events after verification, MIR verifier fixtures own isolated `Vxxxx` mutation recipes, C-emission fixtures own the bounded verified scalar source-to-C unit facts, and package fixtures own resolved instance and direct-visibility facts. Detailed language, MIR, package, ABI, runtime, compatibility, diagnostics, and Atlas documents refine those decisions. Generated output, compiler caches, benchmark reports, external AI responses, and emitted maps are evidence only and cannot override these sources.
