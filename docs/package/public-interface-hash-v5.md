# Public interface hash v5

- Status: Experimental executable contract
- Decision: `docs/adr/0345-hir-contract-state-interface-and-cache-identity.md`
- Fixture source: `spec/fixtures/module-interface/manifest-v5.json`
- Historical source: `docs/package/public-interface-hash-v4.md`
- Implementation source: `internal/moduleinterface/`
- Fixture runner: `internal/moduleinterfacefixture/`

## Contract

V5 preserves every v4 public item fact and adds the exact persistent-state layout identity owned by
the package. A layout record contains its canonical contract UID and the full 32-byte
`fairyc-contract-state-layout/v1` hash. Changing field order, name, kind, codec, prefix, or contract
UID therefore changes the package interface and every dependent cache key.

At most one state layout belongs to one build-local package. Package numbers select the layout during
extraction but are not encoded. Renumbering a resolved workspace therefore preserves the interface.
The v5 fixture injects its layout after source analysis instead of reserving source syntax. Packages
without a layout encode an empty list, and the fixture retains all prior source-only same/different
relations under one fixed injected layout.

## Encoding

The binary stream begins with ASCII `fairyc.public-interface`, a zero byte, and version byte `05`.
A length-framed state-layout count follows. Each record contains a length-framed contract UID and
the raw 32-byte layout hash. The existing v4 module and item stream follows unchanged.

SHA-256 hashes the complete stream. The only text form is:

```text
fyif5-sha256-<64 lowercase hexadecimal digits>
```

V1 through v4 hashes remain historical fixture facts and fail parsing under the v5 consumer.

## Boundary

The layout originates in source-independent typed HIR and is lowered through the compiler-private
transaction MIR. Process-local HIR state transactions do not enter this public interface because
they are function-body behavior rather than a public layout promise. V5 does not admit source
syntax, transaction-body serialization, a runtime ABI, or a target backend.
