# Private Linux candidate workspace new v1

## Shared starter bytes

`project_starter.body` owns the five deterministic starter files, with explicit
name, UID and bundled language inputs. The Windows `file_cli` delegates body
construction and preserves its own host creation/rollback logic. Paths remain
`.gitignore`, `README.md`, `fairy.toml`, `src/main.fy` and `tests/main.fy`.
No lock is created; the first ordinary workspace command owns lock publication.

`TestCandidateStarterTemplates` compares all five file bodies byte-for-byte with
the reference Go scaffold, emits both target profiles and executes the checks in
low/optimized host profiles. The candidate source workspace passes both `check`
and `check --locked`. The Linux adapter and dispatch reuse the same bytes without a frontend subprocess.

## CLI connection

`linux_file_cli` connects `new <package-name> --uid <package-uid>` and its help
aliases. `linux_new_cli` uses the shared manifest name/portable-segment and UID
validators before filesystem access; it accepts no destination, force, merge,
network or template surface. Argument storage uses the existing dynamic process
reader. The bundled SDK language is passed explicitly to the shared template.
A completed creation reports the same success bytes as the reference scaffold.
Invalid invocation returns 2, allocation/output failure 4 and filesystem failure 7.
Host paths and raw errno values do not enter product diagnostics.

`new` does not load a parent manifest, create a lock, resolve a compiler, invoke
a companion or create build-cache records. The first check/test still owns graph
admission and lock publication. Existing preparation choices are unchanged: the
Linux snapshot differs from authored sources in exactly nine platform-selection
modules, and the authored Windows entry retains its original host adapter.

## Fixed rooted creation

`linux_starter` renders all five bodies and their native-size lengths before
opening the current directory. Aggregate content is bounded to 65,536 bytes; the
native length transport is a repr(C) five-usize record. A borrowed fd+1 parent
handle and one validated child leaf enter `fairyc_private_linux_starter_create_v1`.
The adapter retains descriptors for the new root, src and tests directories.
Creation uses `mkdirat` and exclusive CLOEXEC/NOFOLLOW file opens, with ordinary
0755 directory / 0644 file modes subject to the caller's umask. Files receive
partial-write loops and file sync; all three directories and the parent sync
before success. Recorded device/inode/type identities are checked before completion.

Existing target files, directories and symlinks, including dangling symlinks,
fail exclusive root creation and remain untouched. No unchecked path prefix or
user-supplied nested destination is traversed. No shell or Go scaffold runs.

On ordinary failure the adapter attempts reverse-order removal of only recorded
created entries. Every unlink is relative to a pinned parent and requires the
recorded nofollow identity/type. It removes at most five files and three
directories; it never enumerates or recursively clears a tree. Unknown entries,
changed identities or creation without an observed identity block removal and
produce a rollback-failure diagnostic rather than deleting unowned entries.
All owned descriptors close once, including fd zero; uncertain closes are not retried.

This is not an atomic directory publication or a security boundary against a
noncooperating same-user writer. Such interference can race an identity check.
A crash or filesystem failure can leave a partial tree. Final directory-close
failure retains the completed tree and reports failed rollback; parent-close or
output failure can also leave a completed project while returning failure. Inspect
remaining entries before manual recovery; no subsequent sweep claims ownership.

## Evidence and remaining boundary

The shared template oracle compares all five bodies with the Go reference in
both emitted targets and low/optimized host executions. Windows POSIX substitutes
cover exclusive creation, partial writes, create/open/stat/write/sync/close errors,
rollback, identity replacement, unknown entries and fd-zero ownership. The Fairy
wrapper tests mode-independent ABI lengths, failure mapping, close ownership,
invalid input and inconsistent native results in low/optimized host profiles.
Full Linux entry emits both target C profiles, contains no Windows foreign symbols
and selects closed Linux native support; its portable dispatcher executes in both
host profiles. Source check/locked checks pass with 562 sources and 7,101 MIR
functions; the Linux-prepared workspace passes both with 562 sources and 7,102
functions. The updated companion cross-build is ELF64 little-endian x86-64.
`go vet ./...` and diff checks pass.

Linux-native GCC/Clang probes are authored for exact binary files, no-overwrite,
symlink preservation and cleanup. CLI integration covers Unicode/spaced parent
paths, reference file/success equality, invalid invocation, existing targets and
links, lock absence, and the generated check/semantic-test/native-test/run/build
pipeline. These Linux executions are skipped on the Windows development host;
no Linux-native success or clean-machine readiness is claimed. Cache wiring,
candidate C emission, a Fairy GNU backend and distribution/self-host evidence
remain separate; Go stays the default compiler.
