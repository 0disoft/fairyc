# Whole-package cache key and stable linking v2

- Status: Executable prototype contract
- Decision: `docs/adr/0174-coordinate-stable-linking-and-cache-key-v2.md`
- Fixture source: `spec/fixtures/package-cache/manifest-v5.json`
- Implementation: `internal/packagecache/`, `internal/packageartifact/`, `internal/workspacecompiler/`

## Contract

The current package remains exact: its `PackageInstanceId` and source digest invalidate its own
artifact. Compiler, language, MIR, target, ABI, toolchain, backend, runtime, standard-library,
profile, and feature inputs remain mandatory. A dependency entry is the sorted unique pair
`PackageCoordinateId + fyif3 public-interface hash`. The package-cache key encoding remains v2;
fixture manifest v3 records the newer independently versioned interface component.

The canonical encoding is `fairyc.package-cache-key/v2`; its only text form is
`fycache2-sha256-<64 lowercase hexadecimal digits>`. V1 keys are incompatible.

`FunctionID` v2 (`fyfn2-sha256-...`), type-context v2 (`fyctx2-sha256-...`), body v2, and stable
package MIR v3 use coordinates for cross-package symbols and nominal types. Stable package records
separately retain their exact instance. Rebinding requires a one-to-one coordinate map, resolves
imports and resource drops, merges only equal declarations, and reruns MIR verification.

Private current-package edits miss only that package. Private dependency edits hit the caller and
miss the provider. Public dependency edits miss both. The measured six-entry staircase is
`2, 3, 4, 5, 6, 6, 6, 6, 6`, with two evictions, zero quarantines, and 22444 final retained bytes.
These are observations, not product defaults or performance promises.
