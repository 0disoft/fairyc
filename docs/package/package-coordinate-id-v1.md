# PackageCoordinateId v1

- Status: Executable prototype prerequisite
- Decision: `docs/adr/0172-source-independent-package-coordinate.md`
- Implementation source: `internal/packagegraph/coordinate.go`

## Purpose

`PackageInstanceId` identifies one exact resolved source tuple and must change when source content
changes. `PackageCoordinateId` is a separate public-link namespace that deliberately survives a
source-only edit. It does not identify source bytes, authorize cache reuse, replace a lockfile
identity, or prove that two implementations are compatible.

The intended later consumer must combine the coordinate with an exact public-interface hash. A
private source edit may then preserve imported symbol and nominal-type identity only when the public
contract is unchanged. The current cache and stable artifact continue to use exact instance
identity until that complete integration lands.

## Input tuple

The input is:

1. exact `PackageUid`;
2. exact version identity;
3. the sorted unique resolved feature set.

Source identity is absent by construction. Components are nonempty valid UTF-8. Features are
nonempty valid UTF-8 and already sorted and unique by exact byte order.

## Canonical encoding and digest

The encoder writes ASCII `fairyc.package-coordinate`, `00`, version byte `01`, then tagged `u` and
`v` strings and the tagged `f` feature list. Strings and counts use unsigned 64-bit network-order
lengths. SHA-256 hashes the complete bytes. The only text form is:

```text
fypc1-sha256-<64 lowercase hexadecimal digits>
```

Readers reject a missing or different prefix, uppercase or non-hexadecimal text, and short or
extended digests.

## Boundary

- `PackageInstanceId`: exact source/version/features identity and lock/cache record ownership.
- `PackageCoordinateId`: source-independent namespace candidate for public linking.
- public-interface hash: equality of the selected public semantic projection.
- cache key: complete compatibility decision; coordinate equality alone is never enough.

The workspace loader exposes exact instance and coordinate maps separately. Callers must choose the
identity matching their contract rather than converting one digest into the other.
