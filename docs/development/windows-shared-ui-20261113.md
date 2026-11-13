# Shared Win32 UI source bundle — 2026-11-13

`modules/win32_ui` now owns native window/child creation, text conversion and
control primitives, focus, DPI awareness/suggestion/application, icon lifetime
and accessibility names/live properties/cleanup. `native_ui.fy` exposes reusable
control helpers directly to Fairy C; `window_metrics.fy` provides DPI scaling,
capped/centered content and raw navigation classification. Hash and the starter
use the same native implementation and Fairy layout helpers. Starter navigation
also delegates to the common Fairy helper.

Product window procedures/queues, control IDs, layout, drawing, dialogs, IME,
clipboard/drop and command/cancellation meanings remain with each application.
The UI module retains no Fairy callback or worker borrow. HWND lifetime and UI
thread ownership remain caller responsibilities. This is a source bundle, not
an SDK promotion or updater. Both generators install independent copies; Hash
snapshots and compiler-embedded inputs have drift guards. `sync_win32_ui.py`
uses the existing install/check mechanism without adding a package dependency.

Native MSVC /W4 /WX production-code probe: 27 checks passed for Unicode roundtrip,
empty/invalid/NUL/oversized text, copied/invalid DPI rectangles and argument guards.
The shared pump decision also covers nonblocking draining after sticky Close,
correcting the starter cleanup-wait case; Hash retains its prior behavior.
No CreateWindow, SetFocus, process DPI setup or input injection occurs in the
successful probe path. Pure Fairy layout/navigation tests passed. Both native
apps compiled; SDK macro naming and duplicate UIAutomation constant issues found
during extraction were corrected before the final successful links.

Final validations passed:
- Generated app: four pure tests; release build; 26 real CLI cases (9 async read,
  12 safe save, 5 original). New package CLI passed outside the repository using
  only a copied compiler, empty-PATH creation and no manual resource preparation.
- Python generator: independent build/package/resources, same 26 CLI cases,
  six rejected creations and existing-project preservation.
- Hash: check, three affected pure tests, locked release and 57 extracted-package
  CLI cases (19 hash/report + 16 comparison + 22 checksum).
- Scoped scaffold/CLI tests, Go vet, Python syntax, both source snapshot checks
  and git diff whitespace check. Final indentation-only cleanup does not change
  runtime behavior; canonical bundle receipts were refreshed afterward.

| Release EXE | Before this three-step task | After |
| --- | ---: | ---: |
| Fairy Hash | 213,504 bytes (208.5 KiB) | 214,016 bytes (209 KiB) |
| Starter | 157,696 bytes (154 KiB) | 172,032 bytes (168 KiB) |

The complete safe-save demonstration and shared UI additions cost 14 KiB for the
starter and 512 bytes for Hash. Compared with step 2, shared UI alone adds 512
bytes to the starter and 1 KiB to Hash. CLI-generated probe ZIP is 93,584 bytes;
Python-generated branded probe ZIP is 93,608 bytes; Hash ZIP is 114,191 bytes.
These are measured artifacts, not isolated per-helper size guarantees.

Local compiler: `.cache/windows-jobs-20261113/fairyc.exe`.
Receipts there: `ui-core.json`, `package-ui.json`, `starter-ui.json`, `hash-ui.json`.
Native GUI startup/layout/focus/cancel/close, IME, Narrator and physical monitor
transitions were intentionally not run. C/UI link checks and headless probes do
not certify those behaviors. No broad bootstrap or hosted CI was run.

Ownership docs: `modules/win32_ui/README.md`, CLI command contract and package/CLI
source index; template/Hash/build-tool guides were updated. There is no new CLI
surface in this refactor (step 2 owns package via ADR 0554), language/public SDK,
DB, dependency, runner or version change. Generated artifacts remain ignored.
Local commits only; no push, signing, upload, release or deployment.

Earlier steps: `8a96e5cd` shared safe publication, `163b1702` build/package CLI.
