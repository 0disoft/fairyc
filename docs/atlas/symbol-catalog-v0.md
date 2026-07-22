# Atlas Symbol Catalog v0

- Status: Historical output-only catalog; superseded by v1
- Query prerequisite: `docs/adr/0192-exact-current-symbol-catalog-query.md`
- Historical implementation: retired; `internal/symbolcatalog/` now emits v1
- CLI decision: `docs/adr/0193-experimental-symbol-catalog-command.md`
- Successor: `docs/atlas/symbol-catalog-v1.md`

Current writers emit Symbol Catalog v1 under ADR 0201. This page records the previously emitted v0
shape; no current decoder accepts it and no cross-version cursor migration is promised.

## Envelope

`fairyc.symbol-catalog/v0` is one compact UTF-8 JSON value with encoding
`fairyc.symbol-catalog-json/v0`. It contains the exact-current `SymbolCatalogResult`: compatibility
context, exact filters and cursor, total filtered count, selected compiler-fact identity records,
continuation, completeness, budgets, usage, and explicit omissions.

Every record includes visible and full SymbolId, module, declaration kind, name, and
public/unsafe/foreign flags. It contains no source body, documentation, absolute host path, fuzzy
score, inferred rename, model text, permission, or executable instruction.

## Integrity and bounds

`fairyc.symbol-catalog-checksum/v0-sha256` hashes a domain/version prefix followed by
length-prefixed catalog version, encoding, and canonical logical result bytes. Its text form is
`fyscat0-sha256-` plus 64 lowercase hexadecimal digits. This detects accidental corruption and
binds the exact output shape; it is not authentication.

The query requires positive symbol and selection-byte limits. Encoding separately requires a
positive final-output-byte limit. Either boundary fails atomically. Successful output has no
trailing newline or second JSON value.

## Not enabled

- no decoder or untrusted catalog input;
- no default size, path, file, map-store, cache, or automatic publication;
- no fuzzy/exact-name search, dependency catalog join, or workspace reverse index;
- no source/documentation text, stable anchors, LSP, SCIP, MCP, or model call; and
- no stable CLI or cross-version cursor promise.
