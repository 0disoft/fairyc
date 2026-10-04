# Integer view iteration

Clips integer readings in place through `for (let ref mut i32 item in values)`.
The fixed array produces a named mutable view; the function updates its stored
cells, and a subsequent shared view prints them. No extra array is allocated.

From this directory, run `fairyc check`, `fairyc build --locked` and
`fairyc run --locked`. Use `--release` on build/run for the release profile.

Expected stdout:

```text
0
0
42
9
100
```

This syntax requires ADR 0454. The source works for Windows and Linux targets;
Windows dev/release execution is verified here, Linux execution is deferred.
