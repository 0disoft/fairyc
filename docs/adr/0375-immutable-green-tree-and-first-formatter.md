# ADR 0375: Immutable green tree and first full-file formatter

## Status

Accepted as the first editor-tooling slice. Incremental parsing, LSP, SCIP, semantic rename, structural edits, and compiler authority from editor state remain deferred.

## Decision

`internal/formatter` builds a detached immutable green representation from the existing lossless CST. Green identity is `fairyc-green-tree/v0` SHA-256 over the exact source identity, parser-shape identity, and complete token/trivia values. Nodes and tokens are copied out of parser arenas; callers receive defensive copies of mutable slices.

The first formatter is deliberately conservative. It changes only lexer-owned whitespace trivia: CRLF becomes LF, horizontal whitespace immediately before a newline is removed, and a non-empty valid source gains one final LF when absent. Comments, identifiers, literals, symbols, and every other significant token remain byte-for-byte unchanged.

Before formatted bytes are returned, the compiler reparses them and requires the exact same parser-shape digest and significant-token sequence. If either differs, formatting fails instead of publishing bytes.

Malformed source is never partially rewritten. The formatter returns the original bytes unchanged with `ErrMalformedSource`, while still returning detached green-tree facts for editor inspection.

## Identity

Exact green identity intentionally changes when trivia changes. Parser-shape identity intentionally ignores source spans, token ranges, and trivia so a whitespace-only formatting pass can prove structural equivalence. Source identity remains the SHA-256 of exact immutable input bytes.

These are editor-internal identities, not package, cache, ABI, or language-version identities.

## Performance boundary

The first slice reparses the complete file after formatting. A representative benchmark reports source bytes and full-format cost. Large-file tests exercise hundreds of functions under the normal source budget. No incremental reuse is implemented until edit traces show that full-file rebuilding is a material editor bottleneck.

## Anchor behavior

Formatting may change byte offsets whenever CRLF or trailing whitespace is removed. Existing source anchors therefore remain identities of the pre-format source snapshot; they are not silently transplanted to the formatted document. A later editor integration must explicitly remap anchors against old/new source snapshots.

## Security and correctness

The formatter never interprets runtime format strings, runs source code, resolves packages, reads host configuration, or writes files. It operates only on caller-provided bytes and existing bounded lexer/parser machinery. Parser diagnostics make mutation fail closed.
