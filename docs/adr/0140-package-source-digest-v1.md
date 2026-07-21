# ADR 0140: Hash exact package source independently from its public interface

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/package-source-digest-v1.md`
- Interface source: `docs/package/public-interface-hash-v1.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Context

The public-interface hash deliberately ignores private declarations and ordinary bodies so a
dependent package can be reused when a dependency's source-visible contract is unchanged. The
package being compiled still needs an exact implementation key: comments and formatting affect
spans and debug provenance, while private and public bodies affect generated behavior.

Using a directory timestamp, filesystem enumeration order, absolute path, Git status, or archive
metadata would make the key host-dependent. Hashing only concatenated file bytes would lose path
and file boundaries.

## Decision

1. Package source digest v1 hashes the exact byte content and canonical source-root-relative path of
   every `.fy` file in one already validated package source graph.
2. Files sort by canonical forward-slash path before encoding. Caller enumeration order is inert.
3. Paths must be nonempty valid UTF-8, relative, forward-slash, clean, non-traversing, volume-free,
   unique, and end in `.fy`. The digest never normalizes an accepted path.
4. File bytes are opaque and exact. UTF-8 validation belongs to the parser. BOMs, comments,
   whitespace, line endings, and every implementation byte participate.
5. The versioned domain-separated encoding uses unsigned 64-bit network-order counts and lengths,
   then hashes the complete bytes with SHA-256.
6. The only text form is `fysrc1-sha256-` plus 64 lowercase hexadecimal digits.
7. The digest excludes package identity, features, dependencies, compiler version, language/MIR
   version, target, backend, toolchain, runtime, standard library, public-interface hash, and artifact
   bytes. A cache key combines those facts separately.
8. File discovery and containment happen before this API. The digest does not authorize filesystem
   reads, follow imports, resolve symlinks, acquire sources, or define manifest include/exclude rules.

## Consequences

- Any exact source or canonical path change invalidates the package's own implementation artifact.
- A dependency's source digest may change while its public-interface hash stays equal, preserving
  dependent-package reuse.
- Identical source graphs hash equally across Windows and Unix when the package layer supplies the
  same canonical paths and bytes.
- Public manifests, source discovery, archives, cache records, storage limits, and eviction remain
  separate Phase 3 decisions.

## Verification

- one unit vector freezes canonical bytes and the full SHA-256 result;
- versioned fixtures freeze a baseline and prove input order is inert;
- content, path, and line-ending changes invalidate;
- empty, duplicate, absolute, backslash, traversal, unclean, drive-like, wrong-extension, and
  invalid-UTF-8 paths reject;
- canonical text parsing rejects prefix, width, case, and hexadecimal drift.
