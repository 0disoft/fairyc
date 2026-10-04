# Integer element references

A bounded byte histogram CLI written in Fairy C. It increments stored `usize`
counters through `ref mut counts[index]`, passing the selected counter to a
function. Input bytes are read through shared `ref input[index]` references.
The counters use a fixed 256-element stack array; no heap-backed counter vector
is needed. The argument buffer is explicitly allocated through the public SDK.

```text
fairyc check
fairyc run --locked -- ababa
```

Expected stdout (TAB-separated decimal byte value/count):

```text
97	3
98	2
```

No argument uses `Fairy C`. An empty argument produces empty output. The single
text argument is bounded to 4096 UTF-8 bytes; counts are byte frequencies, not
Unicode character frequencies. Output is sorted by byte value, excludes zero
counts and ends each row with LF. Two or more arguments print usage and return
2. Allocation, argument-copy or output-write failure returns 2. Successful
execution returns 0. All parsing/counting/output logic is Fairy C and public SDK.

Build/run with `--release` for the optimized native profile. Windows execution
was verified; Linux execution is deferred to the owner's batch. ADR 0453 owns
the language behavior. The existing five integer view types support element
references; this example uses u8/usize. No direct fixed-array element references
or reference for-in loops are introduced: first convert an array to a named view.
