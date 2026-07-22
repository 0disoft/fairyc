# ADR 0151: Verify internal call linkage before artifact splitting

- Status: Accepted
- Date: 2026-07-22
- MIR source: `internal/mir/verify.go`
- Mutation source: `spec/fixtures/mir-verifier/manifest-v0.json`
- Package-artifact dependency: `docs/adr/0149-verified-mir-artifact-codec.md`

## Context

The MIR verifier checked local control flow, ownership, traps, aggregate definitions, foreign-call
provenance, and non-unwinding calls. It did not require an ordinary internal call's symbol to exist
in the program. The constrained C backend rejected that graph later while building its function
table, so a corrupted serialized MIR artifact could pass decode-time verification and fail only at
backend handoff.

Splitting a whole program into reusable package artifacts makes the distinction between a local
definition and an explicit imported declaration unavoidable. The whole-program verifier must first
prove its existing closed-world linkage instead of relying on one backend to notice a missing edge.

## Decision

1. Program verification builds one internal function-symbol table and rejects missing, duplicate,
   or internal/foreign-colliding symbols as `V1001`.
2. Every ordinary call must resolve to one internal function. Its argument count and presence or
   absence of a result destination must match the callee signature.
3. Foreign calls retain their existing declaration, ABI, source-span, unsafe-scope, and provenance
   checks. Full call-operand type checking remains a backend responsibility in MIR v0.
4. The mutation contract adds one isolated `normal-call` recipe that replaces the callee with an
   absent symbol and requires exactly `V1001` at `terminator:call`.
5. Package artifacts may later introduce explicit imported function declarations, but ordinary
   whole-program MIR remains closed: an absent definition is never inferred as an import.

## Consequences

- `mircodec.Decode` now rejects a missing internal call target before interpreter or backend use.
- The C backend keeps its own linkage checks as defense in depth rather than the first owner of the
  invariant.
- Package-artifact extraction has a clear next boundary: convert cross-package definitions into
  explicit imports instead of weakening whole-program verification.

## Verification

- one direct verifier unit test freezes missing-callee rejection;
- the versioned verifier manifest now contains 22 isolated mutations across 10 baselines;
- full compiler, interpreter, C backend, fixture, and native tests must continue to pass.
