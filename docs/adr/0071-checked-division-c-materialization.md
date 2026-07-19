# ADR 0071: Checked division C materialization

- Status: Accepted
- Date: 2026-07-19

## Context

Verified MIR checks a divisor for zero and checks signed minimum divided by negative one before a
division or remainder operation reaches its success block. When both operands originate from source
constants, the constrained C emitter previously preserved those constants in the later C expression.
MSVC correctly diagnosed the visible `42 / 0` expression while compiling with warnings as errors,
even though generated control flow could only reach that expression after the zero check succeeded.

Removing the native smoke case would hide a backend transport defect. Moving the language check into
the C runtime would duplicate the verified MIR contract and make the generated operation's safety
depend on an unrelated helper implementation.

## Decision

1. Verified MIR remains the semantic authority for division-by-zero and signed-division-overflow
   traps. The interpreter and checked control-flow shape do not change.
2. A generated C function containing checked division or remainder declares one function-local
   `volatile int32_t` divisor slot. Immediately before the C operation, the emitter stores the
   already-checked divisor in that slot and uses the volatile read as the C divisor.
3. The slot prevents a host C compiler from treating a source constant as an immediately undefined
   C constant expression. It is private transport and creates no Fairy C volatile type, operation,
   observable side effect, or public ABI.
4. One slot is reused by sequential operations within the generated C function. It is automatic
   storage, so recursive invocations retain independent slots.
5. Backend source tests cover both division and remainder by a constant zero. The Windows host smoke
   compiles and executes the division trap through low and optimized MSVC profiles.

## Consequences

- Checked trap control flow can be compiled by strict host C frontends without exposing unreachable
  constant undefined expressions.
- Division and remainder pay one volatile store and read in the Stage 0 constrained-C backend. This
  correctness-first cost can be revisited only with equivalent cross-compiler evidence.
- The change proves the target-neutral Windows host-smoke subset only. It does not promote a Windows
  target, stabilize a C ABI, or replace the future direct backend.
