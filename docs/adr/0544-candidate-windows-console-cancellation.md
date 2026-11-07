# ADR 0544: Authored Windows state command console cancellation

- Status: Accepted for the explicit private Windows state developer command
- Date: 2026-11-07

The actual `tools/dev/fairyc-state/main.fy` process entry now creates an unnamed,
manual-reset, initially clear, non-inheritable cancellation event. After argv
admission, it registers this token before image/source/asset/storage access.
Standalone help and invalid argv retain their asset-free behavior.

Three closed private host operations create the local event, bind a console and
consume a registration. Go/Fairy selectors and the authoritative runtime pack
must agree. Existing named tokens, none tokens and caller APIs remain available.
The sealed Fairy Console owns a native registration and a mutable scalar bound
flag; explicit close disarms automatic cleanup before consuming the registration,
including on failure. This follows the admitted resource-field subset without
adding pointer-field mutation support to the language.

At most one registration is active per process. The native owner duplicates the
event with non-inheritable synchronization/signal rights, so the original Token
can close first. The callback handles only Ctrl-C and Ctrl-Break and sets the
event while holding a shared lock. Close exclusively unpublishes the owner and
drains active uses before releasing its duplicate/allocation. No callback retains
a registration pointer after releasing the lock. Unregister runs outside that
lock. Registration remains busy during close; failed unregister poisons future
binds rather than accumulating handlers or retaining freed state. Original OS
causes survive cleanup; unregister failure without an OS cause becomes
ERROR_INVALID_STATE rather than success.

Missing console error 6 produces an explicit inactive Console, preserving
headless operation. Other bind failures remain failures. The product never
changes inherited Ctrl-C ignore behavior. Console attachment changes while bound
are unsupported: AttachConsole/AllocConsole/FreeConsole reset Windows' handler
table. Close/logoff/shutdown retain OS behavior. These constraints follow
[SetConsoleCtrlHandler](https://learn.microsoft.com/en-us/windows/console/setconsolectrlhandler).

The actual command passes the token through existing bounded build/run/file
paths. Existing CAS rules remain: once publication begins, no cancellation poll
can reinterpret a known acknowledgment or trigger a retry. Registration cleanup
failure reports exit 4 and warns that state may already be published; it does
not undo the operation. Early-return cleanup still releases the registration.

Control tests generate signals only within a hidden new console containing the
child alone. Product execution never enables ordinary Ctrl-C processing on the
user's behalf. Test-only ordinary-processing setup respects
[GenerateConsoleCtrlEvent](https://learn.microsoft.com/en-us/windows/console/generateconsolectrlevent).
No signal is sent to the runner's or user's console.

This changes private Windows adapters and private developer orchestration, not
public CLI/SDK/native ABI, database/provider, dependencies, repository policy or
CI/runners. Native Linux remains deferred. The full implementation queue and
final acceptance remain open.

Evidence: `docs/development/candidate-windows-console-cancellation-20261107.md`.
