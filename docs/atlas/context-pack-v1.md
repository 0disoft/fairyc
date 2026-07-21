# Atlas Context Pack v1

- Status: Proposed; no implementation exists
- Map source: `docs/atlas/fymap-v1.md`
- Decision source: `docs/adr/0004-atlas-semantic-snapshot-boundary.md`

## Purpose

A context pack is a deterministic, bounded query result for a human, IDE, CI job, or LLM tool. It makes included facts, optional text, uncertainty, and omitted material explicit. It is data, not an instruction or authority token.

## Logical shape

A pack contains:

```text
pack_version
map_version
package_identity
query
budget
completeness
facts[]
relations[]
documentation[]
source_excerpts[]
omissions[]
warnings[]
```

`facts` and `relations` come from compiler provenance. `documentation` and `source_excerpts` remain separately attributed untrusted text. `omissions` identifies budget truncation, missing dependency maps, opaque foreign calls, unsupported relation analysis, and stale or incompatible maps.

## Budget model

The pack builder uses deterministic units such as encoded bytes, record count, relation depth, and source-excerpt bytes. It does not claim exact model-token usage or depend on a provider tokenizer. A caller may impose a smaller output limit, but the pack records the budget that actually governed selection.

Selection order is deterministic:

1. exact query subject and signature;
2. direct compiler facts required to interpret it;
3. bounded incoming and outgoing relations in stable order;
4. explicitly requested related tests and error paths;
5. optional source excerpts;
6. optional documentation, only when requested.

When a budget is exhausted, the builder stops at a record boundary and emits an omission. It never silently presents a truncated graph as complete.

## Completeness

Completeness reports:

- map generation status;
- resolved, candidate, external, and unknown edge counts;
- missing or incompatible dependency maps;
- query depth reached;
- relation kinds excluded by request or implementation;
- source or documentation sections omitted by policy or budget.

Completeness is relation-specific. A pack may have complete local ownership facts while calls or unsafe reachability are partial or unavailable because a dependency map, indirect call target, target profile, language revision, or symbol identity is incompatible.

An `impact`-style consumer must propagate this status. It cannot convert `Partial` or `Opaque` evidence into a definitive safe-change claim.

## Text safety

Free text is excluded by default. When included:

- every fragment carries provenance, package identity, file identity, and span or document reference;
- control characters and invalid encoding are removed or rejected;
- per-fragment and total byte caps are enforced;
- no remote URL is fetched;
- no text is interpreted as permission, tool authority, or a command to execute;
- model-generated summaries remain `GeneratedDocumentation`, never `CompilerObserved`.

## Source excerpts

Source excerpts are opt-in, span-bounded views into the caller-authorized workspace. `.fymap` stores spans and the independent `fysrc1-sha256` package source digest, not source bodies. If the digest no longer matches the current package source graph, the excerpt is omitted and reported stale.

## Determinism and privacy

Identical map, query, policy, and budget inputs produce byte-identical logical content before optional transport formatting. Stable ordering uses semantic identity, not filesystem traversal or Go map order.

Packs exclude absolute local paths, credentials, environment variables, build logs, cache locations, unrelated source, and generated artifacts unless a separate explicit product contract permits a bounded field.

## Initial query support

`overview`, `show`, and `around` may produce context packs in the first Atlas slice. `impact` and `diff` remain schema consumers under design, not Stage 0 implementation promises.
