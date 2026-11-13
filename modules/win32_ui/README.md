# Win32 UI source bundle

`native/ui_core.c`/`.h` own shared Win32 window/child creation, bounded strict
UTF-8 text conversion, control text/read/enable/focus/move, client extents,
per-monitor DPI setup/suggested rectangles/application, icon replacement and
accessibility names/live properties/cleanup. Window class resource 101 supplies
the title and resource 1 supplies icons. Link user32/gdi32/ole32 and include the
source in fairy.windows.json; generated projects already do this.

`native_ui.fy` exposes control lookup/text/read/enable/focus/move/extent directly
to Fairy C. `window_metrics.fy` owns logical scaling, capped/centered content and
raw Enter/Tab/Shift+Tab/Esc action classification. Hash and the starter use the
same layout helpers; the starter delegates keyboard policy to the shared helper.
Actual command meaning, control IDs, layout, paint, dialogs, IME/drop specifics,
event queues, sticky Close and cancellation remain with each application.

The native window procedure supplied to window creation is native C transport,
not a foreign-thread Fairy callback. Call UI helpers on the owning window thread.
Handles returned by child lookup are borrowed until parent destruction. Text is
copied synchronously, never retained; invalid UTF-8/embedded NUL is rejected,
and text is bounded to 131,071 UTF-8 bytes. A read error maps to SIZE_MAX in C
and `.err(1)` in Fairy. Caller wrappers may impose smaller limits.

The caller owns window destruction/class unregister, icons, fonts/brushes and
accessibility service/COM lifetime. Clear name/live properties before releasing
the service and destroying children. No process-global input hook or automatic
high-contrast/DPI system change is installed. Calling system_dpi configures only
the app's existing per-monitor v2 process context; theme colors remain app policy.

This is a vendored source bundle, not a public GUI SDK or template updater.
`python -B tools/sync_win32_ui.py` refreshes Hash copies; `--check` detects drift.
Both generators install the same source files and tests. Refresh the embedded
ZIP with `python -B tools/bundle_win32_starter.py` after canonical changes.

`tools/verify_win32_ui.py` compiles production common C with MSVC /W4 /WX and
checks Unicode conversion, copied DPI suggestions and invalid-argument guards
without creating a window, acquiring foreground, setting focus or injecting
input. Pure Fairy tests cover layout/navigation. Build/CLI evidence does not
certify visible layout, real focus, screenreader speech or monitor transitions.

Both app pumps share the Close/pending/wait decision: nonblocking polls keep
draining messages after sticky Close while save commit/cleanup runs. Blocking
idle polls still return a pending event immediately. The headless probe checks
this decision without dispatching any desktop input.
