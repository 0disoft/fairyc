# Private Linux candidate workspace build v1

The initial Linux backend is `tools/candidate/fairyc-candidate-linux-build`, a Go
bootstrap companion. It contains no Fairy frontend. It consumes a fresh bounded
`FYCPARTS1` packet, requires the `x86_64-linux-gnu` target marker, selects the closed
Linux runtime, compiles separate C objects, and publishes the final executable.
It does not invoke MSVC, another Fairy compiler, cached packets or source discovery.

`internal/candidatebuild/linux.go` owns packet/runtime admission and publication;
`internal/cdriver/linux_project.go` owns native GCC/Clang compilation. Existing Linux
toolchain admission applies: native Linux amd64/glibc, GCC >=13 or Clang >=18.
Windows runtime symbols and unknown SDK/vector/host symbols fail selection.

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

`linux_file_cli` connects `build [--release] [--locked]` and build help to
`linux_build_cli`. Duplicate/unknown flags and cache options fail before compilation.
Shared `project_emit.evaluate` owns loading, locks, SDK authentication, entry selection
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

## Matching bootstrap pair

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

## Local evidence

Full-entry Linux GNU/Windows MSVC C emission has no Windows foreign dependencies.
Windows low/optimized substitutes check dispatch, unsuffixed output, companion paths,
wire names/signedness and scratch lifetime/failure cleanup. Original and prepared source
workspaces pass `check`/`check --locked`; runtime admission, ELF rejection and vet pass.
Linux-native tests cover private nested-tree cleanup and outside-link preservation,
GNU object/ELF publication, low/release SDK File-reader builds and execution, locked
build, invalid flags, old-image preservation on compiler/link/frontend/companion failure,
and empty private TMPDIR after CLI calls. Native Linux cases are skipped on Windows.
Workspace `new`, caches and a Fairy-authored GNU backend remain separate work.

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
