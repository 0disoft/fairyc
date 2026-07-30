# ADR 0219: Add a narrow must-use call contract

- Status: Accepted
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Diagnostics source: `docs/diagnostics/catalog-v0.md`
- Depends on: ADR 0008, ADR 0019, ADR 0136, ADR 0173, ADR 0218

## Context

Fairy C already requires every `Result` and live resource obligation to be handled. Plain inert
returns remain freely discardable, which is correct for most scalar helpers but weak for APIs whose
successful call is useless unless the caller observes the returned token, count, status, or handle
identifier. Making every return mandatory would produce warning noise and `discard` boilerplate.

A source-only check is also insufficient. Adding or removing the contract on a public callable can
change whether downstream source is accepted, so the fact belongs to typed HIR and the canonical
public interface. Conversely, extending the full path-sensitive `Result` obligation to ordinary
copyable values would turn a call-site misuse check into a second ownership system.

## Decision

1. The source spelling is a zero-argument `@must_use` attribute on an ordinary internal function,
   associated function, method, or raw foreign function declaration. The attribute may appear once.
   Arguments, duplicates, placement on structs, enums, resources, cleanup blocks, or any other item
   are rejected.
2. An annotated callable must return a non-`void`, non-`Region` inert value. The result cannot be a
   safe borrow, a resource or resource-containing built-in tag, or `Result<T, E>`. `Result` and
   resources keep their stronger existing handling rules and must not produce duplicate diagnostics.
   Scalars, raw pointers, fixed inert arrays, inert structs, restricted inert enums, and inert
   `Option<T>` values are eligible when their existing type rules admit them.
3. The obligation is attached to the resolved call result, not to its type or later storage. A call
   is handled when its value is explicitly discarded or consumed by a binding, assignment, return,
   tail return, argument, receiver, operator, condition, cast, index, slice, aggregate construction,
   or switch. Storing the value ends this call-site obligation; Fairy C does not add general unused-
   local analysis or path-sensitive inert-value tracking in this revision.
4. A resolved annotated call used as a discarded-value expression statement is rejected. Transparent
   grouping and `unsafe` blocks do not hide it. If an `if`, `switch`, or ordinary block is itself a
   discarded-value expression, an annotated call returned by one of its reachable tails is likewise
   rejected. Calls used to compute another expression are handled even if that outer ordinary value
   is later ignored.
5. `discard annotated_call();` is the explicit acknowledgement. It evaluates the call exactly once
   and adds no runtime operation beyond the existing discard lowering.
6. `E4201` owns malformed, duplicate, misplaced, or ineligible `@must_use` declarations. `E4202`
   owns an ignored annotated call and relates the callable declaration. These codes form a separate
   return-value-use diagnostic category rather than changing `E4101` Result Flow.
7. Typed HIR retains `MustUse` on internal and foreign callables. Public-interface extraction and
   hashing retain it because downstream acceptance depends on the fact. The public-interface
   encoding therefore advances from v2 to v3; historical v1 and v2 fixtures remain frozen.
   Function-body MIR, runtime ABI, C ABI, and generated C do not retain the marker after semantic
   acceptance.
8. Stage 0 still has no general function values. A future function-value contract must decide
   whether must-use is part of the function type before annotated callables can flow through that
   boundary; this ADR authorizes resolved direct and method calls only.
9. The exact product language pin advances from `fairyc-language/3` to `fairyc-language/4` because
   a previously unsupported attribute gains source semantics and can make source invalid. The
   previous parser already preserves generic attribute syntax, but `/3` semantic analysis rejects
   `@must_use` as unsupported. The lossless migrator therefore preserves source bytes and requires a
   manual exact-pin update rather than rewriting the attribute.

## Compatibility

The change is source-compatible for declarations without `@must_use`, but pre-1 manifests, SDK
requirements, locks, and cache identities use exact language pins. Adding or removing `@must_use`
on a public callable changes public-interface v3. Runtime ABI, target schemas, C11 transport, MIR
version, SDK source bytes, and SDK version remain unchanged.

## Required evidence

- AST retention for zero-argument, argument-bearing, duplicate, and misplaced attributes.
- Semantic acceptance for eligible internal, method, imported, and foreign calls.
- `E4201` fixtures for malformed, duplicate, misplaced, `void`, borrow, resource, and `Result`
  declarations.
- `E4202` fixtures for direct, grouped, unsafe, block-tail, and conditional-tail ignored calls.
- Passing fixtures for `discard`, binding, assignment, return, argument, operator, condition, cast,
  aggregate, switch, and method-result consumption without duplicate Result diagnostics.
- Public-interface v3 vectors proving private changes do not invalidate while public must-use changes
  do invalidate.
- MIR, interpreter, and C-emission identity evidence proving handled calls add no executable feature.
- `/3` compatibility and migration checks proving byte preservation and manual pin separation.

## Consequences

Libraries can mark a small set of easy-to-ignore inert results without making ordinary C-shaped
functions noisy or importing another ownership rule. The cost is a public-interface encoding bump
and a precise expression-use traversal. Type-wide annotations, custom messages, warning severity,
general unused locals, function-value transport, and automatic fixing remain separate decisions.
