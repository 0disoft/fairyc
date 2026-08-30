# Contract-state transaction MIR v0

- Status: Experimental compiler-private contract
- Decision source: `docs/adr/0344-compiler-private-contract-state-transaction-mir.md`
- State oracle: `internal/contractstate`
- Implementation: `internal/contractstatemir`
- Stable language, MIR artifact, runtime ABI, and SDK identity: unchanged

## Boundary

This submodel proves that one exact persistent-state layout and one explicit transaction exit can be
verified before execution against the process-local semantic oracle. It is intentionally separate
from the stable `internal/mir.Program`: no Fairy source form lowers into it, and no package artifact,
cache, C backend, target runtime, or public compatibility promise serializes it.

The separation is temporary but useful. It lets the compiler prove layout and transaction semantics
before choosing source grammar or allowing a backend to invent commit and rollback behavior.

## Layout

A layout carries the contract UID, canonical hash, and ordered fields. Every field retains its exact
index, name, cell-or-map kind, key and value codec identities, and namespace prefix. Building or
verifying the submodel reconstructs the canonical `contractstate.Layout`; any changed index, prefix,
codec, field order, or hash fails with `CSM1001`.

Only the seven compiler-known canonical codecs are admitted:

- `bool/v1`
- `u32/be/v1`
- `u64/be/v1`
- `bytes/v1`
- `utf8/v1`
- `bytes20/v1`
- `bytes32/v1`

Implementation-defined `FuncCodec` identities remain valid for direct oracle experiments but cannot
enter compiler-private MIR until their canonical byte contract has an owning compiler rule.

## Operations

The first submodel is straight-line and deliberately small:

- `cell-read`, `cell-write`, `cell-delete`;
- `map-read`, `map-write`, `map-delete`.

Every operation names a field by exact layout index. Map keys and written values carry an explicit
codec identity plus canonical bytes. The verifier rejects kind mismatches, missing or extra
operands, type mismatches, noncanonical bytes, and unknown operations with `CSM1002` before a state
snapshot is captured. Reads produce ordered test observations; they are not a public contract return
ABI.

## Exit and execution

Every program ends in exactly one explicit exit:

| Exit | Oracle behavior |
| --- | --- |
| `commit` | publish the complete overlay after all operations succeed |
| `reject` | return a rejected outcome and discard the complete overlay |
| `trap(category)` | discard the complete overlay and re-panic the exact compiler-private trap |

Malformed or implicit exits fail verification with `CSM1003`. Runtime codec, layout, conflict, and
limit errors return from `Execute` and rely on the managed `contractstate.Execute` boundary to roll
back. The verifier runs before `Begin`, so invalid MIR cannot observe a snapshot or mutate state.

## Closed gates

This slice does not add source syntax, HIR state declarations, data-dependent values, branches,
loops, calls, `storage_read` or `storage_write` effect checking, owner-bound state borrows, external
call rules, stable MIR serialization, package/interface/cache identity, deterministic native
backend execution, persistence, migration, gas, or chain integration.

Source-independent HIR declarations now map into this boundary under ADR 0345, and public-interface
v5 plus whole-package cache key v3 carry the exact layout identity. The next compiler slice must add
effect-checked state accesses and transaction capability flow without opening source syntax first.
