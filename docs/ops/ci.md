# CI

- Status: Draft

## Current frontend gate

`.github/workflows/frontend.yml` runs on pushes and pull requests. It validates the Go frontend on
GitHub-hosted Ubuntu and Windows runners using the version declared by `go.mod`.

- Token permissions are limited to `contents: read`.
- Concurrent runs for the same workflow and ref cancel the older run.
- Retained GitHub Actions dependency and build caching remains disabled. The compiler has one small
  pinned TOML parser dependency, and clean runners may fetch it through ordinary `go -mod=readonly`
  resolution; no retained CI cache budget has been justified yet. Local configured validation keeps
  `GOPROXY=off`, reuses only the user's checksum-verified Go module download cache, and retains its
  build and temporary caches under the repository-local ignored `.cache/` root.
- `go vet -mod=readonly ./...` runs before `go test -mod=readonly -count=1 ./...`.
- After those ordinary checks, only the Windows matrix entry runs
  `cmd/fairyc-windows-host-smoke`. The command discovers the hosted runner's local x64 MSVC and
  Windows SDK without WSL, exercises single- and multi-source target-neutral paths plus the declared-target path, safe File and owned-text source lifecycles, layout, ABI,
  raw owned-text and Arena allocation-failure preservation, PE/COFF, and unwind probes, and retains no executable,
  object, generated C, PDB, or report artifact.
- The workflow uploads no build artifact and performs no deploy, release, publication, or write to
  repository state.
- Ordinary tests execute the fixed lexer and parser fuzz seeds. Timed fuzz campaigns remain
  unconfigured rather than pretending a short CI sample proves coverage.
- A separate Ubuntu native-C job invokes `cmd/fairyc-native-fixtures`. The command requires
  Linux/amd64, glibc, Clang, and GCC; compiles all 36 current C-emission cases at `-O0` and
  `-O2` with pedantic warnings as errors; links the scalar trap runtime plus the closed conformance
  drop recorder; plans 140 exact process-status and source-linked drop-sequence comparisons;
  exercises direct `i32` minimum transport plus explicit and omitted-bound immutable byte-view slicing through the same checked MIR and
  generated-C path; and compiles and executes one recursive C11 plus explicit POSIX-typedef probe for all 11 valid target-layout cases under the same
  4 compiler/profile pairs. It additionally recompiles every production C unit once per compiler
  with the closed `address-undefined-o1` ASan+UBSan mode, overrides ambient sanitizer options, and
  requires all 70 executions to produce no diagnostic. The same job generates separate caller and
  callee objects for all 14 C ABI pass plans and crosses Clang-to-GCC plus GCC-to-Clang at both
  profiles for 56 zero-output executions. The same matrix compiles and runs the provisional
  SystemHeap C probe four times, covering empty normalization, supported alignment, growth/shrink
  byte preservation, unsupported-alignment rollback, and release. A second four-run Buffer probe
  covers initialized-prefix growth, relocating self-overlap append, overflow rollback, zeroed
  resize, truncation, malformed-state rejection, drop, deterministic initialization/reserve/append/
  resize allocation failure, exact storage preservation, and live-allocation cleanup. A third
  four-run Arena probe covers
  exclusive Region attachment, zeroed allocation, copy, deterministic payload/node allocation
  rollback, live-allocation cleanup, reset invalidation, stale-use rejection, and drop. A fourth
  four-run text-view probe covers canonical UTF-8 acceptance and
  rejection plus code-point-boundary slicing. A fifth four-run owned-text probe covers String
  copy/move/append/drop, CString interior-NUL, termination, empty-value behavior, deterministic
  initial-allocation and append-reallocation failure, storage preservation, and live-allocation
  cleanup through a probe-only adapter. It
  then drives a resource-bearing Option-and-Result program through uninstrumented
  `experimental build` with Clang/GCC and low/optimized, executes all four resulting temporary
  products, and requires exit `42` with no output. A missing compiler or libc identity fails the job
  instead of producing a skip.
- The native job retains no generated C, object, executable, cache, or report artifact. Its bounded
  `fairyc.native-c-report/v31` JSON report is written only to the job log and contains toolchain
  fingerprints, fixture IDs, the sorted layout-case IDs, and drop-trace, layout-probe, and
  sanitizer, ABI case/direction, SystemHeap, Buffer, Arena, text, Linux File, scalar Fairy-source
  foreign-call, individual exact `c::int`, negative-preserving `c::long`, `c::size_t`, and
  negative-preserving target `c::ssize_t` producer/consumer foreign-calls, the remaining twelve-type
  C-scalar family crossing, raw-pointer foreign-call, register/hidden-memory struct-result
  foreign-call, exact SystemHeap block source-call, exact
  nested Buffer source-call, exact Arena/Region source-call, actual CStr validation source-call,
  actual owned String/CString source-call, actual Linux File source-call, actual
  text-validation source-call, actual text-view
  output-record source-call, compiler-owned byte-, UTF-8-, and C-string source-calls, ordinary product-build,
  automatically linked standard-library product-build, and compiler-owned byte-, UTF-8-, and C-string product-build
  execution counts but no temporary path.
  Each foreign-call count is four only after generated Fairy callers and hand-written C callees
  cross as separate Clang/GCC objects in both directions and profiles and return `42` with no output.
  The text-validation source-call count additionally requires linking the generated caller against
  `TextViewSource()` rather than a mock companion. The output-record count additionally requires
  the generated FY caller to pass a verified direct-local raw address and observe the C-written field.
  The Buffer source-call count requires dependency-ordered nested `@repr(C)` emission, separate
  caller/callee compiler objects, the real `StorageSource()`, zeroed initialized bytes, and normalized
  release.
  The Arena source-call count additionally requires exact self-linked records, a provider assertion
  that `uint64_t` is `unsigned long`, one `void **` output slot, the real `ArenaSource()`, zeroed
  allocation, released nodes, and generation transitions through reset and drop.
  The CStr validation count requires an exact generated view record, nonempty terminated,
  interior-NUL, and unterminated storage, the real `OwnedTextSource()`, and unchanged output after
  both rejected calls.
  The C-string literal source-call count requires generated `fairy\0` static storage to cross the
  explicit unsafe immutable-byte-pointer operation and be read byte-for-byte by a separate C callee.
  Its product-build count separately requires the ordinary CLI artifact to contain nonempty
  compiler-owned storage and execute with exit 42 under all four compiler/profile combinations.
  The byte-string literal source-call count separately requires exact non-terminated `fairy`
  storage, a private non-null `ref u8[]` view with length five, typed-local immutable data-address
  extraction, and a separate C callee reading all five bytes. Its product-build count requires the
  same view path to execute with exit 42 under all four compiler/profile combinations. Empty-slice
  sentinel behavior remains covered by host-independent emitter tests.
  The UTF-8 string literal source-call count requires exact `Fairy λ` storage, a distinct private
  non-null `ref str` view with byte length eight, typed-local immutable data-address extraction, and a
  separate C callee reading all eight UTF-8 bytes. Its product-build count requires the same text-view
  path to execute with exit 42 under all four compiler/profile combinations. Invalid decoded UTF-8
  and empty-text sentinel behavior remain covered by host-independent lexer and emitter tests.
  The struct-result count requires one 8-byte INTEGER-class return and one 20-byte MEMORY-class
  hidden return to cross separate caller/callee objects in both compiler directions and profiles.
  The owned-text source-call count requires the exact nested heap, Buffer, String, CString, view, and
  result records to cross the real `OwnedTextSource()` copy/view/append/drop and
  from-text/view/drop helpers, with observable bytes, terminator, statuses, and normalized release.
  The Linux File source-call count requires exact result, handle, and open-option records to cross
  the real `LinuxFileSource()` open, write-all, sync, close, read, and close helpers, with exact
  counts, repeated-close status, EOF, reconstructed bytes, and normalized closed state.
  The standard-library build-smoke count requires the ordinary product CLI to derive the text-view
  C component from the actual called symbol, supply no companion source, link the scalar runtime,
  and execute exit 42 under both compilers and profiles.

The containing Mustflow workspace provides local parity through `fairyc_go_vet` and
`fairyc_go_test`, plus the Linux-only `fairyc_native_c_test` intent. Those intents additionally
isolate ignored Go caches and disable module network access for local agent verification. The Linux
native intent remains unavailable on the current Windows host and no WSL path is used. The separate
`fairyc_windows_host_smoke` intent uses a bounded temporary source root to exercise both product
paths through locally discovered x64 MSVC: `build-host` must produce a bounded PE that directly
returns 42, and `run-host` must independently return 42 before every temporary artifact is removed.
The same smoke invokes `build-target` for `x86_64-windows-msvc` on the checked-in
`examples/windows_target/main.fy`. Target selection happens before C emission, the generated unit
uses the LLP64 profile and four-byte `c::long`, calls the real project-owned
SystemHeap, Buffer, Arena/Region, String/CString/CStr, and UTF-8 C components, constructs one
move-only pointer-width resource, and the resulting PE must return 42. The heap path allocates eight
bytes at alignment eight, observes the record, frees it, and
observes normalized zero state. The Buffer path selects SystemHeap exactly once, initializes
capacity eight, resizes metadata to length four, drops, and observes normalized zero state; it does
not inspect the zeroed bytes. The Arena path closes over the same SystemHeap component once,
initializes generation 1, begins one Region, obtains a non-null eight-byte allocation and node,
resets to generation 2 with invalidated Region state, and drops to generation 3. It cannot
dereference the returned pointer, so zeroed-byte and failure-matrix evidence remains in the Linux
probes. The target source caps fundamental alignment at the
documented Windows x64 `malloc` guarantee of 16 bytes. The resolver rejects every other foreign or
unproven standard-library symbol before discovering MSVC, so these are closed source-level
crossings rather than broad Windows FFI/runtime evidence. No WSL installation or execution is part
of this lane.
The same smoke copies the checked-in `examples/workspace_quickstart` into its bounded temporary
root, generates one canonical lock in ordinary mode, requires exact `--locked` reuse thereafter,
and compiles its root-to-dependency call through package-aware verified MIR. Low and optimized
`build-workspace` products are retained long enough for independent PE/unwind inspection and native
result 42. `run-workspace` then returns 42 once through the in-process CLI boundary and once through
the separately built `fairyc.exe`, retaining no target image. This adds no registry, solver,
network, WSL, or persistent cache path.
The owned-text path closes over SystemHeap, Buffer, and text view once. Compiler-owned `"Fairy"`
storage is copied to a length-five String, viewed, appended to length ten, viewed again, and dropped
to normalized storage. The same input produces six-byte CString storage, a logical length-five CStr
view, and normalized drop; compiler-owned `c"Fairy"` storage also passes direct bounded validation.
The lane requires zero operation/allocation statuses, non-null live pointers, expected lengths and
alignment, and cleared storage after drop. It does not independently dereference returned views.
The same Windows intent separately compiles the raw owned-text probe and target-specific
implementation as caller and callee translation units at low and optimized profiles. Those
executions observe exact bytes, the terminator, invalid UTF-8, interior NUL, empty CString, null and
malformed opaque storage, failure preservation, self-overlap, deterministic size overflow, and
normal plus repeated drop. The probe-only allocator also forces String/CString initial allocation
failure and String append reallocation failure, preserves the prior storage and bytes, and observes
the live-allocation count returning to zero. Product support source contains no probe controls.
It separately compiles the raw Buffer caller and instrumented target-specific Buffer/SystemHeap
callee at both profiles. Those executions force initialization, reserve, self-overlap append growth,
and zeroed-resize growth failure, preserve exact pointer/metadata/initialized bytes, prove recovery
zeroing, and return the live-allocation count to zero after drop.
It also compiles the raw Arena caller and instrumented target-specific Arena/SystemHeap callee
separately at both profiles. Those executions observe zeroed and copied bytes, force failure before
payload allocation and between payload and tracking-node publication, preserve caller/Arena state,
prove rollback to zero live allocations, and recheck stale Region plus active-drop rejection.
A separate checked-in two-module owned-text program keeps every raw declaration and unsafe cast in
its support module. Its safe entry constructs move-only String/CString resources, appends text,
checks byte lengths, accepts empty CString, rejects interior NUL with status 3, and relies on
verified cleanup. Low and optimized target builds each return 42 after PE and unwind inspection.
The five storage bridges use opaque byte-copy transport and do not create a stable public layout or
safe borrowed text view.
The same smoke invokes `run-target` twice at the optimized profile. The first call recompiles the
checked-in target-lifecycle source through the shared closed target path. The second first builds a
standalone Go `fairyc.exe` in a bounded temporary directory, verifies its help surface, and uses that
separate compiler process to compile and execute `examples/windows_quickstart/main.fy`. Both calls
validate temporary PE and unwind data, require the exact target-labelled native-result receipt for
42, retain no artifact, and use no WSL. The quickstart iterates a byte view with checked `usize`
arithmetic, while separate low and optimized cases require maximum-`usize` addition to trap.
Separate low and optimized `run-host` cases also require matching-type unsigned bitwise operations
and right shifts to return 42, and require a `usize` shift amount of 64 to report `invalid-shift`.
Together with the existing scalar and trap cases, the direct `run-host` table now has 40 outcomes,
including always-checked assertion success and failure at low and optimized profiles.
The smoke also invokes `exec-target` twice with entry `examples/windows_output/main.fy` and explicit
`--source examples/windows_output/windows_console.fy`: once through the in-process CLI test boundary
and once through the separately built `fairyc.exe`. That two-module FY program
requires empty and invalid-input results from the Windows-only stdout helper, writes exactly the 16
ASCII bytes `Fairy C says hi\n`, and exits 42. The CLI must forward those stdout bytes exactly,
produce no stderr, preserve status 42, inspect the temporary PE and unwind data, and remove the
image. The entry module is safe while the explicit support module owns the raw crossing; this is not
a portable `std::io` or public package-build receipt.
The raw declaration, safe text wrapper, and entry must all declare `console`; deleting the entry or
wrapper declaration must fail semantic analysis with `E7102` before MSVC is invoked.
The same intent derives one non-POSIX C11 unit from the eight valid Windows layout cases, then
compiles and executes it under low and optimized MSVC profiles. It also derives separate caller and
callee C11 translation units from all twelve valid Microsoft x64 plans, compiles and links them
separately, and executes the linked probe under both profiles. Those receipts are local Windows
host evidence only; they are not Linux native parity, a hosted Windows receipt, complete Windows
layout coverage, independent compiler/register inspection, PE/COFF object conformance, or complete
Windows call/ABI conformance. The same intent also emits one generated-Fairy caller for an 8-byte
and a 20-byte `@repr(C)` result, compiles its hand-written C callee separately, and requires exit 42
at both profiles under both C11 and C17 consumer modes. That is one same-MSVC source-level result crossing plus focused C17 compatibility evidence, not broad or bidirectional
Fairy FFI. It separately compiles the project-owned
Windows File implementation and its caller probe, then executes create/write/sync/close/reopen,
partial read, EOF, forced close failure, and repeated drop under both profiles. A separate checked-in
FY source pair then executes safe write/sync/read/EOF, consuming close, and implicit-drop release at
both profiles. It also compiles the raw owned-text caller and target-specific implementation as
separate translation units, exercises the failure-preservation and repeated-lifecycle matrix under
both profiles, and inspects both objects plus each linked image. It also compiles the catcher, stack-owning
termination frame, and private exception raiser as three separate translation units, links and
executes them under both profiles, and requires exact exception payload, ordered
`RtlVirtualUnwind` frame discovery, and one abnormal `__finally` execution. Before execution, the
same intent parses the target-neutral product image, both declared-target profile images, the two
package-workspace build images, both package-workspace run images, the two safe FY File lifecycle
images, the two safe FY owned-text lifecycle images, both direct run-target temporary images, both exec-target temporary images, and all eighteen raw or generated-Fairy Windows File, Buffer, owned-text, Arena, layout, ABI,
struct-result, and unwind probe images
and requires bounded AMD64 PE32+ non-DLL console
identity, aligned image/header/section ranges, a readable executable `.text`, and an executable
entry point. This does not inspect object symbols or relocations, unwind/debug payloads, imported
API compatibility, signing, installer packaging, or distribution behavior.
The driver also parses every intermediate `.obj` from `build-host`, every `run-host` case, and both
profiles of all native probes before linking results leave the temporary root. Raw AMD64 COFF,
at most 4,096 sections inside the 64 MiB object budget, bounded section and relocation ranges,
readable executable `.text`/`.text$...`, valid relocation
symbol indexes, and named symbols are mandatory; storage-free `.bss` is admitted only with the
uninitialized-data characteristic. Object bytes are not uploaded or retained. Exact symbol,
COMDAT, relocation-type, unwind, debug, and hosted object expectations remain unconfigured.
The ordinary Go tests include a 154-section positive regression object and reject 4,097 sections;
the exact section count emitted by a future MSVC version remains unstabilized.
All thirty-five linked target or probe PE images additionally traverse the Microsoft x64 exception directory before
execution. The local gate requires sorted nonoverlapping `.pdata` runtime-function ranges,
executable code coverage, DWORD-aligned file-backed unwind records, bounded version 1 prolog slots,
bounded version 2 epilog descriptors, and minimum handler/chained tails. The receipt reports the
retained product's runtime-function and version-2 record counts without stabilizing them. It
executes the closed synthetic exception and virtual-unwind walk described above, but does not
validate complete version-2 epilog semantics, language-specific handler payload parsing, recursive
chains, Fairy/foreign-frame crossings, object-level unwind relocations, or hosted runner behavior.

The Windows matrix now configures that same smoke after its ordinary Go test suite. Configuration
alone is not a hosted receipt: `hosted-windows-receipt` remains missing until GitHub records an
exit-zero run for the exact commit, and a later code or runner-image change requires a fresh run.

All Go jobs set `CGO_ENABLED=0`. The checked-in `.c` files are constrained-backend and runtime
inputs loaded by Go code as data; they are not cgo translation units. This keeps `go vet`, `go
test`, the external-driver Windows smoke, and the Linux C differential independent from whether a
runner image happens to expose a default C compiler. The native lanes still invoke their declared
Clang, GCC, or MSVC executables explicitly through `internal/cdriver`.

Third-party workflow execution is limited to official `actions/checkout` and `actions/setup-go`
v6 commits pinned by full SHA. Moving to a newer major version is a separate compatibility change,
not an automatic side effect of this CI repair.

## Owners

- Primary owner: UNASSIGNED
- Backup owner: UNASSIGNED
- Escalation path: UNDECIDED

## Validation

- Required validation names: VALIDATION.md
- Release blocker status: UNDECIDED; branch protection is external repository state and is not
  claimed by this document.
- Remaining operational risk: remote workflow execution, including the Linux native job and the
  Windows native target smoke, has not been observed at the current commit, and no timed fuzz
  campaign or coverage threshold exists.
