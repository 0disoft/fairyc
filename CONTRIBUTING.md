# Contributing

Use the Go version declared in `go.mod`. Linux native tests require amd64/glibc
and GCC plus Clang on PATH; Windows native tests require x64 MSVC and its SDK.
Go downloads the dependencies declared in `go.mod`/`go.sum` through its normal
module mechanism. There is no requirement for Bun, PowerShell, a maintainer's
Downloads directory or an existing `.cache` tree for the commands below.

The C files in Go package directories are embedded compiler inputs. Set
`CGO_ENABLED=0` for ordinary builds/tests so Go does not treat them as cgo sources.

```sh
export CGO_ENABLED=0
go build -mod=readonly -o ./build/fairyc ./cmd/fairyc
go vet -mod=readonly ./...
go test -mod=readonly ./...
```

From PowerShell, use `$env:CGO_ENABLED='0'` instead of `export`; Go commands are
otherwise identical (an explicit Windows compiler output can end in `.exe`).
The optional `tools/run-go.ts` is a maintainer's offline/local-cache adapter,
not the contributor entry point. A fresh fork does not need its caches.

For a quick application/runtime change, run the related packages:

```sh
go test -mod=readonly ./internal/cstdlibcontract ./internal/cstdlib ./internal/sdkstdlib
go test -mod=readonly -run '^TestTaskManager' -v ./internal/cli
go test -mod=readonly ./internal/adoptionaudit ./internal/testprocess
```

Task manager tests copy checked-in Fairy sources to `t.TempDir`, lower them with
the current compiler and build native low/optimized images. Each CLI command
starts a fresh bounded process and compares exact UTF-8 stdout/stderr, exit code
and store/sidecar bytes. Platform-only tests explicitly skip on the other OS.
Ubuntu's existing Go CI lane runs the Linux publication/argv probes; Windows's
lane runs the existing prepared-save write/sync/close/publication fault and
bounded-kill probes. No new CI job is needed.

Ordinary `fairyc build`, `fairyc run` and `fairyc test --native` select the current
supported host backend. On Linux GNU amd64/glibc, GCC is preferred and Clang is
used when GCC is absent; see [the task manager](examples/task_manager_cli/README.md).
Do not infer observed Linux execution or target conformance from a Go cross-build
or a skipped test.

To check the Linux CLI and prepared-save path together without repeating every
compiler experiment, use the existing Go tests:

```sh
FAIRYC_TEST_EVIDENCE_DIR="$PWD/.cache/linux-native-evidence" CGO_ENABLED=0 go test -mod=readonly -count=1 -v -run '^(TestStableNativePortableExecution|TestStableLinux.*|TestTaskManagerPortableNativeSmoke|TestTaskManagerLinuxPreparedSaveFaults|TestLinuxPreparedPublication|TestExecuteLinux.*)$' ./internal/cli ./internal/cstdlib ./internal/cdriver
```

This requires a real Linux host with GCC and Clang. Save probes retain their image
hashes, source/SDK identities and exact process/file observations under the evidence
directory. No test installs dependencies, fills disks or claims power-loss durability.

The six remaining `.ps1` files under `tools/` reproduce historical Windows
adoption experiments against explicitly supplied external projects/artifacts.
They are optional, retain their historical SDK stage0.78 identity checks, and
are not prerequisites for building or testing a fork. Generic process/output
tests, task-manager CLI tests and retained-evidence auditing have moved to Go.
Historical receipts remain unchanged. To audit one supplied artifact tree:

```sh
go run -mod=readonly ./cmd/fairyc-evidence-audit \
  -kind ledger -receipt /path/to/run/receipt.json -compiler /path/to/retained/fairyc
```

The tool only reads files and emits JSON to stdout. It never executes the retained
compiler or product. Use `-path-map-from <old-root> -path-map-to <new-root>` when
relocating an evidence tree containing absolute paths; retain its archived
harnesses, images, original protected files and raw logs. Normal development
tests do not require these private historical artifacts.
