# Native file SHA-256 utility, 2026-11-11

Owner: [Fairy Hash README](../../examples/win32_hash/README.md) and
`examples/win32_hash/src/{main,application,hash_file,sha256_stream,sha256,ui}.fy`.
This is a separate executable from `win32_minimal`; the existing example and
its [slim-build checkpoint](windows-gui-slim-20261111.md) are unchanged.

## Actual application flow

Launch `build/win32-hash.exe` without arguments. Open chooses a file using
Windows' real `GetOpenFileNameW` dialog. Fairy opens/reads the file through
`std::fs`, updates SHA-256 using one reusable 64 KiB buffer, renders its lowercase
64-digit digest and byte count, and builds a UTF-8 JSON report. Save invokes
the real `GetSaveFileNameW` dialog; Fairy writes the report through `std::fs`.
Cancelling either dialog retains an existing result; a failed new hash clears
the digest and disables Save rather than presenting an old digest as current.

The C bridge owns HWNDs, control access, Win32 events, common dialogs, font/DPI
operations and UI UTF conversion. It does not read input contents, compute a
hash, construct JSON or write a report. Fairy owns selection/hash/save flow,
errors, enabled state, DPI-scaled layout and the event loop. No retained Fairy
callback, public export, UI framework, embedded model/API key or private
automation protocol is introduced. Build inputs use the existing Windows
config schema and CRT entry path from ADR 0552.

The SHA implementation is copied from the compiler example's private
`sha256` / `sha256_stream` modules. The source-transport-specific 96 MiB bound
is replaced with a checked SHA-256 byte-length bound (`2^61-1`). Modular u32
addition still uses checked 16-bit limbs. This is an integrity utility, not a
claim about signing, authenticity or password storage.

Message polling between chunks permits ordinary paint/resize/close processing
on the same UI thread. Byte-status updates are bounded to roughly 1 MiB steps.
Open/Save are disabled while hashing; WM_CLOSE remains pending until Fairy
cancels the computation and resolves shutdown. This is cooperative processing,
not a worker thread; it cannot interrupt an OS read that has not returned, or
provide a snapshot of a concurrently modified file. Native modal drag handling
is not certified for the minimal example's fiber-based live-layout behavior.

## Report and source protection

`--hash <input> [report-path]` performs the same Fairy computation/save without
creating the GUI. Its inherited stdout is a UTF-8 JSON report:

```json
{"algorithm":"SHA-256","file":"C:\\data\\input.bin","bytes":3,"sha256":"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}
```

The example above describes an `abc` input. Failure is exit 1 with
`{"ok":false,"errorCode":N}`; success is exit 0. The GUI subsystem does not
allocate a console, so use a capturing command caller or PowerShell pipeline.

Fairy rejects the source path and existing destination handles identifying the
hashed source (including hard links). Reports use exclusively created sibling
temporary files; collisions move to another bounded candidate without touching
the preexisting name. Fairy writes all bytes, syncs and consumes/closes that
handle before calling `fs::replace_file`. Failure attempts to remove only the
temporary it created. No destination deletion/truncation precedes publication.
These are ordinary local source/publication protections, not hostile filesystem
race resistance or power-loss durability certification.

## Executed verification

The reused authored CLI/companion built the actual example with
`build --locked` and `build --locked --release`. Both images passed:

- `verify_cli.py`: **19 cases** against Python `hashlib.sha256`, covering empty,
  `abc`, padding boundaries, binary input beyond 1 MiB, Unicode names, new and
  existing report destinations, missing input, failed destination, source and
  hard-link protection, exclusive temporary-name collision and cleanup.
- `verify_gui.py`: actual native Open/Save dialogs with empty, `abc` and
  1,048,832-byte binary inputs; native path/digest reads and saved JSON matched
  the independent reference. Paths exceeded 127 characters. Cancelling Open or
  Save preserved the result; overwriting an owned existing report succeeded;
  selecting the input as output was refused without data loss. Closing while
  hashing a 64 MiB file exited cleanly and fixture cleanup completed.

These GUI actions use native `WM_SETTEXT` and button `BM_CLICK`/posted button
clicks from an external process. They are not physical keyboard/mouse tests.
The actual modal dialogs run in the owned app; no selected path/hash/result is
fabricated through a private application command. All fixtures and overwrite
confirmations are limited to the verifier's own temporary directory.

The Go frontend built and verified both profiles in a single focused run:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

**PASS: 31.163s, one top-level integration test, both profile subtests passed,
zero skipped.** Each profile runs both real CLI and GUI verification scripts.
The pure SHA padding/chunk/length-overflow tests also passed using
`fairyc test --locked --file tests/sha256.fy`: **2 passed, 0 failed**. Both Go
and authored checks admitted the sources; the final lock remains checked.

One additional authored release run hashed **100,663,297 bytes (96 MiB + 1)**
and matched the independent streaming reference:
`d3507795de3e2201f9f980a3764452bea3cf7e050de8865e6760d0274040a2ef`.
This verifies that the compiler example's old source-size cap is not retained.

| Compiler | Development | Release |
|---|---:|---:|
| Go CLI | 241,152 bytes | 152,064 bytes |
| Reused authored CLI | 272,896 bytes | **164,864 bytes (161 KiB)** |

The authored release is a GUI-subsystem PE importing only Windows' USER32,
GDI32, COMDLG32 and KERNEL32 DLLs. It uses the static CRT and runs without
Python. Python is needed only for the optional verification helpers.

Receipts and retained authored images are in `.cache/windows-hash-20261111/`:
`go-utility.log`, `sha-unit.log`, `authored-{development,release}-{cli,gui}.json`,
`large-file.json`, `release-size.json` and `summary.json`. The summary records
source/compiler/companion/runtime-pack/image/report hashes. No generated
artifacts or fixture contents are committed.

## Earlier attempts and limits

Early source admission failures were corrected with SDK-owned dynamic buffers,
explicit Result handling and an unambiguous scalar count for dialog replies;
no compiler constraint was relaxed. Initial GUI-verifier attempts failed on
Windows 11's DirectUI filename/confirmation controls and a synchronous OK click
waiting inside the modal overwrite confirmation. The verifier now observes
the actual child controls and posts the native OK button click before handling
confirmation. The stopped attempts are not counted as passes. Production
hash/save behavior was not replaced with test-only handlers.

GUI confirmation selectors cover the installed Korean dialog and English
labels; other dialog locales, accessibility/copy keyboard operation, physical
input, physical cross-monitor DPI transitions and stalled network I/O were not
certified. Same-size concurrent modifications are not detected as a snapshot.
Linux, hosted CI, a fresh whole-compiler bootstrap and the unchanged minimal
example's complete suite were omitted because this task adds an independent
Windows utility and focused verification. No existing min-GUI runtime files
or compiler/SDK implementation changed.

Implementation commits: `2e667c44` (Fairy core/CLI), `2f9a8d46` (native GUI),
`67576396` (native-dialog/Go regression). Public compiler/language/SDK API, DB,
package dependency policy, repository hygiene policy, CI runners/workflows and
existing versions are unchanged. A new example and its product CLI are added;
there is no push, deployment or publication in this task.
