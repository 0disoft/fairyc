# Contract-state transaction MIR v0

- Status: Experimental compiler-private contract
- Decision source: `docs/adr/0344-compiler-private-contract-state-transaction-mir.md`
- Capability and effect source: `docs/adr/0346-contract-state-mir-capability-and-effect-verification.md`
- HIR lowering source: `docs/adr/0347-source-independent-hir-state-access-lowering.md`
- Body identity source: `docs/adr/0348-exact-contract-state-package-body-identity.md`
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

## Capability and effects

Every program owns one opaque compiler-created transaction capability, and every operation must
carry that exact identity. A missing program capability or an operation carrying another identity
fails with `CSM1004`. The token is proof metadata only; it does not expose the oracle transaction or
grant public commit authority.

Program effects are canonical released on-chain HIR effects. Cell/map reads require
`storage_read`; writes and deletes require `storage_write`. Missing required effects,
noncanonical sets, and host-only effects fail with `CSM1005`. Additional on-chain effects remain
legal because this submodel does not yet represent every operation in a complete function.

## HIR bridge

`hir.StateTransaction` names one package and existing ordinary HIR function symbol. The function is
the sole declared-effect authority. Each source-independent access names a layout field rather than
a physical index; lowering resolves the exact package layout, clones encoded values, creates one
fresh capability, attaches it to every operation, maps the static exit, and runs this verifier before
returning a program. Results are deterministic in package/function order and remain process-local.

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

## Package-body identity

ADR 0348 defines `fairyc.contract-state-package-body/v1`. It first verifies the complete HIR
transaction set, then hashes one package's stable coordinate, optional exact layout hash, stable
function descriptors, canonical effects, exact operation sequence and bytes, and static exit.
Build-local package/function numbers, spans, and fresh capability tokens do not participate.

The hash is identity only. Package-cache key v4 consumes it for exact owning-package invalidation,
but no stable artifact transports or rebinds these operation payloads yet.

## Closed gates

This slice does not add source syntax, data-dependent values, branches, loops, calls, owner-bound
state borrows, external-call rules, stable MIR serialization, package-artifact payload transport,
deterministic native
backend execution, persistence, migration, gas, or chain integration.

Source-independent HIR declarations now map into this boundary under ADR 0345, and public-interface
v5 carries the exact layout identity. ADR 0346 verifies transaction
capability identity and storage effects, ADR 0347 lowers source-independent HIR accesses into those
proofs, ADR 0348 gives the verified bodies an exact stable package hash, and ADR 0349 makes cache
key v4 consume that hash. A later compiler slice must add payload artifact transport before ordinary
source or cached compilation consumes the bodies.
