# Contract-state transaction MIR v0

- Status: Experimental compiler-private contract
- Decision source: `docs/adr/0344-compiler-private-contract-state-transaction-mir.md`
- Capability and effect source: `docs/adr/0346-contract-state-mir-capability-and-effect-verification.md`
- HIR lowering source: `docs/adr/0347-source-independent-hir-state-access-lowering.md`
- Body identity source: `docs/adr/0348-exact-contract-state-package-body-identity.md`
- Body payload source: `docs/adr/0350-bounded-stable-contract-state-body-payload.md`
- Fresh-owner rebind source: `docs/adr/0351-checked-fresh-owner-contract-state-body-rebind.md`
- Stable package-compilation source: `docs/adr/0352-stable-package-compilation-artifact.md`,
  `docs/package/stable-package-compilation-v1.md`
- Persistent compilation-cache source: `docs/adr/0353-stable-compilation-cache-records.md`
- Combined cached reassembly source: `docs/adr/0354-combined-cached-state-reassembly.md`
- Unified compiler-result source: `docs/adr/0355-unified-verified-compiler-result.md`
- Closed effect-profile source: `docs/adr/0356-closed-contract-state-effect-profile.md`
- Call-free owner source: `docs/adr/0357-call-free-contract-state-owners.md`
- Straight-line owner source: `docs/adr/0358-straight-line-contract-state-owners.md`
- State oracle: `internal/contractstate`
- Implementation: `internal/contractstatemir`, `internal/compiler/check.go`
- Stable language, MIR artifact, runtime ABI, and SDK identity: unchanged

## Boundary

This submodel proves that one exact persistent-state layout and one explicit transaction exit can be
verified before execution against the process-local semantic oracle. It is intentionally separate
from the stable `internal/mir.Program`: no Fairy source form lowers into it, and no C backend,
target runtime, or public compatibility promise consumes it. ADR 0350 adds a standalone compiler-
private stable body payload, ADR 0352 pairs it with stable package MIR, ADR 0353 admits that outer
artifact into the disposable persistent cache, and ADR 0354 reassembles both members under one fresh
function-symbol graph without changing stable MIR v3. ADR 0355 publishes ordinary and contract-state
MIR atomically through one verified compiler result.

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
noncanonical sets, and host-only effects fail with `CSM1005`. ADR 0356 also rejects
`block_context`, `emit`, and `external_call`: this MIR has no operation that can justify those
authorities. ADR 0357 also rejects every call expression in the current transaction owner body,
including nested loop, block, child, and match-arm positions. This is call exclusion, not transitive
call-graph or reentrancy support. ADR 0358 rejects `if`, `match`, `while`, classic `for`, loop
control, assertions, Result propagation, and short-circuit operators in the same current-body paths.

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

Package-cache key v4 consumes the hash for exact owning-package invalidation. ADR 0350 now transports
the same stable facts through an exact-version, caller-bounded canonical payload. Decode rebuilds
the layout, parses and re-encodes each stable function descriptor, creates fresh private capability
identity, re-enters this verifier, and recomputes the hash. Build-local identities, spans, and
capability tokens are never serialized.

ADR 0352 stores this payload beside stable package MIR v3 in one exact-version outer compilation
artifact and requires both nested artifacts to share a package coordinate. Stable MIR v3 itself is
unchanged. ADR 0353 makes persistent records carry the bundle and checks the cache-selected body hash
before a hit. ADR 0354 makes the all-hit coordinator rebind every cached body through the fresh
coordinate-owned logical-function bindings returned by MIR reassembly. The resulting compiler-private
transactions are executable through the semantic oracle; `Session.Check` does not execute them.
ADR 0355 gives cold, process-local all-hit, and persistent all-hit paths the same
`compiler.Result.StateTransactions` ownership boundary.

## Fresh-owner rebind

ADR 0351 requires the caller's cache-selected expected body hash, a one-to-one current coordinate
map, an exact current HIR layout, and byte-identical current logical function descriptors. It maps
only the payload's transaction owners to fresh package/function symbols; unrelated current functions
do not enter the boundary. The current HIR transaction-operation side table is not required.

Every rebound transaction receives another new opaque capability, attaches it to every operation,
and re-enters verification. Repeated rebinds therefore reconstruct equal work with distinct process
authority. Missing or shared symbols, coordinate ambiguity, layout/signature drift, and valid-but-
unexpected bodies fail before publication.

## Closed gates

This slice does not add source syntax, data-dependent state values, conditional transaction MIR,
call admission, owner-bound state borrows, transitive call-graph rules, deterministic native backend execution, persistence,
migration, gas, or chain integration.

Source-independent HIR declarations now map into this boundary under ADR 0345, and public-interface
v5 carries the exact layout identity. ADR 0346 verifies transaction
capability identity and storage effects, ADR 0347 lowers source-independent HIR accesses into those
proofs, ADR 0348 gives the verified bodies an exact stable package hash, ADR 0349 makes cache key v4
consume that hash, and ADR 0350 adds bounded capability-free payload transport with verifier
re-entry. ADR 0351 adds checked fresh-owner rebind, and ADR 0352 binds the payload to stable package
MIR through a separate outer artifact. ADR 0353 adds cache publication, exact expected-body lookup,
and record quarantine. ADR 0354 rebinds and aligns both cached members before the workspace compiler
returns executable compiler-private state transactions. ADR 0355 makes those transactions part of
the ordinary atomic verified compiler result. ADR 0356 rejects every effect not represented by the
current storage-operation set. ADR 0357 rejects every current owner-body call during direct HIR
lowering and stable fresh-owner rebind. ADR 0358 rejects dynamic control flow in those same paths.
