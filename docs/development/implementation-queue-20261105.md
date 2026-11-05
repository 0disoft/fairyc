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
