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
- [ ] Linux experimental Atlas command selection: connect impact/MCP/edit/symbols to the
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

- Linux fetch now has an explicit monotonic 120-second deadline prerequisite.
  Independent Windows low/optimized timestamp/overflow/native-error and Fairy
  deadline boundary probes pass, as do authored check/locked (691 files / 8,604
  MIR functions). Clock errors expire fail-closed. Native Linux replay is authored
  and stays batched. Offline restore, HTTPS, CLI and all broader/final gates stay
  open; see the Linux fetch receipt.

- Linux archive-cache storage now validates pins/size/input hashes before I/O,
  rehashes every hit, preserves corrupt/linked/aliased/leftover entries and syncs
  anonymous files before one no-replace publication attempt. Independent Windows
  cold/warm, simulated collision, failure/close and lifetime controls pass both
  profiles, as do reference check/locked (690 files / 8,599 MIR functions).
  This closes the private storage API only: offline restore, HTTPS, CLI and
  actual Linux execution remain open. See the Linux fetch receipt.

- Linux package filesystem composition now preflights contained paths and case/
  type conflicts, reads exact stable single-linked binary files, syncs anonymous
  files before exclusive publication and clears owned stages under entry/depth
  bounds. Held-directory cloning preserves object identity across lease release.
  Independent Windows two-profile filesystem/lifetime/error probes, scoped vet
  and reference check/locked pass (689 files / 8,594 MIR functions). Cache,
  offline restore, transport/CLI and native Linux remain open; see the fetch receipt.

- Linux package-file observation now rejects hard links without changing generic
  file semantics; conditional cleanup preserves exact metadata and partial-error
  results. Package scans admit 8,192 rows while ordinary scans retain 4,096 and
  the same work budget. Independent Windows low/optimized C/Fairy probes and
  reference check/locked pass (688 files / 8,576 MIR functions). Native Linux
  file replay is authored but remains batched; package filesystem/cache, offline,
  HTTPS and CLI integration remain open. See the Linux fetch receipt.

- Fairy retained package-directory capabilities now bind the native prerequisite
  to owned child/parent handles, retained names and publication/removal phases.
  Independent malformed/partial output, borrowed-root preservation and lifetime
  probes pass Windows low/optimized; rooted/directory/retention/cache regressions
  and reference full-source check pass. See the Linux fetch receipt. Package
  filesystem/cache/HTTPS/CLI integration and native Linux remain open.

- The Linux package-directory native prerequisite now selects retained parent/
  child creation/edit, identity-checked empty removal and no-replace rename.
  Independent Windows low/optimized boundary probes and existing Linux adapter
  regressions pass; GCC/Clang native replay is authored but remains batched.
  The following capability slice is recorded separately; package/cache
  publication, HTTPS and CLI remain open. See
  `candidate-linux-package-fetch-20261106.md`; no broad queue closes.

- Named implicit shared/exclusive owner receivers now enter the existing
  argument, move, origin, detached-body, C and interpreter owners. The Go
  returned-borrow prerequisite is committed as `ecab5f4f`; Atlas move collection
  now distinguishes those borrows from consuming receivers. The current
  Windows acceptance receipt is `candidate-implicit-receiver-20261106.md`.
  Grouped/projected/temporary receivers, broader source-family parity and every
  final gate remain separate. Native Linux remains batched; no push or dispatch.

- Declared-effects production now agrees with Go completeness: every Package
  declaration passes the source schema, legacy header factories retain unavailable
  evidence, and effects do not inflate executable relation counts. Twelve authority
  names plus ordinary/foreign/generic/cleanup and cross-package source cases pass
  both Windows profiles (18 processes / 28 independent maps). Explicit collection
  controls pass; optimized generic/SDK Vec production and six real authored-main
  CLI groups pass, as do authored check/locked, scoped vet and map/pack/snapshot
  regressions. Exact commands, failed attempts and skipped scope are recorded in
  `candidate-atlas-effects-20261106.md`. The earlier effects-completeness difference
  below is resolved. Broader source-family parity, Linux implementation, native
  state, deferred designs and every final gate remain open; no push or dispatch.

- Complete-bound borrow maps now preserve original spans, shared/mutable kinds
  and escape flags in ordinary/generic/cleanup summaries, counts and exact
  completeness. Raw-IR legacy factories retain unavailable evidence. The actual
  impact/MCP/edit producer now enters the same Complete-bound versioned factory.
  Optimized source/version/current/join replay passes 218 processes / 381 maps;
  six actual optimized CLI groups pass, and seven new Pack families pass both
  profiles (48 new commands). The separate prototype-header fix preserves native
  limits. See `candidate-atlas-borrow-production-20261106.md` for SDK, failed-run
  accounting and exact Windows scope. Effects completeness remains a Go/Fairy
  difference; Go itself still marks reads/writes/test-calls unavailable. Broader
  implementation and every final gate remain open. Linux stays batched; no push.

- Borrow observations now attach to the actual Complete owner, follow every linker
  alias and reconstruct from current checks on unit/project artifact restoration.
  The independent twenty-eight-family source matrix checks exact ordinary/generic/
  cleanup facts, cold/restored equality and executable unit re-encoding. Work-fold
  boundaries and an accidental 65,536 cumulative-project cap are covered separately;
  see `candidate-atlas-borrow-observations-20261106.md` for final Windows receipts.
  Borrow map/count/completeness/commands, broader implementation and every final
  gate remain open. Native Linux stays batched; no push or dispatch.

- Fairy checked-body semantic borrow retention and private single/batch collection
  pass twenty-seven independent Go source families and twenty controls per Windows
  profile in 77.221s: ninety-four native processes without selected skips. Exact
  source spans, kind, escapes, canonical order, owner/frame multiplicity and work/
  count boundaries pass with empty PATH and unchanged workspaces. Authored check/
  locked pass 681 files / 8,506 MIR functions; scoped vet and hygiene pass. See
  `candidate-atlas-borrow-sites-20261106.md`. Complete-bound/remapped borrow metadata,
  production/completeness/commands and all broader/final gates remain open. Native
  Linux stays batched; no push or dispatch.

- Seventeen independent Go borrow source/map-round-trip cases and two unsupported
  source controls pass in 1.264s. Returned unmaterialized parameters retain their
  expression spans; retained loans keep their written borrow sites and escape
  flags. Snapshot/map/pack regressions and scoped vet pass. See
  `candidate-atlas-borrow-sites-20261106.md`. This freezes the Go prerequisite;
  Fairy collection/production/commands and all broader/final gates remain open.
  Native Linux stays batched; no push or dispatch.

- Complete-bound Result-propagation production retains original question spans
  in ordinary, specialized and concrete cleanup summaries, relation counts and
  exact returns-error completeness. Six independent source families cover root
  and dependency coordinates and all schema selections under both Windows
  profiles: 149 native processes / 243 current maps, with the full old source
  corpus also replayed under optimized. Six selected actual optimized CLI groups
  pass, and six new pack cases pass both profiles (42 processes, no selected
  skips). Authored checks pass 679 files / 8,480 MIR functions; scoped vet and
  hygiene pass. See `candidate-atlas-return-error-sites-20261106.md`. Borrow
  collection/production, broader implementation and all final gates stay open;
  native Linux stays batched, no push or dispatch.

- Fairy verified CFG Result-propagation collection retains exact written question
  tokens after completed operands and excludes earlier terminal paths. Nineteen
  independent Go source cases and twenty frame/work/count controls pass both
  Windows profiles in 406.107s: 78 native processes, no selected skips. Authored
  checks pass 679 files / 8,479 MIR functions, as do scoped vet and hygiene. See
  `candidate-atlas-return-error-sites-20261106.md`. Complete-bound error maps,
  counts/completeness and actual commands are still required, along with borrows,
  every broader implementation/final gate and batched Linux work.

- Complete-bound Region allocation production retains original full-call spans
  in ordinary, generic and concrete cleanup summaries, relation counts and exact
  allocates completeness. Windows source/version/current/join comparisons pass
  128 native processes and 219 independent maps; all five allocation source
  families pass every version selection under both profiles, while optimized
  also replays the full existing source corpus. Six selected actual optimized
  CLI groups pass. All five actual allocation pack cases also pass both profiles,
  thirty new main-command processes without selected skips; bounded low parents
  exit zero in 697.183s and 1079.871s against the same authenticated image. See
  `docs/ops/windows-candidate-atlas-allocations-20261106.md`. Error/borrow
  collectors, broader source/command work and every original final gate stay
  open; native Linux remains batched, no push or dispatch.

- Fairy verified CFG Region allocation collection now retains full original
  call spans and terminal-operand reachability in ordinary, generic and cleanup
  frames. Twenty independent source cases and twenty frame/work/count controls
  pass both Windows profiles: eighty unique generated compiler processes,
  empty PATH and unchanged workspaces. The observed-relations plan records
  bounded low parents, earlier timeout failures and a separate current-bootstrap
  renewal. Allocation map/header/completeness/CLI production and all remaining
  original implementation/final gates stay open; native Linux stays batched.

- Go snapshots now retain one fully identical source call when defer lowering
  copies it onto mutually exclusive exits, with completeness counted after
  canonicalization. Fourteen error-site source/codec controls and three new
  ordinary/cleanup/generic call-inventory cases pass in 1.134s; the complete
  snapshot/map/pack suites and scoped vet pass. The owning snapshot contract
  clarifies source-site multiplicity. See the observed-relations plan for the
  original failures and separate native Result receipt. Fairy collectors/maps,
  broader implementation and all final gates remain open; Linux stays batched.

- Thirteen independent Go Result-propagation source/codec controls now retain
  exact written question-token spans and partial-operand reachability. A
  reproduced MIR invariant failure preserves the original terminal operand;
  the required MSVC Result-tag read barrier also fixes optimized C4702 without
  weakening warnings. All six Windows native executions pass with exact trap
  receipts/early return, in a 9.775s combined command. The observed-relations
  plan records the separate deferred-call canonicalization defect and remaining
  Fairy collector/map/CLI work. Native Linux and all broader/final gates stay open.

- Fairy Region `alloc_zeroed_bytes` and `copy_bytes` now use resolved lexical
  capabilities, contextual `usize`/byte-view checking, original Region origins,
  retained lifecycle descriptors, verified CFG/IR and the existing native runtime.
  Same-spelled record methods remain ordinary calls. Windows low/optimized pass
  all eight source cases in 371.984s: sixteen independent Fairy interpreter/C
  emissions and thirty-two emitted-program executions. See
  `candidate-region-bytes-20261106.md`. Atlas allocation collection/production,
  the broader source queue and every final gate remain separate; native Linux
  stays batched, no push or dispatch.

- The Region allocation prerequisite now freezes fourteen original Go call-span
  observations, including byte placements, partial argument order and heap-call
  exclusions. It reproduces and fixes a Go MIR internal failure after a trapping
  placement operand; place/zeroed-bytes/copy-bytes now preserve the original trap
  without inventing a placement. All fourteen Go controls and six actual Windows
  low/optimized SDK/Arena trap executions pass (11.640s combined), as do the full
  Go MIR package (1.208s), scoped vet and hygiene. The observed-relations plan
  retains the explicit direct-Arena source limit and failed probe commands.
  Fairy allocation collection/production, low full-CLI replay and all remaining
  original work stay open; native Linux stays batched, no push or dispatch.

- Complete-bound move production now preserves original sites in ordinary,
  generic and concrete cleanup summaries, relation counts and exact moves
  completeness. An independent source case also fixes checked-but-unexecuted
  parent calls: a private verified-CFG completion ledger preserves earlier
  operand/call observations and treats deferred invocation separately from capture.
  Windows source/current/join validation passes 196 native processes and 294
  independently prepared current maps across both profiles; bounded low parents
  match all 93 optimized source/version selections without skips. Actual new
  move pack cases pass both profiles, and all six selected actual optimized CLI
  groups pass in 93.43s. Authored check/locked pass (676 files, 8,423 MIR functions),
  Go atlasmap/contextpack, scoped vet and hygiene pass. See the observed-relations
  plan for failed attempts, exact commands/logs and remaining low full-CLI scope.
  Allocation/error/borrow collection, broader work and every original final gate
  remain open; native Linux stays batched, no push or dispatch.

- Fairy verified CFG move-site collection now retains original evaluated operand
  spans through bindings, calls, returns, owned fields/tags, scoped/conditional
  results, generic bodies and concrete cleanup. Thirty-four admitted sources,
  two retained implicit-receiver source-factory gaps and twenty controls per
  Windows profile pass 112 native executions without selected skips (low
  368.35s, optimized 71.76s). The expanded run reproduced and fixes a missing
  if-then tail observation; independent Go facts remain unchanged. Authored
  check/locked pass (675 files, 8,417 MIR functions), as do scoped vet and hygiene.
  See the observed-relations plan. Complete-bound move production/commands,
  allocation/error/borrow collectors, broader source work, unsafe low replays
  and every original final gate remain open; native Linux stays batched.

- Original Go move-site observations are fixed by fourteen source/codec controls
  (Windows 1.123s). A consumed first argument remains observed when a later
  argument traps; an unevaluated later argument, discard and internal Result
  propagation do not create MoveSites. The observed-relations plan records the
  resulting operand-level collection requirement. This is a Go prerequisite;
  Fairy move collection/production, unsafe low CLI replay and every remaining
  implementation/final gate stay open. Native Linux remains batched.

- Complete-bound unsafe production now retains original Go sites in ordinary,
  specialized and concrete cleanup summaries and counts them with calls/drops/
  traps. Only uses-unsafe becomes newly exact. Windows source/current/join
  comparisons pass both profiles (low 789.366s, optimized 97.914s), 124 native
  processes and 166 independent current maps. All six selected actual optimized
  CLI groups pass in 179.891s. The ordinary/generic/SDK Vec production roots
  pass both profiles in 929.227s. Both three-group low CLI parents exceeded
  their aggregate 20-minute test budget; those commands remain failed, with
  completed group assertions recorded separately. Source/SDK packs now pass
  separate 15-minute replays in 312.971s/449.408s against the same authenticated
  image. Four earlier low groups still lack separate zero-exit parents, so
  complete low command acceptance remains pending. Authored check/locked pass (674
  files, 8,384 MIR functions), as do Go atlasmap/contextpack regressions and
  scoped vet/hygiene. See the observed-relations plan and
  `docs/ops/windows-candidate-atlas-unsafe-cli-20261106.md`. Every remaining
  collector, broader implementation and final gate stays open; Linux remains
  batched, no push or dispatch.

- The shared JSON tree now indexes each object's decoded keys with binary
  search, retaining duplicate spans, original token/source order and the same
  16,777,216 work limit. New Windows low/optimized key/budget controls and the
  unchanged source-plan corpus pass in 61.225s (28 new native executions plus
  1,698 source-plan cases). Authored check/locked (674 files, 8,383 MIR functions),
  format and diff checks pass. See `candidate-json-key-index-20261106.md`.
  Unsafe map/command production retains its separate acceptance gate; all
  remaining implementation/final work and batched Linux stay open.

- Fairy verified CFG unsafe-site collection passes 29 source fixtures and
  twenty frame/work/count controls per Windows profile: 98 native processes,
  no selected skips (low 479.689s, optimized 84.173s). Safe calls in unsafe
  scopes remain absent, while raw views/addresses, foreign calls and SDK storage
  operations retain exact Go spans. Folded Vec calls keep their original
  checked helper observations separately rather than duplicating caller sites.
  Authored check/locked (674 files, 8,382 MIR functions), scoped vet and hygiene
  checks pass. See the observed-relations plan. Complete-bound unsafe map/header/
  command connection, every remaining collector and broader/final gate stay
  open. Native Linux remains batched; no push or dispatch.

- Complete-bound Fairy Atlas production now retains sorted trap categories in
  ordinary/generic/cleanup summaries, counts them with calls/drops and marks only
  may-trap newly exact. The versioned source/current/join comparison passes
  Windows low/optimized in 676.152s: 100 native probes, 130 independent current
  maps. All six selected actual authored-main command groups now pass both
  profiles at `53728a07`, twelve combinations without selected skips. The
  optimized parent passes in 158.813s; bounded low parents and authenticated
  image digests are recorded in `docs/ops/windows-candidate-atlas-traps-cli-20261106.md`.
  The ordinary/generic/SDK Vec production-root command passes both profiles in
  754.834s, including independent maps and refusal controls. Authored check/locked
  (673 files, 8,345 MIR functions), Go
  atlasmap/contextpack regression and scoped vet/format/diff pass. See the
  observed-relations plan. All remaining collectors, broader implementation and
  final gates remain open; native Linux stays batched, no push or dispatch.

- Shared Fairy Atlas drop/trap collection now uses one whole-IR proof and one
  private CFG traversal per frame while keeping standalone count policies
  isolated. Fifteen drop and twenty-eight trap sources, eighteen controls per
  collector and twelve combined controls per Windows profile pass: 182 native
  probe processes in 769.619s, no selected skips. Authored check/locked (673
  files, 8,344 MIR functions), scoped vet/format/diff pass. The observed-relations
  plan records earlier rejected attempts and exact timings. Trap map/header/
  command production is a separate in-progress change; all remaining collectors,
  broader implementation and final gates stay open. Native Linux remains batched.

- Fairy verified CFG trap-category collection now retains the six canonical
  Go categories through scalar arithmetic, assertions, bounds and implicit
  iteration checks. Single/batch comparisons cover ordinary, generic and
  concrete cleanup owners; all 28 source fixtures plus eighteen controls per
  Windows profile pass in 367.888s (92 native compiler processes, no selected
  skips). Authored check/locked (672 files, 8,339 MIR functions), scoped vet,
  formatting/LF/diff checks pass. The observed-relations plan records the
  reproduced iteration gap and earlier source/harness failures. Complete-bound
  trap summaries/header/completeness, actual commands, remaining collectors and
  every original implementation/final gate remain open. Native Linux stays
  batched; no push or dispatch.

- The final drop-production source at `1e4b9248` now passes actual authored-main
  impact-families, impact-sdk-vector, catalog-ordinary, pack-ordinary,
  pack-sources and pack-sdk under both Windows profiles: twelve selected
  profile/group combinations, no selected skips. Separate low parent commands
  retain every verifier assertion and authenticate the same generated-C/profile/
  toolchain/image digests; product timeouts are unchanged. See
  `docs/ops/windows-candidate-atlas-drops-cli-20261106.md` for exact timings,
  scope and failed-parent accounting. This closes these drop command selections,
  not all 41 CLI groups, remaining collectors or any broader/final gate.

- Fairy Complete-bound v1/v2/v3 production now retains canonical executable drop
  sites in ordinary, generic and concrete cleanup summaries, counts calls plus
  drops and reports only the newly collected drop class exact. Verified batch
  CFG lowering removes repeated whole-IR proofs per summarized function. Windows
  low/optimized collector and source-version roots pass 142 native processes
  and 94 independent current maps in 844.528s, including SDK Vec, current query/
  join admission and exact work/count bounds. Authored check/locked (671 files,
  8,330 MIR functions), selected Go query/pack regressions, scoped vet and format/
  LF/diff pass. The six actual full-main command selections now pass both
  profiles in the separate receipt above; see also
  `candidate-atlas-observed-relations-20261106.md`. The remaining collectors and
  every original broader/final gate remain open; native Linux stays batched.

- Fairy verified body/cleanup CFG drop-site collection now preserves executable
  reachability, operand source spans and canonical deduplication under independent
  work/count bounds. Fifteen actual source fixtures plus ten controls per Windows
  profile pass fifty native processes against independent Go facts in 252.399s.
  Authored check/locked (671 files, 8,317 MIR functions), scoped vet, format/LF/diff
  pass. Complete-bound map/summary/count/completeness production and command
  comparisons are required next; the collector does not claim production coverage.
  See `candidate-atlas-observed-relations-20261106.md`. All original broader and
  final gates remain open; native Linux stays batched, no push or dispatch.

- Fairy actual `experimental atlas-pack overview/show/around` now derives exact
  requested v1/v2/v3 maps from fresh Complete-bound source, selects one unique
  current package and emits canonical no-newline packs without a Go fallback.
  Four of 41 full authored-main verifier registrations pass both Windows
  profiles: 200 native processes across ordinary, source, SDK and help groups.
  Output/selection bounds, selectors, source-family refusals, stale/locked and
  unlocked refresh, stdout failure, generics, cleanup and SDK Vec pass against
  independently scoped original Go facts and checksums. Authored check/locked
  (670 files, 8,305 MIR functions), existing Go pack regressions and scoped vet/
  format/LF/diff pass. See the Context Pack plan for timings, digests and failed
  oracle attempts. Broader production/non-call families, generated consumers,
  other full-CLI verifiers and every original implementation/final gate remain
  open. Native Linux stays batched; no push or dispatch.

- Fairy current-admitted Context Pack around now traverses canonical local
  adjacency with separate relation/visited-node ledgers, per-depth membership,
  ordered frontiers and exact relation-before-node refusal/omission behavior.
  Eighteen real source/package/schema combinations pass 6,038 independent Go
  encoder/checksum Windows low/optimized observations in 610.390s. The shared
  assembler's show regression passes 260 observations in 41.285s. Authored
  check/locked (668 files, 8,300 MIR functions), scoped vet/format/LF/diff pass.
  Fresh source-backed actual atlas-pack dispatch and all original implementation
  and final acceptance gates remain required; see the Context Pack plan.

- Fairy current-admitted Context Pack show now performs exact mandatory
  context/subject selection, canonical incoming/outgoing/both relation selection
  and omission counts under independent record/relation/selection/output/work
  limits. Adjacency lists and a lazy heap merge retain self-call deduplication.
  Fifteen actual source/package/schema combinations pass 2,420 independent Go
  encoder/checksum Windows low/optimized observations in 156.692s. Authored
  check/locked (667 files, 8,294 MIR functions), scoped vet/format/LF/diff pass.
  The Context Pack plan retains around traversal/packs, source-backed actual
  dispatch and every original implementation/final acceptance gate.

- Fairy current-admitted relation projection and canonical O(R log R) ordering
  now preserve all ten kinds, both specialization endpoints, cleanup ownership,
  optional spans/details, ABI/link names and escaping borrows. Fifteen actual
  source/package/schema combinations pass 210 Windows low/optimized native
  controls and 130 independent Go relation comparisons in 28.754s. Reversed
  input, retained duplicates and exact work boundaries pass. Authored check/
  locked (663 files, 8,252 MIR functions), scoped vet/format/LF/diff pass. The
  Context Pack plan retains show/around selection/traversal/assembly and actual
  dispatch plus every original implementation/final acceptance gate.

- Fairy exact-current query-node projection now matches original declarations,
  recursive types, scalar/aggregate specialization and cleanup counts across
  all map families. The twelve actual source/package/schema combinations pass
  144 Windows low/optimized native controls and 130 independent Go node byte
  comparisons in 25.382s, including mandatory work and out-of-range refusals.
  Authored check/locked (661 files, 8,237 MIR functions), scoped vet/format/LF/
  diff pass. Relations and show/around selection/traversal and actual CLI remain
  required; the Context Pack plan retains every original implementation/final gate.

- Current-source node comparison reproduces and fixes Go query declaration/type
  nil/empty drift across ordinary v1, generic v2 and aggregate/cleanup v3 map
  transport. Detached nested snapshots now retain canonical map-restored
  spelling before show/around selection measurement. Actual source/map round-
  trips, both query kinds and mandatory byte boundaries, full atlasmap/contextpack
  suites, existing Go pack CLI and scoped vet/format/LF/diff pass. See the
  Context Pack plan. Fairy node projection, query selection and actual dispatch
  and every original implementation/final gate remain separate required work.

- Fairy current-admitted Context Pack overview v1/v2/v3 now preserves explicit
  snapshot projection, mandatory package and ordered record/byte selection,
  exact omissions and canonical body/envelope checksums. Independent Go encoders
  and checksum recomputation pass 976 Windows low/optimized observations in
  86.667s, including cumulative one-short/exact/one-more and atomic refusal
  boundaries. Authored check/locked (659 files, 8,227 MIR functions), scoped vet,
  format/LF/diff pass. See the Context Pack plan. Show/around and actual CLI,
  generated consumers and all original implementation/final gates remain open.

- Context Pack implementation reproduced and fixes Go overview header nil/empty
  array drift across unchanged map transport. Four detached header arrays now
  follow the existing canonical empty-array contract before selection costs are
  measured. Real source/map round-trip, exact/one-short/one-more mandatory
  bounds and caller immutability controls pass, as do the complete atlasmap/
  contextpack suites, experimental Go pack commands and scoped vet/format/LF/
  diff checks. Exact failed/passed evidence is in the Context Pack plan. This
  Go prerequisite does not finish Fairy overview/show/around, actual dispatch
  or any original implementation/final gate.

- Fairy atlas-pack overview/show/around argument selection now matches 317 Go
  parser cases and three invalid queries per Windows profile, 640 native
  observations (33.583s). Exact versions, lexical ten-relation filters, action-
  specific required/forbidden flags, positive limits and around depth 0..64,
  UID/source selectors and the full 43-word invocation pass without workspace
  I/O. See `candidate-context-pack-20261106.md`. Current-admitted pack projection,
  selection/traversal/checksums and actual CLI remain required; this does not
  close atlas-pack or any original broader/final queue item.

- Fairy actual `experimental atlas-calls list` now derives requested v1/v2/v3
  maps from one fresh Complete, reauthenticates every package, joins exact current
  calls and emits canonical v0/v1/v2 catalogs. Argument/help, product error codes,
  selector/join/output refusal, stale/locked/unlocked behavior and stdout failure
  pass in the authored full-main Windows low/optimized images. Three appended
  calls groups and help pass both profiles; existing symbols-source regression
  also passes optimized. The three bounded selections pass in
  369.726s/271.206s/164.136s, 117 native observations total. This covers four of
  thirty-eight registrations at both profiles, with a fifth optimized only.
  Authored check/locked, Go calls/symbols regression, scoped vet, formatting/LF/
  diff pass. See `candidate-workspace-calls-20261106.md` for exact scope/digests.
  Atlas-pack, remaining full CLI and generated consumers, broader source/non-call
  families and every original implementation/final acceptance gate remain open.
  The complete objective remains active; native Linux stays batched, no push or
  dispatch.

- Fairy joined-call catalog v0/v1/v2 now retains actual join limits, exact source/
  specialization/cleanup identities, canonical O(N log N) ordering/deduplication,
  whole-ledger usage, current context and independently checked SHA-256 output.
  Final Windows low/optimized native comparisons pass 318 observations; the
  combined parser/generic-join/impact/catalog command passes in 183.703s with no
  selected skips. Authored check/locked, scoped vet, formatting/LF/diff pass.
  See `candidate-workspace-calls-20261106.md`. This is the private encoder only;
  fresh source-backed actual calls dispatch, atlas-pack and all original broader
  implementation and final acceptance gates remain required.

- Fairy `atlas-calls list` argument selection now matches 87 Go parser cases and
  three invalid actions per Windows profile, 180 native processes total
  (13.161s). It retains catalog 0/1/2 to map 1/2/3, exact source IDs, direction,
  canonical bounds and locked selection without I/O. Authored check/locked,
  scoped vet, formatting/LF/diff pass. Joined-call output and actual dispatch are
  required next; see `candidate-workspace-calls-20261106.md`. All original
  broader implementation/final gates remain open; this is only a prerequisite.

- Fairy now dispatches actual `experimental atlas-symbols list` through fresh
  Complete-bound v1/v2/v3 production, exact UID/current admission and canonical
  catalog output. Arguments/help precede I/O; locked requests preserve source
  bytes and unlocked requests refresh the ordinary lock. Disjoint final Windows
  low/optimized full-main selections pass all 174 native observations across
  five of thirty-five registrations, including source/generic/aggregate/SDK,
  page/byte boundaries, errors, existing impact and help. The original combined
  fifteen-minute command failed on its parent budget and is preserved as failed;
  bounded replays pass in 605.333s/282.344s/172.973s/179.152s. Authored check/locked,
  scoped vet, original Go command regression, formatting/LF/diff checks pass.
  See the symbol-catalog plan for exact IDs, coverage, failed attempts and costs.
  The other thirty full-CLI groups, actual calls/pack commands, Linux dispatch,
  broader source/collection families and all original final gates remain open.
  This closes the admitted symbols command only, not the full objective.

- Version-selected Fairy source production now includes used/unused SDK Vec v3
  and ordinary SDK `std::num`/`std::bytes` under v1/v2/v3. Refusal-only source
  preflight avoids expensive unsupported-family compilation without bypassing
  Complete authentication. The two disjoint Windows low/optimized replays pass
  in 304.204s/462.263s, covering nine families, 64 native processes and 78
  independently admitted maps. Authored check/locked, scoped vet, formatting,
  LF audit and diff checks pass. See the symbol-catalog plan for scope, limits
  and exact earlier-attempt accounting. Actual symbols dispatch, calls/pack and
  all original implementation/final gates remain open; this does not finish
  the broader Atlas or full implementation objective.

- Explicit Fairy source-selected v1/v2/v3 factories now authenticate their
  Complete source closure, enforce each source family and render matching
  facts/header/envelope/current admission without map conversion. Ordinary v2
  generic-identity presence now matches Go's actual compilation lane. The
  six-family Windows matrix passes both profiles in 288.017s: 46 native processes,
  48 independently admitted maps and exact source-backed catalog comparisons.
  Authored check/locked, scoped vet, formatting and diff checks pass. An extra
  SDK attempt hit the generic helper's one-minute diagnostic cap and is not
  acceptance; explicit SDK early selection/production and the actual symbols
  dispatcher remain next. The symbol-catalog plan records failed attempts,
  retained unchanged-input evidence and all remaining final gates.

- Current-admitted Fairy catalog v1/v2/v3 page selection and canonical output
  now preserve exact filters, full-ID ordering/cursor membership, original
  declarations, omissions and independent selection/final byte limits. The
  page/parser Windows low/optimized replay passes in 69.969s with 800 native
  processes (584 pages plus 216 parser controls), independent Go byte/checksum
  comparisons and atomic refusal controls. Authored check/locked, scoped vet,
  formatting and diff checks pass. See the symbol-catalog plan for precise scope
  and earlier-attempt accounting. Schema-specific Complete source production,
  actual symbols dispatch, calls/pack and all final gates remain required.

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
