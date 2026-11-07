# Vector materialization reserve, 2026-11-08

This bounded sidecar changes only the optional `std::vec` implementation and a
focused SDK Go test. The materialized AST cap remains **1,048,576**. No compiler
accounting, generic planning, contracts, public signatures, FFI signatures, source
limits, error codes, owner-borrow intrinsics or checks were changed.

## Implementation and preserved ordering

Five private, non-generic helpers share the existing initialization, commit,
release and non-null checks. Generic `Vec<T>` cleanup, `pop<T>` and `truncate<T>`
use the checked pop helper; the raw pop still happens before the null assertion,
which still happens before `take_slot<T>`. Cleanup and truncation retain their
explicit **`let T`** temporaries and the same iteration scopes, so concrete
resource types and reverse destruction order are retained.

`at<T>` and `at_mut<T>` directly feed their one-use checked address to the same
explicitly typed owner-borrow intrinsic. Bounds assertion, typed address
calculation, null assertion and owner borrow remain in that order. The scalar
`status` locals infer `u32` from the unchanged extern returns; `count` infers
`usize` from the unchanged length field. The push slot retains its explicit
pointer type because a null initializer alone does not establish that type.

Push still prepares allocation, checks the slot, writes the moved element,
checks the write result, commits, checks the commit result, then returns length.
Allocation failure still returns the original element. Empty construction still
initializes before returning, and cleanup still releases only after all elements
are destroyed. Helpers accept only raw pointers and scalar metadata; they add
no owning temporaries. Assertions retain the same assertion-failure trap kind.
Runtime equivalence is not claimed from these source/Flow/MIR checks alone.

## Measured snapshots

All compiler builds and candidate checks ran in
`.cache/materialization-reserve-20261108/`, with `GOMAXPROCS=1`. Production Go
compiler files were copied, not edited; their hashes were checked against the
live files after measurement. Instrumentation reuses the earlier sidecar's task
reporting while leaving the real budget enforcement and accounting unchanged.
The candidate source was refreshed independently for the later snapshot.

| Sample (KST, HEAD `b5af44e4`) | Original SDK | This change | Net reduction | Remaining reserve | Full check |
|---|---:|---:|---:|---:|---|
| Initial copy, 2026-11-08 03:47:33 | 1,047,697 | 1,038,116 | 9,581 | 10,460 | Both PASS; 773 files, 9,697 / 9,702 MIR functions |
| Refreshed concurrent worktree, 2026-11-08 03:52:52 | 1,051,669 | 1,042,088 | 9,581 | 6,488 | Original E4503; changed PASS, 775 files / 9,721 MIR functions |

The original SDK in both rows already contains `a52bf883`'s earlier 1,568-node
reduction. The new **9,581** saving is additional, not cumulative with that
previous result. The historical 1,047,666 figure described an older source
sample and is not the refreshed candidate baseline.

The later snapshot includes concurrent unit-body IR, cleanup lowering, C backend
and new closure modules/tests. `current-inventory.json` records its capture time,
HEAD, Git status and SHA-256 values. The table describes those frozen inputs,
not every subsequent edit made by other workers.

| Existing task | Nodes/copy before -> after | Copies | Nodes saved |
|---|---:|---:|---:|
| `Vec` cleanup/fields | 67 -> 53 | 337 | 4,718 |
| `at` | 41 -> 33 | 334 | 2,672 |
| `at_mut` | 41 -> 33 | 115 | 920 |
| `empty` | 44 -> 43 | 335 | 335 |
| `push` | 79 -> 77 | 335 | 670 |
| `pop` | 45 -> 36 | 19 | 171 |
| `truncate` | 54 -> 44 | 17 | 170 |
| `reserve` | 43 -> 42 | 15 | 15 |
| Five new helpers, counted once each | 21 + 22 + 15 + 13 + 19 | 1 | -90 |
| **Net** | | | **9,581** |

Every changed task belongs to `std::vec`; all other task work is identical.
Existing copy counts and concrete specialization requests are unchanged after
normalizing the SDK content-coordinate prefix, which naturally changes with SDK
content. Helpers are real non-generic source functions and remain fully counted.

The residual `std::vec` work is 90,087 nodes. Its largest tasks are `push` (25,795),
`Vec` (17,861), `empty` (14,405) and `at` (11,022). The refreshed candidate still
has only 6,488 nodes available for further ordinary source growth; no additional
unsafe or contract-widening reduction was forced.

## Validation and handoff

- `go test -mod=readonly ./internal/sdkstdlib -count=1` on the copied final SDK:
  PASS. The new focused test verifies scalar `u32`, nominal struct and owning
  resource elements through reserve/push/at/at_mut/pop/truncate/clear and final
  cleanup, using authenticated SDK source analysis, Flow and verified MIR.
- `go test -mod=readonly -overlay ../overlay-sdk.json ./internal/compiler -run
  '^TestPrivateVectorSDKCandidate(Encapsulation|BorrowInvalidation)$' -count=1`:
  PASS. Encapsulation and live-borrow clear/reserve/pop/move rejections retain
  their expected diagnostics, including E2101/E2105.
- Both snapshot pairs used separately copied candidate trees and locks.
  `fairyc-base.exe check` / `fairyc-reserve.exe check` results are in the table.
  Only the private copies' normal CLI locks were updated.
- Public and raw FFI signatures compared equal to the captured original source.
  Final source bytes equal the SDK payload used by the measured overlay.
- Scoped `git diff --check`: PASS.
- Native execution, C builds, self-hosting, Linux and remote CI were not run;
  Native validation remains reserved for the parent.

Parent integration subsequently passed the live SDK package tests (0.550s),
the full live candidate `check` (775 sources, 9,721 MIR functions), and the unit
body/CFG producer on Windows low and optimized profiles (11 original-source
controls per profile; 90.46s). Those native runs use this optional SDK, including
the resource-valued body tables and their vector operations/cleanup. They are
not whole-compiler self-hosting or a standalone native destructor-order oracle.
Logs: `.cache/closure-vector-reserve-sdk-20261108.log`,
`.cache/closure-unit-body-main-check-20261108.log`, and
`.cache/closure-unit-body-native-20261108.log`.

Evidence: `inventory.json`, `current-inventory.json`, `comparison.json`,
`overlay-base.json`, `overlay-reserve.json`, `overlay-sdk.json`,
`baseline.result.json`, `reserve.result.json`, `current-baseline.result.json`,
`current-reserve.result.json`, matching stats/stdout/stderr, `sdk-test.log`,
`sdk-embedded-test.log` and `borrow-tests.log` under the cache directory above.

No shared lock, other worker source, Git index or commit was changed by this
sidecar. No push, deployment, dependency, DB, runner or repository-hygiene change
was made. This development note is evidence, not a normative contract change.
