# ADR 0374: Canonical contract-state Merkle commitments

## Status

Accepted as a compiler-internal contract-state oracle slice. This decision does not admit a chain target, persistence provider, consensus protocol, network proof service, or public Fairy API.

## Context

The contract-state oracle already defines canonical physical keys, encoded values, append-only layout identity, store versions, and atomic commit. It had no versioned state commitment or bounded inclusion/non-inclusion proof format.

A commitment must not hash only logical values. Proof identity has to include the exact layout, physical key, encoded bytes, deletion/absence state, and committed store version or values could be replayed across incompatible schemas or snapshots.

## Decision

The internal algorithm ID is `fairyc.contract-state.merkle/v1` and uses SHA-256 with separate domains for empty trees, padding leaves, value leaves, internal nodes, and the final root.

Every value leaf hashes the layout hash, store version, 32-byte physical key, and length-prefixed encoded value. Leaves are sorted by binary physical key. The leaf layer is padded to the next power of two with position-bound padding hashes. Every internal node again binds layout hash and store version. The final root additionally binds leaf count.

Tree construction snapshots layout hash, store version, keys, and values under one `MemoryStore` read lock. Commitment calculation therefore cannot combine a pre-commit layout with post-commit values or vice versa.

Inclusion proofs contain the exact leaf index, key, encoded value, and one ordered sibling hash per tree level. Verification recomputes the complete root and rejects algorithm drift, header drift, impossible indices, unexpected depth, oversized values, or a root mismatch.

Non-inclusion proofs contain the absent query and the authenticated predecessor and/or successor at its sorted insertion point. Empty state requires no neighbors. Boundary proofs require index zero or the final leaf; interior proofs require consecutive indices and strictly ordered keys. This proves absence without giving proof verification any commit authority.

Deletion is represented by absence in the next committed version. There is no tombstone leaf in v1.

## Bounds

Every build and verification call receives mandatory `CommitmentLimits` covering maximum leaf count, maximum tree depth, and maximum encoded value bytes. There are no guessed defaults. Verification checks these limits before accepting hostile proof material.

The v1 implementation caps configured depth at 31, keeping padded leaf counts representable by `uint32`.

## Security properties

The layout hash prevents cross-layout replay. Physical keys prevent cross-field or cross-map ambiguity. Store version prevents replay across committed snapshots. Length-prefixing prevents byte concatenation ambiguity. Domain separation prevents interpreting a leaf as a node or root. Exact algorithm IDs fail closed on version drift.

Proof generation and verification are read-only helpers. They do not call `Begin`, `Execute`, `commit`, `Upgrade`, or persistence adapters and cannot publish state.

## Evidence

`internal/contractstate/commitment_test.go` contains a fixed root golden vector, generated store inclusion and absence proofs, deletion-to-absence behavior, layout/version binding, configured-limit rejection, and mutations of values, algorithms, sibling paths, and absence boundaries.

## Deferred work

Production persistence adapters may cache or materialize commitment nodes later, but must preserve the same observable v1 root or introduce a separately versioned commitment algorithm. Network serialization, proof compression, alternative hashes, chain-specific trie formats, consensus integration, signatures, and public SDK exposure remain closed.
