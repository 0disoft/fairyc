# Atlas Symbol Catalog v1

- Status: Implemented output-only exact catalog with optional exact-name filtering
- Query prerequisite: `docs/adr/0192-exact-current-symbol-catalog-query.md`
- Version decision: `docs/adr/0201-exact-name-symbol-catalog-v1.md`
- Implementation: `internal/symbolcatalog/`
- CLI: `fairyc experimental atlas-symbols list`

## Envelope

`fairyc.symbol-catalog/v1` is one compact UTF-8 JSON value with encoding
`fairyc.symbol-catalog-json/v1`. It contains the exact-current `SymbolCatalogResult`: compatibility
context, optional exact module and declaration-name filters, exact declaration-kind filter, full-ID
cursor, total filtered count, selected compiler-fact identity records, continuation, completeness,
budgets, usage, and explicit omissions.

Canonical catalog field order is `context`, optional `module`, optional `name`, `kinds`, optional
`after`, `total`, `symbols`, optional `next_after`, `complete`, `budget`, `usage`, and `omissions`.
Every symbol record includes visible and full SymbolId, module, declaration kind, exact name, and
public/unsafe/foreign flags. It contains no source body, documentation, absolute host path, fuzzy
score, inferred rename, model text, permission, or executable instruction.

## Exact-name filter

The optional `name` is one ASCII Stage 0 identifier. Matching is exact byte equality and is
case-sensitive. Atlas performs no normalization, case folding, prefix, substring, edit-distance,
ranking, or guessed rename. Equal names in different modules or kinds remain distinct records.

Module, name, and kind filters define the complete candidate set before `total` and cursor
validation. A canonical full-SymbolId cursor must belong to that exact current filtered set. Zero
matches are a complete empty result, not an error or an invitation to broaden the query.

## Integrity and bounds

`fairyc.symbol-catalog-checksum/v1-sha256` hashes a domain/version prefix followed by
length-prefixed catalog version, encoding, and canonical logical result bytes. Its text form is
`fyscat1-sha256-` plus 64 lowercase hexadecimal digits. The checksum detects accidental corruption
and binds the filter and records; it is not authentication.

The query requires positive symbol and selection-byte limits. The mandatory context and echoed
filter record counts toward selection bytes. Encoding separately requires a positive final-output
byte limit. Either boundary fails atomically. Successful output has no trailing newline or second
JSON value.

## v0 relationship

V1 supersedes the output-only v0 writer because v0 had no `name` field. Current code accepts no v0
input and provides no decoder or cursor migration. `docs/atlas/symbol-catalog-v0.md` remains the
historical contract for previously emitted bytes.

## Not enabled

- no decoder or untrusted catalog input;
- no default size, path, file, map-store, cache, or automatic publication;
- no fuzzy, normalized, prefix, substring, dependency catalog, or workspace reverse search;
- no source/documentation text, stable anchors, LSP, SCIP, MCP, or model call; and
- no stable CLI or cross-version cursor promise.
