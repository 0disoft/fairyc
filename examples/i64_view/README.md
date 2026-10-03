# Read-only i64 array views

`ref i64[]` borrows a named fixed array without copying its elements. The example
reads signed extrema, sums a subrange, passes/returns views, checks empty/nested
ranges and edits the owner after the last view use.

```fairyc
let i64[3] values = [4000000000, -3, 5];
let ref i64[] view = ref values;
let ref i64[] tail = view[1..];
assert(tail.len == 2 && tail[0] == -3);
```

Use the ordinary `fairyc check`, `fairyc build`, and `fairyc run` commands from
this directory; use `--release` for an optimized build/run. Windows execution
has been verified. Linux execution is deferred for batch validation.

Indices and lengths count elements. Invalid bounds trap. A live view prevents
owner mutation, and a view of local storage cannot escape its function.
Mutable i64 views, Vec views, other slice element types and byte-iteration syntax
are outside this first i64 admission. See [ADR 0430](../../docs/adr/0430-shared-i64-array-views.md).
