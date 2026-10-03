# Shared i64 vector views

`std::vec::view_i64(ref values)` returns `ref i64[]` over the initialized
elements of a `Vec<i64>`. It does not allocate or copy elements. Indices and
half-open slice bounds count elements, with the usual `bounds` trap on invalid
ranges. Empty vectors and empty end slices are supported.

```fy
let ref i64[] numbers = vec::view_i64(ref values);
let ref i64[] selected = numbers[1..3];
```

While a view, alias, returned view or subview is still used, the compiler rejects
vector mutation and move, including `push`, `reserve`, `pop`, `truncate`, `clear`
and `at_mut`. After the last use, mutation is allowed; obtain a new view after
growth. A local vector's view cannot escape its owner. There is no mutable i64
view or generic `Vec<T>` slice conversion.

From this directory, use `fairyc check`, `fairyc build`, `fairyc run --locked`
and `fairyc run --release --locked`. The example checks signed extremes,
values above i32, subrange sums, shared siblings, nested and tagged views,
growth beyond the old capacity and mutation after the last use.

Windows development/release execution is covered by `TestVecI64ViewNative`.
The same cases are ready for the later Linux validation batch; Linux execution
has not been verified in this change.
