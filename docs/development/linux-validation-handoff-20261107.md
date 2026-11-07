# Clean committed Linux validation handoff, 2026-11-07

`tools/package-linux-validation-handoff.py` builds a local source ZIP and an
**unexecuted** Linux validation plan. It does not run Go, a compiler, the native
collector, an installer, Linux, hosted CI, push, dispatch or release. This worker
adds only the builder, its 15 portable unit tests and this note. The coordinator
owns commits, the shared queue/index and the actual clean-snapshot bundle.

## Coordinator invocation after all changes are committed

From a clean repository root, using a fresh output name:

```text
python -B tools/package-linux-validation-handoff.py --output .cache/linux-validation-handoff-20261107
```

`--timeout-seconds` configures the future per-group Go deadline (1..3600, default
3600); it does not execute tests. Output is a new Git-ignored directory below
this repository's `.cache`, containing `handoff.zip` and `handoff.zip.sha256`.
Stdout reports the source revision/tree, source file count, ZIP entry count,
byte size, SHA-256 and `linuxNativeExecuted: false`. Existing directories/files
are never reused or overwritten. No actual repository bundle has been built
by this worker; tiny test repositories are the only packaging inputs used here.

## Snapshot and archive boundaries

The builder refuses dirty tracked/staged files, nonignored untracked files,
assume-unchanged and skip-worktree entries. It reads the full committed Git tree
and `git archive`, never local source/cache bytes. Every archived file must match
the Git tree's blob hash before packaging. This preserves committed LF bytes
despite a clean Windows CRLF worktree. Archive omission/substitution (including
export attributes or EOL attributes that actually transform archive bytes) is
refused instead of silently claiming an exact snapshot. HEAD/cleanliness is
checked again before and after writing the output.

All tracked files must be admitted; there is no silent partial source export.
Tracked caches/builds, node_modules, toolchain directories, offline/archive
payloads, binary executables and credential filenames are refused, as are known
private-key/token markers. This is a bounded artifact/credential guard, not a
general secret audit. Git metadata, ignored local files, environment variables,
toolchains and dependency caches are never bundled. Author-written source and
historical documents stay byte-exact; generated metadata contains no local
absolute checkout/staging paths.

Git symlinks, submodules, tar links/special files, duplicate/case-aliased paths,
absolute paths, traversal, backslash/drive/ADS paths, noncanonical names and
Windows device names are rejected. Tar members are read without extractall.
Output ancestors are checked for symlinks/reparse points, directories are
created one component at a time, output identity is rechecked and leaf creation
is exclusive. Cleanup only removes this attempt's own leaves, never a foreign
file encountered during exclusive creation. These are local packaging guards,
not transactional isolation against hostile concurrent filesystem replacement.

## ZIP payload and replay

The ZIP has a single `fairyc-linux-validation/` root:

- `source/`: all admitted tracked regular blobs and their original Git modes.
- `plan.json`: the committed collector's `make_plan` output with portable cwd
  `.` relative to `source/`, relative owner paths and existing anchored argv.
- `remaining-observations.json`: explicit unexecuted observations and links to
  the exact included Linux authored receipts.
- `snapshot.commit`: the original raw Git commit object, without a Git directory.
- `README.md`: checksum, prerequisites, Git reconstruction and concrete plan /
  later authorized native replay commands.
- `manifest.json`: revision, tree, object format, source count/bytes and each
  source's Git blob/mode plus payload SHA-256/size. Its `files` excludes the
  manifest and `SHA256SUMS` to avoid recursive hashes.
- `SHA256SUMS`: checks every other payload, including `manifest.json`.

ZIP entries are sorted, have a fixed timestamp and Unix file modes, and use
standard DEFLATE level 6 to reduce source transfer size while preserving every
payload byte. Identical revision, timeout and Python/zlib implementation produce
identical ZIP bytes, counts and hashes; different compression implementations
may produce different ZIP bytes with the same verified source contents.
The external SHA-256 sidecar covers the entire ZIP.

The receiver must use a new extraction directory and verify the fixed ZIP and
payload checksums. Python ZipFile extraction does not automatically preserve
Unix executable permissions. The README's reconstruction code verifies all
manifest paths, bytes and regular-file/single-link facts before applying
`os.chmod` to its own extracted source files: `100644` becomes `0644` and
`100755` becomes `0755`. Symlinks/reparse points, hard links and outside paths
are refused before any chmod. This is only for this verified, link-free ZIP.

The same code initializes fresh local Git metadata, rebuilds the index using
the manifest's original modes/blob IDs, checks the exact tree, imports the raw
commit and restores HEAD to the original SHA. It checks source cleanliness.
This lets the existing collector's Git probes retain the original snapshot
identity without inventing a synthetic validation commit. Ancestor history is
not supplied. Restoration changes only the extracted receiver tree; it is not
run against the authored repository.

Planning imports the committed collector from verified temporary source under
the new output directory and invokes only `make_plan`. The selected groups are
`candidatebuild`, `trap-runtime`, `runtime`, `fetch-host`, `fetch-native`, `cli`.
The exact current group/root ownership remains in
`docs/development/linux-native-batch-20261107.md`; no root or tested status is
invented. The README includes the explicit later `--execute` command, but the
builder never calls that execution path. The `cli` group uses the existing GCC
low/optimized full entry, not an invented full-CLI GCC/Clang matrix.

Linux runtime observations remain pending: positive trusted TLS with real
libcurl/CA provisioning; Atlas CLI/MCP stdio, default SIGPIPE and closed-pipe
behavior; real cold/warm cache and offline publication; consecutive full Linux
self-build and broad/full acceptance. Existing bounded groups cover authored
portions only. Collector completeness never closes those wider gates, and
historical Windows receipts do not become current Linux passes.

## Windows validation

```text
python -B -m unittest discover -s tools -p test_package_linux_validation_handoff.py -v
```

Final Windows run: **15 tests pass in 52.092s**, exit 0. Both Python sources
pass syntax parsing; all three authored files pass UTF-8/no-BOM, LF/final-newline
and scoped `git diff --no-index --check` checks. There is no known test blocker.
The tests use tiny temp Git repositories and the existing collector with tiny declaration fixtures;
they invoke Git/Python only. They verify byte integrity (including binary and
CRLF worktree cases), deterministic ZIP/hash/count, exact six-group roots/argv,
dirty/index refusal, unsafe source/tar/output refusal, export-attribute refusal,
credential/artifact refusal, moving HEAD cleanup, foreign-leaf preservation,
portable metadata, executable ZIP/Git modes, README Git restoration, and
path/link refusal before chmod. Windows chmod call/mode/index assertions do not
claim native Linux permission or process behavior.

No source-of-truth contract, collector, queue, index, ADR, product API, DB,
dependency, CI workflow or runner is changed. Native Linux execution and actual
repository ZIP/blob/hash/receiver reconstruction remain the coordinator's
separate authorized batch. This worker makes no commit or remote operation.

## Coordinator transfer-size correction

The first real clean-source ZIP at `06d5bb23` contains 7,547 committed files
and is 217,487,231 bytes with stored entries. The coordinator changes only ZIP
encoding to standard DEFLATE level 6, retaining exact payload/blob hashes,
fixed metadata and original file modes. A new repetitive-source test verifies
over 90% size reduction, exact decoded bytes, modes, CRC and repeatability.
The new compression test plus existing complete-blob/determinism and README
original-commit restoration tests pass: **three checks in 10.027s**, no skips,
log `.cache/linux-handoff-compression-tests.log`. Other snapshot/plan/safety
logic is unchanged from the 15-check worker run. The final compressed repository
ZIP and receiver verification are recorded separately after the clean commit.

## Actual committed repository handoff and receiver

The coordinator builds the final compressed snapshot from clean commit
`14862b709530b750230d8bb25b6001e6f3bebfe9`, tree
`981d9a0318e028093ac63b28dd65343ee9c94e24`:

```text
python -B tools/package-linux-validation-handoff.py --output .cache/linux-validation-handoff-compressed-20261107
```

Exit 0. The ZIP contains **7,547 exact committed source blobs / 7,553 entries**,
is **60,890,523 bytes**, about **72% smaller** than the first stored archive.
SHA-256 `d960553fc4c0cb3cdd33bb89d4e263a54a8a9dcbe9d8fbeb61b0fcca1df97bfe`.
Artifact `.cache/linux-validation-handoff-compressed-20261107/handoff.zip` and
its adjacent checksum file. Build log
`.cache/linux-validation-handoff-compressed-build.log` records exact revision,
tree, sizes and the explicit `linuxNativeExecuted:false`.

The coordinator independently checks ZIP member/path/mode facts, every payload
checksum and each source's original Git blob hash, then extracts into a new
`.cache/linux-handoff-extracted-20261107` receiver. Executing only the README's
Git/Python restoration code reproduces exact original HEAD/tree and a clean
source tree. The extracted collector runs in plan mode only; its six groups,
source-owner paths and exact command argv match the bundled portable plan.
The receiver stays clean. No Go command, compiler, native Linux test or network
installation is invoked. Log `.cache/linux-handoff-receiver-verification.log`;
record `receiver-verification.json` beside the final ZIP.

This subsequent documentation receipt does not change the archived snapshot or
its commit identity. The transferable handoff requirement is closed; Linux
execution, trusted TLS and consecutive self-build/full acceptance remain open.
No product API/ABI, DB/provider, dependency, CI/runner or repository-policy
change, push, dispatch, release or deployment is introduced.

## Current source handoff at 6270e367

The source-only handoff is refreshed at clean
`6270e3673e8a5551a927422550a853c3cbb109e9`, original Git tree
`3c0757814438faf56e07e69994eb8da5b3aa3113`. The ZIP contains 7577 original
source files and is 60976825 bytes. Artifact:
`.cache/current-linux-handoff-6270e367-20261107/handoff.zip`, SHA-256
`dcd1268372efd85b02892bfcf480f0a9396bf5783a10b5cfe6a38b02448ad4a6`.

The existing packager and a fresh receiver verification each run once. All
payload checksums, original blobs and file modes match; the receiver restores
the exact commit/tree and remains clean. The portable collector's six groups
and exact argv match the bundled plan. The original repository HEAD, tree,
index and source bytes are unchanged. Evidence is `handoff-verification.json`
and `receiver-verification.log` beside the artifact.

No Go/compiler/native Linux execution, CI dispatch, installation or network
download is part of this source-package proof. Unchanged tiny packager checks
reuse their previous valid results. Native Linux and authored self-build/fixed-
point remain pending; the later documentation commit does not change this
snapshot's identity.
