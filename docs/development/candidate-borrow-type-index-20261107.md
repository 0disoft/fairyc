# Candidate borrow type index: Windows regression evidence, 2026-11-07

This records a trial in `.cache/parallel-borrow-index-20261107`, based on
`7407fed11d1b4480b84f5016fa96d4645d68954c`. The parent owns the three production
Fairy files and lock. This sidecar added only the Go test, its Fairy fixture,
and this document. No commit, main-worktree edit, or Linux execution occurred.
This is implementation/test evidence, not a new language or memory contract.

## Measured motivation and focused counterexample

The completed original full diagnostic trace exited 2 after 389.337s: the last
ordinary body succeeded with 73,822 work, but Values could not add it to
1,044,990 accumulated work (attempted 1,118,812; cap 1,048,576). Its place-borrow
calls cost 58,162 work (78.79%). Across the failing Values build, place borrow
cost 726,613 (67.65% of body work); builtin canonicalization cost another
252,369 (23.50%). These are work-accounting amounts, not CPU-time measurements.
Existing canonical hits in that trace did not establish frozen-prefix membership.

The parent's focused receipts were read directly, without rerunning the harness:

| Runtime implementation | Exit | Selected native_cli value analysis | Seconds |
| --- | --- | --- | --- |
| Original main | 2 | code 10, expected 1048576, actual 1118812 | 2.567818 |
| Trial borrow index | 0 | work 606196, contexts 216 | 2.337364 |

Both receipts retain all 646 source inputs and select module 391 only as the
analysis range. Both report unchanged inputs (`sourceChangedAfter: []`), the same
source snapshot SHA256 `95efe98ad91d455badcc7074fadee07a7e4a08f4bf3b5893d31124898708ed8c`,
and manifest SHA256 `9c4928f4c13298f480f706cfdc01facf67f6afa4eb1a33c7652d934681969586`.
Metadata/source inputs are retained; this is not a smaller source closure.
The original receipt is in the main worktree's
`.cache/current-native-value-focus-20261107/run.json`; the trial receipt is at
the same relative path in the trial worktree. Both explicitly set
`diagnosticOnly: true` and `acceptanceEvidence: false`.
Neither result is full authored self-build or fixed-point acceptance.

## Production behavior inspected

`module_unit_interfaces` builds an owned, zero-initialized `Vec<usize>` with
`2*(G+1)` slots for the validated immutable Graph prefix. Each reference row
is indexed by target ID and shared/mutable kind, retaining the first matching
canonical ID. The public `borrow_type` getter returns zero for target zero,
one-past-prefix, and absent reference rows.

`module_unit_value_place::borrow` retains its required signature and validation
order. The new `borrow_cached` shares that validation, checks a cached row against
the current ValueGraph record, and returns its original ID on a hit. It relies
on Values validating the exact imported prefix and subsequently appending types.
`module_unit_value_scalar` alone calls the new internal path. Other callers keep
the ordinary canonical path; permission and output semantics remain common.

Budget behavior is deliberately different on a cache hit: one lookup work unit
replaces the old `current_graph_count+1` precharge. A prefix lookup miss costs
one unit **plus** the ordinary precharge. A target outside the prefix skips that
lookup and pays the ordinary precharge. A miss does not mutate the prefix index.

## Regression fixture and observed results

Run from the trial worktree on Windows AMD64:

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateBorrowTypeIndex$' -count=1 -timeout 8m -v
```

`internal/cli/candidate_borrow_type_index_test.go` uses the existing
`candidateAtlasProbeSize` helper with the real authored import closure, MSVC
low/optimized profiles and a 16MiB stack reserve. It does not build the whole
CLI. It hashes the three production inputs before and after each run and logs
the fixture and native PE hashes. The generated probe executables are temporary.

The fixture at `internal/cli/testdata/borrow_type_index/main.fy` verifies:

- Real scanner/parser, signature, binding, and Interfaces builders; no forged
  Interfaces object or access to its private index.
- All target IDs from zero through one-past-prefix, both reference kinds, with
  an independent ordered six-field linear canonical oracle: 310 queries total.
  First/last shape targets and separate shared/mutable IDs are exercised.
- Constant one-unit fast hits for a 5-type prefix, an unrelated-type-expanded
  146-type prefix, and a ValueGraph extended to exactly 4096 records.
- Initial work 1048575 succeeds and reaches 1048576; initial work 1048576 fails
  code 10 without changing work. The ordinary borrow path still fails at
  1048575 because it requires its full graph precharge.
- Unaddressable mutable borrow returns 1008 before shared-borrow error 2 and
  before budget failure, even at exhausted budget; immutable/resource-field
  mutable borrows also return 1008. Shared borrowing of the resource-field
  place remains allowed. Existing `borrow` agrees on the unaddressable cases.
- An uncached prefix scalar uses canonical fallback, including its extra lookup
  charge. A value-added Option target beyond the prefix uses ordinary fallback;
  repeated canonicalization retains the original ID.
- The public canonical API fills distinct scalar-array records to 4096, existing
  cached IDs still succeed, and both a new 4097th record and a missing mutable
  reference are rejected. This independently tests the Graph cap, rather than
  substituting the source module's 128-declaration limit.

Both profiles printed:

```text
small_types=5 large_types=146 small_setup=142 large_setup=2821 queries=310 fast=1 graph_cap=4096
ok
```

Final test PASS: 36.40s; low subtest 6.07s, optimized subtest 25.32s. No skip.
Log: `.cache/borrow-type-index-sidecar-20261107/native-verified.log` in the trial.
Earlier preparation logs contain fixture-only failures: an initial 80-struct
single-module fixture exceeded the independent 128-declaration limit, then an
incorrect scalar code selected u8 rather than i32. The final fixture uses
48 structs/functions (96 declarations) and scalar code 1. These earlier runs
are not PASS evidence and required no production edits.

## Setup and memory bounds

Source inspection establishes `2*(G+1)` initialized slots and a setup charge of
`2*(G+1)+G = 3*G+2`: zero-fill visits plus one visit per prefix record. For
`G<=4096`, that is at most 8194 initialized slots / 65,552 bytes of logical
usize payload on Windows AMD64, and 12,290 charged setup visits. The test observes
total Interfaces work of 142 and 2821, including other builder work. It cannot
isolate index setup through the public API, and does not claim to do so.

`internal/cstdlib/stage0_vector_storage.c::fy_private_vector_reserve_v0` grows
capacity by powers of two from eight slots. A fresh 8194-slot reservation can
therefore hold 16,384 usize elements: **131,072 bytes (128KiB) reserved payload**.
Allocator bookkeeping and the vector owner are additional. 65,552 bytes is
neither a total memory cap nor a measured peak. OOM behavior and exact allocator
overhead were not fault-injected. The 4096 boundary is exercised on ValueGraph;
the largest real Interfaces prefix constructed by this fixture is 146 types.

## Exact tested inputs

| File | SHA256 |
| --- | --- |
| module_unit_interfaces.fy | d91bbca2088642763570f300dcef102a6a1e7ecc91d9147fd9ee603771fb26ea |
| module_unit_value_place.fy | ea3102b7db1ed836bb17918918bca50df767a0f21b6bc70e0203793fc823b450 |
| module_unit_value_scalar.fy | 606b68e7086a41c4ad85f576f35a61c1b8303e5536a7eb906e678a9c51b9833e |
| borrow_type_index/main.fy | 4389c39c404d53d31cba86f68e741f149c5140d5a3dce85a0b4f13b7cf74d7af |
| low temporary PE | 47d2dd4e3d1b769951e4018a9c582be76dbb8f430efb7984c0f665cad364d847 |
| optimized temporary PE | dcb3e91824fc17448d865562610f1d3de1c682389ae77e75351afffbf4b8b46c |

Full source check/lock and focused A/B are parent-owned evidence. No Linux run,
new whole-CLI seed, full authored self-build, parity suite, or fixed-point run
was performed by this sidecar. Full acceptance remains pending integration.

## Parent integration checks

The parent runs full isolated source check/locked in this worktree: 13.172s and
14.832s, 755 files and 9510 MIR functions, all 1225 source hashes matching. The
verified lock is copied back only after both checks. Evidence is
`.cache/borrow-index-source-check.log`. Scoped
`go vet -mod=readonly ./internal/cli` and `git diff --check` pass too.
The focused A/B uses a newly compiled 40-part optimized probe, preserving all
original input bytes. It remains diagnostic evidence; the full authored CLI
seed, normal self-build and consecutive-generation acceptance are later gates.
