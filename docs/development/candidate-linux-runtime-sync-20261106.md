# GNU companion runtime synchronization, 2026-11-06

This continues the Linux fetch CLI work. Native Linux execution remains batched;
Windows runtime-source assembly is not evidence of Linux syscalls or TLS.

## Selector closure

The pre-change `TestLinuxGNUCompanionRuntime` fails in both Windows profiles:
the Go inventory includes package-directory/file, monotonic-clock and HTTPS
symbols that the Fairy selector rejects. The selector now retains the existing
eleven host bits and adds seven source bits for those adapters. Every HTTPS
entry selects the clock plus protocol/worker/reader/parent; package-directory
and package-file operations select their rooted-file prerequisite.

The oracle reads the live Go symbol switch and actual C files, including all
seven additions. It also checks overlapping HTTPS/clock and package/HTTPS
dependencies. The final command passes 118 plans in both Windows profiles
(4.991s), with both-target emission and empty PATH for generated executables:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxGNUCompanionRuntime$'
```

Original full-source `check` and `check --locked` pass 697 source files / 8,635
MIR functions. Scoped vet, formatting and diff checks pass. At the selector-only
commit the pin pack/assembly still needed the matching assets; the following
slice closes that implementation gap without claiming Linux self-build.

Inspection also confirms `linux_cli_data` already preserves the previous full
Linux help, with the fetch line added; no help regression is reproduced.

## Pinned assets and complete source assembly

The matching runtime-pack change adds seven raw source files, preserving previous
asset IDs. The new pack has 32 sources, 172,006 bytes and SHA-256
`ca0edf095672a72be24fff110ce61bb8476236de4bbf9ddca343a75dcf58f724`.
The regenerated source pin rejects old, truncated, extended or altered packs.
Assembly retains the Go order: feature macro, rooted/process/capture/input/paths,
directory, package-directory/file, retention/workspace/scratch/starter/GNU, clock,
HTTPS protocol/worker/reader/parent, then standard support and scalar runtime.

The expanded oracle reproduces failure with the old pack/assembly in both Windows
profiles. The final asset test passes in 9.923s: 76 full C outputs per profile,
covering every new entry, overlapping dependencies, all hosts, and trap receipts
both enabled and disabled, plus the three pin-tampering controls. Generated
executables run with empty PATH. An initial helper returned an unwrapped usize;
the compiler rejected it before execution and it was corrected to `Result::ok`.

```powershell
python -B tools/package-candidate-linux-runtime.py --update-pin
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxGNUCompanionRuntimeAssets$'
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^(TestLinuxFileCLI|TestLinuxGNUCompanionEntryEmission)$'
```

Full CLI/companion both-target emission passes (47.486s combined). The CLI has
23 partitions and a 1,817,749-byte header; portable Windows dispatch passes both
profiles in 7.22s. The outer CLI test explicitly skips native Linux execution.
This is not a native CLI pass or a Linux self-build. Original `check`/`check --locked`
pass 697 files / 8,636 MIR functions. Fresh isolated Linux CLI and companion
preparations each pass check/locked with 697 / 8,637 and 697 / 8,634 respectively.
Pin regeneration/verification, scoped vet, Go formatting, UTF-8/LF/final-newline
and diff checks pass. No new source lock changes beyond the example graph.

Public CLI/SDK/ABI policies remain unchanged; the implementation architecture
`candidate-linux-build-v1.md` is refreshed with the current pack/evidence. Native
Linux compilation, optional libcurl headers/runtime/CA provisioning, trusted TLS
and consecutive self-build stay open. Reuse the authored native companion and
full CLI tests in the batched Linux handoff; no Windows substitute establishes
those gates.

No public API, SDK ABI, DB, dependency, timeout, CI/runner or repository policy
changes. No push, hosted dispatch, release or deployment.
