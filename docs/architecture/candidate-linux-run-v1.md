# Private Linux candidate workspace run v1

`linux_file_cli` connects `run [--release] [--locked] [-- <program args>]`.
The shared native option parser owns duplicate flags, the argument separator,
UTF-8 argument transport and NUL rejection. Optional `--cache-root <existing directory>`
and `--cache-status` select the [verified IR cache](candidate-linux-cache-v1.md)
before `--`; all arguments after it belong to the program.
Run help is available without loading a workspace. This is the private Linux
candidate path; the reference compiler's default host policy is unchanged.

## Compilation and ownership

`linux_build_cli.run(true)` uses the same authenticated workspace load, entry,
lock and C packet as build. Ordinary and cached `project_emit` evaluation retain
the current compilation snapshot for trap locations, including cache hits.
Only the sibling `fairyc-candidate-linux-build`
companion is invoked; no shell, PATH search or Go frontend fallback is used.
The companion's `--run-trap-control` mode selects the existing Linux trap writer
and restricts publication to the private stage's `program` leaf. It retains the
GNU object/link/ELF checks. The client workspace's `build/<name>` is untouched.

`linux_run_cli` launches that absolute image with the client workspace as its
working directory and passes every argument after `--`, including empty and
flag-looking arguments. Stdin and the environment are inherited. Program stdout
and stderr are captured independently as binary bytes, at most 65,536 bytes each.
The program gets a separate process group. The 60,000 ms observation/drain budget
starts after launch; timeout, output overflow and early failures use the existing
owned-group cleanup. This does not bound process creation, scheduling or final reap.
The backend retains its separate ten-minute outer compilation budget.

The existing mode-0700 descriptor-owned stage covers packet, objects and image.
Explicit cleanup runs before reporting an execution result; resource cleanup also
runs on early returns. Cleanup failure returns 6. Filesystem errors may leave a
partial private tree; same-user interference and mount changes retain the build
adapter's documented boundary. No arbitrary workspace tree is recursively deleted.

## Results and trap channel

Normal completion returns the Linux low eight exit bits (0..255), including
129..135 when the private channel is empty. It cannot preserve a full signed i32
process result. No build-success message is added to program output.

A third pipe is inherited only as fd 3. `linux_run_capture` drains at most 276
receipt bytes alongside stdout/stderr, validates kind/value/PID and rejects
malformed, mismatched, oversized or signal-associated receipts. Program output
is never interpreted as control data. This is a compiler protocol with the
executed program, not an authentication boundary against malicious native code.
`native_run_control` shares the existing Windows classification and location
formatting; Windows public wrapper functions keep their existing signatures.
A valid receipt prints `trap: <category>`, optionally the source location from the
compilation snapshot, and returns 5. Unknown or invalid source ranges retain the
category-only report. Signals, timeouts, output limits and launch/control failures
return 6 with a diagnostic; output write failure returns 4. Collected streams are
forwarded before reporting a completed trap/signal/timeout/output-limit result.
Frontend, argument and lock errors keep the existing statuses (3, 2 and 7).

## Evidence

Windows low/optimized POSIX substitutes cover the fd 3 setup, closed standard
handles, empty/exact/oversized receipts and descriptor cleanup. Fairy wrapper
substitutes cover binary streams, ordinary reserved exits, trap records, signals,
timeouts, limits and invalid control/launch results. Shared receipt tests cover
all seven categories, malformed records, UTF-8 and snapshot location formatting.
Full-entry emission targets both Linux GNU and Windows MSVC and rejects Windows
foreign dependencies. Portable dispatch tests cover run selection, private output
selection and eight-bit status conversion.

The ordinary source workspace passes `check` and `check --locked` with 556 sources
and 7,071 MIR functions. The prepared Linux workspace passes both modes with
556 sources and 7,072 functions. The Linux amd64 companion cross-build is an
ELF64 little-endian x86-64 image. `go vet ./...` and `git diff --check` pass.

Linux-native tests are authored for GNU private-image compilation, fd 3 capture,
signals/timeouts/output limits, and full CLI runs in development/release profiles.
CLI cases cover arguments, binary streams, workspace cwd, low-eight-bit exits,
trap source locations, locked loads, invalid flags, frontend/companion failure,
private TMPDIR cleanup and preservation of a prior published image. They are
skipped on the Windows development host; no Linux-native success is claimed.
Cached cases also cover cold/warm development-to-release reuse, binary stderr
after the requested cache status, cache-looking program arguments, warm trap
locations and uncached fallback when the selected directory is missing.
Distribution and Linux self-host evidence remain separate. Go stays the default.

Native [workspace tests](candidate-linux-test-v1.md) reuse the run collector and
trap protocol, adding explicit EOF/inherited/replay stdin and per-test reporting.
