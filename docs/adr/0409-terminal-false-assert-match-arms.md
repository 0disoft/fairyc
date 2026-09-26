# ADR 0409: Terminal false assertions in Result and Option switch arms

- Status: Accepted; implementation pending
- Date: 2026-09-27
- Product source: [product specification](../product/02-spec.md)
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)
- Extends: [ADR 0402](0402-result-return-only-match-arm.md)

## Workload and decision

Private vector and lexer programs currently put a fabricated value after
`assert(false)` in a value-producing `Result` or `Option` switch arm. Examples
include `assert(false); bytes::empty()` after a failed byte copy and
`assert(false); make(-1)` after an absent token. The value is unreachable, but
the type checker currently requires it to give both arms the same type.

Admit a case block with no tail expression whose **final statement** is the
literal `assert(false);` as a terminating arm in a shallow, exhaustive built-in
`Result` or `Option` switch. The arm supplies no value to the join; at least one
other arm must continue and supply the switch value. Earlier statements in the
terminating arm are checked and executed in order. A nonliteral assertion is
still a possibly continuing statement and must supply a value when the switch
requires one. Empty and other void arms remain type errors.

This is a narrow control-flow fact, not a source `never` type. It does not admit
an arbitrary trapping expression, nested divergence inference, direct-return
Option arms, user-enum arms, `let-else`, pattern guards, or new syntax. Existing
Result direct-return arms remain as specified by ADR 0402.

## Lowering and cost boundary

The checker ignores the terminating arm when selecting and comparing the
continuing arm's type. The existing HIR block, verified MIR trap terminator,
`assertion-failed` category, interpreter, and constrained-C lowering handle the
arm. A trap performs no `defer`, resource drop, or Region reset. No runtime
helper, type metadata, generic instance, ABI representation, or backend
operation is added. The check is local to two-arm built-in switches; unrelated
source does not enter a new global inference pass.

Implementation must prove the success branch's result and ownership, the
failure branch's trap category with live resources, preserved rejection of a
nonliteral assertion or empty arm, and native low/optimized behavior. Compare
the same program with and without the unreachable fabricated value to inspect
generated C and linked size. The cumulative independent-C runtime and image
budget remains 8%; this ADR alone makes no performance claim.

## Compatibility

This changes which value-producing source is accepted. Implementation advances
the exact language pin from `fairyc-language/19` to `/20` and the bundled SDK
from `0.0.0-stage0.69` to `.70`, with matching manifests, locks, cache inputs,
fixtures, and examples updated in the same implementation cutover. The source
bytes of existing valid programs need no rewrite. Runtime ABI, MIR version,
public-interface schema, and C transport remain unchanged. This accepted
design does not activate the form or change a pin until implementation passes
its tests and compatibility update.
