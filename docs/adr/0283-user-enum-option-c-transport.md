# ADR 0283: Carry restricted user enums through built-in Option in generated C

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`
- Depends on: ADR 0029, ADR 0050, ADR 0280, ADR 0282

## Context

ADR 0280 gave restricted inert user enums a target-owned private C layout and admitted that layout
inside built-in `Result`. The same enum remained rejected inside `Option` even though Option needs
only one payload slot and the existing generic tagged-record path already derives every outer size,
alignment, and offset from the target description.

That asymmetry prevented ordinary optional domain values from reaching generated native code. A
compile-only relaxation would not prove that `.some` preserves the enum tag and named payload or
that `.none` avoids reading inactive storage under optimization.

## Decision

1. Admit a restricted inert user enum as the payload of generic built-in `Option` using the exact
   target-owned private tag-plus-union layout established by ADR 0280.
2. Reuse the existing generic Option record, aggregate-zero initialization, target-derived layout
   assertions, construction, extraction, and switching paths. No second enum representation or
   public C ABI is introduced.
3. Add one deterministic source-to-C fixture that constructs `.some(.Value { .amount = 20 })`,
   extracts and exhaustively switches the enum, observes `.none`, and returns 42.
4. Compile, link, inspect, and execute that generated-Fairy program under low and optimized Windows
   MSVC profiles. Both profiles must return 42.
5. Continue rejecting nested enums, built-in tags inside user-enum payloads, recursive enums, and
   resource-bearing user enums.
6. Language remains `fairyc-language/8`, SDK remains `0.0.0-stage0.25`, runtime ABI remains
   `fairyc-runtime/0`, and no public or private native symbol changes.

## Consequences

- `Option<UserEnum>` can now represent absence or preserve an exhaustive domain value through the
  interpreter, deterministic C emission, and both Windows native optimization profiles.
- The change removes an unnecessary outer-tag asymmetry while retaining one target-owned enum
  representation and the existing fail-closed payload restrictions.
- This is private generated-C transport, not a promise of public tagged layout or foreign ABI.

## Evidence

- `internal/cbackend/tagged_types.go` admits the same restricted user-enum payload classifier for
  generic Option and Result.
- `spec/fixtures/c-emission/cases/pass/copy-option-enum/src/main.fy` owns the interpreter and
  deterministic source-to-C case.
- `cmd/fairyc-windows-host-smoke/enum_result_sources.go` owns the bounded native source input.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes the program under low and
  optimized MSVC profiles.
