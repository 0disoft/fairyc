# Remaining implementation execution, 2026-11-05

User authorization: implement the remaining work discussed after `fdf7e6bb`, in
small Windows-verified commits. This is the full ongoing objective, not a claim
that finishing one slice finishes the queue. Native Linux execution remains
batched at the user's request; no push, CI dispatch, release or deployment is
authorized by this queue.

## Implementation

- [x] Fairy `experimental atlas-mcp`: bounded protocol core, explicit command
  policy, real stdio, current locked Fairy query, negative/native/full CLI tests
  pass on Windows low/optimized. Linux dispatch and execution remain below.
- [x] Fairy `experimental atlas-edit` in its admitted ordinary-source subset:
  current identity/source checks, single-body preview/apply, failure-atomic
  publication and Windows low/optimized command-leaf/full CLI acceptance.
  Broader source families and Linux dispatch/execution remain separate below.
- [ ] Broaden Fairy Atlas production to the Go-supported source families,
  including SDK, aggregates, foreign declarations and generic facts; retain
  independent current-source expectations and explicit completeness.
- [ ] Linux Fairy package fetch: retained-directory filesystem/cache publication,
  verified HTTPS transport, online/offline CLI selection and authored Linux tests.
- [ ] Linux experimental Atlas command selection: connect impact/MCP/edit to the
  Linux entry and select compatible host facades before the combined native replay.
- [ ] Generated-C/native persistent state execution through the existing exact
  layout/version compare-and-publish port, without weakening conflict/rollback
  or one-attempt behavior.

## Deferred designs to resolve and implement

- [ ] General inline bodies, composite generic transport, multi-operation static
  interfaces and first-class closures: inspect current boundaries, specify each
  expansion and implement in independent verified steps.
- [ ] Remote registry/version selection, signing/trust, Git sources and updates:
  prepare concrete contracts and local implementations; external registry or
  trust-provider selection requires the owner's actual choice.
- [ ] LLVM/QBE backend experiments and additional targets: preserve the C oracle,
  measure representative paired workloads and complete target-owned ABI/runtime
  contracts before adoption. Target selection remains UNDECIDED.
- [ ] Chain/VM adapters and production state provider: prepare adapter boundaries
  and local conformance tests. Chain/VM/provider selection remains UNDECIDED.

## Final acceptance work

- [ ] Refresh complete current-source Go/Fairy comparisons and consecutive
  self-build equality after implementation.
- [ ] Collect the authored Linux commands and remaining runtime observations into
  one transferable source/validation handoff; native execution remains deferred.
- [ ] Prepare and locally verify the current Windows distribution ZIP and
  clean-environment verifier. Clean-machine execution needs an actual environment.
- [ ] Keep same-commit hosted CI acceptance pending until an authorized push or
  dispatch. Existing historical receipts cannot attest the new commit.

## Progress

- Baseline: clean `main` at `fdf7e6bb`; all five implementation families and the
  deferred-design/acceptance families above remain in scope.
- Go MCP fallback error bounds are fixed in `e72b5564`; escaped IDs cannot exceed
  the response allowance even when the original raw ID was admitted.
- Fairy MCP protocol core passes Windows low/optimized Go-oracle replay. Its
  pure transition module, owned selectors and response validation are complete.
  Actual command/session and current-source query wiring now pass focused native
  command (209.544s) and full CLI (494.945s) acceptance in both profiles. See
  `docs/ops/windows-candidate-mcp-20261105.md`; this does not complete the queue.
- Fairy body edit syntax-only planning now passes 50 Go-oracle cases per Windows
  profile, including exact raw-byte spans/hashes, Unicode whitespace and injection
  refusal. Current root/symbol selection, complete edited-workspace verification
  and bounded preview reports also pass 21 native process calls per profile
  (242.935s). The actual Windows body-file preview command also passes in both
  profiles (183.671s); atomic apply and refreshed full CLI acceptance remain open. See
  `docs/ops/windows-candidate-edit-20261105.md`.
- Fairy edit invocation syntax also passes sixteen native invocations per Windows
  profile (5.552s), including Go flag/preview comparisons and an explicitly parsed
  apply bit. This is not yet the actual body-file/apply command.
- Exact Fairy edit fingerprints pass nine native invocations per profile
  (66.486s), with independent Go byte/hash construction and fresh locked-source
  refusal/recovery. The fresh port is ready for apply; publication remains open.
- Preview-owned fingerprints and complete fresh overlay revalidation now pass
  eight process-barrier drift scenarios per Windows profile (199.02s). Raw metadata
  drift and relocked dependency changes reject without writing; atomic application
  and its actual CLI acceptance remain open.
- Windows atomic application is now connected to the actual Fairy edit command:
  exclusive lock, complete fresh validation, staged write/sync, final closure and
  byte checks, one rename, cleanup and explicit post-publication errors. Existing
  previews and twelve apply controls pass per profile; three after-sync drift
  controls pass per profile after a Go-only live-tree harness repair. See the
  detailed reuse/failure accounting in the edit receipt. Refreshed full candidate
  CLI acceptance remains open; the broader implementation queue is not complete.
- Full edit-CLI input preparation exposed a 2,110,183-byte Go bootstrap header,
  13,031 bytes over its unchanged 2 MiB bound. Omitting definition-local names
  from shared prototypes reduces it to 1,919,958 bytes (190,225 bytes saved).
  Partitioned native/type/callback/inline and unchanged single-unit checks pass;
  complete native CLI replay and Fairy emitter/self-build acceptance remain
  separate. See `docs/ops/windows-bootstrap-header-20261105.md`.
- Refreshed full authored CLI acceptance now passes both Windows profiles
  (619.74s), including impact/MCP regressions, structural previews, atomic apply,
  old-lock refusal and ordinary-check lock regeneration matching Go bytes. Help
  dispatch includes atlas-edit. This closes only the bounded Windows edit family;
  broader Atlas sources, Linux paths, native persistence, deferred designs and
  final cross-platform/self-build/distribution/hosted gates remain open.
- The shared Fairy Atlas verified-type projection now matches all 16 ordinary
  function signatures in a two-package Go v3 oracle on Windows low/optimized
  (86.55s), including C scalars, nominal coordinates, borrows/slices, pointers,
  arrays and Option/Result. Invalid ID/work/coordinate controls reject. This is
  the type prerequisite; producer wiring, SDK and generic families remain open.
  See `docs/ops/windows-candidate-atlas-types-20261105.md`.
- The complete-source declaration port now preserves all 27 ordinary/aggregate/
  foreign declarations in a two-package Go v3 oracle, with exact JSON bytes on
  Windows low/optimized (75.94s). Six source/coordinate/budget/selector rejection
  controls pass per profile. Preparation binds the catalog to the Complete
  owner's exact provenance, including unused C declarations. Producer inventory,
  separate callable counts/summaries, external-call completeness, SDK/generic
  facts and actual command integration remain open; no queue family closes here.
- Five declaration families now enter the actual current-source producer. Function
  counts exclude aggregates/foreign declarations; calls preserve exact C ABI/link
  names and report external coverage explicitly. File and module tables retain
  their independent canonical orders. `TestCandidateAtlasProduction` passes both
  Windows profiles (77.39s): Go decodes every produced partition and matches all
  declarations plus ordered calls. The complete candidate CLI also passes both
  Windows profiles (1,008.38s): 17 selected declarations by visible/full IDs,
  exact-current contexts, resolved-call limits, existing MCP/edit/application and
  post-edit lock recovery. SDK/generic facts and the broader Atlas queue item
  remain open. See
  `docs/ops/windows-candidate-atlas-families-20261105.md`.
- Generic declaration binding and SymbolId arity now match the independent Go
  original-declaration oracle for functions, structs and resources. Windows
  low/optimized verify alpha-renaming, nested binder/nominal types, aggregate self
  types, 63/64 type-depth boundaries and malformed inputs. Ordinary map validation
  still rejects generic declarations explicitly. Existing wire/current/impact
  regressions pass; this is a prerequisite, not complete generic-map/SDK source
  production or CLI acceptance. See
  `docs/ops/windows-candidate-atlas-generics-20261105.md`.
- The v3 transport reader now preserves separate function, aggregate and resource
  cleanup specialization records, ordered type arguments and concrete call IDs.
  Used/unused Go v3 maps pass exact-byte native replay in both Windows profiles
  (9.816s), while unknown/omitted/duplicate bodies, noncanonical optional fields,
  v1 generic fields and byte/work overruns reject. Generic semantic admission
  remains explicitly closed until implementation ownership, substitutions,
  coverage and join checks are connected; no broad queue item closes here.

- Explicit v3 semantic generic admission now retains concrete tables and checks
  implementation identity/ownership/arity/scalar bindings, source aggregate
  substitution, contained resource-cleanup facts, exact local implementation
  calls, relation counts and conservative completeness. Windows low/optimized
  Go-oracle semantic tests pass (21.062s); existing binder/wire/current/impact
  regressions pass (119.684s). The ordinary/current reader remains closed until
  complete joins and actual source production are connected. SDK/CLI and final
  cross-platform gates remain open; see the generic receipt.

- Explicit independent current-source v3 generic reading and complete joins now
  preserve implementation endpoints, cleanup ownership and raw spans. Remote
  implementations resolve against the exact target declaration; Go-exact impact
  JSON preserves unavailable template-body coverage. Windows low/optimized
  generic joins pass (17.481s), including real unused-template source drift,
  missing/wrong-owner/erased remote IDs and zero-call unused templates. Actual
  Fairy generic source production, SDK/full CLI and final acceptance remain open;
  see the generic receipt.

- Original generic source declarations now project binders, nested/imported types,
  source spans and declared effects/cleanup with exact Go JSON. Windows source-only
  aggregate and complete generic-function plus ordinary schema tests pass
  (203.210s). Actual Fairy compilation still rejects user aggregate templates at
  catalog admission (4501); their complete instantiation/layout/cleanup path must
  be implemented, not inferred from the reader or source-only header port. Generic
  map production/SDK/CLI and the full queue remain open; see the generic receipt.

- Complete-bound generic function map production now preserves original used and
  unused headers, scalar argument tuples, separate implementation summaries and
  exact specialization call endpoints. Source-derived `g0_` IDs resolve through
  the checked program's call-target port rather than frame-number assumptions.
  Windows low/optimized production and current-source semantic admission/join
  pass against Go v3 (112.33s), including seven implementations and a zero-instance
  unused-template workspace. Actual generic command/full CLI, SDK and compiler
  aggregate execution remain open; the broader Atlas item is not complete.

- The actual generic `atlas-impact` command leaf now passes thirteen native
  invocations in each Windows profile (158.77s): visible/full original selectors,
  cross-package calls, exact generic counts, call/package limits, stale-lock
  refusal and unused-template source/interface/instance changes after refresh.
  SDK and copied-closure source rejection remain covered. The full CLI test now
  includes this verifier, but its complete authored-main/MCP/edit replay has not
  been rerun. SDK, aggregate compiler execution and the broader queue remain open.

- Aggregate execution's source-plan prerequisite now shares the exact source
  binding/catalog/planner owners through an explicit private route. Windows
  low/optimized match Go over five three-package variants (three roots, nine
  implementations each), public aggregate closures and cleanup-helper edits;
  seven rejection variants plus ordinary-gate/work controls pass (16.21s).
  Concrete fields/layout, ownership/cleanup, MIR/C, cache/Atlas and actual commands
  remain separate required steps. See
  `docs/development/candidate-generic-aggregates-20261105.md`; this does not finish
  aggregate execution or any broader queue family.

- Aggregate concrete type preparation now retains the sealed source plan, ordered
  scalar tuples, deduplicated struct/resource types and structurally substituted
  fields, with function/annotation/cast/constructor requests using the same graph.
  Windows native Go-oracle tests cover the source/type matrix and bounded failure
  controls; see the aggregate plan for final commands and results. Ordinary
  signature/interface and layout admission, ownership/cleanup, MIR/C, cache/Atlas
  and full CLI execution remain required; the broad aggregate queue stays open.

- The private aggregate source preparation now enters the actual ordinary
  `module_signatures::Program` owner, with explicit graph validation/mode and
  mode-preserving scalar/aggregate batch and specialization resolution. The
  original plan/fields remain owned for subsequent layout and provenance joins.
  Windows Go-Flow/expansion signature comparison and source/graph refusal controls
  are authored; final replay results are in the aggregate plan. Field/layout and
  Interfaces admission, ownership/cleanup, executable MIR/C, cache/Atlas and full
  commands remain required; no broader queue family closes here.

- Aggregate shape construction now joins exact retained instance/source facts and
  preserves specialized fields, abilities, repr(C) and cleanup declarations. The
  private field-contract route rejects illegal ownership/borrows/array elements
  and recursive by-value layouts while allowing nominal raw-pointer cycles.
  Fourteen three-package Windows low/optimized layout inputs plus freshness,
  instance, budget and repr guards pass (23.90s); see the aggregate plan. This is
  not target size/alignment proof or actual Interfaces/executable admission.
  Constructor/cleanup/Flow/MIR/C, cache/Atlas/commands and final acceptance remain
  required; the full implementation queue remains open.

  Final combined aggregate plan/type/signature/layout and ordinary Atlas type
  replay passes in 168.073s with both Windows profiles and no selected skips.
  Scoped vet, Go formatting and diff checks pass; full CLI/Linux/self-build/
  distribution/hosted acceptance remains open.

- Actual aggregate Interfaces now retain specialized shapes, constructor/body-type
  facts, resource cleanup tuples and generic helper requests through the existing
  whole-source and selected-unit owners. An explicit source-join route and
  scalar-only resource argument resolution preserve ordinary admission gates.
  The focused Windows low/optimized probe passes in 38.26s over eighteen source
  variants, selected root/import modes, field privacy and refusal controls; see
  the aggregate plan. Ownership/construction body checking, actual cleanup,
  Flow/MIR/C execution, cache/Atlas/full commands and final acceptance remain
  required. This does not finish the aggregate family or the full objective.

  The final seven-root aggregate/Atlas/legacy scalar admission replay passes in
  277.633s on Windows with no selected skips. Scoped vet, formatting and diff
  checks pass. Full CLI, Linux, self-build, distribution and hosted gates remain
  open.

- Aggregate construction values now pass through the actual call-closure/value
  owners, including ordinary/imported constructors, generic factories, record
  arrays, cleanup-body construction, whole-owner borrows and explicit burn.
  Resource drop requirements/contexts and exclusive field-borrow restrictions
  remain enforced. Thirty-five three-package Windows low/optimized inputs pass
  (54.84s), with Go Flow/expanded-field oracles and integer-boundary/invalid-field
  controls; see the aggregate plan. This is not resource ownership/Flow checking,
  executed cleanup, verified aggregate MIR/C, cache/Atlas or actual CLI acceptance.
  Those and the full implementation/final acceptance queue remain open.

- Aggregate ownership now enters the actual checked-body factory: move-only
  classification, origin/loan checks, cleanup obligations, no-drop blockers and
  cleanup-effect authority. The fifty-two-input three-package Go-oracle probe
  covers moves, nested/tagged ownership, cleanup receiver state, invalid field
  transfers, effects and escaping borrows. The final Windows low/optimized Flow
  and value regression replay passes in 139.713s with no selected skips; scoped
  vet, formatting and diff checks pass. See the aggregate plan. Executed cleanup
  and verified aggregate MIR/C, identities,
  cache/Atlas, actual CLI and the complete final acceptance queue remain open.

- Source-independent aggregate cleanup MIR now preserves resource recipes,
  exact cleanup-frame identities, ordered fields and semantic drop requests.
  A private verified route admits scalar tuples of length 1..8 while ordinary,
  project and artifact entry points remain closed. The final fifty-three-input
  Windows low/optimized probe passes in 72.19s with thirteen reconstruction
  refusal controls; see the aggregate plan and private cleanup MIR contract.
  This is not whole-body MIR, executable cleanup/C, linking/restoration,
  cache/Atlas or actual commands. Those and the complete final queue remain open.

  The unchanged Flow root passes both Windows profiles in 74.47s. The newly
  exposed four-check ordinary cleanup MIR regression initially finds stale 4 MiB
  fixture expectations against the unchanged 16 MiB product work limit; its
  corrected boundary/invalid-handoff replay passes both profiles in 46.26s.
  Scoped vet, formatting and diff checks pass; see the aggregate plan for exact
  failed-command/reused-result accounting.


- Owned aggregate body MIR now retains exact checked frame/node/local/call/field
  and layout facts. Eight tuple-in-value mutations, invalid frame and exhausted
  work controls reject, while unchanged bodies pass. The fifty-three-input Windows
  low/optimized body replay passes in 78.81s; unchanged cleanup/Flow roots pass in
  75.92s/70.55s. Their combined command failed only on a missing isolated ordinary
  test dependency. After that harness correction, all five authored ordinary body
  checks pass both profiles in 50.93s. Scoped vet, formatting and diff checks pass;
  see the aggregate plan for exact failure/reuse accounting. CFG/drop/defer
  sequencing, C emission/native values/traces, source identities/restoration,
  cache/Atlas/actual commands and every final queue gate remain open.


- Aggregate C acceptance exposed missing Go MIR cleanup of already completed
  owning fields when a later initializer propagates an error. The lowering now
  retains private typed prefix obligations and transfers them only on successful
  parent construction. Eight interpreter/codec/dual-target C-emission controls,
  complete MIR tests and selected nested/ignored-resource regressions pass on
  Windows; see `docs/ops/windows-partial-aggregate-cleanup-20261105.md` for exact
  failed/passed accounting. This does not close native aggregate C execution or
  any other broad queue item; that implementation continues next.


- Aggregate single-buffer C now admits concrete struct/resource records, inert
  record arrays and structural cleanup through the existing body/CFG C owner.
  The final twenty-input Windows low/optimized host replay passes in 146.16s:
  fifteen accepted C programs x both consumers x both hosts = 60 native runs;
  six independent Go references x both consumers = 12 native runs. Fixed values
  and nested/defer/partial-propagation/continue/break cleanup traces match in all
  72. Tuple metadata still rejects a C value spelling. Scoped vet, formatting
  and diff checks pass. Existing private consumer flags are used unchanged;
  an attempted stable `/W4 /WX` single-unit build reports pre-existing labels/
  narrowing warnings and is not claimed as passing. Partitioned C, remaining CFG
  families, source identities/restoration/cache/Atlas/SDK/actual commands and all
  final gates remain open. See the aggregate plan for exact failure/pass scope.


- Aggregate partitioned C now emits bounded shared header/literal/body packets
  through an explicit private route, preserving ordinary project admission.
  Fifteen accepted programs x two selection/split modes x two consumer profiles
  x two Windows host profiles pass all 120 native executions with fixed values
  and exact cleanup traces. Source/boundary/gate refusals and independent packet
  validation pass; single-buffer/Go regression, scoped vet, formatting and diff
  checks pass. See the aggregate plan for times and unchanged-input reuse.
  Source-bound identities, linking/restoration/cache/Atlas/SDK/actual commands,
  broader resource families and every final queue gate remain open. This closes
  only the bounded aggregate partition consumer, not the implementation objective.


- Aggregate unit source identities now bind concrete kind-20/21 descriptors and
  their ordered scalar tuples to original-source nodes, retaining exact source
  graph prefixes and cleanup receiver/call identities. The reproduced zero-type-
  identity cleanup refusal is fixed. Final twenty-input Windows low/optimized
  acceptance passes in 79.73s, and three unchanged ordinary SDK Vec identity/
  freshness/work/lifetime checks pass both profiles in 36.18s. Scoped vet,
  formatting and diff checks pass; the aggregate plan preserves failed harness/
  reproduction and valid reuse accounting. Project type/frame linking/rebasing,
  restoration/cache/Atlas/SDK/actual commands and every final gate remain open.


- Aggregate project linking now preserves ordered tuple edges, kind-21 resource
  recipes and both project/aggregate proof modes. Three independently checked
  source units retain original aggregate/cleanup/call identities through canonical
  type/frame merging and normal named C partitioning. The exposed unused-resource
  lowering refusal is fixed under the existing liveness rule; used missing layouts
  remain rejected. Fifteen programs x two output modes x two consumer profiles x
  two Windows hosts pass all 120 native executions (low 104.29s, optimized 151.22s).
  Ordinary type/link regressions pass in 130.709s, including limits, missing/duplicate
  bodies and four separate native runs. Scoped vet/format/diff pass; exact failure
  and unchanged-input reuse accounting is in the aggregate plan. Complete interfaces,
  restoration/cache/Atlas/SDK/actual commands and every final gate remain open.


- Aggregate Complete publication now derives original source exports, exact
  coordinate-bound concrete type spellings, ordinary/source hashes and ordered
  specialization sets after actual checking/linking. Tuple numbering preserves
  its second edge; metadata tuples and private instances cannot become public
  value spellings. The final twelve-positive/four-negative three-package probe
  passes Windows low/optimized in 83.87s, with 72 exact Go interface comparisons
  and explicit work/ordering/admission controls. This closes the bounded cold
  Complete-interface slice; artifact/current-source restoration, unit/disk cache,
  Atlas/SDK production, actual commands and every final acceptance gate remain
  open. Admitted spelling code shares the projection but does not yet admit
  aggregate artifacts. See the aggregate plan for failed-harness/final evidence.

  The final combined public-interface/ordinary-inline-artifact command passes in
  227.924s with no selected skips. Ordinary restoration/partitioning passes both
  profiles (143.92s) with four generated consumer executions returning 23. Scoped
  vet, formatting and diff checks pass; aggregate restoration is the next required
  implementation slice, not established by this ordinary regression.


- Detached aggregate cleanup/program/project artifacts now retain their explicit
  admission mode through distinct private markers and fresh full verification.
  Aggregate instance/cleanup snapshot identities and provenance module bounds are
  retained; ordinary readers and current-source Complete restoration still reject
  the new family. The final twelve-positive/four-negative transport matrix passes
  Windows low/optimized in 100.77s, with 48 exact codec roundtrips and 72 shared Go
  interface comparisons, plus corruption/work/admission controls. This is not an
  aggregate cache hit or executed reconstructed C. Current-source rebasing/fresh
  admission, unit/disk cache, Atlas/SDK, actual commands and all final gates remain
  open; see the aggregate plan and private cleanup MIR contract.

  The final transport/ordinary regression command passes in 243.269s, no selected
  skips. The ordinary restoration/inline root passes both profiles in 142.35s
  with four generated executions returning 23. Scoped vet/format/diff pass;
  current aggregate source rebinding and fresh admission continue next.


- Current-source aggregate artifact restoration now has an explicit private
  `Complete` factory. It reauthenticates exact sources/imports/coordinates,
  rederives original instance identities and ordered scalar arguments, rebases
  aggregate types while preserving cleanup mode, and re-enters full current
  body/Flow/recipe/state/SDK coverage before regenerated interfaces escape.
  The final fifteen-positive/four-negative Windows low/optimized probe passes
  in 176.71s, with 90 Go interface comparisons, 30 exact same-source C/artifact
  comparisons and 64 separately compiled/executed C consumers. A reversed-module,
  owner-shifted old packet also restores and executes; source/template/private
  body drift, swapped valid identities, altered resource abilities and exact/
  one-short work controls pass. See the aggregate plan for initial test-harness
  failures and final accounting. This closes current-source executable
  restoration only. Aggregate unit/disk cache keys and reuse, Atlas/SDK production,
  actual `check/build/run/test` integration and every final acceptance gate remain
  required. Native Linux execution stays batched; no push or dispatch.

  The final restoration/ordinary-inline-artifact command passes in 317.463s,
  no selected skips. Ordinary restoration passes both profiles in 140.64s, with
  four generated consumers returning 23 and empty output. Detached transport
  passed in 99.74s on unchanged inputs and is retained after the test-only
  reserved-identifier repair. Scoped vet/format/diff pass. Aggregate unit/disk
  cache reuse continues next; the full objective remains active.

- Aggregate package-unit persistence now uses a separate authenticated key and
  envelope family, fresh body/Flow/identity/table/state/SDK admission and the
  existing whole-project linker. Cold units publish only after Complete
  interfaces succeed. The final Windows low/optimized replay passes in
  217.23s/139.79s, with 36 fresh processes and 68 generated native executions.
  It covers exact reuse/invalidation counts, corruption repair, failed Complete
  preserving all cache bytes, exhausted-I/O cold fallback and nested/partial/
  loop cleanup. Independent cold public interfaces and same-grouping forced-cold
  C bytes agree; ordinary cold/warm and cross-family refusal controls pass.
  Scoped vet/format/diff pass. See the aggregate plan for failed-harness accounting.
  Aggregate whole-project cache restoration/reuse, Atlas/SDK production, actual
  command integration and every final acceptance gate remain open. Native Linux
  remains batched; no push or dispatch. The full objective stays active.

- Aggregate whole-project persistence now has explicit authenticated key,
  lookup, envelope and current-source restoration factories. Aggregate and
  ordinary routes/ledgers are separate, with no cross-family fallback. The
  Windows low/optimized replay passes in 208.69s/136.24s: 34 fresh processes,
  64 generated C executions, exact cold C/interfaces, source and key drift
  refusal, corrupt-generation repair, failed Complete preserving cache bytes
  and exact/one-short work controls. Existing ordinary keys match Go for all
  15 package pairs per profile over five source graphs (141.56s). Scoped
  vet/format/diff pass; see the aggregate plan for preparation failures and
  unchanged-input reuse. This closes the private whole-project cache API only.
  Atlas/SDK source production, actual aggregate commands and every final
  acceptance gate remain open. Linux stays batched; no push or dispatch.
