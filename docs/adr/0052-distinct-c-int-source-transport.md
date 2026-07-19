# Distinct `c::int` Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- C backend source: `docs/mir/c-backend-safety-v0.md`

## Context

The executable source FFI slice admitted fixed-width Fairy integers but did not admit any source C
scalar. Treating C `int` as an `i32` alias would preserve the current target's width by accident
while erasing the type identity needed to detect incompatible declarations and emit the correct C
prototype. Opening every target-listed C scalar at once would also require unresolved floating,
plain-`char`, width, literal, and arithmetic contracts.

## Decision

1. The first source C-scalar slice admits only `c::int`. Typed HIR and MIR preserve it as the
   distinct canonical type `c::int`; it is never canonicalized to `i32`.
2. Source `c::int` maps explicitly to the existing target-layout key `c.int`. The constrained C
   backend spells it exactly `int` and emits size and alignment static assertions from the selected
   target description.
3. The admitted executable path is transport-only: a foreign C call may return `c::int`, the value
   may occupy a typed MIR local, and a later foreign C call may consume that local. Context-typed
   `c::int` integer literals, arithmetic, comparison, casts, and interpreter execution remain
   deferred. No `OperandI32` reuse may erase the distinction.
4. Foreign declarations mapped to one link name remain compatible only when their exact parameter
   and result types match. `i32` and `c::int` declarations for one external symbol conflict.
5. This slice does not add `c::int` fields to the implemented source `@repr(C)` or restricted-enum
   backend subset. Those aggregate crossings require their own source construction and native
   evidence.
6. A dedicated native producer/consumer probe links generated Fairy callers with hand-written C
   and expects four Clang/GCC low/optimized executions. `fairyc.native-c-report/v16` adds
   `foreignCIntCallExecutions`; the eighteen production emission cases and their 72/36 execution
   counts do not change.
7. `c::size_t` remains a later ISO C typedef/header slice in this decision and must not alias
   `usize` or `uintptr_t`; ADR 0053 implements that independent slice. `c::ssize_t` remains a later
   target-library/POSIX slice and must not alias `isize` or `intptr_t`; ADR 0054 implements that
   independent target-qualified transport.
8. `c::long` remains excluded by this decision and must not alias `i64` or `isize`; ADR 0055 later
   implements the exact LP64-sensitive keyword transport.

## Consequences

- Fairy code can preserve one real C scalar across imported calls without lying about its identity.
- The interpreter still fails closed at the first foreign call and invents no host binding.
- Except for the independently accepted later slices, the target's other C scalar entries remain
  layout and ABI-classifier vocabulary, not source language support.
