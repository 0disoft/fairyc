# Atlas attributed text v0 policy

- Status: Internal logical model, exact local source-excerpt extraction, canonical output-only JSON encoding, and experimental source CLI implemented; no decoder
- Logical policy: `fairyc.attributed-text/v0`
- Decision: `docs/adr/0195-attributed-text-boundary-v0.md`
- Encoding decision: `docs/adr/0197-canonical-attributed-text-json-v0.md`
- CLI decision: `docs/adr/0198-experimental-atlas-text-source-command.md`
- Documentation attachment decision: `docs/adr/0199-c-friendly-declaration-documentation-attachment.md`
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

ADR 0197 implements the separately versioned `fairyc.attributed-text-json/v0` output. ADR 0198
assigns that exact artifact only to `experimental atlas-text source`; it does not reserve a
composite version. A future composite Context Pack still requires a new pack version and an explicit
decision about embedding versus checksum-linking this companion artifact.

## Admitted content

The logical `kind` set is closed:

- `source-excerpt`: exact validated Fairy C source bytes for one compiler-selected declaration span;
- `declaration-documentation`: exact documentation-comment bytes attached to one declaration by a
  frontend-owned attachment contract.

Ordinary implementation comments, repository Markdown, issue text, build logs, generated summaries,
model output, URLs, environment values, and arbitrary caller strings are not admitted. ADR 0199 now
distinguishes physical-line-leading `///` as documentation trivia and carries a conservative exact
attachment span through CST, AST, and typed HIR. Blank lines, ordinary comments, block comments,
same-line trailing comments, and nested declarations detach. Attributed-text extraction and CLI
selection for this kind remain unavailable; the existing source producer still selects only
`source-excerpt` and never pretends all trivia is documentation.

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
- format `fairyc-source-utf8` or the future `fairyc-declaration-doc-utf8`;
- exact text bytes represented as a UTF-8 string value in any future encoding; and
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

Text extraction starts only after the ordinary workspace has passed manifest, graph, bounded file,
syntax, semantic, Flow, verified-MIR, snapshot, partition, and exact-current query-index admission.
It consumes the already validated immutable source closure that produced the package identity. It
does not reopen a path after compilation or accept caller-supplied bytes beside compiler facts.

Before slicing, the extractor recomputes the package source digest plus selected file content and
line-map identities from those retained bytes and matches every value against the exact query
context. A mismatch rejects the entire text request as stale or internally inconsistent. It never
returns a best-effort line, a same-length replacement, or text from a different package instance.

The requested span must be in range and begin and end on UTF-8 scalar boundaries. The emitted bytes
are the exact slice: no newline conversion, Unicode normalization, trimming, indentation rewriting,
comment-marker removal, HTML rendering, or Markdown expansion occurs.

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
wire encoder additionally requires its own positive final-output byte limit. The current source-
only producer requires exactly `source-excerpt` and `workspace-authored`; it does not infer them.

Records follow the exact query's canonical subject/node order, then kind order
`declaration-documentation`, `source-excerpt`. A fragment is atomic. If it exceeds the per-fragment
or remaining aggregate budget, the complete fragment is omitted rather than prefix-truncated.
Results report actual fragment/text-byte usage plus stable omissions for unavailable kind, excluded
origin, stale source, invalid span, invalid UTF-8, disallowed control, fragment limit, aggregate
limit, and count limit. An empty successful result cannot claim that no relevant text exists.

Budgets count exact UTF-8 bytes, not characters, display columns, tokens, model context, or billing
units. A future relevance or token estimator is a separate non-authoritative layer.

## Consumer obligations

Consumers of a future encoded form must:

- verify its strict version, checksum, canonical form, record invariants, and fact-pack linkage;
- preserve the authority and origin labels beside rendered text;
- distinguish omitted, unavailable, rejected, and empty text;
- avoid executing Markdown, HTML, links, code blocks, or embedded commands as authority;
- enforce file, process, network, credential, approval, and model permissions outside Atlas data;
- avoid merging text into compiler-fact fields or using it to override completeness; and
- discard the text when package, file, line-map, SymbolId, or linked fact-pack identity changes.

## Implemented first gate

ADR 0196 implements the internal attributed-text model and validator plus workspace-authored exact
`source-excerpt` extraction from retained validated bytes. The implementation currently proves:

1. local model validation, strict enums, byte ranges, scalar boundaries, checksum vectors, deep
   detachment, and canonical ordering;
2. pipeline linkage from exact workspace/query identities to one source slice, including CRLF, a
   leading BOM outside the selected span, non-ASCII text, absent final newline, and visible/full SymbolId
   equivalence;
3. stale-byte, wrong-package, wrong-file, wrong-line-map, invalid-span, invalid-UTF-8, disallowed-
   control, and post-validation mutation rejection;
4. explicit guard-bound request budgets, the one-fragment source-excerpt byte boundary, a separate
   final-output limit, and atomic omissions; collection count and remaining aggregate behavior
   belongs to a future multi-record producer;
5. printable prompt-injection fixtures retained only as `untrusted-text` and unable to change facts,
   completeness, command options, output destinations, or host authority; and
6. default compiler, `.fymap` v1, Context Pack v1, build, cache, and native output byte identity when
   attributed text is not explicitly requested.

The canonical encoder is exposed only through `fairyc experimental atlas-text source`. That command
requires exact package and visible/full SymbolId selectors, `workspace-authored` origin, explicit
fragment/count/text/final-output budgets, and optional locked no-write mode. It emits a checksummed
omission successfully when the exact source does not fit the text budget, but atomically rejects a
final envelope that does not fit the output budget. Documentation attachment, dependency text,
generated text, persistent text storage, a decoder, stable CLI, composite pack, token budgeting,
anchors, structural editing, LSP, SCIP, MCP, model calls, embeddings, and permissions remain
unimplemented.
