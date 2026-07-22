# ADR 0154: Bind package artifact bytes to one compiler process

- Status: Accepted
- Date: 2026-07-22

## Context

`packagecache.MemoryStore` enforces byte and entry limits over opaque byte payloads. The first
package MIR artifact is deliberately a process-local Go value because its shared type context still
contains build-local identities. Storing it as an unbounded side table would bypass the existing
memory accounting. Calling ordinary JSON or gob output a persistent format would freeze accidental
field names and compatibility semantics.

## Decision

Add `packageartifact.ProcessCodec`. Each codec instance receives a random 256-bit non-secret
identity and a mandatory caller byte limit. Encoded envelopes include the exact process identity,
one closed schema and version, and one independently verified package artifact.

Decoding accepts only bytes from the same codec identity, rejects unknown fields, trailing values,
oversized input, and noncanonical JSON, then reruns package artifact verification. A different codec
instance fails with a stable foreign-process error even inside the same executable.

The codec identity is also available as a compatibility member for an in-memory package cache key.
It is isolation, not authentication or a secret.

## Consequences

- process-local package artifacts can use the existing byte-bounded LRU without unaccounted payloads;
- copied bytes cannot be treated as compatible by a later compiler process;
- corruption still fails before reassembly, execution, or backend emission;
- the encoded byte size is valid process-cache accounting evidence but not a disk-format size claim;
- ADR 0161 supplies a separate stable type-context codec, but persistent storage still requires
  stable body encoding, checked local-ID rebinding, and a compatibility and disk-policy decision.

## Rejected alternatives

- An unbounded `map[Key]Artifact` side table was rejected because LRU byte limits would become fake.
- A raw pointer or integer handle stored in `MemoryStore` was rejected because eviction would not own
  the referenced memory.
- A cross-process JSON or gob promise was rejected because build-local package and type facts remain.
