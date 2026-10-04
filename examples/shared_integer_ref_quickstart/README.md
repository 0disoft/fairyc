# Shared integer reference iteration

`find` searches without copying the array and returns a reference to its original
stored element in `Option<ref i64>`. `for (let ref T item in source)` supports
u8, i32, i64, u32 and usize over a named fixed array or immutable named view
binding (`ref T[]` or `ref mut T[]`). Shared items cannot be written through.
Owner writes/exclusive reborrows are blocked until the loop and any live returned
references end. References to local or by-value array storage cannot escape.

From this directory, run `fairyc check`, then `fairyc run --locked` or
`fairyc run --release --locked`. Expected stdout is `4000000000` followed by a
newline, exit 0 and empty stderr. Requires the compiler implementing ADR 0458.
Windows dev/release execution is verified; Linux execution is deferred.
