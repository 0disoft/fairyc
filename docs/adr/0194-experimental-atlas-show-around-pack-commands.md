# Experimental Atlas show and around pack commands

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Query prerequisite: `docs/adr/0188-bounded-current-fymap-queries.md`
- Pack prerequisite: `docs/adr/0189-canonical-compiler-fact-context-pack-v1.md`
- Discovery prerequisite: `docs/adr/0193-experimental-symbol-catalog-command.md`

## Context

The compiler already owns exact-current `show` and `around` queries and canonical compiler-fact
Context Pack encoders. ADR 0193 now gives external callers a bounded exact SymbolId discovery path,
so keeping relation packs internal no longer protects identity correctness. Exposing them still must
not invent name search, graph completeness, hidden traversal limits, a map-store default, or a free-
text ingestion path.

## Decision

1. Add `fairyc experimental atlas-pack show` and `fairyc experimental atlas-pack around`.
2. Require exact `--package-uid`, canonical visible or full `--symbol`, one
   `--direction <outgoing|incoming|both>`, one or more exact `--relation` filters,
   `--max-records`, `--max-relations`, `--max-selection-bytes`, and `--max-output-bytes`.
   `around` additionally requires `--max-depth`; `show` rejects it and fixes depth to zero.
3. Accept only the relation families implemented by the current query contract. Reject duplicate
   relations and sort the accepted filter canonically. Numeric inputs are canonical decimals.
   Records and relations are positive and capped at 65,536; selection and output bytes are positive
   and capped at 1 GiB; around depth is nonnegative and capped at 64.
4. Reuse the manifest-owned workspace, ordinary or exact no-write `--locked` behavior, exact running
   executable identity, package-aware semantic analysis, verified MIR, current partition planning,
   and exact package-UID selection used by overview and symbol-catalog commands. Select no map-store
   path and write no `.fymap`, cache, or source artifact.
5. Run `show` for one exact subject or deterministic breadth-first package-local `around`, preserving
   query omissions for relation filters, dependency maps, incoming package maps, and caller budgets.
   Full SymbolIds are accepted selectors but the canonical pack subject remains the visible ID.
6. On success write exactly one checksummed `fairyc.context-pack-json/v1` value to stdout with no
   trailing newline, path, source body, documentation text, banner, or human receipt.
7. Use code 2 for invalid options or selectors, unknown packages or symbols, and insufficient query
   or final-output budgets; code 3 for source diagnostics; code 4 for compiler invariants,
   executable-identity failures, or output failure; and code 7 for manifest, workspace, graph,
   source-loading, or lock failure.
8. Keep name or fuzzy search, dependency-map joins, workspace-wide reverse callers, pack decoding,
   attributed text, map publication, stable command promotion, and agent authority outside these
   commands.

## Evidence

- CLI tests discover a real function through `atlas-symbols list`, then require visible-ID `show`
  and full-ID `around` to emit canonical no-newline JSON with exact query descriptors, relation
  facts, nonzero traversal, empty text sections, and no source body or host path.
- Ordinary and locked `show` output must be byte-identical for an already current lockfile.
- Parser and execution tests reject missing or duplicate policy, invalid selectors, directions,
  relations and depths, zero or over-ceiling budgets, unknown packages or symbols, insufficient
  selection/output bytes, and output-writer failure with the documented code family.

## Consequences

An IDE, human, CI job, or LLM-facing adapter can now perform exact discovery followed by bounded
fact retrieval without reading a hidden store or asking the compiler to guess a name. The result is
still only a package-local, explicitly incomplete compiler-fact view. It is not a safe-change claim,
an authenticated artifact, a permission token, or a text-import contract.
