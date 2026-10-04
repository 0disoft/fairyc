# Copied items from a mutable struct view

Each `for (let records::Record before in items)` iteration copies the current
stored record. The example clamps its original value, reports the saved before
value alongside the updated value, and verifies nested-field copy independence.
A retained `Option<Record>` remains valid after the original Vec is cleared.

With a compiler implementing ADR 0460, run from this directory:

```sh
fairyc check
fairyc run --locked
fairyc run --release --locked
```

Expected stdout (exit 0 and empty stderr):

```text
before=-5 after=0
before=4000000000 after=10
before=7 after=7
```

The iterable is an immutable named `ref mut Record[]` descriptor. Items are
immutable independent values; use a separate `var Record copy = before` to edit
a copy. Writes through the mutable descriptor are allowed and later iterations
observe earlier changes to not-yet-visited elements. The whole Vec stays borrowed
through loop exit, so clear/reserve/move remain rejected inside the loop.
References to a local copied item cannot escape. Existing live element-reference
loans still restrict competing writes. Referents remain plain scalar-leaf structs;
resource graphs, computed iterable roots and fixed struct arrays are excluded.
Windows dev/release execution is verified; Linux execution is deferred.
