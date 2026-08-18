# Contract State Transaction Kernel v0

- Status: Experimental executable contract
- Product source: `docs/product/04-contract-profile-candidates.md`
- Decision source: `docs/adr/0335-typed-persistent-state-transaction-kernel.md`
- Implementation: `internal/contractstate`
- Compatibility identity: `fairyc-contract-state/*/v1` domains

## Boundary

The kernel defines host-independent layout and transaction behavior for a future Fairy C contract
profile. It deliberately stops before source syntax, HIR, MIR, a standard-library API, persistence,
a chain target, consensus, gas pricing, or Merkle storage.

`MemoryStore` is an executable semantic oracle. A production backend may use a database, state tree,
host calls, or copy-on-write pages, but it must preserve the same observable transaction rules.

## Typed declarations

A layout is created in source order. Generic Go handles provide an implementation-side equivalent of
future compiler-known storage cells and maps:

```go
builder, err := contractstate.NewLayoutBuilder("example.token")
if err != nil {
    return err
}

totalSupply, err := contractstate.DeclareCell(builder, "total_supply", contractstate.U64)
if err != nil {
    return err
}

balances, err := contractstate.DeclareMap(
    builder,
    "balances",
    contractstate.Byte20,
    contractstate.U64,
)
if err != nil {
    return err
}

layout, err := builder.Build()
if err != nil {
    return err
}
```

A codec owns one stable `TypeID`, deterministic encoder, and strict decoder. The package supplies
canonical bool, big-endian `u32`, big-endian `u64`, bytes, UTF-8 text, 20-byte, and 32-byte codecs.
`FuncCodec[T]` permits explicit implementation-only codecs; changing its ID or byte representation is
a storage-layout compatibility break.

## Canonical layout identity

Each field records:

- zero-based declaration index;
- ASCII identifier;
- `cell` or `map` kind;
- map key codec identity when applicable;
- value codec identity;
- 32-byte field prefix.

Field prefixes use SHA-256 over the domain `fairyc-contract-state-field/v1`, contract UID, index,
kind, name, and codec identities. A map entry key uses the separate domain
`fairyc-contract-state-map-entry/v1`, the field prefix, and a length-delimited canonical encoded key.
The complete layout hash uses `fairyc-contract-state-layout/v1` and includes every ordered field plus
its prefix.

The domains and length delimiters prevent accidental cross-kind or variable-length concatenation
collisions. SHA-256 collision resistance is an explicit dependency of this prototype; a later target
may replace the digest only through a new compatibility identity.

## Transaction execution

Every transaction captures the complete committed `MemoryStore` state and version under one read
lock. Reads consult the private write overlay first and then the immutable snapshot. A write never
mutates committed state directly.

```go
receipt, err := contractstate.Execute(store, layout, limits, func(tx *contractstate.Transaction) error {
    supply, found, err := totalSupply.Get(tx)
    if err != nil {
        return err
    }
    if !found {
        supply = 0
    }
    if err := totalSupply.Set(tx, supply+10); err != nil {
        return err
    }
    return balances.Set(tx, recipient, 10)
})
```

`Execute` has three outcomes:

| Body outcome | Overlay outcome | Caller outcome |
| --- | --- | --- |
| returns `nil` and commit succeeds | every write publishes under one store lock | receipt returned |
| returns an error | complete overlay discarded | same error returned |
| panics | complete overlay discarded | panic re-raised |

Commit checks both the exact layout hash and the snapshot version before applying any key. A changed
layout returns `ErrLayoutMismatch`. Any intervening successful write or upgrade returns
`ErrConflict`. Neither failure applies a prefix of the overlay.

The prototype uses store-wide optimistic serialization. Disjoint concurrent writers can conflict.
That is intentional: it proves atomic semantics before a later backend introduces key-level conflict
tracking or MVCC.

## Read-only views

`View` captures the same immutable snapshot but rejects every write with `ErrReadOnly`. It always
closes the transaction without incrementing the store version.

## Limits

Callers must provide all limits explicitly:

```go
limits := contractstate.Limits{
    MaxWrites:      1024,
    MaxWriteBytes:  1 << 20,
    MaxValueBytes:  64 << 10,
    MaxMapKeyBytes: 1024,
}
```

`MaxWrites` counts distinct overlay keys. Replacing an existing pending write does not consume a
second key. `MaxWriteBytes` counts each fixed 32-byte key, one operation byte, and encoded value
bytes. Deletion therefore remains distinguishable from storing an empty value. A rejected mutation
leaves every earlier pending mutation intact and does not alter its accounting.

The kernel does not invent universal defaults because target runtimes will price and cap storage
differently.

## Layout upgrades

`MemoryStore.Upgrade` accepts the identical layout as a no-op. A changed layout must preserve the
contract UID and every existing field byte-for-byte at the same index. New fields may only be
appended. The upgrade increments the store version, invalidating every open transaction.

The first rule intentionally rejects field removal, reorder, rename, codec changes, namespace reuse,
and in-place migration. A later migration system needs explicit old/new codecs, bounded work,
restart behavior, authorization, and rollback evidence instead of silently weakening this gate.

## Copy and alias rules

The store, snapshots, overlays, codecs supplied by the package, `Dump`, and transaction reads copy
byte slices at their boundaries. A caller cannot mutate committed state through an aliased Go slice.
Future Fairy source integration must replace these implementation copies with verified owner-bound
borrows where profitable without changing transaction visibility.

## Required future compiler mapping

A source-level state implementation must preserve these ownership boundaries:

1. one compiled contract layout maps to one exact layout hash;
2. each entry execution owns exactly one transaction capability;
3. state handles cannot escape that capability or cross contracts implicitly;
4. `storage_read` and `storage_write` effects remain visible at function boundaries;
5. verified MIR owns commit, rollback, and trap edges rather than backend-invented cleanup;
6. external calls cannot occur while an uncommitted exclusive state borrow remains live;
7. cache and package identities include the state layout hash.

The current package proves the state engine. It does not claim those frontend and backend gates are
implemented.
