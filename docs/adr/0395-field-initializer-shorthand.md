# ADR 0395: Field initializer shorthand

- Status: Implemented
- Date: 2026-09-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Admission source: ADR 0218

## Evidence and decision

The existing lexer candidate repeats field names in scanner Token construction,
body-state nodes, and records Field construction. A library cannot remove this
syntactic repetition. Admit `.field` in a record initializer as exactly
`.field = field`, resolving the value through ordinary lexical name resolution.
Explicit and shorthand fields can be mixed, with optional trailing commas.
This applies equally to named, qualified, generic-prepared, and contextual enum
payload records; it does not admit any previously unsupported aggregate type.

Omission is recognized only before `,` or `}`. There are no default values,
spreads, implicit clones, field reordering, new conversions, or implicit borrows.
Existing duplicate/missing/private-field, type, move and loan checks remain owners
of rejection. A resource value moves exactly as in the explicit initializer.

## Representation and cost boundary

The parser constructs the same field and path nodes as explicit initialization.
The implicit path span covers the written identifier, excluding the dot and
trivia. Source tokens remain lossless; formatter output preserves the chosen form.
AST, typed HIR, MIR, cleanup, interpreter, and C emission reuse existing paths.
No new executable operation, tag, function specialization or runtime metadata is
introduced. Identical generated C is the executable-code oracle, with source
location-sensitive operations excluded from that identity claim.

Parsing adds a bounded local terminator check per designated field. Benchmarks
compare explicit and shorthand input with allocation counts; this is not a claim
of universally unchanged end-to-end compile time or debug/cache size. Separately
measure old-syntax input before/after when claiming no compiler regression.

## Compatibility

Advance the exact language pin from `/14` to `/15` and SDK from stage0.59 to
stage0.60. Current manifests, scaffold defaults and current test inputs move
together. Existing programs need the new manifest pin and refreshed SDK locks;
their source need not change. Historical ADRs and evidence retain their identities.
The existing legacy migration parser rejects shorthand; this change does not
repair or expand its historical migration coverage. Do not use it as a general
`/14` migration tool. The Fairy-written experimental parser is not promoted and
does not yet admit this new surface; its own source keeps explicit initializers.
Runtime ABI, MIR, C ABI and public-interface schemas are unchanged. Cache keys
already include language revision. A compiler release containing this addition
requires at least a minor release; this change creates no release or tag.

## Verification scope

Parser acceptance/recovery and source spans; formatter preservation/idempotence;
ordinary semantic and ownership diagnostic parity; byte-identical C for scalar
records and moved resources; stable check/build/run in both Windows profiles;
exact-pin rejection and current manifest/scaffold consumers. Full cross-target
release validation remains separate from this bounded implementation.

The focused 11-package suite, eight Atlas/identity consumer packages and Windows
stable CLI tests passed locally. Scalar records, contextual enum payloads and
moved resources produce byte-identical C to their explicit counterparts.
Formatting and scoped Mustflow strict validation passed. Full-suite, Linux native
execution, final object/executable size comparison and CI were not run here.

On the local Ryzen 5 7430U, five 200ms old-syntax parser samples had a baseline
median of 2.112ms and current median of 2.044ms, with wide overlapping ranges
(1.247-2.665ms and 1.793-2.911ms). Both typically used 3,541 allocations and about
710KB per parse. A separate three-sample explicit/shorthand comparison used
3,541/3,140 allocations and about 710KB/581KB; its timing was also noisy.
These are bounded parser observations, not proof of end-to-end speed parity.
