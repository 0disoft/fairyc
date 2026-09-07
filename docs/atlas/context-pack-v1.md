# Atlas Context Pack v1

- Status: Implemented internal compiler-fact-only encoder with experimental overview/show/around CLI exposure
- Pack version: `fairyc.context-pack/v1`
- Encoding: `fairyc.context-pack-json/v1`
- Checksum: `fairyc.context-pack-checksum/v1-sha256`
- Implementation: `internal/contextpack/`
- Decision: `docs/adr/0189-canonical-compiler-fact-context-pack-v1.md`
- Query source: `docs/atlas/fymap-query-v1.md`

This remains the default and explicit `--pack-version 1` output. Its encoders accept only v1 maps.
The generic-aware opt-in `--pack-version 2` contract is in `docs/atlas/context-pack-v2.md`.

## Purpose and boundary

A Context Pack is deterministic bounded data for a human, IDE, CI job, or LLM-facing tool. It is
not an instruction, prompt, permission, tool authority, or claim that an incomplete graph is
complete.

The v1 encoder accepts a current `atlasmap.QueryIndex`, a query request, the query's mandatory
record/relation/depth/selection-byte budget, and a separate positive final-pack byte limit. It runs
`overview`, `show`, or `around` itself; it does not accept a caller-mutated result presented as
compiler truth. Query-index construction has already required exact map transport, source, file,
line-map, compiler, and public-interface compatibility.

The encoder performs no filesystem access, source loading, dependency discovery, process execution,
network request, model call, tokenizer call, or compiler feedback.

## Canonical envelope

The exact top-level field order is:

```text
pack_version
encoding
checksum_algorithm
checksum
pack
```

`pack` contains, in order:

```text
map_version
package_identity
compatibility
query
budget
usage
completeness
facts
relations
documentation
source_excerpts
omissions
warnings
```

`package_identity` retains package UID, exact instance, coordinate, and source digest.
`compatibility` retains compiler build, language revision, public-interface hash, map generation
status, and map-level omissions. `query` records kind, canonical subject, direction, and relation
filter. Exactly one of `facts.overview`, `facts.show`, or `facts.around` is non-null. Relations remain
separate compiler-observed records rather than being rewritten into prose.

Go's compact JSON encoder over fixed structs defines the only current byte representation. Arrays
that have no entries encode as `[]`, and disabled fact variants encode as `null`. Field omission,
pretty printing, key reordering, or another JSON spelling is not this encoding.

## Checksum and bounds

The checksum is `fyctx1-sha256-` plus 64 lowercase hexadecimal digits. SHA-256 hashes a domain and
version prefix followed by length-prefixed pack version, encoding, and canonical logical `pack`
bytes. The checksum field itself is excluded. It provides canonical-integrity and corruption
evidence, not authentication or signer identity.

Two independent bounds apply:

1. the query budget limits selected logical records, relations, traversal depth, and deterministic
   selection bytes; and
2. the pack encoder rejects the complete canonical envelope when it exceeds the caller's positive
   final byte limit.

Selection bytes are not final envelope bytes and neither value claims model-token equivalence.
Budget stops, relation filters, depth stops, unjoined dependency targets, and unavailable incoming
package maps remain machine-readable omissions with known counts only where proven.

## Text safety

Compiler-fact-only v1 always emits empty `documentation`, `source_excerpts`, and `warnings` arrays.
It does not accept even locally sourced prose. This makes the absence of free text visible and keeps
source comments, generated summaries, prompt injection, control characters, secrets, absolute host
paths, and unrelated files outside the first output contract.

The separately encoded attributed-text capability implemented by ADR 0196 and ADR 0197, and exposed
by the ADR 0198 source and ADR 0200 documentation commands, remains outside this schema. ADR 0202's
Evidence Bundle v0 can wrap the complete unchanged Context Pack beside two independently checksummed
attributed-text documents, but it does not link or embed text inside this pack. These v1 arrays cannot
silently start carrying text.

## Determinism and privacy

Identical current map, query, budgets, and encoder version produce byte-identical output and
checksum. Query ordering uses semantic identity and deterministic breadth-first traversal, never Go
map iteration or filesystem order.

Packs exclude source bodies, documentation bodies, absolute local paths, credentials, environment
variables, build logs, cache locations, timestamps, unrelated source, and generated artifacts.

## Implemented query support

- `overview` packs the exact header and declaration-kind counts plus budget-selected completeness
  and module records.
- `show` packs one exact visible SymbolId subject, its declaration/function facts, completeness, and
  bounded relations.
- `around` packs one exact subject, bounded breadth-first local nodes, relations, completeness, and
  omissions.

Dependency-map joins and workspace-wide reverse callers remain absent, so their omissions survive
unchanged into the pack. `impact` and semantic `diff` remain deferred.

## Not enabled

- no Context Pack decoder or acceptance of untrusted pack bytes;
- no source excerpt, documentation import, generated summary, or warning producer in this pack;
  ADR 0195 through ADR 0200 keep separate attributed-text extraction and output outside these empty
  v1 arrays, and ADR 0202 composes only the complete unchanged child outputs;
- no dependency-map join, workspace-wide index, name search, fuzzy match, or rename inference;
- no stable command, default map path, or automatic publication; experimental CLI exposure is
  limited to the explicit overview contract in ADR 0190 and exact-SymbolId show/around contract in
  ADR 0194;
- no LSP, SCIP, MCP, structural edit, agent permission, embedding, or vector database behavior; and
- no output-token, latency, retained-size, rewrite-speed, or safe-change claim.
