# Atlas Evidence Bundle v0

Generic-aware composition is separately available as [Evidence Bundle v1](evidence-bundle-v1.md).
Default output and explicit `--bundle-version 0` retain this v0 contract.

- Status: Implemented output-only single-symbol composer with experimental CLI exposure; no decoder
- Bundle version: `fairyc.atlas-evidence/v0`
- Encoding: `fairyc.atlas-evidence-json/v0`
- Checksum: `fairyc.atlas-evidence-checksum/v0-sha256`
- Implementation: `internal/evidencebundle/`
- Decision: `docs/adr/0202-single-symbol-atlas-evidence-bundle-v0.md`
- Facts component: `docs/atlas/context-pack-v1.md`
- Text components: `docs/atlas/attributed-text-v0.md`

## Purpose and hard boundary

An Evidence Bundle is one deterministic, bounded wrapper around three exact-current outputs for one
exact symbol:

1. compiler facts as a complete Context Pack v1 `show` document;
2. the exact declaration source span as a complete attributed-text v0 document; and
3. exact attached declaration documentation, or an explicit omission, as another complete
   attributed-text v0 document.

It avoids three independent workspace compilations and prevents accidental cross-state composition.
It is not a prompt, proof, signature, authentication result, policy, permission, approval, model
context guarantee, or claim that the package graph is complete. The word “evidence” means only that
the wrapper preserves the owning compiler outputs and their integrity metadata.

## Authority separation

The wrapper repeats the authority class of every component:

| Field | Fixed value | Meaning |
| --- | --- | --- |
| `authority.bundle` | `none` | The wrapper creates no independent authority. |
| `authority.facts` | `compiler-fact` | The child is the compiler-fact-only Context Pack v1 output. |
| `authority.source` | `untrusted-text` | Exact source bytes remain data, never instructions. |
| `authority.documentation` | `untrusted-text` | Exact `///` bytes remain data, never instructions. |

These labels do not authenticate a compiler executable or author. Consumers must keep filesystem,
process, network, credential, approval, and model/tool policy outside all bundle contents. Prompt-like
text is preserved as attributed data and cannot alter facts, limits, completeness, or host policy.

## Construction

The encoder receives one retained validated `workspaceloader.Workspace`, one matching immutable
`semanticsnapshot.Snapshot`, one exact-current `atlasmap.QueryIndex`, one exact visible or full
SymbolId, and explicit relation/text/output limits. It resolves the canonical subject once, then
calls the existing owning encoders:

- `contextpack.EncodeShow`;
- `attributedtext.EncodeWorkspaceSource`; and
- `attributedtext.EncodeWorkspaceDocumentation`.

It accepts no caller JSON, text, path, query result, or checksum. It performs no path reopening,
filesystem discovery, network request, registry fetch, process execution, model call, tokenizer call,
text rendering, or compiler feedback. All three children derive from the same retained workspace and
exact-current identities.

## Canonical envelope

The top-level compact JSON fields occur in this exact order:

```text
bundle_version
encoding
checksum_algorithm
checksum
bundle
```

`bundle` contains, in order:

```text
authority
subject
components
limits
```

`subject` contains canonical visible/full SymbolId, package UID, and exact package instance.
`components` contains `facts`, `source`, and `documentation` as nested JSON values, not escaped JSON
strings. Their bytes are the complete canonical outputs of their owning encoders and retain their own
version, checksum algorithm, checksum, selection, budget, usage, records, and omissions. `limits`
echoes the separate facts, per-text-document, and final bundle byte limits.

Go's compact JSON encoder over fixed structs is the only current representation. Field omission,
reordering, pretty printing, alternate number spelling, a trailing newline, or a second JSON value is
not this encoding.

## Wrapper checksum

The checksum text is `fyevidence0-sha256-` plus 64 lowercase hexadecimal digits. SHA-256 receives:

1. ASCII `fairyc.atlas-evidence-checksum` and bytes `00 00`;
2. unsigned 64-bit big-endian length plus exact bundle-version bytes;
3. unsigned 64-bit big-endian length plus exact encoding bytes; and
4. unsigned 64-bit big-endian length plus the canonical logical `bundle` JSON bytes.

The checksum field itself is excluded. Because the logical document embeds the exact child values,
the wrapper checksum binds all child bytes, child checksums, authority labels, subject identity, and
echoed limits. It detects deterministic corruption; it does not establish signer identity, review,
safety, or authenticity.

## Explicit budgets

The CLI requires every policy value; there are no environment, configuration, or command defaults:

- query: positive `max-records`, `max-relations`, and `max-selection-bytes`;
- text: positive `max-fragment-bytes` and `max-text-bytes`, applied independently to the one source
  fragment and one documentation fragment;
- child documents: positive `max-facts-bytes` for Context Pack v1 and positive
  `max-text-document-bytes`, applied separately to each attributed-text document; and
- wrapper: positive `max-output-bytes` for the complete Evidence Bundle envelope.

The existing implementation guards still apply. Byte limits count exact UTF-8 or JSON bytes, not
characters, lines, display columns, tokens, model context, or billing units. Changing a limit changes
the canonical document because the chosen policy is checksummed output evidence.

## Experimental command

The implemented shape is:

```text
fairyc experimental atlas-evidence show
  --package-uid <uid>
  --symbol <visible-or-full-symbol-id>
  --direction <outgoing|incoming|both>
  --relation <kind> [--relation <kind> ...]
  --max-records <count>
  --max-relations <count>
  --max-selection-bytes <bytes>
  --max-fragment-bytes <bytes>
  --max-text-bytes <bytes>
  --max-facts-bytes <bytes>
  --max-text-document-bytes <bytes>
  --max-output-bytes <bytes>
  [--locked]
```

The command writes one canonical JSON value with no trailing newline. It uses the current manifest
workspace and exact package UID, never a default map store. Ordinary mode may maintain `fairy.lock`;
`--locked` verifies without writing. The spelling and provisional process-code mapping are not stable
CLI promises.

## Omissions and failures

Attributed-text rejection stays inside the relevant child when it is safe to return an attributable
omission. An undocumented declaration produces `kind-unavailable`; a fragment that exceeds its text
budget produces a size omission. Both are successful bundle outputs and do not imply that no text
exists.

Malformed or duplicate policy, an unknown package or exact SymbolId, insufficient query selection,
a child document exceeding its own final bound, a wrapper exceeding its final bound, stale or
contradictory identities, and compiler invariants reject the complete bundle before encoder output.
The encoder never emits a partial JSON value. A downstream writer can still accept a prefix before
returning an I/O error; stream rollback is outside the format.

## Compatibility and unchanged contracts

Evidence Bundle v0 is an independent output family. It neither revises nor decodes its children:

- `.fymap` v1 remains body-free;
- Context Pack v1 still emits empty `documentation`, `source_excerpts`, and `warnings` arrays;
- attributed-text v0 retains its exact source/documentation selection and checksums; and
- Symbol Catalog v1 remains the exact-name/SymbolId discovery path.

Changing the wrapper field meanings/order, JSON spelling, authority labels, subject identity,
component set, limit semantics, checksum input, digest, or text prefix requires an explicit
compatibility decision. Child version changes require a new wrapper acceptance decision; v0 does not
silently translate them.

## Not enabled

- no decoder, untrusted bundle input, persistence, cache, default output path, or signature;
- no `overview`, `around`, multi-symbol, fuzzy-name, dependency-map, or workspace-wide text bundle;
- no dependency, generated, or compiler-bundled source/documentation text;
- no multiple source fragments, context lines, rendered docs, summaries, ranking, token estimation,
  embeddings, or vector database;
- no model call, MCP execution, LSP/SCIP promise, structural edit, safe-change claim, or agent
  permission; and
- no source, query, serialization, latency, retained-size, or model-quality performance promise.
