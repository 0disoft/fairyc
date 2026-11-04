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
- [ ] Fairy `experimental atlas-edit`: current identity and source checks,
  single-body preview/apply, failure-atomic publication and native CLI tests.
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
