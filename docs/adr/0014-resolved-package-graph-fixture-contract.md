# Resolved Package Graph Fixture Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Region fixtures, Atlas cross-package identity, future module resolution, and cache keys need a real package-instance boundary. Defining `fairyc.toml`, a lockfile, registry coordinates, or a version solver now would freeze package-manager choices before the compiler can even consume a resolved graph.

Module paths also contain no package coordinate. If dependency visibility is left implicit, a transitive dependency or two direct dependencies with the same module path can silently change which declaration an import names.

## Decision

1. `spec/fixtures/package/manifest-v0.json` owns Stage 0 resolved-package-graph acceptance data.
2. The fixture input begins after dependency resolution and uses opaque identity tokens. It is not a public manifest, lockfile, registry, or solver format.
3. Package-instance equality uses package UID, resolved source identity, version identity, and the sorted feature set. The concrete `PackageInstanceId` algorithm remains undecided.
4. Every package instance owns one source root. File-to-module mapping remains package-local and unchanged.
5. Module identity includes the package instance. Equal module paths in isolated dependency sets do not collide.
6. Imports see the current package plus direct dependencies only. Equal-instance edges are deduplicated first. Transitive modules are hidden; zero instance providers is missing and multiple distinct instance providers are ambiguous.
7. Dependency bindings are unique per owner but do not rewrite source module paths.
8. The standard library is an explicit package instance and direct dependency, never an implicit prelude or host-discovered magical namespace.
9. The dedicated `pass.package.stdlib-arena-region` graph owns the bounded opaque `std::alloc::Arena` fixture type used by Region v1. Package pass means resolution succeeds; later Flow rejection remains owned by the Region fixture.
10. Pass cases assert instance-equivalence classes and successful module lookups. Reject cases use fixture-rule reasons rather than reserving user diagnostic codes prematurely.
11. The first cases cover explicit standard-library injection, bounded Arena resolution, identical-instance deduplication, feature-split instances, isolated equal module paths, ambiguous direct providers, hidden transitive modules, and duplicate dependency bindings.

## Consequences

- Compiler package-graph tests can begin before public package-manager syntax is chosen.
- A future resolver cannot search transitive dependencies as a fallback.
- `PackageUid` and `PackageInstanceId` remain separate concepts in Atlas, caches, and diagnostics.
- The package fixture names the standard-library instance and opaque Arena type; Region v1 owns the later lexical-escape assertion.

## Revisit conditions

Version or replace this schema when public manifest or lockfile input is accepted, dependency bindings enter source syntax, feature unification changes resolved identity, workspace members gain shared lookup, or the compiler needs a serialized concrete `PackageInstanceId`.
