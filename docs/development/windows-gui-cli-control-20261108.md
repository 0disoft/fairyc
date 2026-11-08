# Win32 controls and agent CLI, 2026-11-08

The [example README](../../examples/win32_minimal/README.md) owns this private
application protocol. [ADR 0552](../adr/0552-minimal-windows-gui-build.md) still
owns the compiler's native build inputs; its language/FFI boundary is unchanged.

`WM_NCCREATE` now saves the per-window state and delegates to `DefWindowProcW`,
which retains normal caption initialization. Smoke mode also checks the real
window title. This fixes the blank title observed during the earlier UI attempt.

The opt-in `--automation-stdio` mode accepts bounded commands over inherited
pipe handles. The ordinary GUI path opens no control channel. `control.py`
provides JSONL requests for an agent's existing command tool; it starts and
owns one visible GUI process, with timeouts and EOF/error cleanup. No model,
API credential, network listener or third-party Python package is required.

Input uses the real EDIT `SetWindowTextW` notification; clicking uses BUTTON
`BM_CLICK`. Both reach the ordinary WndProc through `WM_COMMAND`. Returned
text and event counters come from that running window's controls/state.

## Executed validation

- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure)$' -count=1 -v`: PASS, 7.532s, no skips.
  Development/release GUI smoke and Python controller verification both pass.
  A temporary copy with the button counter broken still fails smoke with code 5.
  Compiler C invocation retains `/W4 /WX`; Go test runs its normal vet checks.
- From the example, `go run -mod=readonly ../../cmd/fairyc build --release`
  and `python -B control.py --verify`: PASS.
- Reused the previously bootstrapped authored Fairy CLI/companion low host:
  `fairyc.exe run -- --smoke`, `fairyc.exe build --release`, then
  `python -B control.py --verify` against that fresh application: PASS.
  This refreshes native application inputs, not the compiler bootstrap.
- Each controller verification reads ten successful observations, tests empty
  input, Korean/Greek text, JSON quote/backslash escaping, replacement, 127-unit
  input, clear/apply and clean close. Five raw native requests reject: unknown
  command, overlong input, malformed UTF-8, embedded NUL and overlong line.
  Rejection leaves the last valid input intact.
- An actual persistent `python -u -B control.py` terminal session received
  `input` with `제로디 Win32 이벤트 확인`, then `click`. The visible window
  reported `Result: 제로디 Win32 이벤트 확인`, `EN_CHANGE` then `BN_CLICKED`,
  edits 1 and clicks 1. A JSON request with an unknown extra field rejected;
  subsequent `state` retained the same displayed result. Process metadata also
  reported title `Fairy C - Win32` and `Responding=True`.
- `gofmt` on the modified Go test and `git diff --check`: PASS.

Development image: 119,296 bytes. Release image: **117,760 bytes (115 KiB)**,
including the optional native protocol. Python is an external optional client,
not an application runtime dependency. Logs: `.cache/windows-gui-control-final-20261108.log`,
`windows-gui-control-verification-20261108.log`, `windows-gui-control-fairy-20261108.log`.
The source regression and controller verifier own checks; logs are local receipts.

## Boundaries

Computer Use was not needed for this run. This verifies programmatic native
control events and observable text, not physical keyboard/mouse delivery,
screen appearance, IME or DPI behavior. Those checks, full repository tests,
Linux, hosted CI and installer/distribution checks were skipped because this
batch changes a Windows example and its optional controller. The full authored
compiler bootstrap was reused rather than repeated. The Go GUI regression skips
only its optional controller subtests when Python is absent; Python was present
and neither subtest skipped here.

API change: private example control protocol `fairyc.win32-example-control/v1`
and its JSONL client. No compiler CLI, Fairy main/FFI ABI, DB, dependencies,
workflow/runner or repository-hygiene policy changed. No push or deployment.
Commits: `37a09fbc` title initialization, `89278a3a` native protocol/client;
regression wiring and this record follow in a separate commit.
