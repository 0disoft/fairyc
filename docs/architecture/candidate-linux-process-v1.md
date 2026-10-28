# Private Fairy Linux process transport v1

`examples/lexer_candidate/src/linux_process.fy` implements a private blocking
runner for one Linux amd64 child. Fairy owns input preparation, start/wait
sequencing and the `Running` resource; `internal/candidatehost/linux_process.c`
provides the three private start, wait and discard ABI functions.

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
1..64). Start errors remain distinct from exit status. Wait first observes exit
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

## Platform and verification boundary

The GNU/glibc extensions use the unchanged [Linux toolchain baseline](../abi/linux-toolchain-v0.md):
Ubuntu 24.04 amd64, glibc >= 2.39, GCC >= 13 or Clang >= 18. When combining C
sources, `_GNU_SOURCE` must precede every system header; the private source
selector returns process C before path C to preserve that ordering.

Current Windows checks pass MSVC low/optimized adapter execution with updated
POSIX substitutes: spawn-attribute failure cases 17–19, exact
`WNOWAIT` -> negative group kill -> positive leader kill -> reap ordering,
`ECHILD` without signaling, and PID 1 rejection. CLI process-policy tests with
ABI stubs and argv tests pass in both profiles, as do Linux and Windows C
emission. Candidate `check` and `check --locked` pass with 542 files and 6,924
MIR functions; `go vet`, `gofmt` and diff checks also pass.

The authored native Linux GCC/Clang probe also uses a subreaper to observe a
leader exiting normally with status 17 while leaving a grandchild. It expects
the grandchild to terminate by `SIGKILL` and reaps it, with a three-second alarm
guarding failure. This probe is skipped on this Windows host; Windows substitutes
do not prove native Linux process behavior. This slice adds no new
`_GNU_SOURCE` requirement: its existing placement before system headers remains.
No public SDK, CLI dispatch or default compiler changes are included. Deadlines
are a separate next slice; bounded I/O and the filesystem/CLI bridge remain
future work. Containment of descendants that escape the group remains outside
this runner's guarantee.
