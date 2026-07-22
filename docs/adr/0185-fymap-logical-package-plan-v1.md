# `.fymap` logical package plan v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Atlas source: `docs/atlas/fymap-logical-plan-v1.md`

## Context

The immutable semantic snapshot, source-file identities, and provisional `SymbolId` supply the
facts needed by Atlas, but they do not yet prove that a future map contains one complete,
compatible package partition. Jumping directly to a byte codec would freeze section layout and
checksum choices before package ownership, compatibility metadata, and incomplete-relation behavior
have executable evidence.

The workspace compiler also computed public-interface hashes through a private loop. Atlas needs
the same complete package set without copying that policy or risking different ordering and failure
behavior.

## Decision

1. Add `moduleinterface.ComputePackages` as the single deterministic, fail-closed operation for a
   caller-supplied package ID set and the coordinate metadata required by those interfaces. The
   workspace cache coordinator and Atlas preparation both use it; Atlas supplies the exact workspace
   package set.
2. Add the internal `fairyc.fymap-logical/v1` model in `internal/atlasmap`. It is an immutable
   in-memory planning contract, not the `.fymap` transport schema.
3. Require a nonzero caller-supplied exact compiler executable identity and exactly one nonzero
   `fyif2-sha256` public-interface hash for every snapshot package. Reject missing, duplicate,
   unknown, or extra package inputs.
4. Build one deterministic partition for each exact `PackageInstanceId`, ordered by the snapshot's
   canonical package order. Assign every file, module, declaration, `SymbolId`, and function summary
   to exactly one owning package.
5. Record the snapshot model, compiler-build algorithm and value, language, package UID, instance,
   coordinate, source digest and kind, runtime ABI, feature and dependency sets, source-file and
   line-map algorithms, `SymbolId` algorithm, interface-hash algorithm and value, logical counts,
   generation status, and stable omissions in every header.
6. Mark this first plan `target-independent` with an empty target value. A target-dependent fact set
   requires an explicit later scope rather than silently changing this model.
7. Derive generation status from relation-specific snapshot completeness. Exact relations have no
   reason; partial or unavailable relations require nonempty stable reasons and produce sorted
   `<relation>:<status>:<reason>` omissions.
8. Return detached copies from every partition accessor. The plan performs no filesystem access,
   executable discovery, source reparsing, encoding, publication, or query work.

## Consequences

- Package ownership and compatibility failures are observable before transport design begins.
- Atlas and package-cache coordination share one interface-hash computation path.
- Consumers can distinguish an exact logical inventory from a partial semantic graph without
  treating absent facts as negative facts.
- The exact compiler executable remains caller-owned, so tests can supply a deterministic identity
  and production code cannot accidentally hash the wrong process image inside a pure planner.
- No `.fymap` bytes, section directory, byte order, checksum, atomic publication, current-source
  revalidation, dependency-map join, query API, or rewrite-cost claim is created by this decision.

## Evidence

Unit tests cover canonical complete multi-package interface hashing, caller order independence,
empty and duplicate package sets, and missing coordinates. Real checked-in workspace tests cover
repeatable snapshot-to-plan assembly, exact header algorithms, package partition coverage,
parseable identities, explicit omissions, no host paths, and detached accessors. Negative tests
reject zero compiler identity and missing, extra, or zero interface hashes.
