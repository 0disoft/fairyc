# Integer fixed-array views

`ref i32[]`, `ref u32[]`, `ref usize[]` and their `ref mut` forms borrow
matching fixed arrays. Indices and lengths count elements. No allocation or
copy is needed for the pointer/length descriptor. The example checks signed
and unsigned boundaries, shared readers, mutable subranges, tagged returns,
empty ranges, ordered bound evaluation and owner reuse after the last view use.

From this directory, using a freshly built Fairy C compiler:

```sh
fairyc check
fairyc build
fairyc run --locked
fairyc build --release --locked
fairyc run --release --locked
```

Portable Go verification from the repository root:

```sh
go test ./internal/compiler ./internal/semantic -run 'IntegerView' -count=1
go test ./internal/cli -run '^TestIntegerViewsNative$' -count=1
```

The native test runs development and release profiles on Windows MSVC or Linux
GCC/Clang; Linux execution is deferred to the owner's later batch. Bounds trap,
shared stores are rejected, and live loans block owner replacement/writes.
Exclusive parent and sibling conflicts follow ADR 0415. Fixed arrays retain the
4096-element implementation guard. Bool, aggregate views and implicit conversion
are outside this admission. See ADR 0437 and the evidence receipt.
