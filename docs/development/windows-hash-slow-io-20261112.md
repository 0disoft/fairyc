# Fairy Hash responsiveness during pending I/O, 2026-11-12

## Implementation and ownership

The GUI now issues open, read and metadata requests to a private Win32 transport.
Fairy owns validation, SHA-256, progress, event/cancel policy and report output.
`native/reader.c` only performs Win32 operations and chunk/metadata transfer. The
synchronous CLI remains on `hash_file::compute` / `std::fs`, with the optimized
Fairy SHA implementation unchanged.

Each native context owns its UTF-16 path, 64 KiB read buffer, request/ready/stop
events and reader-thread handle. The GUI, read worker and cancellation worker
each hold a reference. The GUI copies completed chunks before acknowledging the
next read; the worker never retains a Fairy buffer, callback or HWND. Metadata
is immutable while its ready event is signalled. A private `@repr(C)` output
pointer carries snapshots without adding a public compiler/SDK ABI.

Fairy's compute loop polls UI events, checks reader readiness, consumes/hash-updates
one chunk and renders progress at most every 100ms. A pending open starts with
unknown total size; zero total is not treated as completion. The native wait
wakes for either readiness or window input, with a 16ms upper wait interval.
Cancel/Close is checked before consuming results. Final size, byte count and
file identity come from the same open handle; prepared saves still protect the
source and hard-link aliases.

Dropping a job signals cancellation and releases only the GUI reference. Both
the read API and CancelSynchronousIo execute outside the UI thread; the GUI
never joins either worker. Storage is freed only after all references finish.
At most **two live reader contexts / four helper threads** are permitted, even
when a driver ignores cancellation. A third job returns GUI error 9/retry text.
One pending cancelled read can coexist with a new job. No result callback or
global completion message can overwrite a different reader's state.

The cancellation worker is necessary: an initial direct UI-thread cancellation
passed the authored/release gate runs but hung the Go development GUI on the
deliberately reissued/uncancellable-read fixture. Those failed attempts remain
failures (`go-pending.log`, `go-pending-development.log`). Moving the cancellation
request off the UI fixes that same development case and both final profiles.
Windows documents cancellation as a request rather than completion; see
[CancelSynchronousIo](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelsynchronousio)
and [cancellation buffer lifetime](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelioex).

## Real pending-I/O CLI verification

`verify_reader.py` builds `testing/reader_probe.c`, which includes the unchanged
production transport with MSVC `/W4 /WX /O2 /MT`. It checks **1,234,567** actual
patterned file bytes, short-buffer refusal, size/identity consistency, EOF,
a named-pipe ReadFile observed pending by GetThreadIOPendingFlag, cancellation
and **zero live contexts** after both workers finish. The measured drop interval
is below GetTickCount64's resolution (reported 0ms), not a zero-latency guarantee.
Receipt: `.cache/windows-hash-slow-io-20261112/reader.json`.

`verify_slow_io.py` copies production sources/config into an owned temporary
workspace. Only there, `testing/delayed_reader.c` gates open/read/metadata calls
with a real blocking named-pipe ReadFile. The controller verifies the pipe's
client PID and pending native I/O, sends standard control/window messages and
releases owned pipe servers. Retained-read fixtures deliberately reissue after
ERROR_OPERATION_ABORTED to emulate a backend that does not honor cancellation.
This is actual OS pending I/O in a controlled test transport; it is not a slow
physical disk/SMB certification, a UI-thread sleep or a production debug protocol.

Final authored gate command, using the same reused compiler:

```powershell
python -B examples/win32_hash/verify_slow_io.py --compiler .cache/windows-gui-fairy-cli-20261109/low/fairyc.exe
```

All **9** cases pass on the final implementation:

| Case | Observation |
|---|---|
| Pending read then release | Correct `abc` SHA; elapsed time advances while blocked |
| Pending read cancellation | 21.536ms; window stays usable |
| Pending open cancellation | 21.899ms |
| Pending metadata cancellation | 21.551ms |
| Late result isolation | A new job finishes while the cancelled old read remains pending |
| First retained read cancellation | 21.918ms |
| Second retained read cancellation | 21.906ms |
| Capacity/reuse | Two contexts remain bounded; retry succeeds after release |
| Close while read pending | Clean process exit in 91.192ms |

Pending-I/O GUI ping max is **12.971ms**. All pending resize/layout checks stay
below **25.074ms**; physical input and desktop pixels are explicitly not tested.
These are observed host timings, with 500ms response bounds, not real-time OS
guarantees. Receipt: `pending-authored-final.json` in the same cache directory.

## Production and Go integration

The authored CLI `check`, `build --locked --release`, existing CLI **19** cases
and differential SHA **24** cases pass on the final production image. The normal
GUI job-only check additionally passes empty/Unicode/binary files, saved JSON
reports and source/hard-link overwrite refusal before exercising a 64 MiB job:

```powershell
python -B examples/win32_hash/verify_cli.py
python -B examples/win32_hash/verify_sha.py
python -B examples/win32_hash/verify_gui.py --job-only --job-mib 64
```

The final 64 MiB native job has correct digest, **42** coherent progress samples,
GUI-thread ping p95/max **8.985/20.178ms**, resize/layout **54.002ms** and cancel
responses **22.359/25.803ms**, followed by successful calculation in the same
window. Observed busy time is **8.815s**; concurrent host work/observer overhead
means this is not a throughput comparison with the earlier SHA benchmark.
Receipts: `final-cli.json`, `final-vectors.json`, `final-gui-job.json`.

Go frontend development/release, each with CLI 19 + SHA 24 cases:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_SLOW_IO='1'
# Unset FAIRYC_HASH_JOB_ONLY and FAIRYC_HASH_USABILITY.
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
# Then unset FAIRYC_HASH_SLOW_IO and set FAIRYC_HASH_JOB_ONLY=1 for ordinary jobs.
```

| Integration | Result | Development | Release |
|---|---|---|---|
| Pending I/O, 9 cases/profile | PASS, 67.563s | Cancel 21.291–22.715ms; close 62.126ms | Cancel 21.225–21.816ms; close 68.144ms |
| Ordinary files + 16 MiB job/profile | PASS, 76.703s | Ping max 67.580ms; two cancels + reuse | Ping max 13.775ms; two cancels + reuse |

Normal Go development/release images are **284,160 / 176,128 bytes**. Gate builds
contain test code and are larger; they are not production size measurements.
Receipts: `go-pending-final.log`, `go-normal-final.log`. No unchanged successful
check was counted as covering the initial failed cancellation design.

## Size, scope and limits

Final authored production EXE: **194,560 bytes (190 KiB)**, **+4,096 bytes (4 KiB)**
over the preceding 190,464-byte SHA build. SHA-256:
`4ef1143fa14a4355e1c3575ab710c5419a35b8ad8dd5a289ae8e91d269046f23`.
System DLL imports remain USER32, GDI32, COMDLG32, COMCTL32, OLE32 and KERNEL32.
The production image contains neither the test gate environment-variable literal
nor retained-read markers. No framework or verification runtime is shipped.
Receipt: `size.json`. Build/cache output remains ignored.

No public compiler/language/SDK API, DB, dependency policy, CI workflow, runner
or normative source-of-truth document changed. The example's private bridge,
Fairy source lock, focused Go/Python verification and execution docs changed.
No version bump, push, publication or deployment.

Cancellation remains best-effort at the OS level. Pending file handles/storage
may outlive logical cancellation; the two-context cap prevents unbounded growth.
Third-job retry may remain necessary until a stalled backend returns. Native
file dialogs and synchronous report saving are outside this reader change.
Concurrent file changes are not snapshots. No real slow disk, network outage,
filesystem filter/driver fault, physical keyboard/drag, screen-reader voice,
different-monitor DPI, high-contrast or desktop-pixel suite was certified here.
The unchanged SHA unit suite, general compiler suite, fresh bootstrap and hosted
CI were not rerun; current differential/native integration covers hash output.
