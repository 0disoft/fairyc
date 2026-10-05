# ADR 0465: Direct slice and subview iteration

- Status: implemented; Windows native and public CLI verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0457, 0458, 0459, 0461, 0462

## Decision

Admit `for (let T item in source[start..end])` over a bare named fixed array
or an immutable named `ref T[]` / `ref mut T[]` view binding. Bounds may be
explicit or omitted, matching ordinary slicing. The root must resolve to that
bare name; computed, call, field and nested-slice iterable roots and mutable
(`var`) descriptor bindings remain closed. ADR 0470 extends the closed field-root rule for a
direct `for` over a named plain struct scalar-array field (ADR 0468). The item set is exactly the existing
`bool`, `u8`, `i32`, `i64`, `u32` and `usize` scalars and nonempty acyclic
plain scalar-field structs, copied, shared-reference or exclusive-reference.

```fairy
var i32[8] data = [0, 1, 2, 3, 4, 5, 6, 7];
for (let i32 value in data[1..3]) { assert(value >= 1); }
for (let ref i32 value in data[..]) { assert(*value >= 0); }
let ref mut i32[] window = ref mut data;
for (let ref mut i32 value in window[2..]) { *value = *value + 1; }
```

A whole named array is borrowed implicitly as shared for copied and shared
items; exclusive `ref mut` items require a mutable array. Slicing a named view
preserves the parent view's mutability even for copied or shared items, so a
`ref mut T[]` binding always yields a mutable subview generation. That
generation freezes its parent descriptor and every alias for the written loop's
duration under ADR 0415; mutable siblings never prove disjointness. Owner
loans, `break`/`continue`/`return` cleanup, Result propagation and local or
by-value escape rules are inherited unchanged from the named-array and
named-view loops.

## Evaluation and lowering

The written slice evaluates its receiver, explicit `start` and explicit `end`
exactly once, in that order, before the length snapshot and every element access;
an omitted `start` is zero and an omitted `end` reads the selected descriptor's
length. The runtime checks `start <= end <= len` before the loop body, so omitted
bounds and empty subviews are valid and an out-of-range slice traps with `bounds`
before any element is addressed. Length is captured once and indices advance in
ascending order.

An array root slices through a private typed whole-array view binding created
inside a source block expression, with the written slice as the block tail;
ordinary array slicing outside this loop form stays closed.
`internal/ast/byte_for_lower.go` extends the prepared view source to accept a
three-child slice whose base is a named path and keeps that written slice in the
descriptor. `internal/semantic/slice_for.go` validates the resolved root and
element and builds the descriptor: a named array root gets the private view
binding, an immutable named view root is sliced directly, and both reuse the
resolved header tree so effectful bounds keep their symbols. `array_for.go`'s
shared `prepareForDescriptor` hoists the private outer subview exactly once ahead
of the header, and `resetArrayForPreparation` discards the prior synthetic header
entirely and recreates it from the retained written `ByteFor`, so repeated
checking of the same retained AST yields identical encoded MIR.

A bounds expression that returns early previously left the slice with an invalid
operand. `internal/mir/lower.go`'s `lowerViewSlice` now stops after lowering the
receiver, start and end once the block has already terminated, matching the
existing `lowerCall` guard (ADR 0464), so no invalid operand reaches the slice
operation and the remaining bounds and body are skipped.

No syntax production, MIR operation, wire field, artifact schema, SDK export,
runtime symbol or host flag is added; the existing view-slice, checked-bounds and
element-address operations are reused. Language /20, SDK stage0.86, runtime /0,
public ABI and target maturity stay unchanged, and this is an additive admission.

## Verification and limits

The related Go suite passes 994 tests with no failures or skips:

```text
bun tools/run-go.ts test -json ./internal/ast ./internal/semantic ./internal/compiler ./internal/mir ./internal/interpreter ./internal/cbackend -run '^Test(.*For.*|.*Slice.*|.*Subview.*|.*Array.*|.*Indexed.*)$' -count=1
```

The Windows native harness passes 10 Go test entries with no failures or skips:
8 cases run in both dev and release for 16 actual outcomes plus a separate cache
test. The cases cover the matrix, a scalar generic mutable-view subview,
past-end, reversed, start-past-end, checked i64 overflow, array-bound-return and
view-bound-return.

The standalone public CLI passes the matrix `check`/`build`/`run` in dev and
release for 2 outcomes and rejects five forms: an immutable exclusive source, an
owner write (`E2101`), a local escape (`E2102`), a bool bound (`E1006`) and a
nested-slice root. It records raw UTF-8 argv, cwd, stdout, stderr, exit status
and the EXE SHA-256 together with the compiler identity receipt at
`.cache/slice-for-cli/receipt.json`. The native harness and public CLI account
for 16 and 2 real outcomes respectively.

Compiler round-trip tests confirm retained-AST MIR byte identity, MIR
codec/interpreter execution and both Windows MSVC and Linux C emission. A
verifier mutation that removes the slice guard is rejected by the verifier, codec
and backend, and the start-return and end-return cases are interpreted and C
emitted. A bound expression that returns early previously left `lowerViewSlice`
with an invalid operand; the terminating `blockTerminated` guard fixed that.
Formatting checks, the diff check and `go vet` pass. Evidence is recorded as
`docs/ops/evidence/slice-for-20261005.json` with its `.cli.json` and
`.native.json` companions. Linux actual execution remains deferred to the
owner's batch. No benchmark, installation, CI change, push or release is
included.
