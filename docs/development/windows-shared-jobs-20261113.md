# Shared Win32 jobs — 2026-11-13

## Delivered scope

`modules/win32_jobs` owns reusable native reader/writer transport, Fairy C
bindings, overflow-safe progress/elapsed formatting, a terminal job-state
controller and pure policy tests. Fairy Hash keeps standalone copies refreshed
by `python -B tools/sync_win32_jobs.py`; both Win32 generators install the same
sources. Scaffold tests reject stale Hash copies or embedded template inputs.

The controller verifies open/chunks/EOF/final source metadata before publishing
completion. Active, cancelled and failed work never reports 100%. Cancellation
is sticky; leaving the Reader resource scope requests native cancellation and
releases caller ownership without joining the I/O worker. Native storage survives
until OS completion; retiring readers/writers are bounded to two/one contexts.
Drivers that ignore cancellation can delay slot reuse. This is not an atomic
filesystem snapshot or an arbitrary Fairy-thread executor.

Hash retains SHA-256 processing and its save/publication policy in Fairy C.
The shared writer transports prepared writes; source checks, commit authorization
and cleanup waits remain the caller's responsibility. Cancellation after commit
starts is not rollback. Synchronous reader guard helpers are intended for CLI
use, not a GUI thread facing a slow filesystem.

The Win32 starter now offers file reading, LF counting, progress, elapsed time
and Cancel while retaining its text transformation example. Fairy C owns the
event-pumping loop and chunk processing; native workers only perform I/O. GUI
wiring was compiled, but actual visible control/keyboard behavior was not tested.

```powershell
fairyc new scanner --uid example.scanner --template win32
# Build/package inside scanner with its documented commands, then:
.\build\scanner.exe --scan .\input.bin
.\build\scanner.exe --scan .\input.bin --cancel-after 65536
```

`--scan` is a generated-app option, not a compiler option. It emits NDJSON
progress/result records and exits 0 for verified completion, 2 for cancellation,
or 1 for failure. Thresholds are evaluated at chunk boundaries, up to 64 KiB;
zero cancels before waiting for open/read. Counts named `lines` mean LF bytes.

## Executed checks

- Hash: locked check/release build, pure job-policy tests, native reader probe,
  three writer safety cases and 57 extracted-package CLI cases passed.
- The native reader consumed 1,234,567 bytes, cancelled an actual pending
  `ReadFile` with a measured caller drop time of 0 ms, retired to zero live
  readers and successfully read another file in the same process. Overwrite
  denial and same-size metadata-change detection passed. The 0 ms measurement
  is this probe's result, not a latency guarantee for arbitrary storage drivers.
- Writer cancellation/failure preserved existing results, removed temporary
  files and retired to zero live writers.
- Starter: 14 real CLI cases, including nine async cases, passed through both
  generators. Empty/Unicode/binary files, intermediate progress, immediate and
  mid-read cancellation, cancellation before final verification, completion past
  the threshold, missing files and invalid arguments were covered. Both generated
  pure test files passed.
- `tools/verify_win32_cli.py`: compiler copied outside the repository; creation
  with empty PATH, ten rejected creations, unchanged console scaffold bytes,
  independent build, packaging, extracted CLI and PE resources passed.
- `tools/verify_win32_starter.py`: independent Python generation/build/package,
  six rejected creations and existing-project preservation passed.
- Scoped CLI/scaffold Go tests, scaffold snapshot checks, Python syntax checks
  and `tools/sync_win32_jobs.py --check` passed.

The local compiler and JSON receipts are under
`.cache/windows-jobs-20261113/` (ignored). The development generated app is
`.cache/windows-jobs-20261113/scanner/`.

| Artifact | Before | After |
| --- | ---: | ---: |
| Fairy Hash release EXE | 213,504 bytes | 213,504 bytes (208.5 KiB) |
| Starter release EXE | 137,728 bytes | 157,696 bytes (154 KiB) |

The starter's complete added example costs 19,968 bytes (19.5 KiB), including
its GUI/CLI demonstration. This is not a standalone cost estimate for the module.
Final CLI-generated probe ZIP: 86,062 bytes; Python-generated probe ZIP: 86,072
bytes, with different product metadata. Hash's validated ZIP was 114,028 bytes.

## Boundaries and ownership

No GUI launch, desktop/keyboard/clipboard automation, Narrator, physical DPI or
slow-driver GUI check was performed. No full compiler bootstrap, broad benchmark
or hosted CI run was needed. Native probes and generated CLI paths provide real
I/O evidence; visible cancellation/focus behavior remains unverified.

The language and public SDK API, DB, dependencies, versions and CI runners were
unchanged. Source ownership/usage was updated in `docs/cli/command-contract.md`,
`docs/development/source-of-truth-packages-cli-and-sdk.md`, the module guide and
Hash/starter/tooling guides. Embedded source ZIP/checksums were refreshed;
generated binaries and receipts remain ignored. Local commits only; no push,
publication or deployment.

Implementation commits: `0b1f4c6f` (shared extraction) and `9dd8aeb7` (starter
integration/CLI). This record accompanies the final documentation and reuse-check
commit.
