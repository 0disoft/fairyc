# Authored Windows state CLI preparation, 2026-11-07

Status: preparation, Python unit tests and coordinator low/optimized native
build/command integration verified. Historical worker-only preparation evidence
below is separate from the subsequent actual native observations.

## Usage and scope

Run from the repository root, choosing a previously unused output directory:

```powershell
python -B tools/prepare-candidate-state-cli.py --output .cache/state-cli-next --bootstrap .cache/fairyc-bootstrap-state-graph-cache.exe
```

`--bootstrap` is explicit and required. Preparation hashes the executable but
does not execute it, infer its capabilities, or build a replacement. The tool
has no `--build` mode. It prepares an isolated workspace and records exact build
argv and cwd in `prepare-manifest.json`.

The tool copies the authored `examples/lexer_candidate/fairy.toml` and `src/`
tree. Its isolated `src/main.fy` receives the exact bytes of
`tools/dev/fairyc-state/main.fy` instead of the example dispatcher. No source
text is rewritten, and authored input files remain unchanged. The original
example main remains represented in the input inventory, while `copiedFiles`
identifies the developer entry actually installed in the prepared workspace.

`fairy.lock` is not copied. The manifest's `package.source = "src"` remains
unchanged, and SDK selection belongs to the explicit bootstrap's embedded
stdlib and loader. No SDK source overlay, dependency installation, internet
access, Go execution fallback, or substitute product executor is added.

The existing runtime/frame packagers are loaded explicitly with importlib and
executed from their inventoried Python source bytes, without using cached
bytecode. Their `pack_and_pin()` functions generate the packs, and the existing
pin comparison policies must pass. No pin update is performed. The generated
files are executable siblings at:

- `build/fairyc-state-graph-runtime.pack`
- `build/fairyc-state-graph-frame.pack`

Output must be a newly created directory strictly below this repository's
`.cache/`. Existing outputs, parent traversal, output escape, symlinks and
Windows reparse points/junctions are refused. Files are created exclusively;
the tool does not overwrite or recursively delete an earlier workspace.
Failure can leave partial output for inspection; such a directory must not be
reused. Authored inputs and bootstrap bytes are reread before the successful
manifest is written, so detected concurrent changes do not produce a success
manifest. This is a byte snapshot, not a claim about a Git commit.

## Build commands are alternatives

The manifest records both `low` and `optimized` for target
`x86_64-windows-msvc`. Choose one profile for this prepared workspace. Both
commands deliberately name the same output; they are alternatives, not an
instruction to run two builds or overwrite a previously verified executable.
Use separate fresh workspaces when retaining both profiles.

From the manifest's workspace cwd, the command shape is:

```powershell
& '<absolute explicit bootstrap path>' experimental build-workspace --target x86_64-windows-msvc --profile low --output build/fairyc-state.exe
```

The optimized alternative replaces `low` with `optimized`. No `--locked` is
recorded for the fresh workspace without a lock. Any later lock creation is
confined to the isolated workspace. The command syntax and `build/` output
policy follow the repository's existing experimental build-workspace command;
recording this command does not verify that a particular bootstrap can build
the complete current sources.

## Preparation evidence

Executed preparation command:

```powershell
python -B tools/prepare-candidate-state-cli.py --output .cache/state-cli-prepared-20261107-worker02 --bootstrap .cache/fairyc-bootstrap-state-graph-cache.exe
```

It exited 0. Evidence files:

- `.cache/prepare-state-cli-workspace-02.log`
- `.cache/state-cli-prepared-20261107-worker02/prepare-manifest.json`

The manifest reports `prepared-not-built`, **748 copied files**, **757 input
files**, and **2 packs**. Each input and copied file has its byte length and
SHA-256; each copied row also names its source and destination. Pack hashes,
bootstrap identity, target, alternative profiles, exact argv and cwd are
recorded separately. Both build command records have `executed: false`.

Recorded developer entry (`src/main.fy`), 155 bytes:

```text
a3f36e25e8ded35df9d320dd6245e9f6f2d844036fedfe2525984f0756a8a9d5
```

Recorded bootstrap, `.cache/fairyc-bootstrap-state-graph-cache.exe`,
24,790,016 bytes:

```text
4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084
```

Recorded runtime pack, 37,346 bytes:

```text
726db6eb5ac26d437e829170a3709bf3f412b0974f7b057bb1cbad5f1837c1a2
```

Recorded frame pack, 8,284 bytes:

```text
320ac1faf4137b3915ebc7c4c48505b55ed56b149a0d0ba2d86df595cc761335
```

These identities describe worker02's recorded snapshot. A fresh bootstrap or
later authored source changes require a new preparation directory and manifest;
they do not retroactively change this evidence.

## Unit-test evidence

```powershell
python -B -m unittest discover -s tools -p test_prepare_candidate_state_cli.py -v
```

**9 tests passed in 0.722s**, with no skips, recorded in
`.cache/prepare-state-cli-unit-02.log`. Tests use temporary injected checkouts
with the real packagers, runtime/frame sources and pins. No Go compiler, C
compiler or product executable is run.

Covered checks: fresh exact copies and entry replacement; inventory and pack
hashes; sibling filenames and exact build argv; authored source immutability;
refusal of reused directories and existing output files; output escape, source
overwrite, cache-root output and parent traversal; pin drift; manifest
source/dependency drift; concurrent source-change detection; Windows reparse
attribute refusal; and actual filesystem symlink refusal. The reparse test
injects the junction attribute; it does not claim an actual junction fixture.

## Native support connection and failure evidence

The initial prepared build fails because `nativesupport.Options.CandidateHost`
only connected Linux adapters: Windows `fairyc_private_windows_assign_v1`
incorrectly reaches closed standard-library selection. This is an actual native
build failure, retained in `.cache/state-cli-console-build-final-low.json`.
Earlier source-subset failures are retained separately in
`.cache/state-cli-console-build-low.json`. Neither is counted as a pass.

ADR 0545 connects the existing exact Windows allowlist only for Windows opt-in.
Complete private spellings are validated before adapting them to C declarations;
unknown/injected/bare-prefix/wrong-target symbols fail. Existing ordinary,
vector/trap source order and caller storage are preserved. Unit selection checks
pass in **0.043s** and existing private host selectors pass in **0.043s**:

```powershell
go test -mod=readonly ./internal/nativesupport ./internal/candidatehost -run 'TestWindowsCandidateHost|TestLinuxCandidateHost|TestPrivateWindows' -count=1 -v
```

Log `.cache/nativesupport-windows-host-unit-01.log`. Linux checks here are source
selection checks on Windows, not Linux compilation or execution.

## Actual prepared native builds

The coordinator builds the current bootstrap explicitly:

```powershell
go build -mod=readonly -o .cache/fairyc-bootstrap-state-host.exe ./cmd/fairyc
```

Bootstrap: 24,839,168 bytes, SHA-256
`e85bfc031c24b7e5c9a59da0f7c4f8a5e1140a9017d669c135557ae9015ca36f`.
Two fresh manifests retain independent source/pack/bootstrap snapshots:

- `.cache/state-cli-host-low/prepare-manifest.json`
- `.cache/state-cli-host-optimized/prepare-manifest.json`

The recorded low/optimized build argv are executed from their respective
workspace cwd using that explicit bootstrap. Both commands exit 0 with empty
stderr and an actual MSVC 14.51.36231 `built workspace` response. No warning or
deadline policy is relaxed. Build records contain exact argv/cwd/stdout/stderr,
elapsed time and output hash:

| Profile | Full build elapsed | Image bytes | Image SHA-256 |
| --- | --- | --- | --- |
| low | 19.184s | 2,732,544 | `632a7a641c07f9ffe0ced59be4d6b9fdba84386607a34eb626e12930fed70e5b` |
| optimized | 28.118s | 1,119,232 | `5d70051706070d91d0250ec91a1f2603c41a66368332900ca3de69378a20434c` |

Records `.cache/state-cli-host-build-low.json` and
`.cache/state-cli-host-build-optimized.json`; corresponding `.log` files retain
the complete observations. The preparation manifests remain preparation records
with `executed:false`; these separate build records establish actual execution.

## Executing the resulting files

Both prepared executables run real help and init/snapshot/execute/final-snapshot
on an authored `ledger.fy` with one u32 cell and read-after-write transaction.
An independently built current `./tools/dev/fairyc-state` Go command receives the
same source/profile and separate real stores. All **eight paired command
observations / sixteen processes** have exit 0, empty stderr, byte-identical JSON
and byte-identical durable state after every action. The low and optimized
images use their corresponding native execution profiles. This supplements,
rather than replaces, the six-fixture command matrix in the console receipt.

Logs `.cache/state-cli-prepared-smoke-low.log`,
`.cache/state-cli-prepared-smoke-optimized.log`; full argv/stdout/stderr records
`.cache/state-cli-prepared-smoke/observations.json` and
`optimized-observations.json`. Both independent final durable records hash to
`c93b2ec4a66636ea6becb49dc3b3eb634766752f7a10d13954b8cf11ad744a2a`.

The coordinator verifies that every recorded source/copy/pack/bootstrap hash
still matches current bytes after builds. Scoped vet, formatting and diff hygiene
pass. Native Linux, installed/clean-machine distribution and final full compiler
Go/Fairy/self-build/hosted acceptance remain open. Private Windows support
selection changes as described above; public CLI syntax/SDK/native ABI, DB/provider,
dependencies, repository policy and CI/runners are unchanged. No push, dispatch,
release or deployment.
