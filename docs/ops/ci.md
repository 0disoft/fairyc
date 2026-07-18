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
  Linux/amd64, glibc, Clang, and GCC; compiles all seventeen current C-emission cases at `-O0` and
  `-O2` with pedantic warnings as errors; links the scalar trap runtime plus the closed conformance
  drop recorder; plans 68 exact process-status and source-linked drop-sequence comparisons; and
  compiles and executes one recursive C11 probe for all 9 valid target-layout cases under the same
  4 compiler/profile pairs. It additionally recompiles every production C unit once per compiler
  with the closed `address-undefined-o1` ASan+UBSan mode, overrides ambient sanitizer options, and
  requires all 34 executions to produce no diagnostic. It
  then drives a resource-bearing Option-and-Result program through uninstrumented
  `experimental build` with Clang/GCC and low/optimized, executes all four resulting temporary
  products, and requires exit `42` with no output. A missing compiler or libc identity fails the job
  instead of producing a skip.
- The native job retains no generated C, object, executable, cache, or report artifact. Its bounded
  `fairyc.native-c-report/v4` JSON report is written only to the job log and contains toolchain
  fingerprints, fixture IDs, the sorted layout-case IDs, and drop-trace, layout-probe, and
  sanitizer, and product-build execution counts but no temporary path.

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
