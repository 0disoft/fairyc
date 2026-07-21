# PackageInstanceId v1

- Status: Executable prototype contract
- Decision: `docs/adr/0138-package-instance-id-v1.md`
- Fixture source: `spec/fixtures/package-identity/manifest-v1.json`
- Implementation source: `internal/packagegraph/identity.go`
- Fixture runner: `internal/packageidentityfixture/`

## Input tuple

The input is the already-resolved tuple defined by `package-graph-v0.md`, in this exact order:

1. `PackageUid`;
2. source identity;
3. version identity;
4. the resolved feature set.

The first three values must be nonempty valid UTF-8 strings. Features must each be nonempty valid
UTF-8 and the list must already be sorted and unique by exact Go/UTF-8 byte order. The encoder does
not normalize a value. In particular, visually equivalent Unicode strings, path separators, URL
spellings, Git references, version spellings, and case variants remain different byte strings unless
their owning resolver normalized them before constructing this tuple.

## Canonical byte encoding

All lengths and counts are unsigned 64-bit integers in network byte order. No native word size,
endianness, string terminator, locale, or filesystem rule participates.

| Order | Bytes | Meaning |
| --- | --- | --- |
| 1 | ASCII `fairyc.package-instance` | Domain separator |
| 2 | `00` | Domain terminator |
| 3 | `01` | Encoding version |
| 4 | ASCII `u`, length, bytes | `PackageUid` |
| 5 | ASCII `s`, length, bytes | Source identity |
| 6 | ASCII `v`, length, bytes | Version identity |
| 7 | ASCII `f`, count | Feature list header |
| 8 | length, bytes repeated `count` times | Features in canonical sorted order |

Tags and fixed-width lengths make every component boundary unambiguous. The byte encoding is the
contract; JSON fixture spelling and Go structs are only inputs to it.

## Digest and text form

The digest is SHA-256 over the complete canonical byte sequence. Consumers serialize the complete
digest as:

```text
fypi1-sha256-<64 lowercase hexadecimal digits>
```

Readers reject missing or different prefixes, uppercase hexadecimal, non-hexadecimal bytes, and
short or extended digests. There is no alternate base32, truncated, bare-hex, or filename form.

## Equality and collision handling

Two resolved instances are equal only when their complete input tuples are equal. The digest is an
index into that equality relation, not permission to skip it. Whenever two live records present the
same digest, the consumer compares the complete tuple or a trusted canonical-byte record. Unequal
inputs under one digest fail closed as a collision.

The current graph resolver retains source tuples and performs that comparison. Future lockfiles,
caches, and Atlas maps must either retain enough canonical identity data to perform the same check or
reject ambiguous records before use.

## Compatibility boundary

`PackageInstanceId` v1 is independently versioned from manifest, lockfile, source archive, cache,
module-interface, `.fymap`, `SymbolId`, runtime ABI, and compiler release formats. A cache mismatch is
a safe miss. A lockfile or map claiming this algorithm with malformed text or contradictory tuple
data is corrupt and rejected.
