# Shared struct reference iteration

`tokens::find` searches a borrowed token slice and returns `Option<ref Token>`.
Each loop item points at original stored data, including the nested `Span`.
The item is not a whole-token copy. A caller can explicitly copy with `*token`.

Run from this directory with a compiler implementing ADR 0459:

```sh
fairyc check
fairyc run --locked
fairyc run --release --locked
```

Expected stdout: `kind=2 value=4000000002 span=2..12` followed by a newline.
Exit is 0; stderr is empty. Windows dev/release execution is verified.
Linux execution is deferred to the maintainer's batch.

Shared items traverse named immutable `ref Token[]` or `ref mut Token[]`
bindings. Referents must be nonempty acyclic plain structs with scalar leaves.
Writes and exclusive borrows through the owner/view are blocked during the loop
and while returned references remain live. Local-owner references cannot escape.
Resource/pointer/borrowed/array/tagged fields, mutable descriptor bindings,
computed iterable roots and fixed struct arrays remain outside this feature.
