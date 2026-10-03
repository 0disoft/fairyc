# ADR 0425: Stable Linux build and run

- Status: Accepted
- Date: 2026-10-03

## Decision

On Linux GNU amd64 with glibc, ordinary `fairyc build` and `fairyc run` use the
existing verified-MIR Linux C11 transport and production SDK companions. GCC is
selected first from PATH, with Clang used when GCC is absent. Compiler selection
occurs after source and lock validation. A failing selected compiler is reported;
there is no retry through another compiler.

Build publishes an executable `build/<package-name>` through the existing synced
staging path. Run uses a temporary ELF, the invoking workspace as cwd, verbatim
arguments after `--`, caller stdin and separated streams under the shared 64 KiB
output budget and 60-second timeout. Linux preserves only the low eight bits of
ordinary process status. The private fd-3 trap receipt distinguishes an ordinary
status 131 from division-by-zero trap 3; a recognized trap appends the category and
returns compiler status 5. Windows behavior is unchanged.

Native `fairyc test` was Windows-only at this decision; [ADR 0426](0426-linux-native-workspace-tests.md) adds the Linux adapter. Linux target conformance, binary
distribution and power-loss durability are not implied by command admission.
SDK stage0.79, language/20 and runtime/0 identities are unchanged.

## Validation boundary

Linux integration tests exercise development/release builds, executable mode,
retained-image preservation, stdin, UTF-8 stdout, stderr, normal/trap distinction,
process arguments and relative store paths. Save tests use test-only POSIX call
injection and bounded child kills, checking exact live/temp/lock bytes and restart
refusal. Both GCC and Clang are required for the save matrix. These tests run in
the existing Ubuntu Go test job, with no additional runner.

The current Windows checkout can type-check both host builds and execute Windows
regressions, but cannot execute a Linux ELF. Linux-only tests explicitly skip
here. Subsequent [Drimi Linux evidence](../compatibility/linux-drimi-20261003.md)
records the exact tested snapshot, i64 compiler fix and remaining stale-lock
test expectation failure. It is not a current-HEAD full-suite pass.
