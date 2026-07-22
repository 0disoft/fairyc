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

Each local package retains:

- canonical workspace-relative package path (`.` for the root);
- parsed manifest;
- runtime-only absolute package and source-root paths;
- sorted canonical source paths, exact bytes, and parsed AST modules;
- sorted module/import records;
- package-source digest v1.

The root-reachable graph uses the source digest text as local source identity and creates one node
per exact member path and per-edge feature set. It does not unify features. Existing package-graph
resolution supplies full PackageInstanceId classes, cycle rejection, and direct-only module lookup.
`SemanticInput` assigns disposable build-local package numbers and binds every resolved import to
its exact provider for `compiler.CheckPackages`.

When any root-reachable package imports `std` or `std::*`, ADR 0147 injects one exact compiler-bundled
SDK package as a direct `stdlib` dependency. The SDK source bytes, package UID, version, language,
runtime ABI, source digest, and composed source identity are compiler-owned and host-path-free.
`fairyc.std`, the `stdlib` dependency binding, and local `std` module namespace are reserved. A
missing SDK module still rejects through ordinary package-graph lookup; no prelude or installation
search exists.

## Isolated semantic-test overlay

ADR 0181 adds an explicit in-memory overlay after an ordinary production workspace is already
loaded and its lock is maintained or verified. Only the root package's exact-lowercase `tests/`
directory is considered. Ordinary `Load`, check, build, run, emit, package identity, public
interface, cache, and lock paths never read that directory.

The optional directory is contained, non-symlink, disjoint from the production source root and
workspace members, and shares the same aggregate source budgets. Each test path retains its
`tests/` prefix before syntax analysis, so `tests/net/http.fy` declares
`module tests::net::http;`. Test sources are marked in the ephemeral catalog, sorted with production
sources, included in an ephemeral source digest, and used to rebuild module lookup and exact
dependency visibility. That digest is compile-local test evidence and is never written to
`fairy.lock`.

## Deferred boundary

This loader does not access a registry, solve ranges, fetch sources, link target standard-library C
objects, persist a cache, recursively discover member test trees, or choose test declarations. The
CLI harness owns `@test` validation. The separate lockfile library projects production local and SDK
source origins, but absolute paths and test overlays remain unserialized.
