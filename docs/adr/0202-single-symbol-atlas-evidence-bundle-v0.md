# Single-symbol Atlas Evidence Bundle v0

- Status: Accepted
- Date: 2026-07-23
- Compiler-fact source: `docs/atlas/context-pack-v1.md`
- Attributed-text source: `docs/atlas/attributed-text-v0.md`
- Bundle contract: `docs/atlas/evidence-bundle-v0.md`
- CLI source: `docs/cli/command-contract.md`

## Context

An exact symbol can already be inspected through three independently bounded commands: Context Pack
v1 for compiler facts, attributed-text v0 source output for its exact declaration span, and
attributed-text v0 documentation output for its exact attached `///` span. A consumer that needs all
three has to repeat workspace loading, compilation, exact-current validation, package selection, and
SymbolId resolution. That is wasteful and makes it easier for an adapter to accidentally combine
artifacts from different source states.

Silently filling the empty `documentation` and `source_excerpts` arrays in Context Pack v1 is not a
valid shortcut. Those empty arrays are a deliberate compiler-fact-only contract, and changing them
under the existing checksummed encoding would blur the trust boundary between facts and prose.

## Decision

1. Add output-only `fairyc.atlas-evidence/v0`, encoded as compact canonical
   `fairyc.atlas-evidence-json/v0`. It composes one exact-symbol Context Pack v1 document, one exact
   source attributed-text v0 document, and one exact declaration-documentation attributed-text v0
   document. Each child remains a complete independently checksummed canonical JSON value.
2. Construct every child through its owning encoder from one already loaded workspace, one matching
   immutable semantic snapshot, and one exact-current query index. The composer accepts no
   caller-authored JSON, text, query result, path, or checksum and provides no decoder.
3. Keep authority visibly separate. The bundle has authority `none`; its facts component is labelled
   `compiler-fact`; source and documentation are each labelled `untrusted-text`. The wrapper does not
   authenticate, endorse, sanitize, summarize, or promote any child and grants no host, tool, model,
   filesystem, process, network, credential, or approval authority.
4. Bind the wrapper with `fairyc.atlas-evidence-checksum/v0-sha256`, rendered as
   `fyevidence0-sha256-` plus 64 lowercase hexadecimal digits. The checksum covers the version,
   encoding, exact nested child bytes, authority labels, canonical visible/full SymbolId, package
   identity, and caller limits. It is deterministic integrity evidence, not a signature or proof of
   authorship.
5. Require independent positive limits for query records, relations, and selection bytes; source and
   documentation fragment bytes and text bytes; Context Pack bytes; each attributed-text document;
   and the final bundle. The one-fragment text budget applies separately to source and documentation,
   as does the attributed-text document limit. No value is a token budget or product default.
6. Expose only `fairyc experimental atlas-evidence show`. It requires exact package UID and visible
   or full SymbolId selectors, explicit direction and one or more relation kinds, every numeric limit,
   and optional `--locked`. There is no `overview`, `around`, name selector, implicit relation set,
   store path, configuration fallback, or stable command promise.
7. Preserve child behavior. An unavailable documentation attachment or text safety/size rejection is
   a successful checksummed omission inside that child. Unknown subjects, invalid policy, insufficient
   query or child/final document bounds, stale identities, and invariant failures reject the complete
   bundle before stdout is written. An external writer may still accept a prefix before reporting its
   own failure.
8. Leave `.fymap` v1, Context Pack v1, attributed-text v0, Symbol Catalog v1, and their exact bytes
   unchanged. The bundle is a new output family, not a new input format or a compatibility alias.
9. Keep dependency/generated/compiler-bundled text, multiple source fragments, rendered docs,
   summaries, token estimation, retrieval ranking, map publication, persistent bundle storage,
   signatures, model calls, MCP, agent permissions, and safe-change claims outside this decision.

## Evidence

- Package tests independently recompute the wrapper checksum, validate all three nested documents,
  prove visible/full SymbolId byte identity, retain one loaded workspace after disk mutation, reject
  each component/final byte bound independently, and keep prompt-like documentation labelled as
  untrusted text rather than compiler fact.
- CLI tests require ordinary/locked byte identity, canonical no-newline stdout, exact source versus
  documentation separation, successful `kind-unavailable` and text-budget omissions, closed flags and
  actions, unknown package/symbol rejection, final and child limit failures, path exclusion, and
  output-write failure classification.
- Existing Context Pack tests continue to require empty `documentation`, `source_excerpts`, and
  `warnings`; existing attributed-text encodings and checksums are reused without modification.

## Consequences

One bounded command can now produce a coherent exact-current evidence set for an exact symbol while
the child contracts remain independently verifiable and visibly different in authority. The command
reduces repeated compiler work, but it intentionally does not make the prose trustworthy or the local
package graph complete.

The wrapper is experimental output only. Consumers that need to persist or ingest it must first define
a strict decoder, compatibility acceptance policy, and resource limits in a separate decision.

## Rejected alternatives

- Populate Context Pack v1 text arrays: breaks the existing compiler-fact-only checksummed contract.
- Return source and documentation as plain wrapper strings: discards exact attribution, omissions,
  budgets, and independent integrity.
- Accept caller-provided child JSON: permits forged or cross-workspace composition.
- Give the bundle one undifferentiated `evidence` authority: hides the fact/text trust boundary.
- Add defaults or an automatic “best context” query: makes cost, graph policy, and relevance implicit.
- Add `around` immediately: multiplies text-selection and per-node budget policy before the
  single-symbol contract has real consumer evidence.
