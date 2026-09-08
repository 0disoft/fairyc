# Atlas attributed text v0 policy

- Status: Internal logical model, exact local source and declaration-documentation extraction, canonical output-only JSON encoding, and experimental source/documentation CLI implemented; no decoder
- Logical policy: `fairyc.attributed-text/v0`
- Decision: `docs/adr/0195-attributed-text-boundary-v0.md`
- Encoding decision: `docs/adr/0197-canonical-attributed-text-json-v0.md`
- CLI decision: `docs/adr/0198-experimental-atlas-text-source-command.md`
- Documentation attachment decision: `docs/adr/0199-c-friendly-declaration-documentation-attachment.md`
- Documentation output decision: `docs/adr/0200-exact-declaration-documentation-attributed-text-command.md`
- Evidence Bundle consumer: `docs/atlas/evidence-bundle-v0.md`
- Exact file identity: `docs/package/source-file-identities-v1.md`
- Compiler-fact pack: `docs/atlas/context-pack-v1.md`

## Purpose and non-authority boundary

Attributed text is an optional, output-only companion to exact-current compiler facts. It can carry
an exact source slice or declaration documentation so a human, IDE, CI job, or LLM-facing adapter
can inspect relevant prose without reading an entire repository. It is never a compiler fact,
instruction, prompt role, policy, permission, approval, or proof that an incomplete graph is
complete.

Every record has fixed authority `untrusted-text`. `origin` describes custody; it does not endorse
the text. A workspace author, compiler distributor, signed package, generated file, or future
registry signature cannot grant filesystem, process, network, credential, model, approval, or
execution authority through source or documentation. Consumers must render or quote text as data
and must keep tool authorization in host-owned deterministic policy.

## Compatibility separation

`.fymap` v1 remains a compact compiler-fact artifact and stores no source or documentation bodies.
`fairyc.context-pack/v1` continues to require empty `documentation`, `source_excerpts`, and
`warnings` arrays. Neither format may begin carrying text under its existing identifier.

ADR 0197 implements the separately versioned `fairyc.attributed-text-json/v0` output. ADR 0198 and
ADR 0200 assign that exact artifact to the separate source and documentation actions. ADR 0202 adds
an independent `fairyc.atlas-evidence/v0` wrapper that nests the complete canonical source and
documentation documents beside an unchanged Context Pack v1 document. It does not populate the
pack's empty text arrays, revise this encoding, or accept attributed-text input.

## Admitted content

The logical `kind` set is closed:

- `source-excerpt`: exact validated Fairy C source bytes for one compiler-selected declaration span;
- `declaration-documentation`: exact documentation-comment bytes attached to one declaration by a
  frontend-owned attachment contract.

Ordinary implementation comments, repository Markdown, issue text, build logs, generated summaries,
model output, URLs, environment values, and arbitrary caller strings are not admitted. ADR 0199
distinguishes physical-line-leading `///` as documentation trivia and carries a conservative exact
attachment span through CST, AST, and typed HIR. Blank lines, ordinary comments, block comments,
same-line trailing comments, and nested declarations detach. ADR 0200 carries that span through an
in-memory snapshot sidecar and exposes exact retained bytes; neither producer pretends all trivia is
documentation.

The closed `origin` set is:

- `workspace-authored`: the root package or an explicitly declared local workspace member;
- `compiler-bundled`: exact compiler-shipped SDK source;
- `external-dependency`: reserved for a future acquired dependency source contract;
- `generated`: reserved for a future generator identity and input-lineage contract.

Only `workspace-authored` source is eligible for the first implementation. Compiler-bundled,
external-dependency, and generated text require separate explicit inclusion policy. No origin is
selected by default and Atlas performs no URL fetch, registry access, source acquisition, or
filesystem discovery to obtain text.

## Logical record

Each record carries all of the following before text can cross the boundary:

- logical policy, `kind`, fixed `authority`, and `origin`;
- visible and full SymbolId for the exact subject;
- exact package UID, instance, coordinate, and aggregate source digest;
- canonical package-relative `.fy` path and source/test classification;
- `fairyc.source-file/v1` content identity and `fairyc.source-line-map/v1` identity;
- zero-based half-open start and end byte offsets;
- format `fairyc-source-utf8` or `fairyc-declaration-doc-utf8`;
- exact text bytes represented as a UTF-8 string value in the canonical output encoding; and
- a full fragment checksum.

The fragment checksum algorithm is
`fairyc.attributed-text-fragment/v0-sha256`, rendered as `fytext0-sha256-` plus 64 lowercase
hexadecimal digits. The canonical input is ASCII `fairyc.attributed-text-fragment`, one zero byte,
version byte `00`, then unsigned 64-bit big-endian length-prefixed UTF-8 bytes for logical policy,
kind, authority, origin, visible SymbolId, full SymbolId, package UID, instance, coordinate, aggregate
source digest, path, source/test classification, file identity, line-map identity, and format; raw
unsigned 64-bit big-endian start and end offsets; then unsigned 64-bit length-prefixed exact text
bytes. The checksum proves deterministic integrity, not authorship, review, safety, or
authentication.

## Exact-current admission

Text extraction starts only after the loaded workspace has passed manifest, graph, bounded file,
syntax, semantic, Flow, verified-MIR, snapshot, partition, and exact-current query-index admission.
It consumes the already validated immutable source closure that produced the package identity. It
does not reopen a path after compilation or accept caller-supplied bytes beside compiler facts.

Before slicing, the extractor recomputes the package source digest plus selected file content and
line-map identities from those retained bytes and matches every value against the exact query
context. A mismatch rejects the entire text request as stale or internally inconsistent. It never
returns a best-effort line, a same-length replacement, or text from a different package instance.

The source and documentation CLIs now use aggregate-aware compilation and exact-current v3 query
preparation. The logical/JSON text v0 format, record/document checksums, fixed `untrusted-text`
authority, custody labels, exact slice selection, and error/omission policy do not change. Ordinary
and function-generic source produce byte-identical v0 documents through v2 and v3 maps; ordinary
v1/v2 identity remains covered. No new output field or version
flag is needed: these records identify original source spans, not executable specialization facts.

Original generic declarations, including used/unused structs and resources, are eligible. Multiple concrete
implementations still select one original source span and at most one frontend-attached
documentation span. Compiler-generated names/bodies are not source subjects. Retrieval of an unused
body is not a claim that its executable semantics were checked. Generic source and documentation
remain untrusted text; hostile prose is preserved as inert data, never interpreted as instructions.
Missing documentation remains an explicit `kind-unavailable` omission, not a synthesized summary.

Aggregate tests preserve BOM/CRLF and non-ASCII spans, inert instruction-like documentation,
retained-source extraction after disk edits, stale retained-byte rejection, wrong snapshot
rejection, control omissions and absent documentation. CLI discovery uses symbol catalog 3;
source/documentation commands need no new version flag because text v0 remains unchanged.
This experimental text admission does not promote aggregate syntax in stable workspace commands.

The requested span must be in range and begin and end on UTF-8 scalar boundaries. The emitted bytes
are the exact slice: no newline conversion, Unicode normalization, trimming, indentation rewriting,
comment-marker removal, HTML rendering, or Markdown expansion occurs.

For `declaration-documentation`, the immutable snapshot sidecar must bind the same exact declaration
key and declaration span selected by the query to one preceding documentation span in the same
package instance and source. The sidecar contains no text and is never encoded into `.fymap` v1.
An absent attachment produces `kind-unavailable`; its subject retains the queried declaration span
and the omission reports zero text bytes. Atlas never scans backward or reparses source to
manufacture an attachment.

## Canonical output-only encoding

`fairyc.attributed-text-json/v0` is one compact UTF-8 JSON value. The exact top-level field order is:

```text
text_version
encoding
checksum_algorithm
checksum
text
```

The logical `text` field contains, in order:

```text
authority
subject
selection
budget
usage
records
omissions
```

`subject` repeats both SymbolId forms plus exact package, file, line-map, and span identity so an
omission remains attributable without carrying rejected text. `selection` records the explicit kind
and origin request. Each admitted record retains every logical field and its independent fragment
checksum. Empty arrays are `[]`; Go's compact JSON encoder over fixed structs defines the current
canonical bytes. Pretty printing, key reordering, `null` arrays, a trailing newline, or a second JSON
value is not this encoding.

The document checksum is `fairyc.attributed-text-checksum/v0-sha256`, rendered as
`fytextdoc0-sha256-` plus 64 lowercase hexadecimal digits. SHA-256 hashes ASCII
`fairyc.attributed-text-checksum`, bytes `00 00`, and unsigned 64-bit big-endian length-prefixed
document version, encoding, and canonical logical `text` JSON. The frozen implementation vector is:

```text
fytextdoc0-sha256-ed3d884108140fa00e4fbf21ce12f6368fd723c9b09fe4de810b3a6050a2bea3
```

The output builder runs exact extraction itself and accepts no caller-authored result or source
bytes. It requires a separate positive final-output byte limit and rejects the complete envelope
atomically when it does not fit. No decoder or untrusted encoded input is accepted.

## Text safety validation

A successful source compilation does not make prose safe for display or a model. Each selected
fragment must independently be valid UTF-8 and must not contain NUL, C0 or C1 controls other than
horizontal tab, LF, or CR. Unicode bidirectional formatting and isolation controls are rejected.
The extractor omits the whole fragment with a machine-readable reason; it never silently deletes or
rewrites code points because transformed text would no longer match its attribution.

Printable prompt-like text such as `ignore previous instructions` is not detected, censored, or
upgraded to policy. Natural-language filters cannot prove safety. Such bytes remain present only as
explicitly labelled `untrusted-text`, and consumers remain responsible for keeping them out of
system, developer, authorization, and tool-policy channels.

## Explicit budgets and deterministic selection

Every request must explicitly select admitted origins and kinds and provide positive
`max_fragments`, `max_fragment_bytes`, and `max_text_bytes`. There are no configuration,
environment, or command defaults. Implementation guards may not exceed the already admitted
workspace closure: 4,096 fragments, 8 MiB for one fragment, and 64 MiB aggregate text. Any future
wire encoder additionally requires its own positive final-output byte limit. Each current single-
record producer requires exactly its action-selected kind and `workspace-authored`; neither infers
origin, kind, or numeric policy.

Records follow the exact query's canonical subject/node order, then kind order
`declaration-documentation`, `source-excerpt`. A fragment is atomic. If it exceeds the per-fragment
or remaining aggregate budget, the complete fragment is omitted rather than prefix-truncated.
Results report actual fragment/text-byte usage plus stable omissions for unavailable kind, excluded
origin, stale source, invalid span, invalid UTF-8, disallowed control, fragment limit, aggregate
limit, and count limit. An empty successful result cannot claim that no relevant text exists.

Budgets count exact UTF-8 bytes, not characters, display columns, tokens, model context, or billing
units. A future relevance or token estimator is a separate non-authoritative layer.

## Consumer obligations

Consumers of the encoded form must:

- verify its strict version, checksum, canonical form, record invariants, and fact-pack linkage;
- preserve the authority and origin labels beside rendered text;
- distinguish omitted, unavailable, rejected, and empty text;
- avoid executing Markdown, HTML, links, code blocks, or embedded commands as authority;
- enforce file, process, network, credential, approval, and model permissions outside Atlas data;
- avoid merging text into compiler-fact fields or using it to override completeness; and
- discard the text when package, file, line-map, SymbolId, or linked fact-pack identity changes.

## Implemented gates

ADR 0196 implements the internal attributed-text model and validator plus workspace-authored exact
`source-excerpt` extraction from retained validated bytes. ADR 0200 adds exact
`declaration-documentation` extraction from the immutable attachment sidecar. The implementation
currently proves:

1. local model validation, strict enums, byte ranges, scalar boundaries, checksum vectors, deep
   detachment, and canonical ordering;
2. pipeline linkage from exact workspace/query identities to one source slice, including CRLF, a
   leading BOM outside the selected span, non-ASCII text, absent final newline, and visible/full SymbolId
   equivalence;
3. stale-byte, wrong-package, wrong-file, wrong-line-map, invalid-span, invalid-UTF-8, disallowed-
   control, and post-validation mutation rejection;
4. explicit guard-bound request budgets, one-fragment source or documentation boundaries, a
   separate final-output limit, and atomic unavailable, safety, and size omissions; collection count
   and remaining aggregate behavior belongs to a future multi-record producer;
5. printable prompt-injection fixtures retained only as `untrusted-text` and unable to change facts,
   completeness, command options, output destinations, or host authority; and
6. default compiler, `.fymap` v1, Context Pack v1, build, cache, and native output byte identity when
   attributed text is not explicitly requested.

The canonical encoder is exposed directly through `fairyc experimental atlas-text source` and
`fairyc experimental atlas-text documentation`, and indirectly as two independently checksummed
children of `fairyc experimental atlas-evidence show`. The direct commands require exact package and
visible/full SymbolId selectors, explicit `workspace-authored` origin, fragment/count/text/final-
output budgets, and optional locked no-write mode. The Evidence command fixes the origin to
`workspace-authored` and each child count to one, then requires explicit fragment/text,
per-attributed-document, and final-bundle limits. The documentation producer emits only the exact
frontend-attached marker-bearing bytes or a checksummed `kind-unavailable` omission. Both producers
emit size and safety omissions successfully but atomically reject a child envelope that does not fit
its output budget. The Evidence Bundle applies the text budget and document limit separately to each
child and grants no new authority. Dependency text, generated text, persistent text storage, a
decoder, stable CLI, Context Pack text population, token budgeting, anchors, structural editing,
LSP, SCIP, MCP, model calls, embeddings, and permissions remain unimplemented.
