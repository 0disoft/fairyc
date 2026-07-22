# `.fymap` transport v1

- Status: Implemented in memory
- Schema: `fairyc.fymap/v1`
- Encoding: `fairyc.fymap-json/v1`
- Checksum: `fairyc.fymap-checksum/v1-sha256`
- Decision: `docs/adr/0186-bounded-canonical-fymap-transport-v1.md`
- Implementation: `internal/atlasmap/`
- Logical input: `docs/atlas/fymap-logical-plan-v1.md`

## Envelope

One map is one compact UTF-8 JSON object:

```text
schema
encoding
checksum_algorithm
checksum
map
  header
  files[]
  modules[]
  symbols[]
  declarations[]
  functions[]
  completeness[]
```

The decoder accepts exactly one value, rejects unknown fields at every depth, and requires an exact
schema, encoding, and checksum-algorithm match. It then validates the checksum and every semantic
record before re-encoding the result and requiring byte-for-byte equality. Whitespace, reordered
fields, duplicate fields, `null` in place of canonical empty arrays, trailing values, and alternate
number spellings are therefore not accepted representations.

All arrays are emitted as arrays, including empty arrays. Spans and counts use nonnegative unsigned
JSON integers. Decoding rejects values outside the host's `int` range before constructing compiler
fact records.

## Checksum input

The canonical checksum text is `fymap1-sha256-` followed by 64 lowercase hexadecimal digits. The
SHA-256 input is:

```text
"fairyc.fymap-checksum" || 0x00 || 0x01
|| u64be(len(schema))   || schema
|| u64be(len(encoding)) || encoding
|| u64be(len(map-json)) || canonical-map-json
```

The checksum value and checksum-algorithm field are outside the digest. Readers require the exact
algorithm identifier before comparing the digest. The frozen empty-document fixture vector is:

```text
fymap1-sha256-f9b1e4db19b424ec1a121eec5cb18d71a56ca9da4bb886253bcdff9eb6f8e016
```

This detects corruption and binds the logical document to this schema and encoding. It is not a
MAC, signature, provenance credential, or authority grant.

## Semantic revalidation

The decoder rejects a map unless all of these remain true:

- every algorithm identifier is the exact implemented version;
- compiler, package, source, file, line-map, interface, and symbol digest spellings are canonical
  and nonzero;
- feature, dependency, omission, file, module, declaration, symbol, function, call, span, borrow,
  and completeness order is canonical and duplicate-free where the producer guarantees uniqueness;
- header counts equal the actual payload and function relation count;
- files and modules are bijective and every span names a map file;
- declarations, symbols, and ordinary function summaries have exact local ownership and coverage;
- type structure is valid and no deeper than 64 nodes along one recursive path;
- generation status and omissions exactly rederive from the completeness ledger; and
- every stored visible/full `SymbolId` exactly rederives from its declaration.

Symbol rederivation uses the header's canonical package namespace references. That list contains
exactly the current package and every direct dependency, each with UID, `PackageInstanceId`, and
`PackageCoordinateId`. It is sorted by instance and cannot silently introduce an unrelated package.

## Bounds and failure

Both `Encode` and `Decode` require a positive caller-selected maximum byte count. Oversize maps fail
before JSON decoding. The transport defines no default because a CLI, editor, server, or embedded
consumer may need different budgets.

Any transport error rejects the entire package map. No partial record set is returned as semantic
truth, and compilation remains unaffected.

## Not implemented here

- output path and naming policy;
- root containment, temporary files, atomic replacement, directory durability, or retention;
- current-source digest recomputation and stale-map rejection;
- dependency-map discovery and cross-map joins;
- `overview`, `show`, `around`, Context Pack, LSP, SCIP, or MCP behavior;
- compression, chunking, incremental rewrites, or performance budgets; and
- signing, authentication, or remote distribution.
