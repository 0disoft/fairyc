# Canonical attributed-text JSON v0

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Policy prerequisite: `docs/adr/0195-attributed-text-boundary-v0.md`
- Extraction prerequisite: `docs/adr/0196-exact-retained-source-excerpt-extraction.md`

## Context

ADR 0196 can produce one exact local source excerpt or an explicit omission, but an in-memory Go
value is not a portable output contract. Encoding it by populating Context Pack v1 would violate
that pack's required empty text arrays. Accepting a caller-assembled extraction result would also
let arbitrary text or forged omissions impersonate compiler-selected output.

## Decision

1. Define document version `fairyc.attributed-text/v0`, encoding
   `fairyc.attributed-text-json/v0`, and document checksum
   `fairyc.attributed-text-checksum/v0-sha256` with text prefix `fytextdoc0-sha256-`.
2. Expose only an output builder that accepts the retained `Workspace`, exact-current `QueryIndex`,
   an explicit request, and a positive final-output byte limit. The builder performs extraction
   itself and accepts no caller-authored `Result`, record, source bytes, omission, path, or checksum.
3. Require every request to provide one exact visible or full SymbolId, explicit `kinds`, explicit
   `origins`, and positive count, per-fragment, and aggregate text budgets. The current producer
   supports exactly `source-excerpt` plus `workspace-authored`; it infers no selection defaults.
4. Emit one compact UTF-8 JSON value. The top-level order is `text_version`, `encoding`,
   `checksum_algorithm`, `checksum`, `text`. The logical `text` order is `authority`, `subject`,
   `selection`, `budget`, `usage`, `records`, `omissions`. Empty arrays are `[]`; there is no
   trailing newline or second value.
5. Fix document authority to `untrusted-text`. Bind the exact subject to both SymbolId forms,
   package UID/instance/coordinate/source identity, canonical relative file and kind, file and line-
   map identities, and half-open byte span even when the fragment is omitted.
6. Preserve every admitted logical record in full, including its independent fragment checksum.
   Require usage to rederive from records and require exactly one record or one atomic omission in
   this source-only producer. An excluded SDK subject reports actual origin `compiler-bundled`; it
   never relabels the source as the requested `workspace-authored` origin.
7. Compute the document checksum over ASCII `fairyc.attributed-text-checksum`, bytes `00 00`, then
   unsigned 64-bit big-endian length-prefixed document version, encoding, and canonical logical
   `text` JSON. The checksum excludes the envelope checksum field and proves integrity, not
   authentication, review, safety, or authority.
8. Apply the caller's final byte limit to the complete canonical envelope atomically. Selection
   budgets count exact source bytes and remain distinct from JSON bytes or model tokens.
9. Keep `.fymap` v1 and Context Pack v1 byte contracts unchanged. Define no decoder, CLI, default
   limit, persistent text store, composite pack, documentation attachment, dependency acquisition,
   generated-text lineage, model call, token estimate, or permission behavior.

## Evidence

- Output tests require repeated byte identity, visible/full selector identity, identical output
  under different absolute workspace roots, compact field order, no host path or trailing newline,
  exact CRLF/non-ASCII/prompt-like text, independently recomputed document checksum, and a frozen
  checksum vector.
- Bound tests require zero-limit rejection and exact one-byte-too-small atomic failure.
- Omission tests require fragment-limit and bidirectional-control exclusions to contain no unsafe
  text, and a real compiler-bundled SDK partition to retain actual `compiler-bundled` origin.
- Mutation tests corrupt subject, selection, usage, record text, and result cardinality and require
  document construction to fail closed.

## Consequences

Internal callers have a deterministic, separately versioned attributed-text artifact without
changing compiler-fact formats. ADR 0198 now exposes that artifact through one experimental,
default-free source command; no decoder exists. Any additional public command must separately fix
selector flow, output policy, exit codes, and bounds; any decoder must strictly reject alternate
JSON and revalidate all identities, checksums, budgets, and canonical bytes.

## Rejected alternatives

- Fill Context Pack v1 text arrays: silently changes an accepted canonical schema.
- Encode a caller-provided result: permits provenance and omission forgery.
- Store only a fragment checksum link: leaves an omitted result without exact subject and budget
  evidence and requires a separate storage contract that does not exist.
- Filter prompt-like printable text: destroys exact attribution without creating a security
  boundary.
