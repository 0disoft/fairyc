# ADR 0162: Encode and rebind stable package MIR artifacts

- Status: Accepted
- Date: 2026-07-22
- Implementation: `internal/packageartifact/stable_artifact.go`
- Prerequisite: `docs/adr/0161-stable-type-context-artifact.md`

## Context

ADR 0161 established stable type context, function-body, graph-ID, type, call, and context-rebuild
primitives. Those pieces still had no package envelope. A cache record could not prove that every
external `FunctionID` was declared, that all packages shared one context, or that decoded bytes
alone could rebuild a closed verified program.

## Decision

Add exact-version `fairyc.stable-package-mir/v1` artifacts. Each artifact contains one exact
`PackageInstanceId`, the complete stable workspace context, sorted package-owned function bodies,
and the sorted exact set of imported `FunctionID` values. Canonical bounded JSON decoding rejects
unknown fields, trailing or noncanonical bytes, incompatible versions, inconsistent context IDs,
unsorted or duplicate bindings, unresolved calls, out-of-range foreign references, and imports
that are missing, owned, or unused.

Rebinding a complete artifact set allocates fresh internal and foreign symbols, rebuilds the type
context, restores every function into a fresh local namespace, resolves every import and drop link,
and reruns the ordinary closed-world MIR verifier. The stable context remains workspace-wide in
this version; duplicating it per package is accepted temporary overhead, not a storage target.

## Evidence

The checked-in package workspace is extracted and encoded package by package, decoded without its
process-local artifacts, assigned completely different package, function, foreign, and local
numbers, rebuilt, verified, and interpreted to `42`. Re-extraction and re-encoding after that fresh
rebind must reproduce every original package byte sequence exactly. Codec tests cover bounds,
unknown fields, noncanonical bytes, and import corruption.

## Consequences and limits

- Cross-process-stable verified MIR bytes now exist; the process-random codec is no longer the only
  truthful representation of package MIR.
- No file, cache directory, lock, index, cleanup rule, default budget, or CLI behavior is added.
- The package artifact duplicates the whole workspace context and therefore does not yet permit
  mixed-context package reuse.
- Persistent-store corruption, concurrency, atomic replacement, retention, and measurement policy
  remain separate gates under ADR 0146.
