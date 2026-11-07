# Source-owned unit closure names, 2026-11-08

ADR 0551 unit frontend step. `module_unit_name_walk` previously had no case for
prepared construction kind 36. It now resolves the real private lifted helper
and ordered hidden capture snapshots, independently of an ordinary provider
with the same spelling.

The helper must match current signature/module/declaration/source-span facts.
Captures retain the existing hidden-entry lookup, written order, bounded count
and original operand spans. Bad helper/capture tokens, public or wrong-owner
targets, reordered/repeated captures and unsupported source owners remain
refused. This is name resolution, not full unit value or executable admission.

The Windows coordinator ran:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureUnitNames(Syntax|Oracle|Check|Native)$' -count=1 -v -timeout=240s
```

PASS: package 40.729s, native acceptance 35.01s. The actual parser/preparation
and ordinary name owner cover zero through four captures, snapshot order,
local/provider shadowing and helper/capture/source-owner refusals in both
`run` and `run --release --locked`. Independent original-source Go controls
classify six accepted and eight refused sources. The focused source check
reports 177 files and 2909 MIR functions. The temporary native coordination
gate is removed; only the Windows AMD64 platform guard remains.

The complete compiler source also passes `check` and `check --locked` after
the parallel changes: 771 files and 9684 MIR functions. The shared lock update
records these integrated source bytes.

Unit closure calls/value checking, later body/cleanup IR and C consumption,
project restoration and persistent-cache acceptance remain unfinished. Linux
stays deferred for the requested batch. No normative source-of-truth document,
public SDK ABI, database, dependency, runner or remote state changed.
