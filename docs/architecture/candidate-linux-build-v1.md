# Private Linux candidate workspace build v1

## Authored GNU companion

This section describes the Fairy-authored GNU companion introduced by code commit
`13243c0d`. It is implementation evidence, not a completed target or distribution claim.
The existing Go bootstrap companion remains the default and retains its original owners.

### Source ownership

- `examples/lexer_candidate/src/linux_gnu_build.fy` owns toolchain admission, source and
  object planning, compiler invocations, ELF admission and publication orchestration.
- `linux_gnu_policy.fy`, `linux_gnu_host.fy`, `linux_elf.fy`, `linux_runtime.fy` and
  `linux_runtime_contract.fy` own their named policy, host boundary, object inspection,
  runtime selection and target-marker checks.
- `linux_runtime_assets.fy` validates and selects the pinned runtime asset pack. Its
  generated `linux_runtime_asset_pin.fy` records the exact source count, size and digest.
- `internal/candidatehost/linux_gnu.c` is the Linux OS adapter. It performs POSIX
  marshalling only; Fairy owns source selection, compiler flags, invocation order and
  artifact admission.
- `linux_gnu_companion.fy` owns the private CLI entry, argument admission, packet and
  runtime-pack loading, and build/publication dispatch. `linux_companion_input.fy`
  supplies bounded reads through the retained Linux directory chain.

### Runtime assets and plan selection

`tools/package-candidate-linux-runtime.py` reads 33 authoritative raw C sources from the
repository and verifies their exact bytes while producing a 176,611-byte asset pack. The
loader accepts at most 1 MiB, checks the generated pin, and assembles at most 8 MiB of
runtime C. Windows dependencies and unknown target markers fail admission.

The authored selector is compared with the live Go standard-library contract and Linux
host-source inventory by `TestLinuxGNUCompanionRuntime`. The current focused checks cover
120 plans and 80 complete runtime C outputs in low/optimized profiles. Both-target emission
and Windows MSVC checks pass. These are Windows-development observations, not Linux-native
execution.

The package-directory/file, monotonic-clock and HTTPS protocol/worker/reader/parent
sources are included. HTTPS selects its clock dependency and `_GNU_SOURCE` before
all libc headers; package sources follow rooted/directory sources, matching the Go
assembly byte for byte. Edit-file support follows package-file and retains its
rooted prerequisite. Its native implementation/Windows-model boundary is recorded in
`docs/development/candidate-linux-atlas-edit-20261106.md`; Linux edit selection
and execution remain open. The pin rejects earlier packs. Current synchronization and
remaining native gates are recorded in
`docs/development/candidate-linux-runtime-sync-20261106.md`. Compiling HTTPS support
requires the optional libcurl development headers from ADR 0501; runtime/CA
provisioning and trusted Linux download acceptance remain unverified.

### Toolchain and build contract

Admission requires native Linux amd64 with glibc >= 2.39 and GCC >= 13 or Clang >= 18.
Compiler `auto` tries an admitted GCC and then an admitted Clang. A requested
`--cc gcc|clang` is exact. The companion never invokes or falls back to the Go frontend.

The input limits are 96 MiB for the packet and supplied sources, 8 MiB for assembled
runtime C and 64 MiB for a published output. Each compiler invocation and the link have a
separate 60-second deadline. Generated sources use `-std=c11 -fno-common`,
`-O0`/`-O2`, `-pipe` and `-c`. Every object is inspected as ELF64 x86-64 before link.
Linking passes only `@fy_objects.rsp -o fy_program`; C compilation flags are not repeated
at link time.

When process arguments are selected, a separate startup object initializes the actual
process `argc`/`argv` before calling the candidate entry. The packet and target-neutral
entry bytes are not rewritten for other programs.

### Stage, process and publication boundary

The caller supplies an absolute mode-0700 stage containing only `packet.bin`. The stage
descriptor is read-only. Writes are descriptor-relative and exclusive; reads use
`O_NOFOLLOW` and bounded size checks. Compiler children inherit the caller-owned process
group and do not create a nested group. They receive EOF on stdin and an environment with
`TMPDIR` set to the stage and `LC_ALL=C`/`LANG=C`; the parent environment is unchanged.
Combined child output is limited to 64 KiB.

Each compile or link invocation kills and reaps its leader on a fault. The outer capture
owns descendant cleanup, the ten-minute overall build deadline and stage cleanup.

Publication opens the rooted parent with `O_NOFOLLOW`, rejects `..` and symlink segments,
creates a cryptographically nonce-named mode-0700 temporary file, writes, fsyncs, closes,
and renames into place. Failure preserves the previous image. The contract does not fsync
the parent directory and does not claim crash durability beyond that operation order.

### Entry and explicit CLI selection

The private entry matches the existing companion protocol:
`--packet-input <stage/packet.bin>`, `--stage-root <stage>`,
`--output-root <root>` and `--output <relative>`, with optional `--release`,
`--run-trap-control` and `--cc gcc|clang`. Trap mode requires the stage as output root and
`program` as the output.
The runtime pack is read fresh from the fixed sibling
`fairyc-linux-runtime-assets.pack` through the Linux rooted reader, avoiding Windows
`FileKind` selection.

`tools/prepare-candidate-linux-cli.py --companion` writes an isolated `main.fy` that selects
`linux_gnu_companion`, without applying the unrelated frontend/cache platform rewrites.
Installation of the matching companion and runtime pack beside a
Linux CLI is an explicit opt-in operation. Nothing automatically replaces the Go backend.

From the repository root on a supported Linux host:

```sh
go build -o .cache/fairyc-bootstrap ./cmd/fairyc
python3 tools/prepare-candidate-linux-cli.py --output .cache/linux-gnu-companion --companion
(cd .cache/linux-gnu-companion && ../fairyc-bootstrap experimental build-workspace \
  --target x86_64-linux-gnu --cc gcc --profile low \
  --output build/fairyc-candidate-linux-build)
python3 tools/package-candidate-linux-runtime.py \
  --output .cache/linux-gnu-companion/build/fairyc-linux-runtime-assets.pack
```

Place both generated files beside the prepared Linux CLI for an explicit opt-in install.

### Evidence and remaining gates

Real Fairy policy tests exercise compile, link and publication plus failure-stage reporting
under Windows MSVC low/optimized. The C bridge is exercised through MSVC POSIX substitutes
in both profiles; this is not Linux execution. The native Linux adapter tests are authored
and skipped on Windows.

`TestLinuxGNUCompanionOptions` passes under Windows MSVC low/optimized, checking private
build/run flags, explicit compiler choice, duplicate/unknown flags and trap-output bounds.
`TestLinuxGNUCompanionEntryEmission` emits the complete companion for both targets and
closes the Linux native support set without Windows foreign symbols. The original workspace
passes `check` and `check --locked` with 574 source files and 7,232 MIR functions. The
isolated companion workspace passes those commands with 574 files and 7,230 functions.

`TestLinuxGNUCompanionNative` is authored for GCC/Clang low/optimized. It builds the actual
Fairy entry and exercises runtime-pack loading, compile/link/publication, argument startup,
private run-image output, old-image preservation on compile/link failure and pack corruption.
The harness uses the production outer capture to own compiler descendants. It is skipped
on Windows along with `TestLinuxGNUAdapterNative`; the local WSL query fails with
`Wsl/CallMsi/Install/REGDB_E_CLASSNOTREG`. These native cases have not run here.

The complete focused `TestLinuxGNUCompanion` suite and `TestLinuxGNUAdapterBoundaries`
pass locally, with the native cases explicitly skipped. Runtime-pack pin verification,
Python syntax checks, `go vet -mod=readonly ./internal/candidatehost ./internal/cli` and
`git diff --check` also pass. Earlier Debian results below apply to the Go bootstrap path.
Native Linux GNU launch, fixed-point reproduction and distribution remain unverified.
`x86_64-linux-gnu` maturity is unchanged. Item 2 work introduces no new owner approval gate;
this implementation and documentation imply no push, CI dispatch, release or deployment.

## Historical Go bootstrap backend (retained default)

The following Go-only claims document the original bootstrap path. They remain accurate for
that path and its source owners; they do not describe the Fairy-authored companion above.

The initial Linux backend is `tools/candidate/fairyc-candidate-linux-build`, a Go
bootstrap companion. It contains no Fairy frontend. It consumes a fresh bounded
`FYCPARTS1` packet, requires the `x86_64-linux-gnu` target marker, selects the closed
Linux runtime, compiles separate C objects, and publishes the final executable.
It does not invoke MSVC, another Fairy compiler, cached packets or source discovery.

`internal/candidatebuild/linux.go` owns packet/runtime admission and publication;
`internal/cdriver/linux_project.go` owns native GCC/Clang compilation. Existing Linux
toolchain admission applies: native Linux amd64/glibc, GCC >=13 or Clang >=18.
Windows runtime symbols and unknown SDK/vector/host symbols fail selection.

When the closed header selects `fy_std_process_arg_count_v0` or
`fy_std_process_arg_v0`, the native backend adds a separate Linux startup unit.
It suppresses the packet's generic entry with `FY_CANDIDATE_NO_MAIN`, initializes
`fy_std_process_set_args_v0(argc, argv)`, then calls `fy_candidate_entry`.
The admitted packet bytes and target-neutral entry remain unchanged. This applies
to normal builds, private run images and native test images; empty arguments and
arguments containing spaces retain their original byte boundaries.

The caller supplies a new private staging directory containing only `packet.bin`.
Sources use exclusive creation; objects are bounded and inspected as ELF64 x86-64
relocatables. A response file lists only generated ASCII object basenames. The linked
ELF must be executable/PIE with a nonzero entry and executable load segment. Publication
uses existing rooted atomic replacement and executable mode. Failure before publication
preserves the prior image. C input/runtime/part/count/image bounds remain explicit.

Every compiler invocation and link has a one-minute deadline from the companion;
output is bounded to 64 KiB. The companion's caller owns the outer capture/process
group, overall deadline and staging cleanup. Compiler children inherit that group;
they must not create independent groups. GNU temporary files use the owned stage.
This is a bootstrap backend, not a claim that the Linux native driver is Fairy-owned.

Runtime selection, host/deadline admission and invalid ELF rejection are checked on
the Windows development host. Linux-native packet compilation/publication and
compile/link failure preservation tests are authored and skipped on Windows.

## CLI connection

The Go bootstrap's standalone Linux execution driver starts each target in its
own process group. Its timeout and parent trap-channel failure cleanup terminate
that group, including descendants that inherit it. Timeout errors retain
`context.DeadlineExceeded`; binary stdin and the private fd 3 trap receipt keep
their existing protocol. This is process cleanup, not isolation of programs that
deliberately escape the owned group. Native regression execution for this change
is deferred to the batched Linux queue.

`linux_file_cli` connects `build [--release] [--locked]` and build help to
`linux_build_cli`. Duplicate/unknown flags fail before compilation. Optional
`--cache-root <existing directory>` and `--cache-status` select the shared
[verified IR cache](candidate-linux-cache-v1.md).
Shared `project_emit` owns loading, locks, SDK authentication, entry selection
and C emission. Output is `build/<package-name>` with no `.exe`; the success report
names `x86_64-linux-gnu`, the selected profile and the exact load's lock state.

Prepared sources select `linux_c_file` for real Linux SDK wire tags and the signed
`error_number` in `fy_std_file_result_v0`, and add the Linux packet marker. Field-copy
adapters retain authenticated SDK source records. No emitted-C rewrite, SDK pin update
or Windows runtime pack is involved. The dispatcher invokes only the sibling
`fairyc-candidate-linux-build`, without PATH search, helper renewal or frontend fallback.

`linux_build_scratch` creates an exclusive mode-0700 directory under the selected temp
root and retains its descriptor, identity and absolute path. Trailing root slashes are
normalized. All packet/C/object/response/image files and GNU temporaries stay inside.
Capture owns the companion's process group with a ten-minute limit and 64 KiB per
stream. Descriptor-relative cleanup follows no symlinks, limits work to 16,384 entries
and depth 16, and checks root identity before removal. Consuming close clears its opaque
handle first. Cleanup may fail or leave a partial tree under filesystem failures;
noncooperating same-user writers and mount changes are outside this private protocol.
If publication succeeded before final cleanup failed, status 6 does not roll it back.

## Historical Go bootstrap pair

On the supported Linux host, from the repository root:

```sh
go build -o .cache/fairyc-bootstrap ./cmd/fairyc
go build -o .cache/fairyc-candidate-linux-build ./tools/candidate/fairyc-candidate-linux-build
python3 tools/prepare-candidate-linux-cli.py --output .cache/linux-build-cli
cd .cache/linux-build-cli
../fairyc-bootstrap experimental build-workspace --target x86_64-linux-gnu --cc gcc --profile low --output build/lexer-candidate-linux
cp ../fairyc-candidate-linux-build build/fairyc-candidate-linux-build
```

Run `lexer-candidate-linux build [--release] [--locked]` from a client workspace.
Keep the matching companion beside the CLI. Runtime C is embedded in the companion;
invocation needs neither an external asset pack nor an installed Go toolchain. This is
an explicit bootstrap pair, not clean-machine distribution or self-host proof.

## Historical Go bootstrap compilation

Linux native `build`, `run`, `test`, and bootstrap use `cbackend.EmitBuild`.
When function bodies total more than 2 MiB, it emits roughly 2 MiB function-boundary
groups plus a private shared header; full-entry header measured 1,869,589 bytes.
Literal storage is defined once; callback adapters live in the entry source. Small
programs, Windows, and experimental `emit-workspace-c` keep single-output behavior.
All C files use the same strict flags:
`-std=c11 -pedantic-errors -Wall -Wextra -Werror -fno-common`. Object linking
omits C-only flags, avoiding Clang's unused-argument diagnostic.

At most two workers compile, bounded by `GOMAXPROCS` and available memory:
256 MiB reserved, 512 MiB per worker. Runtime C, all parts, and final link share one
60-second context. Failure cancels sibling process groups and joins workers before
temp cleanup. Large splits use fresh objects/runtime C with no runtime-object cache;
small builds retain it. The companion's sequential `FYCPARTS1` route keeps its
per-invocation 60-second limits.

The workspace frontend returns a C-only build plan before invoking the native
driver, keeping MIR out of the frame that needs lock/trap metadata after building.
For partition inputs larger than 2 MiB, the driver collects unreachable Go objects
and returns idle heap pages to the OS before creating the shared C deadline.
This preparation time is separate from the unchanged 60-second C context and is
included in whole-bootstrap elapsed time. Available memory is sampled after C
files are written, immediately before worker selection; budgets remain unchanged.
Successful artifacts carry `PartitionBuildStats`, also logged by the native CLI
test. Post-admission failures include those facts through `PartitionBuildError`
and its error text, retaining the underlying cause. Go heap counters are not RSS.
No new CLI flags, environment switches, or success output are introduced.

External Debian 13 validation at `62f2a006` passed unchanged manual bootstrap with
GCC/Clang low/optimized under the shared 60-second C deadline. GCC optimized also
passed a second fresh-object build with identical ELF bytes. Whole-bootstrap
times were 70.88/68.64 seconds for GCC optimized, 41.81 for GCC low, 29.95 for Clang
low, and 54.79 for Clang optimized; these include frontend and memory preparation
and are not C-only times. No 600-second diagnostic was needed.

A separate instrumented GCC optimized run observed Go RSS falling from 1,081.71
to 99.88 MiB during 0.096795 seconds of memory preparation. Available memory at
admission was 1,668.23 MiB, selecting two workers with observed overlapping
compiler calls. Its C interval was 56.176 seconds, leaving about 3.8 seconds to
the deadline; it is not a measurement of either unchanged acceptance run.
The four default manual images passed 576 CLI calls, 60 Go reference comparisons
and 12 generated-C executions. Full CLI test-input builds also passed all four
combinations. See the [external validation record](../ops/linux-bootstrap-revalidation-62f2a006.md)
for provenance, measurements, commands and remaining limits. Other host/load
conditions, Ubuntu, clean-machine distribution and self-host acceptance remain
outside this evidence.

## Historical Go bootstrap local evidence

Full-entry Linux GNU/Windows MSVC C emission has no Windows foreign dependencies.
Windows low/optimized substitutes check dispatch, unsuffixed output, companion paths,
wire names/signedness and scratch lifetime/failure cleanup. Original and prepared source
workspaces pass `check`/`check --locked`; runtime admission, ELF rejection and vet pass.
Linux-native tests cover private nested-tree cleanup and outside-link preservation,
GNU object/ELF publication, low/release SDK File-reader builds and execution, locked
build, invalid flags, old-image preservation on compiler/link/frontend/companion failure,
and empty private TMPDIR after CLI calls. Native Linux cases are skipped on Windows.
Cold/warm cache integration cases are authored, including development-to-release
IR reuse and companion failure after a hit. The external Debian execution record
above complements these Windows development checks. The current Fairy-authored GNU
companion is documented at the top of this page; this section remains evidence for the
retained Go bootstrap path.

## Private run-image backend

`--run-trap-control` selects the existing Linux runtime's bounded trap receipt
writer on fd 3. This mode requires output root equal to the caller-owned stage
and output leaf exactly `program`; it cannot publish into the client workspace.
The [Fairy run caller](candidate-linux-run-v1.md) owns program launch and stage lifetime.
Ordinary builds retain their runtime and publication behavior.

The private capture adapter has a separate run opener which redirects a third
pipe to child fd 3 and closes inherited descriptors from fd 4 upward. All pipe
sources are above fd 3, including when standard descriptors were closed. The
existing process-group/deadline cleanup and fair stream polling also drain this
channel. Receipts admit 276 bytes; overflow fails capture. Only fully drained
collection exposes receipt bytes. Ordinary capture continues closing from fd 3.
Windows low/optimized POSIX substitutes cover receipt bounds and descriptor
cleanup; Linux-native probes are authored and skipped on Windows.

The [Linux test caller](candidate-linux-test-v1.md) also uses the private run-image
backend for one separately harnessed test per stage.
