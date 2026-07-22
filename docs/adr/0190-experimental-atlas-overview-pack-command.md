# Experimental Atlas overview pack command

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- CLI source: `docs/cli/command-contract.md`
- Pack prerequisite: `docs/adr/0189-canonical-compiler-fact-context-pack-v1.md`

## Context

Atlas maps, queries, and Context Packs are implemented only as Go APIs. A first command must make the
artifact testable without inventing a hidden store path, numeric defaults, name search, dependency
join, text import, or a broad command family before the output contract has real use.

`show` and `around` require an exact SymbolId discovery workflow that does not exist yet. Exposing
them by fuzzy name would undercut the identity contract. `overview` can safely select one package by
the manifest-owned globally unique workspace UID and needs no relation traversal.

## Decision

1. Add `fairyc experimental atlas-pack overview`. Require explicit `--package-uid`,
   `--max-records`, `--max-selection-bytes`, and `--max-output-bytes`; permit only optional
   `--locked`.
2. Validate the package selector with the manifest-v1 UID grammar and cap it at 256 bytes. Workspace
   loading already rejects duplicate UIDs, so selection is exact and unambiguous.
3. Cap records at 65,536 and both byte limits at 1,073,741,824. Accept only canonical positive
   decimal spellings. Overview fixes unused relation and depth budgets to zero.
4. Use the ordinary manifest-owned workspace and lock boundary. Ordinary mode may atomically refresh
   `fairy.lock`; `--locked` is exact and no-write. Select no map store path and write no `.fymap`.
5. Hash the exact running compiler executable, run package-aware semantic analysis and verified MIR,
   build current partitions and expectations, select the exact package UID, and pass its current
   query index to the Context Pack encoder.
6. On success write exactly the canonical pack bytes to stdout with no trailing newline, banner,
   absolute path, source body, or human receipt.
7. Use code 2 for invalid flags, selectors, or insufficient caller budgets; code 3 for semantic
   diagnostics; code 4 for compiler invariants, executable identity, or output failure; and code 7
   for manifest, workspace, graph, source-loading, or lock failure.
8. Keep `show`, `around`, symbol listing, pack decoding, map publication, default paths, and text
   import outside this command.

## Evidence

Tests cover root and member UID selection, ordinary/locked byte identity, canonical JSON without a
newline, exact pack metadata and budgets, empty text sections, path/source-body exclusion, unknown
UID and undersized-output rejection, workspace source failure, output failure, and missing,
duplicate, unknown, malformed, noncanonical, and over-ceiling options.
