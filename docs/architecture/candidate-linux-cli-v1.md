# Private Linux candidate file CLI v1

`linux_file_cli.fy` is the Linux command dispatcher. The preparation tool selects the dispatcher and
[workspace filesystem boundary](candidate-linux-workspace-check-v1.md) and test executor in an isolated copy of the authored sources. The original
Windows entry and no-argument binary stdin protocol remain available.

## Connected commands

- `--help`, `-h`, `help`, `--version`, and `targets` take no extra arguments.
- `--tokens-file`, `--declarations-file`, `--check-names-file`, `--bindings-file`,
  `--check-types-file`, `--hir-file`, and `--mir-file` take exactly one source path.
- `check`, `check --locked`, and `check --help` use the shared workspace checker.
  `--cache-root <existing directory>` and `--cache-status` select the
  [Linux workspace cache](candidate-linux-cache-v1.md).
- `build [--release] [--locked]` and build help use the [Linux bootstrap backend](candidate-linux-build-v1.md), with optional `--cache-root <existing directory>` and `--cache-status`.
- `run [--release] [--locked] [-- <program args>]` uses the [Linux run path](candidate-linux-run-v1.md), with the same optional cache flags before `--`.
- `test` reuses shared discovery, listing and interpreter execution; `--native` uses the [Linux test path](candidate-linux-test-v1.md), including stdin modes and JSON capture. Both support opt-in `--cache-root <existing directory>` and `--cache-status` before any program delimiter.
- `new <package-name> --uid <package-uid>` uses the [Linux starter path](candidate-linux-new-v1.md).
- `--c-file <path>` emits the shared scalar/nominal C11 transport after Linux source loading.
- `--c-workspace <root> [--locked]` emits a Linux-targeted private
  [partitioned C packet](candidate-linux-c-emission-v1.md).
- Unknown commands or invalid arity for connected commands return 2.
  Dispatcher output failures return 4; shared frontend exit codes remain unchanged.
  The help lists this boundary; the shared target registry is
  target metadata, not a claim that this dispatcher can build either target.

`file_frontend_cli.fy` shares the existing scanner/name/type/HIR/MIR dispatch with the
Windows CLI after host-owned input loading. It does not select a host or link MSVC.
The shared C mode also serves the Linux single-file command. Workspace C uses the
same Linux SDK wire selection and target marker as workspace build. These private
debugging commands do not add a stable product `emit` command.

## Input and native support

`linux_cli_file.load` validates UTF-8 paths of 1..4095 bytes, opens the parent through
`linux_rooted_file`, and opens one non-symlink regular leaf (1..255 bytes). The rooted
adapter's directory-prefix rules still apply. Root `/`, a relative leaf, and nested
paths use the same descriptor-owned policy. Missing paths, directories, symlinks and
failed metadata/close operations fail the input operation.

Reads admit at most 65,536 source bytes, reserve one overflow byte, and issue at most
65,536 bytes per native read. Before/after size, device, inode and modification time
must agree, and actual byte count must equal the final size. These observations detect
ordinary changes; they are not an atomic snapshot against a writer that restores metadata.
Descriptors and buffer storage retain normal resource cleanup on every early return.

`nativesupport.Options.CandidateHost` explicitly selects the closed Linux adapters from
the emitted foreign-symbol inventory. Workspace native compilation enables this option;
ordinary standard-library selection remains unchanged. Unknown private Linux symbols and
private Windows imports fail support selection. No shell, Go frontend subprocess, MSVC
toolchain or Windows runtime asset pack is invoked by the authored Linux dispatcher.

## Preparing and building

On a supported Linux amd64 host, from the repository root:

```sh
go build -o .cache/fairyc-bootstrap ./cmd/fairyc
python3 tools/prepare-candidate-linux-cli.py --output .cache/linux-file-cli
cd .cache/linux-file-cli
../fairyc-bootstrap experimental build-workspace --target x86_64-linux-gnu --cc gcc --profile low --output build/lexer-candidate-linux
./build/lexer-candidate-linux --help
./build/lexer-candidate-linux --tokens-file src/linux_file_cli.fy
```

The preparation tool copies authored sources into a new directory under repository
`.cache`, refuses an existing destination, and never edits the original workspace.
Use the native [Linux toolchain profile](../abi/linux-toolchain-v0.md), including its
glibc and compiler requirements. The Go compiler is an explicit bootstrap builder here;
this is not yet a Linux candidate self-build or distribution procedure.

## Evidence and next boundary

`TestLinuxFileCLI` checks the complete entry, emits Linux GNU and Windows MSVC C, rejects
any Windows foreign dependency, and selects the closed Linux support. A separate small
entry executes argument classification on the available native host in low/optimized
profiles. Linux-only integration cases build and execute the full entry, covering help,
version, targets, source checking, token output, missing/oversize/nonregular/symlink input,
invalid invocation and unsupported workspace commands.

On the Windows development host, complete both-target emission and Linux support
selection pass; portable dispatch execution passes with MSVC low/optimized. Full Linux
execution is skipped, not passed. The owning source workspace passes `check` and
`check --locked`; its lock is refreshed. No hosted runner or distribution is changed.

Linux [workspace check](candidate-linux-workspace-check-v1.md) and
[workspace build](candidate-linux-build-v1.md) share frontend logic and private Linux
adapters. Build requires the matching Go bootstrap backend companion. Workspace cache
integration and actual Linux execution remain separate; Go stays the default compiler.
