# Private Linux candidate workspace check v1

The Linux dispatcher connects `check`, `check --locked`, and `check --help` to the
existing `project_check_cli` path. It performs the same manifest/dependency/source,
SDK authentication, lock policy, visibility, semantic/IR processing and report rendering
as the original candidate. It does not spawn the Go compiler or another frontend.
Unknown or duplicate options and extra arguments return 2 before workspace access.
The opt-in [Linux workspace cache](candidate-linux-cache-v1.md) accepts
`--cache-root <existing directory>` and `--cache-status` through the shared parser.
[Workspace build](candidate-linux-build-v1.md) uses a separate bootstrap companion;
public C emission remains separate work.

## Platform selection

`tools/prepare-candidate-linux-cli.py` prepares the isolated Linux source snapshot:

| Authored module | Linux selection |
| --- | --- |
| `main` | Import `linux_file_cli` under the existing dispatcher alias |
| `project_directory`, `project_source_catalog`, `project_source_file`, `project_source_set` | Import `linux_workspace_fs`; select its explicitly prefixed `Workspace*` types |
| `project_lock_write` | Select Linux directory preflight and collision observation; keep portable SDK exclusive-create/write/sync/close/replace/remove operations and their existing error bookkeeping |

Every expected import/lock-writer operation is checked before replacement. Changes to
these boundaries require an explicit preparation update. Authored modules remain the
source of truth; the copied files are reproducible platform selection, not independent
implementations. Embedded SDK sources, identities and authentication pins are unchanged.

## Workspace filesystem facade

`linux_workspace_fs.fy` owns the reader boundary used by those shared modules. It is
compiler-private and does not change the SDK filesystem API or its admitted public ABI.

Directory resolution starts from `/` or the current directory, skips empty/dot segments,
and opens every subsequent component through the retained parent descriptor. Other leaves
must satisfy the rooted adapter's UTF-8/255-byte/no-parent-traversal rules. Closing each
old descriptor after opening its child prevents intermediate symlinks from redirecting
later traversal. Workspace manifest/source portable-path and canonical-case rules still
come from the existing shared modules.

Directory opening collects at most 4,096 entries with the existing bounded Linux scan,
closes its cursor and descriptors, then exposes an owned name/metadata snapshot through
`next_dir`. A too-small destination buffer does not advance the row. EOF is repeatable;
`close_dir` releases snapshot storage and performs no delayed OS close. Scan work has the
existing 16,777,216-unit cap; copied names and scalar rows are bounded by the entry/name
limits. This snapshot is not an atomic view of concurrent filesystem changes.

Regular source files use nofollow rooted opens. An explicit `linux_rooted_file.into_handle`
transfer leaves exactly one owner in the workspace file facade. Metadata supplies disk
kind, size and `(device,inode)` identity. Reads retain the 65,536-byte per-call bound;
the shared source reader retains its 8 MiB per-file/cumulative budget, repeated canonical
observation and reopened-identity check. Consuming close clears the owned descriptor
before reporting an error so cleanup cannot close it twice. Linux `ELOOP`, `ENOMEM` and
`ENOTDIR` preserve their errno while receiving the corresponding existing path diagnostic
classification; Windows classifications remain supported.

Lock writing retains the existing named, exclusive temporary-file protocol, canonical
payload validation, bounded collision namespace, sync-before-replace order and secondary
failure fields. It does not become anonymous rooted cache publication or gain a new
power-loss/concurrent-writer guarantee. The private bounded `lstat` observation classifies
a collision without following the final symlink; success only selects another temporary
name and never authorizes replacement/removal of the colliding object. The stable CLI
checks the current workspace (`.`), not an arbitrary root supplied as an argument.

## Validation and remaining evidence

The full prepared Linux entry checks and emits both Linux GNU and Windows MSVC C with
no Windows foreign symbols. Closed Linux support selection includes rooted reads,
directory scans, portable SDK lock I/O and the private collision observer. Windows
low/optimized runs exercise the facade using controlled native substitutes: owned
snapshots, small-buffer retry, EOF, metadata/identity/read, prefix symlink rejection,
descriptor transfer and close-error cleanup.

Linux-only full CLI cases cover missing lock, lock creation/reuse, immutable locked check,
source drift, type diagnostics, embedded SDK input, local dependencies/tests, symlinks and
case collisions. Successful locked output is compared against the Go reference. The C
observer also has Linux-native mode/link/missing-path probes. These native Linux cases
are authored but skipped on the Windows development host; emission and substitutes are
not Linux execution proof. No new hosted workflow or runner is introduced.

Build from the isolated snapshot using the [Linux CLI bootstrap procedure](candidate-linux-cli-v1.md),
then run `build/lexer-candidate-linux check [--locked]` from a client workspace. Source
workspace `check`/`check --locked`, affected native policy tests, native support selection,
`go vet` and diff checks are the local verification boundary. Linux native execution and
target-specific public C emission remains open; Go stays the default compiler.
