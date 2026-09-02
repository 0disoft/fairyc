# Contract State Transaction Kernel v0

- Status: Experimental executable contract
- Product source: `docs/product/04-contract-profile-candidates.md`
- Decision source: `docs/adr/0338-typed-persistent-state-transaction-kernel.md`
- Implementation: `internal/contractstate`
- Compiler-private MIR bridge: `internal/contractstatemir`,
  `docs/mir/contract-state-transaction-mir-v0.md`
- Compatibility identity: `fairyc-contract-state/*/v1` domains

## Boundary

The kernel defines host-independent layout and transaction behavior for a future Fairy C contract
profile. It deliberately stops before source syntax, HIR, stable whole-program MIR, a standard-library
API, persistence, a chain target, consensus, gas pricing, or Merkle storage. ADR 0344 adds a separate
experimental compiler-private transaction submodel without opening those stable surfaces.

ADR 0350 gives that compiler-private submodel a bounded canonical body payload. Decode never imports
authority: it reconstructs layouts and operations from data, creates a fresh opaque transaction
capability, re-enters the MIR verifier, and requires the exact ADR 0348 body hash. ADR 0351
additionally binds the expected payload to an exact current HIR layout and fresh function symbols,
then reconstructs new capability authority without retaining source operation bodies. ADR 0352
pairs that payload with stable package MIR in a separate exact-version outer artifact. ADR 0353
admits that bundle into disposable persistent cache records with exact expected-body checking.
ADR 0354 reassembles cached MIR and state bodies under one fresh function-symbol graph and proves
explicit oracle execution. ADR 0355 publishes ordinary and contract-state MIR atomically through one
verified compiler result. ADR 0356 then admits only the represented `storage_read` and
`storage_write` effects. ADR 0357 additionally requires the current transaction owner body to be
call-free on direct lowering and stable rebind. ADR 0358 rejects dynamic control flow in those same
paths so current owners remain straight-line. ADR 0359 rejects unsafe scopes and reachable
raw-pointer types while preserving ordinary safe-borrow and pointer-free resource types. ADR 0360
also rejects receiver-bound and drop-glue owners while preserving static associated entries. ADR
0361 rejects ordinary potentially trapping operations while preserving the transaction program's
explicit static trap exit. ADR 0362 rejects owned resource values and lexical regions while
preserving safe resource borrows and static resource namespaces; production storage and runtime
targets still do not consume it. ADR 0363 freezes the remaining owner HIR as a positive allowlist so
future forms fail closed. ADR 0364 adds a versioned five-case transaction corpus that reaches this
oracle through source-independent HIR and verified transaction MIR.

`MemoryStore` is an executable semantic oracle. A production backend may use a database, state tree,
host calls, or copy-on-write pages, but it must preserve the same observable transaction rules.
`spec/fixtures/contract-state/manifest-v0.json` owns the historical executable cell examples;
ADR 0365 preserves that baseline and makes `manifest-v1.json` the current
cell-and-map corpus. Neither manifest is a production storage format.
ADR 0366 executes both corpora only after the compiler atomically publishes verified ordinary and
contract-state MIR; a state-MIR rejection leaves both executable outputs empty.
ADR 0367 adds a separate deterministic overlapping-transaction corpus: stale commits publish no
overlay or version, open readers retain their start snapshot, and fresh transactions observe the
latest version. Runtime store conflict does not revoke compiler IR.
ADR 0368 extends that durable evidence to maps: distinct entry keys still conflict through the
store-wide base version, while snapshot and fresh-observation rules remain unchanged.

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

`Execute` and `View` own the lifecycle of the transaction passed to their callback. Calling public
`Commit` from such a callback returns `ErrManagedLifecycle`, and public `Rollback` returns false;
callback code therefore cannot publish or close state before the wrapper observes its result.
Direct `Begin` and `BeginReadOnly` transactions retain caller-owned commit or rollback.

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

`internal/contractstate/concurrency_conflict_test.go` also drives overlapping cell and map
transactions through real goroutines. Channels establish the overlap without sleeps, the commit
winner remains scheduler-selected, and Linux CI runs the owning package under the Go race detector.
The tests require one complete winning overlay, one stale conflict, and only complete reader
snapshots; they do not change the store-wide conflict rule.

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

The current oracle plus compiler-private submodel now prove exact layout reconstruction, one managed
transaction capability, explicit commit/reject/trap exits, and straight-line canonical cell/map
operations. Every operation names one opaque program capability, and verifier-owned
`storage_read`/`storage_write` requirements fail before transaction begin. Source-independent HIR
owns one layout per package and attaches transaction plans to existing function effects; lowering
resolves field names, copies value bytes, verifies MIR, and executes through this oracle. The
transaction verifier rejects `block_context`, `emit`, and `external_call` until their operations and
complete call rules exist.
ADR 0348 additionally hashes the verified package body through stable function signatures, exact
operation bytes and order, layout identity, and exit while excluding local symbols and capability
tokens. Public-interface v5 carries only the exact layout hash, while whole-package cache key v4
adds the exact private body hash to the owner without widening dependency entries. The compiler
now also proves a bounded capability-free body payload, fresh capability reconstruction, verifier
re-entry, exact body-hash matching, checked fresh-HIR-owner rebind, and one coordinate-bound outer
stable package-compilation artifact. ADR 0354 also proves complete cache hits rebind state operations
to the matching fresh MIR function symbols and execute through this oracle. ADR 0355 moves cold and
cached transactions into the ordinary atomic verified compiler result. It still does not prove
source declaration/access rules, owner-bound borrows, external-call exclusion, native MIR lowering,
or deterministic target execution.
