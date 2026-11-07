# Fairy grouped static admission, 2026-11-07

Scope: [ADR 0549](../adr/0549-grouped-static-operation-interfaces.md).
The coordinator owns grouped admission, declaration-specific catalog errors,
the independent admission regression and the current source lock.

Groups retain at most sixteen member rows and count a seventeenth only to
reject it before masks/providers. Scalar masks cover the existing thirteen
families. Group size is checked before completeness; incomplete tuples select
the earliest written provider. Existing ungrouped validation order and work
accounting remain intact. Grouped validation checks all operations before
providers/duplicates, matching the Go phase order.

## Executed Windows checks

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateGroupedStaticAdmission$'
go vet -mod=readonly ./internal/genericsource ./internal/compiler ./internal/workspacecompiler ./internal/cli
python -B .cache/grouped-static-source-check-20261107.py all
```

The final admission test passes in **14.365s**. Twenty-four original UTF-8/CRLF
source controls are checked by Go and replayed in each of two actual Windows
low/optimized native probe processes. The probes compare acceptance and exact
E4501 original byte spans. They cover complete/empty/multiple groups, all
thirteen scalars, 16/17 members, 64/65 label bytes, malformed labels and private
members, unused partial tuples, duplicates, signature errors, group labels used
as provider targets, error precedence and the original ungrouped family.

Initial attempts corrected two fixture assumptions: a keyword is not an
attribute path in the existing grammar, and the legacy function/parameter name
collision is already rejected. These did not require product changes. The
22-case native run passed in 13.002s. Two added mixed-error cases then reproduced
the real grouped phase mismatch (6.153s failed run); the grouped-only fix above
passes all twenty-four controls. Failed logs remain separately under `.cache`.

Final current-source `check` / `check --locked` pass in **15.166s / 15.702s**:
758 source files, 9,593 MIR functions. All **1,229** authored Fairy files match
the isolated checked copy, and its lock was copied only after both commands
succeeded. Bootstrap SHA-256:
`00be89383b0049a349b0bf8559ea2c81c86e8af85feddf349069001ce53982fa`.
The final source record is `.cache/grouped-static-source-20261107-all-check.json`;
log `.cache/grouped-static-source-final2.log`. Scoped vet, gofmt and whitespace
checks pass. Source/work/stack/native compilation bounds remain unchanged.

After integration, main's existing test/sample files had thirty newline-only
differences from the detached checkout. Those files were preserved. A fresh
main-source snapshot therefore rechecked the actual bytes: `check` / locked
check pass in **15.445s / 13.345s**, with the same 758 / 9,593 counts and all
1,229 files plus lock matching. Log:
`.cache/grouped-static-main-integration-check.log`. The isolated evidence above
is not substituted for this byte-exact main integration check.

The separate [metadata/artifact receipt](candidate-grouped-static-metadata-20261107.md)
owns planner, digest, restored-artifact and four generated-C product results.
Full CLI/cache is a separate sidecar gate. Self-build/fixed point, distribution,
hosted CI and native Linux are not established by these probes. Linux remains
batched; no push, release, API/ABI, DB, dependency, hygiene or runner change occurs.
