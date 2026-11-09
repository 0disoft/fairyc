# Win32 mouse targeting and combined verification, 2026-11-09

Owner: [example README](../../examples/win32_minimal/README.md),
`desktop_input.py`, private `native/window.c` observations and
`internal/cli/windows_gui_mouse_target_windows_amd64_test.go`.
This follows the [Fairy event implementation history](windows-gui-fairy-events-20261109.md).

## Input changes

The previous helper moved the cursor, observed hover, then submitted a
positionless button-down. Cursor changes between those operations could send
the press to a different native region. The earlier failure recorded hit 11
(`HTRIGHT`) instead of 20 (`HTCLOSE`); it had no native point/token observation
and its exact cause cannot be reconstructed retroactively.

The helper now includes the normalized absolute desktop target and button-down
in one `MOUSEINPUT`; title-bar release includes target plus button-up. Before
pressing it rechecks owned foreground, target occlusion and `WM_NCHITTEST` at
the intended point. It then requires the delivered native hit, point and
nonzero input token to match. C records signed screen coordinates from the
native message, `GetMessageExtraInfo` and the window rectangle at delivery.
The read-only probe cannot generate an application event or change policy.
Receipts expose `targetPoint`, `pressPoint`, `inputToken`, `pressToken` and
`pressWindow`. The token is correlation metadata, not authentication.

Moves set `MOUSEEVENTF_MOVE_NOCOALESCE` so fresh delivery observation does not
depend on coalesced motion. Foreground acquisition now includes its 100-ms
stable interval in the two-second retry budget; a transient activation loss
before input can retry, while foreground loss during input still aborts.
The helper releases its own held button on failure. Fairy's close policy,
normal close reply shape, keyboard policy and real four-point drag checks stay
in place. [MOUSEINPUT](https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-mouseinput)
documents the combined motion/button flags and extra information;
[WM_NCLBUTTONDOWN](https://learn.microsoft.com/en-us/windows/win32/inputdev/wm-nclbuttondown)
provides the observed hit and screen point.

## Regression method

`internal/cli/testdata/windows_gui_mouse_target.py` wraps only the controller's
`SendInput` call. Immediately before forwarding button-down and button-up it
submits real absolute cursor motion to the same owned window's right border.
It verifies this perturbation point is actually `HTRIGHT`. It neither forges
Win32 button/close messages nor changes Fairy policy.

The release example is built once for three independent owned sessions:

- The current helper survives both cursor perturbations and records a native
  `HTCLOSE` press with matching token/coordinates, `SC_CLOSE`, Fairy approval
  and process exit 0.
- A temporary controller copy without press positioning must fail on the
  actual wrong native hit.
- A temporary copy without release positioning must fail on the absent close
  reply after the cursor moves away. Neither broken copy changes production
  sources or skips the native observation.

The focused regression passed in **10.864s**, including both negative controls.
Receipt: `.cache/windows-gui-mouse-target-go-20261109.log`.
Implementation: `8e85b68f`; regression/acquisition follow-up: `1e188280`.

## Desktop interference

Before the successful cursor regression, foreground inspection identified
`KrasisSourceWheelCapture-20996-1` from a running `krasis-input.exe` helper.
Both test processes used the Default desktop; owned-thread queue attachment
succeeded but `SetForegroundWindow` still failed. This establishes an observed
foreground conflict, not the exact cause of every older mouse failure.

The user explicitly authorized stopping that helper. Ordinary termination was
access-denied; a constrained administrator helper checked PID, executable name
and parent before stopping only the approved PID 20996. The stop receipt is
`.cache/krasis-capture-stop-20261109.json`. No Krasis source, preference or
service configuration was changed. Results here require an unlocked, idle
desktop without a competing input owner; they do not certify simultaneous
Krasis input capture or physical mouse/keyboard hardware.

## Combined invocation

Run from the repository root with MSVC, Windows SDK and Python available:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS=Join-Path (Get-Location).Path '.cache/windows-gui-full-driver-20261109'
go test -mod=readonly ./internal/cli -run '^TestWindowsGUI' -count=1 -v -timeout 15m
```

Unlike the earlier seven-test selection, this selects all eleven
`TestWindowsGUI*` tests: example controls, Fairy mutation controls, authored
driver, desktop close, live drag, DPI, keyboard, keyboard policy, mouse target,
workspace build and resources. The parity flag enables fresh authored driver
construction rather than skipping it. The native companion and runtime pack
are retained in the artifact directory. Full raw output is saved to
`.cache/windows-gui-all-20261109.log`.

**Result: PASS, 299.671s, eleven top-level tests passed, zero skipped.**
This is one combined invocation, rather than passing cases assembled from
separate failed runs. Development/release desktop close, live held-button
layout, DPI, keyboard, Fairy denial/mutation controls, the deliberate cursor
race and both untargeted negative controls passed together. Fresh authored
driver/public CLI development and release builds, `run -- --smoke`, packet
handling, console fallback, resource failure preservation, workspace build and
resources also passed. Go-built example images were 141,312 / 136,192 bytes;
the fresh authored CLI produced 161,280 / 138,240 bytes.

`.cache/windows-gui-all-20261109.json` records the implementation revision,
source and retained driver/pack/log hashes, selected test list, timing and zero
skips. The code revision is `1e188280`; the subsequent documentation commit does
not change verification inputs. `gofmt` and `git diff --check` passed.

Earlier exploratory attempts are not counted as passes: protected foreground
activation failed before the approved external-helper stop; initial cursor
fault-injection attempts exposed transient activation loss and the distinction
between `SetCursorPos` and queued input. The final regression uses real
non-coalesced `SendInput` motion for both perturbations and requires the
specific native-hit/close-timeout failures in the broken copies.

Public compiler/language/SDK API, DB, dependencies, repository-hygiene policy,
workflows/runners and versions are unchanged. Build products and receipts remain
ignored; no push or deployment is part of this work. Physical monitor/DPI
transitions, hardware/IME certification, Linux, unrelated repository tests and
hosted CI are outside this Windows input regression scope.
