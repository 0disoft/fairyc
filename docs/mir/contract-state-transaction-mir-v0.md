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
- Safe raw-pointer-free owner source: `docs/adr/0359-safe-contract-state-owner-memory-boundary.md`
- Explicit-entry owner source: `docs/adr/0360-explicit-entry-contract-state-owners.md`
- Potential-trap owner source: `docs/adr/0361-potentially-trapping-owner-operations.md`
- Resource-lifetime-free owner source: `docs/adr/0362-resource-lifetime-free-contract-state-owners.md`
- Closed-form owner source: `docs/adr/0363-closed-form-contract-state-owner-hir.md`
- Data-dependent graph source: `docs/adr/0378-data-dependent-contract-state-transaction-graphs.md`
- Verified helper-call source: `docs/adr/0379-verified-contract-state-helper-calls.md`
- State-borrow transport source: `docs/adr/0381-verified-contract-state-borrow-transport.md`
- Versioned execution fixtures: `spec/fixtures/contract-state/manifest-v4.json` with historical v0/v1/v2/v3 coverage, `docs/adr/0364-versioned-contract-state-transaction-fixtures.md`, `docs/adr/0365-versioned-contract-state-map-fixtures.md`, `docs/adr/0378-data-dependent-contract-state-transaction-graphs.md`, `docs/adr/0379-verified-contract-state-helper-calls.md`, `docs/adr/0381-verified-contract-state-borrow-transport.md`
- Compiler-published fixture source: `docs/adr/0366-compiler-published-contract-state-fixtures.md`
- State oracle: `internal/contractstate`
- Implementation: `internal/contractstatemir`, `internal/compiler/check.go`
- Stable language, MIR artifact, runtime ABI, and SDK identity: unchanged

## Boundary

This submodel proves that one exact persistent-state layout and every reachable explicit transaction exit can be
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

## Operations and values

The storage operation set remains deliberately small:

- `cell-read`, `cell-write`, `cell-delete`;
- `map-read`, `map-write`, `map-delete`.

ADR 0381 adds graph-root-only `cell-borrow`, `map-borrow`, `borrow-view`, and `borrow-copy`.
Borrow identities are transaction-unique but block-confined. They never enter ordinary SSA edges,
parameters, helpers, or linear bodies. Origins retain the exact field, literal map key, value codec,
snapshot, and mutation generation. Exact-address mutation invalidates a live handle; a dynamic
same-map mutation fails closed when non-aliasing cannot be proved. Only `borrow-copy` creates an
ordinary typed SSA value, using copied bytes rather than a pointer or mutable alias.

Every operation names a field by exact layout index. Map keys and written values carry an explicit
codec identity plus canonical bytes. The verifier rejects kind mismatches, missing or extra
operands, type mismatches, noncanonical bytes, and unknown operations with `CSM1002` before a state
snapshot is captured. In graph bodies, reads also define one codec-typed SSA value and one
`bool/v1` found value. Missing reads retain an absent observation and supply only the codec's
canonical zero to the typed SSA result; programs inspect `found` when absence matters. A write or
map key may consume an available same-typed SSA value. Reads produce ordered test observations;
they are not a public contract return ABI.

## Bounded control graph

ADR 0378 admits a graph body only when it is separate from the legacy linear body. It contains at
most 64 dense source-ordered blocks and 1024 operations, has one in-range predecessor-free entry,
and makes every block reachable. Backedges and all other cycles fail with `CSM1006`.

Values are local SSA identities. A use must follow its definition in the same block; cross-block
values travel only as explicit edge arguments into same-typed block parameters. This makes
dominance and joins local to the state verifier instead of borrowing assumptions from ordinary MIR.
`branch` consumes one available `bool/v1`; `jump`, `commit`, `reject`, and `trap` are the only other
terminators. Invalid definitions, uses, and joins fail with `CSM1007`. Every operation and
terminator carries the exact canonical cost `1`; absent or unknown costs fail with `CSM1008`.

## Capability and effects

Every program owns one opaque compiler-created transaction capability, and every operation must
carry that exact identity. A missing program capability or an operation carrying another identity
fails with `CSM1004`. The token is proof metadata only; it does not expose the oracle transaction or
grant public commit authority.

Program effects are canonical released on-chain HIR effects. Cell/map reads require
`storage_read`; writes and deletes require `storage_write`. Missing required effects,
noncanonical sets, and host-only effects fail with `CSM1005`. ADR 0356 also rejects
`block_context`, `emit`, and `external_call`: this MIR has no operation that can justify those
authorities. ADR 0357 originally rejected every call expression in the transaction owner body;
ADR 0379 now admits only the exact bounded static helper graph described below. General calls and
reentrancy remain closed. ADR 0358 still rejects ordinary-owner `if`, `match`, `while`, classic `for`, loop
control, assertions, Result propagation, and short-circuit operators in the same current-body paths.
ADR 0359 rejects unsafe declarations/scopes and raw-pointer types reachable through signatures,
locals, expressions, or nominal aggregate fields. Safe borrows remain ordinary HIR permissions, not
state-MIR permissions. ADR 0360 rejects receiver-bound methods and automatic drop glue as current
owners while preserving static associated functions with no receiver. ADR 0361 rejects retained
ordinary checked arithmetic and bounds operations because their conditional trap edges are absent
from this linear model. ADR 0362 rejects owned resource values and lexical regions because their
drop, burn, placement, and reset lifetimes are also absent.
ADR 0363 then admits only a positive statement, expression, and operator allowlist; every unknown
future owner form or expression-local effect fails closed. ADR 0378 does not relax that ordinary
HIR boundary: compiler-created `hir.StateBlock` values lower through the separate state side table.
ADR 0364 exercises the admitted HIR bridge and this verifier through a versioned five-case oracle
corpus covering commit, reject, trap, delete, and write-limit rollback.
ADR 0365 preserves those historical vectors and adds map write/read, map delete, exact physical-key,
and encoded-key-limit evidence in the current v1 corpus.
ADR 0366 submits every fixture HIR package to the unified compiler coordinator and executes only the
transaction from a verified atomic ordinary/state MIR result.
ADR 0378 adds current v2 graph fixtures for a cell-read branch with an explicit join and a map-read
found branch, plus cycle, join-type, missing-cost, and unavailable-value mutations.

ADR 0379 adds current v3 fixtures for a pure-to-state helper chain and helper rejection rollback,
plus recursion and transitive-effect mutations. Helper calls are exact same-package `void fn()`
edges, share the root transaction capability and overlay, cost one unit each, and are limited to an
acyclic reachable set of 32 helpers and 1024 total root/helper operations. `CSM1009` rejects an
unknown, unreachable, noncanonical, or recursive helper graph. Only the root may commit.

## HIR bridge

`hir.StateTransaction` names one package and existing ordinary HIR function symbol. The function is
the sole declared-effect authority. Each source-independent access names a layout field rather than
a physical index; lowering resolves the exact package layout, clones encoded values, creates one
fresh capability, attaches it to every operation, maps the static exit, and runs this verifier before
returning a program. A graph transaction additionally lowers dense blocks, explicit parameters and
edge arguments, typed read results, dynamic operands, and terminators before the same verifier and
compiler publication boundary. Results are deterministic in package/function order and remain
process-local.

## Exit and execution

Every executed path ends in exactly one explicit exit:

| Exit | Oracle behavior |
| --- | --- |
| `commit` | publish the complete overlay after all operations succeed |
| `reject` | return a rejected outcome and discard the complete overlay |
| `trap(category)` | discard the complete overlay and re-panic the exact compiler-private trap |

Malformed or implicit exits fail verification with `CSM1003`. Runtime codec, layout, conflict, and
limit errors return from `Execute` and rely on the managed `contractstate.Execute` boundary to roll
back. The verifier runs before `Begin`, so invalid MIR cannot observe a snapshot or mutate state.
Graph execution reads and writes through that same transaction, preserving snapshot reads,
read-your-write behavior, atomic overlay publication, and reject/trap/error rollback. `Outcome.Cost`
is the exact sum of executed operation and terminator costs.

## Package-body identity

ADR 0348 defines `fairyc.contract-state-package-body/v1`. It first verifies the complete HIR
transaction set, then hashes one package's stable coordinate, optional exact layout hash, stable
function descriptors, canonical effects, and either the exact legacy operation/exit sequence or the
exact graph blocks, values, edges, operands, costs, and terminators.
Build-local package/function numbers, spans, and fresh capability tokens do not participate.

Package-cache key v4 consumes the hash for exact owning-package invalidation. ADR 0350 now transports
the same stable facts through an exact-version, caller-bounded canonical payload. Decode rebuilds
the layout, parses and re-encodes each stable function descriptor, creates fresh private capability
identity, re-enters this verifier, and recomputes the hash. Build-local identities, spans, and
capability tokens are never serialized. Legacy bodies retain their byte-identical v1 envelope and
hash. Graph bodies use `fairyc.contract-state-package-body-artifact/v2`, reconstruct fresh
capabilities, re-enter graph verification, and participate in the already-required body hash, so
cache key v4 invalidates its owner after any graph change.

Borrow graph bodies use `fairyc.contract-state-package-body-artifact/v4`. Their canonical bytes and
body hash include borrow inputs/results and view bounds, then fresh-owner rebind recreates private
capability before verifier entry. Existing v1 linear, v2 graph, and v3 helper payload bytes and
pre-borrow body hashes remain unchanged.

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

Every rebound transaction receives another new opaque capability, attaches it to every operation
in every block,
and re-enters verification. Repeated rebinds therefore reconstruct equal work with distinct process
authority. Missing or shared symbols, coordinate ambiguity, layout/signature drift, and valid-but-
unexpected bodies fail before publication.

## Closed gates

This slice does not add source state-access syntax, ordinary-HIR conditional lowering, loops,
general call admission, graph-block helper calls, cross-block/helper borrow transfer, mutable or raw-pointer state views, reentrancy, deterministic native backend execution, persistence,
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
ADR 0359 rejects unsafe code and reachable raw-pointer types without banning safe borrows or
pointer-free resources. ADR 0360 rejects receiver and drop-glue entry semantics that the side model
does not represent, while preserving explicit static associated functions. ADR 0361 rejects
potentially trapping ordinary operations while leaving this MIR's explicit static trap exit intact.
ADR 0362 rejects ordinary owned-resource and lexical-region lifetime transitions while preserving
safe resource borrows and static resource namespaces.
ADR 0363 freezes the residual pure owner HIR as a recursive positive allowlist. ADR 0378 separately
admits bounded acyclic SSA state blocks with explicit joins, bool branches, represented costs, v2
graph payload transport, and compiler-published cell/map execution while preserving legacy v1 bytes.
ADR 0379 narrowly reopens only statically resolved same-package no-argument void calls whose
ordinary-HIR sequence matches the state side table. Indirect, foreign, cross-package,
argument-bearing, value-returning, recursive, receiver-bound, and resource-owning calls remain
closed. Helper-bearing stable bodies use v3 transport while v1 and v2 remain byte-identical.
