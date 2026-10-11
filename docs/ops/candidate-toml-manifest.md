# Candidate manifest implementation

This tracks the private Fairy implementation of work item 65. The executable
manifest contract remains `docs/package/project-manifest-v1.md` and the reference
implementation remains `internal/projectmanifest/`. Filesystem discovery,
package graph resolution, locks and project CLI execution belong to later slices.

## Implemented components

- `project_manifest_values.fy` validates decoded package names, UIDs, exact
  versions, language revisions, portable paths and sorted unique lists.
- `project_toml_strings.fy` decodes quoted keys and single/multiline basic and
  literal string tokens. Its Go-reference corpus verifies decoded bytes, cursor
  consumption and rollback on failed decoding.
- `project_toml_model.fy` builds a document graph with separate name/value byte
  pools, nodes and a parent-key hash index. Key components stay distinct: a
  quoted `"a.b"` is one component, whereas a dotted path has separate `a`/`b`
  components. Arrays retain insertion order; arrays of tables contain tables.

The document model keeps mutable columns in the caller until `publish` checks
them and moves them into an immutable `Tree`. This follows the language's
resource-field borrowing boundary. Scratch columns must originate from the
builder APIs; `publish` is the validation boundary for forged column data.

Table states represent implicit parents, dotted definitions, explicit
declarations and closed inline tables. The syntax parser owns when each state
is created and whether a source declaration is legal. Closing an inline table
also seals descendants. The model alone does not establish TOML syntax parity.

Publication checks exact byte-pool spans, UTF-8, parent-before-child ordering,
sibling/child counts, key uniqueness, depth, table-array element types, closed
descendants and the complete hash index. Failed publication returns no tree.
The private source and decoded pools are capped at 1 MiB; node depth is capped
at 512. Builder work and publication collision checks each have a 16 Mi work
budget. `locate` meters key hashing, collision-chain visits and matching key
byte comparisons; syntax must use this lookup while building. Immutable
`lookup` walks at most the published node count. These are candidate limits,
not additional manifest schema fields.

## Remaining connection

1. Parse dotted/quoted paths, assignments, arrays, inline tables and table
   headers into the model, retaining original byte spans.
2. Enforce exact root/package/workspace/dependency fields and required fields,
   then apply decoded-value validators and deterministic dependency ordering.
3. Compare complete documents against the pinned Go decoder and manifest
   parser. Record decoder/spec disagreements explicitly before claiming parity.
4. Connect the validated result to contained source discovery and the later
   package/lock/CLI stages.

The value and string receipts are
`evidence/candidate-manifest-values-20261011.json` and
`evidence/candidate-toml-strings-20261011.json`. The document model receipt is
`evidence/candidate-toml-model-20261011.json`: source check passes 297 files and
3771 MIR functions; three authored native tests pass, and the same test bodies
execute through candidate-generated C under development and optimized release.
The native cases exercise 150 indexed keys, table/array ordering, descendant
sealing, 16 forged column mutations, invalid table-array contents, and the
source/depth/decoded-byte/metered-lookup work boundaries. This does not claim
complete-document Go decoder parity. Linux, full suite and Stage 2 were not run
for this component slice. Public API, SDK ABI, DB, hygiene and CI are unchanged.
