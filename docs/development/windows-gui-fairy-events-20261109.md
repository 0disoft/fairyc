# Fairy-owned Win32 click handling, 2026-11-09

Owner: [example README](../../examples/win32_minimal/README.md) and the private
FFI in `src/main.fy` / `native/window.c`. [ADR 0552](../adr/0552-minimal-windows-gui-build.md)
continues to own native application build inputs.

## Behavior and lifetime

Fairy now owns the application loop, click count, empty-input decision and result
string construction. Its `on_click` reads bounded UTF-8 and produces the text.
C owns Win32 window/control operations, message pumping, character conversion,
native edit telemetry and the existing JSONL transport.

The private functions are `fairy_win32_open`, `fairy_win32_next`,
`fairy_win32_read_input`, `fairy_win32_present` and `fairy_win32_close`.
`next` returns a click to Fairy; `present` renders Fairy's result and mirrored
click count, then acknowledges a pending CLI request. A CLI reply therefore
observes the rendered result, rather than racing the handler. C retains no
Fairy function or input/output buffer. The opaque window lives until close.
The example remains single-window and single-threaded.

An initial synchronous callback implementation passed the Go frontend but was
rejected by the authored Fairy CLI's function-parameter subset. It was replaced
by the event pump before committing. No callback grammar or export support was
added. The first authored pump build also exposed generated C warnings treated
as errors. Both native builders now keep `/W4 /WX` for project C bridges and
retain their existing partition-consumer flags for generated C. No emitter
diagnostic, semantic check or compiler work budget was weakened or raised.

## Executed verification

- `go test -mod=readonly ./internal/cli -run
  '^TestWindowsGUI(Example|EventFailure|Resources|WorkspaceBuild)$' -count=1 -v`:
  PASS, 29.895s. Both target profiles pass GUI smoke and JSONL verification;
  resource loading, subsystem inspection and failed-build preservation pass.
- The negative test now changes **only Fairy source**, replacing `b"Result: "`
  with `b"Broken: "`. The resulting native application's event smoke exits 5.
  This proves the result depends on Fairy policy, not a C fallback.
- Targeted `TestWindowsGUIWorkspaceBuild` rerun with an unused-variable warning
  in an otherwise valid project C bridge: PASS, 18.544s. The warning still
  rejects the build and preserves the prior executable under `/W4 /WX`.
- With `FAIRYC_RUN_PARITY_INTEGRATION=1`,
  `go test -mod=readonly ./internal/cli -run '^TestWindowsGUIAuthoredDriver$'
  -count=1 -v`: PASS, 255.273s. Fresh authored companion and full low CLI host
  (210 C parts) pass development/release `build` and `run -- --smoke`.
  Raw packet/subsystem/resource-preservation fixtures are independent of the
  example's changing application ABI; public CLI checks use the real Fairy app.
- `go run -mod=readonly ../../cmd/fairyc check` from `examples/lexer_candidate`:
  PASS, 777 source files / 9,482 MIR functions; lock refreshed.
- `python -B control.py --verify` passes the Go-built application. The freshly
  authored CLI also builds the example with `build --locked` and
  `build --locked --release`; each resulting image passes the same JSONL
  verifier (ten observations, five invalid native commands rejected, clean exit).
  Korean/Greek text, quotes, backslash, empty input, repeated clicks, maximum
  length, clear/apply and unchanged state after rejection are included.
- A persistent JSONL session then sent `input` with `Fairy C에서 처리한 클릭`
  and `click` to the authored release app. The visible window reported
  `Result: Fairy C에서 처리한 클릭` and `WM_COMMAND: BN_CLICKED`.
- Formatting and scoped `git diff --check`: PASS.

| Builder | Development app | Release app |
|---|---:|---:|
| Go CLI | 122,880 bytes | 119,808 bytes |
| Authored Fairy CLI/companion, low host | 138,240 bytes | 121,856 bytes |

Local logs: `.cache/windows-gui-fairy-events-go-20261109.log`,
`windows-gui-fairy-events-authored-20261109.log`,
`windows-gui-fairy-driver-check-20261109.log` and
`windows-gui-fairy-handler-{development,release}-20261109.json`.
Sources own behavior; ignored logs/images are local receipts.

## Limits

JSONL operations exercise real EDIT/BUTTON notifications and observed control
text without Computer Use. Physical input delivery, IME, DPI, appearance review,
Linux, remote CI, full repository tests, optimized compiler-host bootstrap,
complete self-host/fixed point and installers were not run for this Windows
example change. Generated C can still produce warnings under its existing
consumer policy; this is not a warning-free emitter claim.

Only the private example C bridge API changed; JSONL v1 and the compiler's
public CLI/language/FFI contracts did not. No DB, dependencies, workflows/runners,
repository-hygiene policy, version bump, push or deployment changes.
Implementation commits: `2c574172` native warning scope, `153993c5` Fairy event
handling and source-backed regression. Documentation and strict bridge warning
regression follow separately.
