# Whole-package cache key v4

- Status: Experimental executable contract
- Decision: `docs/adr/0349-contract-state-body-cache-key-v4.md`
- Fixture source: `spec/fixtures/package-cache/manifest-v8.json`
- Historical source: `docs/package/whole-package-cache-v3.md`
- Implementation: `internal/packagecache/`, `internal/workspacecompiler/`

## Contract

V4 preserves every v3 compiler, language, MIR, target, ABI, toolchain, backend, runtime,
standard-library, package-instance, source, current-interface, dependency, profile, and feature
input. It additionally requires the current package's exact
`fairyc.contract-state-package-body/v1` hash immediately after public-interface v5.

The workspace compiler computes body identities from completed typed HIR before lookup. Every
function-owning package needs a nonzero current body identity, including packages with no state
layout or transactions. This makes later source-independent body injection a deterministic owning
package miss instead of permitting stale reuse.

The canonical encoding is `fairyc.package-cache-key/v4`; its only text form is:

```text
fycache4-sha256-<64 lowercase hexadecimal digits>
```

V1 through v3 keys are incompatible misses. Persistent records are disposable and require no
migration.

## Dependency boundary

The current body hash is private implementation identity and appears only in the owning package
key. Dependencies continue to contribute sorted `PackageCoordinateId + public-interface v5` pairs.
A dependency-body change therefore misses that dependency while a caller with the same consumed
interface may remain reusable. Package artifact verification and stable call binding remain the
authority for safe reassembly.

## Closed gates

V4 does not serialize state transaction operations into stable MIR or package artifacts. It cannot
rebind a stable function owner or execute cached state transitions. ADR 0350 independently provides
a bounded capability-free operation payload and verifier re-entry, but no cache record or reassembly
path consumes it. Source syntax, package-artifact integration, native backend execution, and
production persistence remain separate admission gates.
