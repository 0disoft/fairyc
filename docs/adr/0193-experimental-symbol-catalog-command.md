# Experimental symbol catalog command

- Status: Accepted
- Date: 2026-07-23
- Catalog source: `docs/atlas/symbol-catalog-v0.md`
- CLI source: `docs/cli/command-contract.md`
- Follow-up: `docs/adr/0201-exact-name-symbol-catalog-v1.md`

## Context

The internal exact catalog now supplies the prerequisite for `show` and `around`, but an external
caller still cannot obtain SymbolIds. The command must retain exact filters and pagination without
inventing defaults, fuzzy name matching, a map-store location, or a hidden output budget.

## Decision

1. Add `fairyc experimental atlas-symbols list` with required `--package-uid`, one or more `--kind`,
   `--max-symbols`, `--max-selection-bytes`, and `--max-output-bytes` options. Permit optional exact
   `--module`, canonical full-SymbolId `--after`, and `--locked`.
2. Accept only the five current declaration kinds. Reject duplicate kinds and sort the filter
   canonically. Cap symbols at 65,536, byte limits at 1 GiB, package UID at 256 bytes, and module
   input at 512 bytes. Numeric values are canonical positive decimals.
3. Use the same manifest-owned workspace, lock, exact running executable identity, verified MIR,
   partition preparation, and current expectation path as `atlas-pack overview`.
4. Run the exact catalog query and emit one canonical checksummed
   `fairyc.symbol-catalog-json/v0` value to stdout with no trailing newline. Select no map-store path.
5. Use code 2 for invalid options, filters, cursors, unknown package/module, insufficient query
   budget, or insufficient final-output budget; code 3 for semantic diagnostics; code 4 for compiler
   invariants or output failure; and code 7 for manifest, workspace, graph, loading, or lock failure.
6. Keep `show`, `around`, name search, dependency joins, decoder input, text import, and stable CLI
   promotion outside this command.

## Evidence

Tests cover canonical root-package output, no newline/path/source leakage, full-ID continuation to a
different record, ordinary/locked byte identity, required and duplicate kind handling, unknown
kinds/modules, visible cursor rejection, zero and undersized budgets, final output bounds, and
output failure classification.
