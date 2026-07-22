# C-friendly declaration documentation attachment

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Lexer source: `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- Attributed-text policy: `docs/atlas/attributed-text-v0.md`
- Output follow-up: `docs/adr/0200-exact-declaration-documentation-attributed-text-command.md`

## Context

Attributed-text v0 reserves `declaration-documentation`, but the frontend previously classified all
comments and whitespace as indistinguishable trivia. Treating every nearby comment as documentation
would leak implementation notes and prompt-like text under a stronger label. Adding a grammar token
would instead make comments affect parsing and the existing experimental parse token-kind contract.
The attachment relation must therefore be source-exact, conservative, and owned before Atlas text
extraction is enabled.

## Decision

1. Recognize a line comment whose first three bytes are `///` as documentation trivia. It remains
   lexer `Trivia`, is skipped by the parser exactly like other comments, and changes no executable
   syntax or token-kind output.
2. Admit only physical-line-leading `///`: from the beginning of that physical line to the first
   slash, only ASCII space or tab may occur. `///` after code on the same line is not attachable.
3. Attach one or more consecutive documentation lines only to the next top-level `struct`,
   `resource`, `enum`, ordinary function, owner-qualified function, or foreign function. The final
   documentation line and the declaration's first attribute, `pub`, `unsafe`, or keyword must be
   separated by exactly one LF or CRLF followed only by optional ASCII indentation.
4. An empty physical line, ordinary `//` comment, block comment including `/**`, code token, lone CR,
   or another construct breaks attachment. Module declarations, imports, constants, fields,
   variants, parameters, local bindings, cleanup blocks, and statements receive no documentation
   attachment in this revision.
5. Store one half-open byte span from the first slash of the earliest attached line through the last
   byte of the latest attached comment. Exclude indentation before the first marker and the final
   line separator; retain intermediate line endings and indentation because the span is continuous.
6. Preserve the trivia subtype in the lossless token stream, the attachment span in the CST and
   source AST, and the same source-qualified span in typed HIR. Store no documentation body, parsed
   Markdown, tag model, rendered text, or authority.
7. Keep SymbolId, semantic behavior, MIR, generated C, `.fymap` v1, Context Pack v1, and the current
   `atlas-text source` bytes unchanged. Documentation extraction, encoding, CLI selection, and
   attachment to additional declaration kinds require later decisions and tests.

## Evidence

- Lexer tests distinguish BOM, whitespace, ordinary line, documentation line, and block trivia while
  reconstructing the original bytes exactly.
- CST tests require exact LF and CRLF multi-line spans across attributes and visibility, and reject
  blank-line, ordinary-comment, block-comment, same-line-after-code, and nested-field attachment.
- AST and semantic tests require all five currently typed top-level declaration families to retain
  the exact marker-bearing bytes through typed HIR while compiler-created drop glue remains
  undocumented.
- The complete compiler suite proves unchanged semantic, MIR, Atlas, cache, backend, and CLI
  behavior for sources that do not consume documentation metadata.

## Consequences

Fairy C now has a small C/C++-familiar documentation spelling and an executable attachment fact, but
not yet a documentation output feature. Ordinary comments stay ordinary. A later Atlas extractor can
consume the HIR span instead of guessing from proximity or relabelling arbitrary trivia.

## Rejected alternatives

- `/** ... */`: nested block-comment rules make attachment and editing boundaries more complex in
  the first slice.
- `//!`, `///<`, or Rust-style module documentation: adds roles not required by current Atlas
  declaration lookup.
- Strip `///`, indentation, or Markdown: destroys exact source attribution and invents rendering
  semantics.
- Put the body into HIR or `.fymap`: mixes untrusted prose into compiler facts and changes accepted
  transport bytes.
- Attach nearest preceding comment: silently promotes implementation comments and crosses blank
  lines or unrelated trivia.
