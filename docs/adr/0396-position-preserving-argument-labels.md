# ADR 0396: Position-preserving argument labels

- Status: Implemented
- Date: 2026-09-22

## Evidence and decision

`examples/lexer_candidate/src/bodies.fy` contains `expression(..., 1, 0, true)`, where
`minimum`, `depth`, and `allow_record` have independent meanings. The earlier
record-mode flag cleanup is separate and is not evidence for this feature.

Allow `f(positional, .minimum = 1, .depth = 0, .allow_record = true)`.
Argument order and arity are invariant; positional arguments may follow labels.
Each label must exactly match the parameter at that position. Reordering, duplicate
labels, typos, omitted arguments, and default values are not admitted.

Only same-module private non-generic ordinary direct-path calls qualify.
Reject public, external, method, indirect, intrinsic, contextual-constructor,
and explicitly generic calls.

## Representation and validation

The AST retains ordinary value children plus optional call-label metadata.
After type checking, HIR contains ordinary calls. `@const_eval` applies the same
check. Preserve the original explicit-generic mark through generic expansion.
Reuse existing ownership, borrow, effect, and cleanup handling. The parser
distinguishes `.identifier = expression` from variant syntax by `=` lookahead.

Use `E1014` for label or target mismatch, `E1010` for arity, and existing type
and move diagnostics otherwise. Apart from source-location-sensitive expressions,
require identical generated C, side-effect order, and native behavior. This does
not establish unchanged whole-compilation time or debug/cache footprint.

## Compatibility and acceptance

Implementation advances `fairyc-language/15` to `/16` and SDK
`0.0.0-stage0.60` to `0.0.0-stage0.61`. Synchronize current manifests,
scaffolds, and tests; retain historical records.

The Fairy-written candidate parser does not support labels. Keep its own source
unlabeled and exercise the real call triple in a host-compiler fixture.
Acceptance covers mixed positional/labeled arguments, trailing commas, constant
evaluation, side effects, and resource movement. Rejection covers target
eligibility, misspelled/reordered/duplicate labels, arity, and generic expansion.

References: [field shorthand](0395-field-initializer-shorthand.md) and
[admission boundary](0218-modern-language-concept-admission-boundary.md).
