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
MIR and has no source, stable artifact, backend, or target admission.

Source-independent `hir.StateLayout` declarations lower into that submodel. Public-interface v5
carries the exact layout hash, while whole-package cache key v3 consumes the current package
interface as well as dependency interfaces. Layout changes therefore invalidate the owner and its
dependents before source syntax is admitted.

ADR 0346 adds one opaque transaction capability per compiler-private program and requires every
state operation to use it. The same verifier derives `storage_read` and `storage_write` requirements
and rejects missing, noncanonical, or host-only effect declarations before transaction begin.

ADR 0347 adds source-independent HIR transaction plans owned by an existing package function. The
function remains the sole effect authority; lowering resolves field names through the exact layout,
clones encoded bytes, assigns one fresh capability, verifies the program, and proves mixed
read/write behavior through the failure-atomic oracle. These bodies are process-local and are not
yet package-artifact or cache inputs.

These packages are compiler implementation evidence. They are not a public Go API, a persistent
database, a Merkle tree, a consensus implementation, or a Fairy C standard-library promise.

## Admission gates for source and MIR integration

A later PR may introduce source syntax only after it also supplies:

- exact grammar and recovery behavior;
- resolver and type rules for source state declarations and access;
- source-to-HIR effect inference and state access syntax;
- exact package-body/artifact identity and cache invalidation for generated state transaction plans;
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
