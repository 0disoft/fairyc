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
- Stage 0 frontend, verified-MIR coordinator, and MIR implementation: internal/source/, internal/lexer/, internal/syntax/, internal/ast/, internal/frontend/, internal/semantic/, internal/hir/, internal/compiler/, internal/mir/
- Stable workspace check/build/run, diagnostic explanation, and target maturity report plus experimental compiler CLI implementation: cmd/fairyc/, internal/cli/, docs/adr/0175-stable-workspace-check-v1.md, docs/adr/0176-stable-windows-workspace-build-v1.md, docs/adr/0177-stable-windows-workspace-run-v1.md, docs/adr/0178-stable-diagnostic-explain-v1.md, docs/adr/0179-stable-target-maturity-report-v1.md
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
- Shared generated-Fairy/C aggregate-result probe: internal/foreignstructprobe/
- Windows MSVC native layout probe generator: internal/windowslayoutprobe/
- Target C ABI classifiers: internal/cabi/
- C ABI classification fixtures: spec/fixtures/c-abi/manifest-v0.json and spec/fixtures/c-abi/manifest-windows-msvc-v0.json
- C ABI fixture runner: internal/cabifixture/
- Windows MSVC native ABI probe generator: internal/windowsabiprobe/
- Windows x64 native exception and virtual-unwind probe generator: internal/windowsunwindprobe/
- Windows AMD64 PE, COFF object, and x64 unwind structure inspector: internal/pecoff/
- Resolved package graph fixtures: spec/fixtures/package/manifest-v1.json with historical manifest-v0.json coverage
- Resolved package graph validation and lookup: internal/packagegraph/
- Resolved package graph fixture runner: internal/packagefixture/
- PackageInstanceId v1 fixtures: spec/fixtures/package-identity/manifest-v1.json
- PackageInstanceId v1 encoding and fixture runner: internal/packagegraph/identity.go, internal/packageidentityfixture/
- PackageCoordinateId v1 encoding and workspace mapping: internal/packagegraph/coordinate.go, internal/workspaceloader/semantic.go, docs/package/package-coordinate-id-v1.md
- Public-interface hash v2 fixtures: spec/fixtures/module-interface/manifest-v2.json with historical manifest-v1.json coverage
- Public-interface extraction, hashing, and fixture runner: internal/moduleinterface/, internal/moduleinterfacefixture/
- Package-source digest v1 fixtures: spec/fixtures/package-source/manifest-v1.json
- Package-source digest and fixture runner: internal/packagecontent/, internal/packagecontentfixture/
- Whole-package cache-key v2 fixtures: spec/fixtures/package-cache/manifest-v2.json with historical manifest-v1.json coverage
- Whole-package cache key, bounded memory store, and fixture runner: internal/packagecache/, internal/packagecachefixture/
- Exact running-compiler executable identity: internal/buildidentity/, docs/adr/0165-executable-digest-compiler-build-identity.md
- Experimental persistent workspace-cache CLI and schema: internal/cli/workspace_cache.go, spec/cli/experimental-workspace-cache-check-v0.schema.json, docs/adr/0166-experimental-persistent-workspace-cache-command.md
- Bounded immutable disk cache storage and durable publication boundary: internal/packagecache/disk.go, internal/packagecache/durability_*.go, docs/adr/0163-bounded-immutable-disk-cache-store.md, docs/adr/0168-platform-durable-cache-publication.md
- Compiler-private verified whole-program MIR artifact codec: internal/mircodec/, docs/adr/0149-verified-mir-artifact-codec.md
- Compiler-private coordinate-stable function identity: internal/packageartifact/, docs/adr/0174-coordinate-stable-linking-and-cache-key-v2.md
- Process-local package MIR artifact extraction, verification, and reassembly: internal/packageartifact/artifact.go, docs/package/process-local-mir-artifact-v0.md
- Stable package type-context artifact and identity: internal/packageartifact/context.go, docs/adr/0161-stable-type-context-artifact.md
- Stable cross-process package MIR bodies, package-scoped context projection/merge, codec, and checked rebind: internal/packageartifact/body.go, internal/packageartifact/context_projection.go, internal/packageartifact/stable_artifact.go, docs/adr/0171-package-scoped-stable-context.md
- Process-bound package MIR artifact codec: internal/packageartifact/codec.go, docs/adr/0154-process-bound-package-artifact-codec.md
- Bounded process-local package artifact cache: internal/packageartifact/cache.go, internal/packagecache/store.go, docs/adr/0155-bounded-process-package-artifact-cache.md
- Package-aware process-local MIR reuse coordinator: internal/workspacecompiler/, docs/adr/0156-all-hit-process-mir-reuse.md
- Explicit stable disk MIR reuse coordinator: internal/workspacecompiler/, internal/packageartifact/stable_cache.go, docs/adr/0164-verified-stable-disk-mir-reuse.md
- Bounded workspace check-session CLI and protocol schema: internal/cli/workspace_session.go, spec/cli/experimental-workspace-check-session-v0.schema.json, docs/adr/0159-bounded-workspace-check-session.md
- Build-measurement report schemas and configured Windows calibrations: internal/buildmeasure/, cmd/fairyc-windows-host-smoke/, docs/adr/0150-no-cache-windows-build-calibration.md, docs/adr/0157-process-cache-measurement-receipt.md, docs/adr/0158-process-cache-edit-distributions.md, docs/adr/0160-editor-shaped-session-process-measurement.md, docs/adr/0167-paired-persistent-cache-process-measurement.md, docs/adr/0171-package-scoped-stable-context.md
- Human project-manifest v1 parser and fixtures: internal/projectmanifest/, internal/projectmanifestfixture/, spec/fixtures/project-manifest/manifest-v1.json
- Local workspace loader and fixtures: internal/workspaceloader/, internal/workspaceloaderfixture/, spec/fixtures/workspace-loader/manifest-v1.json
- Compiler-bundled Stage 0 standard-library source package: internal/sdkstdlib/
- Generated lockfile v1: internal/lockfile/, docs/package/lockfile-v1.md
- Package-aware Windows workspace build/run CLI and native smoke: internal/cli/workspace_native.go, cmd/fairyc-windows-host-smoke/, examples/workspace_quickstart/, docs/adr/0148-package-aware-windows-build-run.md
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
