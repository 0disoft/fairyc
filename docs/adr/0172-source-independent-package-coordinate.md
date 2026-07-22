# ADR 0172: Separate exact source identity from public package coordinates

- Status: Accepted prerequisite
- Date: 2026-07-22
- Implementation: `internal/packagegraph/coordinate.go`, `internal/workspaceloader/semantic.go`
- Package contract: `docs/package/package-coordinate-id-v1.md`
- Exact identity prerequisite: `docs/adr/0138-package-instance-id-v1.md`

## Context

The persistent growth measurement still misses both packages after a private dependency edit.
Public-interface extraction already excludes that private declaration. The miss remains because a
dependency pair contains full `PackageInstanceId`, and stable imported `FunctionID` values also use
that source-sensitive identity. Removing the instance from the cache key alone would make an old
caller artifact refer to function IDs that the new provider artifact does not own.

One digest cannot truthfully mean both “these exact source bytes” and “this public package namespace
is unchanged.” Weakening `PackageInstanceId` would damage lockfile, source, and cache-record
integrity. Reusing it for public linking guarantees over-invalidation.

## Decision

Add exact-version `PackageCoordinateId` v1 over package UID, version identity, and resolved features,
excluding source identity. The workspace loader computes and exposes both identities for every
semantic package:

- `InstanceID` continues to name the exact resolved source tuple;
- `CoordinateID` is a source-independent namespace prerequisite for later public linking.

The coordinate uses domain-separated, length-framed canonical bytes, full SHA-256, and the
`fypc1-sha256-` text form. Input validation shares the package identity rules for UTF-8 and sorted
unique features.

This ADR does not change cache keys, public-interface type spellings, `FunctionID`, stable MIR, or
reuse behavior. Those surfaces must move together in a later exact-version change so no cached
caller can bind to an incompatible provider.

## Evidence

Unit tests require source-only changes to preserve the coordinate, version changes to alter it,
framing to distinguish ambiguous concatenations, strict parse round trips, and rejection of empty,
unsorted, duplicate, uppercase, or non-hexadecimal inputs. Workspace tests require every semantic
package to expose the same nonzero coordinate through both the metadata list and coordinate map.

## Consequences and limits

- Exact source identity remains fail-closed and unchanged.
- A public-link namespace can now survive private source edits without pretending the source is the
  same.
- Coordinate equality alone permits no reuse or linking. Public-interface equality and complete
  provider resolution remain mandatory.
- UID/version/feature coordinate policy relies on the current resolver rejecting duplicate package
  UIDs in one workspace. Multi-source override or patch semantics remain UNDECIDED and must extend
  the coordinate contract before admission.
