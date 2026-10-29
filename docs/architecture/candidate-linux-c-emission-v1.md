# Private Linux candidate C emission v1

`linux_file_cli` connects two existing emitters without invoking a native compiler
or the Go bootstrap companion:

- `--c-file <path>` loads one source through `linux_cli_file`, then uses
  `file_frontend_cli` mode 8. This emits the existing scalar/nominal C11 unit,
  including its entry wrapper. It retains the shared 64-bit Windows/Linux C
  scalar assertions; it is not the workspace SDK emitter.
- `--c-workspace <root> [--locked]` calls `project_emit_cli::arguments`.
  `project_emit` owns manifest/dependency loading, authenticated SDK input,
  source checking, lock policy, canonical root entry selection and partitioning.
  Only an admitted complete bundle is written through `module_project_cli::packet`.

These are private compiler-debug transports. They do not expose the reference
compiler's `experimental emit-workspace-c` interface or restore stable `emit`.
Module-list C commands, cache flags and target overrides are not accepted here.
The original Windows dispatcher and no-argument binary stdin protocol are unchanged.

## Target and output

The existing preparation tool selects `linux_c_file` for SDK File wire types and
adds `/* fairyc.candidate-target/x86_64-linux-gnu/v1 */` to the project header.
The workspace packet uses the same `FYCPARTS1` framing consumed by the
[Linux bootstrap backend](candidate-linux-build-v1.md). Header, literals and C
units are length-delimited; stdout contains no success summary. This packet is
not byte-identical to the Go reference's single C unit.

Single-file input retains the 65,536-byte limit, UTF-8/nofollow path admission,
regular-file identity observations and close checks of the Linux file loader.
Workspace input uses the existing rooted workspace reader selection. An unlocked
successful load may update `fairy.lock`; `--locked` requires its current input
and leaves it unchanged. No build image or retained native stage is created.

Invalid arity/options return 2. Source/semantic/workspace errors retain their
shared command diagnostics and statuses. Failed admission emits no C. Output
I/O can interrupt an otherwise admitted unit or packet; consumers must check
exit status and framing. Existing shared packet write failure returns 2.

## Validation boundary

`TestLinuxFileCLI` emits the complete Linux entry for GNU and MSVC, selects closed
Linux support and checks portable dispatch in low/optimized profiles.
`TestLinuxCEmissionWindows` executes the shared C paths with Windows filesystem
I/O and Linux emitter selection. It checks packet framing/target, deterministic
locked output, lock preservation, invalid arguments, single-file output, Linux
SDK File wire selection and semantic rejection without partial stdout. This
compiler-host probe reserves 64 MiB of Windows stack for SDK projection; the
default MSVC 1 MiB stack overflowed. Product stack defaults are unchanged.
The complete workspace emitter runs in low profile: MSVC `/O2` exceeded the
one-minute compiler budget. `TestLinuxCFileWindows` separately executes file
emission, compiles/runs the emitted program and rejects invalid source in both
low and optimized profiles. Full optimized workspace execution remains unverified.

Linux-only full-dispatch cases cover missing/oversize/symlink source inputs.
`checkLinuxCEmission` also authors GNU compilation/execution of the partitioned
packet, single-file unit and Go reference unit, plus exact comparison of a
semantic error. These native Linux cases are skipped on Windows; host substitute
execution cannot establish Linux GNU runtime behavior or self-host readiness.
