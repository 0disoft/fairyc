# Aggregate-aware Context Pack v3

- Status: Implemented opt-in encoders and experimental CLI
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Implementation: `internal/contextpack/pack_aggregate.go`
- CLI: `fairyc experimental atlas-pack overview|show|around --pack-version 3`

`EncodeOverviewV3`, `EncodeShowV3` and `EncodeAroundV3` accept exact-current v3
queries only. Legacy v1/v2 encoders retain their exact map admission and output.
The document/envelope ordering and empty text channels follow Context Pack v2.

Overview counts original declarations. Show retains the original struct/resource
and concrete `aggregate` fields separately from optional `cleanup` relation counts.
Neither is a function summary. Cleanup calls and borrows retain `source_body:
"cleanup"`, original resource SymbolId and concrete source specialization.
Calls also preserve target specializations. Around describes a declaration
neighborhood, not a concrete execution path or implicit drop schedule.

Template-body uncertainty and existing relation completeness remain explicit.
Selection limits include full concrete fields; final byte limits include the
whole envelope. A required subject or output that does not fit fails atomically.
Relation/depth limits retain explicit omissions rather than asserting completeness.
Source and documentation arrays remain empty: this compiler-fact pack does not
authorize source extraction, join dependency maps or perform filesystem discovery.

## Integrity

```text
pack version:       fairyc.context-pack/v3
encoding:           fairyc.context-pack-json/v3
checksum algorithm: fairyc.context-pack-checksum/v3-sha256
checksum prefix:    fyctx3-sha256-
```

SHA-256 covers ASCII `fairyc.context-pack-checksum`, bytes `00 03`, then pack
version, encoding and canonical pack JSON. Each of the last three strings is
prefixed by its unsigned 64-bit big-endian byte length. The parser accepts only
the v3 prefix and 64 lowercase hexadecimal digits. Legacy parsers reject v3.
Checksums detect changed bytes, not authenticity, validity or permission.

## CLI and evidence

Discover a resource ID with `atlas-symbols list --catalog-version 3`, then pass
that ID to `atlas-pack show|around --pack-version 3` with the existing required
package/relation/selection/output budgets. Overview needs no symbol. Canonical
pack choices are 1, 2 and 3, at most once; default 1 is unchanged. The explicit
v3 path uses aggregate-aware uncached preparation, not stable source admission.
`--locked` and output/source/invocation failure policies are unchanged.

Tests cover nested concrete fields, two cleanup specializations, call and borrow
ownership, overview/show/around, independently calculated checksums, legacy
rejection and atomic byte limits. CLI tests use discovered resource IDs and
verify locked output, invalid versions, limits and legacy aggregate rejection.
Attributed-text commands separately expose original aggregate source/documentation
without populating this pack's text channels. Evidence Bundle v2 composes these
independent documents without merging their authority.
