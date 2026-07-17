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
| `E51xx` | Unsafe and foreign boundaries |
| `E61xx` | Target layout and ABI |
| `E80xx` | Compatibility and toolchain selection |
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
| `E1101` | Resource used after move or consumption. |
| `E1102` | Operation would create or execute a second cleanup obligation. |
| `E1103` | Resource field move is not supported. |
| `E1104` | Branch join leaves a resource `MaybeLive`. |
| `E1105` | Loop back edge changes the required ownership state. |
| `E1106` | Fallible `drop` or ambiguous close ownership. |
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
| `E5101` | Raw-pointer or unchecked operation appears outside an unsafe boundary. |
| `E5102` | Raw foreign signature contains a safe reference. |
| `E5103` | Foreign unwind, exception, or `longjmp` may cross a Fairy C frame. |
| `E5104` | Safe foreign callback is not supported in Stage 0. |
| `E5105` | A retained foreign-pointer API received stack or region-borrowed storage. |
| `E6101` | Type has no supported ABI classification for the selected target. |
| `E6102` | Required target layout fact is unknown. |
| `E6103` | C ABI v0 excludes the requested calling, layout, callback, TLS, or variadic feature. |
| `E8001` | Selected runtime ABI is incompatible with generated code. |
| `E8002` | Requested language revision is unsupported by this compiler. |
| `A2001` | Dependency map is incompatible; affected Atlas relations are partial or unavailable while compilation remains valid. |
| `A9001` | Deterministic symbol identity collision or corrupt identity evidence; the map is rejected. |

The MIR verifier codes are defined in `docs/mir/mir-v0.md`.

## Diagnostic quality rules

- A move error names the ownership-transfer span.
- A borrow conflict shows the loan creation, conflicting access, and last use when known.
- An origin error names every candidate origin that caused rejection.
- A cleanup error states whether ownership is live, dead, or path-dependent.
- An FFI error names the boundary contract that must contain the unsafe proof.
- A compatibility diagnostic names every mismatched vector member without replacing it with a generic compiler-version error.
- An Atlas warning reports relation-specific completeness and never upgrades unavailable evidence to complete.
- A syntax diagnostic uses a half-open source-byte span, records insertion or skipped-token provenance, and does not claim a recovered file is valid.
- A semantic diagnostic uses a half-open source-byte span, names role-labelled declaration, type, loan, origin, or last-use spans when required, and suppresses errors caused only by a poisoned earlier fact.
- Unsupported Stage 0 features should suggest a handle, index, copy, region, split API, or unsafe wrapper only when that alternative actually fits the rejected operation.

## Compatibility

Codes become release-stable only with an implemented diagnostic and compile-fail fixture. Until then, this catalog reserves meanings for the prototype and may be revised through an ADR plus synchronized fixture plan. The current semantic fixtures are prototype acceptance data and do not claim an implementation exists.
