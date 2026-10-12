# Private Fairy Windows process host v1

`windows_process.fy` owns command encoding, process flags, Job Object setup,
launch/resume/wait order, deadline admission and resource-handle cleanup. The
private `internal/candidatehost/windows_process.c` adapters forward Win32 calls;
`runtime.go` selects their embedded source when registered private symbols occur
in a candidate header. Neither adapter implements the language frontend.

The API requires an executable, argument buffers, a working directory and a
deadline of 1..600000 milliseconds. The executable is passed explicitly to
CreateProcessW. Commands use mutable owned UTF-16; the process starts suspended
and hidden, enters a kill-on-close Job Object, then resumes. Parent standard
handles are inherited. Timeout terminates the job and waits for the parent;
thread/process/job handles are owned resources. Exact stdin/stdout/stderr byte
forwarding passes in development and release: the test child receives seven
stdin bytes including NUL and emits exactly the expected 13-byte stdout and
13-byte stderr. Its full exit code 41 reaches the Fairy caller.

The Windows x64 startup layout is 104 bytes. Its two adjacent WORD fields are
represented by one zero u32. Process information retains two handles and two
u32 IDs. The 144-byte extended job-limit record is zeroed with
JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE at byte offset 16. Native sizeof adapters check
the startup/job sizes before launch. These are private host ABI details; public
SDK ABI and source-language contracts are unchanged.

`Outcome` retains the full u32 exit bits and PID. `Problem` retains kind,
native error and any created PID: 1 input, 2 allocation, 3 ABI, 4 create,
5 job setup/assignment, 6 resume, 7 wait/result, 8 deadline. This does not yet map
to stable build/run status or language-trap receipts. Command encoding follows
the Go Windows host's `syscall.EscapeArg`: empty or space/tab-containing values
are quoted; backslashes before quotes and quoted trailing backslashes double.
Executable names containing quotes reject. The provisional 1024-argument cap
is removed. Admission uses the actual 32767 UTF-16-word limit including NUL;
16382 nonempty one-word arguments can fit with a one-word executable, while
16383 cannot. No successful partial command is published on overflow.

Seventy exact UTF-16 command lines from a Go standard-library oracle cover
ordinary, space/tab, Korean and trailing-backslash executable paths, empty and
combined arguments, quotes, backslashes, newlines and supplementary Unicode.
Three authored Fairy tests also cover invalid UTF-8/NUL and exact length/count
boundaries in Go-native and candidate C development/release.

Three Fairy tests run under candidate C development/release with a Go test child.
They verify special arguments, full exit bits, repeated success/failure handle
counts and a child observed live through its native handle before PID publication.
The deadline case then checks native wait status for both parent and child.
An initial absolute handle-count comparison changed from 49 to 93 once and then
stayed at 93 across four further failures. Repeated failure growth is therefore
measured against its own warmed baseline; the one-time increase is not attributed.
Both ordinary and Korean/space-containing fixture paths pass.

The Go stable native test runner cannot link these private adapters. The private
verifier's explicit `--candidate-only` records zero Go-native tests rather than
claiming a comparison. Go source checking, adapter selection and existing private
bridge deadline/argv checks pass. An ordinary workspace packet built by the new
bridge has exactly the previously verified development image hash, confirming
that unused private adapters do not alter that artifact. Stream fixtures are
checksum-checked by the private verifier and only supplied to native executions.
Linux, broad frontend
differential, the full suite, public build/run and promotion remain pending.

Receipt: [candidate-windows-process-20261012.json](../ops/evidence/candidate-windows-process-20261012.json).
Stream receipt: [candidate-windows-streams-20261012.json](../ops/evidence/candidate-windows-streams-20261012.json).
Argument receipt: [candidate-windows-argv-20261012.json](../ops/evidence/candidate-windows-argv-20261012.json).
Native records and flags follow [STARTUPINFOW](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/ns-processthreadsapi-startupinfow),
[extended job limits](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information),
[CreateProcessW](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw)
and [Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects).
