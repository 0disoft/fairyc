# Source of Truth - Capabilities, State, and Atlas

Part 3 of 3 of the [source index](source-of-truth.md).

## Bounded Atlas tools

- Fairy declared-effects completeness: every `project_atlas_produce::Package` declaration is built through `project_atlas_schema::declaration_named`; its function/foreign/template `effects` and resource `drop_effects` justify the explicit collected-effects path in `project_atlas_header.fy`. Legacy header factories retain unavailable effects. Original Go source expectations and v1/v2/v3/FFI/generic/cleanup controls: `internal/cli/candidate_atlas_effect_completeness_test.go`, `candidate_atlas_source_version_test.go`; implementation and Windows evidence: `docs/development/candidate-atlas-effects-20261106.md`. Effects remain declaration metadata, not added executable relation counts or proof of external/template-body behavior.

- Complete-bound borrow map publication, original spans/kinds/escape flags, relation counts and observation-dependent completeness: `examples/lexer_candidate/src/project_atlas_calls.fy`, `project_atlas_produce.fy`, `project_atlas_header.fy`; actual impact/MCP/edit query producer: `project_atlas_source.fy`. Original Go summary/source/version expectations: `internal/cli/candidate_atlas_source_version_test.go`, `candidate_atlas_generic_produce_test.go`, `candidate_atlas_sdk_vector_produce_test.go`; explicit absent/observed completeness: `candidate_atlas_borrow_completeness_test.go`. Actual pack expectations retain the original Borrow arrays in `candidate_pack_command_test.go` and `candidate_pack_source_command_test.go`. Windows verification and remaining scope: `docs/development/candidate-atlas-borrow-production-20261106.md`.

- Complete-owned borrow observations and exact linker alias remapping: `examples/lexer_candidate/src/module_borrow_observations.fy`, `module_project_borrow_observations.fy`, `module_identified_ir.fy`, `module_project_identified_ir.fy`; current-source unit/project artifact reconstruction and independent work accounting: `module_artifact_borrow_observations.fy`, `module_project_artifact_coverage.fy`, `module_project_artifact_admission.fy`, `module_unit_artifact_admission.fy`. Native original-Go ownership/restore/capacity comparisons: `internal/cli/candidate_atlas_borrow_observations_test.go`. Exact scope and outstanding map/command work: `docs/development/candidate-atlas-borrow-observations-20261106.md`.

- Checked-body semantic borrow retention: `examples/lexer_candidate/src/module_unit_borrow_sites.fy`, `module_unit_checked_bodies.fy`; private single/batch collector: `project_atlas_borrow_sites.fy`. Independent Go source/codec controls: `internal/cli/atlas_borrow_sites_oracle_test.go`; ordinary/generic/concrete-cleanup native comparisons and work/count boundaries: `candidate_atlas_borrow_sites_test.go`. Exact Windows scope and required Complete-bound production/cache/command work: `docs/development/candidate-atlas-borrow-sites-20261106.md`. Collection does not promote map completeness.

- Complete-bound Fairy Result-propagation production: `examples/lexer_candidate/src/project_atlas_calls.fy`, `project_atlas_header.fy`, `project_atlas_produce.fy`. Ordinary/generic/cleanup summaries retain original question-token spans and counts; only `returns-error` becomes newly exact. Independent source inventories and current-source/version/join comparisons: `internal/cli/atlas_return_error_production_sources_test.go`, `candidate_atlas_source_version_test.go`, `candidate_atlas_generic_produce_test.go`, `candidate_atlas_sdk_vector_produce_test.go`. Actual pack expectations preserve original Go sites: `candidate_pack_command_test.go`, `candidate_pack_source_command_test.go`. Both-profile Windows scope, hashes, commands and remaining gates: `docs/development/candidate-atlas-return-error-sites-20261106.md`.

- Fairy verified CFG Result-propagation collector: `examples/lexer_candidate/src/project_atlas_return_error_sites.fy`; exact original Go question spans, ordinary/concrete function and cleanup owners and work/count controls: `internal/cli/atlas_return_error_sites_oracle_test.go`, `candidate_atlas_return_error_sites_test.go`. Private collector contract and subsequent separate Complete-bound map/count/completeness/CLI production evidence: `docs/development/candidate-atlas-return-error-sites-20261106.md`. A collector alone does not promote completeness.

- Complete-bound Fairy allocation production: `examples/lexer_candidate/src/project_atlas_calls.fy`, `project_atlas_header.fy`, `project_atlas_produce.fy`. Original Region call spans survive ordinary, generic and concrete cleanup summaries and relation counts; only `allocates` becomes newly exact. Independent source/version inventories: `internal/cli/atlas_allocation_production_sources_test.go`, `candidate_atlas_source_version_test.go`; original-map comparisons and scoped pack expectations: `candidate_atlas_generic_produce_test.go`, `candidate_atlas_sdk_vector_produce_test.go`, `candidate_pack_command_test.go`. Five actual SDK pack source families: `candidate_pack_source_command_test.go`. Exact validation and remaining collector/CLI/final gates: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Fairy verified CFG allocation-site collector: `examples/lexer_candidate/src/project_atlas_allocation_sites.fy`; original Go call-span oracle and native source/owner/frame/work/count comparisons: `internal/cli/atlas_allocation_sites_oracle_test.go`, `candidate_atlas_allocation_sites_test.go`. Byte placements, terminal operands, partial argument order, generic/cleanup owners and heap exclusions remain independently checked. Exact Windows scope, current-bootstrap renewal and separate Complete-bound map/CLI work: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Go source-call canonicalization across mutually exclusive deferred exits: `internal/semanticsnapshot/build.go`; exact ordinary/cleanup/generic source and completeness inventories: `internal/cli/atlas_deferred_result_calls_test.go`, `atlas_return_error_sites_oracle_test.go`. Owning source-site multiplicity: `docs/atlas/semantic-snapshot-v0.md`. Expanded fourteen-case propagation prerequisite, strict codec evidence and remaining Fairy work: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Original Go Result-propagation prerequisite and terminal operands: `internal/cli/atlas_return_error_sites_oracle_test.go`, `internal/mir/lower_results.go`, `internal/mir/lower_tagged.go`. Native Windows original-trap/early-return replay: `internal/cli/result_propagation_operand_exit_native_test.go`; private MSVC Result-tag snapshot: `internal/cbackend/result_switch.go`. Original question-token spans, failure accounting, explicit volatile cost and separate deferred-call/Fairy collector gaps: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Fairy Region zeroed-byte allocation and byte copying: `examples/lexer_candidate/src/module_unit_region_builtin.fy`, `module_unit_value_region.fy`, `module_unit_region_events.fy`, `module_unit_c_region.fy` and `module_unit_interpreter_values.fy`. Checked lexical Region capabilities retain mutable byte-view origins and the existing runtime ABI. Windows interpreter/emitted-C execution and same-spelled ordinary methods: `internal/cli/candidate_region_bytes_test.go`; exact evidence and limits: `docs/development/candidate-region-bytes-20261106.md`. This does not implement allocation-map production or promote native Linux support.

- Original Go Region allocation-site prerequisite and terminated-operand lowering: `internal/cli/atlas_allocation_sites_oracle_test.go`, `internal/mir/lower_regions.go`; actual three-operation Windows low/optimized trap replay: `internal/cli/region_allocation_operand_trap_native_test.go`. Complete call spans, byte operations, partial argument order and heap exclusions are independent of future Fairy output. Exact source/profile limits and separate Fairy collector/production gates: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Complete-bound Fairy move production and completed-call reachability: `examples/lexer_candidate/src/project_atlas_calls.fy`, `project_atlas_reachability.fy`, `project_atlas_header.fy`, `project_atlas_produce.fy`. Original Go source operands/calls/counts/current-join comparisons: `internal/cli/candidate_atlas_source_version_test.go`, `candidate_atlas_generic_produce_test.go`, `candidate_atlas_sdk_vector_produce_test.go`. Command expectations retain original move spans in `candidate_pack_command_test.go` and source move cases in `candidate_pack_source_command_test.go`. Selected Windows results and separate full-CLI/remaining collector gates: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Fairy verified CFG move-site collector: `examples/lexer_candidate/src/project_atlas_move_sites.fy`; original Go operand oracle and expanded native comparison: `internal/cli/atlas_move_sites_oracle_test.go`, `candidate_atlas_move_sites_test.go`. Binding/call/return/tag/scope consumers retain executable reachability, canonical source spans and isolated frame/work/count policies. Named implicit borrowed owner receivers use the same detached verification as native calls and are not move consumers. Collector evidence and separate Complete-bound production/CLI work: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Named implicit owner receiver borrows: `examples/lexer_candidate/src/module_unit_receiver_borrow.fy`, `module_unit_body_ir_receiver.fy`, argument/move/origin checking, C emission and interpreter owners. Independent original Go admission/origin/move/borrow and source-version comparisons plus native unit/whole-project cache replay: `internal/semantic/implicit_method_borrow_test.go`, `internal/cli/candidate_implicit_receiver_test.go`, `candidate_implicit_receiver_cache_test.go`, `candidate_owner_qualified_complete_test.go`, `candidate_atlas_borrow_sites_test.go`, `candidate_atlas_borrow_observations_test.go`, `candidate_atlas_source_version_test.go`, `candidate_pack_source_command_test.go`. Private body-IR profile `/2` isolates the admission change; exact Windows evidence, failed attempts and remaining scope: `docs/development/candidate-implicit-receiver-20261106.md`.

- Original Go Atlas move-site prerequisite: `internal/cli/atlas_move_sites_oracle_test.go`; fourteen source/codec controls retain exact operand spans, partial-call argument order and the distinct discard/propagation behavior. Collector design consequence and Windows Go evidence: `docs/development/candidate-atlas-observed-relations-20261106.md`. The collector and Complete-bound production records distinguish their own Windows scope from remaining full-CLI and source-family work.

- Complete-bound Fairy unsafe map production and relation accounting: `examples/lexer_candidate/src/project_atlas_calls.fy`, `project_atlas_header.fy`, `project_atlas_produce.fy`; independent original Go sites/counts and current-source comparisons: `internal/cli/candidate_atlas_source_version_test.go`, `candidate_atlas_generic_produce_test.go`, `candidate_atlas_sdk_vector_produce_test.go`. Source-backed command expectations: `candidate_pack_command_test.go`, `candidate_pack_source_command_test.go`, `candidate_catalog_command_test.go`, `candidate_atlas_families_test.go`; actual authored-main selections and exact remaining scope: `docs/ops/windows-candidate-atlas-unsafe-cli-20261106.md`. Generation remains partial and uncollected families remain unavailable; the separate collector record and complete queue retain authority over their own gates.

- Private decoded JSON key index: `examples/lexer_candidate/src/project_json_tree.fy`; sorted per-object token-index segments preserve original tokens/source and decoded duplicate spans. Native ordering/escaping/work controls: `internal/cli/candidate_json_keys_test.go`; existing Go/Fairy source-plan regression: `candidate_source_plan_test.go`. Work/space bounds and Windows evidence: `docs/development/candidate-json-key-index-20261106.md`. No work-limit or wire-schema change.

- Fairy verified CFG unsafe-site collector: `examples/lexer_candidate/src/project_atlas_unsafe_sites.fy`; independent ordinary/specialized/cleanup Go facts, single/batch agreement and native work/count controls: `internal/cli/candidate_atlas_unsafe_sites_test.go`. Verified foreign targets distinguish actual foreign calls from safe calls in unsafe scopes. Source spans, raw views and SDK storage operations remain compiler-observed. Collector and remaining production scope: `docs/development/candidate-atlas-observed-relations-20261106.md`. This private collector does not promote unsafe map/header/CLI coverage.

- Fairy verified CFG trap-category collector: `examples/lexer_candidate/src/project_atlas_traps.fy`, pure category helpers in `project_atlas_trap_kinds.fy` and shared drop/trap proof/traversal owner `project_atlas_drop_sites.fy`; independent Go source/owner comparisons, single/batch/combined agreement and native work/count controls: `internal/cli/candidate_atlas_traps_test.go`, `candidate_atlas_drop_sites_test.go`. Six sorted unique categories preserve scalar arithmetic, assertions, bounds and implicit iteration checks. Standalone count policies remain isolated. Complete-bound `project_atlas_calls.fy` production writes categories into ordinary/generic/cleanup summaries; `project_atlas_produce.fy` counts them and `project_atlas_header.fy` marks only may-trap newly exact. Independent source/current/join comparisons: `internal/cli/candidate_atlas_source_version_test.go`. Exact Windows collector/map evidence and remaining broader work: `docs/development/candidate-atlas-observed-relations-20261106.md`; all six selected actual authored-main groups pass both profiles: `docs/ops/windows-candidate-atlas-traps-cli-20261106.md`.

- Fairy verified CFG drop-site collector: `examples/lexer_candidate/src/project_atlas_drop_sites.fy`; independent ordinary/specialized/cleanup Go source comparisons and native bounds: `internal/cli/candidate_atlas_drop_sites_test.go`. Reachable cleanup operands retain exact source spans, canonical deduplication and shared work accounting. Complete-bound v1/v2/v3 production writes ordinary/generic/cleanup drops, counts them separately from calls and marks drops exact. Owned batched CFG collection proves the whole immutable IR once per package-summary build and shares traversal with trap categories. Windows evidence and remaining relation work: `docs/development/candidate-atlas-observed-relations-20261106.md`.

- Fairy source-backed Context Pack commands: `examples/lexer_candidate/src/project_pack_source.fy`, `project_pack_cli.fy`, `project_experimental_cli.fy`. Fresh Complete-bound requested-family production precedes unique UID/current admission and overview/show/around selection, canonical encoding and final stdout. Arguments/help precede I/O; locked requests preserve bytes and unlocked requests refresh the ordinary lock. Independent original Go facts, explicit call/drop/trap production scope and full authored-main native comparisons: `internal/cli/candidate_pack_command_test.go`, `candidate_pack_source_command_test.go`, `candidate_atlas_cli_test.go`. Exact Windows evidence, digests and remaining broader/full-CLI/Linux scope: `docs/development/candidate-context-pack-20261106.md`, `candidate-atlas-observed-relations-20261106.md`.

- Fairy exact-current around traversal and canonical packs: `examples/lexer_candidate/src/project_pack_around.fy`, `project_pack_query.fy`, `project_pack.fy`; independent actual-source Go encoder/checksum/native controls: `internal/cli/candidate_pack_around_test.go`. Canonical BFS/frontier order, cycles, convergent paths, relation-before-node refusal and exact independent budgets/omissions retain the query contract. Fresh source-backed actual CLI and final acceptance remain separate; evidence: `docs/development/candidate-context-pack-20261106.md`.

- Fairy exact-current show query and canonical packs: `examples/lexer_candidate/src/project_pack_query.fy`, `project_pack_context.fy`, `project_pack_graph.fy`, `project_pack_frontier.fy`, `project_pack.fy`; independent Go encoder/checksum/native controls: `internal/cli/candidate_pack_show_test.go`. Mandatory facts, adjacency merge, all directions and exact budget/omission rules are preserved. Around and actual CLI remain separate; evidence: `docs/development/candidate-context-pack-20261106.md`.

- Fairy exact-current relation projection and canonical ordering: `examples/lexer_candidate/src/project_pack_relations.fy`, `project_pack_relation_order.fy`; independent actual Go queries and all-kind source/native controls: `internal/cli/candidate_pack_relations_test.go`. Concrete endpoints, cleanup, spans/details and ABI/escape facts remain intact; bounded show/around selection/traversal is a separate next step. Evidence: `docs/development/candidate-context-pack-20261106.md`.

- Fairy exact-current query nodes and snapshot projection: `examples/lexer_candidate/src/project_pack_node.fy`, `project_pack_snapshot.fy`; independent actual Go source/query and native controls: `internal/cli/candidate_pack_node_test.go`. Original and concrete generic/aggregate/cleanup facts stay distinct; relation selection and show/around remain separate. Evidence: `docs/development/candidate-context-pack-20261106.md`.

- Canonical detached Go query declarations/types across map transport: `internal/atlasmap/query_node.go`, `query_node_transport_test.go`, `query.go`, `query_generic.go`; owning snapshot/selection rules: `docs/atlas/fymap-query-v1.md`, `context-pack-v1.md`. Original and restored show/around v1/v2/v3 facts and mandatory byte limits agree; source snapshots/map bytes stay unchanged. Exact evidence and remaining Fairy query work: `docs/development/candidate-context-pack-20261106.md`.

- Fairy current-admitted Context Pack overview projection/selection/envelope: `examples/lexer_candidate/src/project_pack.fy`, `project_pack_overview.fy`, `project_pack_projection.fy`; independent actual Go encoder/checksum/native controls: `internal/cli/candidate_pack_overview_test.go`. Windows scope and remaining show/around/source-backed dispatch: `docs/development/candidate-context-pack-20261106.md`.

- Canonical Go overview header arrays across current-map transport: `internal/atlasmap/query.go`, `query_header_test.go`; owning query/pack rules: `docs/atlas/fymap-query-v1.md`, `context-pack-v1.md`. Detached empty arrays normalize before mandatory record measurement, so map restoration cannot change output or selection costs through nil slice state. Reproduction, regression and current scope: `docs/development/candidate-context-pack-20261106.md`.

- Fairy Context Pack argument selection: `examples/lexer_candidate/src/project_pack_options.fy`; independent actual Go parser/native controls: `internal/cli/candidate_pack_options_test.go`. Exact overview/show/around versions, limits, selectors, relation filters and action policy perform no workspace I/O. Windows evidence and the remaining complete pack/query/dispatcher sequence: `docs/development/candidate-context-pack-20261106.md`.

- Fairy source-backed calls command: `examples/lexer_candidate/src/project_calls_source.fy`, `project_calls_cli.fy`, `project_experimental_cli.fy`. Fresh requested-family Complete production precedes exact UID selection, current admission for every package, whole-workspace join and canonical catalog output. Argument/help precedes I/O; locked requests preserve bytes and unlocked requests refresh the ordinary source lock. Independent Go/native command/source/SDK oracles: `internal/cli/candidate_calls_command_test.go`, `candidate_calls_source_command_test.go`; full authored-main registration: `candidate_atlas_cli_test.go`. Exact Windows evidence, fingerprints and remaining broader/full-CLI/Linux scope: `docs/development/candidate-workspace-calls-20261106.md`.

- Fairy current joined-call output: `examples/lexer_candidate/src/project_calls_catalog.fy`, `project_calls_order.fy`, and the construction-budget retention in `project_atlas_join.fy`. Independent actual Go join/catalog/native byte, identity, ordering, checksum and budget/refusal tests: `internal/cli/candidate_calls_catalog_test.go`. Exact scope and Windows evidence: `docs/development/candidate-workspace-calls-20261106.md`. This private output API does not establish actual calls dispatch, source production, full CLI or Linux acceptance.

- Fairy call-catalog argument selection: `examples/lexer_candidate/src/project_calls_options.fy`; actual Go-parser/native comparisons: `internal/cli/candidate_calls_options_test.go`. Catalog 0/1/2 selects map 1/2/3, exact source selectors and caller bounds without workspace I/O. Output and dispatcher sequence, Windows evidence and remaining scope: `docs/development/candidate-workspace-calls-20261106.md`; this parser does not admit maps, join authority or actual CLI availability.

- Fairy source-backed symbols command: `examples/lexer_candidate/src/project_catalog_source.fy`, `project_catalog_cli.fy`, `project_experimental_cli.fy`. Complete-bound requested-family production precedes exact UID selection and current admission; arguments precede I/O, `--locked` preserves the workspace and unlocked queries use ordinary lock refresh. Independent Go-source/catalog checks: `internal/cli/candidate_catalog_command_test.go`, `candidate_catalog_source_command_test.go`; actual authored-main registrations: `candidate_atlas_cli_test.go`. Windows receipts, explicit non-call omissions and remaining full-CLI/Linux scope: `docs/development/candidate-symbol-catalog-20261106.md`.

- Fairy exact symbol-catalog request selection: `examples/lexer_candidate/src/project_catalog_options.fy`; actual Go-parser/native comparisons: `internal/cli/candidate_catalog_options_test.go`. Contract owners: `docs/atlas/symbol-catalog-v1.md`, `symbol-catalog-v2.md`, `symbol-catalog-v3.md`. Implementation sequence, remaining schema-production/dispatcher work and Windows evidence: `docs/development/candidate-symbol-catalog-20261106.md`. Parser admission alone does not grant map version conversion or actual CLI availability.

- Fairy current-admitted catalog v1/v2/v3 pages: `examples/lexer_candidate/src/project_catalog.fy`, `project_catalog_projection.fy`, `project_catalog_json.fy`; constant-time checked declaration projection: `project_atlas_validate::symbol_declaration`. Exact Go output/checksum, selection/output boundaries, cursor and atomic refusal comparisons: `internal/cli/candidate_catalog_test.go`. Each requested version requires its own current-admitted map; no map conversion, source-production selection or actual symbols dispatcher is implied. Scope and remaining sequence: `docs/development/candidate-symbol-catalog-20261106.md`.

- Explicit Fairy source-selected map families: `examples/lexer_candidate/src/project_atlas_source_version.fy` reauthenticates Complete source/signatures/coordinates and checks the entire source closure. Its refusal-only `select` preflight grants no map authority. `project_atlas_produce::package_version` binds that choice to its Complete-owned program; `project_atlas_facts::collect_version`, `project_atlas_header::header_version`/`completeness_version` and `project_atlas_write::envelope_version` generate the requested family directly. Existing factories retain v3. Real source/current/join/catalog comparisons cover ordinary SDK v1/v2/v3 and used/unused SDK Vec v3, with unsupported-family refusals: `internal/cli/candidate_atlas_source_version_test.go`. Exact acceptance and remaining dispatcher/full-CLI scope: `docs/development/candidate-symbol-catalog-20261106.md`.

- Explicit Fairy v2 map framing: `project_atlas_wire::read_v2` in `examples/lexer_candidate/src/project_atlas_wire.fy`; independent Go codec/native framing and boundary checks: `internal/cli/candidate_atlas_v2_wire_test.go`. Original `read` retains v1/v3; framing does not produce semantic/current-source Ready. Scope, failed-attempt accounting and remaining catalog integration: `docs/development/candidate-symbol-catalog-20261106.md`.

- Explicit function-generic v2 semantic partition: `project_atlas_validate::validate_v2` in `examples/lexer_candidate/src/project_atlas_validate.fy`; Go/native positive and resealed-negative oracles: `internal/cli/candidate_atlas_v2_admission_test.go`. V2 excludes generic aggregates/nominal arguments and remains isolated from ordinary/v3 factories. Current-source Ready and actual catalog integration remain separate; acceptance is recorded in the symbol-catalog plan above.

- Explicit v2 current-source partition: `project_atlas_current::read_v2` in `examples/lexer_candidate/src/project_atlas_current.fy`; independent compatibility inventories, actual unused-template source drift and version/work controls: `internal/cli/candidate_atlas_v2_current_test.go`. The shared current owner retains independent expectations and ordinary/v3 selection. Catalog page production/encoding/dispatch and v2 downstream join/output remain separate; see the symbol-catalog plan.

- Distinct owner-qualified function namespace preparation: `internal/genericsource/expand.go`; isolated collision/immutability controls: `qualified_owner_test.go`; ordinary-check/current-v3/call-join and ordinary-body preview/apply/method-refusal regressions: `internal/cli/atlas_owner_qualified_namespace_test.go`; ownerless AST edit selection: `internal/sourceedit/body.go`. Separate Fairy same-leaf v3 production/current/join proof: `TestCandidateAtlasOwnerQualifiedNamespaceProduction` in `internal/cli/candidate_atlas_owner_qualified_produce_test.go`. Full authored-main impact/MCP/ordinary-body publication and method refusal: `internal/cli/candidate_atlas_owner_namespace_command_test.go`, registered in `candidate_atlas_cli_test.go`. Exact qualified Go CLI filtering: `internal/cli/atlas_symbols.go`, `atlas_symbols_qualified_test.go`; contract amendment: `docs/adr/0201-exact-name-symbol-catalog-v1.md`. The owner-qualified acceptance record below separates these proofs from generated consumers and the missing Fairy symbols dispatcher.

- Owner-qualified struct/resource associated-function and method source production: `examples/lexer_candidate/src/module_signatures.fy`, `module_public_interface_extract.fy`, `module_unit_method_receiver.fy`, `module_unit_calls.fy`, `module_unit_move_walk.fy`, `module_unit_origin_calls.fy`, `project_atlas_declarations.fy`, `project_atlas_schema.fy`, `project_atlas_validate.fy`. Go snapshot/codec/query owners: `internal/semanticsnapshot/build.go`, `internal/atlasmap/codec_validate.go`, `query.go`. Independent source/production/Complete controls: `internal/cli/atlas_owner_qualified_oracle_test.go`, `candidate_atlas_owner_qualified_produce_test.go`, `candidate_owner_qualified_complete_test.go`; unsupported-source validity: `candidate_atlas_cli_test.go`. Actual command verifiers: `candidate_atlas_owner_qualified_command_test.go`, `candidate_mcp_owner_qualified_command_test.go`, `candidate_edit_owner_qualified_command_test.go`; full authored-main registration: `candidate_atlas_cli_test.go`. Contract, exact acceptance scope and outstanding command/image gates: `docs/development/candidate-atlas-owner-qualified-20261105.md`. Owner methods themselves remain noneditable. Enum/generic owners, native Linux and final whole-compiler acceptance remain separate.

- Copied-local closure source projection: `examples/lexer_candidate/src/project_atlas_closures.fy`, `project_atlas_schema.fy`, `project_atlas_calls.fy`, `project_atlas_produce.fy`; Go source/identity oracle and actual Fairy production: `internal/cli/atlas_copied_closure_oracle_test.go`, `candidate_atlas_copied_closure_produce_test.go`. Impact/MCP/ordinary-body editor verifiers: `candidate_atlas_copied_closure_command_test.go`, `candidate_mcp_copied_closure_command_test.go`, `candidate_edit_copied_closure_command_test.go`; full authored-main registration: `candidate_atlas_cli_test.go`. Contract and exact validation scope: `docs/development/candidate-atlas-copied-closures-20261105.md`. Broader closure forms, native Linux and final whole-compiler acceptance remain separate; owner-qualified source production has its own row above.

- SDK Vec unused/used source specialization oracle: `internal/cli/atlas_sdk_vec_oracle_test.go`; exact ten/fourteen implementation sets, preserved original SDK interface, physical fields, cleanup/borrow facts and caller endpoints. Candidate production sequence and evidence boundary: `docs/development/candidate-atlas-sdk-vector-20261105.md`. This Go prerequisite does not admit Fairy Vec maps or commands.

- Core SDK private Atlas producer/current/join APIs: `examples/lexer_candidate/src/project_atlas_facts.fy`, `project_atlas_declarations.fy`, `project_atlas_validate.fy`, `project_atlas_join.fy`; full SDK source/Go-map/native API comparison: `internal/cli/candidate_atlas_sdk_produce_test.go`. Command admission and its shared bounded SDK query ledger: `project_atlas_source.fy`; impact/MCP/user-body-edit command-leaf coverage: `internal/cli/candidate_atlas_sdk_command_test.go`, `candidate_mcp_sdk_command_test.go`, `candidate_edit_sdk_command_test.go`. Acceptance and remaining Vec/SDK-aggregate/full-CLI scope: `docs/ops/windows-candidate-atlas-sdk-20261105.md`. SDK declarations remain noneditable; SDK Vec production remains closed.

- Canonically ordered Atlas declaration/function lookup and symbol uniqueness: `examples/lexer_candidate/src/project_atlas_model.fy`, `project_atlas_validate.fy`; independent native key/work boundaries: `internal/cli/candidate_atlas_ordered_keys_test.go`; ordinary/generic invalid-map coverage: `internal/cli/candidate_atlas_validate_test.go`, `candidate_atlas_generic_validate_test.go`; Windows evidence: `docs/ops/windows-candidate-atlas-sdk-20261105.md`.

- Go v3 SDK source admission: `internal/workspacecompiler/check.go`, `internal/cli/atlas_pack.go`; MIR admission retention/reverification: `internal/compiler/check.go`, `internal/semanticsnapshot/generic_snapshot.go`; SDK Vec map and missing-admission regression: `internal/cli/atlas_sdk_vec_oracle_test.go`; current Windows evidence and outstanding Fairy SDK scope: `docs/ops/windows-candidate-atlas-sdk-20261105.md`.

- Decision: `docs/adr/0499-bounded-atlas-impact-mcp-and-body-edits.md`.
- Private Linux edit-file prerequisite: `internal/candidatehost/linux_edit_file.c`, `linux_paths.go`; independent Windows namespace/fault model and authored native replay: `linux_edit_file_test.go`. Go/Fairy runtime selection and pinned C assembly: `examples/lexer_candidate/src/linux_runtime.fy`, `linux_runtime_assets.fy`, `linux_runtime_asset_pin.fy`, `tools/package-candidate-linux-runtime.py`; live-Go comparison: `internal/cli/linux_runtime_test.go`, `linux_runtime_assets_test.go`. Exact boundary and remaining facade/CLI/Linux work: `docs/development/candidate-linux-atlas-edit-20261106.md`.
- Private Linux impact/MCP/symbol/call/pack command selection: `examples/lexer_candidate/src/linux_experimental_cli.fy`, `linux_file_cli.fy`, `linux_cli_data.fy`, with shared `project_atlas_cli.fy`/`project_mcp_cli.fy`/`project_catalog_cli.fy`/`project_calls_cli.fy`/`project_pack_cli.fy` help injection. Windows command/Go-graph/session/catalog oracle and authored full-entry native replay: `internal/cli/linux_atlas_cli_test.go`, `linux_mcp_cli_test.go`, `linux_catalog_cli_test.go`, `linux_calls_cli_test.go`, `linux_pack_cli_test.go`, `linux_file_cli_test.go`; independent Go command-help policy: `linux_help_test.go`. Exact implementation/evidence boundary, native signal-policy check and remaining commands: `docs/development/candidate-linux-atlas-cli-20261106.md`; native Linux execution stays batched.
- Contract: `docs/atlas/impact-mcp-edit-v1.md`; CLI admission: `docs/cli/command-contract.md`.
- Owners: `internal/atlasmap/impact.go`, `internal/atlasmcp/`, `internal/sourceedit/`, and `internal/cli/atlas_impact.go`, `atlas_mcp.go`, `atlas_edit.go`.
- Receipt: `docs/ops/windows-atlas-tools-20261102.md`; Linux replay: `docs/development/ordered-work-20261031.md`.
- Private Fairy reader/current-source gate: `examples/lexer_candidate/src/project_atlas_wire.fy`, `project_atlas_model.fy`, `project_atlas_symbols.fy`, `project_atlas_validate.fy`, `project_atlas_current.fy`.
- Private Fairy generic declaration binder/identity prerequisite: `examples/lexer_candidate/src/project_atlas_binders.fy`, `project_atlas_model.fy`, `project_atlas_symbols.fy`; Go-oracle/native coverage: `internal/cli/candidate_atlas_binders_test.go`; receipt: `docs/ops/windows-candidate-atlas-generics-20261105.md`. Generic source production and command evidence are tracked below; SDK Vec remains open.
- Private Fairy generic v3 transport: `examples/lexer_candidate/src/project_atlas_wire.fy`; native exact-byte preservation/version/budget checks: `internal/cli/candidate_atlas_generic_wire_test.go`. Function, aggregate and cleanup records are shaped without granting semantic or current-source authority; the same generic receipt records this separate prerequisite.
- Private Fairy explicit v3 semantic generic reader: `project_atlas_validate::validate_generics` and `examples/lexer_candidate/src/project_atlas_specializations.fy`; independent Go/native controls: `internal/cli/candidate_atlas_generic_validate_test.go`. Canonical implementations, scalar bindings, substituted aggregates, cleanup spans, local calls, relations and completeness are checked. Ordinary admission stays closed; explicit generic current admission and complete cross-package joins are now checked separately. Source production remains open; see the same generic receipt.
- Explicit Fairy v3 generic current/complete-join ports: `project_atlas_current::read_generics`, `project_atlas_join.fy`; Go-oracle/native controls: `internal/cli/candidate_atlas_generic_join_test.go`. Exact independent source/generic-interface expectations, implementation owner/ID, cleanup edges and impact JSON pass; source production/SDK/CLI integration remains separate.
- Original generic source headers: `examples/lexer_candidate/src/project_atlas_template_types.fy` and `project_atlas_schema.fy`; Go-byte/native tests: `internal/cli/candidate_atlas_generic_schema_test.go`. Source-only aggregate headers do not grant Complete or executable authority; ordinary/function templates also pass the complete Fairy factory. The separate aggregate Complete path now supplies instantiation/layout/cleanup and private v3 production through `project_atlas_instances.fy` and `project_atlas_aggregate.fy`; see `docs/ops/windows-candidate-atlas-aggregates-20261105.md`. `project_compiler_source_mode.fy` selects that factory from exact current source; `candidate_atlas_aggregate_command_test.go` verifies Windows command-leaf current-source queries. Aggregate MCP session/current-source recovery also passes in `candidate_mcp_aggregate_command_test.go`. `project_edit_preview.fy` derives complete headers and checks aggregate original/edited source; `candidate_edit_aggregate_command_test.go` verifies Windows preview, publication and refusal controls. SDK/aggregate combinations and refreshed full authored CLI replay remain open.
- Private Fairy verified-type and complete-source declaration projection: `examples/lexer_candidate/src/project_atlas_types.fy`, `project_atlas_schema.fy`; native Go-oracle checks: `internal/cli/candidate_atlas_types_test.go`, `candidate_atlas_schema_test.go`; Windows receipt: `docs/ops/windows-candidate-atlas-types-20261105.md`. The production path consumes these ports for ordinary, aggregate and C foreign declarations; Generic and core SDK production use the shared ports above; SDK Vec remains separate.
- Private Fairy join/traversal/output: `examples/lexer_candidate/src/project_atlas_join.fy`, `project_atlas_impact.fy`, `project_atlas_impact_json.fy`; Windows receipts: `docs/ops/windows-candidate-atlas-impact-20261104.md`, `windows-candidate-atlas-families-20261105.md`. Candidate dispatch: `project_experimental_cli.fy`, `project_atlas_cli.fy`, `project_atlas_options.fy`; verified-current production: `project_atlas_source.fy`, `project_atlas_facts.fy`, `project_atlas_declarations.fy`, `project_atlas_calls.fy`, `project_atlas_header.fy`, `project_atlas_produce.fy`, `project_atlas_write.fy`. Five declaration families, checked composite signature types, direct calls and external C calls are admitted in local packages. Drop coverage is exact; other non-call coverage remains explicitly unavailable. SDK Vec, broader language source families and Linux selection remain separate; MCP and body-edit evidence is tracked below.

- Ordinary interface/effects comparison remains the separate `internal/interfacereview/` owner.

## C transport, aggregates, and native proofs

- Private Windows scalar-aggregate workspace command selection: `examples/lexer_candidate/src/project_compiler_command.fy`, `project_compiler_source_mode.fy`, `project_compiler_cached.fy` and `project_test_cache.fy`; full-command/native/interpreter acceptance: `internal/cli/candidate_generic_aggregate_cli_test.go`; scope and current results: `docs/ops/windows-candidate-aggregate-cli-20261105.md`. Windows candidate test-cache flags are explicit opt-in; Go stable aggregate admission and SDK/aggregate Atlas combinations remain separate.


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

- **Bounded local state execution and persistence prototype; chain target UNDECIDED**
  - `docs/adr/0500-local-contract-state-execution-and-persistence.md`
  - `internal/contractstate/persistent_execute.go`, `local_persistence.go`, `local_codec.go`, `local_lock_windows.go`, `local_lock_linux.go`
  - `internal/contractstatemir/execute.go`
  - `tools/dev/fairyc-state/`
  - `docs/ops/windows-local-state-20261102.md`
  - This later local Go admission does not promote earlier oracle/native-C records into production durability or chain execution evidence.


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
