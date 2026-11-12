# Fairy Hash responsive report saving, 2026-11-12

## Ownership and safety boundary

The GUI no longer executes report filesystem calls on its UI thread. Fairy's
`saving.fy` chooses validation, a bounded 32-name exclusive sibling search,
write/sync sequencing, a second destination/source-identity comparison, publish
authorization, cancellation and status. The saved JSON is still constructed in
Fairy and matches the synchronous CLI report. Hash arithmetic is unchanged.

`writer.fy` owns a private native context. `native/writer.c` copies both paths and
report bytes (at most 1 MiB), creates an I/O worker plus a separate cancellation
worker and executes one requested operation at a time. It retains no Fairy view,
callback or HWND. Events publish immutable operation results. The UI observes
readiness, pumps standard window messages and waits for at most 16ms. Cancellation
API calls and cleanup/CloseHandle stay outside the UI thread as well.

An exclusive `CREATE_NEW` temporary is opened with write/delete access and marked
using reversible `FileDispositionInfo` deletion before writing. Fairy requests
write, FlushFileBuffers and another target identity check. Foreign temporary
collisions are skipped without truncation or deletion. Cleanup removes an owned
temporary by closing its handle; it never deletes a possibly reused path after
close. If marking/closing fails, error 10 reports cleanup failure rather than
claiming success. No partial report is published after a write/sync failure.

At Fairy's publication boundary Cancel becomes disabled. The worker clears the
temporary deletion disposition and issues `FileRenameInfo` replacement on the
same handle. A failed rename rearms deletion; a successful rename is never
marked for deletion during cleanup. It closes the handle before reporting final
success. Windows documents the reversible deletion and rename primitives in
[SetFileInformationByHandle](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-setfileinformationbyhandle)
and [FILE_RENAME_INFO](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_rename_info).
This is not a transaction or power-loss durability guarantee.

Open/Copy/Save stay disabled until cleanup completes; the computed digest remains
visible and can be copied/saved again afterwards. A pre-publish Cancel retains
the window/result and cannot publish later. Close before publication requests
cancellation and waits for cleanup. Close during publish lets the commit settle
and then waits for cleanup. `hash_ui_poll` keeps dispatching messages even while
Close is latched, so the window still paints, answers pings and resizes before
exit. A layout error cannot bypass the final cleanup wait. One save context is
bounded, including cancelled cleanup; there is no unbounded retired-save queue.

## Focused CLI verification

```powershell
python -B examples/win32_hash/verify_writer.py
python -B examples/win32_hash/verify_slow_save.py --compiler .cache/windows-gui-fairy-cli-20261109/low/fairyc.exe
# Optional --gate-exe <path> retains the test-only image for focused reruns.
```

The native writer probe builds the unchanged production C with MSVC
`/W4 /WX /O2 /MT`. All three cases pass: discard after write/sync, failed publish
onto an actual read-only destination, and complete replacement. Existing contents
survive the first two, owned temporaries are absent and live contexts return to
zero. This also checks refusal of a second live native context.

`verify_slow_save.py` copies the example into an owned temporary workspace and
adds hooks only in that test image. Named-pipe ReadFile gates block the worker
at destination check/create/write/flush/publish/cleanup. The controller verifies
the pipe client PID and GetThreadIOPendingFlag, drives actual Open/Save dialogs
and real controls using standard Windows messages, then releases the owned pipe.
Retained gates deliberately reissue after cancellation. Disk-full failure is
injected for write/flush/publish; no user's disk is filled. Native write, flush,
rename, identity checking and handle cleanup still execute on local files.
No fabricated result/path/hash is injected into a private app protocol.

All **18** cases pass:

| Group | Cases | Checked behavior |
|---|---:|---|
| Pending check/create/write/flush/publish | 5 | Old result unchanged until commit; responsive ping/resize; exact JSON after release |
| Cancel check/create/write/flush | 4 | Window/result retained, old destination preserved, temporary removed |
| Write/flush/publish failure | 3 | Old destination preserved and temporary removed |
| Retained cancel then reuse | 1 | Busy cleanup stays bounded; no late publish; subsequent save succeeds |
| Destination changed to source hard link | 1 | Recheck refuses publication and source bytes survive |
| Foreign temporary-name collision | 1 | Foreign file preserved; another exclusive name succeeds |
| Close during retained write/publish/cleanup | 3 | UI remains responsive; exit follows cleanup; old or complete new result survives |

The authored test image passed all 18 cases: cancel **26.178–34.070ms**, maximum
ping **10.300ms** across the four runs, maximum pending resize **45.779ms**.
Receipt: `.cache/windows-hash-save-20261112/pending-final.json`. This run preceded
the final additional guard against layout errors bypassing cleanup.

After that guard, the current Go frontend built the same final Fairy sources and
passed all 18 again:

```powershell
go build -mod=readonly -o .cache/windows-hash-save-20261112/fairyc-go.exe ./cmd/fairyc
python -B examples/win32_hash/verify_slow_save.py --compiler .cache/windows-hash-save-20261112/fairyc-go.exe --gate-exe .cache/windows-hash-save-20261112/gated-go-final.exe
```

Final cancel observations are **35.445–39.977ms**; maximum ping across the four
runs is **14.365ms**, pending resize **26.515ms**. Close observations include an
intentional retained-gate hold of at least 100ms and controller checks; their
**231–271ms** wall times are not standalone shutdown benchmarks. All cases verify
old/complete-new content and no owned temporary after normal exit. Receipt:
`pending-go-final.json`. These are host observations with 500ms UI/cancel bounds,
not real-time OS guarantees. Physical input and desktop pixels are not inspected.

## Production and development/release integration

The reused authored compiler passed `check` and `build --locked --release` on the
final source. Production CLI **19** cases passed. GUI job-only checks passed actual
Unicode/empty/binary selections, JSON reports and source/hard-link protection.
A 64 MiB job completed correctly with 39 progress samples; ping max **34.768ms**,
resize **37.320ms**, two hash cancels **26.600/22.110ms** and window reuse passed.
After the last cleanup guard, the final production image passed job-only again
at 16 MiB, including report saves and source/hard-link protection; its ping max
was **49.224ms**, resize **155.095ms**. Host concurrency affects these timings;
this is not a throughput comparison. Receipt: `production-job-final.json`.

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_JOB_ONLY='1'
# Unset FAIRYC_HASH_SLOW_IO and FAIRYC_HASH_USABILITY.
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

Development/release integration passed **100.091s**, including CLI 19, SHA
differential 24 and native GUI report/job/cancel/reuse checks in each profile.
Their images are **300,032 / 182,784 bytes**, separate from the authored image.
This integration preceded the final layout-error cleanup guard; final-source
Go pending-save and authored production checks above cover the subsequent edit.

Two implementation/verification issues found earlier were fixed: the Go emitter
rejects duplicate repr(C) C tags, so the writer uses `WriterObservation`; repeated
save status text could satisfy a verifier before the next asynchronous save even
started, so repeated checks observe a distinct completed-hash state first and
wait for action re-enabling. Failed initial runs were not counted as passes.
Temporary compiler diagnostics used to inspect the C-tag error were removed.

## Size and remaining limits

Final authored EXE: **204,288 bytes (199.5 KiB)**, **+9,728 bytes (9.5 KiB)** versus
the prior 194,560-byte slow-reader image. SHA-256:
`ac1c31450c6f11a7888cf67424a9e0c01a71fa1aa42d49db760214c90251d8e5`.
System DLL imports remain USER32, GDI32, COMDLG32, COMCTL32, OLE32 and KERNEL32.
The save-test environment-variable literal is absent from production, in both
UTF-8/ASCII and UTF-16 forms. No verification runtime/framework is shipped.
Receipt: `.cache/windows-hash-save-20261112/size-final.json`.

A driver that does not honor cancellation can postpone cleanup and normal exit
indefinitely; the window continues pumping while it waits. Normal Close waits
for a safe boundary instead of killing the worker halfway through publication.
Forced process termination or power loss between clearing disposition and rename
may leave an owned temporary. Cleanup failure is surfaced; remote filesystem
semantics or physical faults may also prevent removal. There is no general
concurrent-file snapshot or hostile-directory race guarantee. Native file-dialog
browsing and synchronous CLI saving are outside this background GUI save path.

No real slow SMB/disk, network failure, physical keyboard/drag, screen-reader,
cross-monitor DPI, high-contrast or desktop-pixel suite was certified here.
General compiler suites, fresh bootstrap and hosted CI were not rerun; focused
development/release and native GUI checks cover this example's changed path.
`git diff --check` and Python syntax parsing pass. No public compiler/language/SDK
API, DB, dependency policy, CI workflow, runner or normative source-of-truth
document changed. Only example source/config/lock, focused verification and
development documentation changed. Local commits only; no push/release/deploy.
