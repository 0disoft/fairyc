# Read-only i64 view iteration

```fy
for (let i64 value in view) {
    total += value;
}
```

The iterable must be a named immutable local or parameter of exact type
`ref i64[]`. Bind array/Vec views or subranges to a name first. Each iteration
copies one i64 in index order; an empty view executes no body. The item's scope
is the body. Length is observed once, with checked indexing and arithmetic.

The owner stays borrowed throughout the loop, including a body ending with
`break` or `return`. Mutation/move may occur after loop exit if there are no
later view uses. `continue` advances once, `break` skips the advance, and function
exits preserve the existing defer/resource cleanup order.

From this directory: `fairyc check`, `fairyc build`, `fairyc run --locked`,
and `fairyc run --release --locked`. The example checks fixed-array views,
subranges, empty/singleton slices, signed extremes, ordered summation, nested
shadowed bindings, early return and post-loop mutation. `examples/vec_i64_view`
uses the same loop form over the SDK vector view.

Windows native cases and compiler interpreter/codec tests also check positive
and negative sum overflow. Linux native execution is deferred to the later
validation batch. There is no direct array/Vec iteration, mutable item, callback
or generic iterator protocol in this feature.
