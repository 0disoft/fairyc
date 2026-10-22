# Linux runtime and SDK coverage

Audit: 2026-10-22, source `4f7798689bcd6a6ef797c9079ed4e28092647cca`,
SDK `0.0.0-stage0.86`, language `/20`, runtime `/0`.
This inventory supports the [Phase 2 review](phase-status.md); it does not promote target maturity.
The [runtime contract](../runtime/runtime-stdlib-v0.md), Accepted ADRs, and owning fixtures retain authority.

## Execution evidence

[Frontend run 39589657672](https://github.com/0disoft/fairyc/actions/runs/39589657672)
passed all nine jobs at the source revision above on 2026-10-21.
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
Local Windows tests passed; a hosted Linux v38 execution receipt is still pending.

The current [standard-library build smoke](../../internal/cbackendnative/smoke_build.go)
uses a [raw empty-input UTF-8 call](../../internal/cbackendnative/smoke_text_views.go).
Its `standardLibraryBuildSmokeExecutions: 4` does not establish those public SDK paths.
The [Windows generated failure probes](../../tools/dev/fairyc-windows-host-smoke/generated_failure_probes.go)
provide existing source cases to adapt rather than inventing new APIs.

## Next slice

Collect the v38 Linux receipt, then map remaining mandatory runtime/library rules to explicit fixtures or justified
deferrals before considering conformance. Both `complete-runtime` and `complete-standard-library`
remain missing in the [target spec](../../internal/target/spec.go); `emittable`, `conformant`,
and `supported` remain false. The fixture changes no product API, runtime implementation, DB,
hygiene policy, or CI runner allocation.
