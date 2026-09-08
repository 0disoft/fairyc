# Aggregate-aware Atlas Evidence Bundle v2

- Status: Implemented opt-in encoder and experimental CLI
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Implementation: `internal/evidencebundle/bundle_aggregate.go`
- CLI: `fairyc experimental atlas-evidence show --bundle-version 2`

`EncodeShowV2` accepts an exact-current v3 query, its retained workspace and a
matching semantic snapshot. It composes three byte-exact child documents through
their owning encoders: Context Pack v3, attributed source v0 and attributed
declaration documentation v0. There is no input decoder or caller-authored child.

The subject is one original declaration SymbolId. Aggregate fields and cleanup
observations stay inside compiler facts; original source and documentation stay
in their separately checksummed untrusted-text documents. The bundle has no
authority. Multiple concrete implementations do not duplicate the source text.
Missing, unsafe or oversized text yields the child's explicit omission, not
invented prose. Text never changes compiler completeness or relation facts.

## Version and integrity

```text
bundle version:     fairyc.atlas-evidence/v2
encoding:           fairyc.atlas-evidence-json/v2
checksum algorithm: fairyc.atlas-evidence-checksum/v2-sha256
checksum prefix:    fyevidence2-sha256-
```

Envelope order and compact no-newline JSON follow bundle v1. SHA-256 covers ASCII
`fairyc.atlas-evidence-checksum`, bytes `00 02`, then bundle version, encoding and
canonical bundle JSON. Each final string is prefixed by its unsigned 64-bit
big-endian byte length. The checksum binds child bytes, authorities, subject and
limits. `ParseChecksumV2` accepts only the v2 prefix and 64 lowercase hexadecimal
digits; old parsers reject v2. Checksums are not authentication or permission.

Facts, each text document and the final bundle retain independent output limits.
Any child or final envelope overflow returns no partial bundle. Text fragment
budgets apply separately to source and documentation. No filesystem discovery,
map cache, process, model, network or execution authority is added.

## CLI and evidence

Discover the original resource with symbol catalog 3, then pass its visible/full
ID to `atlas-evidence show --bundle-version 2` with all existing required limits.
Only canonical values 0, 1 and 2 are accepted once. Default 0 and explicit 0/1
retain previous admission and output. `--locked` remains no-write; otherwise lock
refresh may precede a later error. Stable aggregate syntax is not promoted.

Tests compare all three child bytes with direct encoders, check cleanup source
and concrete fields, separate hostile documentation from facts, independently
calculate the checksum, reject wrong maps/snapshots and enforce child/final
limits and omissions. CLI tests discover IDs, compare locked/full-ID output,
check legacy rejection and keep output-write errors visible.
