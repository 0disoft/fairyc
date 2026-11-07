# Safe-function source dependencies, 2026-11-08

ADR 0551 project frontend step. The source planner visited callback type AST
kind 15 but rejected safe-function kind 17 at `generic_source_dependency_walk`.
This preceded unit checking, including for signature-only sources.

The dependency owner now visits kind 17's result and linked kind-16 parameters,
with backward/bounded structural links. It continues using original written
source when closure preparation owns a private helper. Traversal grants no
executable signature or generic transport authority: those source owners keep
their independent refusals. Existing project/unit executable formats stay closed.

Windows coordinator acceptance passed in the combined source/artifact run:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidate(ClosureValueSourcePolicy(Source|Syntax|Check|Native)|ClosureSourceDependencies(Oracle|Check|Native)|NativeMIRArtifact)$' -count=1 -v -timeout=6m
```

Package 153.822s; dependency native acceptance 27.64s. Four original sources
reach the source request owner: signature-only closure, prepared factory,
direct carrier struct and scalar control. Four source-signature refusals, two
forbidden generated-name child refusals, ten malformed owned-link cases and
the exact/one-over work boundary pass in both `run` and `run --release --locked`.
The independent Go controls classify four accepted and seven refused original
sources, plus three malformed signatures; the Go oracle preserves AST bytes.
The focused producer source check reports 172 files and 2807 MIR functions.
The native test retains only its Windows AMD64 platform guard.

This removes one source-planning barrier. Unit calls/value checking and the
later IR, C and restored-project consumers remain independent work. Linux is
deferred for the requested batch. No normative source-of-truth document,
public SDK ABI, database, dependency, runner or remote state changed.
