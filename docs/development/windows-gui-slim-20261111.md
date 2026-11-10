# Distribution build size, 2026-11-11

Owner: the [Win32 example](../../examples/win32_minimal/README.md) and its private
native bridge. Native build inputs continue to follow
[ADR 0552](../adr/0552-minimal-windows-gui-build.md).

## Baseline before separation

The authored release image verified at code revision `b15fe506` is 144,384
bytes (141 KiB), SHA-256
`ed05eee2b0e38cf67dbe794961ce57f344f85b038e13ed527453e25fe2d6c945`.
`python -B size_report.py <image>` reads PE sections and named DLL imports using
only Python's standard library. It does not execute the image.

| File component | Bytes |
|---|---:|
| Headers | 1,024 |
| `.text` | 86,016 |
| `.rdata` | 45,056 |
| `.data` | 3,072 |
| `.pdata` | 5,632 |
| `.fptable` | 512 |
| `.rsrc` | 1,024 |
| `.reloc` | 2,048 |
| **Total** | **144,384** |

There are no unassigned/overlay bytes. Raw section sizes include file-alignment
padding; virtual sizes describe mapped sections, not disk size or process RSS.
The image imports USER32 (50 names), GDI32 (4), IMM32 (6) and KERNEL32 (80).
Those DLLs are Windows dependencies, not bundled files. Static CRT code/data
is included in the image's sections; no CRT DLL is imported. The executable
has no COFF symbol table, so this measurement does not invent exact per-function
or static-CRT byte attribution from section totals.

The baseline includes automation JSONL/pipe handling, smoke scenarios, injected
DPI requests and mouse/IME observation. These are runtime opt-in features in
the existing image, not compile-time exclusions. The distribution work will
exclude these diagnostics while retaining the same Fairy application source,
native event processing, CRT initialization, UTF conversion, IME priority,
resize/DPI layout and resource text. The ordinary build must retain its CLI
verification features. CRT replacement, compression and disabling safety checks
are outside this change.

Local baseline files are retained under `.cache/windows-gui-slim-20261111/`;
`baseline-authored-release.json` records the section/import inventory and image
hash, and `baseline-authored-release.exe` preserves the measured image.

## Separation and measured result

The ordinary config/source selection is unchanged. `native/profile.h` defaults
diagnostics to 1; the two distribution wrappers explicitly select 0 and include
the shared native window/IME implementations. `native/diagnostics.h` holds the
private smoke/JSONL helpers and is included only in the ordinary translation
unit. `window.c` shrank from 39,652 to 23,294 source bytes; the diagnostic include
is 17,935 bytes. Source sizes are maintenance information, not EXE savings.

`build_distribution.py` uses an isolated temporary copy of the same Fairy
source, native files, manifest and lock, substitutes only the two native source
entries, and invokes the selected Go or authored CLI with `build --locked`.
Release is the default. Only a successfully built/inspected image is atomically
published as `build/win32-minimal-distribution.exe`; the original config and
ordinary image are not overwritten. No public build schema/flags are added.

| Compiler | Profile | Ordinary bytes | Distribution bytes | Saved |
|---|---|---:|---:|---:|
| Go CLI | development | 144,384 | 129,536 | 14,848 |
| Go CLI | release | 141,824 | 121,344 | 20,480 |
| Reused authored CLI | development | 165,888 | 150,016 | 15,872 |
| Reused authored CLI | release | 144,384 | 123,904 | 20,480 |

The authored release shrank **141 KiB to 121 KiB (14.18%)**. The measured
ordinary authored release remains byte-identical to the baseline above, so
the CLI/smoke build has not silently become the distribution build.

| Authored release section | Before | Distribution | Reduction |
|---|---:|---:|---:|
| `.text` | 86,016 | 68,608 | 17,408 |
| `.rdata` | 45,056 | 42,496 | 2,560 |
| `.pdata` | 5,632 | 5,120 | 512 |
| All other sections/headers | 7,680 | 7,680 | 0 |
| **Total** | **144,384** | **123,904** | **20,480** |

Distribution still uses the same four Windows DLLs, with USER32/GDI32/IMM32/
KERNEL32 import counts reduced to 37/3/4/77. Private pipe polling, timers,
diagnostic arguments, injected-DPI requests, mouse probe and IME reporting are
compiled out. `ReadFile`, `PeekNamedPipe`, `SetTimer`, `KillTimer`,
`GetMessageExtraInfo`, `ImmGetOpenStatus` and `ImmGetConversionStatus` are absent.
`WriteFile` remains required by the static CRT/runtime; treating every such
shared import as evidence of automation would be incorrect. No CRT/safety,
IME priority, application handler or resource was removed to obtain the saving.

## External verification of the distribution image

`verify_distribution.py` launches an owned image with pipe handles but reads
native text, focus and geometry externally; it never relies on JSONL or the
private mouse probe. Actual `SendInput` delivers Enter/Tab/Shift+Tab, Hangul
two-set strokes and Esc. The setup button case uses native `WM_SETTEXT` and
`BM_CLICK`, while resizing uses `SetWindowPos`. The verifier checks Unicode
Apply, Enter status, both focus directions, actual `한글` composition with the
first Enter reserved for the IME, the second Enter applying, Fairy's resized
layout and Esc exit 0. It requires no pipe output and no private probe/argument
markers or automation-specific imports. It uses physical PMv2 coordinates.

`TestWindowsGUIDistribution` compares ordinary and slim images under the same
Go compiler/profile, requires a real size reduction, and runs this external
verification three times per profile: normal launch, `--smoke`, and
`--automation-stdio`. Both diagnostic arguments must remain inert while all
ordinary controls keep working. The scoped selection passed in **33.266s**,
including all six launches. Raw log:
`.cache/windows-gui-slim-20261111/distribution-go-scoped.log`.

Earlier external-verifier attempts are not passes: the first absence check
incorrectly classified CRT `WriteFile` as diagnostic; the focus check assumed
`BM_CLICK` preserved edit focus although the native button can take focus;
geometry was initially read in DPI-virtualized coordinates; and post-Esc
instrumentation queried a GUI thread that had already exited. Those verifier
assumptions were corrected. No application focus/layout fix was needed for
these observations. The final external receipt is
`distribution-control-final.json` in the same cache directory.

Implementation: `9ac869ae`; distribution regression: `4b4cf66b`.

## Combined verification and final artifacts

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_WINDOWS_GUI_DRIVER_ARTIFACTS=Join-Path (Get-Location).Path '.cache/windows-gui-slim-20261111/driver'
go test -mod=readonly ./internal/cli -run '^TestWindowsGUI' -count=1 -v -timeout 15m
```

**PASS: 616.827s, sixteen top-level tests passed, zero skipped.** This is one
complete run, not aggregated scoped passes. It includes both distribution
profiles and all six launch variants alongside ordinary GUI/control, event
failures, fresh authored driver, desktop close, live drag, DPI, focus and IME
policies, keyboard, mouse perturbation, workspace and resources. All ordinary
development/release smoke runs still pass. The fresh authored driver rebuild
also passes its public CLI build/run smoke checks with the split native files.

The reused authored CLI's final development/release images separately pass
ordinary `control.py --verify`, actual Korean `--verify-ime` (four cases per
profile), and external `verify_distribution.py` on the distribution image.
Receipts are `authored-{development,release}-{control,ime,distribution}-final.json`
under `.cache/windows-gui-slim-20261111/`. `summary.json` records revision
`4b4cf66b`, source/image/compiler/companion/runtime-pack/controller-receipt/log
hashes and the full test set. `all.log` preserves the full combined output.
The ordinary authored release hash matches the original baseline and the
default distribution image matches the retained final authored release build.

`git diff --check` and Go formatting checks passed. The final documentation
commit does not change those verification inputs. The working example retains
both `build/win32-minimal.exe` (diagnostic) and
`build/win32-minimal-distribution.exe` (distribution); build artifacts are ignored.

Only the Windows example's build/verification helpers and private native layout
change. Public compiler/language/SDK API, DB, package dependencies, repository
hygiene policy, CI runners/workflows and versions are unchanged. No push or
deployment is included. Python is needed for these optional local helpers, not
to run either GUI image.

The actual-input results require an idle, unlocked desktop with Korean two-set
input. Other IMEs/candidate windows, physical keyboard hardware, physical
cross-monitor DPI transitions and locked/headless/elevated sessions are not
certified. Linux, unrelated repository-wide tests, hosted CI and a complete
compiler bootstrap were omitted for this Windows example task. Static CRT
function-level byte attribution and process RSS were not measured; the section
totals and before/after builds above are the supported size evidence.
