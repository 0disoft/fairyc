# Canonical compiler-fact Context Pack v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Pack contract: `docs/atlas/context-pack-v1.md`
- Query prerequisite: `docs/adr/0188-bounded-current-fymap-queries.md`

## Context

Bounded query structs are not yet a portable artifact. A consumer needs one deterministic envelope
that identifies the exact package and map compatibility vector, records what was requested and
omitted, and can be size-capped before it crosses into an IDE or LLM-facing boundary.

Opening source excerpts or documentation in the same step would mix compiler truth with untrusted
free text and create a prompt-injection path before attribution, stale-source, control-character,
and fragment-budget rules exist. Accepting caller-built query result structs would also let a caller
mutate facts and ask the encoder to label them compiler-observed.

## Decision

1. Add `internal/contextpack` as an output-only consumer of `atlasmap.QueryIndex`. Its public
   functions run `overview`, `show`, or `around` directly instead of accepting arbitrary result
   structs.
2. Define logical version `fairyc.context-pack/v1`, canonical compact JSON encoding
   `fairyc.context-pack-json/v1`, and checksum algorithm
   `fairyc.context-pack-checksum/v1-sha256`.
3. Retain exact map version; package UID, instance, coordinate, and source; compiler build, language,
   public-interface hash, generation and map omissions; query kind, subject, direction and filter;
   governing budget and actual usage; completeness, typed facts, relations, omissions, and explicit
   empty text sections.
4. Require exactly one typed overview, show, or around fact payload. Keep relations structured and
   separate from prose.
5. Apply the query budget first, then require a separate positive maximum for final canonical pack
   bytes. Do not call either bound a model-token count.
6. Domain-separate and length-prefix the pack version, encoding, and canonical logical bytes before
   SHA-256. Render the full digest as `fyctx1-sha256-` plus lowercase hexadecimal. Treat it as
   integrity, not authentication.
7. Emit empty `documentation`, `source_excerpts`, and `warnings` arrays. Do not add a text input,
   decoder, dependency discovery, process, filesystem, network, or model boundary in v1.
8. Keep pack failure downstream from compilation. It cannot alter diagnostics, MIR, emitted C,
   native artifacts, map publication, or execution.

## Consequences

- Consumers get one byte-identical, self-identifying compiler-fact artifact with explicit partiality
  and no source-body leakage.
- A future decoder is an untrusted-input boundary and must add strict fields, checksum comparison,
  semantic validation, and canonical re-encoding before returning facts. This ADR does not pretend
  an encoder test proves that reader.
- Attributed source or documentation requires a separate decision and cannot begin populating v1's
  empty text arrays.
- No command, default location, dependency join, impact analysis, or model integration is enabled.

## Evidence

Tests build a real three-function Fairy C workspace and exact-current query index; require repeated
byte identity for overview packs; independently recompute the checksum; verify overview/show/around
payload exclusivity, local traversal, package and compatibility identity, query-budget adherence,
empty text and warning sections, absence of absolute paths and source bodies, separate final-byte
rejection, exact query-error propagation, and strict checksum text parsing.
