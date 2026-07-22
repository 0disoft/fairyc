# ADR 0174: Use package coordinates for stable linking and dependency cache keys

- Status: Accepted
- Date: 2026-07-22
- Implementation: `internal/packageartifact/`, `internal/packagecache/`, `internal/workspacecompiler/`
- Contract: `docs/package/whole-package-cache-v2.md`
- Prerequisites: ADR 0172 and ADR 0173
- Supersedes: ADR 0141 dependency-key semantics, ADR 0152 FunctionID v1, and stable context/body identity v1

## Context

Public-interface v2 stopped dependency source-only edits from changing public type identity, but
callers still embedded exact source-sensitive `PackageInstanceId` values in dependency cache pairs,
`FunctionID`, nominal stable types, and foreign declarations. Weakening only the cache key would
have produced hits whose imported symbols could not resolve against a freshly built provider.

## Decision

Migrate the complete public linking boundary together:

- `FunctionID` v2 uses the owning `PackageCoordinateId` and coordinate-based nominal types;
- type-context and function-body artifacts advance to v2 and use coordinates for rebinding;
- stable package MIR advances to version 3, retaining exact `PackageInstanceId` for record ownership
  while adding an explicit coordinate for public linking;
- whole-package cache key v2 keeps the current package exact, but each dependency pair becomes
  `PackageCoordinateId + fyif2`;
- rebinding rejects absent or duplicate local coordinates and re-enters the MIR verifier.

Historical readers fail closed because every changed encoding has a new version or text prefix.

## Evidence

All Go tests and the configured Windows/MSVC smoke pass. Private dependency edits now produce one
verified caller hit and one provider miss. The persistent staircase is `2, 3, 4, 5, 6, 6, 6, 6, 6`,
with two total evictions, zero record quarantine, and final retained bytes `22444` in this local
observation. Public dependency edits still invalidate provider and caller.

## Limits

Coordinate equality does not prove equal source. Exact source remains part of the provider's own key
and record. Duplicate coordinates fail rather than selecting a source. This is bounded control
evidence, not a default cache policy or speed claim.
