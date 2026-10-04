# ADR 0467: Direct array subview borrows

- Status: implemented; Windows dev/release runtime validated; Linux execution deferred to the owner's batch
- Date: 2026-10-05
- Related: ADRs 0415, 0455, 0457, 0461, 0462, 0464, 0465, 0466

## Decision

Admit an explicit `ref` / `ref mut` borrow of a slice range over a bare named local or
by-value parameter fixed array as a view of the original storage:
`let ref Point[] middle = ref points[1..3];` and `adjust(ref mut points[1..3]);`.
Grouping around the array root is preserved, so `ref ((points)[1..3])` selects the same
storage.

Element types are exactly the existing admitted fixed-array set: the `bool`, `u8`, `i32`,
`i64`, `u32` and `usize` scalars and nonempty acyclic plain scalar-field structs
(ADR 0462). Bounds may be explicit, half-open (`..3`, `2..`) or fully omitted (`..`), and an
empty range such as `points[4..4]` is valid. Bounds must be `usize` (`E1006` otherwise) and
evaluate exactly once, start then end, before the length snapshot; the runtime checks
`start <= end <= len` and traps `bounds` on failure.

```fairy
var i64[4] data = [10, 20, 30, 40];
let ref i64[] middle = ref data[1..3];
var usize calls = 0;
let ref mut i64[] window = ref mut data[{ calls += 1; 0 }..{ calls += 1; 2 }];
window[0] = 11;
```

The root must resolve to a bare name; computed, call, temporary, field and nested-slice roots
stay closed, and ordinary array slicing outside this explicit `ref` / `ref mut` form remains
closed. Exclusive borrows require a mutable (`var`) owner (`E1008`); a by-value parameter array
supports shared borrows only.

## Evaluation and lowering

`internal/semantic/array_subview_borrow.go` recognizes the prefix borrow, unwraps grouping to the
slice base, and requires a resolved local or parameter of admitted fixed-array type. It lowers the
borrow to the existing `array-view-borrow` operation whose child is the resolved array root, then
`checkSliceRange` in `internal/semantic/typecheck.go` checks both bounds as `usize` and builds the
existing `view-slice` HIR. Reusing those operations means no copied array, temporary address, new
opcode, wire field, SDK export or runtime helper: the emitted MIR is the same array-view-borrow plus
view-slice used by named view bindings and slice iteration (ADR 0465).

Omitted bounds lower to the existing `omitted-range-bound` node, zero for start and the descriptor
length for end. `internal/mir/lower.go`'s `lowerViewSlice` keeps the terminating `blockTerminated`
guard, so a bound that returns early skips the remaining bound and the slice operation without
leaving an invalid operand.

## Borrows

The whole array is the loan owner and array indices never prove disjointness. A shared subview
coexists with shared reads and borrows of the owner and with sibling shared subviews, but an owner
write or exclusive borrow while it is live is `E2101`. An exclusive subview carries the `subview`
generation flag, freezing the parent descriptor and its aliases while the reference is live under ADR
0415, so owner reads and writes and every competing borrow are `E2101`. A local or by-value parameter
view cannot escape its owning frame, so returning it is `E2102`; retained references and
`Option<ref T[]>` aliases keep the owner loan alive through their last use.

## Compatibility

No syntax production, MIR operation, wire field, artifact schema, SDK export, runtime symbol or host
flag is added. Language `/20`, SDK `stage0.86`, runtime `/0`, public ABI and target maturity stay
unchanged, and this is an additive admission. Exact compiler revision/executable identity separates
cached generations.

## Verification and limits

Semantic rejection tests cover the immutable exclusive source (`E1008`), width and bound-type
mismatches (`E1006`), local, mutable and by-value parameter escapes (`E2102`), the closed roots
(temporary, literal, nested-slice, ordinary slice, field) and the `E2101` owner, sibling, bound and
alias conflicts. Compiler round-trip tests exercise the MIR codec, interpreter execution and both
the Windows MSVC and Linux C emission profiles, and reject a MIR mutation that removes the
`view-slice` bounds check; the bound-return cases are interpreted and C emitted.

Runtime validation is recorded. The Windows native matrix produced 16 outcomes across 8 cases and
both dev and release profiles, the standalone public CLI `check`/`build`/`run` recording passed for
dev and release with the five rejected sources rejecting as expected, and the related suite passes
355 tests with no failures or skips. Compiler build and vet pass. Both the Windows MSVC and Linux C11
profiles emit; Linux actual execution is intentionally deferred to the owner's consolidated batch, so
only C emission is verified for Linux. Evidence is recorded at
`docs/ops/evidence/array-subview-borrow-20261005.json` with `.cli.json` and `.native.json`
companions.
