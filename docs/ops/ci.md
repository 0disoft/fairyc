# CI

- Status: Draft

## Current frontend gate

`.github/workflows/frontend.yml` runs on pushes and pull requests. It validates the Go frontend on
GitHub-hosted Ubuntu and Windows runners using the version declared by `go.mod`.

The first exact hosted native receipt is GitHub Actions run `30442310794` for commit `0bb4199`,
completed successfully on 2026-07-29. ADR 0214 records its Ubuntu frontend, Windows frontend/native,
and Linux C-differential jobs. This observation removes the two hosted-receipt markers but does not
pin a runner image or toolchain baseline and does not promote either target.

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
  Linux/amd64, glibc, Clang, and GCC; compiles all 52 current C-emission cases at `-O0` and
  `-O2` with pedantic warnings as errors; links the scalar trap runtime plus the closed conformance
  drop recorder; plans 208 exact process-status and source-linked drop-sequence comparisons;
  exercises direct `i32` minimum transport plus explicit and omitted-bound immutable byte-view slicing through the same checked MIR and
  generated-C path; and compiles and executes one recursive C11 plus explicit POSIX-typedef probe for all 11 valid target-layout cases under the same
  4 compiler/profile pairs. It additionally recompiles every production C unit once per compiler
  with the closed `address-undefined-o1` ASan+UBSan mode, overrides ambient sanitizer options, and
  requires all 104 executions to produce no diagnostic. The same job generates separate caller and
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
  `fairyc.native-c-report/v32` JSON report is written only to the job log and contains toolchain
  fingerprints, fixture IDs, the sorted layout-case IDs, and drop-trace, layout-probe, and
  sanitizer, ABI case/direction, SystemHeap, Buffer, Arena, text, Linux console, Linux File, scalar Fairy-source
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
the separately built `fairyc.exe`, retaining no target image. The same standalone compiler runs the
root `tests/answer.fy` through stable `test --locked`, requires its exact one-pass report, then
runs text and `test --list --json --locked` discovery and requires their canonical identities,
schema, count, and verified lock state without executing them, then runs a
failing-then-passing native pair selected by reverse-ordered exact identities and the
checked-in exact selected test at low and optimized MSVC profiles. It also selects the local answer
package by manifest UID, requires its semantic JSON report, and runs its member-owned test under
low and optimized MSVC. Every native test uses a separate inspected temporary image and the
retained stable build remains byte-identical. The semantic lane invokes no target toolchain; the
native lane adds no registry, solver, network, WSL, persistent cache, or retained artifact path.
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
That same standalone executable invokes `experimental check-workspace-cache` in nine cold/warm
pairs with a fresh explicit cache directory per pair and mandatory limits. Each invocation is a new
process. Cold receipts must report two misses, lowering, and two retained stable records; paired warm
receipts must retain the same executable build identity and report two verified hits with no
lowering, rejection, eviction, skipped write, or store failure. The caller preserves all 18 raw
process-start-through-exit durations with zero retries. This is configured local persistence and
distribution evidence, not a reference-machine speedup or default-policy decision.
The ordinary Go suite separately kills a helper process at the real disk-store boundary after
temporary-file sync and after durable publication. A new store must see respectively a clean miss
or one canonical hit. This checks process-crash outcomes and platform publication calls; CI does not
cut power to runner storage or certify the runner's filesystem and controller flush behavior.
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
The manifest-owned `examples/std_io_quickstart` is separate from that legacy target-only path. The
smoke runs it through stable locked release execution both in-process and through a separately built
compiler, requires exact `Fairy C says hi\nbytes on stdout\n-2147483648\n` stdout, exact
`Fairy C warns too\nbytes on stderr\n` stderr, and status zero, and also builds and directly
executes the retained PE with EOF, `stdin!\n`, and `123456789`. Those inputs must produce no echo,
the exact short echo, and only `12345678`, respectively, after the integer line; a separately built compiler repeats the
short-input artifact path. Stable run also receives `stdin!\n` through both the in-process CLI and
the separately built compiler and must emit the exact echo. Input is attached only after successful
compilation and inspection; a unit regression requires an invalid workspace to consume zero reads.
Stable run's output streams share the driver's 64 KiB capture ceiling and do not claim cross-stream
ordering.
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
Windows File implementation and its caller probe, then executes create/write/partial-read/exact-read/size/kind/identity/position/absolute-seek/cursor-preserving-length-change,
overflow rejection with cursor preservation, sync/close/reopen, partial read, EOF, forced close
failure, exact invalid-handle size reporting, repeated drop, exclusive-create collision with
unchanged-byte reopening, seek-resistant append concatenation, disk/`NUL` character kinds, and
closed/invalid-handle kind failures, complete exact reads, unexpected-EOF prefix preservation,
untouched suffix bytes, exact-read closed/invalid-handle failures, equal same-file identities,
unequal distinct-live-file identities, null-output rejection, and unchanged identity failure output
under both profiles. A
separate checked-in FY source pair and the public SDK quickstart execute safe partial/exact reads
and shared size, kind, and identity observation,
write/position/seek, missing-file open-or-create plus preserving reopen, exclusive create plus exact
existing-path failure and unchanged reopen, append creation plus exact EOF concatenation after
repositioning, existing-file update-open, shrink/extend with explicit rewrite, oversized and
read-only length failure, exact missing-file rejection, sync/read/EOF, consuming close, and implicit-drop
release at both profiles. It also compiles the raw owned-text caller and target-specific implementation as
separate translation units, exercises the failure-preservation and repeated-lifecycle matrix under
both profiles, and inspects both objects plus each linked image. It also compiles the catcher, stack-owning
termination frame, and private exception raiser as three separate translation units, links and
executes them under both profiles, and requires exact exception payload, ordered
`RtlVirtualUnwind` frame discovery, and one abnormal `__finally` execution. Before execution, the
same intent parses the target-neutral product image, both declared-target profile images, all thirteen
package-workspace build images, all nine package-workspace run images, the two safe FY File lifecycle
images, the two safe FY owned-text lifecycle images, both direct run-target temporary images, both exec-target temporary images, and all twenty-four raw or generated-Fairy Windows File, Buffer, owned-text, Arena, layout, ABI,
REL32-link, archive/COMDAT-selection, struct-result, and unwind probe images
and requires bounded AMD64 PE32+ non-DLL console
identity, aligned image/header/section ranges, a readable executable `.text`, and an executable
entry point. This does not inspect object symbols or relocations, unwind/debug payloads, imported
API compatibility, signing, installer packaging, or distribution behavior.
The driver also parses every intermediate `.obj` from `build-host`, every `run-host` case, and both
profiles of all native probes before linking results leave the temporary root. Raw AMD64 COFF,
at most 4,096 sections inside the 64 MiB object budget, bounded section and relocation ranges,
readable executable `.text`/`.text$...`, valid primary relocation symbols and AMD64 types, and exact
COMDAT definitions are mandatory; storage-free `.bss` is admitted only with the uninitialized-data
characteristic. Object bytes are not uploaded or retained. ADR 0231 fixes exact synthetic primary
symbols, relocation receipts, and COMDAT selection. ADR 0232 additionally recomputes one direct
cross-object `CALL rel32` from the object addend and final preferred-image addresses at both
profiles. ADR 0233 parses both Microsoft archive linker members, checks exact demand and unused
member indexes and raw candidate values, requires the demanded function's final-map owner to be the
indexed archive member, requires the unused non-COMDAT sentinel to be absent, and binds one pick-any
COMDAT map owner to its final PE address and value across four low/optimized input-order executions.
ADR 0234 preserves exact object section indexes, captures the relocation probe's MSVC map, and
recomputes one receipt `ADDR64` value plus all three unwind `.pdata` `ADDR32NB` RVAs across the same
two low/optimized executions. ADR 0235 binds the selected archive function root to its direct
associative `.pdata` and `.xdata` children and final runtime-function entry across the existing four
archive links. ADR 0236 proves both ANY-COMDAT function parents contributed, binds the winner, and
requires the losing unwind payload to be absent from every final reference and file-backed section.
ADR 0240 additionally binds the low `__GSHandlerCheck_SEH` and optimized
`__C_specific_handler` unwind tails from `frame.obj` through exact MSVC map rows to executable
final-image RVAs before the native exception probe runs. ADR 0241 reuses the relocation images to
bind one missing weak alias to its local fallback and one present alias to a strong `target.obj`
definition, then executes both paths. ADR 0242 parses one 65,535-record extended relocation table,
binds `overflow.obj` through the map to the final semantic REL32 displacement, and executes its leaf function
in the same two images. ADR 0243 parses one bounded version-2 `bigobj.obj`, binds its 20-byte-symbol
REL32 through the map, and executes its leaf function in the same images. ADR 0244 versions the
generated weak, overflow, and bigobj objects by complete-byte digest and structural receipt. ADR
0245 proves NOLIBRARY suppresses one available archive member while LIBRARY extracts another.
Other relocation formulas, high-section and bigobj
auxiliary/COMDAT evidence, and chained
unwind tails, non-unwind and broader archive/COMDAT
selection and retention modes,
debug data, and a fresh hosted same-commit object receipt remain unconfigured.
The ordinary Go tests include a 154-section positive regression object and reject 4,097 sections;
the exact section count emitted by a future MSVC version remains unstabilized.
All fifty-nine linked target or probe PE images additionally traverse the Microsoft x64 exception directory before
execution. The local gate requires sorted nonoverlapping `.pdata` runtime-function ranges,
executable code coverage, DWORD-aligned file-backed unwind records, bounded version 1 prolog slots,
bounded version 2 epilog descriptors, and minimum handler/chained tails. The receipt reports the
retained product's runtime-function and version-2 record counts without stabilizing them. It
executes the closed synthetic exception and virtual-unwind walk described above, but does not
validate complete version-2 epilog semantics, language-specific handler payload parsing, recursive
chains or chained-tail object relocations, Fairy/foreign-frame crossings, or hosted runner behavior.

The Windows matrix runs that same smoke after its ordinary Go test suite. Run `30442310794` is the
first recorded exit-zero hosted receipt. A later native-code, workflow, or runner-image change
requires fresh hosted evidence; the observation is not a stable MSVC or SDK baseline.

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
- Remaining operational risk: ADR 0214 records one exact green Linux/Windows native workflow run,
  but future native-code, workflow, or runner-image changes require fresh receipts, and no timed
  fuzz campaign or coverage threshold exists.
