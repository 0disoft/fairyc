# Diagnostic Catalog v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`

Diagnostic codes are stable identifiers for a semantic category. Exact wording and repair hints may improve, but a code cannot silently change meaning. A diagnostic includes a primary span, relevant origin or conflicting spans, the stable code, and a concise explanation.

## Namespaces

| Range | Category |
| --- | --- |
| `E00xx` | Syntax, modules, and imports |
| `E01xx` | Name resolution and source visibility |
| `E10xx` | Type and initialization |
| `E11xx` | Resource ownership and drop |
| `E21xx` | Borrowing and return origins |
| `E31xx` | Regions and arenas |
| `E41xx` | `Option`, `Result`, and propagation |
| `E42xx` | Return-value use obligations |
| `E43xx` | Bounded compile-time evaluation |
| `E45xx` | Explicit generic source preparation |
| `E51xx` | Unsafe and foreign boundaries |
| `E61xx` | Target layout and ABI |
| `E71xx` | Effects and external capabilities |
| `E80xx` | Compatibility and toolchain selection |
| `E90xx` | Test declarations and harness |
| `Axxxx` | Atlas warnings or hard map-integrity errors |
| `Vxxxx` | MIR verifier defects; never ordinary user diagnostics |

## Provisional Stage 0 codes

| Code | Meaning |
| --- | --- |
| `E0001` | Source is not valid UTF-8 or identifier syntax is unsupported. |
| `E0002` | Module declaration does not match its package-relative file identity. |
| `E0003` | Import lacks an alias or uses a forbidden wildcard, selective, cyclic, or re-export form. |
| `E0004` | Integer or floating literal has an invalid radix digit, separator, or excluded spelling. |
| `E0005` | String-family literal has an unknown, truncated, out-of-range, or disallowed escape. |
| `E0006` | C-string literal contains a decoded interior NUL. |
| `E0007` | Parser inserted one required missing token at a zero-width byte span. |
| `E0008` | Parser skipped an unexpected real token into an error node. |
| `E0009` | Comparison operators are chained without explicit parentheses. |
| `E0010` | Shared parser budget is exhausted before a complete bounded result is available. |
| `E0101` | Name or qualified path cannot be resolved in its owning scope. |
| `E0102` | Declaration duplicates a name in the same owning scope. |
| `E0103` | Resolved item is not source-visible from the use module. |
| `E1001` | Source local declaration has no initializer. |
| `E1002` | Inert struct contains a resource. |
| `E1003` | Named aggregate contains a safe borrow. |
| `E1004` | Restricted user enum contains a resource or safe-borrow payload. |
| `E1005` | Byte view is used as text without explicit validation. |
| `E1006` | Expression type is incompatible with its expected type. |
| `E1007` | Condition or logical operand is not `bool`. |
| `E1008` | Assignment target is not a mutable place. |
| `E1009` | Return expression is incompatible with the declared return type. |
| `E1010` | Call argument count does not match the resolved declaration. |
| `E1014` | Call argument label does not match its position or the call target does not support labels. |
| `E1012` | `break` or `continue` has no enclosing loop. |
| `E1013` | Integer literal is outside the range of its selected `i32`, `u8`, `u32`, or `usize` type. |
| `E1101` | Resource used after move or consumption. |
| `E1102` | Operation would create or execute a second cleanup obligation. |
| `E1103` | Resource field move is not supported. |
| `E1104` | Branch join leaves a resource `MaybeLive`. |
| `E1105` | Loop back edge changes the required ownership state. |
| `E1106` | Fallible `drop` or ambiguous close ownership. |
| `E1107` | `@abilities` is empty, duplicated, malformed, names an unknown resource ability, or declares `key` without `store`. |
| `E1108` | Compiler-scheduled implicit destruction reaches a resource whose explicit ability set omits `drop`. |
| `E1109` | Explicit `burn` is not authorized for this value, resource declaration, or module. |
| `E2101` | Access conflicts with an active shared or mutable loan. |
| `E2102` | Returned borrow originates from a local value. |
| `E2103` | Returned borrow has multiple possible input origins. |
| `E2104` | Borrow-returning function cannot become a function value or callback. |
| `E2105` | Borrow outlives or moves its owner. |
| `E3101` | Region-derived value escapes the region. |
| `E3102` | Plain region allocation contains a type that needs drop. |
| `E3103` | Region capability is stored, captured, moved out, or returned. |
| `E3104` | Source attempts to construct a compiler-owned `Region` capability. |
| `E4101` | `Result` value is not handled, returned, propagated, or explicitly discarded. |
| `E4102` | `?` is applied to a non-`Result` value. |
| `E4103` | `?` error type is incompatible with the enclosing return type. |
| `E4201` | `@must_use` is malformed, duplicated, misplaced, or applied to an ineligible result. |
| `E4202` | A resolved `@must_use` call result is ignored without explicit `discard`. |
| `E4301` | A module constant or `@const_eval` declaration is malformed, misplaced, public, or otherwise ineligible. |
| `E4302` | A well-typed expression is outside the admitted compile-time subset. |
| `E4303` | Compile-time evaluation traps on arithmetic, division, shift, or cast semantics. |
| `E4304` | A compile-time dependency cycle or deterministic evaluator budget prevents a result. |
| `E4305` | A module-level `static_assert` evaluates to false. |
| `E4401` | The reserved `source_location` intrinsic name is declared, bound, or used without its exact call form. |
| `E4402` | A `source_location()` call has no valid canonical source path or exact retained line geometry. |
| `E4501` | Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding. |
| `E4502` | A generic call requires supported explicit scalar type arguments with the declared arity. |
| `E4503` | A generic declaration or concrete request exceeds a catalog or expansion limit, or triggers a specialization cycle. |
| `E5101` | Raw-pointer or unchecked operation appears outside an unsafe boundary. |
| `E5102` | Raw foreign signature contains a safe reference. |
| `E5103` | Foreign unwind, exception, or `longjmp` may cross a Fairy C frame. |
| `E5104` | Safe foreign callback is not supported in Stage 0. |
| `E5105` | A retained foreign-pointer API received stack or region-borrowed storage. |
| `E5106` | `@link_name` is duplicated or does not contain exactly one nonempty string literal. |
| `E6101` | Type has no supported ABI classification for the selected target. |
| `E6102` | Required target layout fact is unknown. |
| `E6103` | C ABI v0 excludes the requested calling, layout, callback, TLS, or variadic feature. |
| `E7101` | `@effects` is empty, duplicated, malformed, or names an effect outside the Stage 0 vocabulary. |
| `E7102` | A call or compiler-scheduled resource cleanup requires one or more effects not declared by the enclosing function. |
| `E7201` | A package state declaration has malformed UID authority, invalid visibility, or more than one package owner. |
| `E7202` | A package state field name is declared more than once. |
| `E7203` | A package state field uses an unsupported or malformed storage codec declaration. |
| `E8001` | Selected runtime ABI is incompatible with generated code. |
| `E8002` | Requested language revision is unsupported by this compiler. |
| `E9001` | `@test` must appear exactly once without arguments on a private safe zero-argument free function returning `void` in isolated test source. |
| `A2001` | Dependency map is incompatible; affected Atlas relations are partial or unavailable while compilation remains valid. |
| `A9001` | Deterministic symbol identity collision or corrupt identity evidence; the map is rejected. |

The MIR verifier codes are defined in `docs/mir/mir-v0.md`.

## Diagnostic quality rules

- A move error names the ownership-transfer span.
- A borrow conflict shows the loan creation, conflicting access, and last use when known.
- An origin error names every candidate origin that caused rejection.
- A cleanup error states whether ownership is live, dead, or path-dependent.
- An FFI error names the boundary contract that must contain the unsafe proof.
- An effect error names the rejected declaration or call and relates the callee effect contract.
- A compatibility diagnostic names every mismatched vector member without replacing it with a generic compiler-version error.
- An Atlas warning reports relation-specific completeness and never upgrades unavailable evidence to complete.
- A syntax diagnostic uses a half-open source-byte span, records insertion or skipped-token provenance, and does not claim a recovered file is valid.
- A semantic diagnostic uses a half-open source-byte span, names role-labelled declaration, type, loan, origin, or last-use spans when required, and suppresses errors caused only by a poisoned earlier fact.
- In resolved multi-package analysis, a diagnostic span retains build-local package identity
  separately from its original relative source path. A renderer must never expose the private
  numeric package ID; a future package-aware CLI will supply the user-facing package label.
- Unsupported Stage 0 features should suggest a handle, index, copy, region, split API, or unsafe wrapper only when that alternative actually fits the rejected operation.

## Compatibility

Codes become release-stable only with an implemented diagnostic and compile-fail fixture. Until then, this catalog reserves meanings for the prototype and may be revised through an ADR plus synchronized fixture plan. The semantic v0 resolver, type, local-borrow, whole-resource, Region v1, Result v0, and narrow return-value-use diagnostics are implemented as prototype acceptance behavior.
