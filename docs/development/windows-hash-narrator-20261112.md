# Normal-desktop Narrator observations — 2026-11-12

`examples/win32_hash/narrator_session.py` adds the normal Windows shell launch
path for Narrator. `ShellExecuteExW/open` succeeds without `runas`; the isolated
desktop's CreateProcess error 740 does not imply that normal Narrator startup
requires an administrator prompt. The session refuses an already-running
Narrator and cleans up only its own returned process handle.

The launch smoke passed, with the owned Narrator confirmed running and stopped:

```powershell
python -B examples/win32_hash/narrator_session.py
```

Receipt: `.cache/windows-hash-narrator-20261112/launch.json`.

Local speech diagnostics used the actual Narrator+Ctrl+X last-spoken-phrase
command, with clipboard preservation and owned-foreground checks. The Open
button's name, button role and Alt+O were observed in Narrator's speech history.
A diagnostic also captured the computation-complete announcement. These are
partial observations, not a complete passing speech suite. File-path, full
64-digit digest, all buttons and error speech were not reliably certified.
Audio was not recorded. Diagnostics encountered focus changes and speech-queue
timing; inconclusive runs are not passes.

The user stopped further desktop/input testing because it interfered with
their work. Experimental full-suite changes were discarded. The app and its
existing verification commands remain unchanged; only the verified shell-launch
helper and this documentation are retained. No Narrator or Hash test process
remained at the cleanup check. Future desktop tests must be coordinated with
the user's available time; no unattended input testing is scheduled here.

EXE remains 188,416 bytes (184 KiB); no rebuild, public API, DB, normative
source-of-truth, dependency or CI/runner change was needed. Existing untracked
`docs-site/` is preserved. Local commits only; no push/release/deployment.
