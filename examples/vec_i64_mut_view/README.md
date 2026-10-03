# Mutable i64 Vec views

This Fairy C example modifies the initialized prefix through
`std::vec::view_mut_i64`. It reverses a subrange while preserving endpoint
extrema, changes nested/returned/tagged views, passes a shortened shared reader,
and checks the original Vec after loan release. It grows the allocation,
obtains a fresh mutable view, and performs later structural operations/move.

From this directory with the revised compiler on PATH:

```text
fairyc check
fairyc build
fairyc run --locked
fairyc build --release --locked
fairyc run --release --locked
```

Successful execution returns 0 with no output. A live mutable view blocks all
other owner access, even reserve/truncate operations that would do nothing.
Subviews allocate nothing and copy no elements. Retained siblings are rejected
conservatively; generic mutable Vec slices and mutable for-in are not supported.

Windows development/release execution is verified. Linux execution is deferred
to the later batch. See [ADR 0434](../../docs/adr/0434-exclusive-i64-vector-views.md).
