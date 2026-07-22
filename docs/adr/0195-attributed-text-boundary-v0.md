# Atlas attributed text boundary v0

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Policy: `docs/atlas/attributed-text-v0.md`
- File-identity prerequisite: `docs/adr/0184-source-file-integrity-identities-v1.md`
- Fact-pack prerequisite: `docs/adr/0189-canonical-compiler-fact-context-pack-v1.md`

## Context

Atlas can now discover exact SymbolIds and emit bounded overview, show, and around compiler-fact
packs. The next tempting step is to fill their empty documentation and source-excerpt arrays. That
would silently change Context Pack v1, mix facts with untrusted prose, duplicate source into
`.fymap`, and create a prompt-injection path before source attribution, staleness, control-character,
and fragment-budget behavior is fixed.

The current lexer also treats comments as undifferentiated trivia. Calling every nearby comment
declaration documentation would invent a language rule the grammar and frontend do not own.

## Decision

1. Define logical policy `fairyc.attributed-text/v0` as a separate, output-only companion to
   compiler facts. Define no encoding or CLI in this decision.
2. Keep `.fymap` v1 body-free and keep Context Pack v1 text and warning arrays empty. Any encoded
   attributed text or composite pack receives a new explicit compatibility identifier.
3. Fix every text record to authority `untrusted-text`. Record origin is descriptive and cannot
   grant host or model permissions, alter compiler facts or completeness, or act as an instruction.
4. Admit only exact declaration source excerpts and future frontend-attached declaration
   documentation. Exclude arbitrary comments, Markdown files, logs, URLs, summaries, environment
   data, and caller strings.
5. Permit only explicitly selected origins and budgets. The first implementation is restricted to
   `workspace-authored` source excerpts from the already validated immutable workspace closure.
   It performs no file reopen, discovery, dependency fetch, registry access, or network request.
6. Bind each fragment to both SymbolId forms, exact package identity, canonical relative file,
   content and line-map identities, half-open byte span, format, exact UTF-8 bytes, and a
   domain-separated full SHA-256 fragment checksum.
7. Recompute current package/file identities before slicing. Any stale, mismatched, out-of-range,
   non-scalar-boundary, invalid-UTF-8, or unsafe-control fragment is rejected or explicitly omitted;
   never guess, normalize, sanitize, or return a same-length replacement.
8. Require explicit count, per-fragment, and aggregate byte budgets. Fragments are atomic and are
   omitted whole when they do not fit. Return deterministic usage and omission evidence rather than
   implying an empty or partial result is complete.
9. Do not filter printable prompt-like prose as a security mechanism. Preserve it only as labelled
   untrusted data and require consumers to keep authorization and prompt roles outside Atlas.
10. Require local model/checksum tests, exact-current pipeline tests, stale and malicious-text
    negative cases, budget boundaries, determinism, and proof that the unrequested compiler/map/
    pack/build/cache paths remain byte-identical before exposing an encoder or command.

## Consequences

- The policy closed the security and compatibility decisions without itself claiming text import
  worked; ADR 0196 now supplies the narrow internal source-excerpt implementation evidence.
- The first executable slice remains an internal source-excerpt model instead of a broad parser,
  map-format, command, or model-integration change.
- Documentation comments need a separate grammar/frontend attachment contract before admission.
- Dependency and generated text remain unavailable even when their bytes are signed or bundled.
- Checksums detect mutation but do not prove authorship, review, truth, or safety.

## Rejected alternatives

- Populate Context Pack v1's existing arrays: violates its accepted empty-array compatibility
  contract.
- Store source bodies in `.fymap`: increases rewrite and retained size and duplicates authoritative
  current source.
- Reopen files from stored paths: creates a time-of-check/time-of-use and containment boundary.
- Strip suspicious phrases: natural-language filtering is bypassable and destroys exact attribution.
- Treat workspace prose as trusted instructions: lets source code change agent authority.
- Infer documentation from nearby trivia: invents attachment semantics and breaks on formatting or
  refactoring.
