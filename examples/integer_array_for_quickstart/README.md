# Direct integer array iteration

Clamp five stored i64 values into `[0,10]`, then print them using the SDK's
allocation-free i64 writer. Both loops use the array directly.

```fairy
for (let ref mut i64 value in values) {
    if (*value < 0) { *value = 0; }
    if (*value > 10) { *value = 10; }
}
for (let i64 value in values) {
    discard io::write_i64(value);
}
```

From this directory use `fairyc check`, `fairyc build --locked` and
`fairyc run --locked`; build/run also accept `--release`.
The full example checks output failures and writes newlines explicitly.
Expected stdout is `0\n0\n10\n10\n7\n`; stderr is empty.

Direct iteration supports named nonempty fixed arrays of u8, i32, i64, u32
and usize. Value items are copies; mutable reference items update original
storage and require a mutable array binding. The array is borrowed through
loop exit. Copied loops permit shared reads but prevent mutation; mutable
loops prevent separate original-array access. After the loop the array can
be used normally. Bool/struct arrays, shared reference items, literal/call
iterables and array fields are outside this feature.
