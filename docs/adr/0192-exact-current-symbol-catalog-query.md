# Exact-current SymbolId catalog query

- Status: Accepted
- Date: 2026-07-23
- Query source: `docs/atlas/fymap-query-v1.md`
- Identity source: `docs/atlas/symbol-id-v1.md`

## Context

`show` and `around` correctly require a canonical SymbolId, but overview does not list those IDs.
Guessing by name would be wrong for repeated names, owner-qualified declarations, renames, and
truncated results. Large packages also require deterministic pagination. The visible 128-bit ID is
an exact lookup form after collision validation, but only the stored full digest is an unambiguous
page boundary.

## Decision

1. Add `QueryIndex.CatalogSymbols` as a separate exact-current package-local query. Construction
   retains the existing current source, compiler executable, interface, and transport checks.
2. Order candidates only by canonical full SymbolId. Return visible and full forms plus module,
   declaration kind, name, and public/unsafe/foreign flags. Include no source text or fuzzy rank.
3. Require an explicit nonempty declaration-kind filter and permit one optional exact module filter.
   Unknown modules, unsupported kinds, and duplicate kinds fail closed.
4. Require positive symbol-count and selection-byte limits. Selection bytes include one mandatory
   context/request/total record plus each admitted symbol record's compact JSON size and separator.
5. Permit only a canonical full SymbolId as the exclusive `after` cursor. It must exist in the
   current filtered candidate set. Visible IDs, arbitrary digests, and cursors from another filter
   are rejected rather than silently skipping records.
6. A partial page returns the last emitted full SymbolId as `next_after` and a known remaining count.
   If the first candidate cannot fit, return `ErrQueryBudgetTooSmall`; never emit an unpageable empty
   partial result.
7. Keep output encoding, CLI spelling, name search, dependency catalogs, stable anchors, and
   `show`/`around` command exposure outside this decision.

## Evidence

Real-workspace tests require repeated deterministic results, exact full-ID ordering, complete count
agreement, one-record stateless pagination, exact module/kind filtering, and closed rejection for
missing, duplicate, or unsupported kinds, unknown modules, visible or foreign cursors, and zero or
insufficient budgets.
