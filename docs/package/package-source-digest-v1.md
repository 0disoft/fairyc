# Package source digest v1

- Status: Executable prototype contract
- Decision: `docs/adr/0140-package-source-digest-v1.md`
- Fixture source: `spec/fixtures/package-source/manifest-v1.json`
- Implementation source: `internal/packagecontent/`
- Fixture runner: `internal/packagecontentfixture/`

## Input

One package source graph is a nonempty set of `(canonical relative path, exact bytes)` records. The
package loader has already established source-root containment, file type, size budgets, and the set
of files. This contract neither discovers nor reads files.

Canonical paths:

- are valid nonempty UTF-8;
- use `/`, never `\`;
- are relative, clean, and contain no `.` or `..` traversal segment;
- contain no volume or colon spelling;
- are unique by exact bytes;
- end in `.fy`.

Accepted path bytes are not case-folded or Unicode-normalized. Source bytes are not decoded,
newline-normalized, trimmed, or reformatted.

## Canonical encoding

The encoder sorts files by exact path. It writes:

1. ASCII `fairyc.package-source`;
2. one zero byte;
3. version byte `01`;
4. unsigned 64-bit network-order file count;
5. for each file, unsigned 64-bit path length and path bytes;
6. unsigned 64-bit content length and exact content bytes.

Lengths preserve boundaries even when two different path/content splits would otherwise concatenate
to the same byte sequence. No native endianness, filesystem order, timestamp, permission, owner,
inode, archive header, or absolute root enters the encoding.

## Digest and compatibility

SHA-256 hashes the complete encoding. The canonical text is:

```text
fysrc1-sha256-<64 lowercase hexadecimal digits>
```

Changing the path rules, byte encoding, digest, or prefix requires a new source-digest version.
Malformed or unknown versions are cache misses and stale Atlas maps, never partial matches.

This digest identifies exact package source only. An artifact cache key additionally includes
`PackageInstanceId`, compiler and language compatibility, MIR, backend, target/ABI, profile,
toolchain, runtime, standard-library identity, features, and dependency public-interface hashes.
The current package's public-interface hash is stored as output metadata for dependent invalidation;
it does not replace the exact source digest.
