# Public interface hash v4

- Status: Historical executable contract; superseded by public-interface hash v5
- Decision: `docs/adr/0341-public-interface-resource-abilities-v4.md`
- Fixture source: `spec/fixtures/module-interface/manifest-v4.json`
- Historical source: `docs/package/public-interface-hash-v3.md`
- Implementation source: `internal/moduleinterface/`
- Fixture runner: `internal/moduleinterfacefixture/`

## Contract

V4 preserves every v3 public item fact and adds the canonical effective ability set of each public
resource. A legacy resource without an explicit `@abilities` attribute is encoded as `drop`, which
is its existing semantic meaning. A public resource that changes between `drop`, `store`, or `key`
therefore changes its interface hash even when its private representation and cleanup body do not.

The only admitted order is `drop`, `store`, `key`, with omitted entries removed. Unknown,
duplicated, or noncanonical entries fail closed, and `key` requires `store`. Private resource
fields, bodies, spans, local names, and build-local package numbers remain excluded.

## Encoding

The binary domain remains `fairyc.public-interface`, but the version byte is `04`. One
length-framed ability string list follows the existing effect list for every item. Non-resource
items encode an empty ability list. SHA-256 hashes the complete stream. The text form is:

```text
fyif4-sha256-<64 lowercase hexadecimal digits>
```

V1 through v3 hashes remain historical fixture facts and fail parsing under the v4 consumer.

## Boundary

This encoding transports resource ownership policy between packages. It does not itself enable
`@abilities` in stable package compilation, add a storage backend, or define explicit destruction.
Those language and execution changes require separate revision cuts.
