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
- `project_manifest.fy` consumes the verified document, checks closed fields,
  required fields and the decoded-value rules, and retains a read-only manifest
  with dependency bindings sorted by a bounded merge sort.

The document model keeps mutable columns in the caller until `publish` checks
them and moves them into an immutable `Tree`. This follows the language's
resource-field borrowing boundary. Scratch columns must originate from the
builder APIs; `publish` is the validation boundary for forged column data.

Table states represent implicit parents, dotted definitions, explicit
declarations and closed inline tables. The syntax parser owns when each state
is created and whether a source declaration is legal. Closing an inline table
also seals descendants. The model alone does not establish TOML syntax parity.

Publication checks exact byte-pool spans, UTF-8, exact boolean spelling, parent-before-child ordering,
sibling/child counts, key uniqueness, depth, table-array element types, closed
descendants and the complete hash index. Failed publication returns no tree.
The private source and decoded pools are capped at 1 MiB; node depth is capped
at 512. Builder work and publication collision checks each have a 16 Mi work
budget. `locate` meters key hashing, collision-chain visits and matching key
byte comparisons; syntax must use this lookup while building. Immutable
`lookup` walks at most the published node count. These are candidate limits,
not additional manifest schema fields.

## Document syntax

`project_toml_document.fy` connects quoted/bare/dotted keys, assignments,
strings, ordered arrays, inline tables and ordinary/array table headers to the
model. It retains byte spans, checks full-source UTF-8 and exact statement
terminators, and seals inline descendants. It is private syntax for manifest
value kinds, not a general TOML decoder.

The document receipt is `evidence/candidate-toml-document-20261011.json`. Source
check passes 298 files and 3788 MIR functions; three native tests and both
candidate-generated C profiles execute 39 authored fixed inputs plus the
over-limit input. Go agrees on 32 fixed inputs. The seven differences are
recorded individually: three use numeric/bool/date values outside this
component, and four extend inline/dotted/already-declared tables that this
component rejects under TOML 1.1 table rules. The pinned Go decoder accepts
those four. Neither full TOML parity nor manifest-parser parity is claimed.
The public Go parser and executable manifest contract remain unchanged.

Typed boolean nodes were subsequently added so a syntactically valid boolean
in an unknown field reaches the closed-field check. This preserves the Go
`unknown-field` reason in the existing `mystery = true` fixture. Booleans cannot
fill an admitted manifest field; known fields reject the mismatched type.
Numeric/date values still require future syntax/category reconciliation.

The declaration and inline-table rules are specified at
<https://toml.io/en/v1.1.0#table> and
<https://toml.io/en/v1.1.0#inline-table>. Resolve the pinned-decoder compatibility
boundary before promotion; keep the recorded differences observable.

## Manifest field validation

`project_manifest.fy` rejects unknown root/package/workspace/dependency fields,
proves required fields, checks exact string/array/table types and applies the
existing UID/name/version/language/path and sorted-unique-list validators.
Dependencies use a bounded merge sort and are returned in binding-name order.
The owned result exposes borrowed package/member/dependency/feature bytes after
publication; error results retain reason, field ID and bounded original spans.
Final public diagnostic rendering and complete field-path text remain later work.

`evidence/candidate-manifest-20261011.json` records 58 Go manifest comparisons
(9 accepted and 49 rejected), including all eight existing contract fixtures.
Accepted results match all five package fields, workspace presence and members,
dependency binding/path/version values and features. Rejected results match
reason categories. The corpus includes 130 reverse-declared dependencies to
exercise index rebuilding and merge-sort order. Two native tests and both
candidate-generated C profiles pass. Source check passes 299 files and 3815 MIR
functions. Validation lookup, shape/list processing and sorting share a private
16 Mi work budget, and failed validation publishes no partial manifest.

The typed-boolean document regression is separately recorded in
`evidence/candidate-toml-document-boolean-20261011.json`: three native tests and
both C profiles pass on 41 fixed syntax inputs plus the over-limit input. Go
agrees on 35, with four table-definition policy differences and two
numeric/date value-kind differences retained. The earlier document/model
receipts above are historical snapshots of their respective implementation
commits. Complete pinned-decoder compatibility remains unfinished.

No public API, SDK ABI, DB, hygiene, CI or runner contract changed. Linux, full
suite and Stage 2 were skipped for these component slices; no push occurred.

## Typed scalar rejection

`project_toml_scalars.fy` classifies integer, float and date/time atoms and the
document/model retain their exact raw bytes. This allows valid unknown scalar
fields to reach `unknown-field`, while invalid syntax/range and wrong known
field types remain parse failures. Integer range is signed 64-bit; finite float
overflow uses the exact binary64 rounding threshold. Signed zero, underflow and
explicit inf/nan are retained as valid tokens. Date/time validation checks the
calendar, leap years, optional seconds, fractions and the pinned Go offset
bounds. Neither floating-point nor date types were added to Fairy or manifest
fields.

`evidence/candidate-toml-scalars-20261011.json` records 372 Go classifications
(170 invalid, 17 integer, 35 float and 150 date/time). Native 2 and candidate C
development/release pass. `evidence/candidate-manifest-scalars-20261011.json`
records 104 complete manifest acceptance/reason/value comparisons through
native 2 and both C profiles. `evidence/candidate-toml-document-scalars-20261011.json`
records the document native 3 and both C profiles: 42 fixed inputs, 38 Go
matches and the four remaining table-definition differences. Source check
passes 300 files and 3826 MIR functions. These receipts supersede the earlier
scalar-exclusion observations, which remain historical evidence.

## Replacement arena foundation

The pinned decoder sometimes replaces arrays with tables during dotted context
establishment. `replace_table` preserves arena IDs, marks superseded children
as scratch tombstones and clears the owner's child/value links. Metered lookup
skips those tombstones. `publish_compacted` rebuilds live nodes, byte pools and
indexes before invoking the ordinary publication verifier. Ordinary `publish`
still rejects tombstones and retains its forged-graph checks; the compacting
entry point is for trusted parser scratch. Parser metadata integration is next.

`evidence/candidate-toml-replacement-20261011.json` records three native cases
and both candidate C profiles: repeated replacement drops old key/value bytes,
old keys cannot be found, strict publication rejects discarded scratch, and a
50-level branch is removed before its later sibling. The existing document
model's three native cases and both C profiles also pass in
`evidence/candidate-toml-model-replacement-regression-20261011.json`. Source
check passes 300 files and 3829 MIR functions.

`candidate-toml-table-contexts-v1.json` captures 111 context transitions for the
next parser slice. The pinned Go oracle currently accepts 49; no Fairy parity
claim is made for that corpus yet. Logical implicit/type metadata must follow
the decoder across dotted paths, inline values and array-table entries.

## Logical decoder context foundation

`project_toml_context` interns decoded key components into bounded logical paths.
Physical array items contribute no component, so repeated array-table entries
and replacement tables share implicit/type metadata. Quoted dots and empty
components keep their distinct mapping identities. The pinned decoder's empty
assignment-key type behavior is represented separately from mapping identity.
`define` consumes implicit metadata once, distinguishing retained tables from
Array-typed replacement. Missing physical entries retain global implicit flags.

Mutable name, entry and index owners stay separate. Their allocation, collision
lookup and physical ancestor traversal share the document model's work budget;
key bytes, entries and depth have explicit bounds. Three native tests and both
candidate C profiles pass, including physical entry sharing, replacement,
quoted/empty component identity, collision chains and depth/work exhaustion.
Source check passes 301 files and 3849 MIR functions. Evidence is
`evidence/candidate-toml-context-20261011.json`. The 111 Go table observations
were refreshed (49 accepted); parser parity remains pending integration.

## Dotted context and table declaration integration

The document parser now uses logical implicit/type metadata for dotted prefixes
and table headers. A first dotted definition can be promoted to an explicit
table; later dotted context establishment consumes that implicit flag. Dotted
paths can enter already declared tables and the latest array-table entry.
Array-typed prefixes are replaced with tables, with old children discarded and
live-only pools rebuilt at publication. Unchanged documents use ordinary
publication directly.

Native assertions check retained values when tables are promoted, discarded
array and prior-table contents on repeated replacement, and shared implicit
metadata across physical array-table entries. The document corpus now has 48
inputs: 46 Go acceptance matches and two recorded inline-table differences.
This reduces the earlier four differences in that authored corpus to two;
it does not claim complete parity for the separate 111-context corpus. Ignored
implicit-table assignments and inline-table decoder behavior remain to connect.
The 104-case manifest acceptance/reason/full-value corpus still matches Go.

Evidence is `evidence/candidate-toml-document-context-20261011.json` and
`evidence/candidate-manifest-context-20261011.json`. Three document and two
manifest native tests and both candidate C profiles pass. Source check passes
301 files and 3850 MIR functions. The private comparison runner's known-difference
set and authored corpus were updated; no public command or CI workflow changed.

## Separate decoder mapping and literal values

`project_toml_values` adds bounded value handles and independently indexed map
slots. Decoder-global mappings and returned inline-table mappings can refer to
the same literal; overwriting a global slot does not mutate the returned map.
Publication projects only root-reachable values into the existing strict model,
duplicates shared values and excludes unreachable scratch names and payloads.
This models the pinned decoder's inline-table return behavior without changing
its contract. Parser integration follows this foundation.

Three native tests and both candidate C profiles pass for mapping replacement,
shared-value projection, empty keys, array-table ordering, index rebuilds,
invalid references and cyclic scratch depth exhaustion. Source check passes
302 files and 3885 MIR functions. Evidence:
`evidence/candidate-toml-values-20261011.json`.

## Decoder mapping integration

The parser now keeps the decoder-global mapping distinct from returned literal
values. Inline tables establish the global context, update it through the
implicit/type rules, and independently populate their returned map. Arrays set
type metadata before reading elements. Final assignments consume global implicit
flags after the entire literal is read, retaining or replacing the mapping just
as the pinned decoder does. Only reachable values are projected into the strict
immutable document model; ignored literals and temporary contexts are omitted.
The string and scalar codecs retain their existing bounds and syntax checks.

The 56 authored document inputs all match Go acceptance, with zero remaining
recorded differences in that corpus. Native value assertions cover inline-table
extensions, ignored scalar assignments, array replacement, the different
`a={b.c=1,b=2}` and `a=[{b.c=1,b=2}]` results, and header-component source spans.
An empty key inside an inline array element can change the decoder's metadata
and retain its temporary array-table mapping. The former test assumed an
ordinary two-element array; the actual Go output has one table, and the native
assertions now require that observed result. Inline maps remain extensible under
the pinned decoder rather than being sealed by the candidate parser.

The separate 111-transition corpus also matches Go acceptance (49 accepted,
62 rejected). `tools/verify-candidate-toml-contexts.py` regenerates its binary
oracle and checks one authored native test and both candidate C profiles. This
corpus proves acceptance, not all generic decoded values. The manifest corpus's
104 acceptance/reason/full-value comparisons still pass. Source check passes
302 files and 3892 MIR functions. Document native 3, table-context native 1,
manifest native 2, and every associated candidate C development/release run pass.

Current evidence: `evidence/candidate-toml-document-decoder-20261011.json`,
`evidence/candidate-toml-contexts-decoder-20261011.json`, and
`evidence/candidate-manifest-decoder-20261011.json`. Earlier receipts and their
known differences remain historical. Complete TOML parity is not inferred from
these finite corpora. No public CLI, SDK ABI, DB or CI workflow changed.

## Remaining connection

1. Keep the pinned decoder comparison coverage with parser changes; complete
   TOML parity remains unclaimed beyond the observed corpora.
2. Connect the validated result to contained source discovery and the later
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
