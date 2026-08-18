# Resource Abilities v0

- Status: Draft experimental contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0335-resource-abilities-and-linear-assets.md`
- Diagnostic source: `docs/diagnostics/catalog-v0.md`

## Purpose

Ordinary Fairy C resources model unique ownership plus compiler-scheduled cleanup. That is correct for files, buffers, arenas, and operating-system handles, but it is insufficient for conserved assets. A token, capability ticket, or ownership proof must not disappear merely because control reaches the end of a lexical scope.

Resource abilities let a declaration opt into a closed set of compiler-known properties. The first slice controls whether a resource may be destroyed implicitly and reserves storage identities needed by later contract profiles. It does not add copying, chain storage, serialization, or a blockchain runtime.

## Declaration syntax

Abilities are written as one resource-owned attribute:

```fy
@abilities(store, key)
resource Coin {
    u32 amount;

    cleanup {}
}
```

The closed v0 vocabulary is:

| Ability | Meaning |
| --- | --- |
| `drop` | Compiler-scheduled destruction is permitted when a live value reaches a cleanup edge. |
| `store` | The declaration is intended to be eligible for later persistent-storage profiles. No storage backend or layout promise exists in v0. |
| `key` | The declaration is intended to be eligible as a later top-level storage identity. `key` requires `store`. |

A resource remains move-only regardless of its declared abilities. `copy` is not an ability and is rejected. Abilities are not user-defined traits, methods, interfaces, reflection metadata, or runtime flags.

## Backward-compatible default

A resource without `@abilities` retains the existing Stage 0 behavior and permits implicit destruction:

```fy
resource File {
    ptr mut void handle;

    cleanup {
        // close the handle
    }
}
```

The compatibility default is equivalent to having `drop` for cleanup analysis. It does not synthesize `store` or `key` metadata. New conserved-asset declarations must spell an explicit ability set that omits `drop`.

## Linear destruction rule

When an explicit ability set omits `drop`, a live value cannot reach any compiler-scheduled cleanup edge. Rejected edges include normal function exit, explicit return that leaves another value live, nested block exit, `break`, `continue`, and replacement of a live resource during reassignment.

```fy
@abilities(store, key)
resource Coin {
    u32 amount;
    cleanup {}
}

fn void lose(Coin coin) {
    // E1108: `coin` would be destroyed at function exit.
}
```

Whole-root ownership transfer remains valid. Moving a value into another local, passing it by value, or returning it transfers the single obligation rather than destroying the asset:

```fy
fn Coin forward(Coin coin) {
    return coin;
}
```

## Recursive containment

The no-drop rule follows compiler-owned cleanup recursively. A droppable wrapper cannot launder destruction of an inner linear asset:

```fy
@abilities(drop)
resource Vault {
    Coin coin;
    cleanup {}
}

fn void lose(Vault vault) {
    // E1108 relates the diagnostic to Coin's ability declaration.
}
```

The same rule follows resource-bearing `Option<T>` and `Result<T, E>` values. Cleanup permission is therefore a property of the complete owned value graph, not merely its outer nominal type.

## Declaration validation

`@abilities` must appear at most once, contain at least one identifier, use only the closed vocabulary, contain no duplicate ability, and declare `store` whenever `key` is present. Violations are `E1107` and prevent later Flow from manufacturing secondary cleanup errors.

An implicit cleanup of a valid explicit no-drop value is `E1108`. The primary span identifies the cleanup edge. A related `ability-declaration` span identifies the declaration that withheld `drop`, including an inner resource reached through containment.

## Explicit destruction boundary

`discard`, an unsafe block, an empty cleanup body, and raw-pointer conversion do not grant permission to destroy a no-drop resource. A future explicit `burn` operation must discharge one ownership obligation in verified MIR and must be authorized by a declaration-owned API or contract capability. Implementing it as a generic escape hatch would recreate the exact asset-loss bug this feature prevents.

No explicit burn operation is admitted by this v0 slice. Until a separate decision defines its source contract, MIR operation, verifier rule, and backend behavior, a no-drop resource must be transferred or returned.

## Deferred work

Persistent state, account keys, storage layout, serialization, minting, burning, splitting, merging, event emission, contract entry points, gas accounting, and chain-specific ABI generation remain outside this contract. `store` and `key` are validated metadata only and create no backend promise.
