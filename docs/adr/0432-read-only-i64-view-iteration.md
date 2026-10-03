# ADR 0432: Read-only i64 view iteration

- Status: Implemented (Windows native evidence; Linux execution deferred)
- Date: 2026-10-03
- Related: ADR 0403 byte-view iteration, ADR 0430 array views, ADR 0431 vector views

## Decision

Admit the already parsed form `for (let i64 value in view)` for a named immutable
local or parameter of exact type `ref i64[]`. Each body entry copies one checked
i64 load in index order. Observe length once before entering the hidden scalar
loop; empty views execute no body. The item is immutable and scoped to the body.
Signed extrema and values beyond i32 preserve their full width.

This works for fixed-array or SDK vector views and named subranges. Direct
array/Vec values, mutable view bindings, slice/call expressions in the iterable,
reference items and other item types remain unadmitted. Bind a view expression
to an immutable local first. Existing u8 iteration is preserved.

## Ownership and exits

The i64 loop holds the shared owner until every loop-body exit, including
`break`, `return` and error propagation. Mutation/replacement/move inside the
body is rejected even if no next iteration is possible. After loop exit,
mutation is permitted when no later view/subview use retains the loan.
Nested loops retain their respective loans and preserve item scope/shadowing.

`continue` advances the hidden usize index once; `break` skips that advance.
The largest executed index is `len - 1`, so the final increment remains
representable for `len == SIZE_MAX`. Ordinary checked i64 arithmetic still
traps on overflow. Defer, resource cleanup and Region reset use the existing
scalar-loop exit paths; there is no iterator allocation or additional runtime.

## Implementation and identities

AST preparation extends the existing byte-loop path to i64 while preserving
the written source node. Semantic admission matches the exact item/view pair.
Flow explicitly holds the i64 view through evaluation of the prepared loop,
including early exits. The legacy `ByteForLoop`/`byte-for` internal names remain;
renaming them or changing serialized shapes is unnecessary for this addition.
HIR/MIR use the existing length snapshot, usize counter, checked view load,
scalar loop and cleanup edges. The interpreter and C backend need no new opcode.

Language `/20`, SDK stage0.80, runtime `/0`, parser tokens and MIR/artifact schemas
remain unchanged. This extends the admitted type of an existing parsed form,
following the additive i64-view admissions. Exact compiler-executable identities
separate disposable caches. The vector sum example now uses this source form;
SDK source bytes and API are unchanged. No performance improvement is claimed.

## Evidence and limits

`examples/i64_for` checks empty/singleton/subrange and full-array sums, negative
values, signed extrema, values above i32, continue/break, nested shadowed items,
early return and owner mutation after the last loop use. The SDK vector-view
example executes its sum through the same loop.
Compiler tests execute a verified MIR codec round trip and emit both Linux and
Windows C profiles. Interpreter and Windows low/optimized native cases check
sum overflow and underflow. Cleanup observers compare exact defer/drop order
on continue, break, early return and Result propagation in interpreter and
both Windows profiles.
Source rejections cover wrong item/view/source forms, mutable item/scope escape,
array mutation and Vec mutation/replacement/move, including early loop exits.
Native images, source/image hashes, toolchain identity and raw outcomes are
retained under `.cache/i64-for-native/windows/`. Linux native test cases are
ready but actual execution is deferred to the user's batch.
