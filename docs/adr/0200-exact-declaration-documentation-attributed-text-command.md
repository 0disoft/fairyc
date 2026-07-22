# Exact declaration-documentation attributed-text command

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Attachment prerequisite: `docs/adr/0199-c-friendly-declaration-documentation-attachment.md`
- Attributed-text policy: `docs/atlas/attributed-text-v0.md`
- CLI contract: `docs/cli/command-contract.md`

## Context

ADR 0199 carries an exact frontend-owned `///` attachment span through typed HIR, but no Atlas
consumer can use it. Re-inferring documentation from retained source would duplicate parser policy;
putting the span into `.fymap` v1 would change an accepted body-free transport; and accepting a
caller-provided span would let unrelated comments impersonate compiler-attached documentation.

## Decision

1. Add an immutable in-memory `SemanticSnapshot` sidecar that binds one exact declaration key and
   declaration span to its frontend-attached documentation span. It contains no text and is excluded
   from declaration records, SymbolId derivation, logical map planning, `.fymap` v1, and Context
   Pack v1.
2. Add an attributed-text documentation extractor that accepts the retained `Workspace`, exact-
   current `QueryIndex`, and the snapshot produced by the same compilation. Require the snapshot to
   contain exactly one matching declaration and, when documented, one matching preceding span in
   the same package instance and source file. Never reparse source or infer a nearby comment.
3. Resolve bytes only through `Workspace.ResolveExactSource`, revalidating package source, selected
   file, line-map, package, SymbolId, declaration, and sidecar identities before slicing. Preserve
   the exact marker-bearing bytes without trimming, newline conversion, Markdown parsing, or marker
   removal.
4. Reuse the already reserved logical kind `declaration-documentation`, format
   `fairyc-declaration-doc-utf8`, authority `untrusted-text`, fragment checksum, UTF-8/control rules,
   and explicit count, fragment, aggregate, and final-output byte limits.
5. When a selected declaration has no attached documentation, return one successful checksummed
   `kind-unavailable` omission with zero text bytes. A text-safety or fragment-limit omission binds
   the exact attached span and byte count. Never substitute the declaration body or an ordinary
   comment.
6. Add only `fairyc experimental atlas-text documentation`. Require the same explicit
   `--package-uid`, canonical visible/full `--symbol`, `--origin workspace-authored`,
   `--max-fragments`, `--max-fragment-bytes`, `--max-text-bytes`, `--max-output-bytes`, and optional
   `--locked` policy as `atlas-text source`. The action fixes the selected kind; there is no
   redundant `--kind` or numeric default.
7. Emit the existing canonical `fairyc.attributed-text-json/v0` envelope because that closed model
   already reserved the documentation kind and format. Add no fields and preserve existing
   `atlas-text source` bytes, checksum vector, failure codes, and no-newline stdout contract.
8. Keep decoding, persistent text storage, SDK/dependency/generated text admission, composite packs,
   model calls, token estimates, rendering, authentication, and permission authority unimplemented.

## Evidence

- Snapshot tests require all supported declaration families to retain detached stable sidecar spans,
  reject overlapping spans, and prevent compiler-private drop glue from acquiring documentation.
- `.fymap` tests build the same exact workspace with and without only the in-memory sidecar and
  require byte-identical v1 plans and encodings.
- Attributed-text tests require exact CRLF/non-ASCII/prompt-like documentation, visible/full and
  independent-root byte identity, retained-byte behavior after a disk edit, stale workspace and
  mismatched snapshot rejection, canonical checksums, exact final bounds, and no declaration body.
- Negative tests require `kind-unavailable`, fragment-limit, invalid-UTF-8, and bidirectional-control
  omissions to remain atomic and source-attributed.
- CLI tests require visible/full/locked byte identity, exact marker-bearing output, successful
  undocumented and size omissions, no host paths, and the existing invocation/output failure
  classes.

## Consequences

Users and tooling can now fetch the exact `///` block that the current compiler attached to one
exact SymbolId. The stored map remains small and body-free; therefore a documentation request still
performs a current workspace compilation rather than reading text from a persisted map alone.

## Rejected alternatives

- Store documentation spans or bodies in `.fymap` v1: changes an accepted transport and increases
  retained rewrite cost.
- Scan backward from the declaration inside Atlas: duplicates frontend rules and can promote
  ordinary comments after future syntax changes.
- Return an empty record when documentation is absent: confuses unavailable content with an exact
  zero-byte document, which the logical model forbids.
- Strip `///` or parse Markdown: destroys exact attribution and invents rendering semantics.
