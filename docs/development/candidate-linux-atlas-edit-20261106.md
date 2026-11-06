# Linux Atlas edit implementation, 2026-11-06

## Native file prerequisite

`internal/candidatehost/linux_edit_file.c` is a private prerequisite for the
existing bounded single-body edit contract. It is selected after rooted/package
file support and before retention. This slice does not select edit in the Linux
CLI or claim Linux execution. The Fairy lifetime facade is implemented below. Linux preview/apply composition,
entry/help selection and full command acceptance remain required.

The caller holds the exclusive edit lock and owns a private same-filesystem
staging directory. `edit_file_prepare_v1` accepts only an anonymous regular file,
sets exactly the supplied 0777 permission bits with `fchmod`, and syncs that held
file. The caller must finish preparation before rechecking the workspace closure.
It never modifies the old source file's permissions.

`edit_file_publish_v1` requires distinct held stage/destination directories and
an anonymous prepared file with matching permission bits. It rejects read-only
originals using the Go editor's owner-write 0200 policy. The exact retained
device/inode/size/mtime/mode must still match a named regular destination;
symlinks and malformed metadata fail. Other hard links to the old inode are
preserved by replacement of only the selected path, matching the Go editor. It creates one link named `body`
inside the private stage, rechecks the original, then performs one `renameat`.
There is no destination removal, publication retry or fallible post-rename step.
On pre-publication failure, cleanup removes only the stage name which still
identifies the held file; primary and cleanup errors are returned independently.
Input metadata and all borrowed descriptors remain caller-owned.

This follows the opened-directory and atomic-replacement behavior described by
[rename(2)](https://man7.org/linux/man-pages/man2/rename.2.html), anonymous-file
publication in [open(2)](https://man7.org/linux/man-pages/man2/open.2.html), and
descriptor-based permissions in [chmod(2)](https://man7.org/linux/man-pages/man2/fchmod.2.html).
The authority scope remains cooperating editors in trusted local directories.
Hostile writers, network-filesystem commit ambiguity, ACL/owner preservation
and physical-media durability are not established by this implementation.

## Windows evidence and authored Linux replay

`TestLinuxEditFileAdapterBoundaries` compiles the actual new C adapter against an
independent POSIX namespace/failure model under MSVC low/optimized. All 512
permission masks are checked for preparation; separate cases cover handle/path/
metadata rejection, six freshness fields, same-directory/cross-device refusal,
symlink refusal, nanosecond range and truncation, read/EINTR handling,
chmod/sync/link/rename failures, secondary cleanup failures and substituted stage
names. Every failure preserves the old target and rename attempts never exceed
one. The prerequisite commit execution passed in 1.935s (body 1.84s; low 0.84s, optimized
0.76s). The first fixture compilation rejected two signed/unsigned comparisons;
the model's expected error expressions were explicitly typed before this pass.
Related rooted/package-file/package-directory selection and boundary regressions
pass in 7.019s on their unchanged adapter inputs.

`TestLinuxEditFileAdapterNative` is authored for GCC/Clang low/optimized. It checks
real binary replacement, permission retention, an old opened descriptor's bytes,
stale-source/read-only refusal, old hard-link alias preservation and final cleanup. It explicitly
skips on Windows and has not executed here. The two new native metadata range
cases are checked by the Windows model, not claimed as actual kernel behavior.

## Companion integration

The Go and Fairy runtime selectors admit only the two named new exports. Host
bit 18 selects rooted-file plus edit-file support; asset ID 33 preserves all old
IDs and assembles the new C immediately after package-file support. The new
prerequisite commit pack had 33 sources and 176,458 bytes, SHA-256
`ff5a498c279769191ebbc8931f486a146e1228a705ff1490f2d8207275b5623a`.
The subsequent alias-aligned current pin is recorded below.

Both Windows profiles pass 120 independently derived live-Go runtime plans and
80 complete C output comparisons plus pack-tampering controls. The final combined
command exits 0 in 30.062s: whole companion both-target emission 2.81s, complete
assets 19.71s and selector 7.36s. Generated programs run with empty PATH. Actual
GNU compilation, syscalls and Linux self-build remain in the native batch.

```powershell
go test -mod=readonly -count=1 -timeout 4m -v ./internal/candidatehost -run '^Test(PrivateLinuxEditFileSelection|LinuxEditFileAdapterBoundaries|LinuxEditFileAdapterNative)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxGNUCompanion(Runtime(Assets)?|EntryEmission)$'
python -B tools/package-candidate-linux-runtime.py --update-pin
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-edit-file-final-20261106
```

Original check/locked passes 698 files / 8,643 MIR functions; freshly prepared
Linux check/locked passes 698 / 8,644. Scoped vet, formatting, canonical text,
pin verification and diff checks pass. No public CLI/SDK ABI, DB, dependency,
timeout, runner or repository-policy change; no push, dispatch or deployment.
The broader implementation queue and all final acceptance gates stay open.

## Fairy file lifetime facade

`linux_edit_file.fy` now owns the anonymous descriptor, immutable original
metadata, cumulative source byte count and four private phases: writable,
prepared, published and sealed after a native failure. Creation validates the
original regular/writable metadata before opening a file. The source snapshot
is copied during construction and never replaced by later preparation calls.

Writes share `linux_rooted_file::write_all`, retaining partial progress and
native protocol checks. An explicit unsafe owned-staging-handle constructor is
the inverse of the existing consuming `into_handle`. The facade clears its own
handle before transferring ownership into a named rooted File and restores it
only after that File is consumed. It adds no projected resource-field borrow or
new source-language admission. Every ordinary exit retains exactly one close
owner. Explicit consuming close clears the handle before the native call and
reports whether publication already occurred; an uncertain close is not retried.

The cumulative bound is 8 MiB. A new write invalidates prior preparation; a
failed native write or chmod/sync seals the file. A valid publication attempt
also seals it before crossing the host boundary, and success marks publication.
Invalid leaf/budget arguments reject before native calls. Native primary and
cleanup causes are retained separately. A malformed success-with-cleanup result
still records known publication, so callers cannot blindly replay the edit.

The initial native prerequisite rejected multiply-linked originals. Review of
the Go editor's opened-root rename showed that this was unnecessarily narrower:
the target path can change while other links retain the old inode and bytes.
The current C adapter and independent Windows model now admit that case. The
new staging file still must be anonymous. The authored Linux probe also checks
old alias bytes after replacement; actual Linux execution remains unrun.

`TestLinuxEditFilePolicy` passes five source groups under both Windows profiles
in 4.007s initially and 2.95s in the final regression (low 1.09s, optimized 1.51s).
It checks before-I/O metadata rejection, 8 MiB minus-one/exact/one-over writes,
preparation invalidation, copied metadata, one-attempt publication, partial
writes, primary/cleanup protocol errors, automatic lifetime cleanup and explicit
close errors before and after publication. Generated programs run with empty
PATH and emit for both targets.

Early prototypes were rejected for reserved cleanup names, an empty annotated
cleanup, projected resource/record mutation and short-circuit consuming fixture
cleanup. These were corrected using the existing grammar, an immutable factory
snapshot and explicit handle ownership transfer; no admission or warning policy
was relaxed.

The final policy/rooted/full-entry regression exits 0 in 41.882s. Rooted policy
passes in 3.76s; current full Linux entry emits both targets with 25 partitions
and a 2,032,407-byte header. Portable Windows dispatch passes in 6.93s. The outer
34.86s full-entry test explicitly skips native Linux execution. It does not
execute a Linux edit command.

The alias-aligned C adapter Windows model passes in 1.402s (body 1.33s: low
0.58s, optimized 0.60s); its native Linux test explicitly skips. The current pin
has 33 sources / 176,611 bytes, SHA-256
`0f77b04d88f856ef03c9f1102bcfba532a71ed80faad30505947ecd8e507bcaf`.
Both profiles pass the renewed 120 live-Go plans and 80 complete C outputs in
17.983s combined. Original check/locked passes 699 files / 8,654 MIR functions;
fresh Linux preparation passes 699 / 8,655. Scoped vet and text/pin checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^(TestLinuxEditFilePolicy|TestLinuxRootedFilePolicy|TestLinuxFileCLI)$'
go test -mod=readonly -count=1 -timeout 4m -v ./internal/candidatehost -run '^Test(PrivateLinuxEditFileSelection|LinuxEditFileAdapterBoundaries|LinuxEditFileAdapterNative)$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxGNUCompanionRuntime(Assets)?$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-edit-facade-final-20261106
```

Linux edit lock/stage ownership, original-source reads/metadata, preview/apply
composition, preparation, command/help selection and actual command acceptance
remain required. In particular, the source reader must admit regular hard-link
aliases rather than select the package-download reader's stricter provenance
policy. This facade does not complete edit or any final acceptance gate.
Public API/SDK ABI, DB, dependencies, product deadlines, CI/runners and repository
policy remain unchanged; no push, dispatch, release or deployment.

## Edit-specific source reads

`linux_edit_fs.fy` now reads portable workspace inputs without selecting the
package-download observer or its single-hard-link provenance rule. The caller
retains the root. The reader opens every directory without following symlinks,
requires exact byte spelling and a regular final file, and compares the opened
file's device/inode/size/mtime/mode against the named observation. The same six
fields must still agree after reading. A size-plus-one buffer and explicit EOF
probe reject both truncation and growth; the maximum admitted bound is 8 MiB.
Read-only regular files and regular hard-link aliases remain valid read inputs.

The detailed `read_observed` port moves one owned byte Buffer and copies out its
metadata only after successful file and directory closes. Failure leaves that
metadata argument unchanged. Primary errors, consuming file/cursor close errors,
and owned-directory close errors are separate. The `read` compatibility port
retains the shared editor's Buffer/u32 signature. There are no publication or
namespace-mutation operations in this module.

`linux_directory::lookup_edit` streams one exact spelling using a 256-byte row
and detached scalar metadata, rather than allocating a listing of all siblings.
The shared budget admits 100,000 non-dot entries across path components. A
separate 64 MiB work budget charges every native record and EOF probe, including
dot/vanished rows, so skipped records cannot create an infinite scan. Dot entries
do not reduce the entry limit. Unlike package tree validation, unrelated names
need not be portable or UTF-8 and a differently cased sibling does not hide an
existing exact name. Existing directory/package scan policies are unchanged.

`TestLinuxEditSourcePolicy` executes five source groups under Windows low and
optimized with an independent descriptor/stream model. It covers binary/empty
reads, metadata snapshots, hard-link admission, read-only files, exact spelling,
unrelated invalid UTF-8, symlink refusal, all six open/read freshness fields,
8 MiB minus-one/exact/one-over, truncation/growth/partial reads, 100,000 entries
minus-one/exact/one-over, shared path budgets, malformed records, vanished/dot
streams, I/O and combined close failures. Model handles/cursors consume exactly
once and no shell helper is available. `TestLinuxEditSourceGoReference` separately
reads real multiply-linked binary files with the current Go editor, checking
exact/one-short byte bounds and portable/case rejection on this Windows host.

The first probes rejected untyped branch constants, resource-field extraction,
and two missing statement separators. The final implementation returns a Buffer
directly and copies metadata only on success; no language admission or warning
policy was weakened. The combined source/directory/package-reader/edit-file/
rooted regression passes in 19.275s, including source policy 4.20s (low 1.84s,
optimized 1.99s) and the independent Go reference. Both-target C emission passes.
This remains a modeled Linux ABI on Windows, not actual Linux filesystem replay.

`TestLinuxEditSourceNative` independently emits its actual-filesystem client for
both targets and selects the real complete Linux runtime (0.30s on Windows).
GCC/Clang low/optimized executions are authored for Linux: read-only hard-link
aliases, binary/empty/nested inputs, symlink components, wrong spelling and
oversized/nonregular inputs. The fixture root contains a space; generated
programs run with empty PATH. This Windows run explicitly skips those native
executions, and no such execution is included in the passing model count.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestLinux(EditSource(Policy|GoReference)|DirectoryPolicy|RemoteFileSystemPolicy|EditFilePolicy|RootedFilePolicy)$'
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestLinuxEditSourceNative$'
go vet -mod=readonly ./internal/cli
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-edit-source-20261106-final
```

Original check/locked passes 700 source files / 8,675 MIR functions; freshly
prepared Linux check/locked passes 700 / 8,676. Scoped vet, Go formatting, UTF-8,
LF, final-newline and diff checks pass.
Linux edit lock/stage composition, preview/apply, preparation selection, public
command/help wiring and actual command acceptance remain required. Native Linux,
the complete current-source/self-build/distribution/hosted acceptance and every
other implementation queue family stay open. Public CLI/SDK ABI, DB, dependencies,
timeouts, CI/runners and repository policy are unchanged; no push or dispatch.

## Consuming directory-lease close

Edit lock and staging cleanup need observable close results. The existing
`linux_package_directory::Directory` now owns its retained parent and child
descriptors directly and exposes a borrowed rooted child view. Ordinary rooted
directories and independently duplicated children retain their own close owner.
The unsafe borrowed-view constructor requires an external owner which outlives
every use; safe `directory(ref lease)` access keeps the lease's borrow lifetime.
A borrowed view's drop does not close the owner and consuming rooted close
rejects that view before entering the native boundary.

The new consuming lease `close` clears both owned handles, closes parent then
child exactly once, and returns both native causes. It never removes the named
directory or retries an uncertain close. Automatic cleanup preserves the same
parent-before-child order. This avoids resource-field moves, mutable projected
resource borrows and hidden descriptor duplication. No native export, runtime
asset/pin, public SDK ABI or language feature is added.

`TestLinuxPackageDirectoryPolicy` and `TestLinuxRootedFilePolicy` pass Windows
low/optimized in 6.676s. New cases cover borrowed view drop/close, ordinary and
published/erased lease close, each individual close failure and combined failures.
Existing malformed ownership, partial acquisition, independent duplicate and
automatic cleanup counts remain unchanged. Related edit-source/directory/cache/
scratch/starter/package filesystem and online/offline fetch regressions pass in
86.775s; the direct source native test emits both targets/selects the real Linux
runtime but explicitly skips actual Linux execution here.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestLinux(RootedFilePolicy|PackageDirectoryPolicy)$'
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestLinux(EditSource(Policy|GoReference|Native)|RemoteFileSystemPolicy|RemoteCachePolicy|RemoteOfflinePolicy|RemoteOnlinePolicy|CacheFilePolicy|DirectoryPolicy|BuildScratchPolicy|StarterPolicy)$'
```

Linux edit publication composition, prepared source selection, actual commands
and the broader implementation/final acceptance queue remain required. This
prerequisite makes close failures observable; it does not establish an applied
Linux edit or native Linux execution. Original check/locked passes 700 source
files / 8,677 MIR functions; a fresh Linux preparation passes 700 / 8,678. Scoped
vet, Go formatting and canonical text/diff checks pass. No push, dispatch or deployment.

## Linux application policy composition

`linux_edit_apply.fy` composes the real retained-directory, edit-file and
edit-source owners. It takes the already verified Preview and shared Options,
acquires the cooperating `.fairyc-edit.lock`, rechecks the complete overlay,
retains the exact source parent and reads immutable original bytes/metadata.
Read-only or different originals reject before creating a staging directory.
At most sixteen definitely existing-name collisions select an unused private
stage; EINTR, other errors and secondary acquisition failures never retry.

An anonymous file in that same-filesystem stage receives the complete new bytes,
preserved permissions and sync. A fresh exact closure fingerprint follows
preparation. The original bytes and all six metadata fields must still agree
before the single native publication attempt. There is no destination deletion,
publication retry, recursive cleanup or implicit lock refresh. Existing names
belonging to competitors are left alone.

Each owner has an outer cleanup layer: consuming file close, identity-checked
empty staging erase and two-descriptor close, retained source-parent close,
then empty lock erase and close. Primary and first cleanup causes retain their
precedence. A known publication remains known through file, directory, lock
or native-protocol cleanup failure, so the command can tell callers to inspect
hashes rather than blindly replay. Failed cleanup leaves inspectable names.

The shared `project_edit_snapshot::capture_path` port retains existing capture
semantics without exposing one platform's Directory type to another application
owner. It closes its opened root explicitly. Windows `TestCandidateEditSnapshot`
passes low/optimized in 25.695s: independent Go fingerprint equality for existing
and new entry points, source/lock drift rejection, restoration and immutability.
Linux preparation/command selection remains the next separate integration step.

`TestLinuxEditApplyPolicy` runs the actual application/file/reader/directory
modules against the independent namespace model. Only preview recheck and
closure snapshot ports are controlled; the Options factory is appended inside
its real module to preserve private construction. Twenty-eight scenarios per
profile cover successful binary replacement and surviving old hard links,
fifteen/sixteen existing-stage collisions, lock contention, anonymous-file
failure, partial writes, preparation, overlay/closure failures, read-only and
changed originals, I/O/close errors, final bytes/metadata races, non-replayed
EINTR, malformed success-with-cleanup, and combined cleanup failures. Every
owned descriptor/cursor is consumed and expected competing/failed-cleanup names
are checked; the native rename model never observes more than one attempt.
These policy results do not execute the real Linux kernel or the real source
checker inside the controlled ports. Separate C adapter and Windows snapshot
receipts retain those narrower boundaries; full prepared command acceptance
and actual Linux execution are still required.

Prototype probes rejected a reserved fixture parameter, private Options
construction, a missing statement separator, an ambiguous fixture borrow and
MSVC's unsafe strcpy warning. The final probes use a module-owned factory,
explicit control flow and bounded literal copies. Product admission and warning
policies are unchanged. The initial complete policy run passes in 3.890s;
the related shared-helper/edit-source/root/lease regression passes in 18.883s.
The final link-count/alias-consistent namespace replay passes in 3.972s
(low 1.28s, optimized 2.12s), including exact preservation of every pre-existing
collision directory and its foreign child.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestLinux(EditApplyPolicy|EditSourcePolicy|RootedFilePolicy|PackageDirectoryPolicy|CommandTransport)$'
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestCandidateEditSnapshot$'
```

This closes the application-policy composition, not Linux CLI wiring or the
full objective. Original check/locked passes 701 source files / 8,689 MIR
functions. Public CLI/SDK ABI, DB, dependencies, default budgets/timeouts,
CI/runners and repository policy are unchanged; no push or dispatch. Fresh Linux
preparation check/locked passes 701 files / 8,690 MIR functions. Scoped vet,
Go formatting, UTF-8/LF/final-newline and diff checks pass.
