# Project cleanup drop capacity, 2026-11-07

This bounded change starts from `5ab962c2bea3b7af1308f25c1359a09a5352fea6`.
Only linked project cleanup owners admit up to **32,768 drop sites**. Ordinary
unit/raw cleanup owners retain **16,384**, including aggregate-enabled units.
`module_unit_cleanup_mir_model::drop_limit(project)` owns this distinction.

The operand count and per-frame local operand ID remain 16,384; the operand
uniqueness index remains 32,768 slots. Unit producer work remains 1,048,576,
project producer/link work remains 16,777,216, and the independent verifier and
artifact transport ledgers retain their existing bounds. No type, frame,
layout, field, callable, parameter, name or source-count policy changes here.
In particular, this worktree does not include the parallel project-layout
capacity change.

## Guard inventory

| Owner | Policy |
| --- | --- |
| Cleanup model | One `drop_limit(bool project)` helper: 16,384 / 32,768. |
| Cleanup verifier | Count guard uses the owner's project flag before traversal. All structural/authority checks and error order remain in place. |
| Project cleanup builder | The selected-drop append guard uses the project limit. Unselected operands map to zero and their drops remain excluded. |
| Cleanup artifact decoder | Drop-count preflight uses the already decoded project flag, before allocation. The remaining-byte check is unchanged. |
| Cleanup artifact encoder | Unchanged: independently verifies the owner before writing counts and rows. |
| Unit cleanup lowerer | Unchanged: final unit assembly enters the unit verifier; its operand guard stays 16,384. |
| Artifact cleanup rebase | Unchanged: copies the already checked rows and re-enters assembly/verification. |
| Unit drop-analysis edges / generic cleanup dependencies | Separate 16,384 bounds, unchanged. |

`FYCLEAN0` / `FYCLEAN1`, private compiler markers, schema revisions, source-bound
cache identity, provenance, `Restored`/`Complete` and publication gates are
unchanged. Increasing a detached table capacity grants no executable or
current-source authority.

## Focused Windows verification

Command, from the isolated worktree:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateProjectDropCap$' -count=1 -v -timeout 8m
```

`TestCandidateProjectDropCap` passed both Windows AMD64 MSVC profiles in
**19.15 seconds** (low 5.83 s, optimized 5.08 s; package 19.298 s).
The fixture uses fresh owned vectors. Temporary test-only constructors inside
copied modules create small input wrappers and plans for the project cleanup
builder; these wrappers carry no checked-body or Complete authority. Repeated
valid drop references test detached structural capacity, not execution of
duplicate cleanup.

Both profiles observed the same results:

| Selected project drops | Append result | Link work |
| ---: | --- | ---: |
| 16,384 | PASS | 16,393 |
| 23,100 | PASS | 23,116 |
| 32,767 | PASS | 32,783 |
| 32,768 | PASS | 32,784 |
| 32,769 | Refused, 1002 | 32,792 |
| 32,768 plus a discarded 16,384-drop unit | PASS, discarded drops excluded | 49,174 |

Every selected output drop is compared with its original row and independently
expected operand remapping. Borrowed input owners retain their local IDs and
pass verification after building. Unit 16,384 succeeds; unit 16,385 and project
32,769 refuse with 1002. A local ID of 16,384 succeeds, 16,385 refuses with
1001; 16,385 operands refuse with 1002. Existing unit/project producer-work
boundaries are checked at the exact limit and one over.

Ordinary project artifacts at all four admitted counts, an ordinary unit at
16,384, and an aggregate-enabled project at 32,768 roundtrip byte-for-byte.
An independent Go big-endian golden compares every encoded scalar, count and
row, rather than relying on encoder/decoder agreement alone. At 23,100 drops
the packet is 739,576 bytes and encode/decode work is 896,088 / 896,089. At
32,768 it is 1,048,952 bytes and work is 1,253,804 / 1,253,805, below the
unchanged 8 MiB payload and 16 MiB transport limits.

Refusal coverage includes truncated header/count/row/tail boundaries, trailing
bytes, a count of 32,769, a unit flag with 32,768 drops, an invalid project flag,
crossed ordinary/aggregate markers, and an invalid final drop retaining its
original 1001 span. Independent verifier and encode/decode ledgers succeed at
their exact work boundary and refuse one work unit short. Production and fixture
hashes are checked before/after each profile. `gofmt` and `git diff --check` pass.

## Parent diagnostic context and remaining acceptance

The parent's count-only experiment over 157 checked units from frozen 646
original sources measured **2,950 selected operands and 23,100 selected drops**.
Its local `current-project-cleanup-count-20261107/run.json` records 24.565 s,
stable inputs and an intentional stop before assembly (44802).
Thus 23,100 exceeds the old project limit by 6,716 and leaves 9,668 sites below
the new ceiling. This is a capacity correction, not a claimed work reduction.

The separate parent replay with both proposed drop and layout policies linked
10,292 frames in 25.848 s. Both diagnostic receipts explicitly say
`diagnosticOnly: true` and `acceptanceEvidence: false`; their shared input index
SHA-256 is `84bf616e6dbfce0432649281951818878dcb5669fe404622cf98ad82ea2a0d20`.
They do not establish normal source compilation or a full self-build pass.

Linux native execution is deferred to the batched lane. No broad suite,
full self-build, root source/lock/cache edits, push, CI or release was performed.
Parent integration, root source checks and fresh normal acceptance remain with
the parent. No public API, DB, SDK/ABI, runner or repository-hygiene configuration
changes are included.
