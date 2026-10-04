# Plain struct arrays and views

Translate a fixed polygon range and sum its x coordinates without a Vec
allocation. Product logic is Fairy C in `src/main.fy` and `src/points.fy`.

```sh
fairyc check
fairyc run --locked
fairyc run --release --locked
```

Expected stdout, exit 0 and empty stderr:

```text
vertices=4 sum_x=16000000020
```

Plain structs with bool/integer leaves, including nested plain structs, support
fixed arrays, value copies, list/repeated initialization, whole-element stores,
shared/exclusive views and copied/shared/exclusive loops. Repeated initialization
evaluates once, then makes independent element values. `src/main.fy` updates
stored fields directly before translating and summing:

```fairy
polygon[0].y = 9;
polygon[0].x += 10;
polygon[3].x -= 10;
```

Direct indexed field stores (`polygon[i].field = value`), nested scalar and
nested plain-struct paths and checked scalar compound assignment write into the
array element itself. An exclusive element reference or a mutable view can change
stored fields too. These direct updates leave the stdout and exit status
unchanged.

A direct field reference lends a stored field in place instead of copying the
whole element, for example `ref polygon[0].x`, `ref mut polygon[3].x` or a nested
path such as `ref mut polygon[0].position.x`:

```fairy
points::add_x(ref mut polygon[0].x, 10);
points::add_x(ref mut polygon[3].x, -10);
let ref i64 first_x = ref polygon[0].x;
assert(*first_x == 4000000010);
```

Shared field references come from immutable or by-value array bindings; mutable
ones need a mutable array binding. Fields and indices never prove disjointness,
so a live mutable field reference blocks other access to the same array until its
last use, while a shared field reference blocks writes to the array. These
updates leave the stdout and exit status unchanged.

The current array bound is 1..=4096. Exact nominal types remain distinct, even
when layouts match. Resources, pointers, reference-typed fields, tagged fields,
nested arrays and arrays inside struct fields remain unsupported. Whole-array
copies have a cost proportional to array contents; borrowed views do not copy
backing storage. Bounds are checked. Local storage references cannot escape, and
simultaneous mutable sibling loans remain unsupported.

Windows dev/release execution is verified. Linux execution is deferred to the
owner's consolidated batch. See ADR 0462, ADR 0463 and ADR 0464.
