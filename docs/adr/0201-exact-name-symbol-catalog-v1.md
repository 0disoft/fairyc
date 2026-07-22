# Exact-name Symbol Catalog v1

- Status: Accepted
- Date: 2026-07-23
- Query prerequisite: `docs/adr/0192-exact-current-symbol-catalog-query.md`
- Previous output: `docs/atlas/symbol-catalog-v0.md`
- Current output: `docs/atlas/symbol-catalog-v1.md`
- CLI source: `docs/cli/command-contract.md`

## Context

`experimental atlas-symbols list` can enumerate exact SymbolIds, but a caller that knows a
declaration name still has to page through the whole selected kind set. That defeats the fast
navigation purpose for large packages. Fuzzy or substring search would introduce ranking,
normalization, and guessed-match policy, while omitting the requested name from the output would
make the selection impossible to audit.

Symbol Catalog v0 has a closed canonical result shape without a name-filter field. Adding that
field under the same encoding identifier would silently change a checksummed machine contract.

## Decision

1. Add one optional exact declaration-name filter to `SymbolCatalogRequest` and
   `SymbolCatalogResult`. The name must be one ASCII Stage 0 identifier and matches declaration
   names byte-for-byte and case-sensitively. There is no Unicode normalization, case folding,
   prefix, substring, token, or fuzzy matching.
2. Apply module, name, and declaration-kind filters before computing `total`, paging, or validating
   the exclusive full-SymbolId cursor. A cursor outside that complete filtered set is rejected.
3. Preserve full-SymbolId ordering. Repeated exact names in different modules or declaration kinds
   remain separate results; Atlas never chooses a preferred or supposedly unique match. Zero exact
   matches are a successful complete empty result.
4. Count the echoed optional name in the mandatory selection-byte record. Preserve positive symbol,
   selection-byte, and final-output bounds and the existing no-empty-partial-page rule.
5. Supersede the output-only v0 writer with `fairyc.symbol-catalog/v1`, encoding
   `fairyc.symbol-catalog-json/v1`, checksum algorithm
   `fairyc.symbol-catalog-checksum/v1-sha256`, and text prefix `fyscat1-sha256-`. Keep the checksum
   domain and canonical field order, with optional `name` between `module` and `kinds`.
6. Permit `--name <identifier>` on `fairyc experimental atlas-symbols list`. Cap the CLI value at
   512 bytes, reject duplicates, and echo it in v1 output. The flag remains optional; no hidden
   search term or default is introduced.
7. Keep dependency catalogs, workspace-wide reverse indexes, fuzzy search, stable anchors,
   decoding, text import, map-store policy, model calls, and stable CLI promotion outside this
   decision.

## Evidence

- Query tests require exact matches across two modules, exact module-plus-name narrowing, successful
  zero matches, deterministic ordering, and rejection of malformed names and cursors outside the
  exact filtered set.
- Encoder tests independently recompute the v1 checksum, enforce its final byte bound, reject the v0
  prefix, and prove that the exact name is filter metadata rather than a fuzzy score or source body.
- CLI tests require exact-name and zero-match output, v1 identity, ordinary/locked determinism,
  lexical validation, the 512-byte ceiling, duplicate rejection, and existing failure classes.

## Consequences

Humans and tools can obtain exact SymbolIds from a known declaration spelling without scanning an
entire package catalog. Ambiguity stays visible as multiple exact records, so the feature improves
navigation without pretending that a bare name is a stable symbol identity.

Current writers emit only v1. There is still no catalog decoder or cross-version cursor promise;
v0 remains historical compatibility documentation rather than an accepted input format.

## Rejected alternatives

- Add `name` to v0 without a version change: silently changes the closed checksummed result shape.
- Return the first exact match: hides legal same-name declarations in other modules or kinds.
- Add fuzzy or substring matching now: creates ranking and normalization policy without evidence.
- Accept arbitrary text as a name: expands the trust boundary beyond the language's identifier
  contract and complicates deterministic selection.
