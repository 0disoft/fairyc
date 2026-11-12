# Fairy Hash source changes during calculation, 2026-11-12

## Policy and implementation

Fairy now compares the initial and final source size, volume/file ID, creation
time, last-write time and change time. It also observes the selected path again
while the original handle remains protected. File access time is deliberately
excluded: ordinary reads can change it. The original streaming SHA implementation,
64 KiB buffer and progress cadence remain unchanged. There is no second hash pass
or per-chunk metadata query.

Windows normal write/delete sharing remains denied. A normal overwrite attempt
therefore fails with ERROR_SHARING_VIOLATION instead of changing bytes behind the
reader. The additional comparisons reject unexpected changes and backends whose
observations differ despite this protection. Windows documents share modes and
attribute-access exceptions in
[CreateFileW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew),
and timestamp update timing/precision in
[File Times](https://learn.microsoft.com/en-us/windows/win32/sysinfo/file-times).
Modification time alone is insufficient; the NTFS restored-mtime fixture below
therefore checks ChangeTime separately.

The private native reader transports raw `FileBasicInfo` observations in its
existing result. C obtains metadata and handles, while `reader::unchanged` in
Fairy decides equality. At EOF phase 3 reports final open-handle observations;
Fairy compares them, acknowledges readiness, and the worker then reopens the
selected name for a phase-5 observation. Phase 6 reports a failed final name
check. All GUI filesystem calls stay on the existing reader worker, with the
separate cancellation thread and two-context bound retained. Cancel/Close still
abandons the GUI job without joining file/cancellation workers.

The synchronous CLI retains its `std::fs` reading and report save. An example-only
native Guard owns a separate read-protected handle and observations; it does not
inspect the SDK File resource layout or introduce a public SDK API. Fairy compares
guard handle/path observations and the SDK file ID before constructing a Digest.
The first SDK open still owns ordinary path/kind error handling.

Source-change/final-observation failure returns example error **11**. The GUI's
ready flag stays false, displayed digest stays empty, progress never completes
to 100%, and Copy/Save remain disabled. A previously valid selection cannot supply
a stale result. Open stays usable for a new calculation. The CLI returns exit 1
and only `{"ok":false,"errorCode":11}`; report formatting/save never starts, so an
existing output stays unchanged. A new stable selection can be calculated and
saved successfully in the same window.

## Native transport and actual mutations

```powershell
python -B examples/win32_hash/verify_reader.py
python -B examples/win32_hash/verify_source_changes.py --compiler .cache/windows-gui-fairy-cli-20261109/low/fairyc.exe
```

The production reader probe passes MSVC `/W4 /WX /O2 /MT`, actual **1,234,567**
patterned bytes, handle/path metadata, real pending ReadFile cancellation, zero
live readers, a real rejected write open (error 32), and a same-size metadata
change through SetFileTime on a FILE_WRITE_ATTRIBUTES handle. Attribute access
does not require relaxing production write sharing.

The GUI/CLI change verifier copies the example into an owned temporary workspace.
Only that test build adds `testing/changing_reader.c`, named-pipe ReadFile gates
and relaxed write/delete sharing for names beginning `change-`. Production and
the protected/CLI cases retain FILE_SHARE_READ. Gates observe the client's owned
PID and actual pending native I/O. The read gate is after the first 64 KiB chunk
has reached Fairy/SHA; the EOF gate is after the original-handle comparison but
before the final selected-name observation. Real disk files are modified, not a
fabricated result/status/path in a private app protocol.

All **9** cases pass with both the current Go CLI build and reused authored CLI:

| Case | Actual observation |
|---|---|
| Protected overwrite | Write open rejected with OS error 32; original bytes and correct digest retained |
| Protected timestamp mutation | Attribute-only timestamp edit under production sharing; changed result rejected |
| Same-size overwrite | Already-hashed first 4 KiB overwritten; file length unchanged; changed result rejected |
| Restored last-write time | Same-size bytes changed, exact original LastWriteTime restored, ChangeTime differs; result rejected |
| Grow | Real append; result rejected |
| Shrink | Real truncation; result rejected |
| Replace at EOF | Same-size replacement via test-only Windows POSIX rename while old delete-shared handle stays open; result rejected |
| Window reuse | Stable file recalculated and correct JSON saved after rejected jobs |
| CLI mutation | Real attribute edit at final guard observation; no digest emitted, exit/error 1/11, old report preserved, no temporary left |

For every rejected GUI result, the verifier observes an empty digest, disabled
Copy/Save and enabled Open. It sends real native BM_CLICK operations to both
disabled controls and checks unchanged clipboard sequence and absence of a Save
dialog. It preserves clipboard contents and injects no physical keyboard/mouse
input. The test's relaxed-sharing flag is explicit; this is not evidence that
production allows these ordinary writes. An initial MoveFileEx-based test fixture
could not replace an open target; its test-only POSIX rename fixes that fixture,
without changing production sharing or publication behavior.

The Go-built source-change run reports GUI ping maximum **2.478ms**. The authored
source-change run reports **4.387ms** and all 9 cases pass. These are host
observations against a 500ms bound, not real-time guarantees. Authored receipt:
`.cache/windows-hash-source-changes-20261112/authored-changes.json`.

## Normal and pending-I/O regressions

The reused authored compiler passed `check` and `build --locked --release`.
Production CLI **19** cases pass. The current Go release also passed differential
SHA **24** and job-only native report/window checks before the authored build.
Hash arithmetic was not changed.

Final authored production command:

```powershell
python -B examples/win32_hash/verify_cli.py
python -B examples/win32_hash/verify_gui.py --job-only --job-mib 64
```

The 64 MiB job matches Python's SHA reference, has **48** coherent progress
samples, ping p95/max **11.882/20.016ms**, resize **51.690ms**, and two hash cancels
**25.402/23.131ms**, followed by successful window reuse. Native Open/Save,
empty/Unicode/binary results, JSON reports and source/hard-link protection pass.
Observed job wall time is **10.595s** while other builds run; it is not a measured
throughput regression or a storage benchmark. Receipts: `cli-final.json` and
`gui-final.json` in the same cache directory.

The existing pending-I/O integration passes both current Go profiles:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_SLOW_IO='1'
# Unset FAIRYC_HASH_JOB_ONLY and FAIRYC_HASH_USABILITY.
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

PASS **56.741s**; development/release images **342,016 / 220,160 bytes**. Each
profile passes CLI 19, SHA 24 and all 9 pending-I/O GUI cases. Development cancel
observations are **21.521–23.017ms**, close **101.127ms**, ping max **3.812ms**;
release cancel **21.314–21.676ms**, close **55.904ms**, ping max **0.356ms**. Late
cancelled results, retained workers and the two-context cap/reuse still pass.
The test hook was narrowed to FileIdInfo and explicitly targets only the first
reader open, leaving the second synchronous guard open untouched. An earlier
stale hook match failed and was corrected before the final successful run.

## Size, scope and limits

Final authored EXE: **208,384 bytes (203.5 KiB)**, **+4,096 bytes (4 KiB)** versus
the preceding 204,288-byte save build. SHA-256:
`aa8be15e6fd6b5d0dc72a74ce22c0287a5cc71ebff3cd6daaf5b87d4198d4d59`.
System DLL imports remain USER32, GDI32, COMDLG32, COMCTL32, OLE32 and KERNEL32.
The change-gate environment variable is absent from production in ASCII/UTF-8
and UTF-16 forms. No test runtime or framework is bundled. Receipt: `size-final.json`.

This is guarded observation, not a filesystem snapshot. Coarse/stale timestamps,
unusual remote/device behavior, or changes that bypass sharing and restore every
observed field can escape comparison. The application does not continuously watch
files after a completed calculation, and does not claim power-loss durability.
An attribute-only change can conservatively invalidate unchanged contents. Last
access changes alone do not invalidate a hash. NTFS is the local tested filesystem;
no slow physical disk/SMB fault or other filesystem timestamp behavior was certified.

Physical keyboard/drag, screen-reader voice, cross-monitor DPI, high-contrast,
desktop pixel suites, general compiler suites, fresh bootstrap and hosted CI were
not rerun because this slice changes source observations/result eligibility.
`git diff --check`, Python syntax parsing, native transport and focused Go/GUI
checks pass. Example CLI error 11 and private bridge observations were added;
public compiler/language/SDK API, DB, dependency policy, CI workflow, runner and
normative source-of-truth documents did not change. Local commits only; no push,
version bump, publication or deployment.
