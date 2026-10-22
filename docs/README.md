# Documentation

- Status: Draft

## Source of Truth

The topical index of every contract, implementation, and fixture owner lives in the development pages:

- [Source of truth index](development/source-of-truth.md)
- [Language, frontend, and backend](development/source-of-truth-language-and-backend.md)
- [Packages, CLI, and standard library](development/source-of-truth-packages-cli-and-sdk.md)
- [Capabilities, contract state, and Atlas](development/source-of-truth-capabilities-contracts-and-atlas.md)

Supporting pages:

- [Implementation status](development/implementation-status.md) - phase history moved out of the root README
- [Phase status](development/phase-status.md) - current Phase 0–4 exit evidence and remaining work
- [Linux runtime coverage](development/linux-runtime-coverage.md) - implemented APIs, execution evidence, and remaining gaps
- [Phase 2 remaining work](development/phase2-remaining-work.md) - satisfied minimums, execution gaps, capability limits, and the next fixture
- [Linux toolchain](abi/linux-toolchain-v0.md) - prototype version floors, installation, and CI provisioning
- [Prototype usage](development/prototype-usage.md) - manual developer examples moved out of the root README
- [Agent context map](../.agents/context-map.md) - workflow and skill routes

### Additional pointers

These entries name paths that are not yet covered by the index pages above, so they stay here.

- Checked x64 integer conversions: [ADR 0418](adr/0418-sdk-checked-integer-conversions.md), `internal/sdkstdlib/sources/std/num.fy`
- Bounded native runtime object reuse, invalidation and fallback: docs/adr/0428-native-runtime-object-reuse.md, internal/cdriver/runtime_cache.go
- Compiler-published versioned contract-state transaction fixtures: spec/fixtures/contract-state/manifest-v3.json with historical manifest-v0.json through manifest-v2.json coverage, internal/contractstatefixture/, internal/compiler/check.go, docs/adr/0364-versioned-contract-state-transaction-fixtures.md, docs/adr/0365-versioned-contract-state-map-fixtures.md, docs/adr/0366-compiler-published-contract-state-fixtures.md, docs/adr/0378-data-dependent-contract-state-transaction-graphs.md, docs/adr/0379-verified-contract-state-helper-calls.md
- Bounded read-only contract-state commitments and inclusion/absence proofs: internal/contractstate/commitment.go, docs/adr/0374-contract-state-merkle-commitments.md
- Vendor-neutral contract-state persistence port and process-local reference adapter: internal/contractstate/persistence.go, internal/contractstate/persistence_test.go, docs/adr/0376-contract-state-persistence-port.md
- Bounded data-dependent contract-state transaction graphs and v2 stable transport: docs/adr/0378-data-dependent-contract-state-transaction-graphs.md, spec/fixtures/contract-state/manifest-v2.json, internal/contractstatemir/graph_verify.go, internal/contractstatemir/graph_execute.go
- Bounded verified contract-state helper calls and v3 stable transport: docs/adr/0379-verified-contract-state-helper-calls.md, spec/fixtures/contract-state/manifest-v3.json, internal/contractstatemir/helpers.go, internal/contractstatemir/helper_execute.go
- Experimental bounded static generic planner: docs/adr/0339-bounded-static-generic-specialization-kernel.md, internal/genericplan/; no source syntax, HIR substitution, MIR, or backend surface is admitted
- Package-stable function IDs, process-local MIR slice reassembly, process-bound codec, and all-hit reuse coordinator: internal/packageartifact/, internal/workspacecompiler/, docs/package/package-function-id-v1.md, docs/package/process-local-mir-artifact-v0.md
- Package-scoped stable context and current persistent-cache growth staircase: internal/packageartifact/context_projection.go, internal/buildmeasure/persistent_cache_growth.go, docs/adr/0171-package-scoped-stable-context.md
- Source-independent package coordinate prerequisite: internal/packagegraph/coordinate.go, docs/package/package-coordinate-id-v1.md, docs/adr/0172-source-independent-package-coordinate.md
- Stable exact recursive test-directory selection: internal/cli/workspace_tests.go, internal/cli/workspace_tests_test.go, tools/dev/fairyc-windows-host-smoke/, docs/adr/0304-stable-test-directory-selection.md
- Stable local-workspace check v1: internal/cli/workspace.go, docs/adr/0175-stable-workspace-check-v1.md
- Stable local Windows workspace build v1: internal/cli/workspace_native.go, internal/cli/build.go, docs/adr/0176-stable-windows-workspace-build-v1.md
- Stable diagnostic explanation v1: internal/cli/run.go, internal/diagnostic/catalog.go, docs/adr/0178-stable-diagnostic-explain-v1.md
- Stable target maturity report v1: internal/cli/target.go, internal/target/spec.go, docs/adr/0179-stable-target-maturity-report-v1.md
- Private workspace C11 bootstrap transport: internal/cli/workspace_emit.go, internal/cbackend/, docs/adr/0180-stable-workspace-c-emission-v1.md, docs/adr/0335-private-c11-bootstrap-transport.md
- Stable isolated workspace semantic and Windows native tests with text/JSON identity listing and exact selection: internal/workspaceloader/test_sources.go, internal/cli/workspace_tests.go, internal/cli/workspace_native_tests.go, internal/interpreter/, internal/cbackend/, internal/cdriver/, spec/cli/stable-test-list-v1.schema.json, docs/adr/0181-stable-workspace-semantic-test-v1.md, docs/adr/0265-stable-isolated-windows-native-tests.md, docs/adr/0267-stable-exact-test-selection.md, docs/adr/0268-stable-test-identity-listing.md, docs/adr/0273-stable-json-test-identity-listing.md
- Manifest/lockfile readiness and local-exact recommendation: docs/package/manifest-lockfile-readiness.md
- Local workspace loader v1: docs/package/workspace-loader-v1.md
- Planned C-shaped pre-1 source cutover: docs/adr/0136-c-shaped-source-surface.md
- Constrained C11 transport baseline: docs/adr/0137-c11-transport-baseline.md
- Operational standard source: docs/ops/00-operational-contract.md
- Validation source: VALIDATION.md
- Repository hygiene source: .editorconfig, .gitattributes, .gitignore
- CLI command and cross-command stable contract sources: docs/cli/command-contract.md, docs/cli/stable-cli-contract-v0.md
- CLI config source: docs/cli/configuration.md
- CLI output and exit-code source: docs/cli/output-and-exit-codes.md
- Monorepo source: docs/monorepo/README.md

## Authority order

Product scope is set by `docs/product/02-spec.md`. Accepted ADRs constrain that scope. `spec/grammar/stage0.ebnf` owns exact prototype spelling and precedence. Parser fixtures own syntax acceptance, semantic fixtures own stable name, type, and borrow assertions, resource fixtures own whole-root state transitions, Region fixtures own capability origins, Result fixtures own handling and propagation, MIR fixtures own source-linked normalization events after verification, MIR verifier fixtures own isolated `Vxxxx` mutation recipes, C-emission fixtures own the bounded verified scalar source-to-C unit facts, and package fixtures own resolved instance and direct-visibility facts. Detailed language, MIR, package, ABI, runtime, compatibility, diagnostics, and Atlas documents refine those decisions. Generated output, compiler caches, benchmark reports, external AI responses, and emitted maps are evidence only and cannot override these sources.
