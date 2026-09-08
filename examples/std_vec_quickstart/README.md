# Scalar vector quickstart

Requires the SDK bundled with Fairy C `0.0.0-stage0.58` and Windows AMD64 MSVC.
From this directory:

```text
fairyc check
fairyc run --locked
fairyc test --native --locked
```

The program prints `42` without a newline and exits successfully. It constructs
an empty vector, handles both push results, borrows an element, pops the last
element, and clears the remaining initialized prefix. Dropping the vector frees
its retained allocation. Add `--release` to run or native test for optimization.

`push<T>` returns the original element in `.err` if allocation fails. This small
example deliberately asserts success; applications should recover or propagate
that error. `at<T>` traps for an invalid index. A live element reference prevents
vector mutation or movement. Mutable access, iteration, nested Vec types and
non-Windows execution are not part of this release.
