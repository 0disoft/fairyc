# Exact retained source-excerpt extraction

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Policy prerequisite: `docs/adr/0195-attributed-text-boundary-v0.md`
- File-identity prerequisite: `docs/adr/0184-source-file-integrity-identities-v1.md`
- Query prerequisite: `docs/adr/0188-bounded-current-fymap-queries.md`

## Context

ADR 0195 fixed the attributed-text trust, identity, integrity, and budget boundary but deliberately
did not prove that exact text could cross it. The first executable slice needs source bytes without
turning stored paths into read authority, changing `.fymap` or Context Pack v1, or pretending that
ordinary comments are declaration documentation.

## Decision

1. Add an internal immutable `fairyc.attributed-text/v0` record with closed kind, origin, format,
   authority, identity, span, UTF-8/control, checksum, and explicit budget validation.
2. Add an exact query handoff that resolves visible or full SymbolId to a detached package context,
   declaration node, and selected file-identity record. It supplies no source text, path-opening
   authority, or name search.
3. Resolve source only from the workspace loader's retained package closure. Clone every retained
   package source, recompute the aggregate package digest, then recompute the selected file and line-
   map identities before exposing a detached byte slice. Never reopen the canonical relative path.
4. Admit at most one `workspace-authored` declaration `source-excerpt` for this slice. Match the
   exact package instance, coordinate, source digest, package UID, file path, file kind, file digest,
   line-map identity, both SymbolId forms, and half-open compiler span.
5. Preserve exact bytes, including CRLF and non-ASCII text. Reject an invalid span or inconsistent
   identity. Return an explicit atomic omission for invalid UTF-8, disallowed controls, excluded
   origin, or a fragment-byte limit instead of rewriting or truncating text.
6. Keep `.fymap` v1 and `fairyc.context-pack/v1` unchanged. Define no wire encoding, decoder,
   command, default budget, persistent store, documentation attachment, dependency acquisition,
   generated-source lineage, model call, or permission behavior.

## Evidence

- Model tests freeze the domain-separated checksum, closed enum and format sets, symbol-pair and
  package/file/line-map validation, half-open span length, UTF-8 and control policy, internal
  mutation rejection, printable prompt-like text classification, and guard-bound budgets.
- Real-workspace tests compile a BOM-prefixed CRLF source containing non-ASCII and printable prompt-
  like text, then require visible/full SymbolId equivalence, exact relative path and identities,
  detached validated records, and no host path.
- Negative integration tests overwrite the source after loading without changing retained output,
  reject mutated retained bytes and a foreign workspace/query pairing, and report atomic size,
  invalid-UTF-8, and bidirectional-control omissions.
- The full Go suite and Windows host smoke remain the default-path regression gates; no attributed-
  text encoder or command is reachable from the compiler CLI.

## Consequences

Internal consumers can now request one exact local declaration excerpt without a time-of-check/time-
of-use file read. This is implementation evidence for the first ADR 0195 gate, not a public output
contract. A canonical encoding and any CLI must be decided and tested independently, while
declaration documentation remains blocked on frontend-owned attachment semantics.

## Rejected alternatives

- Reopen the stored relative path: makes post-validation disk state authoritative and reintroduces
  containment and race problems.
- Accept caller bytes beside a SymbolId: allows unrelated text to impersonate compiler-selected
  source.
- Put the excerpt into Context Pack v1: changes an existing canonical format under the same version.
- Infer documentation from trivia: the current frontend has no exact attachment rule.
