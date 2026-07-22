# ADR 0173: Use package coordinates in public-interface hash v2

- Status: Accepted
- Date: 2026-07-22
- Implementation: `internal/moduleinterface/`, `internal/moduleinterfacefixture/`
- Contract: `docs/package/public-interface-hash-v2.md`
- Coordinate prerequisite: `docs/adr/0172-source-independent-package-coordinate.md`
- Supersedes: `docs/adr/0139-public-interface-hash-v1.md`

## Context

Public-interface v1 correctly excluded private implementation, but external nominal type spellings
embedded full source-sensitive `PackageInstanceId`. A transitive private edit could therefore alter
an otherwise equal public interface. ADR 0172 introduced a separate coordinate without weakening
exact source identity.

## Decision

Advance the public-interface encoding and hash text prefix to v2. Preserve every selected semantic
fact from v1, but render external nominal types through `PackageCoordinateId`. Keep local public
types under `self::`, so the package's own private source edit remains inert.

Advance the executable fixture manifest and schema to v2 while retaining v1 files as historical
vectors. The frozen v2 baseline is
`fyif2-sha256-5942803d5a2f559975825d870916c24c3f5012fbaae13ab8c1075679d837dd6d`.

The workspace compiler now supplies the explicit coordinate map to interface extraction. Cache keys
still contain exact dependency instances, and stable `FunctionID` still contains exact source
identity, so this decision alone does not enable new reuse.

## Evidence

Eight source fixtures preserve or invalidate the same semantic classes as v1 under the new exact
baseline. Unit tests cover strict text parsing, package renumbering, coordinate changes, private
type leakage, public shapes, effects, and foreign link names. The full Go suite passes with all cache
fixtures consuming the v2 interface hash format.

## Consequences and limits

- Public interface equality no longer changes merely because an external package's source digest
  changed while coordinate and public facts stayed equal.
- Version, feature, or UID changes still alter external nominal type identity.
- Stable MIR symbols and whole-package dependency keys must migrate together before private
  dependency reuse is safe.
- Native ABI, inline bodies, serialized module interfaces, registry compatibility, and multi-source
  override semantics remain outside this hash.
