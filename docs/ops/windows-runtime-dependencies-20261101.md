# Windows CRT and runtime DLL dependencies - 2026-11-01

Status: Windows verified for the named paths. Linux remains deferred; clean-machine deployment
and exhaustive execution-path coverage are not claimed.

## Contract and implementation

The private dependency inspector (`00a82403`) starts with the existing bounded AMD64 PE console
image checks. It reads ordinary import descriptors, DLL basenames, name/ordinal functions and
IAT RVAs, including the missing-OriginalFirstThunk fallback. Descriptor/thunk/string termination,
mapped ranges, integer overflow, empty names and invalid ordinal flags are checked. Bound and
delay-import directories are explicit errors, including partially populated directory entries.
The named images have neither form. Fourteen malformed-input mutations fail. This is not an
OS DLL parser, a loader implementation or complete dependency analysis for arbitrary PE images.

The separate `/MT` native tracer (`6b906918`) launches each target with
`DEBUG_ONLY_THIS_PROCESS`, records LOAD_DLL and UNLOAD_DLL events through process exit, and
preserves exit status and exact stdout/stderr bytes. The first loader breakpoint is continued;
other exceptions are passed back to normal handling. A job object, native 30-second deadline
and Go 40-second deadline bound the process and clean it up. The tracer is never linked into
the target. Debugger execution is the named observation, not proof of every non-debugger path.

DLL paths come from the event file handle, with a process module-path fallback. Each path must
be a real file directly under the API-queried Windows system directory; its SHA-256 is recorded
and rechecked. No application-local DLL or arbitrary PATH lookup is admitted in this corpus.
The child receives only SystemRoot/WINDIR, empty PATH and private temporary directories; it
inherits no compiler LIB/LIBPATH/INCLUDE, user home or symbol-server environment. Standard input
is NUL and stdout/stderr use separate files. The helper preserves empty, quoted, spaced and
Unicode child arguments, as checked by the existing `process_args_quickstart` workload.

For each ordinary import, a separate system-loader reference lookup uses
`LOAD_LIBRARY_SEARCH_SYSTEM32`, checks every named/ordinal export, and requires the returned
module path/hash to occur in the actual child trace. This is a reference loader check, not an
inspection of the child's patched IAT or every forwarded-function target. DLL API-set contract
names may resolve to host DLLs through that loader; none occurs in this measured direct-import
corpus. Child load/unload events are retained separately, so a transient module cannot disappear
from the evidence just because it was absent in a later snapshot.

## Environment and CRT ownership

Observed Windows build: **26300**, x64; MSVC **14.51.36231**; SDK/UCRT library path
**10.0.26100.0**; local Go **go1.27.1**. `go.mod` still requires Go 1.26.0 and is unchanged.
These are observed inputs, not new pinned toolchain policy or minimum deployment requirements.

The five generated-Fairy SDK workspaces use the ordinary C11 `/MT` builder in both low and
optimized profiles. Actual object defaults, searched archive paths, loaded archive members,
final map symbols and PE entry prove these bindings in all ten images:

| Static archive | Owned final symbol | Loaded member |
| --- | --- | --- |
| `libcmt.lib` | `mainCRTStartup` | `exe_main.obj` |
| `libucrt.lib` | `__acrt_initialize` | `initialization.obj` |
| `libvcruntime.lib` | `__vcrt_initialize` | `initialization.obj` |

Absolute archive paths and SHA-256 are retained with each observation. The direct import table
of every generated image names only `kernel32.dll`: 70 functions per image, except the vector
example's 71. There is no direct `ucrtbase.dll`, `api-ms-win-crt-*`, `vcruntime*` or `msvcp*`
import. This confirms the selected static CRT mode for this corpus, without admitting `/MD`,
`/MTd`, `/MDd`, C++ runtime or another compiler policy.

Actual loader traces still contain OS DLLs. The native cases load six modules:
`ntdll.dll`, `kernel32.dll`, `KernelBase.dll`, `apphelp.dll`, `kernel.appcore.dll` and
`msvcrt.dll`; one unload event occurs. The system `msvcrt.dll` observation is recorded separately
from the selected static UCRT/VCRuntime archives. A DLL appearing in a trace does not by itself
identify a direct `/MD` linkage or application-local redistributable requirement.

## Named runtime paths

| Path | Executions | Evidence |
| --- | ---: | --- |
| bytes, allocation, text, vector and process-argument SDK workspaces | 10 | Actual generated Fairy/C runtime, both profiles, exact ordinary output/exit, CRT ownership and import resolution |
| Original Windows rooted-file adapter | 2 | Real create/write/rename/missing-file NTSTATUS path; all three native exports exercised |
| Rooted adapter changed-copy missing-export controls | 6 | Each of three export names replaced with a nonexistent name in a fixture-only copy, both profiles; original error-127 guards exercised |
| Retained-PDB developer lookup | 2 | Fresh Go tool process, both PDB profiles, exact generated `main` lookup and seven real DbgHelp calls |
| Fresh Go CLI bootstrap `--help` | 1 | Real `cmd/fairyc` binary, clean runtime environment, successful help output |

The manifest contains **21** rows. It does not count the two native tracer smoke controls,
separate PSAPI query, or two loader-failure executions as successful matrix rows.

The original `candidatehost.WindowsSource` is used for rooted operations. Its three declared
runtime exports are `ntdll!NtCreateFile`, `NtSetInformationFile` and `RtlNtStatusToDosError`.
An explicit inventory rejects a new undeclared lookup in that source. Creation and rename
exercise the first two; a missing-file operation exercises NTSTATUS-to-Win32 error mapping.
The C driver uses `wmain` only for lossless fixture paths. The generated-Fairy `main` CRT
archive proof above is not incorrectly applied to that different startup routine.

The developer lookup loads the system `dbghelp.dll` and exercises `SymInitializeW`, `SymCleanup`,
`SymSetOptions`, `SymGetOptions`, `SymLoadModuleExW`, `SymFromAddr` and `SymGetLineFromAddrW64`.
Its direct Go import table has 48 `kernel32.dll` functions, while the observed DLL set contains
16 modules, including system `ucrtbase.dll`, `msvcp_win.dll`, COM libraries and Winsock. The Go
CLI help path has the same direct import count and 11 observed system modules. Their indirect
system-UCRT observation is retained; neither path proves every future Go CLI operation.

The separate in-test-process PSAPI query calls the actual memory probe, checks positive memory
peaks and verifies `psapi!GetProcessMemoryInfo` and the system file hash. Commit `f34827f3`
changes that existing lazy load to `windows.NewLazySystemDLL`, eliminating application/PATH
search from that private measurement consumer. It is not counted as a child DLL trace.

## Failure behavior and evidence

A changed native image importing missing `fairy404.dll` really exits with **0xc0000135**;
another changed image with a nonexistent named import exits with **0xc0000139**. Both fail the
reference resolution check too. Error dialogs are suppressed in the isolated tracer. No system
DLL is renamed, removed or modified. Runtime missing-export controls return failure and
`ERROR_PROC_NOT_FOUND` (127); the test wrapper then exits 42 after validating that failure.
They are not successful calls to the missing API. A nonexistent native export, missing system
file, non-system owner and changed expected file hash are separately rejected.

The private `fairyc.windows-dependencies/v1` JSON records source revision, OS build, Go/MSVC
versions, tracer-source hash, each image/unit/runtime hash, imports and reference resolutions,
all DLL paths/bases/hashes and unload bases, exact stream hashes, exit codes, static CRT anchors,
archive identities, declared native/DbgHelp resolutions and the separately labelled PSAPI
observation. It is local evidence, not a stable public deployment manifest or signed provenance.
Loader-negative results are retained in the verbose test log, outside those 21 JSON rows.

Local ignored outputs:

- `.cache/windows-dependencies-20261101.json`
- `.cache/windows-dependencies-final-20261101.log`

Passed commands on Windows:

```powershell
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindowsDLLLoadTrace$' -count=1 -v
$env:FAIRYC_DEPENDENCY_EVIDENCE = Join-Path $PWD '.cache/windows-dependencies-20261101.json'
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^(TestWindowsDependency(Corpus|MissingInputs)|TestProcessMemoryProbeReadsPositiveWindowsPeaks)$' -count=1 -v
Remove-Item Env:FAIRYC_DEPENDENCY_EVIDENCE
go test -mod=readonly ./internal/pecoff -count=1
go vet -mod=readonly ./internal/pecoff ./tools/dev/fairyc-windows-host-smoke
git diff --check
```

Linux/WSL remains batched by request. Full `go test ./...`, full host smoke, hosted CI, arbitrary
native compiler paths and clean-machine transfer were not run. These results do not promote
the target or close full deployment, transitive-import parsing, bound/delay imports, Windows
version-range, third-party DLL or all-path runtime-load coverage. Public API/SDK ABI, DB,
dependencies, production compiler flags, repository hygiene rules, CI workflows and runners
are unchanged. No push, workflow dispatch, publication or deployment ran. The ABI evidence and
ordered-work documents were updated for the bounded Windows result.

Platform basis: [CRT library choices](https://learn.microsoft.com/en-us/cpp/c-runtime-library/crt-library-features),
[DLL debugging events and handle cleanup](https://learn.microsoft.com/en-us/windows/win32/api/debugapi/nf-debugapi-waitfordebugevent),
[system-directory loading](https://learn.microsoft.com/en-us/windows/win32/api/libloaderapi/nf-libloaderapi-loadlibraryexw).
