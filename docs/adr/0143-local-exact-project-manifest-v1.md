# ADR 0143: Adopt a strict TOML local-exact project manifest v1

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Manifest source: `docs/package/project-manifest-v1.md`
- Readiness audit: `docs/package/manifest-lockfile-readiness.md`
- Fixture source: `spec/fixtures/project-manifest/manifest-v1.json`

## Context

The compiler now validates resolved package identity, visibility, source/interface invalidation,
cache keys, and dependency acyclicity, but users still have no project file. Reusing the resolved
graph fixture would make people author post-resolution IDs and test-only node handles. JSON would
avoid a dependency but makes a human-edited project file hostile to comments and routine changes.
A project-owned partial TOML parser would look small until escaping, multiline strings, duplicate
keys, arrays, tables, and source-positioned failures became permanent compiler maintenance work.

## Technology decision

Decision state: `adopt_with_constraints`.

| Candidate | Result | Reason |
| --- | --- | --- |
| No manifest | Reject | Prevents ordinary project builds and leaves source closure manual |
| JSON | Reject for human manifest | Standard-library implementation is cheap, but comments and editing ergonomics are poor |
| Project-owned partial TOML | Reject | Creates a misleading dialect and a long-lived parser/security burden |
| Pelletier TOML v2 | Defer | Viable adapter alternative, but no stronger repository-local evidence justified a second dependency candidate |
| BurntSushi TOML v1.6.0 | Adopt with constraints | Mature reflection API, TOML 1.1 support, duplicate-key parse errors, source-positioned errors, strict undecoded-key metadata, MIT license, Go 1.18 floor, and no transitive module dependencies |

The dependency is pinned by `go.mod` and `go.sum`, isolated behind `internal/projectmanifest`, and
recorded in `THIRD_PARTY_NOTICES.md`. Replacing it affects only the byte-decoder adapter as long as
the semantic `Manifest` and `ValidationError` contracts remain unchanged. Local configured checks
keep `GOPROXY=off` and read only exact checksum-verified modules already present in the user's
standard Go module cache. CI uses ordinary read-only module resolution and retains no dependency
cache until measurements justify one.

## Manifest decision

1. The human file name is `fairy.toml`; manifest schema identity is `fairyc.manifest/v1`.
2. Input is valid UTF-8 and at most 1 MiB. TOML syntax errors and duplicate keys reject before
   semantic validation. Unknown fields reject through decoder metadata.
3. `[package]` requires `uid`, `name`, exact `version`, exact `language`, and explicit `source`.
4. `uid` has at least two lowercase dot-separated kebab segments. `name`, dependency bindings, and
   feature names use lowercase kebab-case.
5. Versions are exact SemVer 2.0.0, including valid prerelease/build suffixes. Ranges are not syntax.
6. Language revision uses exact `fairyc-language/<nonnegative integer>` spelling.
7. Source, workspace-member, and dependency paths are canonical portable relative directories:
   forward slash, no volume, colon, absolute root, `.`/`..`, traversal, backslash, or trailing slash.
8. `[workspace]` is optional. When present, `members` is required, sorted, and unique. There is no
   parent search or implicit recursive discovery.
9. Each `[dependencies.<binding>]` requires path, exact version, and an explicit sorted unique
   feature array, including `features = []` when empty. Table order is irrelevant; validated output
   sorts dependencies by binding.
10. V1 admits local path declarations only. Git, URL, archive, registry, optional dependency,
    default feature, version range, target conditional, environment expansion, and user-specified
    standard-library dependency fields are unknown and reject.
11. The byte parser does not touch the filesystem. Workspace containment, member consistency,
    source discovery, symlink/case policy, exact standard-library injection, resolved graph
    construction, and lockfile writing are the next boundary.

## Consequences

- Users get a comment-capable conventional manifest without making TOML parsing a compiler project.
- Manifest v1 remains intentionally weaker than Cargo or Go modules: no solver and no network.
- Exact required fields avoid hidden defaults and make LLM/project tooling extraction mechanical.
- One external dependency enters the Stage 0 compiler, so offline bootstrap must obtain it once or
  use a future audited vendoring/reproducible-source policy. Tests themselves do not enable network.
- Rolling back the dependency means replacing only `internal/projectmanifest` parsing while
  preserving fixtures and semantic output.

## Verification

- eight versioned complete-file fixtures cover minimal/workspace passes and unknown, duplicate,
  missing, path, feature-order, and schema failures;
- unit tests cover exact semantic output, dependency sorting, required fields, UID/name/version/
  language/path rules, prerelease leading zeros, invalid UTF-8, and the 1 MiB bound;
- full Go format, test, build, and vet validation runs through the offline local wrapper.
