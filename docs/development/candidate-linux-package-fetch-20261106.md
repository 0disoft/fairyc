# Linux Fairy package fetch, 2026-11-06

Part of `implementation-queue-20261105.md`. This receipt does not complete
Linux fetch, broader source parity, deferred designs or final acceptance.
Native Linux execution remains batched at the maintainer's request. No push,
hosted dispatch, release or deployment is authorized by this work.

## Retained-directory native prerequisite

`internal/candidatehost/linux_package_directory.c` supplies four private
directory operations selected by `LinuxSource` after rooted I/O:

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

## Authored Linux replay and remaining implementation

```sh
go test -mod=readonly -count=1 -timeout 10m -v ./internal/candidatehost -run '^TestLinuxPackageDirectoryAdapterNative$'
```

This test is authored, not executed. It selects GCC/Clang x low/optimized,
checks real no-replace collisions, binary contents, empty-tree cleanup,
symlink/replaced-inode refusal and continued use after closing the original
parent and renaming its path. It remains required in the combined Linux replay.

Next implementation: Fairy-owned directory capability/facade, package file and
cache publication, offline restoration, verified HTTPS and online/offline Linux
CLI selection. The Windows substitute does not prove those paths. Complete
Go/Fairy comparison, consecutive self-build, Windows clean-machine distribution
and same-commit hosted acceptance remain open.

No public CLI/SDK ABI, dependency, DB, CI/runner, repository hygiene or existing
timeout/budget policy changes. The package policy in `remote-sources-v1.md` is
unchanged; only the implementation routing index and queue gain this receipt.
