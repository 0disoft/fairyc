# ADR 0155: Connect verified package artifacts to the bounded process cache

- Status: Accepted
- Date: 2026-07-22

## Context

The process-bound package codec produces bytes suitable for `packagecache.MemoryStore`, but no
typed cache boundary owns encode, decode, public-interface checking, rejection accounting, or
quarantine. A caller assembling those steps ad hoc could count raw byte hits as successful reuse or
delete a valid record that concurrently replaced a rejected payload.

## Decision

Add `packageartifact.Cache` over one `ProcessCodec` and one byte/entry-bounded `MemoryStore`.
`Put` verifies and encodes the complete artifact before insertion. `Get` requires the expected
public-interface hash, exactly one MIR payload, same-process decoding, canonical bytes, and complete
artifact verification before returning `hit`.

Missing keys return `miss`. Contradictory or corrupt records return `rejected`, contain no artifact,
and are quarantined with `MemoryStore.DeleteIfMatch`. The compare-and-delete operation removes a
record only when its complete key, output interface, artifact kinds, and bytes still match the copy
that failed validation. A newer concurrent replacement survives.

Cache statistics separate verified hits, misses, and rejections from the underlying store's raw
hits, misses, retained bytes, and evictions.

## Consequences

- the bounded LRU now owns all bytes retained for process-local package MIR reuse;
- a raw lookup is never reported as compiler reuse until decode and verification finish;
- corruption and interface contradiction become observable safe misses;
- concurrent replacement cannot be erased by stale quarantine;
- no compile path consults the cache yet, so this ADR makes no build-speed claim.

## Rejected alternatives

- Counting `MemoryStore.Get` as a cache hit was rejected because corrupt bytes can pass that layer.
- Unconditional deletion after decode failure was rejected because it races with replacement.
- Keeping decoded artifacts in an unbounded parallel map was rejected because byte accounting and
  LRU eviction would no longer own the real memory payload.
