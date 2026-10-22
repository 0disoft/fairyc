# ADR 0476: Repr(C) struct foreign arguments

- Status: Accepted for prototype
- Date: 2026-10-23
- ABI source: `docs/abi/c-abi-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Supersedes: ADR 0061's rejection of foreign record parameters, for the record shapes below only

## Decision

Admit by-value foreign C arguments whose type is a nonempty acyclic `@repr(C)` record already
accepted by the backend's field and layout rules. Emit the exact public C struct tag in the
prototype and pass the typed record value. The C compiler owns register/stack placement; the
emitter must not flatten fields, pass a pointer instead, or select a host-dependent surrogate.
MIR retains the nominal operand type and existing evaluation order. These copy-only records do
not transfer resource ownership. A callee's changes to its parameter do not modify the caller's
record. Argument support is reported by the `foreign-c-struct-argument` emission feature.

Private non-`@repr(C)` records remain rejected. This decision adds no resource, enum, array,
safe-borrow, tagged-value or callback foreign argument representation and no new repr(C) field
family. Existing foreign record results and target-derived layout assertions remain unchanged.
There is no grammar, language-version, runtime-symbol or public CLI change, and no target-maturity
promotion.

## Execution evidence

Extend the shared `internal/foreignstructprobe` source used by the existing Linux native lane
and Windows native smoke. It retains both original return observations and adds:

- An 8-byte record argument and result. The C callee validates both fields and a trailing scalar,
  changes its copy, and returns it; Fairy checks both the changed result and the unchanged original.
- A 20-byte record argument and hidden-memory result, with scalar arguments before and after it.
  Every field is checked, and the original stays unchanged after the returned copy is modified.
- A nested record with an existing `c::double` field, after six integer arguments and before a
  scalar tail. On SysV AMD64 this checks mixed INTEGER/SSE argument rollback when integer registers
  are exhausted. The independent C callee checks every scalar, nested field and floating value.

Linux runs generated-Fairy caller/hand-written C callee in both Clang/GCC directions at low and
optimized profiles. Windows uses separate MSVC caller/callee objects in C11/C17 consumer modes
and low/optimized profiles. The shared probe still exits 42 only when all checks pass. The
existing `foreignCStructResultCallExecutions` counter and report schema remain unchanged: each
execution still checks both original results and now also checks arguments. A historical receipt
with that counter does not attest the new argument checks; the source revision must include this
decision's fixture.

Local emission/package tests pass. Native execution results are recorded in the
[F1 implementation update](../development/phase2-target-scope.md#f1-implementation-update).
Hosted Linux execution of the extended fixture is pending; no runner or workflow is added.
