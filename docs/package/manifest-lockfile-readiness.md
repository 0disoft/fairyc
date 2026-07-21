# Manifest and lockfile readiness audit

- Status: Design audit; wire formats remain UNDECIDED
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Resolved graph source: `docs/package/package-graph-v0.md`
- Identity source: `docs/package/package-instance-id-v1.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Result

The compiler side is ready to consume a resolved local package graph, but the repository is not yet
ready to claim a public manifest or lockfile schema. The missing pieces are product semantics, not
JSON/TOML field spelling. Reusing the resolved-graph fixture as a user format would expose fixture
node handles, source-root test layout, and post-solver identities as if users were meant to author
them.

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

## Decisions that are still missing

| Area | Required decision | Why code cannot guess it |
| --- | --- | --- |
| Human file | file name, syntax, comment and rewrite policy | Choosing JSON, TOML, or a custom grammar changes UX and dependencies |
| Package coordinate | legal `PackageUid`, display name, and rename rules | Identity persistence and registry naming cannot be inferred from module paths |
| Version request | exact versions, ranges, prereleases, and solver ordering | Different solvers can produce different graphs from the same text |
| Source kinds | workspace/path, registry, Git, archive, vendor, checksums | Network trust and reproducibility differ by source kind |
| Features | per-edge instances, unification, defaults, weak/optional dependencies | Existing fixtures only preserve the resolved set; they do not choose it |
| Workspace | member discovery, shared dependencies, duplicate UID handling | Filesystem walking and parent lookup can make builds ambient and slow |
| Source discovery | default root, include/exclude rules, symlinks, case collisions | These rules define source identity and security boundaries |
| Standard library | how compiler/SDK selection injects and locks its package | Host installation paths must not become semantic identity |
| Lock updates | frozen mode, ordinary update mode, stale local paths, merge policy | Silent rewrite would damage reproducibility; never rewriting would hurt local work |
| Cycles | package-cycle policy and diagnostics | The current resolved graph does not yet reject all cycle forms |

## Recommended Stage 0 local-exact profile

The smallest coherent profile is intentionally less ambitious than Cargo, Go modules, or npm:

1. Use one project-root manifest and one generated lockfile. Prefer `fairy.toml` and `fairy.lock`
   only after the parser/dependency cost and lossless human-file rewrite policy are accepted.
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
7. Reject package dependency cycles in Stage 0. Initialization and duplicate mutable package state
   semantics are not mature enough to make cycles useful.
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

This profile is a recommendation, not an accepted wire contract. It deliberately postpones public
registry coordinates, network acquisition, signatures, version-range solving, feature unification,
implicit workspace discovery, vendoring commands, and package publication.

## Minimum acceptance evidence before implementation

- an ADR accepts the ten local-exact choices and the manifest parser strategy;
- positive fixtures cover one package, explicit workspace members, a direct path dependency,
  feature-distinct instances, and exact standard-library injection;
- negative fixtures cover unknown/duplicate fields, path escape, absolute paths, escaping symlinks,
  duplicate UID, module ambiguity, dependency cycles, unsorted/duplicate features, stale lock,
  contradictory `PackageInstanceId`, and unsupported network sources;
- lock output is byte-deterministic across input enumeration order and uses atomic no-partial-write
  replacement;
- Windows path, case, separator, and rename behavior is tested without WSL;
- the CLI distinguishes parse, resolve, locked, integrity, and source-containment failures;
- no cache, generated output, timestamp, absolute host path, credential, or environment variable is
  treated as source truth or serialized into semantic identity.
