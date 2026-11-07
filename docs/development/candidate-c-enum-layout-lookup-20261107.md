# Shared C enum layout lookup, 2026-11-07

This bounded follow-up removes repeated layout scans in C enum construction
and enum switch branches. It is independent of the checked-CFG drop projection
and does not itself establish a complete source-bound emission or self-build.

## Implementation

`module_unit_c_sdk::build` already builds and charges a
`module_unit_c_layout_index::Index` before SDK admission. Its immutable `Plan`
now retains that index. `module_unit_c_function::render` passes the same index
to enum expression and switch branch emission, across every rendered body and
partition retry. No additional index construction or cap increase is added.

The original standalone scanner entry points remain available. Both routes
share their tag/payload rendering and member checks. Original layout row IDs,
field IDs, C bytes and statement order are retained. Index construction still
rejects duplicate, zero and out-of-range layout types with 1001; a missing
layout is rejected by the expression/branch boundary with 1001. Each indexed
lookup retains its work charge, sticky errors and 1002 budget exhaustion.

Only the SDK emission plan, body caller, enum emitter and switch emitter
change. The project emitter, CFG model, reachability, frame index, public SDK
ABI, artifact formats, DB, dependencies and CI/runners are unchanged.

## Motivation and limits

The authenticated diagnostic 646-source/157-IR archive contains 244 enum
construction nodes across its 10,292 linked frames and 1,116 layouts. The old
scanner visits all 1,116 rows for each query, including a duplicate search:
272,304 visits for those nodes, before enum switch queries. These are linked
IR counts, not a count of reachable emitted expressions or a measured runtime
speedup. The parent subsequently reported successful saved-IR linking and
emission after its independent reachability fix; this patch does not claim
that the layout scan caused that earlier failure.

## Focused regression and validation status

`TestCandidateCEnumLayoutLookup` prepares a disposable fixture using test-only
constructors appended inside copied owner modules. Production visibility and
proof gates remain unchanged. Raw malformed records are used only to exercise
the lookup/error boundary; they are not represented as verified programs.

The passing fixture checks:

- An independent scanner against all 2,048 reverse-ordered layout rows:
  4,194,304 scanner visits versus 2,048 indexed lookup charges.
- The index retained by the actual SDK plan, without a second build charge.
- Enum payload and switch output against both standalone scanner emission and
  independent literal C expectations; 2,050 versus 3 charged work per example.
- Duplicate/invalid layout rejection, absent and invalid type queries, member
  mismatch, sticky errors, and existing 1,048,576 work exhaustion.

Executed Windows commands in this isolated worktree, with process-local
`GOMAXPROCS=1`, Go package/test parallelism one and at most one CL process:

```powershell
$env:GOMAXPROCS = '1'
go test -p 1 -parallel 1 -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateCEnumLayoutLookup$/^low$'
go test -p 1 -parallel 1 -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateCEnumLayoutLookup$/^optimized$'
```

Both pass with empty native stderr and the exact expected output:

```text
layouts=2048 scan=4194304 index=2048 sdk=8198
enum=equal switch=equal work=2050->3 malformed=1001 budget=1002
```

| Profile | Go package result | Native build/execution subtest | Command wall time | Log under `.cache/` |
| --- | --- | --- | --- | --- |
| low | PASS, 18.070s | PASS, 4.05s | 48.534s | `c-layout-lookup-low-2.log` |
| optimized | PASS, 23.337s | PASS, 9.30s | 27.978s | `c-layout-lookup-optimized.log` |

The CL allowance is the original one minute in both successful runs. The
initial low attempt failed in Fairy source checking before native compilation:
two new switch wrappers and one raw fixture conditional needed explicit
`usize` types. Those fixes are included; the failed log remains at
`.cache/c-layout-lookup-low.log`. Go formatting and staged diff hygiene pass.

An optional existing-authored-test harness was attempted separately, but its
temporary workspace reported `missing-visible-module` before native compilation.
It is excluded from the patch; `.cache/c-layout-lookup-existing.log` and
`.cache/c-layout-lookup-existing-2.log` retain both failed attempts. No existing
enum/switch regression pass is claimed. The unchanged production inputs of the
passing focused runs are reused; no additional test run is needed for removing
that separate harness.

## Integrated Windows validation

The isolated combined tree adds the checked-CFG drop projection to this change.
Source `check` and `check --locked` pass with 755 files and 9541 MIR functions
in 14.270s / 13.959s. Its refreshed lock is copied into the integrated checkout
only after the prior source-bound build finishes. Production source paths and
bytes match that checked combined tree exactly.

`TestCandidateCEnumLayoutLookup` also passes both profiles on that combined
tree in 32.064s (low 4.69s, optimized 9.69s), with the same independent C bytes,
lookup work and malformed/budget checks. Evidence is
`.cache/parallel-c-layout-lookup-20261107/.cache/merged-source-check.log` and
`merged-layout-native-shared-cache.log`. The cold isolated Go-cache attempt was
cancelled before tests began; it is not counted as a pass. The completed run
reuses the repository's existing Go build cache.

The preceding normal self-build at `b4008e0e` passes both profiles, but predates
these four emitter changes. A source-bound build/fixed point of the combined
revision and full current-source parity remain pending. Linux native checks
stay batched; CI, push and publishing remain unexecuted.
