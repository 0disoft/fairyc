# Generic-aware package-local queries v2

- Status: Implemented internal opt-in query API; workspace join and CLI integration pending
- Entry point: `NewQueryIndexV2`
- Store: `docs/atlas/fymap-store-v2.md`
- Shared query policy: `docs/atlas/fymap-query-v1.md`
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`

## Admission and scope

NewQueryIndexV2 requires a valid v2 partition and independently derived current-source expectations,
including its exact compiler and original ordinary/generic interface identities. It applies the
same compatibility checks as the v2 store. NewQueryIndex remains exact-v1. No constructor discovers
dependencies, opens source files, or grants filesystem authority.

Overview, catalog, show, and around retain their explicit filters, deterministic source-symbol
selection, byte/record/relation/depth budgets, omission reporting, and detached outputs. Catalogs
contain original declarations once, including unused templates; generated implementation names
are not source symbols. Existing declaration-kind filters still apply.

## Concrete facts without universal template claims

A generic QueryNode contains its original declaration and ordered type-parameter names. Its ordinary
`function` field is absent: checking one or more implementations does not prove a universal body.
Its optional `specializations` list contains each retained implementation ID, ordered concrete
`type_arguments`, and compiler-observed function relation counts. An unused template has no concrete
summaries or observed outgoing body relations.

Relations add optional `source_specialization`; call targets add optional `specialization`.
Both endpoints retain their source declaration identity. Ordering distinguishes implementation IDs
even when the original source span and declaration are the same. Incoming and outgoing selections
remain keyed by original source symbols and expose all selected, individually attributed concrete
edges. Around is a declaration-neighborhood traversal over those edges, not proof that a chain can
execute for one particular type binding. Cross-package targets remain explicitly unjoined.

Query context includes the generic source-interface hash and preserves both concrete-specialization
completeness and unavailable template-body completeness independently of the selected relation
filter. No empty result turns unknown universal bodies into exact empty behavior.

## Bounds and evidence

The required node's complete specialization metadata participates in the existing compact-JSON
selection-byte budget. If the required subject does not fit, the query fails with
ErrQueryBudgetTooSmall; it does not truncate a declaration or silently omit implementations.
Relation limits still yield explicit omission records. These selection bytes are not a stable
wire format or a provider-token estimate.

Tests exercise compilation through v2 storage and querying, used/unused templates, separate borrow
and helper-call facts for two scalar bindings, exact original-symbol catalog counts, detached
accessors, relation and byte budgets, declaration-neighborhood traversal, incompatible-interface
rejection, and the legacy v1 admission boundary. Existing v1 query tests remain active.

Context Pack output, cross-package specialization validation, and CLI activation require their own
consumer updates. The existing v1 workspace-call join still rejects v2 rather than discarding its
implementation identities.
