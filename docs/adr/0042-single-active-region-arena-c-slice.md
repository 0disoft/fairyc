# Single-Active-Region Arena C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation source: `internal/cstdlib/`

## Context

Lexical Region safety depends on more than bulk freeing. A stale Region handle must not allocate
after reset, a second Region must not attach to the same exclusively borrowed Arena, allocation
failure must not publish a partial node, and Arena drop must not race an active Region.

## Decision

1. The provisional Arena owns a private LIFO chain of SystemHeap allocations, a generation, and a
   single-active-Region flag. Node representation and allocation grouping are not public contracts.
2. Region begin captures the Arena head as a mark and the current generation. It fails if another
   Region is active.
3. `alloc_zeroed_bytes` and `copy_bytes` allocate storage and a private tracking node before
   publishing either to the Arena or caller. Failure releases temporary storage and leaves the
   Arena chain and output unchanged.
4. Zero-size allocation validates alignment, allocates no storage or node, and returns normalized
   empty storage. Nonzero newly visible bytes are initialized before return.
5. Region reset releases every node added after its mark, clears the Arena's active flag, advances
   the generation, and invalidates the Region record. Later use of that record fails.
6. Arena drop fails closed while a Region is active. With no active Region it releases all remaining
   nodes and advances the generation.
7. The helper supports only inert byte storage. Resource placement, fallible reset, public marks,
   nested Regions on one Arena, and representation access remain excluded.
8. A closed native probe covers initialization, exclusive begin, second-begin rejection, zeroed
   allocation, byte copy, unsupported-alignment rollback, normalized empty allocation, active drop
   rejection, reset, stale-use rejection, and final drop across Clang/GCC low/optimized profiles.
9. `fairyc.native-c-report/v8` adds `arenaExecutions`, expected to be four only after a successful
   hosted run. No hosted receipt has been observed.

## Consequences

- The one-allocation-per-object prototype is intentionally simple and not a performance claim.
  Chunking requires measurement while preserving mark/reset and failure atomicity.
- Source Flow remains the primary exclusive-borrow proof; the C transport independently rejects
  obvious concurrent or stale Region misuse.
- The existing interpreter Region trace remains an oracle layer, not proof that this C helper is
  source-connected.
