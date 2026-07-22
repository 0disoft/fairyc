# Source file identities v1

- Status: Executable prototype contract
- Decision: `docs/adr/0184-source-file-integrity-identities-v1.md`
- Implementation: `internal/packagecontent/file_identity.go`
- Workspace projection: `internal/workspaceloader/semantic.go`
- Snapshot projection: `internal/semanticsnapshot/`

## Boundary

The package-source digest identifies an entire canonical path-and-byte graph. Atlas additionally
needs to reject a stale individual file before using a stored byte span. Two independent identities
therefore accompany every source-relative file record:

- exact content identity: `fairyc.source-file/v1`;
- byte line-map identity: `fairyc.source-line-map/v1`.

Neither identity contains a path, package, filesystem root, timestamp, permission, source text,
host separator, or build-local compiler number. The surrounding snapshot record supplies the exact
`PackageInstanceId`, canonical package-relative path, and source/test kind.

## Exact content identity

The canonical byte sequence is:

1. ASCII `fairyc.source-file`;
2. one zero byte;
3. version byte `01`;
4. unsigned 64-bit big-endian source byte length; and
5. exact source bytes.

SHA-256 hashes that sequence. The only text form is:

```text
fyfile1-sha256-<64 lowercase hexadecimal digits>
```

Source bytes are not decoded, normalized, trimmed, or newline-converted. Empty bytes have a valid
identity even though the language loader independently rejects an empty or invalid source module.

## Line-map identity

The canonical byte sequence is:

1. ASCII `fairyc.source-line-map`;
2. one zero byte;
3. version byte `01`;
4. unsigned 64-bit big-endian file byte length;
5. unsigned 64-bit big-endian line-start count; and
6. every zero-based line-start byte offset as unsigned 64-bit big-endian values.

Offset zero is always present. Every LF byte adds the offset immediately after it, including an EOF
offset when the file ends in LF. CR is an ordinary byte; therefore CRLF and LF produce different
byte geometry. Non-newline content with equal byte length and equal LF positions intentionally has
the same line-map identity while its content identity differs.

SHA-256 hashes the sequence. The only text form is:

```text
fylm1-sha256-<64 lowercase hexadecimal digits>
```

## Handoff and validation

The validated workspace loader computes both identities from its already-contained immutable
source bytes. The explicitly requested `SemanticFiles` projection returns canonically package/path-
sorted detached metadata; ordinary `SemanticPackages` and cache compilation pay no per-file Atlas
hash cost. Before emitting records it recomputes `fysrc1-sha256` from the retained validated bytes
and rejects an in-memory workspace whose bytes no longer match its package identity.
`SemanticSnapshot` requires a nonempty complete file set for every package, requires exactly one
typed module for every file and vice versa, removes build-local package numbers, and stores no
source bytes.

A future `.fymap` reader must recompute the package-source digest and relevant file identities from
caller-authorized current source before serving spans or excerpts. Mismatch rejects stale evidence;
it is not downgraded to a guessed position. This contract does not yet implement map bytes,
publication, current-source lookup, excerpt loading, or stale-map diagnostics.

## Compatibility and evidence

Changing domains, version bytes, widths, offset rules, digest, or text prefixes requires new
identity versions. Unit vectors freeze exact empty, LF, CRLF, and final-newline behavior plus strict
text parsing. Workspace and snapshot pipeline tests cover production/test classification, stable
ordering, post-validation byte mutation rejection, complete file-to-module coverage,
input/accessor detachment, and package-number independence.
