# Two-level Option transport — Go slice, 2026-11-14

ADR 0556 admits root `Option<Option<S>>` for the seven existing extended scalar
leaves. Source generic admission and C candidate selection share that exact
bounded shape. Triple Option, Result/Option mixtures, composite arguments and
foreign/storage paths retain their independent refusal contracts.

The first actual C compile caught a declaration-order defect: inert nested
transport entered the aggregate graph before its non-aggregate flat child had
been defined. Both are now emitted by the tagged dependency visitor. Numeric
type IDs and previously admitted layouts are unchanged.

Focused nested/flat Option tests pass in 7.1s. The new source checks preserve the
authored AST and emit both target profiles. Independent C observers check 33
state/value cases, with two copies each, across source and codec-restored MIR in
low/optimized native profiles. Their expected values cover outer-none,
inner-none, present payload, signed/unsigned extrema and floating signed zero.
Evidence: `.cache/nested-option-go-20261114.log`.

Candidate source/artifact and common CLI/cache integration remain pending; Go
success does not establish those gates.
