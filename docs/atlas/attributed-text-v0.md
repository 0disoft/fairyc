# Atlas attributed text v0 policy

- Status: Accepted design; no implementation, encoding, or CLI
- Logical policy: `fairyc.attributed-text/v0`
- Decision: `docs/adr/0195-attributed-text-boundary-v0.md`
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

The first implementation must produce a separately versioned attributed-text value. A future
composite Context Pack requires a new pack version and an explicit decision about embedding versus
checksum-linking the companion artifact. This policy does not reserve that version, wire encoding,
or command spelling.

## Admitted content

The logical `kind` set is closed:

- `source-excerpt`: exact validated Fairy C source bytes for one compiler-selected declaration span;
- `declaration-documentation`: exact documentation-comment bytes attached to one declaration by a
  future frontend-owned attachment contract.

Ordinary implementation comments, repository Markdown, issue text, build logs, generated summaries,
model output, URLs, environment values, and arbitrary caller strings are not admitted. The current
lexer retains comments only as undifferentiated trivia, so `declaration-documentation` remains
unavailable until the grammar/frontend defines exact attachment, detachment, and source-span rules.
The first executable slice may therefore implement `source-excerpt` only without pretending that
all trivia is documentation.

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
wire encoder additionally requires its own positive final-output byte limit.

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

## First implementation gate

The first code change after this policy is a small internal attributed-text model and validator,
followed by workspace-authored exact `source-excerpt` extraction from retained validated bytes. Its
test ladder must separately prove:

1. local model validation, strict enums, byte ranges, scalar boundaries, checksum vectors, deep
   detachment, and canonical ordering;
2. pipeline linkage from exact workspace/query identities to one source slice, including CRLF, a
   leading BOM outside the selected span, non-ASCII text, absent final newline, and visible/full SymbolId
   equivalence;
3. stale-byte, wrong-package, wrong-file, wrong-line-map, invalid-span, invalid-UTF-8, disallowed-
   control, and post-validation mutation rejection;
4. count, fragment, aggregate, and future final-output budget boundaries with explicit omissions;
5. printable prompt-injection fixtures retained only as `untrusted-text` and unable to change facts,
   completeness, command options, output destinations, or host authority; and
6. default compiler, `.fymap` v1, Context Pack v1, build, cache, and native output byte identity when
   attributed text is not explicitly requested.

Documentation attachment, dependency text, generated text, persistent text storage, a decoder,
public CLI, composite pack, token budgeting, anchors, structural editing, LSP, SCIP, MCP, model calls,
embeddings, and permissions remain outside that first implementation.
