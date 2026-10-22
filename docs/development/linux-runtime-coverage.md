# Linux runtime and SDK coverage

Hosted evidence baseline: 2026-10-22, source `ea0957e6d45dc63bf248108cdb5f47b6a0f3002d`,
SDK `0.0.0-stage0.86`, language `/20`, runtime `/0`.
This inventory supports the [Phase 2 review](phase-status.md); it does not promote target maturity.
The [runtime contract](../runtime/runtime-stdlib-v0.md), Accepted ADRs, and owning fixtures retain authority.

## Execution evidence

[Frontend run 39774562874](https://github.com/0disoft/fairyc/actions/runs/39774562874)
passed all nine jobs at the audited source revision on 2026-10-22.
Its [native Linux job](https://github.com/0disoft/fairyc/actions/runs/39774562874/job/119814527459)
records report `fairyc.native-c-report/v38` on Ubuntu 24.04 amd64, glibc 2.39, Clang 18.1.3 and
GCC 13.3.0: `sdkStorageExecutions: 4`, interpreter/C and drop-trace executions 208 each,
sanitizer executions 104, bidirectional ABI executions 56, and layout probe executions 4.
SystemHeap, Buffer, Arena, text-view, owned-text and public-File probe counters are each 4.
The four SDK storage executions cover both Region byte operations with normal, empty, and forced
payload and tracking-node allocation failures, recovery, preserved data, and zero live allocations.
That SDK probe is separate from the 104 sanitizer executions. The same run's hosted Go lanes and
existing Windows native, parser, and lexer jobs passed.

The previous [Frontend run 39768680761](https://github.com/0disoft/fairyc/actions/runs/39768680761)
at `54cbeb17ebdc28a2797cc19c6b45882381c6cc3a` remains historical evidence for its recorded
revision. Its expanded public Buffer/text fixture and four compiler/profile SDK executions are
described below.

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
| Pure bytes and number helpers | [SDK bytes](../../internal/sdkstdlib/sources/std/bytes.fy), [SDK num](../../internal/sdkstdlib/sources/std/num.fy) | [Equality](../../internal/cli/bytes_equal_test.go), [decimal parsing](../../internal/cli/checked_i64_decimal_test.go) check interpretation/emission; their native sections require Windows. No new target C symbols are needed, but full Linux execution coverage is not established here. |

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
[Frontend run 39774562874](https://github.com/0disoft/fairyc/actions/runs/39774562874) now covers
both byte operations in four compiler/profile SDK executions, including normal, empty, forced
payload and tracking-node OOM, recovery, preserved data, and zero live allocations. This receipt
establishes the current Linux runtime cases; the SDK probe is distinct from the separate 104
sanitizer executions. Related Go/fixture suites and vet pass locally. The local full repository Go
suite was not rerun, while the hosted Go lanes in the receipt passed.

## Remaining Phase 2 review

The mandatory minimum is the thin runtime plus representative `SystemHeap`, `Arena`, `Buffer`,
text/`String`/`CString`, and Linux `File` paths, with differential, layout, bidirectional ABI, and
sanitizer coverage ([roadmap](../product/01-roadmap.md#phase-2-first-hosted-executable-slice)).

| Requirement | Existing fixture owner | Actual gap |
| --- | --- | --- |
| Thin runtime and SystemHeap | [runner](../../internal/cbackendnative/runner.go), [trap locations](../../internal/cbackend/trap_location_native_test.go), [generated heap crossing](../../internal/cbackendnative/smoke_heap.go) | Current runtime/source probes pass. SystemHeap remains internal support by contract. |
| Arena | [Combined SDK source](../../internal/cbackendnative/sdk_storage_source.go); [Region byte tests](../../internal/workspacecompiler/region_bytes_test.go); [raw Arena](../../internal/cbackendnative/smoke_arena.go) | Public placement, rollback, reuse, and reset pass in v38. The current receipt also executes `alloc_zeroed_bytes` and `copy_bytes` with normal, empty, forced payload and tracking-node OOM, recovery, preserved data, and cleanup. |
| Buffer | Combined SDK check; [raw Buffer](../../internal/cbackendnative/smoke_buffer.go) | The combined fixture now executes public `empty`, `resize_zeroed`, `truncate`, and `view_mut`, including zeroing and regrowth. The raw fixture supplies the earlier byte-level and failure evidence. |
| Text | Combined SDK check; [raw owned text](../../internal/cbackendnative/smoke_owned_text.go), [raw text views](../../internal/cbackendnative/smoke_text_views.go) | The combined fixture now executes empty `String`, exact multibyte UTF-8, and successful/empty `from_buffer` roundtrips. Raw C supplies the byte-level cases. |
| Linux `File` | [Public File source](../../internal/cbackendnative/smoke_files.go), [raw Linux File](../../internal/cbackendnative/smoke_linux_file.go) | The representative safe File path passes four executions; partial I/O and close failure are raw evidence. The Windows-only operations listed above are outside the Phase 2 minimum. |
| Differential, layout, ABI, sanitizer | [runner](../../internal/cbackendnative/runner.go), [layout probes](../../internal/cbackendnative/probes_layout.go), [ABI probe](../../internal/cbackendnative/abi_probe.go) | The current receipt records 208 differential/drop comparisons, 4 layout probes, 56 ABI crossings, and 104 sanitizer executions for the admitted fixtures. |

The expanded `sdkStorageSource` covers those public `Buffer` and text cases in the current hosted
receipt; it requires no API or target-gate change.

## Next work

Complete the remaining Phase 2 contract-to-fixture review by mapping mandatory runtime/library rules
to fixtures or justified deferrals before considering conformance. The Region byte Linux evidence is
current, so no additional receipt is needed for those cases.
Both `complete-runtime` and `complete-standard-library`
remain missing in the [target spec](../../internal/target/spec.go); `emittable`, `conformant`,
and `supported` remain false. This receipt documentation task changes no API, runtime ABI, SDK
source bytes, DB, hygiene policy, CI runner allocation, or version. The earlier lexer job timeout
increase remains historical; no tests or runners were added.
