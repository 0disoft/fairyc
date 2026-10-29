# Private Fairy Linux process transport v1

`examples/lexer_candidate/src/linux_process.fy` implements private blocking and
timed runners for one Linux amd64 child. Fairy owns input preparation, start/wait
sequencing and the `Running` resource; `internal/candidatehost/linux_process.c`
provides the private start, wait, wait-for and discard ABI functions.

## Launch and ownership

The runner uses the owned UTF-8 packed argv from
[linux_command](candidate-linux-command-v1.md), including an explicit count and
`argv[0]` equal to the program path. Empty and trailing empty arguments survive.
The 1 MiB logical budget charges string bytes including NULs plus
`8 * (argc + 1)` pointer bytes. The native adapter separately allocates the
null-terminated pointer table; cwd storage and allocation metadata are outside
that budget. OS argument/environment limits can still reject a launch.

Fairy rejects malformed UTF-8, embedded NULs and empty program/cwd paths; paths
are limited to 4095 bytes before the terminator. The C adapter rechecks byte/count
agreement, terminators and bounds before launch. It releases its pointer table
and initialized file actions and spawn attributes on every return path.

`posix_spawn` changes cwd in the child and closes descriptors >= 3 through
`posix_spawn_file_actions_addchdir_np` and `addclosefrom_np`. The child inherits
the environment and standard streams. No shell, quoting or PATH search is used;
relative paths remain relative and are resolved by the native launch.

The [private bounded capture runner](candidate-linux-capture-v1.md) reuses the
spawn helper with stdout/stderr pipe redirection and an opaque owned session.
Its separate Fairy wrapper adds binary output limits and a post-open observation/
drain budget. This process module's `run`/`run_for` still inherit standard streams.

`posix_spawnattr_setpgroup(..., 0)` with `POSIX_SPAWN_SETPGROUP` gives each
child its own process group, initially `pgid == pid`; see
[posix_spawn(3)](https://man7.org/linux/man-pages/man3/posix_spawn.3.html).
The owning call alone may reap this child; no other thread or signal handler may
reap it. Other actors reaping it violate ownership. The host must keep the child
waitable: it must not explicitly set `SIGCHLD` to `SIG_IGN` or use `SA_NOCLDWAIT`.
These are caller requirements, not signal dispositions installed by the adapter.

## Problems, outcomes and cleanup

`Problem.kind` is 1 for invalid input, 2 for the private bound, 3 for Fairy-side
encoding/path allocation, 4 for start, 5 for wait and 6 for an ABI mismatch.
Start/wait failures retain the native error in `os_code`; native pointer-table
allocation failure is a start problem with `ENOMEM`. Input problems retain the
command encoder's argument index and byte offset.

`Outcome` retains the PID, kind and value. Kind 1 means ordinary exit 0..255,
including a genuine exit 127; kind 2 means termination by signal (accepted values
1..64); kind 3 with value 0 means timeout and is accepted only by `run_for`.
Start errors remain distinct from exit status. Blocking wait first observes exit
with `waitid(P_PID, pid, ..., WEXITED | WNOWAIT)`, leaving the leader unreaped.
It then sends `SIGKILL` to the original group (`-pid`) and separately to the
leader (`pid`), which may have left that group, before `waitpid(pid, ..., 0)`
reaps the leader. Interrupted observations and final waits retry `EINTR`.
The final wait preserves the leader's already-observed original exit status;
same-group descendants receive cleanup even after an ordinary leader exit.
See [wait(2)](https://man7.org/linux/man-pages/man2/waitpid.2.html) for `WNOWAIT`
and waitability requirements. Invalid PID/outcome values become ABI problems;
PID 1 is explicitly rejected so group signaling can never become `kill(-1, ...)`.

`Running` owns the child until successful wait clears its PID. Cleanup first
queries ownership with `waitid(P_PID, pid, ..., WEXITED | WNOWAIT | WNOHANG)`.
On success, whether the child is running or already exited, it uses the same
group/leader signaling and final reap sequence. `ECHILD` at this ownership query
returns without signaling. Keeping the leader unreaped until signaling prevents
PID/PGID reuse under the exclusive-ownership and waitable-child requirements.
An already-cleared PID 0 needs no cleanup.

The adapter propagates observation, signaling (except `ESRCH`) and final-wait
errors; discard treats the initial `ECHILD` as no remaining owned child.
Fairy's resource cleanup still discards the discard error, so termination and
reaping are not guaranteed when cleanup fails. Descendants that escape the
original process group or session are not contained. This is same-group cleanup,
not an arbitrary process-tree guarantee, and only the direct child is reaped.

## Timed exit observation

`run` remains blocking. `run_for(program, arguments, directory, timeout_ms)`
accepts 1..86,400,000 ms and rejects other budgets as input problems before
allocation or launch. It uses `fairyc_private_linux_process_wait_for_v1` after
spawn returns. The budget covers post-start exit observation; launch, cleanup,
final reaping and scheduling delays are not bounded by it. This is not a hard
deadline for the entire call.

The adapter computes one absolute nanosecond deadline from `CLOCK_MONOTONIC`.
Each poll queries `waitid` with `WEXITED | WNOHANG | WNOWAIT`, reserving the
leader until cleanup. Completion observed at a poll wins even at the deadline
and preserves ordinary exit/signal results. If that poll still finds a running
child at the deadline, the adapter kills the original group and the leader,
reaps the direct child, and returns timeout kind 3/value 0 on success.
Between polls, absolute `clock_nanosleep(CLOCK_MONOTONIC, TIMER_ABSTIME, ...)`
targets advance by at most 10 ms, capped at the deadline. `EINTR` retries do not
reset that deadline. See
[clock_nanosleep(2)](https://man7.org/linux/man-pages/man2/clock_nanosleep.2.html).

Clock, sleep, query, kill and reap failures become wait problems (kind 5) with
the native code in `os_code`; `clock_nanosleep` supplies its return code directly,
not `errno`. Clock overflow is `EOVERFLOW`; a reading before the start is `EIO`.
During continuous `waitid` interruptions, each iteration checks the clock and
returns `EINTR` at the deadline rather than signaling without a successful
ownership query. Resource cleanup can then take additional time and still has
the same failure and escaped-descendant limits.

## Platform and verification boundary

The GNU/glibc extensions use the unchanged [Linux toolchain baseline](../abi/linux-toolchain-v0.md):
Ubuntu 24.04 amd64, glibc >= 2.39, GCC >= 13 or Clang >= 18. When combining C
sources, `_GNU_SOURCE` must precede every system header; the private source
selector returns process C, then capture C when selected, then optional path C
to preserve that ordering and share the process static helpers.

Process-group ownership was committed in `cadce568`, with low/optimized POSIX
substitute checks for attribute failures, observe/signal/reap ordering, `ECHILD`
without signaling and PID 1 rejection. Current timed-wait POSIX substitutes pass
in both profiles: 25 ms across a second boundary, interrupted sleeps without
budget reset, early exit, initial/mid-wait clock failures, preserved sleep errors,
`ECHILD` without signaling, interrupt-storm budget, invalid budget/PID,
clock overflow/backward readings and kill `EPERM`.

`TestLinuxCommandTransport` and `TestLinuxProcessPolicy` pass in low/optimized
profiles with both-target C emission. Policy checks cover timeout kind 3,
native-error/ABI classification, invalid budgets without start, ordinary exit 127,
signal 9 and rejection of fabricated timeout kind 3 from unlimited `run` as ABI
problem kind 6. The earlier process-only snapshot passed candidate `check` and
`check --locked` with 542 files and 6,926 MIR functions. The current capture
integration passes both checks with 543 sources and 6,935 MIR functions; the lock
is updated and verified, and `go vet -mod=readonly` passes with exit 0. Final
capture/process C adapter and CLI capture/process/argv checks pass in low/optimized
profiles with both-target emission. This does not establish native Linux execution.

The authored native Linux GCC/Clang probe also uses a subreaper to observe a
leader exiting normally with status 17 while leaving a grandchild. It expects
the grandchild to terminate by `SIGKILL` and reaps it, with a three-second alarm
guarding failure. The timed probe adds a slow process tree, waits for grandchild
readiness before the 25 ms timed wait, then expects `SIGKILL` and reaps the
grandchild. Native probes are skipped on this Windows host; Windows substitutes
do not prove native Linux process behavior. This slice adds no new
`_GNU_SOURCE` requirement: its existing placement before system headers remains.
No public SDK, CLI dispatch or default compiler changes are included. The
[bounded I/O primitive](candidate-linux-capture-v1.md) is implemented separately;
the filesystem/CLI bridge and native Linux proof remain open. Containment
of descendants that escape the group remains outside this runner's guarantee.
