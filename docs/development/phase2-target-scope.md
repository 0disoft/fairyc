# Phase 2 target scope assessment

- Date: 2026-10-23
- Baseline inspected checkout: `3b6f58c9de4ee12f212e68b152d374d055b81b55`
- Current hosted F2-5 receipt: `f79e55c6bdc904c677c117789feb4839b53d4b80`,
  [Frontend run 39849506646](https://github.com/0disoft/fairyc/actions/runs/39849506646), all nine
  jobs passed; [native job 120076773359](https://github.com/0disoft/fairyc/actions/runs/39849506646/job/120076773359)
  and [Linux Go job 120071663704](https://github.com/0disoft/fairyc/actions/runs/39849506646/job/120071663704)
- Historical hosted F1–F3 receipt: `82b760cbb743eef4608441e127f9fa9e41d1f513` (below)
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

The baseline [Frontend run 39817319718](https://github.com/0disoft/fairyc/actions/runs/39817319718)
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

F1, F2-1 through F2-4 and the first bounded F3 callback have the historical
[hosted receipt](#hosted-f1f3-receipt-2026-10-23) at `82b760cb`. F2-5 adds the remaining five
integer-array families and has local MSVC, compiler and backend evidence. Its
[current hosted receipt](#hosted-f2-5-receipt-2026-10-23) at `f79e55c6` passed all nine jobs.
The old receipt does not cover F2-5. Additional field operations and callback signatures
require separate admissions; none of these updates opens excluded resource representations.

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

The local checks above are historical pre-push validation. Local WSL discovery failed with
`REGDB_E_CLASSNOTREG`, and no Linux installation or host configuration was changed. The
[hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) records the configured Linux directions and
profiles at `82b760cb`; the earlier full CI receipt above cannot attest these checks. The full
repository Go suite was not rerun locally. Phase 2 and all target maturity flags remain unchanged;
no DB, repository setting, workflow or runner changes.

The F2-1 update below implements the first field family. Its checks are included in the hosted
F1–F3 receipt.

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
  -run '^TestReprCRecordFieldNativeCrossing$'`: MSVC 14.51.36231 passes all four C11/C17
  low/optimized separate-object combinations, each returning 42 with empty output.

The initial full Windows smoke stopped at the expanded record probe because its runtime trap
symbol was not linked. After adding the canonical runtime, the focused native test above passed;
the entire smoke was not repeated. No product runtime implementation or link policy changed.
Changed-document relative links and `git diff --check` pass.
The focused test was renamed from `TestReprCArrayFieldNativeCrossing` when ADR 0478 added enums;
the command above uses its current name.

The local validation above is historical pre-push validation. The
[hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) executes this expanded probe in all four
Linux compiler/profile combinations. The full repository Go suite was not rerun locally. Phase 2
remains partial and all target maturity flags remain unchanged. No CLI or SDK API, DB, repository
setting or workflow changes were made.

The F2-2 update below implements the next field family. Fairy floating scalars and remaining integer
fields follow it; bounded callbacks (F3) remain a separate step after the field families.

## F2-2 explicit C enum fields

[ADR 0478](../adr/0478-explicit-c-enum-fields.md) admits nonempty, payload-free enums with one
`@repr(c::integer)` base and zero-based declaration ordinals. Metadata commit `9b8ba65f` preserves
the base through HIR/MIR, public interface hashing and stable package contexts/rebinding. The C
backend now uses the selected scalar typedef and existing target C-enum layout, with exact size,
alignment, containing-field offsets and provider checks. Construction, switch, record copies,
direct enum-field replacement and foreign record arguments/results are connected. Plain enum
fields in repr(C), payloads, explicit discriminant values, enum arrays, enum-field borrows, nested
enum leaf replacement and bare foreign enum signatures remain closed.

The shared native fixture includes unsigned char/short/int, long long and long enum fields,
separately declared C records, a small record and a large record, scalar neighbors, field
replacement and unchanged caller copies. The same existing native lanes consume it; no job,
report counter or workflow was added.

Local validation passed:

- Full HIR, semantic, MIR, MIR codec, package-artifact and public-interface package tests for the
  metadata slice; full MIR, interpreter, backend, emission-fixture and shared-probe tests after
  connecting execution.
- Focused compiler tests cover codec/interpreter behavior, all 13 C integer bases in Linux and
  Windows emission profiles, field-store/borrow regressions, and rejection of plain repr(C) enum
  fields and bare foreign enum signatures. Windows correctly rejects unavailable `c::ssize_t`.
- `go test -mod=readonly -count=1 -v ./tools/dev/fairyc-windows-host-smoke
  -run '^TestReprCRecordFieldNativeCrossing$'`: MSVC 14.51.36231 passes all four C11/C17
  low/optimized combinations. The fixture observes 1/2/4/8-byte enum fields and LLP64 C long storage,
  along with the previous record and array checks.
- `go vet -mod=readonly` for all affected implementation packages and the Windows smoke tool.
- Changed-document relative links and `git diff --check`.

The local validation above is historical pre-push validation; earlier receipts attest their
original sources only. The [hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) executes this
expanded fixture. Locally, the full repository suite and full Windows smoke were not repeated; the focused
native crossing covers the changed probe. No SDK/CLI API, DB, repository setting or workflow
changes. Phase 2 stays partial and all target maturity flags stay unchanged.

The F2-3 update below connects Fairy `f32`/`f64` field transport. Remaining fixed-width integer
fields follow it.

## F2-3 floating field transport

[ADR 0479](../adr/0479-repr-c-floating-field-transport.md) connects distinct Fairy `f32`/`f64`
types to repr(C) fields and local/internal value copies. Type/cache commit `db47dc2f` preserves
the identities through MIR encoding, public interfaces and stable package contexts/rebinding.
The C backend emits float/double using target language-scalar size/alignment plus binary-format
provider assertions. Explicit reconstruction and whole-record copies, including nested repr(C)
records, retain exact source types without implicit width or C-scalar conversions.

This is a transport admission. Floating literals, operators, comparisons, casts, arrays/views,
direct floating-field stores, tagged/resource payloads, direct private floating fields and bare
foreign floating signatures remain closed. The interpreter continues to reject foreign calls.

The expanded shared native probe includes a 4-byte floating record, a 16-byte double pair, a
16-byte mixed integer/floating record and a 24-byte record. Each configured execution checks
finite values, negative zero, infinities, quiet NaNs and the smallest positive subnormals, including
Fairy field reads, internal value calls, reconstruction and C by-value mutation with caller copies
unchanged. Every floating member is observed by independent C code. Quiet NaN classification is
checked; signaling NaN behavior and NaN payload identity are not promised.

Local validation passed:

- Full HIR, semantic, public-interface, package-artifact and MIR-codec package tests for the type
  slice; backend and emission-fixture package tests for C transport. Focused compiler tests cover
  both emission targets, MIR codec preservation, foreign-call interpreter rejection and boundaries
  including `<float.h>` macro-name collisions.
- `go test -mod=readonly -count=1 -v ./tools/dev/fairyc-windows-host-smoke
  -run '^TestReprCRecordFieldNativeCrossing$'`: MSVC 14.51.36231 passes C11/C17 low/optimized,
  four combinations with five floating cases each, along with earlier record/array/enum checks.
- The shared probe's Linux-profile C emission test and affected-package `go vet -mod=readonly`.
- Changed-document relative links and `git diff --check`.

The local validation above is historical pre-push validation; previous C-scalar or record receipts
do not attest this expanded Fairy floating source. The
[hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) covers it. Locally, the full repository suite and
full Windows smoke were not repeated; the changed native probe was run directly. There is no
SDK/CLI API, DB, workflow, runner, runtime or target-maturity change. Phase 2 stays partial.

The F2-4 update below connects the remaining integer field families. Bounded callbacks (F3)
follow separately.

## F2-4 remaining integer field transport

[ADR 0480](../adr/0480-repr-c-integer-field-transport.md) connects Fairy `i8`, `i16`, `u16`, `u64`
and `isize` fields to target layouts and C record arguments/results. Type/cache commit `cc48bf52`
preserves their distinct builtin identities through verified MIR, MIR encoding, public-interface
hashes and stable package contexts/rebinding. Field reads, typed locals, internal value calls,
whole-record copies and explicit reconstruction work, including nested repr(C) records.

C storage uses `int8_t`, `int16_t`, `uint16_t`, `uint64_t` and `intptr_t`, with target-derived
size/alignment and signedness assertions. In particular `isize` remains pointer-sized on LLP64;
it is not C `long`. The new provider typedef names are reserved against public C-name collisions.
Literals, operators, comparisons, casts, arrays/views, direct scalar-field stores, direct private
fields, scalar resource/tagged/enum payloads, bare foreign signatures and pointers to these new
scalar types remain closed. Existing nested-record operations remain admitted. Foreign calls
remain unavailable in the pure interpreter.

The shared native probe adds a 24-byte mixed record, its nested envelope and individual
1/2/8-byte scalar records. Four cases cover zero, signed minima/maxima, minus one, unsigned
maxima and values above signed 64-bit range. Independent C code observes every member after
Fairy field reads, internal calls, reconstruction and record calls. C by-value mutation changes
all members while original and reconstructed caller records remain unchanged.

Local validation passed:

- HIR, semantic, public-interface, package-artifact and MIR-codec package tests for type identities;
  focused compiler MIR round-trip, both target emission profiles and unsupported-operation checks.
- Backend emission tests, emission fixtures and the shared probe's Linux-profile source emission.
- `go test -mod=readonly -count=1 -v ./tools/dev/fairyc-windows-host-smoke
  -run '^TestReprCRecordFieldNativeCrossing$'`: MSVC 14.51.36231 passes C11/C17 low/optimized,
  four combinations with four integer cases each, alongside prior record/array/enum/float checks.
- Affected-package `go vet -mod=readonly`, changed-document relative links and `git diff --check`.

The local validation above is historical pre-push validation; earlier receipts do not attest the
new integer probe. The [hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) covers it. Locally, the full
repository suite and full Windows smoke were not repeated; the changed crossing was run directly.
No SDK/CLI API, DB, workflow, runner or target-maturity changes. Phase 2 remains partial. The F3
update below connects bounded callbacks. Further field-family admissions remain separate changes
rather than implicit consequences of this transport slice.

## F3 bounded C callbacks

[ADR 0481](../adr/0481-bounded-c-callbacks.md) admits one non-capturing callback signature:
`unsafe extern C fn i32(i32, ptr mut void)`. It appears as the penultimate unsafe import parameter,
followed by final `ptr mut void` user data. Calls pass a named unsafe Fairy function with the exact
signature and no declared effects. C adapters keep ordinary Fairy functions private. Callback
locals, fields, arrays, results, casts, null callback arguments and indirect calls remain closed.

Source typing is committed at `7071809c`; verified MIR, C transport and package/cache preservation
are committed at `f49b7433`. MIR checks callback origin, exact signature, unique use and no local
overwrite. Callback temporaries are formed after user-data evaluation and consumed only by the
designated foreign argument in that block. This permits a nested user-data producer without
extending the callback value across a call. Process and stable artifacts preserve callback target
dependencies; stable targets rebind through FunctionID rather than saved process-local symbols.

The unsafe import promises same-thread synchronous calls ending before it returns. The external
callee must not retain call-bounded data, unwind or `longjmp` through Fairy. Raw data validity is
the caller's obligation; existing borrow-to-pointer checks apply. These C obligations are explicit
contracts, not compiler-proven properties or retained/asynchronous callback support.

The shared native probe runs six callback invocations per configured execution: null data with a
pure handler, a Fairy stack counter updated through an independent C helper, and nested C user-data
production. C observes the expected invocation count and data updates. Prior record/array/enum/
float/integer checks remain in the same unit.

Local validation passed:

- HIR, semantic, public-interface, MIR, MIR-codec, interpreter, package-artifact, C-backend and
  emission-fixture package tests; focused compiler tests cover both emission profiles and forged
  callback origins, null substitution, duplicate use and overwrite rejection.
- Process/stable package reassembly with fresh symbols/package numbers and identical re-encoding;
  unresolved callback targets are rejected. Cross-package callback targets survive memory cache
  hits and disk hits in an independent session without MIR lowering.
- The shared probe's Linux-profile C emission test and MSVC 14.51.36231 C11/C17 low/optimized
  execution: four combinations, six callback invocations each, alongside all prior probe cases.
- Affected-package vet, changed-document relative links and `git diff --check`.

Locally, the full repository suite and full Windows smoke were not repeated because the changed compiler
packages and native crossing were checked directly. Old receipts do not attest callbacks; the
[hosted F1–F3 receipt](#hosted-f1f3-receipt-2026-10-23) does. There are no SDK/CLI API, DB, workflow,
runner or target-maturity changes. Phase 2 remains partial.

## F2-5 remaining integer-array transport

[ADR 0482](../adr/0482-repr-c-integer-array-transport.md) extends [ADR 0477](../adr/0477-repr-c-integer-array-fields.md)
to `i8`, `i16`, `u16`, `u64` and `isize`, completing the ten Fairy integer-array families for lengths
`1..=4096`. Source typing is committed at `fe4beed8`; backend and native proof are committed at
`af181d65`. `internal/foreignstructprobe/integer_arrays.go` extends the shared probe. Scope remains
whole-array and record value transport; element indexing/stores/views/
iteration, nested stores, bare foreign arrays, new scalar pointers and direct resource/enum/tagged
payload arrays remain closed.

Local validation passed: focused compiler `TestIntegerArray*` in both profiles with boundary and
1/4096 lengths; full `cbackend` (65.557s), `cbackendfixture`, `cbackendnative` and
`foreignstructprobe`; affected backend/probe vet; HIR, semantic, MIR, codec, moduleinterface and
packageartifact suites; and Linux-profile `TestForeignCStructResultCallSmokeSourceReachesCEmission`.
MSVC 14.51.36231 passes C11/C17 low/optimized, all four executions, expected 42 and empty output.
The new cases cover five families in 72-byte mixed/nested records, length-3 arrays, internal value
arguments/results, borrowed nested reads, direct and referenced mutable replacement, all-element C
observation, reconstruction/mutation with saved originals unchanged, and per-family `[1]` records.

Locally, the full repository suite and full Windows smoke were not repeated, and no local Linux
native run was made. Hosted CI and Windows smoke passed for this commit.

## Hosted F2-5 receipt (2026-10-23)

[Frontend run 39849506646](https://github.com/0disoft/fairyc/actions/runs/39849506646) at
`f79e55c6bdc904c677c117789feb4839b53d4b80` completed successfully with all nine jobs passing.
The [native job 120076773359](https://github.com/0disoft/fairyc/actions/runs/39849506646/job/120076773359)
reports `fairyc.native-c-report/v38` for `x86_64-linux-gnu` with glibc 2.39, Clang 18.1.3 and
GCC 13.3.0 at low and optimized profiles: 56 cases, 224 outcomes, 224 drop traces, 112
sanitizers, 56 ABI crossings, four layout probes, four SDK executions and
`foreignCStructResultCallExecutions=4`.

The four shared executions include four new integer-array cases each and cover both
Clang-caller/GCC-callee and GCC-caller/Clang-callee directions at low and optimized profiles as
separate objects. These checks are inside the four shared executions; they add no report counters
and are not independently sanitizer-proven. The [Linux Go job
120071663704](https://github.com/0disoft/fairyc/actions/runs/39849506646/job/120071663704) reports
`fairyc.phase2-linux-evidence/v4`, `complete:true`, 32 passed cases and empty `missing`,
`skipped` and `errors` lists.

## Hosted F1–F3 receipt (2026-10-23)

[Frontend run 39839678199](https://github.com/0disoft/fairyc/actions/runs/39839678199) at
`82b760cbb743eef4608441e127f9fa9e41d1f513` completed successfully, with all nine jobs passing.
The [native job 120043200670](https://github.com/0disoft/fairyc/actions/runs/39839678199/job/120043200670)
records `fairyc.native-c-report/v38` on Ubuntu 24.04 amd64 with glibc 2.39, Clang 18.1.3 and GCC
13.3.0 at low and optimized profiles: 56 cases, 224 outcome comparisons, 224 drop-trace
comparisons, 112 sanitizer executions, 56 ABI crossings, four layout probes and four SDK
executions. It also reports `foreignCStructResultCallExecutions=4`.

[probes_foreign.go](../../internal/cbackendnative/probes_foreign.go) and
[probes_source.go](../../internal/cbackendnative/probes_source.go) show that this counter runs the
current shared `internal/foreignstructprobe` source. The four executions cover F1 record arguments;
F2-1 integer arrays, F2-2 explicit C enums, F2-3 floating fields and F2-4 remaining integer
fields; and F3 callbacks in both Clang-caller/GCC-callee and GCC-caller/Clang-callee directions at
low and optimized profiles as separate objects, with expected exit 42. Each execution checks five
floating cases, four integer cases and six callback invocations. They are checks inside four
shared executions, not separate report counters, and are not separately sanitizer-proven.

The separate [Linux Go job 120037158582](https://github.com/0disoft/fairyc/actions/runs/39839678199/job/120037158582)
reports `fairyc.phase2-linux-evidence/v4`, `complete:true`, 32 cases and empty `missing`,
`skipped` and `errors` lists. The [Windows native job 120037158483](https://github.com/0disoft/fairyc/actions/runs/39839678199/job/120037158483)
and [Windows Go job 120037158472](https://github.com/0disoft/fairyc/actions/runs/39839678199/job/120037158472)
succeeded. Phase 2 remains partial; the two completeness markers and all three maturity flags stay
unchanged.

## Remaining source boundaries after F1–F3

The `82b760cb` F1–F3 receipt remains historical. The later `f79e55c6` F2-5 receipt covers the
remaining integer-array families, so the current boundaries are:

| Surface | Implemented subset | Remaining boundary or exclusion | Owners |
| --- | --- | --- | --- |
| repr(C) integer array fields | All ten Fairy integer families at lengths `1..=4096`; whole-field reads, replacement and record/argument value copies. ADR 0482's five newer families are transport-only and have no element indexing, arithmetic or views. | C scalar and C enum arrays need separate admissions. | [HIR guard](../../internal/hir/views.go), [layout resolution](../../internal/cbackend/aggregate_types.go), [ADR 0477](../adr/0477-repr-c-integer-array-fields.md), [ADR 0482](../adr/0482-repr-c-integer-array-transport.md) |
| Fairy `f32`/`f64` arrays | Scalar field transport is implemented. | Array values are deferred; this is the recommended next bounded field family. | [C ABI](../abi/c-abi-v0.md#c-representable-types), [ADR 0479](../adr/0479-repr-c-floating-field-transport.md) |
| Exact C-scalar arrays | Scalar fields are implemented. | Array members are deferred to a separate admission. | [C ABI](../abi/c-abi-v0.md#c-representable-types), [layout resolution](../../internal/cbackend/aggregate_types.go) |
| Explicit C enum arrays | Explicitly based enum fields are implemented, including replacement and record transport. | Array members are deferred to a separate admission. | [ADR 0478](../adr/0478-explicit-c-enum-fields.md), [layout resolution](../../internal/cbackend/aggregate_types.go) |
| repr(C) record-element arrays | Private scalar and plain-struct arrays remain supported in their existing paths. | Repr(C) member arrays are deferred and currently rejected. Private array support does not imply C record-element arrays. | [HIR guard](../../internal/hir/views.go), [semantic checks](../../internal/semantic/resource_types.go) |
| Other array element shapes | The ten integer families and lengths are executable in repr(C). | Nested arrays, `bool` arrays and raw-pointer element arrays are not admitted by ADR 0477. These admission limits do not establish permanent C ABI exclusions. | [ADR 0477](../adr/0477-repr-c-integer-array-fields.md), [HIR guard](../../internal/hir/views.go) |
| Foreign raw pointers | [foreignPointerElementCType](../../internal/cbackend/foreign_types.go) permits `void`, `u8`, `i32`, `i64`, `u32`, `usize` and admitted repr(C) records as pointees; `ptr mut ptr mut void` is the only nested shape. | Pointers to the new integers, floats, C scalars and C enums remain closed. Raw dereference, arithmetic, general casts and other nesting remain closed. | [Pointer guard](../../internal/cbackend/foreign_types.go), [C ABI](../abi/c-abi-v0.md#c-representable-types) |

Resources, safe borrows, slices and tagged values remain explicit exclusions as C ABI values or
record fields under the [C ABI contract](../abi/c-abi-v0.md#c-representable-types).

[ADR 0481](../adr/0481-bounded-c-callbacks.md)'s `ptr mut void` callback data is a separate,
single-signature admission with the same non-retention and same-thread obligations. It does not
open general callback signatures or pointer nesting.

The next bounded implementation should add `f32`/`f64` repr(C) array value transport: whole-field
copies, locals, internal value arguments/results, reconstruction and replacement. It should not add
floating literals, arithmetic, indexing or views. C-scalar arrays and explicit C enum arrays should
follow as separate admissions; repr(C) record-element arrays remain deferred.

Phase 2 remains partial. Keep the completeness markers and all target maturity flags unchanged.

## Validation of the original assessment

- Reran the existing `TestEmitReprCStructForeignResult`, `TestEmitRejectsReprCStructForeignParameter`
  and `TestEmitRejectsNestedResourceTag` tests with `-mod=readonly -count=1`; all pass.
- Rechecked the recorded hosted run's source SHA and successful conclusions for all nine jobs.
- Zero-field, conditional-flag, field-family and callback boundaries above are source/contract
  findings, not new end-to-end reproductions.
- No full suite, native build or hosted workflow was rerun for this documentation-only assessment.
  No language contract, target flags, product API, DB, repository settings or runner changes.
