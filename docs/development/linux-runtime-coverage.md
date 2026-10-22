# Linux runtime and SDK coverage

Hosted evidence baseline: 2026-10-22, source `5856f62109877c53c6388a4a03e0ac56c101a7a6`,
SDK `0.0.0-stage0.86`, language `/20`, runtime `/0`.
Focused Linux evidence: source `7f7362b87c231c30ea89f3907396957f0d599032`, v3/28 cases.
This inventory supports the [Phase 2 exit decision](phase-status.md#phase-2-exit-decision-2026-10-22);
it does not promote target maturity.
The [runtime contract](../runtime/runtime-stdlib-v0.md), Accepted ADRs, and owning fixtures retain authority.

## Execution evidence

[Frontend run 39798530273](https://github.com/0disoft/fairyc/actions/runs/39798530273)
passed all nine jobs at `5856f62109877c53c6388a4a03e0ac56c101a7a6`. Its
[native Linux job](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119899431035) records report `fairyc.native-c-report/v38` on Ubuntu
24.04 amd64, glibc 2.39, Clang 18.1.3 and GCC 13.3.0: 55 C emission cases,
interpreter/C and drop-trace executions 220 each, sanitizer executions 110,
bidirectional ABI executions 56, layout probe executions 4, and `sdkStorageExecutions: 4`.
SystemHeap, Buffer, Arena, text-view, owned-text and public-File probe counters are each 4.
The four SDK storage executions cover both Region byte operations with normal, empty, and forced
payload and tracking-node allocation failures, recovery, preserved data, and zero live allocations.
That SDK probe is separate from the 110 sanitizer executions. The separate
[Linux Go job 119892384769](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119892384769) passed all 24 focused Linux cases without skips.

The previous [Frontend run 39782215105](https://github.com/0disoft/fairyc/actions/runs/39782215105)
at `1413b4b1855f2bdb7f6b2bc2205b062e028b4870` remains historical evidence for its recorded
52-case receipt. Earlier [run 39768680761](https://github.com/0disoft/fairyc/actions/runs/39768680761)
at `54cbeb17ebdc28a2797cc19c6b45882381c6cc3a` also remains historical evidence. Their public
Buffer/text fixture and four compiler/profile SDK executions are described below.

The following v37 receipt records the earlier baseline:

[Frontend run 39589657672](https://github.com/0disoft/fairyc/actions/runs/39589657672)
passed all nine jobs at `4f7798689bcd6a6ef797c9079ed4e28092647cca` on 2026-10-21.
That v37 receipt is historical and does not attest the current source revision.
Its [native Linux job](https://github.com/0disoft/fairyc/actions/runs/39589657672/job/119191708329)
records report v37 on Ubuntu 24.04 amd64, glibc 2.39, Clang 18.1.3 and GCC 13.3.0.
It records 208 differential executions, 208 drop-trace executions, 104 sanitizer executions,
and 56 bidirectional ABI probe executions. Each runtime/source-call counter below is four,
covering both compilers at low and optimized profiles.

The [Ubuntu Go job](https://github.com/0disoft/fairyc/actions/runs/39589657672/job/119187202941)
also passed. Its non-verbose suite output is not an individual-test or no-skip receipt.
The source pointers below identify coverage, not additional executions performed during this audit.

## Focused Linux receipt

- Run: [39805612550](https://github.com/0disoft/fairyc/actions/runs/39805612550)
- Job: [Linux Go 119918006715](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715)
- Source: [`7f7362b8`](https://github.com/0disoft/fairyc/commit/7f7362b87c231c30ea89f3907396957f0d599032)

| Test | Executions | Skips |
| --- | ---: | ---: |
| `TestLinuxFileProgressNative` | 4 | 0 |
| `TestSystemHeapOOMNative` | 4 | 0 |
| `TestLinuxPreparedPublication` | 4 | 0 |
| `TestBytesEqualExecution` | 4 | 0 |
| `TestBytesCompareExecution` | 4 | 0 |
| `TestCheckedI64DecimalExecution` | 4 | 0 |
| `TestLinuxTrapReceiptProgressNative` | 4 | 0 |

Evidence `fairyc.phase2-linux-evidence/v3`: `complete:true`, 28 entries in `passedCases`,
`missing:[]`, `skipped:[]`, `errors:[]`. Each test passed GCC and Clang at low and optimized
settings (4 executions each). The previous [v2 receipt](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119892384769)
at `5856f62109877c53c6388a4a03e0ac56c101a7a6` records 24 cases and remains historical.
The earlier [v1 receipt](https://github.com/0disoft/fairyc/actions/runs/39787424010/job/119854250599)
at `3ea6d1b` records 12 cases and remains historical.

The file public SDK test exercises EINTR retry, short reads/writes, exact error status/OS
code/completed count, partial no-progress/EOF, unmodified suffix, empty, recovery, sync, and
consuming close without retry. Heap coverage exercises direct initial OOM, resize grow/shrink,
alignment-changing resize failure preservation, recovery, empty with no platform allocation, and
zero-live cleanup. Prepared publication exercises exact-read early EOF, remove, and replace
semantics. These close focused Linux evidence gaps only, not complete runtime or standard-library
coverage. The existing Linux `go test -json` stream is checked by
`.github/scripts/linux_phase2_evidence.py`, so no tests are duplicated.

The current validator requires `fairyc.phase2-linux-evidence/v3`, with 28 compiler/profile cases.
Its seventh root, [TestLinuxTrapReceiptProgressNative](../../internal/cruntime/trap_receipt_progress_test.go),
executes production `LinuxRunSource` with only the receipt `write` call replaced by a test hook.
Each GCC/Clang low/optimized case checks short writes, repeated EINTR, zero progress and EIO before
or after progress, including exact fd, pointer, remaining bytes, trace and exit status.
Local Windows runtime tests and Linux Go cross-compilation pass; native execution skips on Windows.
The hosted v3 receipt records all six scenarios passing under every compiler/profile combination
(24 scenario executions): success exits 195, zero-progress/error exits 254, and exact traces match.

## Implemented surfaces

| Area | Implementation and contract | Fixtures and observed evidence |
| --- | --- | --- |
| Thin runtime: traps, entry adaptation, empty storage, memory helpers | [Runtime sources](../../internal/cruntime/runtime.go), [scalar C](../../internal/cruntime/stage0_scalar.c) | [Trap locations](../../internal/cbackend/trap_location_native_test.go), differential/drop/sanitizer counters above; CLI trap receipts are separate from program streams. |
| SystemHeap backing, Buffer, Arena/Region | [C support](../../internal/cstdlib/system_heap.go), [SDK alloc](../../internal/sdkstdlib/sources/std/alloc.fy), [SDK bytes](../../internal/sdkstdlib/sources/std/bytes.fy) | [Runtime/source probes](../../internal/cbackendnative/probes_stdlib.go): `systemHeapExecutions`, `systemHeapSourceCallExecutions`, `bufferExecutions`, `bufferSourceCallExecutions`, `arenaExecutions`, `arenaSourceCallExecutions` are each 4. SystemHeap is internal support, not an exported SDK type. |
| UTF-8 views, String, CStr, CString | [Text view C](../../internal/cstdlib/stage0_text_view.c), [owned text C](../../internal/cstdlib/stage0_owned_text.c), [SDK text](../../internal/sdkstdlib/sources/std/text.fy) | `textViewExecutions`, `ownedTextExecutions`, `ownedTextSourceCallExecutions`, `textValidationSourceCallExecutions`, `textViewOutputSourceCallExecutions`, `cStrValidationSourceCallExecutions` are each 4. These include raw/source crossings, not every public SDK wrapper. |
| Console byte I/O and scalar output | [Linux console](../../internal/cstdlib/stage0_linux_console.c), [SDK I/O](../../internal/sdkstdlib/sources/std/io.fy) | `linuxConsoleExecutions` and `linuxConsoleFailureExecutions` are each 4; [i64 source test](../../internal/cli/io_i64_native_test.go) also admits Linux. |
| File open/read/write/sync/close/drop, position, logical size, bounded seeks | [Linux adapter](../../internal/cstdlib/stage0_linux_file_portable.c), ADRs [0331](../adr/0331-target-neutral-public-file-core.md), [0332](../adr/0332-target-neutral-public-file-positioning.md), [0333](../adr/0333-target-neutral-public-file-size.md), [0334](../adr/0334-target-neutral-public-file-relative-end-seek.md) | [Public File source](../../internal/cbackendnative/smoke_files.go): `linuxFileExecutions`, `linuxFileSourceCallExecutions`, `linuxPublicFileExecutions` are each 4. |
| Exact read, file removal/replacement, UTF-8 argv copying | [ADR 0424](../adr/0424-linux-prepared-publication-and-process-arguments.md), [Linux argv](../../internal/cstdlib/stage0_linux_process_args.c) | [Prepared publication](../../internal/cstdlib/linux_publication_test.go) defines GCC/Clang low/optimized cases; [save-failure test](../../internal/cli/task_manager_linux_save_test.go) defines 40 injected-failure/process-kill cases. Current evidence is suite-level; [older focused receipts](../compatibility/linux-drimi-20261003.md) retain their original revisions. No power-loss guarantee. |
| SDK Vec and native workspace execution | [Vector source](../../internal/sdkstdlib/sources/optional/std/vec.fy), [Linux support](../../internal/cstdlib/vector_storage_test.go), [ADR 0426](../adr/0426-linux-native-workspace-tests.md) | [Portable native tests](../../internal/cli/native_test_portable_test.go), [Linux vector tests](../../internal/cli/vector_linux_candidate_test.go), [loop test](../../internal/compiler/vector_for_linux_emit_test.go). Linux implementation exists; these pointers do not prove every later vector-view case ran on Linux. |
| Pure bytes and number helpers | [SDK bytes](../../internal/sdkstdlib/sources/std/bytes.fy), [SDK num](../../internal/sdkstdlib/sources/std/num.fy) | [Equality/comparison](../../internal/cli/bytes_equal_test.go), [decimal parsing](../../internal/cli/checked_i64_decimal_test.go), and the shared [Linux SDK helper runner](../../internal/cli/linux_sdk_helpers_test.go) reuse independent Go oracles, MIR codec/interpreter, and production C emission. The three tests add `/gcc/low`, `/gcc/optimized`, `/clang/low`, and `/clang/optimized` subtests and retain Windows MSVC coverage. The v2 receipt records all 12 helper profile cases passing without skips. No new target C symbols are needed. |

## Missing Linux implementations

The [SDK filesystem source](../../internal/sdkstdlib/sources/std/fs.fy) and
[target symbol resolver](../../internal/cstdlibcontract/contract.go) still bind these operations
to Windows implementations:

- Directory creation, recursive creation/removal, rename, and enumeration (`ReadDir`).
- File rename, hard links, byte copying, and `set_len`.
- Extended metadata: kind, read-only observation/mutation, link count, allocated size,
  path kind, file identity, and the compound metadata record. Logical `size` is already available.
- File-to-Buffer `read_append` and `read_to_end`; ordinary Buffer allocation is already available.

These are implementation gaps, not all mandatory Phase 2 work. The
[roadmap](../product/01-roadmap.md#phase-2-first-hosted-executable-slice) requires the minimum
runtime/library surface exercised by representative fixtures, rather than Windows API parity.
Portable paths, sockets, mapped files, async I/O, generic allocator interfaces and freestanding
support remain design exclusions; they are not implied implementation tasks.

## Public SDK storage fixture

The [SDK storage fixture](../../internal/cbackendnative/sdk_storage_source.go) imports the bundled
`std::alloc`, `std::bytes`, and `std::text` through the workspace compiler. It checks Arena/Region
allocation and reuse, Buffer growth, String/CString construction, and UTF-8/NUL rejection.
The existing probe allocator forces both Region allocation failures, Buffer reserve failure,
String copy/append failure, and CString allocation failure. SDK observations check preserved values
and returned ownership; the outer C wrapper requires zero live allocations after cleanup.

[Report v38](../../internal/cbackendnative/native_report.go) adds `sdkStorageExecutions` to the
existing native Linux lane: four executions under GCC/Clang at low/optimized profiles, with no new
workflow or runner. The [focused test](../../internal/cbackendnative/sdk_storage_test.go) also
checks Linux emission on Windows and executes the same fixture at both MSVC profiles.
Focused `TestSDKStorageProbe` passes locally: Linux C emission on Windows, MSVC low/optimized
executions, and zero live allocations, covering resize/zeroing, view mutation, truncate capacity,
empty views, Korean/Greek UTF-8, and `from_buffer` roundtrips. The current hosted receipt also
executes those cases plus both Region byte operations and their forced allocation failures under
GCC/Clang at low/optimized profiles, with zero live allocations.
The first Linux attempt exposed C11 label/declaration emission and
an unrelated Atlas crash-helper deadlock; `e149ac29` and `2c8997ee` correct those paths.
[Frontend run 39762433686](https://github.com/0disoft/fairyc/actions/runs/39762433686)
passed on the corrected source, including all four public SDK storage executions in report v38.
The [expanded-fixture attempt](https://github.com/0disoft/fairyc/actions/runs/39767151309)
at `dc71e198` passed the lexer tests but exceeded that job's ten-minute limit during cleanup,
blocking the dependent Linux native lane. The historical `54cbeb17` revision allows twelve minutes for the lexer job;
test selection and runner count are unchanged.

The current [standard-library build smoke](../../internal/cbackendnative/smoke_build.go)
uses a [raw empty-input UTF-8 call](../../internal/cbackendnative/smoke_text_views.go).
Its `standardLibraryBuildSmokeExecutions: 4` does not establish those public SDK paths.
The [Windows generated failure probes](../../tools/dev/fairyc-windows-host-smoke/generated_failure_probes.go)
provide existing source cases to adapt rather than inventing new APIs.

## Region byte evidence

The Region byte contract is committed at `ca44226b`; zeroed allocation lowering is committed at
`e035e79c`. Byte-copy lowering is committed at `0ed33081`. Those three commits and the evidence
commit `ea0957e6` are on `origin/main`. Source typing and recognition are in
[`internal/semantic/region_types.go`](../../internal/semantic/region_types.go), with MIR lowering
and verification in [`internal/mir/lower_regions.go`](../../internal/mir/lower_regions.go) and
[`internal/mir/verify_operations.go`](../../internal/mir/verify_operations.go). C emission is in
[`internal/cbackend/region_bytes.go`](../../internal/cbackend/region_bytes.go); the interpreter
uses [`internal/interpreter/region_bytes.go`](../../internal/interpreter/region_bytes.go).

Targeted workspace tests pass for `alloc_zeroed_bytes` and `copy_bytes`: one-time argument
evaluation, zeroed and copied contents, subviews with independent writes, using a copy after its source local leaves scope, legacy `place`, and the expected `E3101`, `E2101`, and `E1006` rejections. See
[`region_bytes_test.go`](../../internal/workspacecompiler/region_bytes_test.go) and
[`region_copy_test.go`](../../internal/workspacecompiler/region_copy_test.go).

`TestSDKStorageProbe` passed Linux C emission on Windows and low/optimized MSVC execution. The
fixture forces payload and tracking-node allocation failure for both new operations, checks empty
requests consume no allocation, and verifies recovery, prior-data preservation, and zero live
allocations. See [`sdk_storage_source.go`](../../internal/cbackendnative/sdk_storage_source.go) and
[`sdk_storage_test.go`](../../internal/cbackendnative/sdk_storage_test.go).

`TestRegionByteCopyPersistentWorkspace` passed a cross-session disk-cache hit and execution. It
executes all three Region modes before and after disk-cache restoration.
[Frontend run 39798530273](https://github.com/0disoft/fairyc/actions/runs/39798530273) now covers
both byte operations in four compiler/profile SDK executions, including normal, empty, forced
payload and tracking-node OOM, recovery, preserved data, and zero live allocations. This receipt
establishes the current Linux runtime cases; the SDK probe is distinct from the separate 110
sanitizer executions. Related Go/fixture suites and vet pass locally. The local full repository Go
suite was not rerun, while the hosted native and focused Linux Go jobs passed.

## Phase 2 exit decision (2026-10-22)

Decision: keep Phase 2 partial. The representative minimum passes, but `complete-runtime` and
`complete-standard-library` remain missing and `emittable`, `conformant`, and `supported` stay false.
Conformance requires every declared language, layout, bidirectional ABI, runtime, and toolchain
fixture to pass without skips, under the [C ABI target claims](../abi/c-abi-v0.md#target-status)
and [roadmap exit gate](../product/01-roadmap.md#phase-2-first-hosted-executable-slice). The v38
receipt proves representative success, not blanket completeness.

| Evidence area | What the receipt establishes | Status |
| --- | --- | --- |
| Linux trap-receipt write progress | [runtime.go](../../internal/cruntime/runtime.go) handles partial writes, EINTR, zero progress, and failure; [native tests](../../internal/cdriver/native_test.go) establish ordinary receipt transport. | [Progress tests](../../internal/cruntime/trap_receipt_progress_test.go) pass all four compiler/profile cases in the v3 receipt, with six scenarios per case and no skips. The v2 receipt remains historical and does not attest this test. |
| Pure helper native execution | [Equality/comparison](../../internal/cli/bytes_equal_test.go) and [decimal parsing](../../internal/cli/checked_i64_decimal_test.go) have admitted contracts and independent Go oracles. The shared [Linux SDK helper runner](../../internal/cli/linux_sdk_helpers_test.go) executes production emitted C with native support. | The current source adds GCC/Clang low/optimized subtests to the three existing roots. A missing compiler, build or run failure, nonzero exit, or any stdout/stderr fails the case; unsupported hosts still skip. Windows MSVC low/optimized coverage remains. The v2 receipt records all 12 helper profile cases passing, with no missing or skipped cases. |
| Whole-target observation | The v38 process/drop oracle checks admitted fixtures under two compilers and profiles. | [Exit values](../../internal/cbackendnative/oracle_results.go) cover 0–191, while the manifest defers complex resources and effects. Partial compiler-semantic coverage is not a thin-runtime defect or evidence of entire-language conformance. |

## Phase 2 contract-to-fixture review

The mandatory minimum remains the thin runtime plus representative `SystemHeap`, `Arena`, `Buffer`,
text/`String`/`CString`, and Linux `File` paths, with differential, layout, bidirectional ABI, and
sanitizer coverage ([roadmap](../product/01-roadmap.md#phase-2-first-hosted-executable-slice)).
Windows filesystem parity, distribution, and new APIs are not exit requirements. SDK storage probes
remain separate from the 110 sanitizer executions.

| Contract rule | Source owner | Fixture owner | Evidence | Scope / gap |
| --- | --- | --- | --- | --- |
| Thin runtime: traps, entry adaptation, empty storage, memory helpers | [runtime.go](../../internal/cruntime/runtime.go), [stage0_scalar.c](../../internal/cruntime/stage0_scalar.c) | [runner.go](../../internal/cbackendnative/runner.go), [trap_location_native_test.go](../../internal/cbackend/trap_location_native_test.go) | v38 records the differential/drop matrix below; the earlier focused trap-location receipts are linked above. CLI trap receipts remain separate from program streams. | Mandatory. Current fixtures cover the admitted thin-runtime paths, not every possible runtime path. |
| SystemHeap backing and direct resize | [system_heap.go](../../internal/cstdlib/system_heap.go), [stage0_system_heap.c](../../internal/cstdlib/stage0_system_heap.c) | [probes_stdlib.go](../../internal/cbackendnative/probes_stdlib.go), [smoke_heap.go](../../internal/cbackendnative/smoke_heap.go), [stage0_system_heap_probe.c](../../internal/cstdlib/stage0_system_heap_probe.c), [stage0_system_heap_probe_allocator.c](../../internal/cstdlib/stage0_system_heap_probe_allocator.c) | v38 system-heap runtime/source counters are each 4. Resize is allocate-copy-release; the raw probe checks empty, alignment, prefix, invalid-alignment rollback, grow, shrink, and release. The allocator probe enables caller Buffer/Arena/text OOM. | Mandatory. The [focused Linux receipt](#focused-linux-receipt) records four direct OOM executions without skips; local MSVC low/optimized also pass. Caller failure alone is not equivalent. SystemHeap remains internal support, not an exported SDK type. |
| Arena/Region | [stage0_arena.c](../../internal/cstdlib/stage0_arena.c), [alloc.fy](../../internal/sdkstdlib/sources/std/alloc.fy), [region_bytes.go](../../internal/cbackend/region_bytes.go) | [sdk_storage_source.go](../../internal/cbackendnative/sdk_storage_source.go), [smoke_arena.go](../../internal/cbackendnative/smoke_arena.go), [region_bytes_test.go](../../internal/workspacecompiler/region_bytes_test.go), [probes_stdlib.go](../../internal/cbackendnative/probes_stdlib.go) | v38 reports 4 SDK storage executions covering both Region byte operations with normal, empty, forced payload and tracking-node OOM, recovery, preserved data, and zero live allocations; Arena counters are each 4. Targeted workspace tests cover both byte operations and expected rejections. | Mandatory minimum. Public placement, rollback, reuse, and reset are covered; no unchanged Region rerun is needed. |
| Buffer | [stage0_buffer.c](../../internal/cstdlib/stage0_buffer.c), [bytes.fy](../../internal/sdkstdlib/sources/std/bytes.fy) | [sdk_storage_source.go](../../internal/cbackendnative/sdk_storage_source.go), [smoke_buffer.go](../../internal/cbackendnative/smoke_buffer.go), [stage0_buffer_probe.c](../../internal/cstdlib/stage0_buffer_probe.c) | v38 Buffer counters are each 4. The public fixture executes `empty`, `resize_zeroed`, `truncate`, and `view_mut`, including zeroing and regrowth. The raw probe checks overlap, overflow, reserve/resize failure, preserved storage, and cleanup. | Mandatory. The raw and public cases are complementary; this does not establish every public failure path. |
| Text, `String`, `CString` | [stage0_text_view.c](../../internal/cstdlib/stage0_text_view.c), [stage0_owned_text.c](../../internal/cstdlib/stage0_owned_text.c), [text.fy](../../internal/sdkstdlib/sources/std/text.fy) | [sdk_storage_source.go](../../internal/cbackendnative/sdk_storage_source.go), [smoke_text_views.go](../../internal/cbackendnative/smoke_text_views.go), [smoke_owned_text.go](../../internal/cbackendnative/smoke_owned_text.go), [stage0_text_view_probe.c](../../internal/cstdlib/stage0_text_view_probe.c), [stage0_owned_text_probe.c](../../internal/cstdlib/stage0_owned_text_probe.c) | v38 text-view and owned-text counters are each 4. Public checks include empty `String`, exact multibyte UTF-8, and successful/empty `from_buffer` roundtrips; raw probes supply boundaries, NUL rejection, empty, rollback, and OOM. | Mandatory. Coverage is representative; this does not establish every public failure path. |
| Linux `File` core: open/read/write_all/sync/close/drop, position, size, bounded seeks | [stage0_linux_file.c](../../internal/cstdlib/stage0_linux_file.c), [stage0_linux_file_portable.c](../../internal/cstdlib/stage0_linux_file_portable.c), [fs.fy](../../internal/sdkstdlib/sources/std/fs.fy) | [stage0_linux_file_probe.c](../../internal/cstdlib/stage0_linux_file_probe.c), [smoke_linux_file.go](../../internal/cbackendnative/smoke_linux_file.go), [smoke_files.go](../../internal/cbackendnative/smoke_files.go) | v38 reports `linuxFileExecutions`, `linuxFileSourceCallExecutions`, and `linuxPublicFileExecutions` at 4 each. Raw coverage includes partial read/EOF/write_all/sync/consuming close/EBADF/raw drop; safe coverage includes create/read/write_all/seeks/size/close. | Mandatory core. The representative safe path and raw failure path are covered; the remaining progress/error assertions are listed below. |
| `File` exact read, publication, and failure paths | [stage0_linux_file_portable.c](../../internal/cstdlib/stage0_linux_file_portable.c), [ADR 0424](../adr/0424-linux-prepared-publication-and-process-arguments.md) | [linux_publication_test.go](../../internal/cstdlib/linux_publication_test.go), [task_manager_linux_save_test.go](../../internal/cli/task_manager_linux_save_test.go), [task_manager_linux_save_hooks_test.go](../../internal/cli/task_manager_linux_save_hooks_test.go) | `TestLinuxPreparedPublication` defines GCC/Clang low/optimized exact-read early EOF, suffix preservation, remove/replace error, and old-open-owner cases. The save tests define 40 cases (four compiler profiles x ten modes): first/partial writes ENOSPC, sync EIO, close EIO, replace failure, and kill/restart preservation. The focused receipt now records four passing `TestLinuxPreparedPublication` executions without skips; the separate 40 save-failure cases retain suite-level evidence. | Mandatory. The [focused receipt](#focused-linux-receipt) now covers deterministic File EINTR retry, zero progress, and exact partial error counts through the public SDK. The save-failure cases separately assert application errors and data. Process kill is not power-loss proof. Reuse test-only POSIX seams; no new public API. |
| Differential, layout, ABI, sanitizer | [execution.go](../../internal/cbackendnative/execution.go), [oracle_results.go](../../internal/cbackendnative/oracle_results.go), [probes_layout.go](../../internal/cbackendnative/probes_layout.go), [abi_probe.go](../../internal/cbackendnative/abi_probe.go) | [runner.go](../../internal/cbackendnative/runner.go), [c-emission fixture](../../spec/fixtures/c-emission/README.md) | v38 reports 55 emission cases, 220 differential and drop-trace executions, 4 layout probes, 56 bidirectional ABI executions, and 110 sanitizer executions. Execution/oracle results compare exit status/trap code and source-linked drop IDs. | Mandatory. The fixture README says active built-in payload selection is observed; no separate native payload event stream exists. The [active-payload mapping](#active-payload-observation-mapping) identifies three direct-cleanup fixtures; their Linux execution is included in this receipt. The existing observer and wire format are unchanged. |
| Target maturity | [spec.go](../../internal/target/spec.go) | [spec_test.go](../../internal/target/spec_test.go) | The spec stores `complete-runtime` and `complete-standard-library` as missing. The test validates structural maturity ordering and nonempty missing values. | Mandatory review gate. The test does not compute fixture completeness. Both missing markers remain; `emittable`, `conformant`, and `supported` stay false. |

Outside the representative Phase 2 minimum ([Linux File contract](../runtime/runtime-stdlib-v0.md#linux-file-v0)): directory lifecycle/enumeration, rename/hardlink/copy/`set_len`, extended metadata, and File-to-Buffer. Admission depends on Linux target adapters and owning native fixtures. `Vec<T>` and extra pure helpers are later surfaces, not required full SDK parity. Portable paths, sockets, mapped files, async I/O, generic allocators, and freestanding support remain design exclusions. No deferral or waiver is claimed for the missing mandatory evidence.

## File progress fixture

The [File progress fixture](../../internal/cli/linux_file_progress_test.go) adds
`TestLinuxFileProgressEmission` plus [syscall
scripts](../../internal/cli/linux_file_progress_hooks_test.go) and a source-level companion that
exercise the public SDK write/write_all/read/read_exact paths. The native cases assert exact
payloads and untouched suffixes, status/OS-code/completed-byte errors, EINTR retries, short
transfers, initial and partial errors, zero-progress, EOF/empty requests, recovery, and sync
retry/error plus consuming close failure without retry. Test-only POSIX hooks reuse the existing
pattern, leaving production API, runtime ABI, SDK bytes, DB, hygiene, and CI runners unchanged.

`TestLinuxFileProgressEmission` passes on Windows. The [focused Linux receipt](#focused-linux-receipt)
records all four GCC/Clang low/optimized `TestLinuxFileProgressNative` executions passing without skips. The scripts supply syscall outcomes, while production File
algorithms and SDK error conversion remain in the compiled program.

## Direct heap OOM fixture

The [direct heap OOM fixture](../../internal/cstdlib/system_heap_oom_test.go) uses the existing
test-only allocator adapter against real target-selected SystemHeap C. It checks initial OOM with
empty storage, then growth, shrink, and alignment-changing resize failures preserving pointer, size,
alignment, bytes, and one live allocation; recovery, zero requests that leave the injected failure
armed, and final cleanup are checked too. MSVC low and optimized local native builds and runs pass.
The [focused Linux receipt](#focused-linux-receipt) now records all four GCC/Clang low/optimized
executions passing without skips. This closes the direct heap OOM evidence gap for those profiles.
The exit decision above keeps Phase 2 partial; this focused receipt does not close either missing marker.

## Active-payload observation mapping

Phase 2 exit ([roadmap](../product/01-roadmap.md)) requires interpreter/C agreement on values,
status, trap, and active-payload/drop traces. Concrete tagged payload types and `MovePayload` are
owned by [ADR 0034](../adr/0034-resource-tagged-c-transport.md); active-only source/type/span drop
observation by [ADR 0035](../adr/0035-native-drop-trace-differential.md). Existing drop
instrumentation suffices for bounded admitted paths; no new payload event stream, product observer,
public API/runtime ABI, or emitter algorithm is added.

| Observation | Fixtures | Source | Expectation |
| --- | --- | --- | --- |
| Scalar payload selection | `option-some/none`, `result-propagate-ok/err` | [manifest](../../spec/fixtures/c-emission/manifest-v0.json) | selected scalar observable via final value |
| Resource extraction/move | `resource-option-some/none`, `resource-result-ok/err`, `resource-result-propagate-ok/err` | [manifest](../../spec/fixtures/c-emission/manifest-v0.json) | exact resource type and binding span |
| Propagation | `resource-result-propagate-ok/err` | [manifest](../../spec/fixtures/c-emission/manifest-v0.json) | move/propagation preserves exact drop trace |
| Direct Option cleanup (new) | `resource-option-drop` | [main.fy](../../spec/fixtures/c-emission/cases/pass/resource-option-drop/src/main.fy) | `Token` dropped at `present` only; absent `None` yields zero drops; return 42 |
| Direct Result cleanup (new) | `resource-result-drop-ok/err` | [ok](../../spec/fixtures/c-emission/cases/pass/resource-result-drop-ok/src/main.fy), [err](../../spec/fixtures/c-emission/cases/pass/resource-result-drop-err/src/main.fy) | explicit discard; `Success` drops at `wrapped` for `Ok`, `Failure` at `wrapped` for `Err`; other alternative never dropped; return 42 |

[active_payload_test.go](../../internal/cbackendfixture/active_payload_test.go) authors independent
expected resource type and exact binding span for all nine resource cases; `None` expects zero
drops, so interpreter and C cannot agree on an omitted or wrong-source expectation.
[runner.go](../../internal/cbackendfixture/runner.go) compares interpreter outcome to the manifest
and emits production/closed trace units;
[oracle_results.go](../../internal/cbackendnative/oracle_results.go) maps source/type/span to IDs;
[execution.go](../../internal/cbackendnative/execution.go) compares exact runtime drop ID sequence
and process status. Moves must not introduce extra drops; there is no new event wire format.

Counters: the previous hosted run 39782215105 at `1413b4b1` remains historical evidence for
the prior 52 cases (208 process-status/drop comparisons, 104 sanitizer executions). The current
hosted run 39798530273 at `5856f621` covers all 55 cases with 220 process-status/drop comparisons
and 110 sanitizer executions, including the three direct-cleanup cases at GCC/Clang low and
optimized profiles. No new jobs or runners were required. Local `go test
-mod=readonly -count=1 ./internal/cbackendfixture ./internal/cbackendnative` passed, and
`TestEmissionTransportAcrossResponsibilitySplit` preserves the production and instrumented C units
of the original 52 cases byte-for-byte; the new 3 are excluded from the historical hash and covered
by independent golden observations, not rebaselined.

The exit decision above leaves both missing markers and all three maturity flags unchanged. Nested
resource tags and effectful cleanup are not admitted by this mapping. This representative receipt
does not prove complete runtime or standard-library coverage, and no requirement is waived.

## Next work

Next single task: use the contract-to-fixture review above to identify the next required runtime
or SDK observation gap. Keep both missing completeness markers and all three target flags unchanged
until their owning requirements have evidence; excluded API surfaces remain separate decisions.
