# Source Link-Name Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- C backend source: `docs/mir/c-backend-safety-v0.md`

## Context

The source FFI slice previously required a Fairy declaration name to equal the external C symbol.
That was enough to prove the first crossing, but it forced raw target-owned spellings into Fairy
wrappers and prevented the already-contracted `@link_name` attribute from reaching executable MIR.
HIR, MIR, and the C backend already carried a separate link-name field; AST lowering discarded the
attribute before semantic analysis could populate it.

## Decision

1. `@link_name("symbol")` applies only to an individual `unsafe extern C fn` declaration. The Fairy
   declaration name remains the source lookup identity; the cooked attribute string is only the
   external link identity.
2. AST attribute arguments preserve raw spelling, lexer-cooked content, and source span. String
   cooked bytes are decoded exactly once at the AST boundary rather than re-parsed by semantic
   analysis.
3. A foreign declaration may contain at most one `@link_name`, with exactly one nonempty string
   literal. A duplicate or malformed attribute is `E5106`. Unknown foreign-function attributes
   remain explicit unsupported semantics.
4. Typed HIR and MIR preserve both names. The MIR verifier requires a nonempty declaration and the
   existing unsafe-call provenance; it does not reinterpret source attributes.
5. The MIR verifier and constrained C backend share one signature-equivalence rule for declarations
   mapped to the same link name. The verifier rejects conflicts as `V1001`. The backend validates
   the external name against its C11 and private-namespace policy, emits that name in the prototype
   and call, and coalesces compatible declarations that map distinct Fairy names to one external
   symbol.
6. The MIR interpreter remains fail-closed for every foreign call. `@link_name` grants no host
   binding or intrinsic authority.
7. Existing scalar and internal text-view native probes now use distinct Fairy and C names. Their
   configured execution counts do not change; hosted native evidence remains unobserved on the
   Windows development host.

## Consequences

- Fairy wrappers can use source-appropriate names without changing the linked C ABI.
- Allowlisted internal `fy_std_*_v0` symbols no longer need to leak into the wrapper's callable name.
- Symbol versioning, platform decoration, ordinals, dynamic lookup, exports, and arbitrary linker
  arguments remain outside this slice.
