# Korean IME composition and Fairy shortcut policy, 2026-11-10

Owner: [example README](../../examples/win32_minimal/README.md),
`src/main.fy::on_key` / `on_tab`, `native/ime.c`, and
`internal/cli/windows_gui_ime_windows_amd64_test.go`.
This follows the [Tab focus checkpoint](windows-gui-focus-20261110.md).

## Reproduced behavior and policy

Actual Korean two-set virtual-key strokes produced `한글` while `글` was still
composing. Before the change, the same physical key ending composition also
executed the application's Enter/Apply, Esc/close, or Tab/focus action. The
before/after observations are retained in
`.cache/windows-gui-ime-conflicts-{before,after}-20261110.json`.

This example now reserves the first Enter/Esc/Tab for native composition. It
neither applies nor closes nor changes focus. A subsequent ordinary key runs
Fairy's existing action. This is the example's chosen shortcut policy, including
requiring a second Tab to leave an edit that was composing. The installed IME
owns whether Esc commits or cancels the uncommitted syllable; the verifier
accepts either outcome while requiring preservation of the committed `한`.

The native bridge subclasses only its edit control to observe
`WM_IME_STARTCOMPOSITION`, `WM_IME_COMPOSITION` and `WM_IME_ENDCOMPOSITION`.
It reads the actual composition string and conversion/open state using the
edit's IME context. `VK_PROCESSKEY` is resolved to its original virtual key
before translation. An end-composition boundary can precede the original
shortcut even when the current composition flag/string is already empty, so
that boundary is retained for the next native edit key.

For an IME-owned shortcut, the pump dispatches to the native edit/IME before
reporting event 6 with flag bit 32. Fairy's `on_key` and `on_tab` return no
application action for IME-owned flags. C handles native composition and
observation; Fairy still decides Apply, close and focus policy. No callback or
new public FFI export is introduced.

Focus loss, text replacement and edit clicks clear the boundary before and
after calling the original edit procedure. That procedure can end composition
reentrantly: clearing only before the call left a stale boundary that swallowed
the next ordinary Enter after returning to the edit. The fourth verification
case checks this regression.

The new `native/ime.c` / `ime.h` are explicit example build inputs;
`imm32.lib` links Windows' system IME API. No separate IME engine/runtime is
packaged. Application C continues to build with `/W4 /WX`.

## Actual-input CLI

```powershell
python -B control.py --verify-ime
```

The command requires a Korean two-set layout (`language:1042`, `0x0412`) on an
idle, unlocked interactive desktop. It sends `SendInput` virtual-key down/up
pairs, not Unicode packets or fabricated composition messages. It checks:

- `gksrmf` gives input `한글`, composition `글`, and native start/update events.
- Backspace revises the composition to `그` and input to `한그`; `f` rebuilds it.
- First Enter/Esc/Tab ends native composition with `keyFlags:32`,
  `keyAction:0`, input focus 201, zero Apply actions and no close.
- Second Enter applies exactly once; second Tab moves to Apply 202;
  subsequent Esc closes normally with exit 0.
- Leaving the edit while composing, applying from the button, returning and
  pressing Enter applies again with ordinary flags 0.

The four sequential owned sessions are closed on completion or failure. The
verifier toggles the owned Hangul mode through an actual Hangul key if needed,
and restores it before closing on success. Initial layout mismatch, missing
composition or wrong output is an error, never an automatic skip/pass.

Interactive `ime-strokes` supports a bounded list of 1..32 lowercase a..z or
named keys. `ime-focus` only selects the owned input; injection verifies actual
foreground and child focus. State polling requires the expected native-key
count and stable observations, rather than assuming that `SendInput` insertion
means the IME processed all keys. Existing held-input/foreground guards apply.

State adds an `ime` object with layout, conversion, open/composing state,
composition text, start/update/end/result counters, native-key count and last
original/raw key. `keyDuringComposition` describes the state observed at key
retrieval; boundary ownership can remain true after native composition ends.
Receipts report `verification:"actual-korean-ime"`,
`inputMethod:"physical-two-set-keys"`, `syntheticComposition:false` and four
observations. Here "physical" names virtual-key strokes as opposed to Unicode
packets; it is not a physical keyboard hardware certification.

Three negative controls must fail the verifier for the same application-key
collision: allowing an IME-owned key in Fairy, changing Fairy's IME-owned Tab
target, and removing native end-boundary ownership. These controls modify
isolated copies and do not replace the actual input path.

Implementation commits: `f9f3cda4` (observation/input CLI), `b28fe96e`
(IME priority), `835fdab3` (revision/boundary verification and regressions).

## Final verification

The reused authored Fairy CLI/companion built both profiles with `build --locked`
and `build --locked --release`. Each final image passed `--verify-ime` with all
four actual-input cases. Development/release images are **165,888 / 144,384
bytes**; receipts are
`.cache/windows-gui-ime-authored-{development,release}-final-20261110.json`.

The complete Windows GUI suite ran with actual desktop and authored-driver
integration enabled:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS=Join-Path (Get-Location).Path '.cache/windows-gui-ime-driver-20261110'
go test -mod=readonly ./internal/cli -run '^TestWindowsGUI' -count=1 -v -timeout 15m
```

**PASS: 332.604s, fifteen top-level tests passed, zero skipped.** This was one
combined run, not an aggregation of separate passing selections. It includes
development/release IME and three broken-priority controls alongside ordinary
GUI/control, event policy, desktop close, live border drag, DPI, Tab navigation,
keyboard policy, cursor perturbation, workspace/resources and authored driver.
Go-built development/release GUI images were 144,384 / 141,824 bytes. The fresh
authored CLI built 165,888 / 144,384-byte GUI images and passed `run -- --smoke`
in both profiles. The separate authored final IME receipts above exercise the
reused CLI, while this combined run rebuilds the authored driver independently.

Raw output: `.cache/windows-gui-ime-all-20261110.log`. The matching JSON receipt
records verified code revision `835fdab3`, source, compiler, companion,
runtime-pack, executable, controller-receipt and log SHA-256 values. Source
hashes cover the native IME module/header, config, Fairy source, controller,
desktop helper and GUI tests. `git diff --check` also passed. The following
documentation commit does not change those runtime/test inputs.

## Earlier attempts and limits

The first scoped run stopped on foreground loss before completing the input
cases (89.352s); those guards remain enforced. A later scoped run (49.683s)
completed actual input but failed Go receipt decoding because an observation
used `closed` for an object where the shared decoder expects a boolean. That
object was renamed `closeReply`; the same run's three policy controls passed.
Logs are `.cache/windows-gui-ime-scoped-20261110.log` and
`windows-gui-ime-priority-20261110.log`. Neither failed run is counted as green.
An earlier authored attempt also observed incomplete Hangul composition; its
rerun passed, and no cause is claimed. Final native-key-count/stable-state
checks and composition diagnostics do not turn an incomplete input into a pass.

These results cover the installed Korean two-set IME on one unlocked Windows
desktop at 125% scaling. They do not certify other IMEs/layouts, candidate/Hanja
UI, physical keyboard hardware, auto-repeat timing, locked/headless/elevated
desktops or physical monitor/DPI transitions. Concurrent user input can still
invalidate foreground ownership. Linux, unrelated repository-wide tests,
hosted CI and a full compiler bootstrap were omitted for this Windows example
change; the authored driver was rebuilt as described above.

Only the private example JSONL/flag surface expands. Public compiler, language
and SDK API, DB, package dependencies, hygiene policy, runners/workflows and
versions are unchanged. The one native link-input addition is Windows system
`imm32.lib`. No push/deploy is included.
