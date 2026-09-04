# ADR 0382: Admit a private single-cell contract-state native handoff

- Status: Accepted
- Date: 2026-09-04

## Context

Verified contract-state transactions could reach the semantic oracle, stable package artifacts, and
the compiler result, but no backend boundary consumed the verifier-owned transaction together with
its exact layout and package-body identities. A broad native runtime would prematurely commit Fairy C
to maps, graphs, helpers, persistence, and a public ABI before one narrow failure-atomic path had
native evidence.

## Decision

1. Add a compiler-private `Prepare` boundary that lowers source-independent HIR, verifies the
   resulting transaction MIR, computes the exact package-body hash for the supplied stable package
   coordinate, and returns an opaque handoff. Callers cannot construct or replace its MIR/hash pair.
2. Admit only one straight-line transaction over field zero when that field is a `cell<u64>`.
   Cell read, write, and delete lower to target-neutral C runtime entry points. Graph blocks, maps,
   helpers, borrows, multiple fields, and every other codec fail before C emission.
3. Admit emission only for `x86_64-windows-msvc`. The generated unit embeds and checks the exact
   layout and package-body hashes before opening a transaction. Other targets fail closed.
4. Keep the runtime stack-only and bounded to one 64-byte encoded value. `begin` validates canonical
   store length/presence and copies a snapshot into an overlay. Reads observe that overlay; commit
   publishes it once; reject, trap, and operation failure clear it without modifying the store.
5. Map commit, reject, and trap to private entry statuses `0`, `1`, and `2`. Trap category bytes stay
   exact and compiler-owned. Hash mismatch, malformed store state, and runtime operation failure use
   separate negative private statuses.
6. Execute commit, reject, and trap programs under low and optimized x64 MSVC. The native probes
   require read-your-write value `42`, commit-only publication, reject/trap rollback, exact trap
   category, hash/canonical-store rejection, no stdout or stderr, and zero live runtime resources.

## Consequences

The first verified transaction now crosses a real compiler/backend/runtime boundary with exact
identity and native failure-atomic evidence. The runtime symbols, C structs, entry statuses, and
emitted unit are compiler-private and may change without compatibility guarantees. This does not
change the Fairy language, SDK version, public runtime ABI, stable CLI, or deployable-target claims.

Map and multi-field storage, graph/SSA execution, helpers, borrows, source state operations, public
examples, durable providers, concurrency, retry policy, gas, reentrancy, Linux admission, and a
stable runtime ABI remain separate slices.

## Evidence

- `internal/contractstatenative/`
- `internal/cstdlib/stage0_contract_state.c`
- `internal/cstdlib/contract_state_test.go`
