# Fairy Hash keyboard, accessibility and display checks, 2026-11-12

Scope: the Hash utility, separate from the minimal Win32 example. Production
application policy remains in Fairy; C performs native Windows operations.

## Environment and first actual input

One monitor is connected: 1920×1080 at 125%. A physical transition between
different monitor scales cannot be certified on this inventory. No virtual
monitor/driver or persistent display scaling was installed/changed.

The retained authored release from the progress work was tested through actual
SendInput: Tab focused native path control 201, Shift+Tab returned to Open 202.
The new `native_keyboard.py` validates process ownership, requires stable
foreground, refuses held user keys/buttons and releases only its own partially
injected keys after an incomplete send. It observes focus with GetGUIThreadInfo;
it does not inject WM_KEYDOWN or use a private application diagnostic protocol.
Receipts: `.cache/windows-hash-usability-20261112/baseline-keys.json`.

Earlier foreground wheel-capture activity was observed, but this actual-key
probe acquired foreground without terminating any external process.
