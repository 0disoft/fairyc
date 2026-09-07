# Generic-aware `.fymap` transport v2

- Status: Implemented opt-in internal codec and store; query integration pending
- Logical model: `fairyc.fymap-logical/v2`
- Transport: `fairyc.fymap/v2`
- Encoding: `fairyc.fymap-json/v2`
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`
- Implementation: `internal/atlasmap/codec*.go`

## Version boundary

`EncodeV2` and `DecodeV2` are explicit entry points. Legacy `Encode` and `Decode` still accept only
v1. Neither decoder guesses a version or silently migrates a document. The caller supplies a
positive byte limit; oversized input is rejected before JSON decoding. Output is compact canonical
JSON, with strict known fields, no trailing JSON value, and byte-for-byte canonical reencoding.

The checksum algorithm remains `fairyc.fymap-checksum/v1-sha256`, including its `fymap1-sha256-`
digest prefix. This is the algorithm version, not the document version. Its existing domain and
length-delimited schema, encoding, and canonical-document inputs bind the new v2 schema and encoding
independently. A v1 checksum cannot authenticate a v2 envelope. Neither checksum proves authorship,
compiler provenance, or that the document's claims came from an actual execution.

## Additional records

The common record shapes retain their v1 fields and add these optional nonempty fields:

- Declarations carry ordered `type_parameters` display names. Type references use `parameter` with
  a canonical decimal zero-based binder index, recursively within supported signature types.
- Maps carry `specializations`, ordered by their full compiler-provenance ID. Each record contains
  `id`, the original `declaration`, ordered `type_arguments`, and a separate `function` summary.
- Calls carry `specialization` when they target a concrete generic implementation.
- Header counts carry `specializations` when nonzero. Relation counts include both ordinary
  function summaries and specialization summaries; function counts exclude specializations.
- Headers carry `generic_interface_algorithm` and `generic_interface_hash` together when source
  generic interface facts are present. The algorithm is `fairyc.generic-public-interface/v0` and
  the canonical digest prefix is `fygif0-sha256-`. The ordinary interface field excludes generated
  specializations; the additional hash includes exported template closures, even when unused.

Empty additions are omitted, preserving existing v1 canonical bytes. Decoding explicitly empty or
null additions still requires canonical reencoding and does not create another accepted spelling.
Source text, documentation text, and documentation attachment records remain outside this transport.

## Validation

Generic-aware workspace planning requires one nonzero generic source-interface identity for every
package in the analyzed snapshot, bound to its exact semantic package and stable coordinate.
The original compiler-provenance records, not generated HIR exports, supply these identities.
Missing, extra, zero, or coordinate-mismatched source facts reject planning. Headers with generic
declarations cannot omit the generic interface pair, and v1 rejects these additional identities.
CurrentWorkspaceExpectationsV2 derives the matching source-and-interface expectation set. Reader
compatibility compares both ordinary and generic hashes, without treating an absent expectation
as a wildcard. An unused exported template-body edit changes the generic hash without changing
the ordinary interface hash.

The v1 header, file, module, span, declaration, symbol, ordinary-summary, and completeness checks
remain applicable. Source SymbolIds are rederived using original generic arity and positional types,
not generated function names. Generic declarations cannot have an ordinary executable summary.

Specializations require a canonical lowercase `g0_` plus 64-hex ID, a matching package-local generic
declaration, the exact argument arity, and a summary belonging to that same declaration. Concrete
arguments currently admit only the compiler's scalar set: bool, signed and unsigned fixed/pointer
width integers, f32, and f64. Repeated IDs or repeated declaration/argument bindings are rejected.
Unbound type parameters, malformed nested types, and contradictory counts are rejected.

A local generic call requires a matching specialization record and original declaration. A
specialization call must be a resolved call, not an opaque foreign call. Cross-package calls still
require the exact declared package-reference closure; checking the remote specialization's existence
belongs to the subsequent workspace join, not this single-package decoder.

When a partition declares templates, its completeness ledger must retain unavailable template-body
facts and exact concrete-specialization facts. Counts in that ledger describe the source snapshot
and can exceed this partition's local inventory; they cannot undercount its retained templates or
implementations. Explicit omissions remain in the header. Decoding does not upgrade template-body
uncertainty to executable knowledge.

## Evidence and remaining boundary

The codec suite covers used and unused generic declarations, multiple concrete specializations,
ordinary multi-package compatibility, canonical round trips, exact byte limits, corruption,
unknown fields, trailing data, and cross-version rejection. Resealed mutations test arity, binding
uniqueness, ownership, summary identity, IDs, calls, concrete types, counts, parameter binding, and
template completeness independently from checksum integrity.

This codec does not itself publish maps, authorize source reads, or enable v2 queries. Its explicit
store consumer is documented in `docs/atlas/fymap-store-v2.md`, including current-source and generic
interface validation. Cross-package implementation joins and downstream CLI consumers remain
separate integration requirements.
