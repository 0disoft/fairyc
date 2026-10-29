# Private Fairy Linux capture transport v1

`examples/lexer_candidate/src/linux_capture.fy` provides private bounded binary
capture through `run_for(program, arguments, directory, timeout_ms, output_limit,
error_limit)`. `internal/candidatehost/linux_capture.c` supplies opaque-session
open, one-shot collect and drop ABI functions. This adds no public SDK API or CLI
dispatch; the existing process `run`/`run_for` still inherit standard streams.

## Fairy inputs and owned output

The wrapper accepts 1..86,400,000 ms and separate stdout/stderr requested limits
whose sum is at most 1,048,576 bytes. Either limit may be zero. It validates these
budgets before allocation or open, then prepares the existing packed argv/cwd.
It owns SDK `bytes::Buffer` values, uses `resize_zeroed` before passing raw mutable
views to collect, checks returned lengths and truncates the buffers to those
lengths. Allocator rounding/metadata, argv/cwd storage and kernel pipe buffers
are outside the requested output budget.

`Captured` owns both buffers and exposes `kind`, `value`, `pid`, `stdout` and
`stderr` accessors. Output is binary: NUL and 0xFF survive without UTF-8 parsing.
The private `Session` resource owns all native handles and the child and drops
before `run_for` returns, including error returns; `Captured` retains only data
and the original PID as outcome information.

`Problem.kind` is 1 for invalid input, 2 for a requested bound, 3 for Fairy-side
allocation, 4 for open, 5 for collect and 6 for an ABI mismatch. Open/collect
errors retain the native code in `os_code`. A null successful session, invalid
native outcome/PID or lengths beyond the requested limits is an ABI problem.

## Launch and pipe ownership

`open` accepts the existing packed argv and cwd transport from the
[process runner](candidate-linux-process-v1.md). It clears the output token before
work and publishes it only on success. Two `pipe2(..., O_CLOEXEC)` calls create
separate stdout and stderr pipes; see
[pipe(2)](https://man7.org/linux/man-pages/man2/pipe.2.html).
Any pipe FD in 0..2 is moved to an FD >= 3 with `F_DUPFD_CLOEXEC`, so closed
standard streams cannot alias the child redirection sources. Only the parent
read ends receive `O_NONBLOCK`; child writes remain blocking.

The shared `fy_linux_process_spawn` helper adds stdout/stderr `dup2` actions
before `addclosefrom_np(..., 3)` closes the remaining child descriptors. See
[posix_spawn_file_actions_adddup2(3p)](https://man7.org/linux/man-pages/man3/posix_spawn_file_actions_adddup2.3p.html).
The existing own-process-group policy applies. Stdin and the environment remain
inherited; launch uses neither a shell nor PATH lookup. Normal process start
passes no redirection FDs and keeps its existing behavior.

The parent closes both writers immediately after the spawn attempt. On success,
the token alone owns the child PID and both readers. Failed setup closes acquired
pipe ends and frees the token; if a child was started, it attempts existing
process discard before returning the setup or writer-close error.

## Collection, outcomes and deadline

Collect is one-shot after valid arguments are accepted, including when subsequent
collection fails. Each round attempts at most one nonblocking read of up to
4,096 bytes per stream, so continuous stdout cannot indefinitely starve stderr
or the clock. Read `EINTR`/`EAGAIN` and interrupted polls return to the loop
without resetting the absolute `CLOCK_MONOTONIC` deadline. Both readers enter
[poll(2)](https://man7.org/linux/man-pages/man2/poll.2.html) with a requested wait
of at most 10 ms, capped by the remaining budget rounded up to milliseconds.
Clock, read, poll and process-cleanup errors are returned as collect failures.

Filling a buffer exactly is not truncation. Once full, the next read requests one
extra byte into stack storage: EOF permits the ordinary outcome; an extra byte
selects output-limit kind 4, value 1 for stdout or 2 for stderr. The retained
prefix stays within capacity, including zero-capacity streams. Limit detection
stops the owned process group/direct child and reaps the child before success.

| Kind | Value | Meaning |
| --- | --- | --- |
| 1 | 0..255 | Ordinary exit, including genuine exit 127 |
| 2 | 1..64 | Terminating signal |
| 3 | 0 | Timeout during exit observation or pipe drain |
| 4 | 1 / 2 | Stdout / stderr limit exceeded; retained prefixes available |

The budget starts in collect after open and covers both exit observation and
pipe drain. Creation, final cleanup/reap and scheduling are not bounded; this is
not a hard deadline for the entire call. On leader exit, existing group/leader
cleanup precedes final reap. Successful reap clears the token PID to 0, then
bounded draining continues until both streams reach EOF. If pipes remain held
open until expiry, collect returns timeout without signaling the old PID again,
avoiding PID reuse. A collected ordinary exit/signal with both EOFs wins over
timeout at that round's deadline check.

## Drop and failure boundary

`drop(NULL)` succeeds. Dropping a live token calls the existing process discard,
including its interrupted wait retries and same-group/leader cleanup, then closes
both readers and frees the token even if child cleanup fails. The returned code
prefers the child-cleanup error, then the first reader-close error. The token is
consumed on every drop and must not be reused. Exclusive, waitable child ownership
and escaped-descendant limits are the same as for the process runner.

Each close invalidates the stored FD before calling Linux `close` once. It never
retries `EINTR`, avoiding a second close of a potentially reused descriptor.
Cleanup failures are returned; termination/reaping are not guaranteed on failure.
Fairy's `Session` cleanup discards the drop error. A successfully reaped session
has PID 0, so later drop only closes remaining readers and frees the token.

`LinuxSource` selects process C, then capture C, then optional path C. This keeps
`_GNU_SOURCE` before system headers and makes the process static helpers available
to capture in the same translation unit. The Linux toolchain baseline is unchanged.

## Evidence and remaining work

`TestLinuxCaptureAdapterBoundaries` passes Windows MSVC POSIX substitutes in low
and optimized profiles for normal/closed stdio, first/second pipe failure, FD normalization,
get/set flags, both dup2 failures, allocations, existing attribute/spawn failures
and close `EINTR`. Their FD/allocation/action/attribute counters show no leaks;
existing process regressions also pass. Collector checks cover exact/zero limits,
both stream overflows, binary output, 9,000-byte cross-stream fairness, continuous
output deadlines, held pipes after leader reap without a second signal, clock/read/
poll errors, interrupt storms, one-shot rejection, invalid capacities and canaries.

`TestLinuxCapturePolicy` passes low/optimized Fairy policy execution and both-target emission:
problem/outcome classification, invalid budgets before open, null session,
invalid counts/kind/PID, drop-once ownership, NUL/0xFF prefixes and zero output.
The final C capture/process adapter retest and CLI capture/process/argv checks
pass in both profiles, including both-target emission. `go vet -mod=readonly`
passes with exit 0. Candidate `check` and `check --locked` pass with 543 sources
and 6,935 MIR functions; the lock is updated and verified.

The authored native Linux GCC/Clang capture probe covers exact 5-byte binary
stdout/3-byte stderr with exit 127, stderr-first flooding of 131,072 bytes per
stream, overflow, a silent 25 ms timeout and zero-output/zero-capacity execution.
Subreaper cases cover ordinary leader exit 17 with a same-group grandchild and
an escaped `setsid` grandchild holding pipes until the 25 ms drain timeout without
a second PID signal. Grandchild readiness is handshaken before collect. The test
explicitly owns, kills and reaps the escaped grandchild after capture; this does
not extend the adapter's containment guarantee.

The native probe is skipped on Windows: no native Linux compilation or execution
is established by the substitute/emission checks. The bounded I/O primitive is
implemented. Native Linux proof and the filesystem/CLI dispatch bridge remain open.
