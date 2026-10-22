# Phase 2 target scope assessment

- Date: 2026-10-23
- Inspected checkout: `3b6f58c9de4ee12f212e68b152d374d055b81b55`
- Scope: `x86_64-linux-gnu`; roadmap Phase 2, runtime minimum, C ABI and the identified resource boundaries
- Decision: the representative runtime/SDK minimum passes; full target conformance remains incomplete

## Required scope and existing evidence

The [roadmap](../product/01-roadmap.md#phase-2-first-hosted-executable-slice) requires both a
representative runtime/library minimum and a `conformant` target. These are different conditions.
The [C ABI contract](../abi/c-abi-v0.md#target-status) does not promote the full target from its
current constrained emission subset. The [coverage inventory](linux-runtime-coverage.md#phase-2-contract-to-fixture-review)
maps the runtime minimum to implementation and probe owners.

| Required family | Implementation / fixture owner | Current result |
| --- | --- | --- |
| Target facts, toolchain and C-driver linking | [TargetSpec](../../internal/target/spec.go), [toolchain contract](../abi/linux-toolchain-v0.md), [native runner](../../internal/cbackendnative/runner.go) | Ubuntu 24.04 amd64, glibc 2.39, GCC 13.3.0 and Clang 18.1.3 provision and execute the declared low/optimized profiles. |
| Thin runtime, traps, entry and memory helpers | [runtime](../../internal/cruntime/runtime.go), [scalar C](../../internal/cruntime/stage0_scalar.c), [trap progress test](../../internal/cruntime/trap_receipt_progress_test.go) | Native differential and focused trap-receipt execution pass. |
| SystemHeap, Buffer and Arena/Region | [stdlib probes](../../internal/cbackendnative/probes_stdlib.go), [public SDK storage caller](../../internal/cbackendnative/sdk_storage_source.go) | Raw and generated-source execution pass, including public byte placement and allocator failure observations. |
| UTF-8 views, String, CStr and CString | [text probes](../../internal/cbackendnative/smoke_text_views.go), [owned text probes](../../internal/cbackendnative/smoke_owned_text.go) | Raw and generated-source observations pass for the representative minimum. This does not cover every public wrapper or future text API. |
| Linux File core and prepared publication | [Linux File](../../internal/cstdlib/stage0_linux_file_portable.c), [public File probe](../../internal/cbackendnative/smoke_files.go), [publication test](../../internal/cstdlib/linux_publication_test.go), [progress test](../../internal/cli/linux_file_progress_test.go) | Required open/read/write/sync/consuming close/drop, exact read, single-file removal and prepared replacement have execution evidence. |
| Scalar/tagged outcomes, active payload and drop order | [emission manifest](../../spec/fixtures/c-emission/manifest-v0.json), [mutation observer](../../internal/compiler/nested_resource_test.go) | E1/E2 pass: 56 cases, 224 outcome and 224 exact drop-trace comparisons, 112 sanitizers, plus the focused mutation/sanitizer observations. |
| Target layout and bidirectional C ABI plans | [layout manifest](../../spec/fixtures/layout/manifest-v0.json), [ABI manifest](../../spec/fixtures/c-abi/manifest-v0.json), [ABI probe generator](../../internal/cbackendnative/abi_probe.go) | All 16 layout and 17 ABI manifest cases run through their Go runners; the native report records four layout probes and 56 cross-compiler ABI executions. These generated C probes do not establish all Fairy-source foreign-call forms. |

The full [Frontend run 39817319718](https://github.com/0disoft/fairyc/actions/runs/39817319718)
at `5f62c9c3e8d4ceeed8830c96b82215c96784ea29` passed all nine jobs.
Its [native job](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119965856725)
records the 56-case matrix. Its [Linux Go job](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119959193676)
records v4, 32 required compiler/profile cases, `complete:true`, and empty missing/skipped/error
lists. The two nested-mutation sanitizer executions are separate from the native report's 112.
The subsequent `3b6f58c9` commit changes documentation only.

## Contracted C ABI gaps at the assessment baseline

The language/ABI contract includes more than the currently executable source subset. Passing
ABI classification and generated C-to-C probes cannot fill the following source-to-C gaps.
They block a claim about the complete contracted target; they are not newly observed runtime bugs.

| ID | Contract / current implementation | Required completion |
| --- | --- | --- |
| F1 — by-value record arguments | [C-representable types](../abi/c-abi-v0.md#c-representable-types) includes record arguments and returns. [foreignCType](../../internal/cbackend/foreign_types.go) admits record results but rejects record arguments. `TestEmitRejectsReprCStructForeignParameter` confirms that a typed, verified source call still cannot emit. | Admit the already-supported nonempty record shapes as foreign arguments and check actual Fairy-source calls against separately compiled C callees in both Clang/GCC directions and both profiles. Cover register and MEMORY-class argument transport and preserve record-result behavior. |
| F2 — remaining C-representable field families | The same contract includes fixed arrays, explicitly based C enums, floating Fairy scalars and the other fixed-width integers. [aggregate layout resolution](../../internal/cbackend/aggregate_types.go) keeps repr(C) array fields closed, rejects enum fields and admits only its listed Fairy scalar subset. `c::float`/`c::double` transport is distinct from Fairy `f32`/`f64`. | Connect each contracted field family through source typing, verified MIR, emission, layout assertions and real native crossings. Implement one family at a time; layout-engine support alone is insufficient. |
| F3 — bounded C callbacks | [Functions and callbacks](../abi/c-abi-v0.md#functions-and-callbacks) specifies a non-capturing synchronous same-thread call-bounded form. [foreignCType](../../internal/cbackend/foreign_types.go) has no callable parameter transport. The layout fixture's function-pointer case does not establish this source path. | Define and implement the exact source-to-C callable transport and user-data/lifetime checks, then execute the bounded callback. Retained/asynchronous callbacks remain excluded. |

## Resource boundaries: not three new runtime tasks

| Boundary | Owning decision and source | Classification |
| --- | --- | --- |
| Zero-field resources | [ADR 0033](../adr/0033-flat-i32-resource-c-transport.md) explicitly excludes empty resources from its transport. [MIR construction](../../internal/mir/lower_aggregates.go) has no empty-resource support, before the separate [backend layout guard](../../internal/cbackend/resource_layout.go). | Explicit prototype implementation deferral, not an uncovered SystemHeap or SDK path. Any broader resource admission must address both MIR and C storage; an empty cleanup body on a nonempty resource is already supported and is different. |
| Nested owning tags, e.g. `Option<Option<Token>>` | [ADR 0034](../adr/0034-resource-tagged-c-transport.md) leaves nested tags to a separate gate; [the rejection test](../../internal/cbackend/emit_test.go) still requires a capability error. [ADR 0392](../adr/0392-nested-resource-c-storage-and-cleanup.md) admits directly nested resource fields, not arbitrary tagged-resource fields. | Explicit emission-subset deferral. Direct nested-resource E1/E2 evidence does not admit this shape. A wider resource profile needs its own representation and recursive active-payload cleanup evidence. |
| Conditional root drop flags | [Resource ownership](../language/0001-values-and-resources.md#affine-resource-ownership) rejects source-visible `MaybeLive` roots. [Drop elaboration](../mir/drop-elaboration-v0.md#obligations) reserves flags for compiler-owned partial construction or later specified conditional obligations; [backend analysis](../../internal/cbackend/analysis_resources.go) rejects flagged resource drop. | Not a missing current source feature. Do not implement flags to make branch-dependent source ownership valid. Partial-construction admission must be assessed separately; this guard alone does not demonstrate a Phase 2 defect. |

## Scope exclusions and evidence limits

Linux directory lifecycle/enumeration, Windows filesystem parity, extended metadata, File-to-Buffer
operations and later Vec views are outside the representative Phase 2 minimum. Some have Windows
implementations or Linux implementations without a focused receipt for every later operation;
neither fact makes them a missing minimum requirement. See the
[Linux inventory](linux-runtime-coverage.md#missing-linux-implementations) and
[runtime contract](../runtime/runtime-stdlib-v0.md#explicit-exclusions).

Portable paths, sockets, async, custom allocator protocols, freestanding support, C variadics,
C++ ABI, foreign unwinding and retained/asynchronous callback safety are explicit exclusions.
Maintained distributions belong to the separate `supported` gate. None is added as a Phase 2
implementation task by this assessment.

## Decision and next implementation

Keep Phase 2 `partial`, both completeness markers present and all three target maturity flags false.
The runtime/library minimum and E1/E2 are closed; F1–F3 identify concrete gaps against the stated
C ABI contract. The resource boundaries above remain explicit prototype deferrals, not newly
admitted features or additional minimum-runtime tasks. The two static completeness labels are not
an executable checklist or a reason to repeat the completed runtime failure tests. This assessment
does not silently narrow the C ABI contract or claim an exhaustive test of every SDK operation.

F1 and the F2-1 integer-array subset are implemented in the updates below; hosted Linux execution
remains pending. Other F2 families and F3 stay separate follow-ups; no callback or resource
representation is added by either update.

## F1 implementation update

[ADR 0476](../adr/0476-repr-c-struct-foreign-arguments.md) admits foreign arguments using the same
nonempty repr(C) record shapes already supported as results. Private record arguments still
produce an explicit capability error. The shared `foreignstructprobe` fixture now checks small,
large and nested mixed-class arguments, scalar neighbors, exhausted SysV integer registers,
unchanged caller copies, and the original small/large results. Existing Linux and Windows native
lanes consume this fixture without new jobs, reports or execution-count fields.

Local validation passed:

- `go test -mod=readonly -count=1` for `internal/cbackend`, `internal/foreignstructprobe`,
  `internal/cbackendnative` and `internal/cbackendfixture`; the original 56 emission cases remain
  valid, and private record arguments still fail explicitly.
- `go vet -mod=readonly` for the backend, shared probe and native-runner packages.
- `go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke`: correctness smoke passes with
  MSVC 14.51.36231. Its shared record probe executes four separate-object combinations:
  C11/C17 consumer modes, each at low/optimized profiles, all returning 42. The command also
  passes its existing target/workspace/runtime/layout/ABI checks. Optional measurements are skipped.
- Changed-document relative links and `git diff --check` pass.

Linux GCC/Clang directions and profiles are configured but have not run for this source revision;
local WSL discovery failed with `REGDB_E_CLASSNOTREG`. No Linux installation or host configuration
was changed. F1's hosted evidence remains open, and the earlier full CI receipt above cannot attest
these new checks. The full repository Go suite and hosted CI were not rerun locally. Phase 2 and
all target maturity flags remain unchanged; no DB, repository setting, workflow or runner changes.

The F2-1 update below implements the first field family. Batch the current commits for one
authorized push to collect F1 and F2-1 Linux receipts before marking that evidence complete.

## F2-1 integer array fields

[ADR 0477](../adr/0477-repr-c-integer-array-fields.md) admits fixed array fields of `u8`, `i32`,
`i64`, `u32` and `usize`, with lengths `1..=4096`, in nonempty acyclic repr(C) records.
Implementation commit `0171fe50` carries source typing, verified MIR and raw C member emission.
Whole-field reads and replacement copy elements between raw members and private array values;
nested replacement and record argument/result copies preserve value semantics. Raw-field
references/views, direct element stores/iteration, other element families and bare foreign arrays
remain closed.

The existing shared record probe now includes independently declared 64-byte and 8-byte C records
with arrays, size/alignment/offset assertions, every element family, caller-copy isolation and
whole-field replacement after a copied read. Existing native lanes consume the expanded source;
there is no new job, execution-count field or runner. The Windows probe now links the canonical
scalar runtime with its companion C unit for checked indexing; the Linux probe already links it.

Local validation passed:

- `go test -mod=readonly -count=1` for HIR, semantic, MIR, C backend, emission fixtures, shared
  probe and native-runner packages; focused compiler tests cover MIR codec round-trip, interpreter
  copy/replacement behavior and both target emission profiles.
- `go vet -mod=readonly` for HIR, semantic, MIR, C backend, compiler, shared probe and native runner.
- `go test -mod=readonly -count=1 -v ./tools/dev/fairyc-windows-host-smoke
  -run '^TestReprCArrayFieldNativeCrossing$'`: MSVC 14.51.36231 passes all four C11/C17
  low/optimized separate-object combinations, each returning 42 with empty output.

The initial full Windows smoke stopped at the expanded record probe because its runtime trap
symbol was not linked. After adding the canonical runtime, the focused native test above passed;
the entire smoke was not repeated. No product runtime implementation or link policy changed.
Changed-document relative links and `git diff --check` pass.

Hosted Linux GCC/Clang execution is pending for this expanded probe. Earlier F1 and runtime
receipts do not attest the array observations. The full repository Go suite was not rerun locally.
Phase 2 remains partial and all target maturity flags remain unchanged. No CLI or SDK API, DB,
repository setting or workflow changes were made.

Next implementation: explicitly based C enum fields, followed by Fairy floating scalars and
remaining integer fields. Bounded callbacks (F3) remain a separate step after the field families.

## Validation of the original assessment

- Reran the existing `TestEmitReprCStructForeignResult`, `TestEmitRejectsReprCStructForeignParameter`
  and `TestEmitRejectsNestedResourceTag` tests with `-mod=readonly -count=1`; all pass.
- Rechecked the recorded hosted run's source SHA and successful conclusions for all nine jobs.
- Zero-field, conditional-flag, field-family and callback boundaries above are source/contract
  findings, not new end-to-end reproductions.
- No full suite, native build or hosted workflow was rerun for this documentation-only assessment.
  No language contract, target flags, product API, DB, repository settings or runner changes.
