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
- [x] Windows candidate scalar-aggregate `check/build/run/test` selection,
  native/interpreter execution and explicit project/test-unit cache reuse.
  Go stable promotion, SDK aggregate combinations and native Linux stay separate.
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

- The final v2 current/semantic, ordinary/v3 current/semantic/complete-join and
  catalog-parser replay passes seven roots under both Windows profiles in
  134.914s. A pre-commit byte audit found CRLF working files against Git's LF;
  those otherwise unchanged files now match the stored checkout and the source
  lock is regenerated from canonical bytes. No otherwise clean source has a Git
  content diff. Earlier mixed-byte receipts are retained but do not attest this
  final input. Catalog page selection/encoding, per-version source production,
  actual dispatcher and the full final acceptance queue remain open.

- Explicit Fairy v2 current-source Ready now checks independent compiler,
  ordinary/generic interface, package/source/runtime and complete file/reference
  facts under the shared bounded ledger. A real unused exported-template edit
  preserves ordinary interface identity but correctly rejects the old map.
  Windows low/optimized execute 258 v2 current cases; existing ordinary/v3
  current and generic/aggregate complete joins also pass (157.111s combined).
  Catalog selection/checksum, schema-specific source production and actual
  dispatcher remain required. V2 downstream join/output needs separate evidence;
  see the symbol-catalog plan. Native Linux and the full queue remain open.

- Fairy v2 semantic admission now preserves bound function signatures while
  rejecting generic aggregates and nominal arguments. All shared ID, source-span,
  implementation/call, type and completeness controls remain active. Forty-one
  inputs plus three entry/budget controls per Windows profile pass 88 native
  executions; v2 and unchanged ordinary/v3 aggregate regressions pass together
  in 76.048s. Current-source Ready, catalog selection/encoding, current production
  and actual dispatcher remain open; see the symbol-catalog implementation plan.

- Explicit Fairy v2 map framing now passes nineteen input cases and four boundary
  modes in each Windows profile (46 native processes). Default wire selection
  still accepts v1/v3 only. V2/native v1-v3 regression passes in 42.832s; unchanged
  v3 generic framing passed in 25.24s and is reused after a test-only seed fix.
  No semantic/current-source authority or catalog dispatcher is granted. V2
  semantic/current admission, schema-specific production and actual catalog
  selection/encoding/commands remain required; see the symbol-catalog plan.

- Fairy symbol catalog argument selection now matches the Go parser for 105
  argument cases and three invalid actions under each Windows profile (33.237s),
  including version, kind, cursor, canonical numeric and exact length bounds.
  It performs no workspace I/O. Catalog selection/encoding, schema-specific
  current production and actual dispatcher remain required; the explicit v2
  framing follow-up above must not silently convert v3 maps. See
  `docs/development/candidate-symbol-catalog-20261106.md`. This is a prerequisite,
  not completion of the symbols command or the complete objective.

- Same-leaf full authored-main impact/MCP/editor groups now pass Windows
  low/optimized (296.675s), including reversed method declarations, ordinary
  same-leaf `get`, visible/full selectors, exact/one-short call budgets, workspace
  immutability, real stdio and ordinary-body publication with method refusal.
  The three appended groups reuse authenticated unchanged images; together with
  the earlier five groups they attest eight of thirty-two registrations. The
  other twenty-four groups, generated consumers and final acceptance remain
  open. See the owner-qualified receipt. Fairy symbols/calls/pack dispatchers,
  enum owners and remaining SDK/aggregate combinations still require work.

- Same-leaf Fairy Atlas production now preserves `Cell::get`, `Other::get` and
  an optional ordinary `get`: both sources pass Windows low/optimized (84.432s),
  returning eight package maps with exact independent Go v3/current/join facts
  and two/three calls. Existing compiler/runtime code already supports these
  namespaces. The original fourteen-case producer and controls remain registered;
  scoped vet/format/diff pass. Actual same-leaf commands, editor publication and
  full-CLI selection are recorded above; generated consumers and final queue
  gates stay open.

- Go generic source preparation now keeps owner-qualified method namespaces:
  valid `Cell::get`/`Other::get` and an ordinary same-leaf `get` no longer collide
  at v3 Atlas preparation. The two complete source/join cases and six bounded
  preparation controls pass, preserving caller ASTs and duplicate/reserved/generic-
  method refusals. Complete generic-source/planner packages and scoped vet pass.
  The original fourteen source oracles and edit oracle still pass. This is a Go
  prerequisite only; the separate Fairy production proof is recorded above.
  Actual same-leaf commands/editing and remaining queue gates stay open.
  Go ordinary-body selection now ignores
  same-leaf methods before uniqueness checking, fixing an independently reproduced
  ambiguous-selection failure. Exact preview/apply bytes and method refusals pass
  with all existing Atlas edit/native tests and the namespace/original edit
  oracles (7.980s), the full sourceedit package and scoped vet. Go's exact qualified-name
  CLI filter now passes all three catalog modes, 512/513-byte and malformed-path
  controls and existing symbol CLI regressions (1.631s); its normative contracts
  are updated. The corresponding Fairy command remains unimplemented.

- Owner-qualified Atlas production now matches fourteen independent Go v3 source
  oracles at both Windows profiles (190.10s): associated functions, shared/exclusive
  and consuming receivers, imported public owners, typed/alias/grouped receivers
  and record/resource dot calls. Complete admission passes thirteen positive and
  six negative controls (60.503s), including a corrected implicit-receiver move
  omission. Current-source `check`/`check --locked` pass with 643 files and 8,110
  MIR functions; the lock is refreshed. The combined ordinary/generic/static-
  operation/copied-closure/owner-qualified producer regression passes in 629.968s
  without selected skips. Final Go source controls, scoped vet, format and diff
  checks pass. Actual impact queries cover all fourteen source families; six
  real stdio MCP families and ordinary-body preview/apply with new associated,
  exclusive and consuming calls also pass both profiles (641.864s combined).
  Direct owner-method editing remains refused. All three verifiers are appended
  to the full CLI without deleting or reordering existing groups. The new full
  authored-main image passes those three groups and refusal/help at both profiles
  in 817.462s, with exact source/profile/toolchain and executable digests. This is
  five of twenty-nine registered groups; the other twenty-four and every final
  queue gate remain open. Additional same-leaf Fairy namespace sources are not
  attested by the original fourteen-family replay. See
  `candidate-atlas-owner-qualified-20261105.md`. Linux stays batched; no push or
  dispatch, and the full implementation objective remains active.

- Copied-closure Atlas impact, actual read-only MCP and ordinary body preview/apply
  now pass in the full authored-main Windows image at both profiles (346.019s).
  Four distinct helper/generic-owner selectors preserve exact six-call contexts,
  cutoffs, stale/relocked source and old-concrete-helper refusal. Ordinary and
  generic MCP sessions recover without source/host leakage or workspace writes.
  Editing changes capture values and replaces an i32 concrete owner with bool,
  with exact independent Go reports/publication and existing failure guards.
  A valid owner-qualified source now owns unsupported-source refusal evidence.
  The three new verifiers are registered in the full CLI; only these three plus
  the refusal group have been replayed on the changed image. The other twenty-two
  groups, broader Atlas sources and all final queue gates remain open. See the
  closure plan for source/image digests and exact scope. Linux stays batched;
  no push or dispatch. The full implementation objective remains active.

- Copied-closure Fairy Atlas production now binds private helper declarations
  and calls to exact checked concrete owner frames and g0 identities. Module-wide
  ordinary-before-concrete ordering agrees with Go, and unused generic templates
  emit no helpers. Seven independent source families pass Windows low/optimized
  production, current-source admission and complete joins (134.22s), with budget
  and missing-coordinate refusals. The combined ordinary/generic/static-operation
  production regression passes in 462.853s without selected skips. Original
  source check/locked pass with 642 files and 8,103 MIR functions; the source lock
  is refreshed. Actual closure impact/MCP/edit and changed full-image acceptance
  remain open, as do broader Atlas and every final queue gate. See the closure
  plan for reproduction, harness-failure and validation accounting.

- Copied-closure Atlas work exposed and fixes Go snapshot documentation-owner
  collisions between distinct helpers sharing a generic closure source span.
  Exact declaration keys plus spans now own attachments, with five refusal
  controls. Complete snapshot/map package tests, five independent CLI source
  oracles and scoped vet pass on Windows. This is a Go prerequisite; actual
  Fairy closure production/current/join/commands remain open. See
  `docs/development/candidate-atlas-copied-closures-20261105.md`.

- Static-operation Atlas production now passes all five independent Go v3
  source families in Windows low/optimized (165.744s): exact original declarations,
  unused templates, concrete cross-package algorithm/operation/provider endpoints,
  provider/private identity policy and current-reader/join/refusal controls.
  All twenty package maps match Go. The existing generic producer already handles
  the family; no product change is needed. Actual impact commands now pass both
  profiles in 175.514s, with exact six-call admission/refusal and stale/relocked
  provider identities. Actual read-only MCP passes both profiles in 100.776s with
  exact Go contexts and same-session stale-source recovery. Ordinary root-body
  editing passes both profiles in 109.395s, including fresh bool implementations,
  exact Go preview/apply reports, dependency-write refusal and publication guards.
  All three new full authored-main groups also pass both profiles, query groups
  in 172.115s and editor groups in 75.013s, with unchanged source/image digests.
  This closes the bounded static-operation Atlas family only. See
  `docs/development/candidate-atlas-static-operations-20261105.md`.
  Broader source families and the complete implementation/final queue remain open.

- Pre-closure authored-main Atlas CLI optimized acceptance passes all twenty
  named groups in 248.753s, retaining every original verifier and ordering.
  This includes ordinary/generic/aggregate/SDK/mixed impact, real stdio MCP,
  body preview/publication/refusal/freshness and help. The existing generated-C,
  runtime/profile/toolchain/stack and image digest policy protects retained-image
  reuse. Low mixed impact/MCP/help passes three selected groups in 628.749s, and
  the completed mixed editor retry passes in 441.443s, with matching source/image
  digests. Seven additional core query groups pass in 1042.132s, giving eleven
  distinct low groups; all three new static-operation groups also pass both
  profiles, giving fourteen low groups and all twenty-three optimized groups
  through the recorded bounded full-image replays. Two SDK impact groups pass
  in 531.301s and five ordinary editor/SDK MCP groups in 641.688s, giving twenty-
  one low baseline groups. The final SDK/SDK Vec editor replay now passes in
  674.058s with the same pre-closure digests, completing 23/23 distinct baseline
  groups in both profiles. The changed closure source still needs fresh image
  acceptance. The full-CLI
  receipt records exact coverage and excludes the earlier interrupted attempt.
  Broader source forms and every final gate remain
  open; native Linux stays batched. This is the exact pre-closure C/image baseline,
  not acceptance of a subsequently changed copied-closure compiler image.

- Mixed SDK/client aggregate body editing now passes actual Windows
  low/optimized commands in 451.186s. Independent Go overlay facts require
  fresh bool SDK Vec and same-name client Vec instances; the resource's real
  cleanup is retained. Exact Go preview/apply reports and published bytes,
  invalid field/SDK-write refusal, unchanged lock, staging cleanup and stale
  lock/hash checks pass. All three mixed impact/MCP/edit verifiers are now
  registered in full authored-main acceptance, which remains the next gate.
  More source forms, edited-client execution/cache, native Linux and every final
  acceptance gate remain separate; see the composition plan and editor receipt.

- Mixed SDK/client aggregate read-only MCP now passes actual Windows
  low/optimized stdio sessions in 221.177s. The shared mixed source preserves
  both aggregate owners and nonempty cleanup. Protocol negotiation, read-only
  tool listing, exact Go contexts, stale-source error and same-session recovery,
  no root/source leakage and workspace immutability pass. Full authored-main
  registration is updated, not rerun. Mixed body edits, broader source forms,
  native Linux and every final gate remain open; see the composition plan and
  MCP receipt.

- Mixed SDK/client aggregate `atlas-impact` now passes actual Windows
  low/optimized commands in 509.294s. One source combines SDK Vec, a same-name
  client Vec and a client resource with nonempty cleanup. Four SDK/root subjects,
  exact/one-short call admission, package limits and stale/relocked current
  contexts match Go without workspace writes. The ordinary SDK optimized
  control passes in 74.990s. Full authored-main registration is updated, not
  rerun. Mixed MCP/edit, broader source forms, native Linux and every final
  acceptance gate remain open; see the composition plan and impact receipt.

- Mixed scalar SDK Vec/user aggregate production now preserves the authenticated
  SDK constructor and existing layout/cleanup type alongside client struct/resource
  instances. The two reproduced preparation/constructor refusals are corrected.
  Five independent Go v3 source/map/current/join families pass Windows low and
  optimized in 751.183s, including same-name client Vec and callable pop/take-slot
  coverage, bounded failures and workspace immutability. Stable client admission
  and intrinsic ownership stay unchanged. See the composition plan and producer
  receipt. Actual mixed impact/MCP/edit, full authored-main, broader source forms
  and every final gate remain open; the broad Atlas queue is not complete.

- SDK/user scalar-aggregate composition now has a Go prerequisite: explicit
  compiler-private source policy combines authenticated SDK storage owners with
  scalar user struct/resource templates. Three current v3 source/map/join cases,
  all three stable-source refusals and six client-intrinsic/nominal controls pass
  on Windows (2.060s). Default source/cache admission and SDK intrinsic ownership
  remain unchanged. Fairy producer and command integration are the next steps,
  not proved by Go preparation. See the composition plan and Windows policy
  receipt; all remaining implementation and final acceptance families stay open.

- SDK Vec root-body editing now passes actual Windows low/optimized preview
  and apply in 429.975s. The overlay replaces i32 operations with newly requested
  bool Vec/type/helper implementations; exact Go preview/apply reports, published
  bytes, unchanged graph lock, staging cleanup, invalid overlays, SDK-write
  refusal and stale lock/hash checks pass. The ordinary SDK optimized control
  passes in 100.729s. The Go overlay's missing SDK authority prerequisite is
  corrected separately at `70396315`. The full authored-main Vec editor verifier
  is registered, not rerun. SDK/user-aggregate composition, native Linux and every
  final acceptance gate remain open; see the SDK vector receipt.

- Authenticated SDK Vec source production now passes the independent Go oracle
  at `1cb4f678`. The actual Windows impact command now accepts unused/at/pop
  sources in both profiles (435.12s), and the matching read-only MCP sessions
  pass (274.46s). Ordinary SDK impact/MCP controls also pass. Current-source
  identities, cutoff/stale/relocked behavior, protocol-only stdout and workspace
  immutability are retained. The complete command totals are 887.890s and
  572.800s. Full authored-main verifiers are registered, not rerun for this slice.
  Vec user-body editing, SDK/user-aggregate combinations and every final gate
  remain open. See the Windows SDK vector receipt. Native Linux stays batched;
  no push or dispatch.

- SDK Vec's independent Go oracle now fixes unused/used source expectations:
  ten implementations already exist for ordinary SDK view signatures; explicit
  root `empty/push/at<i32>` use produces fourteen. Original declarations/generic
  interface identity, exact instance tuples, physical fields, cleanup/borrow
  facts and root call endpoints survive canonical v3 codec reconstruction.
  The focused Windows Go command passes in 2.754s, including the existing removed
  storage-authority refusal. This is not Fairy/native Vec production. The
  source-authority/cleanup/type/header/command implementation sequence and
  failed-harness accounting are in
  `docs/development/candidate-atlas-sdk-vector-20261105.md`; all remaining queue
  families and final acceptance gates stay open.

- Core SDK impact/MCP/user-body-edit command leaves now pass Windows
  low/optimized: impact 617.996s, MCP 445.35s, edit 614.563s, with 28 fresh
  native processes and exact Go current-context/report comparisons. SDK source
  projection retains its 16 MiB ledger; one separate shared 16 MiB ledger
  covers all current-map admission/join/traversal/output operations. Non-SDK
  allowance is unchanged. SDK writes and Vec source queries remain rejected.
  Valid edit apply preserves all four semantic checks and takes 200.126s/6.429s;
  its earlier three-minute diagnostic failure is retained, and final apply uses
  a bounded six-minute test cap without product timeout changes. Ordinary Atlas
  command regression passes both profiles in 761.012s; scoped vet/format/diff
  pass. Full authored-main replay is registered but not rerun. Vec production,
  SDK/user-aggregate combinations, Linux and all final gates remain open; see
  `docs/ops/windows-candidate-atlas-sdk-20261105.md`.

- Core SDK Atlas source production/current-admission/join APIs now pass Windows
  low/optimized in 318.186s, with exact Go declarations, IDs, interfaces and calls
  across all three packages and ten fresh native processes. The hash allowance,
  repeated identity-header parsing and cross-package join lookup prerequisites
  are fixed. Final lookup/invalid-map/generic-join/impact regressions pass in
  272.804s. This API proof uses explicit separate bounded projection and admission
  ledgers; actual SDK command ledger integration/dispatch remains open, as do Vec
  production, SDK aggregate combinations, Linux and all final acceptance gates.
  See the Windows Atlas SDK receipt for failures and the exact acceptance scope.

- Canonical Atlas declaration/function lookup now uses validated sorted tables,
  and symbol ownership uses a bounded bitmap plus adjacent validated ID prefixes.
  Windows low/optimized ordinary/generic invalid-map and 0/1/2/257-row lookup
  boundaries pass in 41.644s. This removes quadratic lookup/duplicate work needed
  by SDK maps; full SDK production/commands and final acceptance remain open.

- Go v3 Atlas SDK Vec preparation no longer loses workspace-authenticated
  storage admission. Snapshot MIR reverification retains that admission and
  rejects when it is removed. Focused Windows source/map/policy/snapshot and
  workspace/cancellation checks pass. Fairy SDK production/commands and every
  final acceptance gate remain open; see the Windows Atlas SDK receipt.

- Actual scalar-aggregate workspace commands now select Complete and matching
  cache families from retained source. The full authored Windows command roots
  pass low (434.19s) and optimized (167.723s), 54 fresh CLI processes in total,
  including both program profiles, exact private-Go check/list/interpreter
  comparisons, value-copy/mutation, arrays/nested/empty resources, cache-hit test
  reruns, failure/invalidation/repair and cold fallback. Concrete interpreter
  records are admitted only under verified aggregate mode; tuple metadata stays
  excluded. Windows explicitly enables the existing private opt-in test-cache
  parser with unchanged flag-free behavior. The separate ordinary SDK control
  passes both profiles in 143.793s; see the aggregate CLI receipt. SDK Atlas, native
  Linux and every final acceptance gate remain open.

- The authored full Atlas CLI snapshot after `157dc58e` passes both Windows
  profiles in 1590.053s, including its aggregate impact/MCP/edit verifiers.
  Its exposed C partition prefix overrun is fixed without changing the 2 MiB
  consumer ceiling. This evidence precedes the ongoing aggregate workspace-command
  selection/interpreter changes; those inputs and every final gate remain open.

- Aggregate root-body editing now passes Windows low/optimized command-leaf
  acceptance in 182.45s (22 native invocations). Exact Go reports, imported
  aggregate parameters, new replacement-body Cell<bool> instantiation, published
  bytes, complete stage/lock cleanup, unchanged fairy.lock, invalid overlays,
  noneditable subjects and stale lock/hash controls pass. Target restrictions
  and locked fresh publication remain unchanged. The full authored CLI verifier
  is registered but its refreshed execution, SDK source production, native Linux
  and all final gates remain open; see the aggregate Atlas receipt.

- Aggregate MCP stdio sessions now pass Windows low/optimized in 101.18s: two
  real processes, eight tool requests, exact Go impact/current-image contexts,
  unknown-package isolation, stale source/lock refusal and relocked recovery in
  the same process. Native requests preserve workspace bytes and do not reveal
  raw source/host paths. The full CLI verifier is registered but its refreshed
  run remains pending. Aggregate editing, SDK, native Linux and the broader
  implementation/final queue remain open; see the aggregate Atlas receipt.

- The actual Fairy aggregate impact command now selects its Complete, signature
  and instance factories from exact retained source. Windows low/optimized
  command-leaf acceptance passes in 144.66s: 22 native invocations, five original
  declarations plus full ID, cross-package cleanup callers, exact Go current
  contexts, call/package bounds and stale/relocked unused-template drift. The
  full authored CLI verifier is wired but has not been rerun for this slice.
  MCP aggregate sessions, edit selection/overlay/application, SDK, native Linux
  and every final gate remain open. See the aggregate Atlas receipt.

- Private Fairy aggregate v3 map production now passes all eight three-package
  Windows low/optimized scenarios in 246.69s: 48 native invocations and 48 returned
  maps, strict independent Go declaration/SymbolId/interface/aggregate/cleanup-call
  comparisons, current-source revalidation, complete joins and refusal controls.
  Source repr(C)/declared effects are retained. The reproduced empty-cleanup
  frame refusal is fixed without inventing an executable summary. See
  `docs/ops/windows-candidate-atlas-aggregates-20261105.md`. Actual aggregate
  query/MCP/edit and check/build/run/test selection, SDK production and every
  final acceptance gate remain open. Linux stays batched; no push or dispatch.

  The final aggregate/function-generic/source-header/ordinary production command
  passes both profiles in 477.908s with no selected skips. Scoped vet, formatting
  and diff checks pass. This closes the private aggregate producer only, not the
  broader Atlas/SDK/command or final-acceptance queue.

- Complete-bound Fairy aggregate instance projection now passes Windows
  low/optimized in 106.21s: six three-package source scenarios, exact Go v3
  instance/type/cleanup-call comparisons and four refusal controls per scenario
  (60 native probe invocations). Original ordered scalar IDs join concrete
  aggregate types and resource cleanup frames; tuple metadata cannot render as
  a value. This is not aggregate map production or actual command admission.
  Atlas/SDK production, aggregate check/build/run/test and every final queue gate
  remain open; Linux stays batched. See the aggregate implementation plan.

- Atlas aggregate-instance preparation exposed a Go snapshot prerequisite:
  generic aggregate headers rejected every attribute, including declared cleanup
  effects used by the existing native aggregate fixtures. The header adapter now
  preserves struct repr(C) and resource declared drop effects, with used/unused,
  detached-view and ten malformed/misplaced attribute controls. Complete
  semanticsnapshot and atlasmap Go suites pass on Windows. This is a Go header
  fix only; Fairy aggregate-instance projection/production and all remaining
  queue items still require their own acceptance.

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
