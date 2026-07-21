# ADR 0144: Fix the Stage 0 local workspace loader boundary

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Manifest source: `docs/package/project-manifest-v1.md`
- Loader source: `docs/package/workspace-loader-v1.md`
- Fixture source: `spec/fixtures/workspace-loader/manifest-v1.json`

## Context

ADR 0143 made `fairy.toml` parseable but intentionally stopped before touching the filesystem. A
loader that merely joins strings and recursively searches for manifests would make the build depend
on ambient directories, host case behavior, and symlink targets. It would also blur the already
fixed distinction between one source digest and feature-distinct package instances.

Stage 0 needs a smaller contract than a general package manager: one explicitly selected local root,
explicit members, exact path dependencies, deterministic source discovery, and a reusable resolved
graph. Registry acquisition, a solver, SDK selection, and lockfile writes were left as later
boundaries; ADRs 0145 and 0147 subsequently close the lock and compiler-bundled SDK parts.

## Decision

1. The caller selects one workspace directory. The loader reads only its `fairy.toml`; it never
   searches parents or recursively adopts manifests.
2. The package catalog is the root plus the root manifest's exact sorted `workspace.members`.
   Members cannot declare another `[workspace]`. Case-folding duplicates, ancestor/descendant member
   overlaps, more than 1,024 packages, and duplicate package UIDs reject.
3. Every portable manifest path must have one exact on-disk segment spelling. Stage 0 rejects any
   symlink in a member path, manifest file, source-root path, or discovered source tree. This stricter
   rule avoids host-dependent containment and identity even for links that currently resolve inside.
4. Every catalog package uses its explicit `package.source`. The loader recursively includes every
   regular `.fy` file below it, with no defaults, excludes, generated-directory exceptions, or
   hidden-file exception. An empty source root rejects.
5. Source paths are sorted by canonical forward-slash spelling. Case-folding path collisions,
   duplicate declared module paths, parser-invalid sources, more than 4,096 files, and more than
   64 MiB of exact source bytes reject. These counts are implementation guards, not language limits.
6. All catalog packages use the root's exact language revision. Every dependency path names an
   explicit non-root member and its exact declared version. Unknown paths and version mismatches
   reject before graph construction.
7. Local source identity is the full package-source digest v1 of canonical relative paths and exact
   bytes. The absolute host path remains runtime-only metadata and never enters package identity.
8. Graph construction begins at the root and includes only reachable instances. Each dependency
   edge's exact feature set creates or reuses one `(member path, feature set)` instance; features are
   not unified. The existing package graph then rejects cycles, duplicate bindings, missing modules,
   and ambiguous visible providers.
9. The local member/source portion performs no network access, version solving, lockfile read/write,
   cache write, environment expansion, or Git inspection. ADR 0147 extends its graph output with
   conditional compiler-bundled standard-library injection without weakening those boundaries.

## Consequences

- The same checked-out bytes produce the same package/source identities across Windows and
  case-sensitive hosts, assuming the declared files have the same exact names and bytes.
- Rejecting all symlinks is intentionally blunt. A later manifest schema may admit a separately
  specified contained-link policy without weakening v1.
- Unreachable workspace members are still parsed and source-validated as catalog members, but only
  root-reachable instances enter the resolved graph and generated root lockfile.
- Rolling back this loader removes `internal/workspaceloader` without changing manifest byte
  parsing or the reusable resolved-graph contract.

## Verification

- complete-directory fixtures cover a feature-distinct pass plus exact-version, duplicate-UID,
  path-case, and cycle rejection;
- unit tests additionally cover nested/overlapping members, unknown dependencies, language
  mismatch, empty or invalid sources, symlink rejection, source digests, and graph shape;
- configured Windows format, test, build, and vet checks exercise the loader without WSL.
