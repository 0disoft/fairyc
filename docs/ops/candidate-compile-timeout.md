# Candidate compiler deadlines

The private `fairyc-candidate-build` bridge now accepts `--compile-timeout`
with a default of 2 minutes and an explicit bound of `(0,10m]`. This deadline
applies separately to each MSVC compile and the final link. Candidate frontend
emission has a separate `--frontend-timeout`, default 1 minute and explicit bound
`(0,10m]`. The selected value reaches `candidatebuild.Options.Timeout` and the
actual frontend process deadline. Program execution limits are separate.

`candidatebuild.Options.CompileTimeout` reaches the real
`BuildWindowsMSVCTargetPartsWithTimeout` driver. The existing target-parts API
retains its 60-second default. No overlay compiler is needed to change the
private project build deadline.

On Windows, timeout cancellation uses the system-directory `taskkill.exe` to
terminate the compiler or candidate frontend process tree, then bounds output-pipe draining. Windows
helpers run hidden. Cancellation failure falls back to killing the direct
process; no guarantee of descendant cleanup is claimed for that failed fallback.
Failed compile/link paths still return no publishable image.

Executed checks:

- `bun tools/run-go.ts test ./internal/cdriver ./internal/candidatebuild
  ./tools/candidate/fairyc-candidate-build -run
  'TestToolCompileDeadline|TestCompileTimeoutRejected|TestWindowsProjectSeparateCompilationAndFailures|TestCachedPacketRejectsChangedBytes'
  -count=1` passed.
- `bun tools/run-go.ts test ./internal/candidatebuild -count=1` passed.
- A real helper parent spawned a child; the 1-second deadline terminated both.
  The child was checked by its native process handle/exit status, and a successful
  helper plus rejected invalid deadlines exercised the same command runner.
- Separate C compilation, reproducible images in both profiles, compile/link
  failures and immutable cached-packet behavior passed their existing tests.

This fixes a real private release compile that exceeded the old 60-second
deadline. Broader project CLI timeout mapping and Linux build/execution behavior
remain separate work. Public CLI, SDK ABI, DB and CI definitions are unchanged.

## Direct complete frontend build (2026-10-12)

`--frontend-timeout 2m` now processes the complete 355-source dispatcher directly
through the Stage2 Fairy frontend, compiles 45 C files and publishes the optimized
image in 365.75 seconds. The 4177920-byte result exactly matches the previously
verified Stage1/Stage2 image. No cached packet or Go frontend fallback is used.
Invalid zero/negative/above-10-minute options reject before changing the previous
image. A real parent/child regression proves deadline cancellation terminates the
candidate child and preserves the previous executable. Frontend range/packet,
existing build-failure and compiler-tree deadline checks pass.

`cdriver.ConfigureCommandCancellation` exposes existing host cancellation to the
private native bridge; stable/public commands and SDK ABI remain unchanged.
Receipt: `evidence/candidate-frontend-timeout-20261012.json`.
