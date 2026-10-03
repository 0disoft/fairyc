# Integer vector views and iteration

SDK stage0.83 exposes `view_i32`, `view_mut_i32`, `view_u32`, `view_mut_u32`,
`view_usize` and `view_mut_usize` in `std::vec`. Each function borrows only the
initialized prefix of its exact Vec type, without allocating or copying.
Lengths and bounds count elements. Empty vectors and empty subranges are valid.

Named immutable descriptors and parameters support `for (let T item in view)`
for shared/exclusive u8, i32, i64, u32 and usize views. Each iteration copies one
checked value; stores to future positions affect later loads. The owner loan
lasts through the loop, including break/continue/return. Direct iterable
expressions, reference items and mutable descriptor bindings are unsupported.

This example checks all three new Vec types, fixed-array iteration, extrema,
shared/exclusive empty loops, singleton loops, returned/tagged origins,
subranges, shared reader reborrowing, in-loop writes, release followed by
reserve/growth, push/pop/truncate/clear, replacement and consuming move.

From this directory:

```sh
fairyc check
fairyc build
fairyc run --locked
fairyc build --release --locked
fairyc run --release --locked
```

Portable repository verification:

```sh
go test ./internal/compiler -run 'Test(IntegerStorageViewsMIR|IntegerViewForInterpreter)' -count=1
go test ./internal/cli -run '^TestVecIntegerView' -count=1
```

The native harness supports Windows amd64 MSVC and Linux amd64 GCC/Clang.
Linux execution is deferred to the user's batch. Live shared views block
mutation/replacement/move, and exclusive views also block other owner access.
ADR 0415 sibling/ancestor rules still apply. There is no generic Vec-to-slice
conversion, reinterpretation, new C runtime function or foreign ABI.
See ADR 0438 and the linked evidence receipt.
