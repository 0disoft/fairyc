# Cleanup recipe lookup index

Code commit `cd72aaef` integrates worker commit `df15a0a2` on root main.
This implementation/evidence note changes no source-of-truth contract; the
existing cleanup MIR contract and source admission remain authoritative.

## Failure and focused diagnosis

The normal 646-source authored self-build at `121035cc` exits 2 after 427.558s:

```text
internal/sdkstdlib/sources/optional/std/vec.fy:5:5:
project phase 2 stage 6 code 1002 (bytes 104..122)
```

No C packet is produced (zero stdout bytes; empty-output SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).
The original invocation and source snapshot remain in
`.cache/current-selfbuild-recipe-fields-20261107/`.

Selected-unit sweeps retain the complete 646-source input closure and metadata.
Units 0..470 pass independently; unit 471, `project_compiler_core`, fails in
owned-body IR field-name work (diagnostic code 14002, provider module 615,
source span 104..122). Evidence is
`.cache/current-owned-ir-focus-20261107/sweep-early.json` and
`sweep-late.json`, with their adjacent stdout files. These are unit diagnostics,
not normal self-build acceptance.

The baseline/indexed engine-only comparison retains all 646 normal source
inputs from `121035cc`, selecting only unit 471. Both receipts authenticate
snapshot SHA-256
`84e6288c8a337158d13374e4d51358815094f96b295416ca21361628f725510b`
and report no source drift:

| Engine | Cleanup work | Cleanup frames | Owned-body IR |
| --- | ---: | ---: | --- |
| Baseline | 948429 | 197 | Fails, code 14002 |
| Indexed | 798289 | 197 | Passes, 197 frames |

The reduction is 150140 charged work (15.83% of baseline cleanup work), leaving
room for owned-body IR under the unchanged work cap. Receipts are
`work-471-baseline.json` and `work-471-indexed.json` in the same directory.
Their 6.023s / 7.150s elapsed times are concurrent diagnostic observations,
not throughput or compiler speedup claims. The compared engine changes while
the authored input snapshot stays fixed; this is not a current normal build.

A subsequent diagnostic refreshes authored source 301 to the integrated lowerer,
retaining the other 645 source bytes/order and the existing generated entry and
version module. Snapshot `5549b6c774ac5234875b68e38ae364dc1f644ead43a1597e8d31db99c21e7cb1`
passes source 301 with 174 owned frames and source 471 with 197 frames (cleanup
work 798302). Both `current-work-301.json` and `current-work-471.json` retain
zero source drift. This source-update diagnostic is not a normal build receipt.

## Implementation and bounds

A lazy, lowering-local lookup stores the last matching shape and the unique
cleanup frame for each canonical type ID. Last-shape selection preserves the
old cleanup scan semantics rather than the Interfaces first-shape index.
Duplicate cleanup frames set a sentinel: rejection still happens after that
owner's ordered fields, with the existing source span. Missing-frame checks,
field owner/ordinal/type records and downstream verification remain in place.

Lookup work changes from O(R*S + N*F) to O(T + S + F + R + N), where T is the
type count, S shapes, F frames, R resource recipes and N nonempty recipes.
Index setup is charged and skipped when no resource recipe needs it.
The lookup has two usize values per row and T+1 rows, at most 4096 type IDs.
On amd64 its maximum logical row payload is 65552 bytes; this is not measured
allocator or process RAM. Work stays capped at 1048576 and unit recipe fields
at 2048; source AST and linked-project field limits remain separate.

## Recorded validation and remaining work

Windows low/optimized index regressions pass in 129.03s, covering synthetic
lookup tables and real source-generated units, last-shape selection, unique
frames, error ordering and unchanged inputs. Field-boundary replays pass in
131.79s: 1023/1024/1025/2047/2048 accepted, 2049 rejected at stage 3 code 10.
These worker regressions are separate from the parent unit-471 comparison.
The tested lowerer SHA-256 is
`5150b5cc49d41d8ffef210fe653b7a45832910edd973fd950a2439e8427d0865`.

Worker evidence is under
`.cache/parallel-cleanup-index-20261107/.cache/cleanup-recipe-index/`:
`windows-native.log`, `windows-fields.log` and `result.json`.
The existing Windows MSVC profiles use `/Od` or `/O2` with `/W4 /WX`;
no compiler flag relaxation is introduced. The worker records passing
`go vet -mod=readonly ./internal/cli`, `git diff --check` and
`gofmt -l internal/cli/candidate_cleanup_recipe_index_test.go`.
Its result.json predates the parent's unit-471 comparison; that pending item
is resolved only by the separate parent diagnostic receipts above.

At integrated `cd72aaef`, the parent reports unlocked/locked source checks
passing: 755 files, 9527 MIR functions, 18.419s / 19.598s, then fairy.lock refresh.
A fresh optimized build-only Go seed and help smoke pass in 140.587s (205 C
parts). The normal authored retry follows. Full normal self-build, consecutive fixed points
and latest-source full parity remain pending. Native Linux is unexecuted and
batched separately. No API, DB, SDK ABI, CI, runner or target-policy change is
introduced. No push, release or broader queue completion is claimed.
