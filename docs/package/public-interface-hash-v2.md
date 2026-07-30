# Public interface hash v2

- Status: Historical executable contract; superseded by public-interface hash v3
- Decision: `docs/adr/0173-public-interface-coordinate-hash-v2.md`
- Fixture source: `spec/fixtures/module-interface/manifest-v2.json`
- Implementation source: `internal/moduleinterface/`
- Fixture runner: `internal/moduleinterfacefixture/`

## Contract

V2 preserves the v1 declaration projection: sorted module paths and public items, public struct and
enum shapes, resource effects, function signatures/effects/unsafe state, and foreign ABI/link names.
Bodies, private items, private resource fields, spans, local names, and build-local package numbers
remain excluded.

The change is external nominal type identity. Types owned by the hashed package still use
`self::<module>::<type>`. External nominal types now use:

```text
dep[<full PackageCoordinateId>]::<module>::<type>
```

The coordinate includes UID, version, and features but excludes source identity. A source-only
dependency edit can therefore preserve this interface when all selected public facts are equal.

## Encoding

The binary domain remains `fairyc.public-interface`, but the version byte is `02`. SHA-256 hashes
the complete length-framed stream. The only text form is:

```text
fyif2-sha256-<64 lowercase hexadecimal digits>
```

V1 hashes are incompatible and fail parsing under the v2 consumer.

## Boundary

Coordinate plus interface equality is a public compatibility fact, not source equality. Exact
package source and provider records still use `PackageInstanceId` and source digest. ADR 0174 carries
the coordinate boundary through stable function/type identity and dependency cache keys, enabling
verified caller reuse across a private dependency edit.
