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
This document records candidate requirements. It does not reserve syntax, publish a target, or change
the Stage 0 language revision.

## Typed persistent state candidate

The first required capability is typed persistent state with transaction-atomic mutation. An
illustrative source form is:

```fy
state {
    StorageCell<u64> total_supply;
    StorageMap<Address, u64> balances;
}
```

The spelling above is not reserved. A later language ADR must decide whether state is a declaration,
a generated owner module, a compiler-known package, or another form after the executable kernel has
produced enough evidence.

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

`internal/contractstatemir` now adds a separate compiler-private layout and straight-line transaction
submodel. It verifies exact layout identity, canonical cell/map operations, and explicit commit,
reject, or trap exits before executing them through the oracle. It is not the stable whole-program
MIR and has no source, stable package-artifact, backend, or target admission.

Source-independent `hir.StateLayout` declarations lower into that submodel. Public-interface v5
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
verifies those operations. ADR 0357 rejects every internal or foreign call in the complete current
owner body during direct lowering and stable rebind; later call admission must prove transitive
operations, effects, state-borrow lifetime, capabilities, and reentrancy. ADR 0358 also rejects every
current `if`, `match`, loop, loop-control, assertion, Result-propagation, and short-circuit path until
conditional transaction MIR has an exact representation. ADR 0359 additionally rejects unsafe
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

These packages are compiler implementation evidence. They are not a public Go API, a persistent
database, a Merkle tree, a consensus implementation, or a Fairy C standard-library promise.

## Admission gates for source and MIR integration

A later PR may introduce source syntax only after it also supplies:

- exact grammar and recovery behavior;
- resolver and type rules for source state declarations and access;
- source-to-HIR effect inference and state access syntax;
- MIR operations whose verifier proves layout identity and transaction ownership;
- interpreter fixtures for success, error propagation, trap, delete, conflict, and limits;
- at least one deterministic contract backend with byte-for-byte state transition fixtures;
- stable package artifact and cache-reassembly transport for state operations;
- diagnostics for duplicate fields, unsupported codecs, stale layouts, illegal borrows, and
  non-atomic external interaction.

## Non-goals of the first slice

The first slice does not provide chain networking, consensus, accounts, balances, gas pricing,
Merkle proofs, event logs, contract-to-contract calls, upgrade authorization, source-level globals,
persistent files, reflection, dynamic serialization, hidden allocation, or a production concurrent
store.
