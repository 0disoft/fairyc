# Fairy-owned Tab focus navigation, 2026-11-10

Owner: [example README](../../examples/win32_minimal/README.md),
`src/main.fy::on_tab`, the private Win32 bridge, and
`internal/cli/windows_gui_focus_windows_amd64_test.go`.
This follows the [mouse/combined verification checkpoint](windows-gui-input-stability-20261109.md).

## Policy and native boundary

The pump captures Tab `WM_KEYDOWN` before `IsDialogMessageW`, using existing
event 6. `fairy_win32_key` axis 2 reads the native source control ID, while axes
0/1 retain code/flags. Fairy's `on_tab` chooses the next target:

- Plain Tab: input 201 to Apply 202, otherwise input 201.
- Shift+Tab: Apply 202 to input 201, otherwise Apply 202.
- Other modifiers/repeated key-down flags: no action.

These two tab stops form a cycle; with two controls, forward and reverse visit
the same pair. Separate policy branches remain in Fairy. C neither chooses an
order nor falls back to native Tab navigation. `fairy_win32_focus` validates a
visible, enabled native tab-stop child, calls `SetFocus` for Fairy's exact ID,
and checks the result. Key action 4 acknowledges focus changes only after that
operation. Enter/Esc retain their existing Fairy handlers.

State adds `focus`, a live `GetFocus`/control-ID observation. `keyFocus` still
records where the key arrived, so the before/after values are independent.
The legacy mouse-close-denial test compares every application-data field while
excluding only live `focus`, because native caption activation may change it.
Dedicated focus tests assert actual destination and re-read live state after
the key reply; there is no weakened focus assertion in those tests.

## CLI verification

```powershell
python -B control.py --verify-focus
```

The controller uses actual `SendInput` Tab/Shift/Control key pairs. It verifies
forward/backward wraparound, ignored Ctrl+Tab, preserved Unicode/text/status
and edit/Apply counts, no delayed native navigation, then Enter on Apply and
Esc exit 0. Eight native key observations must be acknowledged by Fairy.
Changing only Fairy's forward target or Shift branch makes this verifier fail
on focus policy. No focus/keyboard message is fabricated to perform navigation.

Interactive examples:

```json
{"command":"key","key":"tab"}
{"command":"key","key":"shift-tab"}
{"command":"key","key":"ctrl-tab"}
{"command":"key","key":"enter","focus":"keep"}
```

Tab variants default to `focus:"keep"`, which sends the native arm command with
focus sentinel 0; it does not reset to input between steps. Explicit
`focus:"input"` and `"apply"` still work. `arm-key` accepts codes 9/13/27 and
focus 0/201/202. It only arms observation and optionally sets the starting
focus; the actual Tab is delivered separately and Fairy selects its target.
Held-input guards now include Tab. Requests still require owned foreground
and matching native key/count/source-control observation.

The authenticated authored CLI/companion passed `check` (one source, nine MIR
functions) and locked development/release builds. Both images passed actual
`--verify-focus`. Receipts:
`.cache/windows-gui-focus-authored-{development,release}-20261110.json`.
The first scoped Go run passed both focus profiles, both Fairy policy mutation
controls and existing keyboard tests, but failed the legacy mouse-denial exact
state comparison after the new live focus field was added. That result is not
counted as a passing run; the comparison was updated as described above.

Implementation: `5c5cd21a`; CLI/regression: `ebb30c41`.

## Combined verification

The complete Windows GUI selection includes the two new focus tests:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS=Join-Path (Get-Location).Path '.cache/windows-gui-full-driver-20261110'
go test -mod=readonly ./internal/cli -run '^TestWindowsGUI' -count=1 -v -timeout 15m
```

Raw output: `.cache/windows-gui-focus-all-20261110.log`.
The companion/runtime pack are retained in the selected artifact directory.

**PASS: 231.224s, thirteen top-level tests passed, zero skipped.** Both focus
profiles and both Fairy-only direction controls passed alongside ordinary
GUI/resize, mouse-close denial, desktop close, live drag, DPI, keyboard policy,
cursor perturbation, workspace/resources and fresh authored driver verification.
Go-built development/release examples: 142,336 / 136,704 bytes. The fresh
authored CLI built 162,816 / 138,752-byte examples and passed `run -- --smoke`
in both profiles. This is a single combined run, not aggregated scoped passes.

`.cache/windows-gui-focus-all-20261110.json` records code revision `ebb30c41`,
verified source/driver/runtime-pack/log hashes, timing and the complete test set.
`gofmt` and `git diff --check` also passed. The following documentation commit
does not change those verification inputs.

Only private example FFI/JSONL surfaces expand. Public compiler/language/SDK API,
DB, dependencies, repository-hygiene policy, CI runners/workflows and versions
are unchanged. No push/deploy is included. The desktop must be unlocked and
idle; IME, held-key hardware repeat, multiple/disabled dynamic tab stops,
physical cross-monitor DPI, Linux and unrelated repository tests are outside
this two-control Windows example scope.
