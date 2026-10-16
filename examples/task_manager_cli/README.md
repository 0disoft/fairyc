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
preferred, with Clang used when GCC is absent. Native tests use the same host backend.

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

Saving acquires the cooperative exclusive `<store>.lock`, verifies that no pre-existing
`.tmp` or `.bak` files exist, reads and validates the entire live TSV, and stages the updated content.
It creates a sibling `.tmp` file exclusively, performs `write_all`, `sync`, and a consuming `close`,
and then executes an atomic `replace_file(tmp, live)`.
Success is returned only after publication and lock cleanup. The staging slot is protected by the exclusive lock;
collisions result in an immediate error rather than overwriting existing data.
This atomic replacement avoids intermediate states or windows where the store file is absent.

| Surviving state | Next run behavior |
| --- | --- |
| Valid live only | Validates header, version, size, IDs, titles, and formatting, then proceeds normally |
| Live and `.tmp` | Aborts; retains both files. An existing `.tmp` indicates an incomplete write |
| Any `.lock` | Aborts operations (including `list`) until stale lock owners have exited |
| Any legacy `.bak` | Aborts; preserves `.bak` for manual inspection |
| Malformed live | Aborts mutations; never treats a corrupt file as an empty store |

For manual recovery: first ensure all writer processes have stopped, back up all surviving files,
and inspect the copies with `list`. Keep the live file if valid. Move any stray sidecars out of the
reserved names, and remove `.lock` only after verifying that no process holds it.
Do not automatically promote a `.tmp` file without verifying its integrity. Following an unexpected
crash, inspect the live store state before retrying write operations.

> **Note**: Cooperative locking assumes a trusted working directory and does not guard against external hard links or concurrent external editors. Atomic replacement supplies a new file object rather than preserving prior ACLs. See [ADR 0411](../../docs/adr/0411-public-windows-file-replacement.md) and [ADR 0424](../../docs/adr/0424-linux-prepared-publication-and-process-arguments.md).

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
