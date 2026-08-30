# Whole-package cache key v3

- Status: Experimental executable contract
- Decision: `docs/adr/0345-hir-contract-state-interface-and-cache-identity.md`
- Fixture source: `spec/fixtures/package-cache/manifest-v7.json`
- Historical source: `docs/package/whole-package-cache-v2.md`
- Implementation: `internal/packagecache/`, `internal/workspacecompiler/`

## Contract

V3 preserves every v2 compiler, language, MIR, target, ABI, toolchain, backend, runtime,
standard-library, package-instance, source, dependency, profile, and feature input. It additionally
requires the current package's complete public-interface v5 hash immediately after the source
digest.

The current-interface input closes a source-independent invalidation gap: typed HIR may acquire an
experimental state layout without changing acquired source bytes. The owning package now misses on
that layout change, while every dependent misses through its existing sorted
`PackageCoordinateId + public-interface` entry.

The canonical encoding is `fairyc.package-cache-key/v3`; its only text form is:

```text
fycache3-sha256-<64 lowercase hexadecimal digits>
```

V1 and v2 keys are incompatible misses. Persistent records are disposable and require no migration.

## Boundary

Current-interface participation deliberately over-invalidates an owning package when any other
public fact changes, even though the source digest ordinarily changes too. Correctness and generated
HIR coverage take precedence over recovering that redundant hit. This decision adds no persistent
state backend or stable MIR artifact.

Source-independent HIR state transaction bodies are not encoded by v3: they are neither acquired
source bytes nor public-interface layout facts. ADR 0347 therefore keeps their lowering process-local
and outside cached compilation. A later artifact/cache revision must key their exact body identity
before a cached compiler path may consume them.
