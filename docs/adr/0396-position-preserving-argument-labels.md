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

The candidate parser now parses labels for syntax parity, retaining the label
identifier and byte span in the existing argument node with original values
and order intact. This is not a semantic eligibility promotion: limited
adoption labels 11 private `expression` calls in `bodies.fy` passing
`minimum = 1`, `depth = 0`, and `allow_record = true/false`. The unmodified
workspace pipeline self corpus passed low/O2 over all 18 own-source files. Raw
generated C comparison reports nonidentity: whole-AST hashing in
`collectNominalTypes` (`internal/genericsource/nominal.go`) and
`bindNominalContext` (`nominal_packages.go`) lets labels alter nominal
definition and specialization order, so raw C byte identity is false. The
check passed with no C size growth: both raw generated C sizes 1,114,847
bytes, both MIR functions 407. This asserts no executable-size, runtime, or
compile-time guarantee, no semantic adapter promotion, and no measured 8%
cumulative C compliance (tracked separately). No compiler identity algorithm
changes. The parser adoption check is `TestArgumentLabelsParserAdoption`.
The real call triple remains exercised in a host-compiler fixture.
Acceptance covers mixed positional/labeled arguments, trailing commas, constant
evaluation, side effects, and resource movement. Rejection covers target
eligibility, misspelled/reordered/duplicate labels, arity, and generic expansion.

References: [field shorthand](0395-field-initializer-shorthand.md) and
[admission boundary](0218-modern-language-concept-admission-boundary.md).
