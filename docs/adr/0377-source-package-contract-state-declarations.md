# ADR 0377: Source package contract-state declarations

- Status: Accepted
- Date: 2026-09-03
- Owners: Fairy C language and compiler
- Supersedes: the source-syntax closure in ADR 0345

## Context

The compiler already has one package-owned `hir.StateLayout`, canonical layout hashing,
public-interface v5 transport, cache invalidation, checked stable rebind, and a failure-atomic
transaction oracle. Until this decision, tests and compiler clients could construct that HIR only
from Go. That made the layout model executable but left ordinary Fairy packages unable to declare
their own storage schema.

Three shapes were considered:

1. a named user type whose fields implicitly become persistent;
2. a generated owner module built from an external schema;
3. one explicit package declaration that lowers directly to the existing HIR layout.

The first shape confuses ordinary in-memory aggregates with persistent namespaces. The second adds
a second schema and code-generation authority before source packages can express the contract.
The third keeps one declaration, one lowering path, and one layout-hash implementation.

## Decision

Language `fairyc-language/13` admits one package-owned declaration:

```fy
@contract_uid("example.token")
state {
    cell u64 total_supply;
    map bytes20 u64 balances;
}
```

The form is deliberately C-oriented: storage kind comes first, then complete key/value codec names,
then the field name. It does not use a Rust-like field colon, infer a schema from initializers, or
introduce user generics.

`state`, `cell`, and `map` are contextual declaration words rather than globally reserved keywords.
Existing modules, locals, fields, and functions using those names therefore remain valid; the
immediately previous revision still rejects the new complete declaration form.

The declaration has these exact rules:

- a package contains zero or one `state` declaration across all of its modules;
- `pub state` is invalid because package ownership is intrinsic;
- exactly one `@contract_uid("...")` attribute supplies the stable namespace consumed by the
  existing contract-state layout builder;
- fields retain source order and names exactly;
- `cell` takes one value codec and `map` takes key then value codec;
- the first closed codec vocabulary is `bool`, `u32`, `u64`, `bytes20`, and `bytes32`;
- those names select canonical storage codecs only in this declaration and do not broaden ordinary
  executable scalar types;
- duplicate declarations or malformed UID authority produce `E7201`, duplicate field names produce
  `E7202`, and unsupported or malformed codec declarations produce `E7203`.

Semantic analysis calls the existing `internal/contractstate` layout builder before publishing the
HIR layout. The same ordered field facts therefore define namespace prefixes and the layout hash.
Public-interface v5 and whole-package cache key v4 remain the only interface/cache identities:
source field, order, kind, codec, or UID changes invalidate the existing identity, while stable cache
rebind continues to reject a selected layout that differs from current HIR before MIR publication.
No second layout model or hash is added.

The compiler-bundled SDK advances to `0.0.0-stage0.56` solely to carry the exact language-13
identity. Runtime ABI remains `fairyc-runtime/0` because declarations add no runtime entry points.

## Consequences

Parser and semantic fixtures now include accepted source and rejected codec cases. Direct semantic,
compiler, and public-interface tests cover package ownership, deterministic field lowering, invalid
owners, duplicate fields, closed codecs, and interface invalidation. The public
`examples/contract_state_declaration_quickstart/` package proves clean source-to-verified-MIR
compilation; it intentionally performs no state operation.

This decision does not admit source state reads or writes, conditional transaction bodies, helper
calls, native state lowering, persistence providers, Merkle targets, signing, events, external
calls, or a deployable contract target. Those remain separate capability gates.
