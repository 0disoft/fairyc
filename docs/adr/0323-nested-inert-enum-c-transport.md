# ADR 0323: Nested inert-enum C transport

- Status: Accepted
- Date: 2026-08-09
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0050, ADR 0319

## Decision

1. Admit an acyclic restricted inert user enum as a by-value payload field of another restricted
   inert user enum in the private constrained-C transport.
2. Reuse the aggregate layout dependency graph to resolve the nested enum layout and the aggregate
   emission graph to place the complete nested definition before its containing enum, even when
   canonical root-name order would place the container first.
3. Preserve source construction, by-value function transport, named outer and inner payload
   extraction, and exhaustive nested `switch` behavior through verified MIR and generated C.
4. Keep recursive aggregate cycles fail-closed. Resource-bearing user enums, built-in `Option` or
   `Result` fields inside user enums, nested enums in `@repr(C)` records, and a public tagged ABI
   remain outside this slice.
5. Keep language, SDK, runtime ABI, target schema, MIR, cache, lockfile, and public API identities
   unchanged. This is an additive private backend capability for already-valid Stage 0 source.

## Evidence

- The 46th constrained C-emission fixture constructs `Envelope::Present(State::Code(42))`, passes
  the outer enum by value, extracts both payload layers, exhaustively switches both enums, and
  requires the nested `State` C definition before the containing `Envelope` definition.
- The C-backend unit test checks interpreter result 42, required nested C field spelling, and
  dependency-ordered complete definitions.
- A Windows-amd64 Go regression invokes the real `experimental run-host` path at low and optimized
  profiles. Both generated-Fairy MSVC executables return 42 with empty stderr.

## Consequences

Programs can model one more ordinary closed sum-of-sums shape without flattening the inner enum or
inventing a struct wrapper. Layout and emission still reject cycles before producing C. The
generated representation remains compiler-private and target-derived; it is not a stable native or
C ABI promise. The configured Linux differential lane will consume the new fixture when run on its
declared glibc/Clang/GCC host, but no Linux execution is claimed from this Windows machine.
