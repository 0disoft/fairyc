# Contributing

Thank you for your interest in contributing to Fairy C!

## Prerequisites

- **Go**: Use the version specified in `go.mod`. Dependencies are managed standardly through `go.mod` and `go.sum`.
- **C Toolchain** (required for native tests):
  - **Windows**: x64 MSVC and the Windows SDK.
  - **Linux**: amd64/glibc environment with GCC and Clang available on `PATH`.

Standard development does not require Bun, PowerShell, or external cache directories.

## Building and Testing

C source files included within Go packages are embedded compiler resources. Set `CGO_ENABLED=0` so Go does not attempt to compile them as cgo files:

### Linux / macOS (bash/zsh)
```sh
export CGO_ENABLED=0
go build -mod=readonly -o ./build/fairyc ./cmd/fairyc
go vet -mod=readonly ./...
go test -mod=readonly ./...
```

### Windows (PowerShell)
```powershell
$env:CGO_ENABLED = '0'
go build -mod=readonly -o ./build/fairyc.exe ./cmd/fairyc
go vet -mod=readonly ./...
go test -mod=readonly ./...
```

> **Note**: `tools/run-go.ts` is an internal script for offline maintainer builds; contributors should use the standard `go` CLI directly.

## Running Focused Tests

When working on specific runtime or compiler components, you can target individual packages:

```sh
go test -mod=readonly ./internal/cstdlibcontract ./internal/cstdlib ./internal/sdkstdlib
go test -mod=readonly -run '^TestTaskManager' -v ./internal/cli
go test -mod=readonly ./internal/adoptionaudit ./internal/testprocess
```

Task manager tests copy sample Fairy sources into a temporary directory (`t.TempDir`), lower them through the compiler, and run native builds across both debug and release configurations. Platform-specific tests are automatically skipped on unsupported operating systems.

## Native Backend Selection

Commands such as `fairyc build`, `fairyc run`, and `fairyc test --native` automatically select the host's supported C compiler:
- On Windows, MSVC is used.
- On Linux (amd64/glibc), GCC is preferred, falling back to Clang if GCC is unavailable (see [Task Manager CLI](examples/task_manager_cli/README.md)).

To test the Linux CLI and prepared file replacement workflow together:

```sh
FAIRYC_TEST_EVIDENCE_DIR="$PWD/.cache/linux-native-evidence" CGO_ENABLED=0 go test -mod=readonly -count=1 -v -run '^(TestStableNativePortableExecution|TestStableLinux.*|TestTaskManagerPortableNativeSmoke|TestTaskManagerLinuxPreparedSaveFaults|TestLinuxPreparedPublication|TestExecuteLinux.*)$' ./internal/cli ./internal/cstdlib ./internal/cdriver
```

*(This requires a Linux environment with GCC and Clang installed.)*

## Historical Tools and Artifacts

The `.ps1` scripts under `tools/` and audit tools under `tools/dev/fairyc-evidence-audit` are historical validation utilities. They are not required for regular development or testing:

```sh
go run -mod=readonly ./tools/dev/fairyc-evidence-audit \
  -kind ledger -receipt /path/to/run/receipt.json -compiler /path/to/retained/fairyc
```
