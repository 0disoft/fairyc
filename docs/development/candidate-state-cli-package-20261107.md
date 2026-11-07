# Private Windows state CLI developer ZIP, 2026-11-07

Status: local packaging and Python checks verified for the coordinator's existing
low and optimized native build records. No image is executed by this packaging
work. Preparation and private native command scope remain owned by
[ADR 0545](../adr/0545-candidate-state-cli-developer-build.md) and
[the preparation evidence](candidate-state-cli-preparation-20261107.md).

## Usage

From the repository root, choose a new ZIP filename outside the prepared
workspace, strictly below repository `.cache`:

```powershell
python -B tools/package-candidate-state-cli.py --workspace .cache/state-cli-host-low --build-record .cache/state-cli-host-build-low.json --output .cache/state-cli-private-low-next.zip
```

The tool reads an existing `prepare-manifest.json` and the coordinator's JSON
build record. It does not invoke a native compiler, execute the resulting image,
access the network, invoke Git, install dependencies, regenerate pins or modify
authored/prepared sources. `--workspace`, `--build-record` and `--output` are
required. The workspace must be below `.cache`; the output must be a fresh
`.zip` below `.cache` and outside that workspace. The record is an explicit
regular-file input. Existing output files, parent traversal and observed
symlinks/reparse points, including lexical ancestors, are refused. This uses
the preparer's existing local-file policy, not a new adversarial filesystem
sandbox. Failure during ZIP writing can leave a partial file for inspection;
choose another output name rather than overwrite or reuse it.

## Validation before writing

The package operation checks the preparation schema, location, Windows target,
alternative profiles, entry mapping and unchanged-source/embedded-SDK policy.
It reconstructs the complete current authored input inventory with the preparer's
`inventory()` helper and requires exact path, byte count and SHA-256 rows.
It reconstructs the exact copy mapping, checks all copied bytes and refuses
additional copied source files. The bootstrap is reread and must match its
explicit absolute prepared path, byte count and hash.

Both alternative prepared build commands must match the fixed existing command
shape. The record's actual argv must select exactly one of those commands and
its cwd must equal the prepared workspace. Thus the selected profile, Windows
target and `--output build/fairyc-state.exe` must match together. If a record
also supplies `profile` or `output`, those fields must agree. The existing
coordinator record requires integer `exit: 0`, empty `stderr`, text `stdout`,
and an `image` object with the actual byte count and SHA-256. Optional elapsed
time and stdout text do not substitute for those checks.

The image must also have bounded DOS/PE headers, AMD64 machine type, PE32+
console executable flags, bounded section data and an entry point inside a
file-backed executable section. This is structural validation, not loader or
runtime acceptance.

The two live state graph packagers regenerate bytes in memory from their
inventoried authored source. Both current pin policies must pass. Their generated
bytes must match both the preparation pack rows and the actual sibling packs.
No package or pin is rewritten. Inputs, bootstrap, manifest, record, copied files,
image and packs are reread before output creation; detected changes fail closed.
These hashes authenticate a current byte snapshot, not a commit or a signed
attestation of the coordinator's execution.

## Archive contract

The ZIP contains exactly seven root-level regular-file entries:

- `fairyc-state.exe`
- `fairyc-state-graph-runtime.pack`
- `fairyc-state-graph-frame.pack`
- `README.md`
- `LICENSE`
- `metadata.json`
- `SHA256SUMS`

Executable and pack bytes are copied unchanged. `LICENSE` is copied byte for
byte from the current repository. README instructions use sibling/relative
paths. The executable and both packs must stay together. The archive does not
bundle a native toolchain or dependencies needed by the private native graph
command.

Metadata retains relative authored input/copy paths, their hashes, bootstrap
identity, selected profile, target and payload identities. It projects only
validated record fields; raw preparation/build records, absolute local paths,
stdout and arbitrary extra record fields are not copied into JSON. The displayed
build argv uses `bootstrap.exe` as an explicit identity placeholder and `.` for
the workspace cwd; it is not an installed-bootstrap path or a command to replay
from the extracted ZIP. Exact original argv/cwd remain in the local input record.
Binary bytes are never edited to remove embedded compiler data.

`SHA256SUMS` covers the other six entries, including metadata; it does not hash
itself. Entries are sorted, stored without compression, timestamped
`1980-01-01 00:00:00`, and have fixed Unix regular-file mode `0644`, with no
extra fields or comments. Identical validated payloads produce identical ZIP
bytes independent of output filename or input file modification timestamps.

## Executed checks and artifacts

```powershell
python -B -m unittest discover -s tools -p 'test_*candidate_state_cli.py' -v
```

**30 tests passed in 5.855s, no skips**: 21 packaging tests and the existing
9 preparation tests. Tests reuse the real runtime/frame packagers and pins in
temporary tiny checkouts. Their structural PE fixture is never executed and is
not presented as native-build evidence. Coverage includes stale/added/removed
authored inputs, tampered manifests/copies/bootstrap, both profiles, mismatched
cwd/argv/profile/output/exit/stderr/image identity, malformed PE headers,
pack/hash/pin drift, existing files and existing ZIPs, unsafe paths, actual
symlinks and injected Windows reparse attributes, concurrent changes, exact
archive bytes, hashes, relative metadata and deterministic repeat output.
Scoped Python syntax, UTF-8/LF, final-newline and trailing-whitespace checks
also passed; each of the three authored files is below 40 KB.

The following commands each exited 0 against the coordinator's already-built
workspaces and records:

```powershell
python -B tools/package-candidate-state-cli.py --workspace .cache/state-cli-host-low --build-record .cache/state-cli-host-build-low.json --output .cache/state-cli-private-low-package-worker-20261107-01.zip
python -B tools/package-candidate-state-cli.py --workspace .cache/state-cli-host-optimized --build-record .cache/state-cli-host-build-optimized.json --output .cache/state-cli-private-optimized-package-worker-20261107-01.zip
```

| ZIP profile | ZIP bytes | ZIP SHA-256 |
| --- | --- | --- |
| low | 3,117,629 | `62d54c26712db6577b0ccc6c5396299480a88c706665d502d3feceb9478e8f31` |
| optimized | 1,504,329 | `b3cca1dee7819e85ecc41bb619b570986828b5dc6e964c38041c919a3eaf9542` |

Independent read-only ZIP inspection passed CRC checks, the seven-member set,
all recorded checksums, selected profile, local-path exclusion in metadata and
byte-for-byte equality of each executable/pack with its prepared build output.
The retained image hashes are low
`632a7a641c07f9ffe0ced59be4d6b9fdba84386607a34eb626e12930fed70e5b`
and optimized
`5d70051706070d91d0250ec91a1f2603c41a66368332900ca3de69378a20434c`.

The worker did not rebuild or execute images; subsequent coordinator extracted
execution is recorded below. Linux native, installed/clean-machine and
hosted acceptance remain unverified by packaging. Public CLI/API, native ABI,
DB/provider, dependencies, CI/runners and repository policy are unchanged.
The only authored changes are the packaging script, its test and this evidence
document. ADRs, indexes and queues are not changed by this worker. No Git command,
commit, push, dispatch, release or deployment is performed.

## Coordinator extracted-image execution

Both actual ZIPs are independently checked for their exact seven-member set,
six payload checksums, target/profile and local-path-free metadata before any
extraction. Exact bytes are written into new `.cache/state-cli-zip-extracted-low`
and `.cache/state-cli-zip-extracted-optimized` directories. The coordinator runs
the extracted binaries from these new directories, preserving executable/pack
sibling placement rather than running the original prepared image.

Both actual help processes and **eight paired init/snapshot/execute/final-snapshot
observations / sixteen processes** pass against the independent Go native
command. Each profile uses its corresponding native graph execution profile.
Exit codes are 0, stderr is empty, JSON and durable bytes are exact after every
action. The final durable hash for both profiles is
`c93b2ec4a66636ea6becb49dc3b3eb634766752f7a10d13954b8cf11ad744a2a`.
Logs `.cache/state-cli-extracted-low.log`,
`.cache/state-cli-extracted-optimized.log`; each extracted directory retains
full command/exit/stdout/stderr observations in `observations.json`.

This closes local extracted developer-ZIP execution, not clean-machine or public
compiler distribution acceptance. No native rebuild is needed: the exact image
and authored inputs remain unchanged from the verified preparation receipt.
Coordinator updates ADR 0545, source index and implementation queue and makes
the scoped local commit; no push/dispatch/release/deployment is performed.
