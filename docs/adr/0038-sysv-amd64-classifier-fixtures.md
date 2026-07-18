# ADR 0038: Execute the Stage 0 SysV AMD64 classifier contract

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Layout source: `internal/layout/`
- ABI source: `docs/abi/c-abi-v0.md`
- Fixture source: `spec/fixtures/c-abi/manifest-v0.json`
- Implementation source: `internal/cabi/` and `internal/cabifixture/`

## Context

The target description already names SysV AMD64 aggregate and return classifiers, and the layout
engine computes exact size, alignment, and field offsets. A string identity plus correct layout is
not a call contract. Small aggregates can combine integer and floating eightbytes, register
exhaustion can force one whole aggregate to the stack, and a memory return consumes the first
integer argument register through a hidden result pointer.

Delegating the current C backend to Clang or GCC does not remove the need for an executable target
model. Later native backends, ABI diagnostics, bindgen, cache identity, and conformance reports must
agree on the target rule before asking a local compiler. At the same time, a Go classifier cannot
be called native ABI evidence until independent compiler-built callers and callees cross the same
boundary.

## Decision

1. `internal/cabi` accepts only a validated `x86_64-linux-gnu` target whose calling, aggregate, and
   return classifier identities exactly name the Stage 0 SysV AMD64 profile. It never consults Go
   host layout, `unsafe`, a local compiler, or an SDK path.
2. The closed input vocabulary is the existing C-representable `layout.Type` subset: fixed-width
   and C scalars, `f32`, `f64`, raw and function pointers, positive fixed arrays, nonempty naturally
   aligned structs, and explicitly integer-based C enums. Vectors, x87, unions, packed/custom
   alignment, bit fields, atomics, and variadics remain outside the classifier.
3. Integer, boolean, enum, and pointer storage classifies as `INTEGER`; `f32`, `f64`, `c.float`,
   and `c.double` classify as `SSE`. Aggregates of at most two eightbytes recursively merge field
   classes at target-computed offsets, with `INTEGER` dominating `SSE` inside one eightbyte.
   Larger admitted aggregates classify as `MEMORY`. The excluded vector vocabulary means v0 has
   no `SSEUP`, x87, or complex-x87 result.
4. Fixed signatures assign integer arguments through `rdi`, `rsi`, `rdx`, `rcx`, `r8`, and `r9`,
   and SSE arguments through `xmm0` to `xmm7`. If every class of one argument cannot fit, the whole
   argument moves to the stack and consumes no partial register assignment; later parameters may
   still use the unconsumed registers.
5. Register returns use `rax`/`rdx` for integer classes and `xmm0`/`xmm1` for SSE classes. A
   `MEMORY` result uses a hidden pointer in `rdi`, shifts later integer parameters, and returns that
   pointer in `rax`. Void results have no class or register.
6. `fairyc.c-abi-fixtures/v0` owns the repository input and expected-plan vocabulary. Seventeen
   canonical cases cover three fail-closed inputs and fourteen pass cases for scalar, pointer, enum,
   function-pointer, array, homogeneous, mixed, hidden-return, exhaustion, and rollback behavior.
   The schema reuses the layout fixture's recursive type definition instead of copying it.
   Direct array parameters and returns fail because C adjusts array parameters to pointers and
   forbids array return types; fixed arrays cross only inside a C-representable struct.
7. `internal/cabifixture.NativeProbeCases` exposes isolated copies of pass cases only after all
   sixteen cases match. This is preparation for a separate bidirectional native probe, not proof
   that such a probe ran.
8. Exact stack argument-area offsets, call-site padding, assembly/object inspection, source-level
   `@repr(C)` and `extern C` lowering, callbacks, and hosted Clang/GCC crossings remain later gates.

## Consequences

- The target's classifier identity now has host-independent executable meaning instead of being a
  decorative string.
- Register exhaustion and hidden-return interactions can be regression-tested without relying on
  the current machine or parsing compiler-specific assembly.
- The fixture contract is repository evidence, not a public FFI schema, stable native Fairy ABI,
  or target-conformance receipt.
- `bidirectional-c-abi-fixtures` remains missing target evidence until generated callers and callees
  cross compiler boundaries on an eligible Linux host.

## Rejected alternatives

- Treat every aggregate of at most sixteen bytes as two integer registers: breaks floating and
  mixed aggregates.
- Spill only the class that exhausts its register family: violates whole-argument rollback and can
  corrupt later parameter assignment.
- Ask the host compiler for classification during ordinary compilation: makes target semantics
  depend on local paths, versions, and host availability.
- Promote the target after fixture success: confuses an internal model with independent native ABI
  execution.

## External reference

- x86-64 psABI project: <https://gitlab.com/x86-psABIs/x86-64-ABI>
