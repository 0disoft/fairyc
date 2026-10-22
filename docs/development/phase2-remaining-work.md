# Phase 2 remaining work

Audit: 2026-10-23 at `952105fa`. Tested implementation:
`7f7362b87c231c30ea89f3907396957f0d599032`.

## Findings at the audit baseline

The representative runtime/SDK minimum in the [roadmap](../product/01-roadmap.md#phase-2-first-hosted-executable-slice)
has execution evidence. This audit found no additional missing implementation in that minimum.
That does not establish the full target's conformance: the [C ABI contract](../abi/c-abi-v0.md#target-status)
requires every declared fixture to pass without skips; distribution is a separate `supported` gate.

| Classification | Finding and owner |
| --- | --- |
| Satisfied minimum | Thin runtime, SystemHeap, Arena/Region, Buffer, text/String/CString and Linux File have raw and generated-source observations in the [coverage inventory](linux-runtime-coverage.md#phase-2-contract-to-fixture-review). The existing allocation-failure, File-progress and trap-receipt checks should be reused. |
| Execution evidence missing | Directly nested resource cleanup is implemented by [resource layout](../../internal/cbackend/resource_layout.go) and [drop glue](../../internal/cbackend/drop_glue.go), but its dedicated native tests are Windows-only. The 55-case emission manifest has no `nested-resource` feature case. See E1/E2 below. |
| Backend implementation limits | [Resource layout](../../internal/cbackend/resource_layout.go) rejects zero-field resources; [resource analysis](../../internal/cbackend/analysis_resources.go) rejects conditional drop flags. [TestEmitRejectsNestedResourceTag](../../internal/cbackend/emit_test.go) requires rejection of `Option<Option<Token>>`. These are explicit capability boundaries, not newly reproduced runtime or SDK defects. |
| Outside the minimum | Extra Vec operations, Windows filesystem parity, portable paths, sockets, async, custom allocators and distribution. Existing contracts exclude or separately admit these; this audit creates no implementation requirement for them. |

The manifest's broad `deferred` prose is not a current implementation checklist. Later decisions
already admit nonempty cleanup, directly nested resources and Region lowering. In particular,
[ADR 0392](../adr/0392-nested-resource-c-storage-and-cleanup.md) distinguishes directly nested
resource fields from unsupported nested owning tags. Neither can stand in for the other's evidence.

## Historical evidence baseline

- [Frontend run 39805612550](https://github.com/0disoft/fairyc/actions/runs/39805612550):
  all nine jobs and the aggregate run succeeded.
- [Linux Go job 119918006715](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715):
  `fairyc.phase2-linux-evidence/v3`, `complete:true`, exactly 28 compiler/profile cases,
  `missing:[]`, `skipped:[]`, `errors:[]`. Trap progress covers six scenarios in each of four cases.
- [Native job 119925063330](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119925063330):
  v38, 55 cases, 220 differential and 220 drop-trace executions, 110 sanitizer executions,
  56 ABI crossings, four layout probes and four SDK storage executions.
- Local audit checks passed: focused resource/backend tests and
  `TestPrivateVectorNestedResourceDropTrace` under its two MSVC profiles. No Linux test was rerun locally.

## Cleanup acceptance criteria (implemented below)

| ID | Existing implementation and test | Missing Linux observation / completion criterion |
| --- | --- | --- |
| E1 | [TestPrivateVectorNestedResourceDropTrace](../../internal/compiler/nested_resource_trace_test.go): a Box owns two Leaf values; Box cleanup also owns a temporary Leaf. | Interpreter and generated C return 42 and agree on four ordered, source-linked drops: Box, cleanup temporary, last field, first field. Require GCC/Clang low/optimized execution and sanitizer coverage of the same source. |
| E2 | [TestPrivateVectorNestedResourceDrops](../../internal/compiler/nested_resource_test.go), through the Windows-only helper in [vector_storage_test.go](../../internal/compiler/vector_storage_test.go): parent cleanup mutates a child; active/inactive Option/Result paths select cleanup. | Observe the existing independent `10,2,99` sequence three times, with no inactive-payload cleanup, under GCC/Clang low/optimized. E1 alone does not establish mutation visibility or all tagged paths. |

These are admitted cleanup semantics under ADR 0392, not new SDK APIs. An existing Windows proof
does not attest Linux execution. Conversely, adding a Linux fixture does not implement a missing
language feature or remove a backend capability restriction.

## E1 implementation plan (completed below)

Add the existing pure Box/two-Leaf/temporary-Leaf scenario to the emission manifest and native lane.
Retain independent golden resource types and binding spans so interpreter and C cannot agree on
the same incorrectly omitted drop. Production C stays uninstrumented; only the existing trace
variant records drops. Check the return and exact trace in all four compiler/profile combinations,
and run the production unit through the existing sanitizer path.

With one added case, expected totals become 56 cases, 224 differential/drop executions each and
112 sanitizer executions. These are future acceptance counts, not audit results. The focused v3
set stays at 28 cases; no new workflow, runner, product API or report schema is needed. Add
`nested-resource` to the case-feature schema allowlist. Preserve the original 52-case transport hash
in `TestEmissionTransportAcrossResponsibilitySplit`; the new fixture needs independent assertions,
not a rebaseline of unchanged C. Correct stale deferral wording
for this newly observed case without claiming that E2 or nested owning tags are covered.

## Phase 2 completion decision

Use a finite mapping of declared requirements to source owners, fixtures and no-skip hosted results.
The two `Evidence.Missing` strings in [TargetSpec](../../internal/target/spec.go) are static labels,
not executable acceptance lists. Their presence alone is not a reason to add more failure tests.
E1/E2 close the identified cleanup observations; the already satisfied runtime/SDK minimum needs
no blanket rerun. Broader capability rejections require their owning contract and fixture decision
before they can support a wider target claim; this audit neither narrows that claim nor adds them
to the representative SDK minimum.

Keep Phase 2 `partial`, `complete-runtime` and `complete-standard-library` present, and `emittable`,
`conformant`, `supported` false. Removing a marker requires an explicit evidence-backed assessment
of its declared scope; completing E1 alone is insufficient.

## Implementation update (2026-10-23)

E1 is implemented at `ecd0e909`: 56 emission cases, independent parent/temporary/reverse-child
expectations, unchanged original 52-case transport hash, and no production trace instrumentation.
Different FirstLeaf/LastLeaf types make sibling order independently observable.
E2 is implemented at `893cfbc9`: the existing `10,2,99` observer runs under GCC/Clang low/optimized
and both MSVC profiles. Linux low-profile cases additionally require one ASan+UBSan execution per
compiler. These two sanitizer executions belong to the focused Go test, not the v38 native count.
Local fixture tests, both MSVC cases, and the 12 CI-helper tests pass. Hosted execution passes
in the receipt below.
The focused collector now requires v4/32 cases; v3/28 remains historical.

Phase 2 verdict: do not promote the full target merely because its representative runtime/SDK
minimum and these cleanup observations pass. The complete target claim must account for the
backend's explicitly rejected shapes (zero-field resources, conditional drop state and nested
owning tags), the source-level foreign-call boundary, and every declared target fixture. Existing
closed capabilities are not newly admitted by this audit. The two completeness markers and all
three maturity flags stay unchanged; this is a scope/evidence decision, not a request to add more
random runtime failure tests or implement excluded SDK APIs.

## Hosted cleanup receipt (2026-10-23)

[Frontend run 39817319718](https://github.com/0disoft/fairyc/actions/runs/39817319718) at
`5f62c9c3e8d4ceeed8830c96b82215c96784ea29` completed successfully, with all nine jobs passing.

- E1: [native job 119965856725](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119965856725)
  records `fairyc.native-c-report/v38`, 56 cases including `nested-resource-drop`, 224 outcome and
  224 exact drop-trace comparisons, and 112 ASan+UBSan executions. Its Ubuntu 24.04 amd64 toolchain
  is glibc 2.39, Clang 18.1.3 and GCC 13.3.0, with low/optimized profiles. It also records 56 ABI
  crossings, four layout probes and four SDK storage executions.
- E2: [Linux Go job 119959193676](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119959193676)
  records `fairyc.phase2-linux-evidence/v4`, `complete:true`, exactly 32 cases and empty
  `missing`, `skipped` and `errors` lists. The four nested-cleanup mutation cases pass, and the
  log separately confirms one sanitizer execution per compiler. These two executions are not
  part of the native report's 112.
- [Windows Go job 119959193834](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119959193834)
  passes the full core suite, including the existing MSVC low/optimized mutation observer.

E1 and E2 are closed. The remaining Phase 2 decision is the complete declared target scope
described above; both completeness markers and all maturity flags remain unchanged.
