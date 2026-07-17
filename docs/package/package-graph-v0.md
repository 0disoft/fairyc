# Resolved Package Graph v0

- Status: Draft Stage 0 fixture contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0014-resolved-package-graph-fixture-contract.md`
- Fixture source: `spec/fixtures/package/manifest-v0.json`

## Boundary

This contract starts after dependency resolution. It does not define public package coordinates, manifest or lockfile syntax, version-range solving, feature unification, registries, network access, or source download. The fixture graph provides those outcomes as explicit opaque inputs so the compiler can test identity and visibility without accidentally standardizing a temporary package manager.

## Identity

`PackageUid` names a long-lived source project. A fixture package instance is the exact tuple:

```text
(PackageUid, resolved source identity, version identity, sorted feature set)
```

Equal tuples denote one logical package instance inside a resolved graph. A different tuple member denotes a different instance even if all source module paths happen to match. Fixture instance-class labels are local comparison names, not serialized `PackageInstanceId` values. The concrete digest, encoding, collision policy, and lockfile representation remain UNDECIDED.

Feature sets are sorted unique strings. This contract does not decide whether a future solver unifies features; it only requires the resolved result supplied to the compiler to preserve the chosen set in instance identity.

## Source roots and modules

Every instance owns exactly one source root. Source-root-relative `.fy` paths map to module paths using the existing case-sensitive Stage 0 rule. Package coordinates and instance identities never become module-path segments.

Module identity is `(PackageInstanceId, module path)`. Two isolated package instances may both own `util` without colliding. Inside one importing package, a module import may resolve only to:

1. a module in the current package instance; or
2. a module in one direct dependency instance.

Transitive dependency modules are not visible. Direct edges that resolve to the same logical package instance are deduplicated before lookup. Zero visible instance providers rejects the lookup as missing; more than one distinct visible instance provider rejects it as ambiguous. The dependency binding in the resolved graph is lockfile-edge identity and does not silently rewrite source module paths.

## Standard library

The standard library is a normal explicit package instance with the fixture role `standard-library`. It is injected as a direct dependency when a fixture needs `std::*`. There is no implicit prelude, magical host installation lookup, or type created merely because a user package declares the same name.

The initial package fixture proves that `std::alloc` belongs to the declared standard-library instance. It does not yet define the complete `Arena` fixture API or public standard-library package metadata.

## Graph validation

- Node identifiers are fixture-local and unique.
- Dependency owners and targets name declared nodes.
- Dependency bindings are unique within one owner.
- Pass-case instance classes cover every node exactly once and group exactly equal identity tuples.
- Source lists are complete and sorted by unsigned UTF-8 path bytes.
- Import lookup considers only the current instance and direct dependency targets.

Reject reasons in this fixture schema are test-rule identities, not released user diagnostic codes.

## Deferred surface

Public coordinates, manifest and lockfile schemas, source kinds and checksums, registry policy, offline and vendored behavior, version selection, feature unification, dependency cycles, workspace discovery, package signing, interface hashes, and the concrete `PackageInstanceId` algorithm remain separate decisions.
