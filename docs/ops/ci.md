# CI

- Status: Draft

## Current frontend gate

`.github/workflows/frontend.yml` runs on pushes and pull requests. It validates the Go frontend on
GitHub-hosted Ubuntu and Windows runners using the version declared by `go.mod`.

- Token permissions are limited to `contents: read`.
- Concurrent runs for the same workflow and ref cancel the older run.
- Go dependency and build caching is disabled because the current module has no external dependency
  graph and the project has not justified retained CI cache storage.
- `go vet -mod=readonly ./...` runs before `go test -mod=readonly -count=1 ./...`.
- The workflow uploads no build artifact and performs no deploy, release, publication, or write to
  repository state.
- Ordinary tests execute the fixed lexer and parser fuzz seeds. Timed fuzz campaigns remain
  unconfigured rather than pretending a short CI sample proves coverage.
- A separate Ubuntu native-C job invokes `cmd/fairyc-native-fixtures`. The command requires
  Linux/amd64, glibc, Clang, and GCC; compiles all eighteen current C-emission cases at `-O0` and
  `-O2` with pedantic warnings as errors; links the scalar trap runtime plus the closed conformance
  drop recorder; plans 72 exact process-status and source-linked drop-sequence comparisons; and
  compiles and executes one recursive C11 plus explicit POSIX-typedef probe for all 11 valid target-layout cases under the same
  4 compiler/profile pairs. It additionally recompiles every production C unit once per compiler
  with the closed `address-undefined-o1` ASan+UBSan mode, overrides ambient sanitizer options, and
  requires all 36 executions to produce no diagnostic. The same job generates separate caller and
  callee objects for all 14 C ABI pass plans and crosses Clang-to-GCC plus GCC-to-Clang at both
  profiles for 56 zero-output executions. The same matrix compiles and runs the provisional
  SystemHeap C probe four times, covering empty normalization, supported alignment, growth/shrink
  byte preservation, unsupported-alignment rollback, and release. A second four-run Buffer probe
  covers initialized-prefix growth, relocating self-overlap append, overflow rollback, zeroed
  resize, truncation, malformed-state rejection, and drop. A third four-run Arena probe covers
  exclusive Region attachment, zeroed allocation, copy, rollback, reset invalidation, stale-use
  rejection, and drop. A fourth four-run text-view probe covers canonical UTF-8 acceptance and
  rejection plus code-point-boundary slicing. A fifth four-run owned-text probe covers String
  copy/move/append/drop and CString interior-NUL, termination, and empty-value behavior. It
  then drives a resource-bearing Option-and-Result program through uninstrumented
  `experimental build` with Clang/GCC and low/optimized, executes all four resulting temporary
  products, and requires exit `42` with no output. A missing compiler or libc identity fails the job
  instead of producing a skip.
- The native job retains no generated C, object, executable, cache, or report artifact. Its bounded
  `fairyc.native-c-report/v26` JSON report is written only to the job log and contains toolchain
  fingerprints, fixture IDs, the sorted layout-case IDs, and drop-trace, layout-probe, and
  sanitizer, ABI case/direction, SystemHeap, Buffer, Arena, text, Linux File, scalar Fairy-source
  foreign-call, individual exact `c::int`, negative-preserving `c::long`, `c::size_t`, and
  negative-preserving target `c::ssize_t` producer/consumer foreign-calls, the remaining twelve-type
  C-scalar family crossing, raw-pointer foreign-call, register/hidden-memory struct-result
  foreign-call, exact SystemHeap block source-call, exact
  nested Buffer source-call, exact Arena/Region source-call, actual CStr validation source-call,
  actual owned String/CString source-call, actual
  text-validation source-call, actual text-view
  output-record source-call, and product-build
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
  The struct-result count requires one 8-byte INTEGER-class return and one 20-byte MEMORY-class
  hidden return to cross separate caller/callee objects in both compiler directions and profiles.
  The owned-text source-call count requires the exact nested heap, Buffer, String, CString, view, and
  result records to cross the real `OwnedTextSource()` copy/view/append/drop and
  from-text/view/drop helpers, with observable bytes, terminator, statuses, and normalized release.

The containing Mustflow workspace provides local parity through `fairyc_go_vet` and
`fairyc_go_test`, plus the Linux-only `fairyc_native_c_test` intent. Those intents additionally
isolate ignored Go caches and disable module network access for local agent verification. The native
intent is unavailable on the current Windows host because no registered WSL distribution or C
compiler exists; ordinary Go validation is not native parity.

## Owners

- Primary owner: UNASSIGNED
- Backup owner: UNASSIGNED
- Escalation path: UNDECIDED

## Validation

- Required validation names: VALIDATION.md
- Release blocker status: UNDECIDED; branch protection is external repository state and is not
  claimed by this document.
- Remaining operational risk: remote workflow execution, including the new native job, has not been
  observed yet, and no timed fuzz campaign or coverage threshold exists.
