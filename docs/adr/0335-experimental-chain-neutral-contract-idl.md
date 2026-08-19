# ADR 0335: Add an experimental chain-neutral contract IDL and typed binding generator

- Status: Accepted experimental slice
- Date: 2026-08-19
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Depends on: ADR 0001, ADR 0005, ADR 0008, ADR 0218, ADR 0223, ADR 0328

## Context

Blockchain clients need a stable description of external contracts, callable functions, event
payloads, and value-transfer intent. Copying Solidity ABI syntax into the Stage 0 language would tie
Fairy C to one virtual machine before a contract target, storage model, address representation, or
wire codec has earned that commitment. Treating generated client code as handwritten strings would
also discard the language's existing preference for explicit types, effects, and deterministic
artifacts.

The immediate useful boundary is smaller than an on-chain backend. A source author needs one compact
interface declaration, a deterministic chain-neutral IDL, and typed TypeScript and Go clients whose
transport remains supplied by the selected ecosystem. That boundary can be implemented and measured
without claiming contract execution, changing MIR, or weakening the Stage 0 compiler.

## Decision

The repository owns a separate experimental contract-interface profile consumed only by the
`fairyc-contract-idl` command. Its grammar lives at `spec/contract-idl/v0.ebnf`; it does not alter
`spec/grammar/stage0.ebnf`, reserve Stage 0 keywords, or advance `fairyc-language/10`.

A profile file starts with one ordinary `module` declaration. `extern contract Name { ... }` owns an
ordered set of bodyless typed functions. `@event struct Name { ... }` owns an ordered event payload.
Contract functions may carry `@view` or `@payable`, never both. Every function receives the canonical
`external_call` effect in the emitted model and every generated transport request preserves that
effect.

The canonical artifact uses schema identity `fairyc.contract-idl/v0` and is described by
`spec/contract-idl/v0.schema.json`. Declaration order is preserved. The JSON renderer is deterministic
and appends one newline. TypeScript and Go renderers consume the same in-memory document rather than
reparsing source or inventing target-specific metadata.

Generated clients depend on a narrow `ContractTransport`. The transport owns the selected chain,
codec, address representation, signing, submission, confirmation, and response decoding. No source
spelling names EVM ABI, BCS, Borsh, SCALE, RPC vendors, wallet libraries, or network identifiers.
`@payable` produces an explicit value options parameter; no hidden ambient transaction value exists.

The v0 ABI vocabulary admits booleans, fixed-width signed and unsigned integers through `u256`,
`String`, `Bytes`, `Option<T>`, `Result<T, E>`, fixed arrays from one through 65,535 elements, event
records, and opaque nominal paths. `void` is restricted to a direct function result or `Result`
success payload. Safe references, raw pointers, unbounded slices, `usize`, `isize`, `str`, exact C
scalars, arbitrary user generics, zero-length arrays, duplicate declarations, and ambiguous
attributes fail closed.

Input is capped at eight MiB, token count at 1,048,576, and recursive type parsing at 256 levels.
Diagnostics carry stable `CIDL1xxx` codes plus exact byte spans and one-based line and byte-column
coordinates. The command emits no partial artifact after a source diagnostic.

## Deliberate boundary

This slice does not make contract declarations callable from ordinary Fairy C bodies. It does not
lower an external call into HIR or MIR, invalidate a live mutable borrow at a call site, perform
reentrancy analysis, choose a chain address type, generate selectors or topics, deploy bytecode, or
provide a storage model. Those operations require an accepted contract target and executable
borrow/effect evidence. The IDL's explicit `external_call` fact is the prerequisite consumed by that
later checker; it is not presented as proof that the checker already exists.

The next admissible compiler slice must resolve a typed contract handle against this IDL, end every
live mutable storage loan before an `external_call`, and reject or explicitly annotate call paths
that mutate state after a reentrant boundary. It must add semantic, MIR, negative, and target adapter
evidence before changing the product grammar.

## Compatibility

This change adds a standalone experimental command, package, schema, grammar, example, and tests. It
does not modify Stage 0 parsing, semantic analysis, MIR, C emission, runtime ABI, target descriptions,
workspace manifests, lockfiles, package hashes, or the stable `fairyc` command surface. Contract IDL
v0 may evolve incompatibly until a later ADR promotes it.

## Consequences

Blockchain SDK authors can generate one typed client surface for multiple chain adapters while Fairy
C avoids an early commitment to Solidity or a specific serialization system. The compiler gains a
concrete, versioned external-call contract to build on. The cost is an additional experimental parser
and a temporary standalone command until source-level contract calls justify integration with the
main compiler pipeline.
