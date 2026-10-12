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
to the public build/run command yet. Command encoding follows
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

## Private run control channel

`windows_run_control.fy` owns a separate pipe, a copied Unicode environment
block and receipt classification. Only the writer is marked inheritable. The
parent environment is never changed; ordinal case-insensitive name comparison
replaces any inherited `FAIRYC_TRAP_CONTROL_HANDLE_V1`, retaining other entries
and their order. The private snapshot/output allowance is one MiB. The new
`windows_process::run_environment` passes that block with
CREATE_UNICODE_ENVIRONMENT through a thin CreateProcessW adapter.

After the parent exits and its job closes, the caller inspects the pipe without
waiting for EOF. Zero bytes preserve every u32 exit status, including 129..135
and 0xffffffff. Exactly one matching byte recognizes a Fairy trap and maps it
to status 5. Mismatched or multiple bytes reject as private problem 10; pipe and
environment failures use problem 9. Existing process problems 1..8 are retained.

The private C emitter's existing tags are overflow 1, division 2, signed
division 3, shift 4, assertion 5, bounds 6 and unreachable 7. Fairy normalizes
authenticated tags to the public category numbering; stdout/stderr never act
as control data. The private bridge's explicit `--run-trap-control` selects
the existing receipt runtime at exit base 128. Its default stays at base 100.

Four authored tests pass in both candidate C profiles on the Korean/space path.
They cover all seven tags, malformed receipts, exact environment reconstruction,
a real candidate-generated assertion child and an ordinary return, stable
handle counts, parent environment preservation and binary stdin/stdout/stderr.
The three existing process tests also pass in both profiles, including actual
parent/child deadline cancellation. Go-native remains skipped for the private
adapter linkage. The verifier now derives wrapper effects from selected tests.

One earlier post-normalization run failed an unlabelled assertion. That exact
binary subsequently passed unchanged; the transient failure is not attributed.
The final source uses labelled failure assertions and passes both profiles.

## Owned running process

`windows_process::start` publishes `Running` only after job assignment and
resume. It owns the job, process and thread handles and accepts explicit standard
handles for the upcoming bounded-output collector. `poll` reports pending or
completed status; `result` rejects a still-running process and retains all exit
bits after completion. An early return drops the owned job and terminates the
child. Existing blocking APIs use the same owner and retain their deadlines.

Four process tests and all four control-channel tests pass in both candidate C
profiles. The new case observes a live child, rejects premature collection and
checks actual exit after early return, including four repeats without handle
growth. Source check passes 369 files / 4736 MIR functions. Bounded combined
stdout/stderr capture is the next public-run prerequisite.

## Bounded program output

`windows_run_capture.fy` drains stdout, stderr and the private control pipe
round by round while polling the owned process. It inherits stdin directly.
The shared stdout/stderr retention limit is exactly 65536 bytes. Excess bytes
are drained and discarded until completion, preserving the existing runner's
write behavior; overflow returns private problem 11 without publishing buffers.
The control pipe retains at most two bytes while draining any excess, then
validates the receipt. The whole execution has a monotonic deadline, with a
bounded final collection period. Job termination retains the recorded exit bits.

Five authored tests pass in both candidate C profiles: exact combined capacity,
one-byte overflow with proof that the child finished its writes, an 8192-byte
control flood without pipe deadlock, real Fairy assertion/ordinary exits, full
exit bits, live parent/child deadline cancellation and exact binary streams.
Ten standalone C executions verify the signed i32 main ABI for five normal/raw
statuses in both profiles, including the high bit and 0xffffffff. Source check
passes 370 files / 4748 MIR functions; private host selection passes. Go-native
is skipped for private linkage. Public build/run wiring remains pending.

## Native command options

`native_cli_options.fy` implements the existing build/run argument contract:
only optional `--locked`/`--release` before the first run delimiter, duplicate
rejection, and verbatim owned arguments after it. NUL rejection precedes compiler
flag parsing, matching Go's split-before-parse order. Diagnostics retain the
existing command/flag messages. The process reader retries using the SDK's exact
needed size instead of imposing a fixed path/argument buffer limit.

Three authored tests pass in Go-native and both candidate C profiles. A direct
`from_process` probe preserves both flags, empty/Unicode values and an 8192-byte
argument in both profiles. Source check passes 371 files / 4758 MIR functions.
The dispatcher does not expose build/run yet: native driver transport and
publication are still required.

## Fresh native driver input

The private bridge accepts `--packet-input` for a freshly serialized C packet
from the calling Fairy compiler. It reads a regular file into one bounded
snapshot, validates the existing framing/parts contract and invokes MSVC without
starting any frontend. The cache-replay `--packet` mode still requires its exact
SHA256. Input/cache/source modes reject mixing before reading or publication.

Fresh input and checksum-pinned replay produce identical complete images in
development and release from a real Fairy packet; all four images return 134.
Focused tests reject malformed/nonfile/mixed inputs, preserve the prior image
on rejection and execute successful native builds. Public build/run wiring
still requires Fairy packet-file serialization and workspace publication.

`native_packet.fy` now writes the same FYCPARTS1 framing directly to a
caller-owned file. It computes the complete bounded size before the first write,
streams checked header/literals/units and reports exact write progress. The
caller must sync and close successful output before native compilation. No
second whole-packet buffer or frontend subprocess is required.

Two authored tests pass in Go-native and both candidate C profiles. The 3387-byte
file equals existing stdout transport and the Go-native output exactly; the
fresh driver builds that file into programs returning 42 in both profiles.
Read-only write failure reports zero progress and retains prior file contents.
Source check passes 372 files / 4766 MIR functions. Workspace metadata,
temporary-file lifetime and public build/run dispatch remain pending.

Receipt: [candidate-windows-process-20261012.json](../ops/evidence/candidate-windows-process-20261012.json).
Stream receipt: [candidate-windows-streams-20261012.json](../ops/evidence/candidate-windows-streams-20261012.json).
Argument receipt: [candidate-windows-argv-20261012.json](../ops/evidence/candidate-windows-argv-20261012.json).
Control receipt: [candidate-windows-run-control-20261012.json](../ops/evidence/candidate-windows-run-control-20261012.json).
Running receipt: [candidate-running-owner-20261012.json](../ops/evidence/candidate-running-owner-20261012.json).
Capture receipt: [candidate-windows-run-capture-20261012.json](../ops/evidence/candidate-windows-run-capture-20261012.json).
Options receipt: [candidate-native-cli-options-20261012.json](../ops/evidence/candidate-native-cli-options-20261012.json).
Driver receipt: [candidate-native-packet-input-20261012.json](../ops/evidence/candidate-native-packet-input-20261012.json).
Packet-file receipt: [candidate-native-packet-file-20261012.json](../ops/evidence/candidate-native-packet-file-20261012.json).

## Workspace native plan

`project_emit::evaluate` returns an owned plan containing the checked C bundle,
root package name and lock state from the same load. Source/graph owners are
released after emission; failure diagnostics keep their existing owners. The
stdout packet command borrows the plan's bundle, preserving the transport.

Two authored tests pass in Go-native and candidate C development/release:
updated/unchanged/verified locks retain the root name and usable bundle, and
missing locked input plus invalid entries retain diagnostics and exit codes.
The initial missing-lock test expected 2; the existing contract returns 7, so
only that expectation was corrected. Source check passes 372 files / 4769 MIR
functions. Public build/run remains pending.
Plan receipt: [candidate-native-plan-20261012.json](../ops/evidence/candidate-native-plan-20261012.json).

## Owned native scratch directory

`native_scratch.fy` creates one exclusive directory under a caller-supplied host
temporary root. Existing file/directory names select another bounded slot and
never confer cleanup ownership. Automatic resource cleanup removes the created
tree on early return. Explicit close consumes ownership once and returns exact
recursive removal progress or the SDK error; it does not silently retry.

Three authored tests pass in Go-native and candidate C development/release:
collision contents survive early-return cleanup, explicit close removes two
files plus the owned directory, and invalid/missing-parent admission leaves no
directory. Candidate C initially rejected SDK create/remove-tree calls. Its
authenticated file bridge now admits existing rules 36 and 39 with the existing
file-result carrier. A narrow Fairy C emission executable was rebuilt by the
previous Fairy compiler without a Go frontend; it emitted the tested packet.
Unchanged successful Go-native tests were reused after this C-only admission fix.
Source check passes 373 files / 4775 MIR functions. Host temporary-root discovery
and public build/run dispatch remain pending; public SDK ABI is unchanged.
Scratch receipt: [candidate-native-scratch-20261012.json](../ops/evidence/candidate-native-scratch-20261012.json).

## Native host paths

`windows_host_paths.fy` queries the current directory, host temporary root and
compiler image through three direct OS adapters. Fairy validates bounded UTF-16,
rejects malformed surrogates/NULs and owns the resulting UTF-8 paths. The private
driver is selected beside the compiler image as `fairyc-candidate-build.exe`.
This introduces no workspace configuration or environment override.

Two authored tests pass in candidate C development/release: exact Unicode
conversion/malformed rejection, native current/temp paths against an OS oracle,
image path against argv and companion selection. Go-native is skipped because
its runtime does not link private host adapters. The host symbol-selection Go
test passes. Source check passes 374 files / 4786 MIR functions. Public build/run
dispatch remains pending; public SDK ABI is unchanged.
Path receipt: [candidate-windows-host-paths-20261012.json](../ops/evidence/candidate-windows-host-paths-20261012.json).

## Candidate build/run dispatcher

The candidate dispatcher now exposes `build [--release] [--locked]` and
`run [--release] [--locked] [-- <arguments>]`. Fairy owns option parsing,
workspace/lock admission, entry selection, C emission, packet-file sync/close,
output-path selection, driver invocation, scratch lifetime and run classification.
The adjacent private Go driver consumes only the fresh packet and performs MSVC
transport plus atomic image publication. Go compiler/frontend code is unchanged.

Build selects `build/<root-name>.exe` and the existing target/profile/lock success
line. The private `--output-root` transport publishes the relative path through
the caller's opened source root, rejecting a directory escape and retaining the
prior image on failure. Run uses a fresh scratch directory as cwd, leaves stdin
for the child, forwards captured streams, preserves all ordinary 32-bit exit
bits and emits recognized Fairy traps with code 5. MSVC commands and program
execution retain the existing 60-second limit; the outer driver job is bounded
to ten minutes across compilation/link commands.

The 369-source dispatcher was emitted and built through Fairy C in development
and release. Across both profiles, 31 completed invocations verify default and
locked/release builds, artifact execution, duplicate/invalid flags, missing
driver, stale/missing lock, directory escapes, exact empty/Unicode/quoted/8192-byte
argv and binary stdin/streams, ordinary status 134 and 0xffffffff, assertion trap,
invalid entry and output overflow without partial stdout or scratch leaks. One
Go reference run matches the complete program streams. Focused publication tests
cover write/sync/close/rename failure preservation and opened-root escape.
Source check passes 376 files / 4798 MIR functions.

The shipped Go CLI remains unchanged after this candidate command integration. The
current full two-generation image fixed point, full frontend/executable corpus,
Linux/hosted CI and detailed allocation/cleanup failure parity remain unverified.
The initial I/O fixture exceeded the candidate's 4096-element fixed-array bound;
it was changed to the public Buffer while retaining the full 8192-byte argument.
An overly narrow verifier prefix check was also corrected for source-positioned
entry diagnostics. No SDK ABI, DB or CI change; no push.
CLI receipt: [candidate-public-native-cli-20261012.json](../ops/evidence/candidate-public-native-cli-20261012.json).

## Current native dispatcher fixed point

At snapshot `1ce8d6b9`, consecutive complete 369-source C-part generations emit
the same 29998661 bytes and produce identical entire 4257280-byte PE images,
SHA256 `e4f84720790ef4d2150635578178fb299408988e0fe0fcbfea0e4a0d1c7b6cdd`.
The previously successful first-generation build is reused only after matching
source inventory, packet, image and driver hashes. The next generation is newly
emitted/built and passes 249 Go lexer cases/framing plus 15 native CLI invocations.

A 75-case semantic/resource workspace-check probe then finds 11 status and 35
first-diagnostic differences against a freshly built Go oracle. The candidate
accepts eight invalid loop/borrow cases and rejects three valid feature cases;
several source errors also lose their semantic codes behind subset messages.
These gaps must be resolved before promotion. Full semantic fact tables, Linux,
hosted CI and cost/package gates remain unverified. No shipped compiler change
or push. Receipt: [candidate-public-native-fixed-point-20261012.json](../ops/evidence/candidate-public-native-fixed-point-20261012.json).
Native records and flags follow [STARTUPINFOW](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/ns-processthreadsapi-startupinfow),
[extended job limits](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information),
[CreateProcessW](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw)
and [Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects).
