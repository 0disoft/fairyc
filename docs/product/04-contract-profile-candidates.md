# Contract Profile Candidates

- Status: Candidate only
- Owner: Fairy C language and runtime
- Admission rule: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- First executable foundation: `docs/adr/0338-typed-persistent-state-transaction-kernel.md`
- Generic-planning foundation: `docs/adr/0339-bounded-static-generic-specialization-kernel.md`
- Contract-interface foundation: `docs/adr/0340-experimental-chain-neutral-contract-idl.md`

## Purpose

A future Fairy C contract profile may target deterministic state-machine runtimes such as WebAssembly
or RISC-V contract engines. The profile must reuse the ordinary Fairy ownership, checked arithmetic,
`Result`, effect, and verified-MIR rules instead of creating a separate Solidity-shaped language.
This document records candidate requirements. ADR 0377 now owns the declaration-only source syntax;
this profile still does not publish a target.

## Typed persistent state candidate

The first required capability is typed persistent state with transaction-atomic mutation. The
accepted language-13 declaration form is:

```fy
@contract_uid("example.token")
state {
    cell u64 total_supply;
    map bytes20 u64 balances;
}
```

ADR 0377 selects one explicit package declaration over an implicit persistent aggregate or an
external generated-owner schema. It lowers to the existing HIR and layout builder rather than
creating another state model. State access syntax and executable persistence remain closed.

The accepted behavior must satisfy all of the following:

1. Every field has a stable ordered identity, kind, key type, value type, namespace prefix, and
   complete layout hash.
2. State reads are typed and snapshot-consistent for one execution.
3. Writes enter a private overlay and become visible to later reads in that execution.
4. A successful execution publishes the complete overlay atomically.
5. A returned error, explicit rejection, trap, verifier failure, or backend failure publishes none
   of the overlay.
6. Storage layout upgrades are fail-closed. The first admitted rule is identical layout or append-only
   fields; removal, reorder, rename, kind change, and codec change are rejected.
7. Key, value, write-count, and aggregate write-byte limits are explicit target policy.
8. Serialization is canonical and target-independent. Chain-specific ABI formats remain backend or
   package policy rather than core source syntax.

## Current executable evidence

`internal/contractstate` implements a process-local semantic oracle for the requirements above. It
contains typed `Cell[T]` and `Map[K, V]` handles, canonical SHA-256 layout and key derivation,
append-only compatibility checks, snapshot transactions, bounded read-your-writes overlays,
store-wide optimistic conflict detection, atomic commit, and rollback on returned errors or panics.

`internal/contractstatemir` adds a separate compiler-private layout and transaction submodel. It
verifies exact layout identity, canonical cell/map operations, explicit commit/reject/trap exits,
ADR 0378's bounded acyclic SSA blocks with typed reads, joins, bool branches, and unit costs, and
ADR 0379's bounded same-package no-argument void helper DAG with transitive effects and call costs
before executing them through the oracle. It is not the stable whole-program MIR and has no source
state-access, backend, or target admission.

Language-13 source declarations and direct compiler fixtures lower into the same `hir.StateLayout`
submodel. Public-interface v5
carries the exact layout hash, while whole-package cache key v4 consumes the current package
interface and body hash as well as dependency interfaces. Layout changes therefore invalidate the
owner and its dependents, while private body changes invalidate only the owner.

ADR 0346 adds one opaque transaction capability per compiler-private program and requires every
state operation to use it. The same verifier derives `storage_read` and `storage_write` requirements
and rejects missing, noncanonical, or host-only effect declarations before transaction begin.

ADR 0347 adds source-independent HIR transaction plans owned by an existing package function. The
function remains the sole effect authority; lowering resolves field names through the exact layout,
clones encoded bytes, assigns one fresh capability, verifies the program, and proves mixed
read/write behavior through the failure-atomic oracle. These bodies are process-local and are not
yet package-artifact payloads or cache inputs.

ADR 0348 assigns the verified body set an exact stable hash over package coordinate, layout,
function signature and effects, operation bytes/order, and exit. Local package/function numbers,
spans, and capability tokens are excluded. The hash exists before cache or payload admission so a
later consumer cannot invent a weaker identity.

ADR 0349 makes whole-package cache key v4 consume that hash before lookup and keeps dependency
entries public-interface-only.

ADR 0350 adds an exact-version, caller-bounded canonical payload for the layout and transaction
bodies. Decode parses and canonically re-encodes stable function descriptors, reconstructs layout
and operation values, creates fresh private capabilities, re-enters the verifier, and recomputes the
ADR 0348 hash.

ADR 0351 requires the cache-selected expected hash, one-to-one coordinate map, exact current HIR
layout, and byte-identical logical function descriptors before mapping those bodies to fresh
package/function symbols and newly verified capabilities. It does not require current HIR source
operation bodies. The payload is not yet stored or reassembled through package caches.

ADR 0352 pairs that payload with stable package MIR v3 in one exact-version outer artifact and
requires both nested artifacts to share a package coordinate. It does not change either nested
format. ADR 0353 stores that bundle in persistent cache `/2`, requires exact expected-body identity,
and quarantines malformed, legacy, or contradictory generations. ADR 0354 rebinds every all-hit
state body through the matching fresh MIR function symbol and returns executable compiler-private
transactions without running them inside compilation. ADR 0355 makes cold and cached compilation
publish those transactions atomically through the same verified `compiler.Result`. ADR 0356
rejects `block_context`, `emit`, and `external_call` authority until transaction MIR represents and
verifies those operations. ADR 0357 originally rejected every internal or foreign call in the complete current
owner body during direct lowering and stable rebind. ADR 0379 now proves transitive operations,
effects, and shared capability for one bounded static helper DAG; state-borrow lifetime and
reentrancy remain closed. ADR 0358 also rejects every
ordinary-owner `if`, `match`, loop, loop-control, assertion, Result-propagation, and short-circuit
path. ADR 0378 admits conditional state-side blocks without relaxing that ordinary HIR boundary.
ADR 0359 additionally rejects unsafe
declarations/scopes and raw-pointer types through signatures, locals, expressions, and nested
aggregates without conflating ordinary safe borrows with state permissions. ADR 0360 rejects
receiver-bound methods and automatic drop glue as current owners while retaining receiver-free
static associated functions. ADR 0361 rejects ordinary checked arithmetic, division, shifts,
indexing, and slicing until a future model can prove their trap relationship to the transaction
exit. ADR 0362 rejects owned resource values and lexical regions until their drop, burn, placement,
and reset lifetimes have a transaction-level representation; safe resource borrows and static
resource namespaces remain admitted. ADR 0363 requires every residual current owner statement,
expression, operator, and expression-local effect to appear in a positive allowlist.
ADR 0364 adds a versioned five-case transaction corpus for commit, reject, trap, delete, and
write-limit rollback through the current HIR/MIR/oracle path. Passing that corpus does not by itself
admit source syntax, native execution, production persistence, or deployable contract targets.

ADR 0376 additionally fixes the minimum vendor-neutral persistence port and a deterministic
process-local reference adapter. It proves the compare-and-publish boundary and failure mapping but
does not admit a database/provider, durable artifact, native lowering, or production target.
ADR 0365 keeps those five cell cases historical and adds three current map cases, including an exact
physical storage-key vector and map-key-limit rollback, without changing the admission boundary.
ADR 0366 strengthens those cases to consume only compiler-published state transactions and proves
state verification failure withholds ordinary MIR too; source construction and targets remain closed.
ADR 0367 versions optimistic snapshot/conflict behavior independently: stale overlays publish no
state, existing readers retain their start snapshot, and later transactions see the winner. It does
not choose production isolation, scheduling, retry, or consensus policy.
ADR 0368 freezes the current oracle's store-wide version check for distinct map keys and adds exact
map snapshot/storage-key evidence. Key-granular concurrency would require a new semantic version.
Real-goroutine regressions now cover overlapping cell writes, same-key and distinct-key map writes,
and readers racing a complete commit. Their channel-coordinated overlap and Linux race-detector gate
add thread-safety evidence without selecting a production isolation or scheduling policy.
ADR 0371 adds a separate versioned layout-publication lane. It preserves committed bytes across an
append-only extension, invalidates an already-open old-layout transaction, and rejects every current
breaking layout dimension without mutation. Database migration and upgrade authorization remain
outside this oracle.
ADR 0372 fixes retry ownership: the kernel performs one callback evaluation and one commit attempt,
while any bounded retry loop, fresh attempt identity, exhaustion result, cancellation, or backoff is
caller-visible. The versioned retry lane proves fresh-snapshot recomputation after conflict and
complete competing overlays without admitting a public helper or hidden effect replay.

These packages are compiler implementation evidence. They are not a public Go API, a persistent
database, a production Merkle store, a consensus implementation, or a Fairy C standard-library
promise. ADR 0374's read-only bounded commitment oracle authenticates exact in-memory snapshots
without changing that boundary.

## Remaining admission gates for source access and MIR integration

A later PR may introduce source state access only after it also supplies:

- resolver and type rules for source state access;
- source-to-HIR effect inference and state access syntax;
- MIR operations whose verifier proves layout identity and transaction ownership;
- interpreter fixtures for success, error propagation, trap, delete, conflict, and limits;
- at least one deterministic contract backend with byte-for-byte state transition fixtures;
- stable package artifact and cache-reassembly transport for state operations;
- diagnostics for duplicate fields, unsupported codecs, stale layouts, illegal borrows, and
  non-atomic external interaction.

## Non-goals of the first slice

The first slice does not provide chain networking, consensus, accounts, balances, gas pricing,
public or network Merkle proofs, event logs, contract-to-contract calls, upgrade authorization, source-level globals,
persistent files, reflection, dynamic serialization, hidden allocation, or a production concurrent
store.
