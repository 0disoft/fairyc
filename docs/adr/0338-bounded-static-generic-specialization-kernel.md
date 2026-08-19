# ADR 0338: Establish a bounded static generic specialization kernel

Status: Accepted experimental compiler kernel

Date: 2026-08-19

Implementation: `internal/genericplan`

Related decisions: ADR 0003, ADR 0005, ADR 0010, ADR 0136, ADR 0152, ADR 0174, ADR 0218, ADR 0223, ADR 0328

## Context

Fairy C currently permits type arguments only for compiler-known types such as `Option<T>` and
`Result<T, E>`. It has no type-parameter declarations, generic functions, generic aggregates,
implementation coherence, monomorphization budget, or stable specialization identity.

ADR 0223 correctly deferred interfaces until a measured generic workload existed. A
blockchain-oriented SDK supplies that workload. Typed state and ABI generation repeatedly need
`StorageMap<K, V>`, `Codec<T>`, `Event<T>`, and `ContractRef<T>`. Duplicating each operation for every
address, hash, integer, resource, and record type would produce large generated SDKs and duplicate
security-critical encoding code.

That workload does not justify importing Rust traits, C++ templates, Go interfaces, or Solidity
modifiers wholesale. Source syntax is the easy part. Coherence, provider selection, transitive cache
identity, recursive expansion, diagnostics, and code-size limits must be executable first.

The existing compiler also expects concrete types after semantic analysis. Unresolved type variables
must not leak into HIR, MIR, the verifier, interpreter, C transport, package artifacts, or backends.

## Decision

Add a host-independent `genericplan` kernel. It validates generic declarations and produces exact,
dependency-ordered specialization plans. It changes no accepted Fairy C source syntax.

### Immutable catalog

A builder accepts concrete types, static capabilities, exact implementations, and generic
declarations. `Build` validates all references and publishes an immutable catalog. Caller-owned
slices are cloned. Lookup methods return defensive copies. Planning allocates call-local state, so
one catalog supports concurrent compiler requests without locks.

Every type, capability, provider, and generic declaration carries an explicit SHA-256 definition
digest. Missing digests are rejected. The complete catalog receives a deterministic digest, while
individual specialization identities include only reachable definitions so unrelated changes do
not invalidate all generated code.

### Compiler-known abilities

The first closed ability vocabulary is:

| Ability | Planning meaning |
| --- | --- |
| `copy` | A value may be duplicated by value. |
| `drop` | Ordinary control flow may discard the value. |
| `store` | A storage-owning generic may contain the value. |
| `key` | The value may identify persistent state; implies `store`. |
| `inert` | The value has no resource cleanup obligation; implies `drop`. |
| `scalar` | Compiler-known inert scalar; implies `copy`, `drop`, `inert`, and `store`. |
| `resource` | Move-only value with possible cleanup or conservation obligations. |

`resource` is incompatible with `copy`, `inert`, and `scalar`. These are compiler-known properties,
not user-implementable traits. Resource Flow remains authoritative over moves and cleanup.

### Exact static capabilities

A capability has one canonical path and arity. An implementation binds one exact capability and one
exact concrete type tuple to one provider:

```text
sdk::Codec<sdk::Address> = sdk::encode_address
sdk::Codec<sdk::U64>     = sdk::encode_u64
```

Lookup is exact. The kernel has no structural matching, blanket implementation, fallback,
specialization precedence, negative implementation, associated type, default method, dynamic value,
box, reflection object, or runtime table.

Two providers for the same capability and concrete tuple are rejected. The implementation package
must own the capability or at least one argument type. A third package therefore cannot redefine a
foreign capability for foreign types.

### Explicit generic declarations

A declaration owns ordered type parameters. Each parameter may require compiler-known abilities.
Every request supplies all concrete type arguments; there is no inference.

A declaration may require exact capabilities after substitution and may request other generic
specializations as dependencies. Dependency arguments are local parameter references or exact
concrete types. This models a storage write requiring `key` for `K`, `store` for `V`, exact
`Codec<K>` and `Codec<V>` providers, plus reusable `Encode<K>` and `Encode<V>` specializations.

### Deterministic specialization

Planning performs these operations:

1. sort and deduplicate root requests;
2. bind exact concrete types and validate abilities;
3. select exact capability providers;
4. recursively expand and memoize dependencies;
5. reject active dependency cycles;
6. emit every dependency before its consumer;
7. compute a domain-separated specialization ID.

The ID includes the declaration definition, concrete type definitions and canonical abilities,
capability definitions, selected provider definitions, and transitive dependency IDs. A reachable
codec change invalidates its specialization and consumers. An unrelated codec leaves unaffected IDs
unchanged. Catalog insertion order and request order do not alter the result.

The plan contains no runtime dispatch. A future frontend must substitute each plan entry into
ordinary concrete HIR before Flow and MIR lowering.

### Mandatory budgets

Catalog limits bound types, capabilities, implementations, declarations, parameters, constraints,
and dependencies. Plan limits separately bound total specializations, specializations per
declaration, and expansion depth.

Exact recursion produces a cycle rejection. Non-repeating recursive growth hits a depth or count
budget. Compiler memory exhaustion is never the language limit.

### Source syntax remains unreserved

This decision does not reserve `generic`, `where`, `trait`, `capability`, `impl`, or any attribute.
The following is illustrative only:

```fy
capability Codec<T> {
    fn Result<usize, EncodeError> encode(ref T value, ref mut u8[] output);
}

impl Codec<Address> using address_codec;

fn Result<void, StorageError> set<K, V>(
    ref mut StorageMap<K, V> map,
    K key,
    V value,
)
where K: key + Codec<K>, V: store + Codec<V> {
}
```

A later syntax proposal may choose another spelling, but it must preserve or strengthen the kernel's
coherence, determinism, and budget rules.

## Future compiler boundary

The first source-level vertical slice should support explicit generic functions and compiler-known
ability constraints only. It must resolve a catalog from the package graph, request concrete call
specializations, substitute to concrete HIR, and reject any type parameter surviving into MIR.

Generic aggregates, user capability declarations, cross-package implementation publication,
inference, and stable symbol encoding are separate slices. Source promotion also requires explicit
new versions for public interfaces, package caches, stable artifacts, and migration behavior.

## Security and performance

Exact lookup removes provider ambiguity. The orphan rule blocks foreign redefinition. Definition
digests and transitive IDs prevent stale cache reuse. Explicit budgets bound adversarial generic
expansion. The kernel opens no filesystem, environment, network, process, clock, randomness, native
plugin, JIT, allocator, scheduler, raw-pointer, or target-execution path.

Catalog validation is bounded by declared sizes and deterministic sorting. Planning memoizes every
concrete declaration/type tuple once and hashes only reachable semantic inputs. Code-size growth is
visible in specialization counts rather than hidden behind runtime machinery.

## Validation

Focused tests cover ability closure and conflicts, deterministic catalogs, defensive immutability,
duplicate implementation rejection, orphan coherence, required definition identities,
dependency-first storage planning, request-order independence, missing abilities, missing exact
providers, transitive provider invalidation, unrelated specialization stability, cycles, total and
per-declaration budgets, concurrent planning, and an executable `StorageWrite<K, V>` example.

```text
go test ./internal/genericplan
go vet ./internal/genericplan
go test -race ./internal/genericplan
```

## Deferred work and recovery

Deferred work includes parser syntax, inference, generic structs and resources, capability method
signatures, associated items, blanket or negative implementations, specialization, dynamic
dispatch, reflection, higher-kinded types, const parameters, foreign generic ABI, public mangling,
HIR substitution, MIR evidence, interpreter execution, and backend emission.

Revert the package and this ADR to recover. No source program, SDK release, serialized artifact,
runtime state, deployed contract, or persistent storage layout depends on the experimental kernel.
