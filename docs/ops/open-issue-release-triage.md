# Open issue release triage — 2026-09-16

## Owner-approved action update

The owner subsequently authorized pruning low-value issues and resolving concrete
remaining work. The historical 19-issue snapshot below is retained as the review
basis; the current open count after this action is **10**, not 19.

- Closed as **not planned**, never as implemented: #28, #29, #30, #32, #33,
  #34, #41, #42 and #43. Their bodies preserve original proposals and now explain
  the missing concrete workload/target/codec or host contract and a reopening
  condition. Candidate ADRs, existing tools and safety rejection gates remain intact.
- Reconciled #18: fourteen already-completed children are checked with a bounded
  scope caveat; nine deferred proposals are explicitly not implemented; the stale
  branch baseline and formerly-open PR guidance are corrected. The tracker stays open.
- Reconciled #49: only the existing owner/evidence matrix acceptance item is
  checked. Same-commit CI and the second-laptop report are linked without claiming
  toolchain-free deployment closure or complete target maturity.
- Retained concrete work: #27 measurement evidence; #31 Linux profile; #36 portable
  paths; #37 resource arenas; #45 resource-bearing enums; #46 broader burn;
  #47 public tagged C ABI; #49 Windows maturity. These are **not resolved** by pruning.
  #18 and #1 remain tracking/maintenance issues.

GitHub states and `not_planned` closure reasons were reread after mutation. No
compiler changes, issue deletion, push, release or visibility change occurred.
Next implementation/evidence unit is #27: reuse the existing growth probes, but
verify public scalar-workspace coverage before treating retained private-probe
measurements as satisfying the entire issue. That work was not executed in this unit.

## Historical review snapshot

Reviewed local baseline: `7196f699ce86d751874bb8998efafc85e4c5b204`.
GitHub API snapshot: 19 open issues, zero open PRs in the returned collection.
This is a source-publication scope review, not implementation of the backlog,
a security audit, a new test run or authorization to close issues or publish.

The proposed first public scope is experimental Go Stage 0 source with explicit
Windows/MSVC prerequisites and documented unsupported capabilities. It is not
complete Windows distribution maturity, supported Linux, production services,
self-hosting or a deployable contract target. The judgments below depend on that
scope; widening claims reopens the corresponding gates.

## Evidence sources

- [Roadmap gates](../product/01-roadmap.md#later-capability-gates) and
  [product specification](../product/02-spec.md) own admission boundaries.
- [Production workload admission](../adr/0328-production-workload-capability-admission-gates.md)
  keeps Linux promotion, cancellation, interchange and operational context candidate-only.
- [Contract IDL boundary](../adr/0340-experimental-chain-neutral-contract-idl.md)
  separates existing generators from codec, signing, source calls and deployment.
- [Windows maturity matrix](../abi/windows-target-maturity-v0.md) owns #49's
  complete-target and deployment requirements; [readiness](parser-public-readiness.md)
  records the narrower ZIP, second-laptop and successful `34fe041c` CI evidence.
- Source spot checks: [generic growth probe](../../internal/compiler/generic_aggregate_growth_test.go),
  [declaration-owned burn guard](../../internal/semantic/resource_abilities.go),
  [enum rejection tests](../../internal/semantic/semantic_test.go) and
  [target evidence invariants](../../internal/target/spec.go).

Issue bodies were compared with these owning documents and focused source checks.
This does not independently reproduce every acceptance criterion. Closed GitHub
state below is tracker evidence, not proof of unrestricted feature completeness.

## All 19 open issues

| Issue | Current assessment | Experimental source publication | Disposition |
| --- | --- | --- | --- |
| [#1](https://github.com/0disoft/fairyc/issues/1) Dependency Dashboard | Maintenance tracker; body reports no open/pending update branches, not a vulnerability clearance | Not itself a blocker | Keep open; do not toggle the Renovate trigger |
| [#18](https://github.com/0disoft/fairyc/issues/18) Roadmap tracker | Stale child checkboxes and historical baseline; see reconciliation below | Correct misleading tracking before inviting contributors | Keep tracker open; propose metadata reconciliation, not mass implementation |
| [#27](https://github.com/0disoft/fairyc/issues/27) Generic growth evidence | Specialization already implemented; private aggregate growth test exists, but retained public-path measurement deliverable is not established | Does not block limited source preview without growth/performance claims | Keep open as evidence work; do not duplicate the existing probe |
| [#28](https://github.com/0disoft/fairyc/issues/28) IDL source calls | Generator exists; source-call/codec/MIR integration remains outside admitted boundary | Not a blocker while unadvertised | Future feature |
| [#29](https://github.com/0disoft/fairyc/issues/29) Signing/reentrancy | Authority and callback proof remain closed by IDL contract | Not a blocker while no signing/contract execution is promised | Future feature; required before such exposure |
| [#30](https://github.com/0disoft/fairyc/issues/30) Deployable contract target | Kernels and probes do not constitute an admitted deployable target | Not a blocker for source preview | Future target; prerequisite for deployment claims |
| [#31](https://github.com/0disoft/fairyc/issues/31) Linux promotion | Linux CI/probes exist; production-target promotion remains a separate gate | Not a blocker for Windows-first preview | Future target promotion |
| [#32](https://github.com/0disoft/fairyc/issues/32) Cancellation/task ownership | ADR 0328 candidate gate, not an admitted executor | Not a blocker without concurrency promises | Future feature |
| [#33](https://github.com/0disoft/fairyc/issues/33) Data interchange | ADR 0328 requires a concrete bounded workload before general admission | Not a blocker | Future feature |
| [#34](https://github.com/0disoft/fairyc/issues/34) Operational context | ADR 0328 candidate capability contract, not a bundled service framework | Not a blocker | Future feature |
| [#36](https://github.com/0disoft/fairyc/issues/36) Portable filesystem paths | Existing File subset is narrower than Windows/POSIX path parity | Not a blocker with platform limits disclosed | Future portability slice |
| [#37](https://github.com/0disoft/fairyc/issues/37) Resource-aware arenas | Current Region/arena evidence does not admit generalized resource drop lists | Not a blocker while excluded | Future ownership feature |
| [#41](https://github.com/0disoft/fairyc/issues/41) Event emission | Named effects do not provide an SDK/MIR/target operation | Not a blocker while excluded | Future contract capability |
| [#42](https://github.com/0disoft/fairyc/issues/42) Block context | Effect vocabulary does not provide a runtime context capability | Not a blocker while excluded | Future contract capability |
| [#43](https://github.com/0disoft/fairyc/issues/43) External contract calls | Codec, reentrancy and failure atomicity remain separate unadmitted work | Not a blocker while excluded | Future contract capability |
| [#45](https://github.com/0disoft/fairyc/issues/45) Resource-bearing user enums | Built-in Option/Result support does not generalize to user enums; rejection fixture remains | Not a blocker with limitation retained | Future ownership feature |
| [#46](https://github.com/0disoft/fairyc/issues/46) Broader burn | Source guard still restricts burn to direct declaration-owned no-drop values and rejects owned children | Not a blocker with restriction retained | Future ownership feature |
| [#47](https://github.com/0disoft/fairyc/issues/47) Public tagged C ABI | Private tagged layout is not public `repr(C)` enum-field admission | Not a blocker without that ABI claim | Future ABI feature |
| [#49](https://github.com/0disoft/fairyc/issues/49) Windows maturity | Evidence matrix exists; many rows remain partial/blocked | Not a blanket source-preview blocker; blocks complete maturity/distribution claims | Keep open; reconcile only completed evidence work |

Summary: 15 future capability issues, two partially fulfilled evidence issues
(#27/#49), one stale roadmap tracker and one continuing dependency dashboard.
No open issue is recommended for closure from this review alone. Absence of a
confirmed release-blocking bug in this bounded review is not a security assurance.

## Tracker reconciliation proposals

### #18: separate closed child status from capability maturity

GitHub reports #19, #20, #21, #22, #23, #24, #25, #35, #38, #39, #40, #48,
#50 and #51 closed as completed, while #18 still shows their entries unchecked.
#26 and #44 are already checked. Marking the fourteen closed children as closed
would synchronize tracker state; it must not claim complete persistence, native
state access, editor tooling or self-hosting beyond each child's admitted slice.
Keep the fifteen future capability issues and #27/#49 open.

The instruction to branch/rebase from historical `aa8f12f0` should be replaced
with current `origin/main`, retaining the old SHA only as historical context.
References to formerly open PRs should also be refreshed: the current open-PR
collection is empty. No remote checkbox, body, label or issue state was changed.

### #49: distinguish compiler use from generated-program deployment

The first acceptance item, a checkable matrix with owner and evidence paths,
already has a direct source at `windows-target-maturity-v0.md`; it can be marked
fulfilled as documentation work. Current same-commit CI supports hosted execution,
but not all broader bidirectional ABI, linker, unwind or debugger requirements.

The second laptop used an installed MSVC/Windows SDK toolchain. This proves the
same compiler ZIP can be used on another Windows development host, not that
generated executables and their UCRT/VCRuntime closure run without development
tools. Do not mark clean-machine portability or complete distribution maturity
proven, and do not promote target flags from that report.

## Actual next public-launch decisions

1. Reconcile #18 and the completed matrix item in #49 without closing their
   unfinished parent scopes; the proposals above are ready for a separate tracker update.
2. Establish a confirmed private vulnerability-reporting route and its owner.
   [README](../../README.md) currently discloses that this is unconfirmed.
3. Resolve the unassigned release owner and escalation fields in
   [release policy](release.md), and explicitly approve experimental source-only
   visibility. Completing every future language feature is not a substitute.
4. Keep binary release, full target promotion and production claims separate;
   their evidence requirements remain binding when those claims are selected.

Validation: read-only GitHub issue state/body review, source/document comparisons,
local Markdown link and diff checks, and scoped strict contract validation.
No compiler/test behavior or version changes. No fresh full suite, benchmark,
security scan, push, issue mutation, repository visibility change or release.
