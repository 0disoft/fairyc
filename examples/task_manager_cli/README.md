# Task manager with list filters and prepared-file replacement

Windows AMD64 and Linux GNU amd64 example, Fairy C language/20 and
SDK stage0.79 or later. All product
logic uses Fairy C and public SDK APIs. Package version 0.2.0 adds status filters
and title sorting to `list`. `std::bytes::equal` matches commands/header/options;
`std::bytes::compare` orders titles. The TSV format and prepared-file publication
protocol remain unchanged.

Commands: `<store> list [--status open|done|all] [--sort id|title]`, `<store> add "title"`, `<store> done <id>`,
`<store> delete <id>`, and `--help`. Limits: 256 tasks, 512 UTF-8 bytes per title,
262144 store bytes, 4000 UTF-8 path bytes. IDs increase and are never reused.

```powershell
fairyc build
.\build\task-manager-cli.exe .\tasks.tsv add "보고서 작성"
.\build\task-manager-cli.exe .\tasks.tsv list --status open --sort title
.\build\task-manager-cli.exe .\tasks.tsv list --sort title --status done
```

On Linux, build the compiler from this repository with Go (see
[Contributing](../../CONTRIBUTING.md)), then run from this example directory:

```sh
fairyc check
fairyc build --locked
./build/task-manager-cli ./tasks.tsv add "보고서 작성"
./build/task-manager-cli ./tasks.tsv list --status open --sort title
```

Use `fairyc build --release --locked` for the optimized image, or
`fairyc run --locked -- ./tasks.tsv list` for temporary execution. GCC is
preferred, with Clang used when GCC is absent. Native `test` remains Windows-only.

Default listing selects all tasks in ascending numeric ID order. Title order is
unsigned UTF-8 byte order, with ascending ID for equal titles; it does not apply
locale collation, case folding or Unicode normalization. Spaces and distinct
Unicode spellings are preserved. Output remains `ID<TAB>open|done<TAB>title<LF>`;
an empty store or filter with no matches prints `empty<LF>`. Both option orders
are accepted; each option may appear once. Missing/unknown options or values,
duplicates and options on mutation commands are usage errors (exit 2). Option
tokens/values are limited to 15 UTF-8 bytes. CLI options are checked before
opening the store. The entire store is validated before filtering/sorting, so a
corrupt excluded row still fails with exit 1 and no stdout. Listing retains the
existing cooperative lock/sidecar checks and leaves store bytes unchanged.

Saving acquires the cooperative exclusive `<store>.lock`, rejects pre-existing
`.tmp`/`.bak`, reads and validates the complete live TSV, prepares all new bytes,
exclusively creates the sibling `.tmp`, checks `write_all` and its completed count,
checks `sync` and consuming `close`, then calls `replace_file(tmp, live)` once.
Only after publication and lock cleanup does it emit success. The fixed staging
slot is owned by the exclusive lock; collision is an error, never overwrite.
There is no live-to-backup rename, rollback dance or missing-live interval in the
application protocol. Existing `.bak` is still reserved and refused to avoid
silently consuming a legacy interrupted save.

| Surviving state | Next run and disposition |
| --- | --- |
| Valid live only | Fully validate header/version, size, IDs, titles and LF layout, then operate |
| Live and `.tmp` | Refuse; retain both. A complete temp is not evidence of a committed save |
| Any `.lock` | Refuse, including list. Determine that every owner exited before manual recovery |
| Any legacy `.bak` | Refuse; preserve and inspect the legacy transaction |
| Malformed live | Refuse mutation; never reinterpret it as an empty store |

For manual recovery, first stop all writers, copy all surviving files to separate
names, and validate copies with `list`. Keep the live generation if valid. Move
reviewed sidecars outside the reserved names and remove only a confirmed stale
lock. Never promote `.tmp` solely because it exists. There is no automatic
recovery or automatic deletion of another execution's artifacts. After a crash
following replacement, a command may have committed without acknowledging it;
inspect the live state before retrying an `add`.

The directory must be trusted. Cooperative exclusion does not cover different
path aliases, hard links, outside editors or hostile directory changes. The
replacement supplies a new file object; previous ACLs/metadata are not merged.
`sync`/write-through does not guarantee hardware-independent power-loss durability.
FYTASK1 has no checksum or retained validated generation. A checksum/versioned
journal and physical power-cut validation remain separate work, not a claim made
by this example. Linux replacement uses one same-filesystem `rename`; open
handles keep the old inode and do not emulate Windows sharing denial. File sync
does not persist the parent directory entry. See
[ADR 0411](../../docs/adr/0411-public-windows-file-replacement.md) and
[ADR 0424](../../docs/adr/0424-linux-prepared-publication-and-process-arguments.md).

Run `CGO_ENABLED=0 go test -mod=readonly -run '^TestTaskManager' -v ./internal/cli`
from the repository root (PowerShell: set `$env:CGO_ENABLED='0'` first). Go builds
the current checked-in Fairy sources in temporary directories and executes
low/optimized CLI cases covering byte-order ties, distinct Unicode spellings,
512-byte titles, 256 records, invalid input/options, hidden corruption, restart,
ID reuse prevention, sidecar refusal and store preservation. There is no pinned
maintainer compiler, external fixture download, Bun or PowerShell prerequisite.
Windows storage-fault tests retain test-only interception of production Win32
I/O and bounded child kills; application logic still comes from Fairy source.

The [list feature receipt](../../docs/ops/evidence/task-manager-list-options-20261003.json)
is the unchanged historical stage0.78/196-case PowerShell result. It identifies
its original sources/compiler/images and is not a new SDK stage0.79 or Linux
receipt. The replacement Go suite consolidates cases rather than maintaining
that historical case count. Windows native Go tests have been executed locally;
Linux native tests are configured in existing Ubuntu CI but have not been
observed locally or remotely for this change. Physical power loss, hostile
multiwriter aliases and performance remain outside these tests.
