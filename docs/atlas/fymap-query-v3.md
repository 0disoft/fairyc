# Aggregate-aware package-local queries v3

- Status: Implemented opt-in query API, workspace join and catalog CLI consumers
- Entry point: `NewQueryIndexV3`
- Store: `docs/atlas/fymap-store-v3.md`
- Shared query policy: `docs/atlas/fymap-query-v1.md`
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`

## Admission

The constructor requires a valid v3 partition and independently derived current
source, compiler and ordinary/generic interface expectations. V1/v2 query
constructors remain exact-version entry points. No constructor discovers maps,
opens source files or grants filesystem authority.

Overview, catalog, show and around retain deterministic selection, explicit
filters, byte/record/relation/depth budgets and omissions. Catalogs contain original
declarations once, including unused struct/resource templates, never generated
implementations as source symbols.

## Aggregate and cleanup facts

Each concrete specialization contains either `function` facts or an `aggregate`
declaration with substituted field types. Non-empty resource cleanup has a separate
`cleanup` relation-count summary. Neither structs nor empty cleanup acquire a
function summary. The original template's ordinary `function` remains absent;
observed specializations do not prove a universal body.

Relations originating in cleanup retain the original resource symbol, exact
`source_specialization`, and `source_body: "cleanup"`. Calls retain the target
helper's original symbol and concrete specialization. Both incoming and outgoing
queries preserve those identities. Around follows a declaration neighborhood,
not a proof that every selected edge executes for one type binding.

Complete aggregate metadata participates in the required subject's compact-JSON
byte cost. A subject that does not fit fails rather than dropping fields or
implementations. Relation limits produce explicit omissions. All outputs are
detached, including nested aggregate type arguments and cleanup counts. Query
context retains template-body uncertainty even for an empty relation selection.

## Evidence and integration boundary

Tests cover used/unused templates through v3 file storage, original-kind catalog
filtering, concrete fields, separate cleanup calls and borrows, incoming queries,
around, exact subject byte bounds, relation omissions, detached results and
incompatible interfaces. Existing v1/v2 query and downstream tests remain active.
Function-only v2 output does not gain aggregate, cleanup or source-body fields.

`NewWorkspaceCallIndexV3` consumes these relations with cleanup ownership intact;
see `docs/atlas/workspace-call-join-v0.md`. Call catalog v2 preserves the joined
cleanup marker. Experimental symbol catalog 3 and call catalog 2 commands now
explicitly adopt v3, as does Context Pack with `--pack-version 3`. Attributed-text
now uses v3 without changing text v0; Evidence Bundle integration remains pending.
