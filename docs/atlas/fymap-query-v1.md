# `.fymap` bounded package-local queries v1

- Status: Implemented internal query API
- Decision: `docs/adr/0188-bounded-current-fymap-queries.md`
- Exact-name catalog decision: `docs/adr/0201-exact-name-symbol-catalog-v1.md`
- Implementation: `internal/atlasmap/query.go`
- Store prerequisite: `docs/atlas/fymap-store-v1.md`
- Context Pack consumer: `docs/atlas/context-pack-v1.md`

## Admission boundary

`NewQueryIndex` accepts one transport-valid partition plus a complete `LoadExpectation`. It repeats
the exact current package, file, line-map, compiler executable, and public-interface checks used by
the contained store. A caller cannot decode an old map and obtain a query index without supplying
independently derived current workspace expectations.

The index is package-local. It performs no filesystem scan, source read, dependency-map discovery,
network access, process execution, or compiler feedback. All output is detached from the immutable
index. Symbol selectors accept only canonical visible or full SymbolId v1 text.

## Query surface

- `overview` returns the exact map header, declaration-kind counts, then relation completeness and
  module records in deterministic priority order.
- `show` returns one exact symbol, its declaration and function relation counts, then a bounded
  incoming, outgoing, or bidirectional relation list.
- `around` performs deterministic breadth-first traversal from one exact symbol. It returns the root
  first, stable relation order within each depth, and stable full-SymbolId node order within the next
  frontier.
- `catalog symbols` returns exact declaration identity records ordered by full SymbolId. It requires
  an explicit declaration-kind filter, permits exact module and Stage 0 declaration-name filters,
  and pages only after a full SymbolId present in the same current filtered set. Name matching is
  byte-for-byte and preserves every same-name record across modules and kinds.

The explicit relation filter supports `allocates`, `borrows-mutable`, `borrows-shared`, `calls`,
`calls-foreign`, `drops`, `may-trap`, `moves`, `returns-error`, and `uses-unsafe`. An unrequested
relation family is recorded as a filter omission rather than silently disappearing.

Function-summary facts that do not have a semantic target are terminal relations. Local calls carry
the target visible SymbolId and may extend an `around` traversal. Dependency calls retain their
exact package/module/kind/name target but cannot invent a SymbolId without the dependency
declaration and signature. Foreign calls retain their local foreign declaration, opaque certainty,
ABI, and link name.

## Mandatory budgets

Every query requires:

- a positive maximum total selected-record count;
- a nonnegative maximum relation count;
- a nonnegative maximum traversal depth; and
- a positive maximum selection-byte count.

Selection bytes are the sum of each admitted record's compact Go JSON encoding plus one separator
byte. This is a deterministic selection unit, not a stable wire format, final wrapper size, model
token estimate, or provider-specific tokenizer result. The exact subject and its current-map context
are mandatory. If they do not fit, the query returns `ErrQueryBudgetTooSmall`; it never emits half a
signature. Later records stop at the first record that would exceed a count or byte bound.

Results echo the governing budget and actual records, relations, depth, and selection bytes. Budget,
depth, relation-filter, dependency-join, and unavailable-incoming-map omissions use stable machine
codes with known counts where the index can prove them. An unknown count remains explicitly
unknown.

The symbol catalog has separate positive symbol-count and selection-byte limits. Its mandatory
context/request/total record, including echoed optional module and name filters, counts toward
selection bytes, while the symbol limit counts emitted identity records. Module, name, and kind
filters define `total` before paging. A partial page reports a known remaining count and the last
emitted full SymbolId as the exclusive next cursor. A request that cannot fit its first candidate
fails instead of returning an empty page that cannot advance. Zero exact-name matches are a
successful complete empty result.

## Completeness and dependency boundary

Query context carries the map generation status, map omissions, and the completeness records that
govern the requested relation families. Package-local outgoing calls are exact to the stored
compiler facts. Cross-package target nodes are unavailable until dependency maps are joined, and a
single package map cannot know callers from other packages. `show` and `around` therefore emit
`dependency-map-not-joined` and `incoming-package-maps-not-joined` instead of presenting a closed
workspace graph.

`around` traverses only target symbols present in the current package index. A depth or budget stop
is explicit. The implementation does not infer indirect targets, rename matches, transitive impact,
or safe-change claims.

## Not enabled

- no query-result decoder or standalone stable query wire schema; the output-only compiler-fact
  Context Pack is separately owned by `docs/atlas/context-pack-v1.md`;
- no source excerpt or documentation import;
- no dependency-map join or workspace-wide reverse index;
- no fuzzy, normalized, prefix, substring, or guessed-rename search;
- no `impact`, semantic `diff`, LSP, SCIP, MCP, or structural edit behavior;
- no stable CLI, default map path, or automatic publication; ADR 0190 exposes bounded package-UID
  overview packs, ADR 0193 exposes the symbol catalog command, ADR 0201 fixes exact-name Symbol
  Catalog v1, and ADR 0194
  exposes exact-SymbolId `show` and `around` packs; and
- no token-count, latency, retained-size, or rewrite-speed claim.
