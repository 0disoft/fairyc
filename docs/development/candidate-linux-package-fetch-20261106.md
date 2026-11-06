# Linux Fairy package fetch, 2026-11-06

Part of `implementation-queue-20261105.md`. This receipt does not complete
Linux fetch, broader source parity, deferred designs or final acceptance.
Native Linux execution remains batched at the maintainer's request. No push,
hosted dispatch, release or deployment is authorized by this work.

## Retained-directory native prerequisite

`internal/candidatehost/linux_package_directory.c` supplies four private
initial directory operations selected by `LinuxSource` after rooted I/O:

- `create`: create an exclusive 0700 child and retain independently opened
  parent/child descriptors. Both descriptors transfer to the caller on success.
- `edit`: open an existing child with the same retained ownership; missing
  entries are not created, and failures never remove an existing entry.
- `publish`: compare the retained child against its nofollow source name,
  then make one `renameat2(RENAME_NOREPLACE)` attempt. Existing destinations,
  including empty directories, are not replaced. No replacing-rename fallback.
- `erase`: compare device/inode/directory kind, then make one empty-directory
  `unlinkat(AT_REMOVEDIR)` attempt. No recursion and no implicit tree clearing.

Names use the existing terminated single-leaf policy: 1..255 content bytes,
no embedded NUL, slash, dot or dot-dot. Both owned descriptors are close-on-exec;
fd zero retains the existing descriptor-plus-one representation. Read-only
open/stat observations may retry EINTR. Namespace mutations and close do not.
Creation errors roll back only the newly created empty name, close retained
descriptors once and return primary and first cleanup errors separately.

The package layer must still own the workspace fetch lock, case/exact-spelling
and archive checks, byte/file/depth budgets, synced content and Fairy resource
lifetime. As with the existing package policy, hostile external namespace
rewrites are unsupported; identity checks are not an atomic protection against
an adversary changing a name between stat and mutation. No power-loss or parent
directory sync guarantee is added. Unsupported no-replace renames fail closed.

## Windows evidence

The independent POSIX substitute executes the actual C adapter under MSVC low
and optimized. It checks successful ownership, fd zero, exact flags/permissions,
255/256-byte name boundaries, malformed names/handles/output pointers,
creation collision, observation retries, one-attempt mutation/close behavior,
primary/rollback/close errors, edit failure preservation, device/inode/type
replacement refusal, no-replace collision/unsupported behavior and nonempty
directory refusal. Source selection includes combined scan/package support and
unknown/Windows-symbol refusal.

Final command:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/candidatehost -run '^(TestPrivateLinux|TestLinux.*AdapterBoundaries)'
go vet -mod=readonly ./internal/candidatehost
git diff --check
```

The final test command exits zero in 8.220s. Ten boundary roots pass, including
the new package-directory low/optimized executions. Existing build-scratch and
workspace-observation roots perform selection assertions, then explicitly skip
their Linux execution portions on Windows. Their skips are not native passes.
Scoped vet, Go formatting and diff checks pass. Local final log:
`.cache/linux-package-directory-20261106.log` (not staged).

The first selected command fails both new C profiles because the test substitute
redefines UCRT `ENOTEMPTY` and `EOPNOTSUPP` under warnings-as-errors. Conditional
definitions correct only that harness issue; subsequent focused and combined
commands pass. Warning levels and product limits are unchanged.

## Fairy retained-directory ownership

`linux_package_directory.fy` now owns the child rooted-I/O capability, retained
parent descriptor and terminated source name. The existing rooted module adds
one unsafe private adoption function: ownership transfers, no descriptor is
duplicated, and the caller must relinquish the adopted handle. Borrowed access
to the child retains the containing resource's lifetime. Automatic resource
cleanup closes both descriptors and releases the retained name; it does not
implicitly remove a tree or promise to report implicit close errors.

Native errors and first cleanup errors remain distinct. Malformed success,
partial-error outputs, duplicate handles and output handles aliasing the
borrowed original root reject; plausible owned outputs close at most once and
the borrowed root is preserved. After successful publication/removal, the old
name loses namespace authority. A failed mutation retains explicit cleanup
authority without retrying the mutation automatically.

`TestLinuxPackageDirectoryPolicy` compiles the authored Fairy module through
the Go reference frontend, emits both Linux/Windows C profiles and executes
independent native-result substitutes under Windows low/optimized. Four authored
test functions cover ownership/automatic closure, input admission, partial and
malformed outputs, borrowed-root preservation, no-repeat phase transitions and
failed-operation cleanup. This is not real Linux syscall execution or a full
Fairy-authored compiler CLI/self-build acceptance claim.

The combined rooted-file/directory/retention/cache/package policy command passes
both Windows profiles in 14.681s (five roots, ten generated native executions):

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinux(PackageDirectoryPolicy|RootedFilePolicy|DirectoryPolicy|RetentionPolicy|CacheFilePolicy)$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

The full Go-reference source check passes 687 files / 8,570 MIR functions and
refreshes the example lock. After comment-only finalization, both full check and
locked check pass again against the current source; the focused policy command
with the final process-PATH observation passes in 3.099s. Scoped vet, Go formatting
and UTF-8/LF/final-newline checks pass.
The first policy attempt fails on the reserved word `cleanup` in native parameter
and local names; `cleanup_code` corrects those names. The next attempt correctly
rejects direct shared resource-field projections through a mutable reference.
Separate shared-observation helpers shorten that reborrow before changing phase,
without widening compiler admission; the following focused and combined probes
pass. Warning levels, budgets and language/ABI contracts are unchanged.

The common Linux fixture runner now sets an empty PATH only after compilation,
inside each execution subtest; test cleanup restores it before the next build.
The expanded Windows command passes fifteen selected roots / thirty generated
native executions in 40.777s, without selected skips. Fourteen roots use that
runner; host-path emission uses its separate runner. The package substitute
also observes the actual process PATH rather than assuming environment setup.

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestLinux(HostPathsEmissionAndPolicy|CommandTransport|ProcessPolicy|CapturePolicy|RootedFilePolicy|PackageDirectoryPolicy|DirectoryPolicy|RetentionPolicy|WorkspaceFilePolicy|BuildScratchPolicy|RunCapturePolicy|TestCapturePolicy|StarterPolicy|CacheFilePolicy|GNUCompanionOptions|GNUBuildPolicy|GNUPolicy)$'
```

Final local logs: `.cache/linux-package-directory-policy-empty-path-20261106.log`,
`.cache/linux-package-directory-policy-20261106.log` and
`.cache/linux-package-directory-check-20261106.log`. These are not staged. This
does not add a CI job, select actual Linux execution or prove missing package
filesystem/transport/CLI integration.

## Package file observations and scan capacity

`linux_package_file.c` adds one held-file metadata observation which rejects
any link count other than one before releasing package metadata. Generic rooted
file observations keep their previous semantics. The private 40-byte metadata
layout remains unchanged; native `EMLINK` is statically bound to Linux error 31.
The Fairy package-file facade retains complete identity/size/time/mode through
the existing single-link conditional erase primitive. It validates names and
metadata before mutation and preserves applied-removal and separate close errors.

`linux_directory::scan_packages` permits 8,192 rows through the same collector,
work ledger and cursor ownership as ordinary scans. Ordinary `scan` still admits
only 1..4,096. Independent 8,192/8,193 streams, exact/one-short work, end names and
ordinary-limit refusal pass without changing the 16,777,216 work limit.

Windows evidence:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^(TestPrivateLinuxPackageFileSelection|TestLinuxPackageFileAdapterBoundaries)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinux(PackageFilePolicy|DirectoryPolicy|RootedFilePolicy|CacheFilePolicy)$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

The C selection/boundary command passes in 0.818s, both MSVC profiles. It checks
link counts 0/1/2, unchanged generic observations, malformed metadata and handles,
zeroed error output, stat failure and observation EINTR retry. Its first attempt
fails only on a test-local `links` name shadowing the shared stub counter under
warnings-as-errors; renaming that local corrects the harness without suppressions.
The Fairy policy command passes four roots / eight native executions in 10.850s
with empty PATH, both-target emission and no selected skips. It includes native
error/malformed-success results, metadata preflight, stale non-removal and applied
erase with close/native failures. Scoped vet, formatting and diff checks pass.
The reference full-source check and locked check pass 688 files / 8,576 MIR
functions; the example lock is refreshed. Logs are
`.cache/linux-package-file-adapter-20261106.log`,
`.cache/linux-package-file-policy-20261106.log` and
`.cache/linux-package-file-check-20261106.log` (not staged).

This prepares the filesystem facade; it does not yet connect package/cache I/O,
offline restoration or transport/CLI selection.

## Package filesystem composition

`linux_remote_fs.fy` now composes retained directories, package scans, single-link
file observation and anonymous-file publication. It preflights complete paths,
ASCII case aliases, symlinks/types, depth and per-read/write byte limits; missing
read/lookup paths do not create directories. Reads preserve exact binary bytes,
probe one byte past the declared size and compare held-file metadata before and
after reading. Only an exact complete read is returned. Writes sync the anonymous
file before no-replace hard-link publication; errors discard its descriptor.

Private tree publication preserves existing trees, and cleanup charges one
caller entry budget plus the depth bound, refuses special/linked/stale files and
removes only identity-matched empty child directories. Earlier successful
cleanup may remain applied when a later entry fails, as in the existing policy.
The caller must supply its own unpublished stage under the fetch lock.

The directory prerequisite adds `clone`: open `.` relative to the held child,
then transfer a separate rooted capability. Container-directory creation can
therefore retain the same object after the temporary removable lease drops,
without reopening a name or allowing partial resource-field moves. C selection,
failure/handle checks and a native Linux identity/lifetime probe are extended.
Fairy clone guards preserve borrowed child/parent handles, reclaim malformed
partial outputs once and retain both primary and cleanup errors.

Windows evidence:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^(TestPrivateLinuxPackageDirectorySelection|TestLinuxPackageDirectoryAdapterBoundaries)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinux(RemoteFileSystemPolicy|PackageDirectoryPolicy)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxPackageDirectoryPolicy$'
```

The C command passes both profiles in 1.323s. The combined Fairy command passes
four native executions in 5.634s; the final extended clone oracle separately
passes both profiles in 3.228s. Its success and malformed/borrowed/partial output
cases observe descriptor lifetime after the source lease drops. The filesystem
oracle independently models nodes, handles, cursors and namespace transitions;
it observes no leaked handles/cursors on ordinary and failure exits, exact binary
and empty files, partial writes, sync before publication, alias/link/type refusal,
short/growing/changed reads, no-replace/stale cleanup and exact entry/depth bounds.
Programs run with empty PATH and both target C-emission profiles pass. These are
Windows generated-program observations against substitutes, not Linux syscalls.

Reference source check and locked check pass 689 files / 8,594 MIR functions.
Scoped vet and diff checks pass; the example lock is refreshed. Local logs:
`.cache/linux-remote-fs-directory-adapter-20261106.log`,
`.cache/linux-remote-fs-policy-20261106.log`,
`.cache/linux-remote-fs-directory-policy-20261106.log`, and
`.cache/linux-remote-fs-check-20261106.log` (not staged).

## Linux archive-cache publication API

`linux_remote_cache::store` now preflights a lower-case 64-byte SHA-256 pin and
the 16 MiB archive bound, hashes input before creating cache directories and
reuses existing blobs only after a fresh bounded, single-link read and rehash.
It retains `build/package-sources/<sha256>.zip`. Corrupt, linked, aliased or
different existing entries fail without replacement. A pre-existing digest-
derived named download stage also fails without removing it.

New writes use the existing anonymous O_TMPFILE capability, partial-write loop,
sync-before-hard-link publication and consuming non-retried close. A publication
collision accepts a competing entry only after re-reading and matching its hash;
different bytes are retained and rejected. Each store makes at most one anonymous
file creation/publication attempt. Sync/write failures retain I/O classification,
and close errors stay separate from the primary operation. A completed blob may
remain after close reports an error; explicit later reuse must rehash it.

The caller still owns ZIP/package identity validation and the fetch lock. The
storage-layer oracle uses synthetic blobs, not proof of archive acceptance or
an online/offline CLI. No transport is imported or invoked by this API.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxRemote(CachePolicy|FileSystemPolicy)$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

The final combined Windows command passes both profiles / four native executions
in 6.845s, without selected skips. Independent namespace/handle/cursor assertions
cover wrong digest before mutation, 16 MiB+1 preflight, zero-length blob storage,
cold publication and unchanged warm reuse, corruption/hard-link/symlink/case-alias
refusal, preserved named stages, partial/zero writes and sync failures, matching
and differing simulated competing publications, and consumed close errors.
The competitor cases observe exactly one creation and one publication attempt;
they are not real concurrent Linux executions. All native processes observe an
empty PATH and finish with no retained fake handles/cursors. Both-target C
emission passes. Input hashing is intentionally independent of a trusted caller.

Reference check and locked check pass 690 files / 8,599 MIR functions, with the
example lock refreshed. Scoped vet, Go format, UTF-8/LF/final newline and diff
checks pass. Local logs are `.cache/linux-remote-cache-policy-20261106.log` and
`.cache/linux-remote-cache-check-20261106.log` (not staged).
The initial focused cache probe passed in 3.376s; review then separated sync-stage
I/O errors from publication conflicts and added explicit sync/zero-write oracles.
The final combined command above includes that correction and the extended model.

## Monotonic operation deadline prerequisite

`linux_fetch_deadline` uses a private Linux `CLOCK_MONOTONIC` millisecond
observation. The C adapter rejects null output, invalid timespec values and
conversion overflow; syscall errors leave zero output and retain their native
code. Deadline creation propagates observation errors. Subsequent errors or a
backward observation fail closed as expired, without replaying the operation.
The existing 120,000 ms operation budget is unchanged.

Windows low/optimized substitute probes pass exact `SIZE_MAX`, overflow,
submillisecond truncation, native error and invalid timestamp cases. Independent
Fairy policy probes pass elapsed 119,999/120,000/120,001 ms, reverse-time,
near-maximum timestamps and start/subsequent clock failures. Generated programs
run with empty PATH. The C selection/boundary command passes in 0.985s and the
Fairy policy command in 2.822s:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^(TestPrivateLinuxClockSelection|TestLinuxClockAdapterBoundaries)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxFetchDeadlinePolicy$'
```

Reference source `check` and `check --locked` pass with 691 source files and
8,604 MIR functions. Native Linux GCC/Clang replay is authored as
`TestLinuxClockAdapterNative` and remains unexecuted. These probes establish
the deadline prerequisite only; they do not establish offline restoration,
networking or Linux CLI integration.

## Private Linux offline restoration

`linux_remote_offline::restore` now composes the existing source-plan, manifest
binding, ZIP32/ZIP64 stored/DEFLATE and SHA-256 owners with the held-directory
Linux filesystem. It reads the plan and manifest without creating missing paths,
validates binding before mutation, and acquires `.fairyc-fetch.lock` exclusively
with a retained directory lease. Failed acquisition never removes a competing
lock. Release checks the original held directory identity and does not recurse.

Every package reads and rehashes only its exact pinned
`build/package-sources/<sha256>.zip`; missing/corrupt caches fail without transport.
Archive/package identity is checked before staging. The unchanged 16 MiB archive,
64 MiB compressed/expanded, 4,096-file, 8 MiB/file, 8,192-entry and depth-32 budgets
are passed to the existing bounded owners. The monotonic two-minute deadline is
checked around processing, reuse and publication; it does not interrupt native
I/O or decoding in the middle of an entry.

An exact existing tree can be reused, including additional empty directories.
Changed/missing/extra files, aliases and links reject without replacement. New
trees use the digest-derived private stage, synced anonymous file publication and
one identity-checked no-replace directory publication. A pre-existing stage stays
untouched. Ordinary failure clears only the owned current stage under entry/depth
bounds. Partial native acquisition and later cleanup failures retain their error
codes; earlier completed packages may remain. Source pins, manifest, graph lock
and cache bytes are unchanged.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxRemoteOfflinePolicy$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestLinuxRemoteOfflineNativeInputs$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

The policy command passes in 17.546s: 25 real-Windows-filesystem Go reference
cases supply independent outcomes and complete final namespace/byte expectations,
and 15 independent Linux-specific fault controls supply exact error expectations.
All 40 execute in each MSVC profile, 80 generated-program processes, with empty
PATH and no retained model handles/cursors. Both-target C emission passes and
rejects Windows host/HTTP symbol leakage. Go offline references make zero requests.
The generated Fairy program runs over a separate namespace/descriptor model;
this is not evidence that Linux syscalls executed on Windows.

Cases cover stored/deflated/ZIP64 archives, fresh publication and exact reuse,
empty directories, cache miss/corruption/case, changed/extra/incomplete packages,
destination files, held locks, invalid plan/manifest/binding/archive identity,
two-package success and retained earlier success after a later failure. Controls
add hard/symbolic links, stale stages, partial/zero writes, sync and cleanup errors,
deadline/clock failures, simulated directory collision and a competing lock after
observation. Cleanup failure preserves the primary failure and owned leftovers;
neither collision publication nor failed lock acquisition is replayed.

The exact real-syscall replay entry and Linux support assembly are separately
checked on Windows, including no model imports and monotonic/nofollow/O_TMPFILE/
no-replace primitives. `TestLinuxRemoteOfflineNative` is authored for actual
GCC/Clang x low/optimized, with the reference corpus and real symlink/hard-link/
stale-stage preservation cases. It is not executed here.

Reference source check and locked check pass 692 files / 8,612 MIR functions.
The initial source check caught an inferred i32 conditional used for the remaining
usize byte budget; the local binding now explicitly uses usize. Initial Go test
compilation caught a target.ID/string comparison in the harness. The expanded
model initially applied its single-publication assertion across a two-package
operation; it now asserts one attempt only for the injected collision, allowing
the independent two-package success expectation. Both profiles pass after those
corrections. Final logs are `.cache/linux-remote-offline-policy-20261106.log`,
`.cache/linux-remote-offline-native-inputs-20261106.log` and
`.cache/linux-remote-offline-check-20261106.log` (not staged).

This is the private offline API only. Verified HTTPS, online/offline Linux CLI
selection, actual Linux execution, broader queue items and final gates stay open.
The remote-source product contract and its budgets are unchanged.

## Linux HTTPS response and lifetime facade

`linux_remote_https::download` now admits exact UTF-8/NUL-terminated URLs and
explicit maximum body/total timeout budgets through the private pull ABI in
`architecture/candidate-linux-https-v1.md`. It validates input before native I/O,
requires status 200 and absent/exact identity encoding, checks declared and actual
lengths, and reads at most 65,536 bytes with an extra-byte overflow/EOF probe.
Scratch buffers survive consuming request close. Every acquired request closes
once, including partial factory failure; primary native errors and independent
close failures are retained together. It does not retry or access the filesystem.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestLinuxRemoteHTTPSPolicy$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

The final focused policy command passes in 4.406s with both MSVC profiles and
empty PATH. Four authored test groups cover URL/UTF-8/NUL/path admission and
zero/maximum/over-limit time/body bounds; partial/malformed native outputs and
exact native/close causes; response/encoding/declared length, truncation and
over-read; and binary data, preserved escaped/Unicode URLs, exact 65,536 and
65,537 byte chunk/EOF boundaries. All 28 model modes execute, with no retained
fake requests and no selected skips. Both-target emission passes.

The first test attempted a narrowing usize-to-u8 observer cast outside the
current subset; the independent observer now widens each received byte instead.
The first full source check rejected an ownership-dependent resource move inside
an if-expression; an explicit error return before the successful owned-buffer
return preserves cleanup and passes the owner checks.

Reference check and locked check pass 693 files / 8,617 MIR functions. Scoped
vet, Go format, strict UTF-8/LF/final newline and diff checks also pass. Only
the private ABI architecture and implementation-routing/receipt/queue documents
are added or updated; the remote-source product contract is unchanged.

This closes the Fairy facade only. Native TLS/DNS/sockets, absolute I/O deadline
and cancellation, Linux runtime symbol selection, real TLS replay, online
orchestration and command selection remain unimplemented. No substitute native
implementation or fallback is selected for production. Do not describe the model
pass as a successful Linux download. Logs are
`.cache/linux-remote-https-policy-20261106.log` and
`.cache/linux-remote-https-check-20261106.log` (not staged).

## Native HTTPS response producer

ADR 0501 selects an isolated request-child producer using system libcurl and
fixed reference Ubuntu/Debian amd64 library/CA paths. The producer and private
pipe framing are implemented in `linux_https_protocol.c` / `linux_https_worker.c`.
They are embedded but intentionally not admitted by LinuxSource: the owning
parent-side watchdog, descriptor boundary, protocol reader and kill/reap adapter
are still required. Calling the producer from a normal multithreaded host is not
an alternative to that isolation boundary.

Fresh libcurl state uses HTTPS only, verified system certificate and hostname,
TLS 1.2 minimum, no redirect/proxy/ambient auth/cookie state and no automatic
content decompression. Every setopt result is checked. Missing libraries/symbols,
unsupported versions/TLS, allocation/setup/perform and cleanup failures retain
numeric errors while acquired local resources close. No archive buffer, helper
executable or native diagnostic text is retained. The parser caps total headers
at 64 KiB / 1,024 lines and body forwarding at maximum+1. It rejects ambiguous
length/transfer metadata, malformed/folded headers and framing-changing trailers;
it preserves informational headers, identical lengths and benign trailers.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^TestLinuxHTTPSWorkerBoundaries$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

The final Windows command passes both profiles in 1.152s. Independent library/
loader/option/callback substitutes cover 24 producer scenarios plus direct header,
overflow, framing and input controls. They inspect every option, exact output
frames/bytes, one perform attempt, partial acquisition cleanup, native/library
errors, module/global/easy/header lifetimes and 65,536/65,537-byte bodies. This is
actual producer code with substitutes, not real libcurl/TLS or child execution.
Initial harness compilation required callback typedefs for MSVC va_arg, explicit
pointer-array initialization and representable signed-char fixture bytes; warning
levels were not changed. The first run also corrected the harness's expectation
that normal scenario zero should make no perform call. Final output is retained
in `.cache/linux-https-worker-20261106.log` (not staged).

`TestLinuxHTTPSWorkerRejectsUntrustedNative` is authored for GCC/Clang x both
profiles against a fresh loopback TLS certificate, without installing a CA or
disabling verification. It remains unexecuted. It would establish only negative
trust behavior, not positive trusted downloads or the pending process deadline.
The optional producer now requires libcurl development headers and the compatible
system libcurl/CA runtime; Go dependencies and baseline/CI provisioning are
unchanged. No current Fairy source changed, so the prior 693-file / 8,617-function
check/locked receipts retain their unchanged input scope.

## Parent-side bounded protocol reader

`linux_https_reader.c` now consumes the producer's pipe protocol without retaining
caller buffers or an archive-sized allocation. It retains detached metadata and
the unconsumed byte count of one body frame. Reads span frames, reject invalid
kind/order/code/length, and admit at most maximum+1 total body bytes. A successful
end requires actual pipe EOF; trailing bytes and truncated frames fail. An input
error is latched, so another read cannot replay the failed stream.

`go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^TestLinuxHTTPSReaderBoundaries$'`
passes both Windows profiles in 0.942s. That initial helper inherited PATH; the
parent-owner validation below reruns this probe with PATH explicitly empty.
Independent byte-layout
fixtures exercise 29 short-read strides, every truncation of the 197-byte valid
stream, twenty malformed/error cases and exact 65,536/65,537-byte boundaries.
A separate producer/reader composition check preserves NUL/0xff bytes. The first
probe compilation found a harness-global name shadowing a producer local; the
fixture name was corrected without weakening warnings. Runtime selection stays
closed until the process/deadline owner exists. Actual Linux remains deferred.

## Parent-side process, deadline and runtime selection

`linux_https_parent.c` now supplies all four private HTTPS functions, with an
owned partial-factory handle, nonblocking parent pipe and one monotonic deadline.
Repeated poll/read interrupts and EAGAIN do not extend the deadline. Cached
headers and successful consuming close also check it. The streaming reader
retains only detached metadata and one frame's remaining byte count.

The dedicated fork child is limited to the standalone single-threaded generated
CLI with default SIGCHLD and no competing reaper. It installs a parent-death
SIGKILL, checks the fork/prctl race, installs fd 3 and checks close_range plus
closure of 0..2 before invoking the producer. Setup errors preserve their native
code in an end frame. The parent closes each owned descriptor once, observes
its exclusively owned unreaped child before kill/reap and never signals after
ECHILD. Signal/reap/close failures remain cleanup failures; a refused signal
does not enter an unbounded blocking wait. OS rejection of cleanup is not a
proof of reclamation. The private architecture documents this boundary and the
absence of a hard real-time scheduling/kernel-reap guarantee.

LinuxSource selects clock/protocol/worker/reader/parent once, only when an HTTPS
symbol is requested, and places GNU feature selection before all libc headers.
Unrequested/unknown/Windows symbols remain excluded. This enables the optional
runtime ABI; it does not yet connect online fetch orchestration or CLI dispatch.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/candidatehost -run '^(TestPrivateLinux(HTTPS|HostSymbol|Clock)Selection|TestLinuxHTTPSParentNativeInputs|TestLinuxHTTPS(Parent|Reader|Worker)Boundaries|TestLinuxClockAdapterBoundaries)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxRemoteHTTPSPolicy$'
go vet -mod=readonly ./internal/candidatehost ./internal/cli
```

The final owner/native-selection regression command passes in 6.023s; all four
native probe groups run low/optimized without selected skips. The unchanged
Fairy facade passes both profiles in 3.938s; scoped vet passes. All three HTTPS
C probes now explicitly run with empty PATH. Review found the earlier shared
path-probe helper inherited PATH; a private HTTPS runner corrects this and the
producer and reader are rerun, rather than reusing the earlier claim.

Independent parent controls cover malformed inputs, allocation/pipe/fcntl/fork
and parent-writer close failures, partial outputs, clocks before/after fork,
backward time, deadline-1/deadline, cached headers, EOF, one-byte reads and
cross-frame binary data, repeated interrupts/EAGAIN, invalid poll/read results,
native and consuming-close causes, ECHILD, kill/wait/close failures and successful
collection expiring during close. The two interrupt storms each expire after
30 simulated 1,000 ms interruptions. Actual child-branch substitutions inspect
descriptor isolation and exact setup-error end frames. Each admitted model
ownership path ends with zero model allocations/descriptors/children; the kill-
failure fixture models a child which has concurrently become terminal, not a
kernel guarantee that rejected signaling kills a live child.

Initial probe compilation caught a function macro also replacing a struct tag,
and later a signed/unsigned assertion; the harness was corrected without lowering
warnings. An initial assertion incorrectly expected the read-end close fault to
fail open and was moved to consuming close. No `.fy` source changed; the prior
693-file / 8,617-function reference check/locked receipt retains its exact input
scope. No full source rebuild is claimed. Native real-process, parent-death,
negative TLS trust, DNS and stalled TLS handshake commands below are authored,
not run. Positive trusted TLS and ordinary response-header/body socket stalls
remain separate from the controlled producer's real-process replay.

## Private Linux online fetch composition

`linux_remote_online::fetch` now reads/binds the source plan and root manifest,
acquires the same held-directory exclusive fetch lock as offline restoration,
and reuses the admitted source-plan/binding/archive/immutable-cache and offline
package-publication owners. Only `linux_remote_fs`'s exact absent classification
permits HTTPS; corruption, aliases, links, special files and read errors fail
without falling back to transport. Every hit and download is hashed again, then
ZIP/package identity is validated before immutable cache storage or publication.
New verified cache blobs and prior complete packages may remain on a later error.

The existing 16 MiB/archive, 64 MiB compressed/expanded, 4,096 files, 8 MiB/file,
8,192 path entries and depth-32 policies remain unchanged. Pure archive/request
budget helpers reject overflowed/exhausted totals before subtraction and cap each
request by min(30 seconds, remaining two-minute deadline). Deadline checks cover
phase boundaries and the final result; existing decoding/filesystem operations
are not interrupted mid-entry. The HTTPS owner supplies its own I/O deadline.
There is no transport, cache or package publication retry.

The private online problem keeps stage/code plus native/consuming-close causes;
in particular HTTPS errors retain their numeric native cause instead of being
reduced to a generic stage. A later lock-cleanup error cannot overwrite a primary
request-close failure. No public command or SDK type is changed by this module.

```powershell
go test -mod=readonly -count=1 -timeout 8m ./internal/cli -run '^TestLinuxRemoteOnlinePolicy$'
go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestLinuxRemoteOnlineNativeInputs$'
go test -mod=readonly -count=1 -timeout 8m ./internal/cli -run '^TestLinuxRemoteOfflinePolicy$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

Final Windows online policy passes in 22.864s, native replay input checks in
5.776s and the offline regression in 27.603s, without selected skips. Both target
C-emission profiles and native Windows low/optimized execution pass with empty
PATH. The online oracle compares Go outcomes, request counts and complete final
file/directory bytes against an independent Linux ABI namespace/HTTPS model. It
covers cached and downloaded stored/DEFLATE/ZIP64, exact and empty-directory reuse,
corrupt/aliased caches, conflicting trees, invalid binding/archive/package identity,
two-package and partial-success behavior, new cache parents and byte-exact escaped/
Unicode URL delivery. Pure helpers cover archive cap-1/cap/cap+1, exhausted/SIZE_MAX
totals and 29,999/30,000/30,001 ms request caps.

Independent controls observe factory/partial/metadata/read/close/native timeout
causes, status/encoding/byte/ABI rejection, cache zero writes/sync/consuming-close,
matching/differing publication competitors, operation expiry before/after download,
15,000 ms and 1 ms remaining request budgets, start-clock failure, hash mismatch,
one-attempt package collision, competing lock and combined primary/request-close/
lock-cleanup errors. Complete expected namespaces preserve verified blobs and
foreign/failed-cleanup leftovers; model handles/cursors/requests close.
Those simulated competitors and faults are not evidence of Linux kernel races.

The initial online run exposed two harness URL issues: Go Request.URL.String
escapes Unicode, and native C fixtures need explicit byte escapes independent of
the C execution character set. The Go lookup now uses parsed URL serialization;
native addresses use exact UTF-8 octal bytes. The corrected complete run above
passes both profiles. Scoped vet, Go formatting and diff checks pass. Reference
source check and locked check pass 694 files / 8,623 MIR functions, refreshing the
example graph lock for the added authored module. Source pins/manifests and user
workspace graph locks are never modified by the online implementation.

`TestLinuxRemoteOnlineNative` is authored for GCC/Clang x low/optimized. It uses
the actual selected filesystem/clock/parent/worker/libcurl assembly, checks cached
operations open zero loopback connections, rejects a fresh untrusted loopback TLS
certificate without changing trust, and observes no unreaped child after the
Fairy operation. Its exact inputs and no-model runtime assembly pass on Windows;
actual Linux compilation/execution is not run. This is not positive trusted TLS,
CLI dispatch, baseline provisioning or final self-build/distribution acceptance.

## Authored Linux replay and remaining implementation

```sh
go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxPackageDirectoryAdapterNative$'
go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxPackageFileAdapterNative$'
go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxClockAdapterNative$'
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestLinuxRemoteOfflineNative$'
go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxHTTPSWorkerRejectsUntrustedNative$'

go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxHTTPSParent(Process|RejectsUntrusted|DNSDeadline|TLSDeadline)Native$'

go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestLinuxRemoteOnlineNative$'
```

This test is authored, not executed. It selects GCC/Clang x low/optimized,
checks real no-replace collisions, binary contents, empty-tree cleanup,
symlink/replaced-inode refusal and continued use after closing the original
parent and renaming its path. It remains required in the combined Linux replay.
The separately authored file probe checks anonymous/single-linked/hard-linked
files, binary bytes and conditional removal preserving a hard-linked file. Its
GCC/Clang x low/optimized executions are also not run and remain in that batch.

Next implementation: online/offline Linux CLI selection and positive trusted TLS
acceptance. Parent process/clock/descriptor and actual
libcurl replay remain in the combined Linux batch; the Windows substitute does
not prove their Linux behavior. Complete
Go/Fairy comparison, consecutive self-build, Windows clean-machine distribution
and same-commit hosted acceptance remain open.

No public CLI/SDK ABI, dependency, DB, CI/runner, repository hygiene or existing
timeout/budget policy changes. The package policy in `remote-sources-v1.md` is
unchanged; only the implementation routing index and queue gain this receipt.
