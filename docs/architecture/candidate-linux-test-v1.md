# Private Linux candidate workspace test v1

## Portable result boundary

`native_test_result` retains only canonical trap category and binary stdout/stderr.
The host executor validates OS status and its private control channel before
constructing that result. `project_test_options.test_native_report` takes an
explicit target identity and shares the existing text/JSON v1-v4, base64, counts,
selection order, lock and source-location formatter. The Windows wrapper keeps
its existing target and output bytes; semantic reports retain null target/profile.
Windows test execution converts each validated capture before retaining it.

Both-target emission and low/optimized host execution check mixed pass/fail
outcomes, exact binary base64 streams and Windows/Linux JSON target identities.
The Linux host adapter and command integration use the same result boundary.

## Native stdin capture

`linux_test_capture` opens a captured process with fd 3 trap control and one of
three stdin policies: mode 0 inherits caller input without reading it, mode 1
provides an already-closed pipe (EOF), and mode 2 opens the absolute private replay
file afresh. Replay opens use CLOEXEC/NOFOLLOW/NONBLOCK, require a regular file no
larger than 65,536 bytes, and move low descriptors above fd 3 before spawn actions.
Input is duplicated to fd 0 before stdout, stderr and control redirection, then
all remaining descriptors from 4 upward close in the child. Parent input handles
close on success and failure. Source file admission remains with the Fairy caller.

The wrapper shares `linux_run_capture.collect_owned`, its receipt validation,
process-group cleanup and 65,536-byte per-stream capture. Test deadlines admit
1..60,000 ms; the shared collector still serves ordinary run unchanged.
Windows low/optimized POSIX substitutes check all three modes, closed standard
descriptors, open/stat/type/size failures and descriptor cleanup. Both-target Fairy
emission and host wrapper execution check mode/path transport and failed-open
ownership. Native binary stdin/replay/EOF and symlink probes are authored but
skipped on Windows. The dispatcher selects this adapter for native test input.

## CLI connection and platform selection

`linux_file_cli` connects `test`, `test --help` and `test -h`. The shared
`project_test_cli` and `project_test_options` retain full-overlay admission,
production lock maintenance/verification, canonical test discovery, interpreter
execution, listing, JSON, source anchors, exact/file/directory selection and
root-reachable package/workspace selection. Invalid combinations retain the
existing argument diagnostics. Cache flags remain unsupported.

The isolated preparation tool additionally changes the one native-executor
import in `project_test_cli` from `native_cli` to `linux_test_cli`. It asserts the
expected original import exactly once and never changes the authored Windows
entry. Together with existing path/SDK/emitter choices this modifies exactly nine
source modules. The complete prepared entry has no Windows foreign dependencies.
The shared native reporter and timeout formatter take explicit target/platform
labels; the Windows report wrapper retains its target/output bytes and the timeout
wrapper retains its signature.

## Native test lifetime

`linux_test_cli` prepares checked test lowering once, then appends the existing
private i32 harness separately for each selected test. Each fresh bounded C packet
goes to the sibling GNU bootstrap companion with `--run-trap-control`, output
root equal to its fresh mode-0700 stage and output exactly `program`. No shell,
PATH search, frontend fallback or published `build/<name>` artifact is involved.
The companion retains its GNU object/link/ELF admission and ten-minute outer
capture budget. A zero-test selection needs no native companion.

Every test runs in its own temporary directory, with the same exact argument
bytes after `--`. Default stdin is EOF. Explicit `--stdin` is accepted only for
one selected native test after complete overlay/compiler admission and inherits
caller input only on program launch. `--stdin-file` reads one canonical
workspace-relative regular file through the selected rooted filesystem adapter,
with a 65,536-byte limit; each test gets a new private copy opened at offset zero.
Symlink, nonregular, missing and oversized input errors return 7. Replay is written
after compilation so the backend stage initially contains only `packet.bin`.

Native execution uses the fd 3 receipt protocol, per-stream 65,536-byte caps and
configured 1..60-second deadline. Valid traps become per-test failed outcomes
and later tests continue. Ordinary nonzero exit without a trap, signal, timeout,
output/control/backend failure or failed cleanup aborts with 6. Output without
`--json --capture-output` returns 2; native JSON capture retains exact padded
base64 bytes, including pre-trap output. Text/JSON v1-v4 retain canonical ordering,
lock state, locations, totals and 0/5 test-result status. Native reports identify
`x86_64-linux-gnu` and development/release; semantic reports keep null target/profile.

Explicit stage cleanup precedes retaining the result, and resource cleanup covers
early failures. The existing cleanup work/depth bounds and same-user/filesystem
failure limitations apply: failures may leave a partial owned tree. Arbitrary
workspace trees are never recursively removed. Program streams are captured for
the test report, with no live streaming. This is a compiler protocol rather than
a security boundary against malicious native code.

## Evidence and remaining boundary

Windows low/optimized adapter and wrapper tests pass for all stdin modes, receipt
validation and cleanup; existing run capture/process checks remain passing.
Portable report execution verifies both native target labels, pass/fail counts
and binary base64. Full Linux entry emits both GNU/Linux and Windows MSVC C with
no Windows foreign symbols, selects closed Linux support, and its portable command
dispatch executes in both host profiles. Source workspace `check`/`check --locked`
pass with 559 sources and 7,089 MIR functions; the prepared Linux workspace passes
both modes with 559 sources and 7,090 functions. The updated companion cross-build
is ELF64 little-endian x86-64. `go vet ./...` and diff checks pass.

Linux-native integration cases are authored for semantic/list/reference equality,
selectors/locations/package/workspace/empty tests, native development/release
reference equality, trap continuation, exact args and binary JSON, EOF/caller/replay
stdin, input bounds/symlinks, timeout/output overflow, invalid test attributes,
missing companion, prior-image preservation and private TMPDIR cleanup. Native
adapter probes cover actual binary stdin, EOF, replay and symlink rejection with
GCC/Clang. These cases are skipped on the Windows development host; Linux-native
success and distribution/self-host readiness are not claimed. Workspace new/cache
integration and a Fairy-authored GNU backend remain open; Go stays the default.
