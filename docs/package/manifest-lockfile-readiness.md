# Manifest and lockfile readiness audit

- Status: Local-exact manifest, workspace, lockfile, SDK injection, and experimental lock modes accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Resolved graph source: `docs/package/package-graph-v0.md`
- Identity source: `docs/package/package-instance-id-v1.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Result

The compiler side is ready to consume a resolved local package graph. ADR 0143 fixes the
human-authored `fairy.toml` v1 byte/parser and local-exact declaration semantics. ADR 0144 now fixes
explicit member loading, fail-closed path/source discovery, exact dependency consistency, local
source identity, and root-reachable feature-distinct graph construction. ADR 0145 now fixes a
separate canonical generated `fairy.lock` v1 with identity revalidation, stale comparison, and
atomic replacement. ADR 0147 fixes exact compiler-bundled SDK standard-library injection and its
closed lock record. `experimental check-workspace` supplies ordinary refresh and no-write
`--locked` behavior; stable package command spelling remains open.

## Facts already fixed

Any future manifest/lockfile design must preserve these existing contracts:

- one package instance is the exact `(PackageUid, source identity, version identity, sorted feature
  set)` tuple and uses full `PackageInstanceId` v1 in resolved records;
- one instance owns one contained source root and a sorted unique module set;
- imports see the current instance and direct dependency instances only;
- dependency binding names identify graph edges but do not rewrite source module paths;
- two visible providers of the same module path are an ambiguity, not an order-dependent winner;
- the standard library is an exact explicit package instance, not a magical module search path;
- the current package uses an exact path-and-byte source digest while dependents invalidate from
  public-interface hashes;
- manifest schema, lockfile schema, source identity, cache key, and package identity versions remain
  separate compatibility dimensions;
- unknown or contradictory identities fail closed; caches remain disposable misses.
- package dependency edges are acyclic after exact-instance deduplication; cycles reject before
  module lookup under ADR 0142.

## Remaining package-manager decisions

| Area | Required decision | Why code cannot guess it |
| --- | --- | --- |
| SDK evolution | versioning and capability policy for any surface beyond opaque `std::alloc::Arena` | Bundled bytes cannot silently become a broader public library contract |
| Stable lock commands | promotion path from experimental ordinary/`--locked` behavior | Stable spelling, diagnostics, and compatibility need an explicit CLI contract |

## Recommended Stage 0 local-exact profile

The smallest coherent profile is intentionally less ambitious than Cargo, Go modules, or npm:

1. Use the accepted `fairy.toml` v1 project manifest and generated `fairy.lock` v1.
2. Admit only workspace-relative path dependencies in the first implementation. Reject absolute
   paths, root escape, symlinks that escape, Git, URL, archive, and registry sources. The compiler
   performs no network access.
3. Require one exact package version in every package and exact dependency compatibility in Stage 0.
   Do not implement version ranges or a general solver yet.
4. Keep features per resolved dependency edge and allow two feature-distinct instances. Do not
   silently unify features. Require sorted unique feature names and no implicit defaults in the
   first schema.
5. Discover members only from an explicit sorted workspace list. Do not search parent directories
   or recursively adopt ambient manifests.
6. Use one explicit contained source root per package, defaulting only if the schema says so. Walk
   `.fy` files deterministically, reject path/case collisions and escaping links, and feed exact
   canonical paths to package-source digest v1.
7. Preserve ADR 0142 package-cycle rejection in every manifest and lockfile input path.
8. Have the selected SDK inject one exact standard-library lock entry. Record its package instance,
   language requirement, runtime ABI, and source/content verification without recording the local
   SDK path as identity.
9. Make the generated lockfile the complete resolved graph: schema and identity algorithm versions,
   package tuples and full IDs, canonical relative source locations for local packages, exact
   features, direct binding edges, standard-library record, and integrity facts. Recompute every ID
   and reject contradictions while reading.
10. Separate ordinary local refresh from `--locked` behavior. A locked build never rewrites; a stale
    or missing lock is an error. The ordinary command may replace the lock atomically only after a
    complete valid graph is produced.

ADR 0143 accepts the manifest-facing parts, ADR 0144 accepts local workspace loading through
resolved graph construction, and ADR 0145 accepts the generated lock representation. The profile deliberately postpones public registry coordinates,
network acquisition, signatures, version-range solving, feature unification, implicit workspace
discovery, vendoring commands, and package publication.

## Acceptance evidence retained for SDK and CLI integration

- ADR 0143 accepts the manifest parser, ADR 0144 accepts local resolution, ADR 0145 accepts lock
  bytes, and ADR 0147 accepts compiler-bundled SDK selection;
- existing positive fixtures cover one package, explicit workspace members, direct path
  dependencies, feature-distinct instances, and exact standard-library injection;
- existing negative fixtures cover unknown/duplicate fields, path escape, absolute paths,
  symlinks, duplicate UID, module ambiguity, dependency cycles, and unsorted/duplicate features;
  stale lock and contradictory `PackageInstanceId` lock records are covered by lock unit tests;
- lock output is byte-deterministic, strictly revalidated, stale-detecting, and atomically replaced;
- Windows path, case, separator, and rename behavior is tested without WSL;
- the CLI distinguishes parse, resolve, locked, integrity, and source-containment failures;
- no cache, generated output, timestamp, absolute host path, credential, or environment variable is
  treated as source truth or serialized into semantic identity.
