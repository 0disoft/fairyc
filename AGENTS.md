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
- Stage 0 language semantics: docs/language/*.md, docs/adr/0213-classic-scalar-for-loop.md, docs/adr/0216-checked-compound-assignment.md, docs/adr/0217-bounded-lexical-defer.md, docs/adr/0219-narrow-must-use-call-contract.md, docs/adr/0220-bounded-scalar-compile-time-evaluation.md, docs/adr/0221-explicit-source-location-intrinsic.md, docs/adr/0222-retain-explicit-factory-construction.md
- Post-Stage 0 language feature admission policy: docs/adr/0218-modern-language-concept-admission-boundary.md, docs/adr/0219-narrow-must-use-call-contract.md, docs/adr/0220-bounded-scalar-compile-time-evaluation.md, docs/adr/0221-explicit-source-location-intrinsic.md, docs/adr/0222-retain-explicit-factory-construction.md, docs/adr/0223-defer-static-interfaces-until-generics.md, docs/adr/0224-defer-unproven-surface-shorthands.md
- Normative Stage 0 grammar: spec/grammar/stage0.ebnf
- Parser acceptance fixtures: spec/fixtures/parser/manifest-v0.json
- Stage 0 frontend, exact `///` declaration-documentation attachment, classic scalar `for`, checked compound assignment, bounded lexical `defer`, narrow `@must_use`, bounded scalar compile-time evaluation, explicit `source_location()`, explicit resource-factory construction, verified-MIR coordinator, and MIR implementation: internal/source/, internal/lexer/, internal/lexer/documentation.go, internal/syntax/, internal/ast/, internal/frontend/, internal/semantic/, internal/hir/, internal/compiler/, internal/mir/, docs/adr/0199-c-friendly-declaration-documentation-attachment.md, docs/adr/0213-classic-scalar-for-loop.md, docs/adr/0216-checked-compound-assignment.md, docs/adr/0217-bounded-lexical-defer.md, docs/adr/0219-narrow-must-use-call-contract.md, docs/adr/0220-bounded-scalar-compile-time-evaluation.md, docs/adr/0221-explicit-source-location-intrinsic.md, docs/adr/0222-retain-explicit-factory-construction.md
- Stable starter creation, workspace check/build/run/test/C emission, diagnostic explanation, and target maturity report plus experimental compiler CLI implementation: cmd/fairyc/, internal/cli/, internal/projectscaffold/, docs/adr/0175-stable-workspace-check-v1.md, docs/adr/0176-stable-windows-workspace-build-v1.md, docs/adr/0177-stable-windows-workspace-run-v1.md, docs/adr/0178-stable-diagnostic-explain-v1.md, docs/adr/0179-stable-target-maturity-report-v1.md, docs/adr/0180-stable-workspace-c-emission-v1.md, docs/adr/0181-stable-workspace-semantic-test-v1.md, docs/adr/0204-stable-starter-project-creation-v1.md, docs/adr/0206-starter-project-guide-v2.md, docs/adr/0207-stable-windows-workspace-run-v2.md, docs/adr/0208-starter-project-guide-v3.md, docs/adr/0212-stable-windows-workspace-run-v3.md
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
- Public-interface hash v3 fixtures: spec/fixtures/module-interface/manifest-v3.json with historical manifest-v1.json and manifest-v2.json coverage
- Public-interface extraction, hashing, and fixture runner: internal/moduleinterface/, internal/moduleinterfacefixture/
- Package-source digest v1 fixtures: spec/fixtures/package-source/manifest-v1.json
- Package-source digest and fixture runner: internal/packagecontent/, internal/packagecontentfixture/
- Source-file content and line-map identities: internal/packagecontent/file_identity.go, docs/package/source-file-identities-v1.md, docs/adr/0184-source-file-integrity-identities-v1.md
- Whole-package cache-key v2 fixtures: spec/fixtures/package-cache/manifest-v5.json with historical manifest-v1.json through manifest-v4.json coverage
- Whole-package cache key, bounded memory store, and fixture runner: internal/packagecache/, internal/packagecachefixture/
- Exact running-compiler executable identity: internal/buildidentity/, docs/adr/0165-executable-digest-compiler-build-identity.md
- Experimental persistent workspace-cache CLI and schema: internal/cli/workspace_cache.go, spec/cli/experimental-workspace-cache-check-v0.schema.json, docs/adr/0166-experimental-persistent-workspace-cache-command.md
- Bounded immutable disk cache and shared opened-root durable publication: internal/packagecache/disk.go, internal/durablefile/, docs/adr/0163-bounded-immutable-disk-cache-store.md, docs/adr/0169-opened-root-cache-containment.md
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
- Build-measurement report schemas and configured Windows calibrations: internal/buildmeasure/, cmd/fairyc-windows-host-smoke/, docs/adr/0150-no-cache-windows-build-calibration.md, docs/adr/0157-process-cache-measurement-receipt.md, docs/adr/0158-process-cache-edit-distributions.md, docs/adr/0160-editor-shaped-session-process-measurement.md, docs/adr/0167-paired-persistent-cache-process-measurement.md, docs/adr/0171-package-scoped-stable-context.md, docs/adr/0191-atlas-whole-package-rewrite-measurement.md
- Human project-manifest v1 parser and fixtures: internal/projectmanifest/, internal/projectmanifestfixture/, spec/fixtures/project-manifest/manifest-v1.json
- Stable exact starter-project creation: internal/projectscaffold/, internal/cli/project_new.go, docs/adr/0204-stable-starter-project-creation-v1.md, docs/adr/0206-starter-project-guide-v2.md, docs/adr/0208-starter-project-guide-v3.md
- Local workspace loader and fixtures: internal/workspaceloader/, internal/workspaceloaderfixture/, spec/fixtures/workspace-loader/manifest-v1.json
- Compiler-bundled Stage 0 standard-library source package and safe console I/O APIs: internal/sdkstdlib/, docs/adr/0205-target-neutral-stage0-stdout-write.md, docs/adr/0209-target-neutral-stage0-stderr-write.md, docs/adr/0210-target-neutral-stage0-byte-output.md, docs/adr/0211-target-neutral-stage0-stdin-read.md, docs/adr/0215-allocation-free-stage0-i32-output.md
- Generated lockfile v1: internal/lockfile/, docs/package/lockfile-v1.md
- Package-aware Windows workspace build/run CLI and native smoke: internal/cli/workspace_native.go, internal/cli/program_output.go, cmd/fairyc-windows-host-smoke/, examples/workspace_quickstart/, examples/std_io_quickstart/, docs/adr/0148-package-aware-windows-build-run.md, docs/adr/0205-target-neutral-stage0-stdout-write.md, docs/adr/0207-stable-windows-workspace-run-v2.md, docs/adr/0212-stable-windows-workspace-run-v3.md
- Executable MIR semantics: docs/mir/*.md
- Package identity and visibility: docs/package/*.md
- Target and C ABI: docs/abi/*.md, docs/adr/0214-first-hosted-native-ci-receipts.md
- Runtime and standard library: docs/runtime/*.md
- Compatibility and versioning: docs/compatibility/*.md
- Diagnostic meanings: docs/diagnostics/catalog-v0.md
- Atlas map, context, attributed-text, and evidence-bundle contracts: docs/atlas/*.md
- Immutable Atlas compiler-fact snapshot and in-memory declaration-documentation sidecar: internal/semanticsnapshot/, docs/atlas/semantic-snapshot-v0.md, docs/adr/0182-immutable-semantic-snapshot-v0.md, docs/adr/0200-exact-declaration-documentation-attributed-text-command.md
- Provisional deterministic Atlas SymbolId: internal/symbolidentity/, docs/atlas/symbol-id-v1.md, docs/adr/0183-provisional-symbol-id-v1.md
- Deterministic per-package Atlas logical map planning: internal/atlasmap/, docs/atlas/fymap-logical-plan-v1.md, docs/adr/0185-fymap-logical-package-plan-v1.md
- Bounded canonical `.fymap` transport and checksum: internal/atlasmap/codec*.go, docs/atlas/fymap-transport-v1.md, docs/adr/0186-bounded-canonical-fymap-transport-v1.md
- Contained bounded `.fymap` store and current-source validation: internal/atlasmap/store.go, internal/atlasmap/current.go, docs/atlas/fymap-store-v1.md, docs/adr/0187-contained-fymap-store-and-current-source-validation.md
- Bounded exact-current package-local Atlas queries: internal/atlasmap/query.go, docs/atlas/fymap-query-v1.md, docs/adr/0188-bounded-current-fymap-queries.md
- Bounded exact-current workspace call join and reverse-call ledger: internal/atlasmap/workspace_call_join.go, docs/atlas/workspace-call-join-v0.md, docs/adr/0225-bounded-exact-current-workspace-call-join.md
- Canonical workspace call catalog v0 output and experimental CLI: internal/workspacecallcatalog/, internal/cli/atlas_calls.go, docs/atlas/workspace-call-catalog-v0.md, docs/adr/0226-experimental-workspace-call-catalog-command.md
- Exact-current full-SymbolId-ordered catalog query with optional exact declaration-name filtering: internal/atlasmap/query.go, docs/adr/0192-exact-current-symbol-catalog-query.md, docs/adr/0201-exact-name-symbol-catalog-v1.md
- Canonical Atlas SymbolId Catalog v1 output and experimental CLI: internal/symbolcatalog/, internal/cli/atlas_symbols.go, docs/atlas/symbol-catalog-v1.md, docs/adr/0193-experimental-symbol-catalog-command.md, docs/adr/0201-exact-name-symbol-catalog-v1.md
- Canonical compiler-fact-only Atlas Context Pack output: internal/contextpack/, docs/atlas/context-pack-v1.md, docs/adr/0189-canonical-compiler-fact-context-pack-v1.md
- Canonical single-symbol Atlas Evidence Bundle v0 composition and experimental CLI: internal/evidencebundle/, internal/cli/atlas_evidence.go, docs/atlas/evidence-bundle-v0.md, docs/adr/0202-single-symbol-atlas-evidence-bundle-v0.md
- Experimental Atlas overview, show, and around Context Pack CLI: internal/cli/atlas_pack.go, internal/cli/atlas_pack_relations.go, docs/adr/0190-experimental-atlas-overview-pack-command.md, docs/adr/0194-experimental-atlas-show-around-pack-commands.md
- Atlas attributed-text policy, frontend-owned exact `///` declaration attachment, immutable snapshot sidecar, exact retained-workspace source/documentation gates, canonical output-only JSON, and experimental source/documentation CLI: internal/lexer/documentation.go, internal/semanticsnapshot/, internal/attributedtext/, internal/workspaceloader/exact_source.go, internal/cli/atlas_text.go, docs/atlas/attributed-text-v0.md, docs/adr/0195-attributed-text-boundary-v0.md, docs/adr/0196-exact-retained-source-excerpt-extraction.md, docs/adr/0197-canonical-attributed-text-json-v0.md, docs/adr/0198-experimental-atlas-text-source-command.md, docs/adr/0199-c-friendly-declaration-documentation-attachment.md, docs/adr/0200-exact-declaration-documentation-attributed-text-command.md
- Atlas whole-package rewrite measurement: internal/buildmeasure/atlas_rewrite.go, cmd/fairyc-windows-host-smoke/, docs/adr/0191-atlas-whole-package-rewrite-measurement.md
- Executable one-way Atlas package-import boundary and Phase 4 exit evidence: internal/architecturecontract/, docs/adr/0203-enforced-one-way-atlas-import-boundary.md, docs/atlas/phase4-exit-evidence.md
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
