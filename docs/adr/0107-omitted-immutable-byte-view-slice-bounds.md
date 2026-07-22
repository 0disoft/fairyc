# ADR 0107: Omitted immutable byte-view slice bounds

- Status: Accepted
- Date: 2026-07-20
- Language surface: `bytes[..end]`, `bytes[start..]`, and `bytes[..]`

## Context

ADR 0105 admitted checked end-exclusive `&[u8]` slicing but required both bounds. Safe callers had
to spell `0` or repeat `.len` for the common prefix, suffix, and whole-view cases. Repeating the
receiver to obtain its length would be wrong for a general expression because it could duplicate
evaluation. The omitted forms therefore need an explicit normalization rule after the receiver is
materialized once.

## Decision

1. Immutable `&[u8]` slicing admits `bytes[..end]`, `bytes[start..]`, and `bytes[..]` in addition
   to `bytes[start..end]`. The exclusive-end rule is unchanged.
2. An omitted start normalizes to `usize` zero. An omitted end normalizes to the retained length of
   the already-evaluated receiver. Neither default is a source expression or an additional call.
3. The receiver and every written bound evaluate exactly once from left to right. MIR reads omitted
   end metadata from the materialized receiver, so `make_view()[start..]` cannot call
   `make_view()` twice.
4. Typed HIR retains explicit `omitted-range-bound` placeholders. MIR removes those placeholders
   before verification and preserves the existing three-operand `view-slice-check`,
   `checked-view-slice`, and `view-slice` contract.
5. All four spellings use the same `start <= end && end <= len` check and `bounds` trap. Omission
   does not create an unchecked or optimizer-only path.
6. The interpreter and C emitter consume only normalized MIR. Generated C reads the receiver length
   into a local before the existing bounds checks and performs no pointer derivation on failure.
   When normalized MIR carries the literal `usize` zero for an omitted start, the C emitter omits
   the tautological unsigned `0 > end` comparison and retains the required `end > len` check. This
   target-level simplification does not weaken the verified MIR contract.
7. Inclusive ranges, omitted-range syntax outside immutable byte-view slicing, mutable slicing,
   indexed assignment, `str` slicing or indexing, and general collection slicing remain deferred.
8. The interpreter fixture contract grows from 84 to 87 cases. C emission remains 29 cases and the
   derived Linux matrix remains 116 compiler/profile executions and 58 sanitizer executions because
   the existing byte-view fixture now covers the new spelling. The Windows quickstart exercises an
   omitted end through local x64 MSVC without WSL.

## Consequences

- Prefix, suffix, and whole-view slicing no longer require boilerplate bounds.
- Omitted-end lowering remains single-evaluation even when receivers later become effectful.
- The verified MIR and generated-C safety shape do not gain a second slice operation family.
- Whole-view and prefix slices compile cleanly under GCC `-Wtype-limits` with warnings as errors.

## References

- `docs/adr/0105-checked-immutable-byte-view-slicing.md`
- `docs/language/0005-surface-syntax-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
- `docs/runtime/runtime-stdlib-v0.md`
