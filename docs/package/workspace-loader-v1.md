# Local workspace loader v1

- Status: Executable prototype contract
- Decision: `docs/adr/0144-local-workspace-loader-v1.md`
- Implementation: `internal/workspaceloader/`
- Fixtures: `spec/fixtures/workspace-loader/manifest-v1.json`

## Input closure

The caller names one directory. Its root `fairy.toml` and exact explicit `workspace.members` form
the complete package catalog. There is no parent search, wildcard, recursive manifest discovery,
nested workspace, overlapping member, or implicit dependency package.

All catalog manifests must parse under `fairyc.manifest/v1`, declare one unique `PackageUid`, and use
the root's exact language revision. Dependency paths are workspace-root-relative and must name an
explicit member with the exact declared version.

## Filesystem boundary

Every path segment must match its on-disk case exactly. The v1 loader rejects symlinks in package,
manifest, source-root, directory, and source-file positions. Every regular `.fy` file recursively
below the explicit source root participates; other regular files are ignored. Source roots cannot
be empty, source paths cannot collide under case folding, and two files cannot declare the same
module path.

The implementation guards one load to 1,024 catalog packages, 4,096 `.fy` files, and 64 MiB of
exact source bytes. Per-file frontend limits still apply. These are denial-of-service guards for the
prototype and do not define stable language limits.

## Deterministic output

Each loaded package retains:

- canonical workspace-relative package path (`.` for the root);
- parsed manifest;
- runtime-only absolute package and source-root paths;
- sorted canonical source paths, exact bytes, and parsed AST modules;
- sorted module/import records;
- package-source digest v1.

The root-reachable graph uses the source digest text as local source identity and creates one node
per exact member path and per-edge feature set. It does not unify features. Existing package-graph
resolution supplies full PackageInstanceId classes, cycle rejection, and direct-only module lookup.

## Deferred boundary

This loader does not select or inject an SDK standard library, compile a semantic package input
through the CLI, access a registry, solve ranges, fetch sources, or persist a cache. ADR 0145's
separate lockfile library may project this result, but absolute paths remain unserialized.
