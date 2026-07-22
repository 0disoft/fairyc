# Public interface hash v1

- Status: Historical; superseded by public-interface hash v2
- Decision: `docs/adr/0139-public-interface-hash-v1.md`
- Fixture source: `spec/fixtures/module-interface/manifest-v1.json`
- Implementation source: `internal/moduleinterface/`
- Fixture runner: `internal/moduleinterfacefixture/`

## Input boundary

Extraction accepts only typed HIR after complete semantic Flow and one explicit mapping from every
referenced build-local `PackageID` to `PackageInstanceId` v1. It extracts exactly one package at a
time. Missing package identity, duplicate module path, duplicate public `(kind, name)`, unsupported
canonical type, private local type leakage, or noncanonical effect order rejects extraction.

The descriptor is a semantic projection, not serialized HIR. Source text, spans, AST nodes, HIR
symbols, bodies, MIR, target layout, and backend data are not retained.

## Descriptor contents

Modules sort by source-visible module path. All modules participate, including a module with no
public items, because adding or removing an importable module path changes the package surface.
Within a module, items sort by kind then name.

| Item | Included | Excluded |
| --- | --- | --- |
| Struct | name, `@repr(C)`, ordered field names and types | spans and source spelling |
| Enum | name, ordered variants, ordered payload field names and types | match bodies and backend tags |
| Resource | name, sorted cleanup effects | private fields, cleanup body, generated glue |
| Function | owner-qualified name, owner type, unsafe, effects, ordered parameter types, result | parameter names, callee-local `mut`, body, locals |
| Foreign function | source name, C ABI, exact link name, effects, parameter and result types | call sites, object symbols not named by the declaration |

Resource fields are currently module-private. Excluding them makes this a source-interface hash,
not a compiled layout or native ABI hash. Public struct and enum fields remain included.

## Stable type names

Built-in scalars, C scalars, `void`, `str`, `CStr`, `Region`, `u8[]`, `ref`, `ref mut`, `ptr`,
`ptr mut`, fixed arrays, `Option`, and `Result` receive deterministic recursive spellings. The
spelling is interface metadata and does not change Fairy C source grammar.

Nominal types use:

```text
self::<module>::<type>
dep[<full PackageInstanceId>]::<module>::<type>
```

The first form deliberately omits the current package instance, letting a private source or version
change preserve an equal interface. The dependency form deliberately includes the complete external
instance because nominal types from different resolved instances are not interchangeable.

ADR 0173 now provides that new interface-hash version. This file remains the historical v1 byte and
type-spelling contract.

`internal/moduleinterface.ComputePackages` now applies the current v2 extraction and hash contract
to an explicit complete package/coordinate set in deterministic order. It rejects empty, duplicate,
or unmapped inputs and is shared by workspace-cache coordination and Atlas logical planning; it does
not create a new interface-hash version.

## Canonical bytes and digest

The byte stream begins with ASCII `fairyc.public-interface`, a zero byte, version byte `01`, and an
unsigned 64-bit network-order module count. Every string is an unsigned 64-bit network-order byte
length followed by exact UTF-8 bytes. Lists use unsigned 64-bit network-order counts. Booleans use
one byte, zero or one. Item fields are emitted in the fixed order defined by the implementation and
ADR. No platform word size, locale, map iteration, JSON property order, or filesystem order enters
the bytes.

SHA-256 hashes the complete byte stream. The canonical text form is:

```text
fyif1-sha256-<64 lowercase hexadecimal digits>
```

Changing the byte encoding, included facts, type spelling, digest, or prefix requires a new
interface-hash version. It does not require changing `PackageInstanceId`.

## Cache boundary

An equal interface hash proves only equality under this v1 source-interface projection. Whole-package
cache reuse additionally checks implementation/source digest for the package being compiled and the
complete compatibility vector for the artifact being reused. A malformed, unknown-version, or
contradictory cache record is a safe miss. Interface hashing failure never changes language analysis;
it disables reuse for that package.
