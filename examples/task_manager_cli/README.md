# Task manager with prepared-file replacement

Windows AMD64 example, Fairy C language/20 and SDK stage0.72. All product logic
uses Fairy C and public SDK APIs. `fairy.toml` version 0.1.1 preserves the historical
FYTASK1 TSV format and CLI arguments; storage diagnostics and help describe the new
publication protocol. The original nine downloaded workloads remain unchanged.

Commands: `<store> list`, `<store> add "title"`, `<store> done <id>`,
`<store> delete <id>`, and `--help`. Limits: 256 tasks, 512 UTF-8 bytes per title,
262144 store bytes, 4000 UTF-8 path bytes. IDs increase and are never reused.

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
by this example. See [ADR 0411](../../docs/adr/0411-public-windows-file-replacement.md).

`verify.ps1 -Fairyc <pinned stage0.72 compiler>` runs the preserved 118 development
and release CLI cases. Only help and replacement failure expectations differ from
the original oracle. Run in an isolated copy: fixtures and raw results are written
there. Native storage-fault tests exercise compiler-emitted Fairy application
logic with test-only interception of production Win32 I/O, not a C rewrite of the
application. Evidence and limitations are recorded in the workload report.
