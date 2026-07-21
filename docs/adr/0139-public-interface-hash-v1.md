# ADR 0139: Hash the source-visible package interface independently from implementation

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/public-interface-hash-v1.md`
- Identity source: `docs/package/package-instance-id-v1.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Context

Whole-package caching needs to know whether a dependency's source-visible contract changed. Hashing
source files or complete HIR would invalidate every dependent for private helpers, ordinary body
edits, local names, and compiler-generated cleanup glue. Hashing only declaration text would retain
aliases, formatting, and disposable build-local `$pN` package numbers while missing resolved type
identity.

The first package identity now provides a stable full digest, and typed HIR retains source `pub`
state. That makes a semantic interface projection possible without serializing HIR or deciding a
manifest, lockfile, cache-record, or native ABI format.

## Decision

1. Public-interface hash v1 projects one fully analyzed typed package into a deterministic logical
   descriptor, encodes it with a versioned domain-separated binary format, and hashes the complete
   bytes with SHA-256.
2. The canonical text is `fyif1-sha256-` followed by 64 lowercase hexadecimal digits.
3. Module paths participate and sort lexically. Public top-level items sort by `(kind, name)`, so
   declaration and source-file order do not invalidate dependents. Field and enum-variant order is
   retained because it is semantic.
4. The descriptor includes:
   - public structs: name, `@repr(C)` state, ordered field names and resolved types;
   - public enums: name, ordered variants, and ordered payload field names and resolved types;
   - public resources: name and canonical cleanup-effect set;
   - public functions: owner-qualified name, optional owner type, unsafe state, effects, ordered
     parameter types, and result type;
   - public foreign functions: source name, ABI, external link name, effects, ordered parameter
     types, and result type.
5. It excludes source spans, comments, formatting, imports and aliases, parameter names, callee-local
   parameter mutability, function bodies, locals, private items, resource private fields, cleanup
   bodies, compiler symbols, build-local package numbers, MIR, layouts, and generated code.
6. A type owned by the hashed package uses `self::module::Type`. An external nominal type uses the
   complete dependency `PackageInstanceId`. Therefore renumbering `$pN` is inert, while changing the
   nominal dependency instance invalidates the interface.
7. A public signature or public struct/enum shape that exposes a private local type fails extraction.
   It never hashes a private type under an unstable spelling.
8. Resource representation remains private at the source interface. A later compiled-module or
   native-ABI cache may require a separate implementation/layout hash; it must not silently widen
   this source-interface contract.
9. Ordinary public bodies are excluded because Stage 0 has no public inline-body contract. If
   inlining across packages is later enabled, requested inline bodies receive a separate versioned
   hash and invalidation class.
10. A cache key must combine this hash with the applicable compiler/language/MIR/backend/target/
    runtime/standard-library compatibility facts. This hash alone never authorizes cache reuse.

## Consequences

- Private implementation and ordinary body edits can preserve dependent-package reuse.
- Public signature, source-layout, effects, and external-link changes invalidate dependents.
- The hash stays stable across source formatting, declaration order, and build-local package IDs.
- Dependency instance changes remain conservative invalidations even if that dependency happens to
  expose an equal interface; cross-instance nominal type reuse is not inferred.
- Public manifest syntax, lockfile records, cache storage, eviction limits, and module-interface
  serialization remain separate Phase 3 work.

## Verification

- eight versioned `.fy` cases freeze a baseline digest and same/different invalidation relations;
- source integration proves private and public ordinary bodies do not alter the hash;
- unit tests prove public signature, field order, effects, and foreign link changes do alter it;
- unit tests prove build-local package renumbering is inert and dependency identity changes are not;
- private-type leakage and noncanonical effect ordering fail closed.
