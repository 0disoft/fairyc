# Public interface hash v3

- Status: Executable prototype contract
- Decision: `docs/adr/0219-narrow-must-use-call-contract.md`
- Fixture source: `spec/fixtures/module-interface/manifest-v3.json`
- Historical source: `docs/package/public-interface-hash-v2.md`
- Implementation source: `internal/moduleinterface/`
- Fixture runner: `internal/moduleinterfacefixture/`

## Contract

V3 preserves the v2 coordinate-based nominal type projection and every existing public item fact.
It adds one `MustUse` bit to public internal and foreign callables because changing that bit changes
whether downstream source is accepted. Private annotations, bodies, private items, private resource
fields, spans, local names, and build-local package numbers remain excluded.

Types owned by the hashed package use `self::<module>::<type>`. External nominal types use
`dep[<full PackageCoordinateId>]::<module>::<type>`, preserving the v2 source-independent coordinate
boundary.

## Encoding

The binary domain remains `fairyc.public-interface`, but the version byte is `03`. The `MustUse` bit
is length-positioned with the existing item flags before ABI, owner, effects, parameter, and result
facts. SHA-256 hashes the complete length-framed stream. The only current text form is:

```text
fyif3-sha256-<64 lowercase hexadecimal digits>
```

V1 and v2 hashes remain historical fixture facts and fail parsing under the v3 consumer.

## Boundary

The hash records a source acceptance contract, not runtime behavior. `@must_use` does not enter MIR,
generated C, object ABI, or runtime metadata. Exact package source and provider records continue to
use `PackageInstanceId` and source digest; dependency cache keys pair the coordinate with this v3
interface hash.
