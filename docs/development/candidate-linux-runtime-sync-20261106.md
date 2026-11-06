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
MIR functions. Scoped vet, formatting and diff checks pass. The pin pack and
runtime C assembly still need the matching seven assets; this selector change
alone does not complete GNU companion synchronization or Linux self-build.

Inspection also confirms `linux_cli_data` already preserves the previous full
Linux help, with the fetch line added; no help regression is reproduced.

No public API, SDK ABI, DB, dependency, timeout, CI/runner or repository policy
changes. No push, hosted dispatch, release or deployment.
