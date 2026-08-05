# ADR 0280: Carry restricted user enums through built-in Result in generated C

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`
- Depends on: ADR 0029, ADR 0050, ADR 0118

## Context

Verified MIR and the interpreter already preserve restricted inert user enums, including their
declaration-ordered variants and named payload fields. The C backend also emits those enums as a
private `u32` tag plus a C union, but it previously rejected the same enum when it appeared as either
payload of built-in `Result`. That gap forced public APIs such as `std::fs::kind` to use opaque
predicate-based values even though the source type itself was inert and copyable.

Admitting the outer Result by host `sizeof`, a guessed maximum payload, or sequential storage would
make generated C target-dependent or unnecessarily large. The target layout engine therefore needs
to model the existing private union representation directly.

## Decision

1. Add target-owned union layout. Every member begins at offset zero; size and alignment are the
   maxima of member layouts, and final size is rounded to union alignment. Empty unions remain
   rejected.
2. Compute each restricted user enum as `{ u32 tag; union payload; }`. The union always includes the
   existing `u8 unused` member and one declaration-ordered struct member for every payload-bearing
   variant. Enum fields remain limited to admitted scalars, raw pointers, and admitted inert structs.
3. Admit that computed private enum representation as either success or failure payload of generic
   built-in `Result`. User-enum `Option`, nested enums, built-in tags inside enum payloads, and
   resource-bearing user enums remain rejected.
4. Emit target-derived size, alignment, and top-level field-offset assertions for the enum and keep
   the existing outer Result assertions. No private C spelling becomes public ABI.
5. Add one source-to-C fixture that executes both `.ok(user-enum)` and `.err(inert-struct)` branches,
   and execute the same generated-Fairy behavior under low and optimized Windows MSVC profiles.
6. Language remains `fairyc-language/8`, SDK remains `0.0.0-stage0.24`, and runtime ABI remains
   `fairyc-runtime/0` because this change closes a backend capability gap without changing source or
   public standard-library contracts.

## Consequences

- Generated C can now carry the public-shape pattern needed by a future enum-returning API without
  weakening deterministic layout or copying active payloads through an unrelated representation.
- Existing opaque `std::fs::FileKind` does not change automatically. Migrating a published SDK type
  requires a separate compatibility decision and SDK revision.
- ADR 0281 later makes that decision, retains the predicate helpers, and promotes `FileKind` to an
  exhaustive enum under SDK `0.0.0-stage0.25`.
- The private enum union can be larger than its active variant, as ordinary C union layout requires,
  while inactive Result payload fields remain aggregate-zero-initialized.
- ADR 0283 later admits restricted user enums inside `Option`; general nested tags, recursive enums,
  and resource-bearing user enums remain outside the admitted C subset.

## Evidence

- `internal/layout/layout.go` owns target-independent union size, alignment, and offset calculation.
- `internal/cbackend/emit.go` computes and asserts private user-enum layout.
- `internal/cbackend/tagged_types.go` admitted only Result-contained user enums at this decision;
  ADR 0283 later extends the same classifier to Option.
- `spec/fixtures/c-emission/cases/pass/copy-result-enum-success/src/main.fy` covers interpreter and
  deterministic source-to-C behavior for success and failure branches.
- `fairyc_windows_host_smoke` compiles, links, inspects, and executes the generated C at low and
  optimized MSVC profiles.
