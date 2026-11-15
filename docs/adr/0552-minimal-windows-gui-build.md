# ADR 0552: Minimal Windows GUI build inputs

- Status: Accepted for the minimal Windows prototype
- Date: 2026-11-08

The optional workspace-root `fairy.windows.json` uses schema
`fairyc.windows-build/v1`. It has a required `subsystem` of `console` or
`windows`, and optional `libraries`, `sources`, and `resources` string arrays.
Unknown/duplicate fields, nulls, duplicate inputs and trailing JSON reject.
The existing language/package manifest v1 is unchanged.

Libraries are lowercase bare names ending in `.lib` (32 maximum), not paths,
response files or linker switches. Declared `.c` (16) and `.rc` (8) paths are
canonical portable ASCII paths relative to the root, at most 240 bytes, with
no parent, drive, backslash or symlink components. Text is bounded UTF-8,
nonempty and NUL-free: config 64 KiB, each native source 4 MiB, combined 32 MiB.
C sources use C11 and the existing profile/static CRT. C/RC include resolution
can use the project root and SDK: explicit native source is trusted code,
not a sandbox. Transitive headers/assets are not claimed hermetic.

GUI builds keep the Fairy `fn i32 main()` entry. The linker selects
`/SUBSYSTEM:WINDOWS` with the CRT's `mainCRTStartup`; it does not jump directly
to Fairy code or bypass CRT initialization. PE inspection requires the
explicitly selected subsystem; console-only inspection remains console-only.
Resource compilation uses the resolved SDK's x64 `rc.exe`, then links `.res`.
No shell command or raw compiler/linker flag is accepted from this config.

The config is native build input, read freshly for Windows workspace build/run,
separate from source lock, semantic cache and language identity. Native tests
retain console images and do not implicitly acquire application GUI settings.
Missing config preserves the current path. Build/link/resource failure must
not replace a previously published executable. Linux ignores this Windows-only
file. Compiler-host bootstrap never implicitly consumes application settings.

The Go input loader retains each declared file's inspected OS identity, size
and modification time, compares them with the opened handle before reading,
and rechecks that handle after reading. Replacement, observable in-place
changes, read/close failures and cancellation reject the entire project; no
partial C/RC snapshot reaches compilation. Only an absent config at initial
inspection selects the no-config path; disappearance after inspection is an error.
`windowsbuild.Load` keeps its signature. `LoadWithOptions` accepts one optional
`LoadOptions` value with per-call `Context`, `OpenRoot`, `Lstat` and `Open`
dependencies; nil fields retain the host filesystem and Background context.
Cancellation is cooperative between I/O calls with reads of at most 32 KiB,
not preemption of blocked filesystem calls. A growing input is probed only up
to its existing byte limit plus one; 100 consecutive empty successful reads
fail with `io.ErrNoProgress`. Metadata checks do not provide an atomic or
cryptographic snapshot against a writer that restores size and timestamps,
and transitive native inputs remain outside this boundary.

Initial UI integration uses a small project-owned C bridge for WndProc and
message handling, called through the existing unsafe C import boundary. No
retained Fairy callback, public export, raw Win32 SDK import generator or UI
framework is introduced. Native Go and Fairy driver completion is recorded
separately; a parser-only stage is not GUI execution evidence.

The [runnable window example](../../examples/win32_minimal/README.md) and
[local execution record](../development/windows-gui-build-20261108.md) cover
Go CLI, authored Fairy CLI/companion, both target profiles, resource loading,
console preservation and failed-build artifact preservation. Compiler-host
bootstrap remains a separate boundary from application GUI configuration.

References: Microsoft [subsystem](https://learn.microsoft.com/en-us/cpp/build/reference/subsystem-specify-subsystem),
[CRT entry points](https://learn.microsoft.com/en-us/cpp/build/reference/entry-entry-point-symbol),
[resource compiler](https://learn.microsoft.com/en-us/windows/win32/menurc/using-rc-the-rc-command-line-).
