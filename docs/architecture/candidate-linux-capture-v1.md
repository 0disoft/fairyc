# Private Fairy Linux capture transport v1

The first slice in `internal/candidatehost/linux_capture.c` provides only
`fairyc_private_linux_capture_open_v1` and `fairyc_private_linux_capture_drop_v1`.
It owns a child and stdout/stderr pipe readers through one opaque token. There
is no collector, Fairy wrapper, public SDK API or CLI integration in this slice.

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
A future Fairy resource cleanup may discard that error, so this is not a promise
of infallible child cleanup.

`LinuxSource` selects process C, then capture C, then optional path C. This keeps
`_GNU_SOURCE` before system headers and makes the process static helpers available
to capture in the same translation unit. The Linux toolchain baseline is unchanged.

## Evidence and pending work

`TestLinuxCaptureAdapterBoundaries` passes Windows MSVC POSIX substitutes in low
and optimized profiles for normal/closed stdio, first/second pipe failure, FD normalization,
get/set flags, both dup2 failures, allocations, existing attribute/spawn failures
and close `EINTR`. Their FD/allocation/action/attribute counters show no leaks;
existing process checks also pass. `go vet -mod=readonly` passes with exit 0.
These are substitute results, not native Linux proof. The authored native Linux
GCC/Clang pipe-ownership/drop probe is skipped on Windows.

Bounded collection, capture timeouts, output limits and the Fairy wrapper remain
pending for the next implementation slice. No read/drain behavior or capture
deadline is admitted here; blocking writers can fill the pipes until a collector
exists. The filesystem/CLI bridge and native Linux execution proof remain open.
