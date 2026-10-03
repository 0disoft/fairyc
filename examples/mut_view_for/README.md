# Mutable view iteration

```fy
fn void increment(ref mut i64[] view) {
    var usize index = 0;
    for (let i64 value in view) {
        view[index] = value + 1;
        index += 1;
    }
}
```

A named `let` binding or parameter of `ref mut u8[]` or `ref mut i64[]`
can be iterated. The item is an immutable scalar copy, not an element reference.
Length is observed once; each iteration loads the next current element in index
order. Writing a later element affects its later load. Use `view[index]` to
modify elements; assigning to the item is rejected. Empty views do not run the
body. Bind a slice or function result to a name before iterating it.

The loop retains the owner loan until exit, even before `break`, `return` or
Result propagation. Owner mutation/reallocation/replacement/move and live
parent/sibling subview conflicts remain rejected. Element stores, short calls
using the same exclusive view, and shortened shared readers are permitted when
ordinary reborrow rules allow them. Mutating a `var` descriptor binding during
iteration is not admitted. After exit the normal last-use rules apply.

Run `fairyc check`, `fairyc build`, `fairyc run --locked`, and
`fairyc run --release --locked` here. The example checks u8 mutation, modification
of a future element, independent item copies, full-width signed i64 values,
empty/singleton/subrange views, nested loops, continue/break/return/propagation,
and owner access after exit. SDK `Vec<i64>` iteration and numeric traps are
covered by `TestMutableViewForNative`; cleanup ordering by
`TestMutableViewForCleanup`. These Go tests also serve the later Linux batch;
Linux native execution is currently deferred.
