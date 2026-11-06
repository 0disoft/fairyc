# Linux Atlas edit implementation, 2026-11-06

## Native file prerequisite

`internal/candidatehost/linux_edit_file.c` is a private prerequisite for the
existing bounded single-body edit contract. It is selected after rooted/package
file support and before retention. This slice does not select edit in the Linux
CLI or claim Linux execution. The Fairy lifetime facade, Linux preview/apply
composition, entry/help selection and full command acceptance remain required.

The caller holds the exclusive edit lock and owns a private same-filesystem
staging directory. `edit_file_prepare_v1` accepts only an anonymous regular file,
sets exactly the supplied 0777 permission bits with `fchmod`, and syncs that held
file. The caller must finish preparation before rechecking the workspace closure.
It never modifies the old source file's permissions.

`edit_file_publish_v1` requires distinct held stage/destination directories and
an anonymous prepared file with matching permission bits. It rejects read-only
originals using the Go editor's owner-write 0200 policy. The exact retained
device/inode/size/mtime/mode must still match a single-linked regular destination;
symlinks, aliases and malformed metadata fail. It creates one link named `body`
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
hard-link/symlink refusal, nanosecond range and truncation, read/EINTR handling,
chmod/sync/link/rename failures, secondary cleanup failures and substituted stage
names. Every failure preserves the old target and rename attempts never exceed
one. Final Windows execution passes in 1.935s (body 1.84s; low 0.84s, optimized
0.76s). The first fixture compilation rejected two signed/unsigned comparisons;
the model's expected error expressions were explicitly typed before this pass.
Related rooted/package-file/package-directory selection and boundary regressions
pass in 7.019s on their unchanged adapter inputs.

`TestLinuxEditFileAdapterNative` is authored for GCC/Clang low/optimized. It checks
real binary replacement, permission retention, an old opened descriptor's bytes,
stale-source refusal, read-only/hard-link refusal and final cleanup. It explicitly
skips on Windows and has not executed here. The two new native metadata range
cases are checked by the Windows model, not claimed as actual kernel behavior.

## Companion integration

The Go and Fairy runtime selectors admit only the two named new exports. Host
bit 18 selects rooted-file plus edit-file support; asset ID 33 preserves all old
IDs and assembles the new C immediately after package-file support. The new
pinned pack has 33 sources and 176,458 bytes, SHA-256
`ff5a498c279769191ebbc8931f486a146e1228a705ff1490f2d8207275b5623a`.

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
