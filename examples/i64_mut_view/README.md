# Mutable i64 array subviews

This Fairy C example reverses only the selected part of a fixed array. It uses
`ref mut i64[]`, checked indexing/stores, nested and returned subviews, a shared
reader, and values outside i32 including both signed extrema. Assertions check
untouched endpoints and the original array after the last view use.

From this directory, with the revised compiler on PATH:

```text
fairyc check
fairyc build
fairyc run --locked
fairyc build --release --locked
fairyc run --release --locked
```

Successful execution returns 0 with no output. A live child freezes its parent
and owner; retained disjoint siblings are also rejected. No array data is copied
by view creation. Mutable Vec conversion and mutable for-in are not provided.

Windows MSVC development/release execution is verified. Linux execution remains
for the later batch. Windows optimized mutable i64 view access uses an internal
volatile-access workaround; its optimization cost has not been measured.
See [ADR 0433](../../docs/adr/0433-exclusive-i64-array-views.md).
