# ADR 0171: Project and merge package-scoped stable contexts

- Status: Accepted
- Date: 2026-07-22
- Implementation: `internal/packageartifact/context_projection.go`, `internal/packageartifact/stable_artifact.go`
- Measurement: `internal/buildmeasure/persistent_cache_growth.go`, `cmd/fairyc-windows-host-smoke/main.go`
- Supersedes package-envelope semantics in: `docs/adr/0162-stable-package-mir-artifact.md`
- Supersedes the growth measurement contract in: `docs/adr/0170-persistent-cache-retention-staircase.md`

## Context

Stable package MIR v1 put the complete workspace type and foreign-declaration context in every
package artifact. Cache keys were package-oriented, so an unrelated private root declaration could
leave a dependency key unchanged while changing the dependency record bytes. The immutable disk
store correctly treated the two valid generations under one key as a producer conflict, but the
result was avoidable retention, quarantine, fallback, and rewriting.

Changing the store to last-writer-wins would only hide the broken key-to-bytes relation. The package
artifact must instead depend only on semantic declarations used by its bodies.

## Decision

Advance the stable package artifact version to `fairyc-stable-package-mir/2`. Keep the existing
exact-version context and body shapes, but change the package envelope semantics:

- project types mentioned by owned function signatures, locals, operands, operations, and
  terminators;
- close that set transitively over struct, enum, and resource fields;
- include only foreign declarations called by owned bodies;
- include resource drop-function links and compute the exact external `FunctionID` import set from
  body calls plus projected resources;
- sort and validate every projected context before it can enter the package artifact;
- merge decoded package contexts by stable type or foreign-function identity, accepting identical
  repeats and rejecting conflicting definitions;
- rebuild one process-local merged context, then rebind each body against its own `ContextID` and
  local foreign index table;
- rerun ordinary closed-world `mir.Verify` after the complete merge and rebind.

The stable type-context v1 descriptor remains a general deterministic descriptor. A process-local
artifact may still carry the full workspace descriptor for verification, while stable package MIR
v2 stores only its closed projection. The persistent cache key includes the stable artifact version,
so v1 records cannot be interpreted as v2 records.

## Evidence

Unit tests prove that an unreferenced private workspace declaration changes none of the encoded
stable package bytes. Separate tests create distinct package contexts, merge and rebind them with
fresh symbols, pass `mir.Verify`, and execute result `42`. Conflicting repeated type definitions are
rejected. The existing independent decode, rebind, re-extract, and byte-identical re-encode test
continues to pass.

The configured local Windows amd64 smoke with MSVC 14.51.36231 ran the nine-process growth sequence
using `fairyc.persistent-cache-growth-measurement/v1`. Retained entries followed
`2, 3, 4, 6, 6, 6, 6, 6, 6`; retained bytes peaked and finished at 21,700. The sequence performed
three evictions and zero record quarantines. The public no-op and both no-ops after the later private
root edit were complete two-package verified hits with no lowering.

The private dependency edit still produced zero hits and two misses. That is no longer a
key-to-artifact-byte conflict: current public-interface extraction invalidates the root package for
that private dependency declaration. Interface projection is therefore the next efficiency gate.

## Consequences and limits

- One stable cache key now determines one package artifact byte sequence for the covered private
  root-edit regression without weakening immutable conflict detection.
- Unrelated workspace declarations no longer inflate every package payload.
- Mixed package contexts are admitted only through deterministic equality-checked merge; missing,
  ambiguous, or contradictory declarations fail closed.
- The six-entry ceiling and exact byte observations are harness controls, not product defaults or
  representative repository budgets.
- Persistent caching remains experimental. A declared reference environment, broader workspace
  distribution, accepted noise policy, retained-growth budget, and default-path decision remain
  absent.
- Private dependency edits still over-invalidate dependents. Narrowing the public-interface hash
  must preserve visibility and ABI truth before any ordinary-command default is considered.
