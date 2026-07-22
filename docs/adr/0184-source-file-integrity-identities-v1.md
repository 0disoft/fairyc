# Source file integrity identities v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Atlas source: `docs/atlas/fymap-v1.md`

## Context

The package-source digest can reject a stale package as a whole, but a `.fymap` file record and
source span need an independently checkable file identity. The existing semantic snapshot retained
only package aggregate identity and relative span paths. Exporting those spans now would create a
map that could not prove which current file bytes or byte-to-line geometry they describe.

## Decision

1. Add independently versioned exact content and line-map identities before implementing `.fymap`
   transport.
2. Hash domain-separated, length-bounded exact file bytes as `fairyc.source-file/v1`, rendered only
   as `fyfile1-sha256-` plus 64 lowercase hexadecimal digits.
3. Hash file byte length and all zero-based LF-derived line starts as
   `fairyc.source-line-map/v1`, rendered only as `fylm1-sha256-` plus 64 lowercase hexadecimal
   digits. Preserve CR as an ordinary byte and include an EOF line start after a final LF.
4. Compute both identities inside the validated workspace boundary from already-loaded immutable
   bytes. Do not reopen files, discover paths, or expose source text to Atlas.
5. Add an explicitly requested `SemanticFiles` projection with canonical detached metadata and
   extend `SemanticSnapshot` with path-free file records. Do not add per-file hashing to the
   ordinary `SemanticPackages`/cache path.
6. Require a nonempty exact file set for every package and a one-to-one match between file records
   and typed modules. Reject missing, duplicate, extra, noncanonical, zero-identity, or unknown-kind
   metadata.
7. Keep source/test classification explicit. Ordinary workspace loading still excludes `tests/`;
   the stable test overlay marks its files as `test` without changing production source identity.
8. Recompute the package-source digest from retained validated bytes when `SemanticFiles` is
   requested and reject post-validation in-memory mutation before publishing file identities.

## Consequences

- A future map validator can distinguish exact-content staleness from unchanged line geometry and
  can refuse stale spans before returning source excerpts.
- Snapshot preparation performs bounded hashing only when `SemanticFiles` is requested and stores
  only fixed digests. Ordinary check/cache compilation receives no new hashing work. File content is
  scanned with constant auxiliary line-map memory and without path discovery, I/O, subprocesses, or
  network access. Package-source digest computation now streams canonical bytes instead of
  duplicating source bodies and a whole encoded package buffer.
- Package aggregate digest remains authoritative for exact package identity; per-file identities do
  not replace `fysrc1-sha256` or `PackageInstanceId`.
- `.fymap` encoding, compiler-build metadata, interface hashes, checksum/publication, current-source
  revalidation, queries, and excerpts remain later boundaries.

## Evidence

Canonical vectors cover exact file bytes and line starts, content-only versus geometry changes,
CRLF, final LF, empty input, and strict text parsing. Workspace tests cover local/SDK source records
the isolated root test overlay, and post-validation source mutation rejection. Snapshot tests cover canonical order, deep detachment,
file/module bijection, missing/extra file rejection, path-free output, and stable build-number
renumbering.
