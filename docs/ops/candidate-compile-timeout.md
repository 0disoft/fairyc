# Candidate compiler deadlines

The private `fairyc-candidate-build` bridge now accepts `--compile-timeout`
with a default of 2 minutes and an explicit bound of `(0,10m]`. This deadline
applies separately to each MSVC compile and the final link. Candidate frontend
emission keeps its own 60-second limit. Program execution limits are separate.

`candidatebuild.Options.CompileTimeout` reaches the real
`BuildWindowsMSVCTargetPartsWithTimeout` driver. The existing target-parts API
retains its 60-second default. No overlay compiler is needed to change the
private project build deadline.

On Windows, timeout cancellation uses the system-directory `taskkill.exe` to
terminate the compiler process tree, then bounds output-pipe draining. Windows
helpers run hidden. Cancellation failure falls back to killing the direct
process; no guarantee of descendant cleanup is claimed for that failed fallback.
Failed compile/link paths still return no publishable image.

Executed checks:

- `bun tools/run-go.ts test ./internal/cdriver ./internal/candidatebuild
  ./cmd/fairyc-candidate-build -run
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
