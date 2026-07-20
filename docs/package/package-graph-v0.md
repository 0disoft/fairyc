# Resolved Package Graph v0

- Status: Draft Stage 0 fixture contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0014-resolved-package-graph-fixture-contract.md`
- Fixture source: `spec/fixtures/package/manifest-v0.json`
- Implementation sources: `internal/packagegraph/`, `internal/packagefixture/`

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

The initial package fixture proves that `std::alloc` belongs to the declared standard-library instance. The dedicated `pass.package.stdlib-arena-region` case additionally exports an opaque `Arena` resource used as `&mut std::alloc::Arena` by Region v1. It deliberately defines no constructor, capacity operation, representation promise, or public standard-library package metadata. A package-graph pass says resolution succeeds even when a later Flow-owned Region assertion intentionally rejects the source.

## Graph validation

- Node identifiers are fixture-local and unique.
- Dependency owners and targets name declared nodes.
- Dependency bindings are unique within one owner.
- Pass-case instance classes cover every node exactly once and group exactly equal identity tuples.
- Source lists are complete and sorted by unsigned UTF-8 path bytes.
- Import lookup considers only the current instance and direct dependency targets.

Reject reasons in this fixture schema are test-rule identities, not released user diagnostic codes.

## Executable evidence

`internal/packagefixture` strictly decodes the v0 manifest, rejects duplicate JSON keys and unknown
fields, proves that each declared source list exactly matches the contained regular `.fy` files,
parses every source with its package-local module identity, and executes all eight manifest cases
through `internal/packagegraph`. The graph engine compares the exact resolved-instance tuple,
deduplicates equal instances, validates unique direct bindings, and resolves imports through only
the current instance and direct dependency instances. Passing graphs can then produce build-local
semantic package input: one nonzero `PackageID` per exact instance class, one parsed source set per
class, and exactly one import-provider binding per source import from the validated lookup result.
Missing, duplicate, unused, or absent-package bindings reject semantic input rather than falling
back to a same-named module. The
`pass.package.module-path-isolation` case completes semantic Flow with four distinct HIR package
identities even though both dependency instances own `util.fy` and module `util`. The Region v1
runner consumes the same validated `pass.package.stdlib-arena-region` result instead of maintaining
a second package-lookup interpretation.

The numeric IDs and compiler-private module qualification are disposable build facts. Source spans
store the numeric discriminator separately while retaining their original relative path; they never
render `$pN` as part of that path. Neither representation is the deferred public
`PackageInstanceId`, a lockfile encoding, an ABI name, or a stable diagnostic path.

The module-path-isolation result additionally crosses `internal/compiler` through semantic analysis,
MIR lowering, and MIR verification. This proves compiler-pipeline consumption of the resolved graph;
it still does not prove package discovery, a package-aware CLI, or backend support for package names.

This is compiler-side consumption of an already-resolved graph. It is not evidence for a package
solver, public manifest, lockfile, source downloader, registry, cache, or stable diagnostic surface.

## Deferred surface

Public coordinates, manifest and lockfile schemas, source kinds and checksums, registry policy, offline and vendored behavior, version selection, feature unification, dependency cycles, workspace discovery, package signing, interface hashes, and the concrete `PackageInstanceId` algorithm remain separate decisions.
