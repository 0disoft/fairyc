# Restricted Inert Enum Execution

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- MIR source: `docs/mir/mir-v0.md`

## Context

The grammar and product contract already admitted restricted user enums, but the Stage 0 frontend
discarded every declared variant and payload field during AST lowering. Treating user enums as the
built-in `Option` or `Result` families would be wrong: those types own resource-bearing payload
cleanup, while a user enum is restricted to inert copyable payloads.

## Decision

1. AST, typed HIR, and MIR preserve declaration-ordered enum variants and declaration-ordered named
   payload fields. Duplicate variants or fields are `E0102`.
2. A user-enum payload must be inert and structurally copyable. Resource and safe-borrow payloads
   are `E1004`; stored `Region` capability is `E3103`. General resource-bearing user enums remain
   deferred.
3. A payloadless variant is `Enum::Variant`. A payload variant is
   `Enum::Variant { field: value }`. Public enums may use `alias::Enum::Variant` through one explicit
   module import alias. Construction supplies every declared field exactly once.
4. A shallow match names every payload field, binds those fields only inside its arm, and covers
   every declared variant exactly once. Wildcards, guards, nested patterns, and partial matching are
   outside this slice.
5. MIR owns separate `enum-construct` and `enum-payload` operations plus a multi-way `enum` switch.
   The switch records the concrete enum type and declaration-ordered variant table. Forged variant,
   field, type, or switch-table mutations are rejected as `V1001`.
6. The MIR interpreter executes construction, active-variant selection, named payload extraction,
   and exhaustive matching. It does not borrow built-in tagged cleanup or create a user-enum drop
   path.
7. The constrained C backend emits a private numeric tag and private union payload. The current C
   transport admits payloadless variants and fields over admitted scalars, raw pointers, or flat
   inert structs. Nested user enums and built-in tags inside a user-enum payload fail closed until
   dependency ordering and recursive representation rules exist. No public Fairy or C enum ABI is
   established.
8. `fairyc.c-emission-fixtures/v0` adds one interpreter-approved inert-enum case. The configured
   Linux matrix therefore grows to 18 cases, 72 compiler/profile process and drop-trace executions,
   and 36 sanitizer executions. These are planned counts until a hosted receipt is observed.

## Consequences

- Fairy C source can now construct and execute a real restricted user enum through the Go Stage 0
  compiler, verified MIR, interpreter, and generated-C backend.
- The compiler no longer silently throws away enum payload declarations.
- The private C representation is a backend transport detail and cannot be used as an FFI promise.
- Linux native execution remains configured but unobserved on the Windows development host.
