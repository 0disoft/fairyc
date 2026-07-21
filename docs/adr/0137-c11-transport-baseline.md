# ADR 0137: Keep ISO C11 as the constrained transport baseline

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`

## Context

The current constrained backend emits deterministic ISO C11 and invokes supported consumers in
explicit C11 mode. References to C11 describe that private transport, not the Fairy C source
language and not a promise to parse C source. C17 adds no needed emission capability for the
current subset, while selecting C23 spelling would narrow consumer availability and couple source
transport to newer headers and keywords without improving Fairy semantics or ABI.

Compiler language mode, generated-source requirements, target ABI, runtime-library availability,
and validation instrumentation are different compatibility axes. Calling all of them a C profile
would make a validation mode look like a new product backend.

## Decision

1. ISO C11 remains the only normative Stage 0 transport dialect and the only product C-emission
   dialect.
2. There is no C17 emitter. Compiling the identical C11 unit in a C17 mode is consumer validation,
   not a separate output profile.
3. There is no C23 emitter. A future pinned C23 consumer lane may compile the identical C11 unit to
   detect forward-compatibility failures, but that lane does not permit C23-only emission.
4. A nonstandard or rolling compiler-latest mode can provide evidence only. It cannot define the
   transport contract or promote a target.
5. The compatibility model separates:
   - transport dialect: minimum syntax required by generated source;
   - consumer mode: compiler family, version, standard mode, and strictness used to validate it;
   - target ABI: layout, calling convention, object, and symbol behavior;
   - runtime baseline: headers and functions available for the selected target runtime;
   - instrumentation: validation-only observers and sanitizer or trap evidence.
6. The product exposes no user-selectable C17 or C23 dialect flag while their output is byte-identical
   to C11 or unsupported.
7. Before enabling a C23 consumer lane, verbatim external C identifiers must be checked against the
   union of keywords reserved by every enabled consumer mode. Internal Fairy names may continue to
   use private mangling and need not be globally restricted for a consumer-only keyword.
8. C dialect changes never silently change external symbol spelling, scalar mapping, pointer
   representation, aggregate layout, return classification, runtime records, or trap symbols. A
   representation change requires its own ABI decision and compatibility version.
9. C standard-library capability belongs to the target runtime and toolchain probe. A newer
   language mode does not prove that a header, allocator, POSIX API, or UCRT function exists.

## C23 output admission gate

A C23 output dialect can be reconsidered only when all of the following are demonstrated:

- every tier-one consumer has an explicitly supported mode or target-specific support is a declared
  product policy;
- C23 emission has a measured correctness, maintenance, code-size, or capability benefit beyond
  cosmetic spelling;
- the external-identifier keyword policy has shipped before the new mode;
- every required header and function has a target-owned capability check;
- C11 and C23 caller/callee crossings preserve layout, symbols, and observed results;
- deterministic fixtures contain output that is materially and intentionally different from C11.

## Consequences

- Current generated source, drivers, target evidence, and native probes keep one small baseline.
- C17 and C23 validation can be added without multiplying backend identities or cache keys for
  byte-identical source.
- Newer compiler modes cannot be mistaken for new Fairy C source features or ABI support.
- The current C11-only keyword checker remains sufficient for current C11 execution. Expanding
  consumer modes creates a separate implementation and language-compatibility task.

## Verification

- the emitter records `c11` as its only accepted dialect;
- GCC and Clang driver arguments require strict C11 for the Linux lane;
- the Windows host driver requests C11 from MSVC;
- target, ABI, runtime, and instrumentation identities remain independent of the source dialect;
- no document or CLI claims a currently implemented C17 or C23 output profile.
