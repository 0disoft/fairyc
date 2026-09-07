# Atlas Context Pack v2

- Status: Implemented compiler-fact-only output with explicit overview/show/around CLI exposure
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`
- Implementation: `internal/contextpack/pack_generic.go`
- Query source: `docs/atlas/fymap-query-v2.md`
- CLI: `fairyc experimental atlas-pack overview|show|around --pack-version 2`

## Admission and facts

`EncodeOverviewV2`, `EncodeShowV2`, and `EncodeAroundV2` consume only exact-current v2 map queries.
The original encoders remain v1-only and reject v2 maps rather than changing their output contract.
Every builder executes its bounded query itself; it accepts no caller-supplied result, source text,
documentation, or decoded pack. The CLI's explicit version `2` uses the generic-aware uncached
workspace compiler and v2 snapshot/map/currentness preparation. Version `1` is the unchanged default.
Missing, duplicate, or noncanonical versions are invocation errors before workspace loading.

The canonical envelope and logical pack field order follow Context Pack v1. Compatibility adds
optional `generic_interface_hash` after `interface_hash`, retaining the original generic source
interface separately from the compiled executable interface. Overview retains v2 package counts.
Show/around nodes keep the original declaration signature and a separate top-level
`type_parameters` list of its binder names. The existing snapshot declaration JSON remains
unchanged; this explicit query field is included in selection-byte accounting. Separate
`specializations` contain concrete argument bindings and observed function facts. A template never
acquires universal executable `function` facts merely because an implementation exists. Relations
preserve source/target implementation IDs when present.

Template-body uncertainty, omitted relations, depth stops, query budgets, usage, and unjoined
dependency/incoming package facts remain explicit. `around` is a source-declaration neighborhood,
not a concrete implementation path. Cross-package calls can be queried separately through call
catalog v1; this pack does not consume that join or upgrade local completeness.

## Integrity and bounds

```text
pack version:       fairyc.context-pack/v2
encoding:           fairyc.context-pack-json/v2
checksum algorithm: fairyc.context-pack-checksum/v2-sha256
checksum prefix:    fyctx2-sha256-
```

SHA-256 covers ASCII `fairyc.context-pack-checksum`, bytes `00 02`, then the pack version, encoding,
and canonical logical pack JSON. Each of the last three byte strings is preceded by its unsigned
64-bit big-endian byte length. `ParseChecksumV2` accepts only the v2 prefix and 64 lowercase hex
digits; `ChecksumV2.String` preserves the prefix. V1 parsing rejects v2 checksums. Integrity is not
authenticity, compiler trust, or authority to execute content.

The existing record/relation/depth/selection-byte budgets and independent final-output byte limit
cover generic metadata too. Output overflow returns no bytes. Identical inputs produce identical
compact JSON with no newline; empty arrays and inactive `null` fact variants retain v1 spelling.

## Text and command boundaries

Documentation, source excerpts, and warnings remain empty arrays. Generic source bodies, comments,
host paths, model text, instructions, and permissions do not enter the pack. The `--pack-version`
flag is scoped to `atlas-pack`; the shared relation parser does not enable it on unversioned
Evidence Bundle commands. The separate source/documentation CLI now handles original generic
declarations as unchanged attributed-text v0; generic Evidence Bundle integration remains separate.
Lock refresh/no-write behavior, source and workspace errors, bounded selection, and output-writer
failure handling retain existing CLI policies. No map store, compiler cache, or pack file is
published automatically.
