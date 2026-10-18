# Source of Truth - Capabilities, State, and Atlas

Part 3 of 3 of the [source index](source-of-truth.md).

## C transport, aggregates, and native proofs


- **Copy fixed-array Option/Result typing, target-owned private C transport, language `/9` cutover, cache-key invalidation, and native evidence**
  - `internal/semantic/resource_types.go`
  - `internal/cbackend/arrays.go`
  - `internal/cbackend/tagged_types.go`
  - `spec/fixtures/c-emission/cases/pass/copy-tagged-fixed-array/`
  - `spec/fixtures/package-cache/manifest-v5.json`
  - `tools/dev/fairyc-windows-host-smoke/tagged_array_sources.go`
  - `docs/adr/0284-copy-fixed-array-tagged-values.md`
- **Exact explicit signed 32-bit `c::int` and `i32` casts, language `/10` cutover, cache-key invalidation, and native separate-C evidence**
  - `internal/hir/hir.go`
  - `internal/mir/verify.go`
  - `internal/interpreter/interpreter.go`
  - `internal/cbackend/emit.go`
  - `spec/fixtures/semantic/cases/pass/c-int-explicit-cast/`
  - `spec/fixtures/c-emission/cases/pass/c-int-explicit-cast/`
  - `spec/fixtures/package-cache/manifest-v5.json`
  - `tools/dev/fairyc-windows-host-smoke/c_int_cast_probe.go`
  - `docs/adr/0286-explicit-c-int-i32-casts.md`
- **Namespace-safe verbatim C tags, members, and external symbols**
  - `internal/cbackend/c_identifiers.go`
  - `internal/cbackend/aggregate_types.go`
  - `spec/fixtures/c-emission/cases/pass/repr-c-safe-identifiers/`
  - `docs/abi/c-abi-v0.md`
  - `docs/adr/0320-c-abi-identifier-namespace-guards.md`
- **Dedicated Windows native trap control channel and stream-safe classification**
  - `internal/cruntime/`
  - `internal/cdriver/trap_control_windows.go`
  - `internal/cli/workspace_native.go`
  - `tools/dev/fairyc-windows-host-smoke/workspace_execution_proofs.go`
  - `docs/adr/0321-dedicated-native-trap-control-channel.md`
- **Acyclic nested inert-enum private C transport and native Windows proof**
  - `internal/cbackend/aggregate_types.go`
  - `internal/cbackend/emit.go`
  - `spec/fixtures/c-emission/cases/pass/nested-inert-enum-payload/`
  - `tools/dev/fairyc-windows-host-smoke/nested_enum_source.go`
  - `docs/adr/0323-nested-inert-enum-c-transport.md`
- **Scalar Option/Result fields in private aggregates and native Windows proof**
  - `internal/cbackend/aggregate_types.go`
  - `internal/cbackend/emit.go`
  - `spec/fixtures/c-emission/cases/pass/scalar-tagged-aggregate-fields/`
  - `tools/dev/fairyc-windows-host-smoke/scalar_tagged_aggregate_source.go`
  - `docs/adr/0324-scalar-tagged-aggregate-fields.md`
- **Direct copy-scalar Option/Result fields in private aggregates and native Windows proof**
  - `internal/cbackend/tagged_types.go`
  - `internal/cbackend/aggregate_types.go`
  - `spec/fixtures/c-emission/cases/pass/copy-scalar-tagged-aggregate-fields/`
  - `tools/dev/fairyc-windows-host-smoke/copy_scalar_tagged_aggregate_source.go`
  - `docs/adr/0325-copy-scalar-tagged-aggregate-fields.md`
- **Acyclic aggregate-bearing Option/Result field graph and native Windows proof**
  - `internal/cbackend/aggregate_types.go`
  - `internal/cbackend/tagged_types.go`
  - `spec/fixtures/c-emission/cases/pass/aggregate-bearing-tagged-fields/`
  - `tools/dev/fairyc-windows-host-smoke/aggregate_tagged_source.go`
  - `docs/adr/0326-aggregate-bearing-tagged-field-graph.md`
- **Nested compact scalar-tag private C transport and native Windows proof**
  - `internal/cbackend/tagged_types.go`
  - `spec/fixtures/c-emission/cases/pass/nested-compact-tag/`
  - `tools/dev/fairyc-windows-host-smoke/nested_compact_tag_source.go`
  - `docs/adr/0327-nested-compact-tag-transport.md`
- **Nested compact tag fields in private aggregates and native Windows proof**
  - `internal/cbackend/aggregate_types.go`
  - `internal/cbackend/tagged_types.go`
  - `spec/fixtures/c-emission/cases/pass/nested-compact-tag-aggregate-fields/`
  - `tools/dev/fairyc-windows-host-smoke/nested_compact_aggregate_source.go`
  - `docs/adr/0329-nested-compact-tag-aggregate-fields.md`
- **Acyclic generic copy-only tag dependency graph and native Windows proof**
  - `internal/cbackend/aggregate_types.go`
  - `internal/cbackend/tagged_types.go`
  - `spec/fixtures/c-emission/cases/pass/acyclic-copy-tagged-dependency-graph/`
  - `tools/dev/fairyc-windows-host-smoke/copy_tagged_graph_source.go`
  - `docs/adr/0330-acyclic-copy-tagged-dependency-graph.md`

## Language capability admission and releases


- **Production-workload capability candidate gates**
  - `docs/adr/0328-production-workload-capability-admission-gates.md`
  - `docs/product/01-roadmap.md`
  - `docs/product/02-spec.md`
- **Released language-11 resource abilities and linear no-drop analysis with public-interface v4 transport**
  - `internal/semantic/resource_abilities.go`
  - `internal/hir/hir.go`
  - `internal/moduleinterface/`
  - `docs/language/0009-resource-abilities-v0.md`
  - `docs/adr/0342-stable-resource-abilities-and-onchain-effects.md`
- **Released language-11 target-neutral on-chain effect classification**
  - `internal/hir/effect.go`
  - `internal/semantic/effects.go`
  - `docs/language/0008-effects-and-capabilities-v0.md`
  - `docs/adr/0342-stable-resource-abilities-and-onchain-effects.md`
- **Stable language-11 contract capability quickstart**
  - `examples/contract_capabilities_quickstart/`
  - `docs/adr/0342-stable-resource-abilities-and-onchain-effects.md`
- **Released language-12 declaration-owned explicit resource burn and native quickstart**
  - `internal/semantic/resource_abilities.go`
  - `internal/mir/`
  - `internal/interpreter/`
  - `internal/cbackend/`
  - `examples/asset_burn_quickstart/`
  - `docs/language/0009-resource-abilities-v0.md`
  - `docs/mir/drop-elaboration-v0.md`
  - `docs/adr/0343-declaration-owned-explicit-resource-burn.md`

## Contract state


- **Experimental typed contract-state layout, failure-atomic transaction oracle, owner-bound encoded borrow semantics and verified compiler transport
  bounded read-only Merkle commitments, and compiler-published versioned transaction execution fixtures, with declaration-only source layout but no source state access, runtime
  backend, or production persistence admission**
  - `internal/contractstate/`
  - `internal/contractstatefixture/`
  - `internal/compiler/check.go`
  - `spec/fixtures/contract-state/manifest-v4.json` with historical manifest-v0.json through manifest-v3.json coverage
  - `docs/contract-state/transaction-kernel-v0.md`
  - `docs/product/04-contract-profile-candidates.md`
  - `docs/adr/0338-typed-persistent-state-transaction-kernel.md`
  - `docs/adr/0364-versioned-contract-state-transaction-fixtures.md`
  - `docs/adr/0365-versioned-contract-state-map-fixtures.md`
  - `docs/adr/0366-compiler-published-contract-state-fixtures.md`
  - `docs/adr/0374-contract-state-merkle-commitments.md`
  - `docs/adr/0378-data-dependent-contract-state-transaction-graphs.md`
  - `docs/adr/0379-verified-contract-state-helper-calls.md`
  - `docs/adr/0380-owner-bound-state-borrow-oracle.md`
  - `docs/adr/0381-verified-contract-state-borrow-transport.md`
- **Versioned overlapping contract-state snapshot/conflict fixtures plus real-goroutine and race-detector evidence, isolated from compiler IR publication and production persistence**
  - `internal/contractstateconflictfixture/`
  - `internal/contractstate/concurrency_conflict_test.go`
  - `spec/fixtures/contract-state-conflict/manifest-v1.json` with historical manifest-v0.json coverage
  - `docs/adr/0367-versioned-contract-state-conflict-fixtures.md`
  - `docs/adr/0368-versioned-contract-state-map-conflict-fixtures.md`
- **Versioned append-only contract-state layout publication fixtures with open-transaction invalidation and fail-closed breaking-change cases
  isolated from migration tooling and production persistence**
  - `internal/contractstatelayoutfixture/`
  - `spec/fixtures/contract-state-layout/manifest-v0.json`
  - `docs/adr/0371-contract-state-layout-upgrade-fixtures.md`
- **Explicit caller-visible contract-state retry policy and versioned success/exhaustion fixtures, with no automatic kernel retry, public helper, metering, external-effect replay
  or production persistence admission**
  - `internal/contractstate/retry_semantics_test.go`
  - `internal/contractstateretryfixture/`
  - `spec/fixtures/contract-state-retry/manifest-v0.json`
  - `docs/adr/0372-explicit-caller-visible-contract-state-retry.md`
- **Vendor-neutral contract-state persistence port and process-local reference adapter, with no database/provider, native lowering, migration, replication, or deployment admission**
  - `internal/contractstate/persistence.go`
  - `docs/adr/0376-contract-state-persistence-port.md`
- **Language-13 package-owned source state declaration, closed storage codecs, typed-HIR lowering, and existing public-interface/cache identity reuse, with source accesses
  native persistence, and targets still closed**
  - `spec/grammar/stage0.ebnf`
  - `spec/fixtures/parser/manifest-v0.json`
  - `spec/fixtures/semantic/manifest-v0.json`
  - `internal/ast/`
  - `internal/semantic/state.go`
  - `internal/hir/state.go`
  - `examples/contract_state_declaration_quickstart/`
  - `docs/adr/0377-source-package-contract-state-declarations.md`
- **Experimental compiler-private contract-state transaction MIR, opaque capability/effect verifier, represented-storage-only effect profile
  closed-form explicit-entry lifetime-free ordinary-trap-free safe raw-pointer-free current transaction owners, bounded acyclic SSA state blocks
  bounded same-package void helper DAGs, block-confined read-only state borrows, exact package-body identity, cache-key v4 invalidation, v1 legacy, v2 graph, v3 helper
  and v4 borrow payload codecs, checked fresh-owner rebind, stable package-compilation bundle, exact-body persistent cache admission
  combined cached MIR/state fresh-symbol reassembly, unified verified compiler-result publication, and oracle bridge, isolated from source state access, general calls, loops
  raw-pointer exposure, broader native backend execution, public runtime ABI, runtime persistence, and deployable target admission**
  - `internal/contractstatemir/`
  - `internal/contractstate/encoded.go`
  - `internal/hir/state.go`
  - `internal/compiler/check.go`
  - `internal/packageartifact/stable_compilation.go`
  - `internal/packageartifact/stable_cache.go`
  - `internal/packagecache/`
  - `internal/workspacecompiler/`
  - `docs/mir/contract-state-transaction-mir-v0.md`
  - `docs/package/stable-package-compilation-v1.md`
  - `docs/adr/0344-compiler-private-contract-state-transaction-mir.md`
  - `docs/adr/0346-contract-state-mir-capability-and-effect-verification.md`
  - `docs/adr/0347-source-independent-hir-state-access-lowering.md`
  - `docs/adr/0348-exact-contract-state-package-body-identity.md`
  - `docs/adr/0349-contract-state-body-cache-key-v4.md`
  - `docs/adr/0350-bounded-stable-contract-state-body-payload.md`
  - `docs/adr/0351-checked-fresh-owner-contract-state-body-rebind.md`
  - `docs/adr/0352-stable-package-compilation-artifact.md`
  - `docs/adr/0353-stable-compilation-cache-records.md`
  - `docs/adr/0354-combined-cached-state-reassembly.md`
  - `docs/adr/0355-unified-verified-compiler-result.md`
  - `docs/adr/0356-closed-contract-state-effect-profile.md`
  - `docs/adr/0357-call-free-contract-state-owners.md`
  - `docs/adr/0358-straight-line-contract-state-owners.md`
  - `docs/adr/0359-safe-contract-state-owner-memory-boundary.md`
  - `docs/adr/0360-explicit-entry-contract-state-owners.md`
  - `docs/adr/0361-potentially-trapping-owner-operations.md`
  - `docs/adr/0362-resource-lifetime-free-contract-state-owners.md`
  - `docs/adr/0363-closed-form-contract-state-owner-hir.md`
  - `docs/adr/0378-data-dependent-contract-state-transaction-graphs.md`
  - `docs/adr/0379-verified-contract-state-helper-calls.md`
  - `docs/adr/0381-verified-contract-state-borrow-transport.md`
- **Compiler-private single-cell contract-state native handoff, exact layout/body identity, bounded stack-only runtime, and low/optimized Windows MSVC evidence
  without public runtime ABI, maps, graphs, helpers, borrows, durable persistence, or target promotion**
  - `internal/contractstatenative/`
  - `internal/cstdlib/stage0_contract_state.c`
  - `docs/adr/0382-private-single-cell-contract-state-native-handoff.md`
- **Source-declared or compiler-constructed HIR contract-state layout plus public-interface v5 and whole-package cache-key v3 layout identity**
  - `internal/hir/state.go`
  - `internal/contractstatemir/lower.go`
  - `internal/moduleinterface/`
  - `internal/packagecache/`
  - `internal/workspacecompiler/`
  - `docs/adr/0345-hir-contract-state-interface-and-cache-identity.md`
  - `docs/adr/0377-source-package-contract-state-declarations.md`

## Generics, contract IDL, contract profile, and chain primitives


- **Stable language-14 explicit scalar generic workspace functions, coordinate-bound expansion, and separate private generic interface/cache transport**
  - `internal/genericplan/`
  - `internal/genericsource/`
  - `internal/compiler/generic_packages.go`
  - `internal/workspacecompiler/check.go`
  - `docs/adr/0384-explicit-generic-function-source-shapes.md`
  - `docs/adr/0385-stable-explicit-scalar-generics.md`. Nominal generic arguments and source static interfaces remain unadmitted.
- **Experimental chain-neutral contract IDL, schema, and typed Go/TypeScript binding generator, isolated from Stage 0 grammar and the stable `fairyc` CLI**
  - `tools/dev/fairyc-contract-idl/`
  - `internal/contractabi/`
  - `spec/contract-idl/`
  - `examples/contract_idl/`
  - `docs/cli/contract-idl.md`
  - `docs/adr/0340-experimental-chain-neutral-contract-idl.md`
- **Experimental deterministic contract profile, source-only `@max_iter` overlay, verified-MIR authority and target-type rejection, bounded transitive abstract-cost report
  compiler-private meter plan, and explicit contract-state exclusion, with backend, gas, persistent storage, and target selection closed**
  - `tools/dev/fairyc-contract-profile/`
  - `internal/contractprofile/`
  - `spec/cli/experimental-contract-profile-v0.schema.json`
  - `docs/contract/deterministic-contract-profile-v0.md`
  - `docs/adr/0369-experimental-deterministic-contract-profile.md`
- **Compiler-private fixed-width unsigned arithmetic oracle, isolated from source types, MIR, backends, ABI, nominal chain values, cryptography, fixed-point values, gas
  and target selection**
  - `internal/chainprimitives/`
  - `docs/adr/0370-fixed-width-unsigned-arithmetic-oracle.md`

## Contract document families


- **Executable MIR semantics**
  - `docs/mir/*.md`
- **Package identity and visibility**
  - `docs/package/*.md`
- **Target and C ABI**
  - `docs/abi/*.md`
  - `docs/adr/0214-first-hosted-native-ci-receipts.md`
- **Windows target maturity evidence ledger**
  - `docs/abi/windows-target-maturity-v0.md`
  - `tools/dev/fairyc-windows-host-smoke/`
- **Runtime and standard library**
  - `docs/runtime/*.md`
- **Compatibility and versioning**
  - `docs/compatibility/*.md`
- **Diagnostic meanings**
  - `docs/diagnostics/catalog-v0.md`

## Atlas compiler-fact surfaces


- **Atlas map, context, attributed-text, and evidence-bundle contracts**
  - `docs/atlas/*.md`
- **Immutable Atlas compiler-fact snapshot and in-memory declaration-documentation sidecar**
  - `internal/semanticsnapshot/`
  - `docs/atlas/semantic-snapshot-v0.md`
  - `docs/adr/0182-immutable-semantic-snapshot-v0.md`
  - `docs/adr/0200-exact-declaration-documentation-attributed-text-command.md`
- **Provisional deterministic Atlas SymbolId**
  - `internal/symbolidentity/`
  - `docs/atlas/symbol-id-v1.md`
  - `docs/adr/0183-provisional-symbol-id-v1.md`
- **Deterministic per-package Atlas logical map planning**
  - `internal/atlasmap/`
  - `docs/atlas/fymap-logical-plan-v1.md`
  - `docs/adr/0185-fymap-logical-package-plan-v1.md`
- **Bounded canonical `.fymap` transport and checksum**
  - `internal/atlasmap/codec*.go`
  - `docs/atlas/fymap-transport-v1.md`
  - `docs/adr/0186-bounded-canonical-fymap-transport-v1.md`
- **Contained bounded `.fymap` store and current-source validation**
  - `internal/atlasmap/store.go`
  - `internal/atlasmap/current.go`
  - `docs/atlas/fymap-store-v1.md`
  - `docs/adr/0187-contained-fymap-store-and-current-source-validation.md`
- **Bounded exact-current package-local Atlas queries**
  - `internal/atlasmap/query.go`
  - `docs/atlas/fymap-query-v1.md`
  - `docs/adr/0188-bounded-current-fymap-queries.md`
- **Bounded exact-current workspace call join and reverse-call ledger**
  - `internal/atlasmap/workspace_call_join.go`
  - `docs/atlas/workspace-call-join-v0.md`
  - `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- **Canonical workspace call catalog v0 output and experimental CLI**
  - `internal/workspacecallcatalog/`
  - `internal/cli/atlas_calls.go`
  - `docs/atlas/workspace-call-catalog-v0.md`
  - `docs/adr/0226-experimental-workspace-call-catalog-command.md`
- **Exact-current full-SymbolId-ordered catalog query with optional exact declaration-name filtering**
  - `internal/atlasmap/query.go`
  - `docs/adr/0192-exact-current-symbol-catalog-query.md`
  - `docs/adr/0201-exact-name-symbol-catalog-v1.md`
- **Canonical Atlas SymbolId Catalog v1 output and experimental CLI**
  - `internal/symbolcatalog/`
  - `internal/cli/atlas_symbols.go`
  - `docs/atlas/symbol-catalog-v1.md`
  - `docs/adr/0193-experimental-symbol-catalog-command.md`
  - `docs/adr/0201-exact-name-symbol-catalog-v1.md`
- **Canonical compiler-fact-only Atlas Context Pack output**
  - `internal/contextpack/`
  - `docs/atlas/context-pack-v1.md`
  - `docs/adr/0189-canonical-compiler-fact-context-pack-v1.md`
- **Canonical single-symbol Atlas Evidence Bundle v0 composition and experimental CLI**
  - `internal/evidencebundle/`
  - `internal/cli/atlas_evidence.go`
  - `docs/atlas/evidence-bundle-v0.md`
  - `docs/adr/0202-single-symbol-atlas-evidence-bundle-v0.md`
- **Experimental Atlas overview, show, and around Context Pack CLI**
  - `internal/cli/atlas_pack.go`
  - `internal/cli/atlas_pack_relations.go`
  - `docs/adr/0190-experimental-atlas-overview-pack-command.md`
  - `docs/adr/0194-experimental-atlas-show-around-pack-commands.md`
- **Atlas attributed-text policy, frontend-owned exact `///` declaration attachment, immutable snapshot sidecar, exact retained-workspace source/documentation gates
  canonical output-only JSON, and experimental source/documentation CLI**
  - `internal/lexer/documentation.go`
  - `internal/semanticsnapshot/`
  - `internal/attributedtext/`
  - `internal/workspaceloader/exact_source.go`
  - `internal/cli/atlas_text.go`
  - `docs/atlas/attributed-text-v0.md`
  - `docs/adr/0195-attributed-text-boundary-v0.md`
  - `docs/adr/0196-exact-retained-source-excerpt-extraction.md`
  - `docs/adr/0197-canonical-attributed-text-json-v0.md`
  - `docs/adr/0198-experimental-atlas-text-source-command.md`
  - `docs/adr/0199-c-friendly-declaration-documentation-attachment.md`
  - `docs/adr/0200-exact-declaration-documentation-attributed-text-command.md`
- **Atlas whole-package rewrite measurement**
  - `internal/buildmeasure/atlas_rewrite.go`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0191-atlas-whole-package-rewrite-measurement.md`
- **Atlas workspace-call standalone process measurement**
  - `internal/buildmeasure/atlas_call_process.go`
  - `tools/dev/fairyc-windows-host-smoke/atlas_call_process_measure.go`
  - `docs/adr/0227-atlas-call-process-measurement-control.md`
- **Atlas workspace-call multi-shape scaling measurement**
  - `internal/buildmeasure/atlas_call_scale.go`
  - `tools/dev/fairyc-windows-host-smoke/atlas_call_scale_measure.go`
  - `docs/adr/0228-atlas-workspace-call-scaling-control.md`
- **Atlas workspace-call non-star topology correctness corpus**
  - `tools/dev/fairyc-windows-host-smoke/atlas_call_topology_test.go`
  - `docs/adr/0229-atlas-workspace-call-topology-corpus.md`
- **Atlas workspace-call retained-heap control**
  - `internal/buildmeasure/atlas_call_retained.go`
  - `tools/dev/fairyc-windows-host-smoke/atlas_call_retained_measure.go`
  - `docs/adr/0230-atlas-workspace-call-retained-heap-control.md`
- **Executable one-way Atlas package-import boundary and Phase 4 exit evidence**
  - `internal/architecturecontract/`
  - `docs/adr/0203-enforced-one-way-atlas-import-boundary.md`
  - `docs/atlas/phase4-exit-evidence.md`

## Validation, routing, and repository hygiene


- **Validation**
  - `VALIDATION.md`
- **Agent routing**
  - `.agents/context-map.md`
- **Repository hygiene**
  - `.editorconfig, .gitattributes, .gitignore`
