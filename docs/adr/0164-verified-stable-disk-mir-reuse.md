# ADR 0164: Reuse stable package MIR only after complete verified disk hits

- Status: Accepted internal integration
- Date: 2026-07-22
- Implementation: `internal/packageartifact/stable_cache.go`, `internal/workspacecompiler/session.go`
- Storage prerequisite: `docs/adr/0163-bounded-immutable-disk-cache-store.md`

## Context

The stable package codec and bounded disk store existed independently. Treating a raw disk-record
hit as executable MIR would still be wrong: the record could carry the wrong public interface, a
valid artifact for another package instance, an incompatible compiler build, malformed stable MIR,
or a package set whose duplicated workspace contexts came from different compilations.

The compiler also has no stable release version source yet. Guessing one from a branch, moving tag,
or executable filename would let incompatible lowering implementations share a key.

## Decision

Add `packageartifact.StableCache` and an opt-in `workspacecompiler.PersistentConfiguration`.
Persistent sessions require a caller-supplied exact compiler-build identity, disk root, entry bound,
retained-byte bound, record-byte bound, abandoned-temp age, and artifact-byte bound. A Session uses
either the process-local memory cache or the persistent stable cache; combining both is rejected
until an explicit L1/L2 policy exists.

The stable cache accepts process-local verified package artifacts only at publication time. It
projects and canonically encodes stable MIR before `DiskStore` publication. A read is a verified hit
only when the complete cache key finds exactly one valid disk generation, the public-interface hash
matches, the record contains exactly one MIR payload, the stable codec accepts canonical bytes, and
the embedded `PackageInstanceId` matches the current package. A rejected adapter payload removes
only its exact immutable record generation.

The workspace coordinator skips lowering only when every function-owning package is a stable hit.
It then allocates fresh package, function, foreign, and local identities, resolves all imports,
rebuilds the stable type context, and reruns `mir.Verify`. Any miss, rejection, or mixed-context
reassembly failure follows ordinary lowering and refreshes complete current artifacts. Oversized
cache admission remains a skipped optimization rather than a compilation failure.

The persistent cache key prefixes the caller build identity with the exact stable-package and disk-
record format versions. Language, MIR, target, ABI, runtime, standard-library, package, source,
dependency-interface, feature, backend, and toolchain dimensions remain in package-cache key v1.

## Evidence

Unit tests write through one stable-cache adapter and read through another, require deep-copy stable
artifacts, and quarantine wrong-interface, wrong-package, and malformed stable payloads. Workspace
tests compile the checked-in two-package shape with one Session, construct a completely independent
Session over the same directory, require two stable hits and zero lowering calls, rebind and verify
the program, and execute it to `42`. A separate-process test then runs a writer test process and a
fresh reader test process over one workspace and cache directory; the reader reports two hits, zero
lowering, and result `42`. Changing only the exact compiler-build identity produces two misses and
ordinary lowering.

## Consequences and limits

- Cross-Session stable disk reuse is now executable through an internal explicit policy surface.
- ADR 0166 later adds one explicit experimental CLI command and ADR 0165 supplies its executable
  identity. No ordinary-command cache, location default, L1 layer, daemon, background cleanup, or
  speed claim is added.
- Separate writer and reader OS processes exercise cross-process Session reuse, and a separately
  built `fairyc.exe` now exercises the experimental command. ADR 0168 owns the publication-crash
  evidence, and ADR 0169 owns opened-root containment plus the current platform publication primitive.
- Each package still duplicates the complete workspace type context. Partial hits therefore lower
  normally; function-level reuse and package-scoped contexts remain measurement-gated.
- The cache is disposable derived state. Deletion or rejection can only cause ordinary compilation.
