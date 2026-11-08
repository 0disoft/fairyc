# Minimal Windows GUI build, 2026-11-08

Owner: [ADR 0552](../adr/0552-minimal-windows-gui-build.md) and
[CLI command contract](../cli/command-contract.md).

## Result

Go CLI and the authored Fairy CLI/companion build and run the
[Win32 example](../../examples/win32_minimal/README.md) on Windows amd64/MSVC.
Fairy retains `main()`; its explicit C bridge registers the window class,
creates and paints a native window, dispatches messages and destroys it.
SDK `rc.exe` supplies the string table. `--smoke` schedules automatic closure.
No browser/UI framework or additional application runtime is bundled.

| Target profile | Go CLI image | Authored CLI image |
|---|---:|---:|
| development | 111,616 bytes | 111,616 bytes |
| release | 111,104 bytes | 111,104 bytes |

These are measured example sizes, not size limits or a cross-toolchain promise.
The authored CLI host and authored companion were bootstrapped with the Go
frontend, both in the low host profile. Each builds applications in both target
profiles. Application build/run then executes Fairy frontend and companion
binaries, with the companion's pinned C runtime asset pack. This is not a
whole-compiler self-host or fixed-point claim.

The companion's JSON loader builds buffers and fixed metadata in locals before
constructing the owning project. This respects the existing closed mutable
resource-field projection contract. The vector push wrapper reduction in
[the separate record](windows-gui-vector-materialization-20261108.md) retains
the 1,048,576 materialization cap. No compiler budget was raised.

## Executed checks

All commands ran locally with `GOMAXPROCS=1`.

- `go test -mod=readonly ./internal/windowsbuild ./internal/pecoff
  ./internal/candidatebuild ./tools/candidate/fairyc-candidate-build -count=1`:
  PASS, including strict input loading, PE regressions and private packet build.
- `go test -mod=readonly ./internal/sdkstdlib -run
  '^TestVectorSharedChecksPreserveConcreteElements$' -count=1`: PASS, scalar,
  struct and owning resource source/Flow/verified MIR (0.558s).
- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(WorkspaceBuild|Resources)$' -count=1`: PASS (9.682s).
  Development/release build/run, GUI PE selection, console-only rejection,
  actual `user32.lib` linkage, no attached console, `.rc` string loading and
  preservation of the last executable on C compilation failure.
- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|AuthoredDriver)$' -count=1 -v`: initial window and
  companion checks PASS (18.620s). The Go example creates, paints and closes a
  window in both profiles; companion packet checks also cover resource failure
  preservation and ordinary console output.
- Final expanded `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUIAuthoredDriver$' -count=1 -v`, with
  `FAIRYC_RUN_PARITY_INTEGRATION=1`: PASS (135.014s). It freshly bootstraps the
  Fairy companion, tests both GUI target profiles, invalid resource preservation,
  console packets both without a selected root and with an absent config, then
  builds the complete authored CLI (210 C parts). Replacing its Go bootstrap
  builder with the Fairy companion plus pinned pack passes public `build`,
  `run -- --smoke`, `build --release`, `run --release -- --smoke`.
- From `examples/lexer_candidate`,
  `go run -mod=readonly ../../cmd/fairyc check`: PASS, 777 sources / 9,482 MIR
  functions; canonical lock refreshed.
- From `examples/win32_minimal`,
  `go run -mod=readonly ../../cmd/fairyc build --release`: PASS; local standalone
  `build/win32-minimal.exe` is retained in ignored build output.
- `gofmt` on the new Go test and scoped `git diff --check`: PASS.

Local logs are `.cache/windows-gui-packages-20261108.log`,
`windows-gui-final-go-20261108.log`, `windows-gui-authored-test-20261108.log`,
`windows-gui-authored-cli-test-20261108.log`, and
`windows-gui-final-check-20261108.log`. Optional test artifact directories use
`FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS` and `FAIRYC_CLOSURE_CLI_ARTIFACTS`.
The source tests and contracts own behavior; ignored logs/images are receipts.

## Limits and handoff

The first full source checks exposed E4503 and attempts to mutate owning
resource fields exposed E1008; both were resolved before the passing checks.
An old authenticated compiler rejected the changed vector SDK source during
an attempted companion rebuild; final verification uses fresh source builds
and does not reuse that failed packet or claim it passed.

Full `go test ./...`, optimized compiler-host bootstrap, Linux replay, hosted CI,
clean-machine installation and full fixed-point/self-host were skipped: this
batch targets the minimal Windows GUI path and relevant regressions. Appearance,
keyboard accessibility and DPI behavior were not manually reviewed. No skipped
run is counted as a pass. Arbitrary retained Fairy WndProc callbacks, generated
SDK bindings and a UI framework remain outside this slice. Project C/RC code
and transitive headers/assets are trusted native inputs, not hermetic locks.

Public configuration adds `fairyc.windows-build/v1`; the private companion adds
`--windows-root`. Existing Fairy FFI signatures and `main()` ABI are unchanged.
No DB, dependency, workflow/runner, repository-hygiene policy, release/version,
push, publication or deployment changes were made. Native output remains ignored.

Implementation commits: `95ca7b78` config, `4d0549f6` GUI entry/link,
`7e5d7fb5` resource compilation and CLI handoff, `b5bf6054` vector wrapper
reduction, `02c4c9fe` authored native driver, and `06be9a28` example/regression.
This documentation is a separate follow-up commit.
