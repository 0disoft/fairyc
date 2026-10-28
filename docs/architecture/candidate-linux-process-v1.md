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
and initialized file actions on every return path.

`posix_spawn` changes cwd in the child and closes descriptors >= 3 through
`posix_spawn_file_actions_addchdir_np` and `addclosefrom_np`. The child inherits
the environment and standard streams. No shell, quoting or PATH search is used;
relative paths remain relative and are resolved by the native launch.

## Problems, outcomes and cleanup

`Problem.kind` is 1 for invalid input, 2 for the private bound, 3 for Fairy-side
encoding/path allocation, 4 for start, 5 for wait and 6 for an ABI mismatch.
Start/wait failures retain the native error in `os_code`; native pointer-table
allocation failure is a start problem with `ENOMEM`. Input problems retain the
command encoder's argument index and byte offset.

`Outcome` retains the PID, kind and value. Kind 1 means ordinary exit 0..255,
including a genuine exit 127; kind 2 means termination by signal (accepted values
1..64). Start errors remain distinct from exit status. Blocking `waitpid` retries
`EINTR`; invalid PID/outcome values become ABI problems.

`Running` owns the child until successful wait clears its PID. Cleanup first
tries `waitpid(WNOHANG)`, then sends `SIGKILL` and reaps a still-running owned
child, retrying interrupted waits. `ECHILD` never authorizes a kill. Cleanup can
fail, and Fairy discards its error; child termination/reaping is not guaranteed.
Only the owning call may reap this PID. Process groups and descendants are outside
this slice.

## Platform and verification boundary

The GNU/glibc extensions use the existing [Linux toolchain baseline](../abi/linux-toolchain-v0.md):
Ubuntu 24.04 amd64, glibc >= 2.39, GCC >= 13 or Clang >= 18. When combining C
sources, `_GNU_SOURCE` must precede every system header; the private source
selector returns process C before path C to preserve that ordering.

Completed Windows checks cover MSVC low/optimized adapter execution with POSIX
substitutes, authored Fairy process-policy tests with ABI stubs, argv tests, and
Linux and Windows C emission. Candidate `check` and `check --locked` passed with
542 files and 6,924 MIR functions; `go vet`, formatting and diff checks also passed.
The authored native Linux GCC/Clang probe was skipped on Windows: these results
do not prove native Linux process behavior. No public SDK, CLI dispatch or default
compiler changes are included. Deadlines, process-group/tree cleanup, bounded I/O
and the filesystem/CLI bridge remain future work.
