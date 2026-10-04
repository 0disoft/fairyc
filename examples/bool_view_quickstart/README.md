# Bool views: occupancy flags

Mark a bounded slot range, count occupied slots, and locate the next free slot.
The program uses a repeated fixed-array initializer, shared and exclusive bool
views, offset subviews, and copied/shared/exclusive iteration. All product logic
is Fairy C; no script or C shim implements the flag operations.

From this directory, using the current compiler:

```sh
fairyc check
fairyc run --locked
fairyc run --release --locked
```

Expected stdout (exit 0, empty stderr):

```text
occupied=3 next=5 after=4
```

`ref bool[]` and `ref mut bool[]` carry a pointer and length into original
storage. They allocate no backing storage and do not pack bits. A mutable
subview temporarily freezes its parent while live. A shared-reference loop
blocks writes through the owner until loop exit. An exclusive item can update
its original cell; copied items remain independent values. Indices and slice
bounds trap on invalid ranges. Arrays retain the current 1..=4096 element guard.

Loop roots must be named fixed arrays or immutable named view bindings. Create
an immutable named subview before iterating a range. Simultaneous mutable sibling
views are still unsupported. See ADR 0461 for the exact admitted surface.

Windows dev/release execution is verified. Linux actual execution is deferred
to the owner's consolidated batch; the tests use the existing portable native
harness so they can run in that batch.
