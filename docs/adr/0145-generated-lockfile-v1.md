# ADR 0145: Use canonical JSON for generated `fairy.lock` v1

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Workspace source: `docs/package/workspace-loader-v1.md`
- Lockfile source: `docs/package/lockfile-v1.md`
- Implementation: `internal/lockfile/`

## Context

The local workspace loader now produces exact source digests, feature-distinct package identities,
and a validated root-reachable graph. Recomputing that graph is safe, but a locked build needs one
reviewable record that detects stale paths, sources, features, versions, and edges without exposing
internal node handles or absolute host paths.

`fairy.toml` is for humans. Reusing TOML for generated graph data buys little and creates more
canonicalization work around table order and quoting. A compact binary format would be harder to
review and would falsely resemble a stable package artifact. Canonical JSON uses the Go standard
library, has closed decoding, and can be emitted byte-for-byte deterministically.

## Decision

1. The generated file is `fairy.lock`, schema `fairyc.lock/v1`, encoded as UTF-8 canonical JSON:
   fixed struct field order, two-space indentation, sorted arrays, and one final newline.
2. The header records exact package-instance and source-digest encoding identities plus the root
   language revision and root PackageInstanceId.
3. Every root-reachable instance records full PackageInstanceId, role, PackageUid, source kind,
   exact source identity/digest, exact version, and explicit feature array. Local records retain a
   canonical workspace path; SDK records instead retain exact SDK identity and runtime ABI.
4. Every direct edge records owner ID, binding, and target ID. Package records sort by ID; edges
   sort by owner, binding, and target. Build-local graph node names are not serialized.
5. Readers reject oversized input, duplicate or unknown JSON keys, trailing values, null features,
   noncanonical arrays, unknown IDs, duplicate bindings, graph cycles, malformed identities, and
   any PackageInstanceId or source-identity contradiction.
6. Matching a current workspace regenerates the canonical document and compares canonical bytes.
   Any source, version, path, feature, language, role, or edge difference is stale.
7. Writers accept only already-canonical valid bytes, write a same-directory temporary file, flush
   and close it, then atomically rename it over `fairy.lock`. Failures remove the temporary file.
8. The format performs no network access, solving, filesystem SDK discovery, or cache write. ADR
   0147's compiler-bundled SDK uses the same package record with a closed source-origin variant.

## Consequences

- Generated diffs are stable and readable while remaining explicitly non-authoritative input.
- The lock records local relative locations for refresh but never makes an absolute checkout path
  part of semantic identity.
- The SDK record has no local path and fails closed on SDK identity, runtime ABI, source digest,
  source identity, package UID, version, role, or feature drift.
- JSON field order is part of canonical output v1 even though strict readers identify fields by key.
- Ordinary update versus CLI `--locked` policy remains a CLI integration decision; the library
  already exposes generation, strict parsing, workspace matching, and atomic replacement.

## Verification

- unit tests round-trip a feature-distinct real workspace, revalidate every identity, reject
  duplicate/unknown JSON fields and tuple/source contradictions, detect staleness, require array
  spellings, and replace an existing lock atomically on Windows;
- all Go format, test, build, and vet checks remain the configured validation boundary.
