# ADR 0490: Ordinary synchronous C callbacks

- Status: Accepted for prototype
- Date: 2026-11-01
- Contract: [C ABI v0](../abi/c-abi-v0.md#functions-and-callbacks)
- Extends: [ADR 0481](0481-bounded-c-callbacks.md)

## Decision

An unsafe C import may take a penultimate `unsafe extern C fn` parameter followed by
an explicit user-data pointer. Its exact signature has zero through eight value parameters
followed by one user-data parameter; the result is a supported value or `void`.

Values are `u8`, `i32`, `i64`, `u32`, `usize`, `i8`, `i16`, `u16`, `u64`, `isize`,
`f32`, `f64`, target-available `c::*` scalars, and admitted raw pointers. A raw pointer
has one layer over `void`, `u8`, `i32`, `i64`, `u32`, `usize` or an exact C scalar;
mutable `ptr mut ptr mut void` and exact C-scalar `T **` output slots are also admitted.
The user-data parameter has exactly one pointer layer over those same pointees. Its
mutability, pointee and type identity must equal the C import's final parameter.
This does not open standalone foreign signatures for the newer Fairy scalars.

A callback argument names an ordinary non-capturing `unsafe fn` with exactly the
parameter/result types and no declared effects. Existing source syntax is reused.
An internal C adapter preserves the exact function-pointer signature, including `void`
return and stack arguments. No public export or general function value is created.
The original `i32(i32, ptr mut void)` type and generated typedef retain their identity.

The C provider must call synchronously on the same thread and finish before the
import returns. It must not retain callback/data pointers, deliver asynchronously or
across threads, unwind or `longjmp` through Fairy. Compiler signature and nonescape
checks do not prove that an unsafe C implementation obeys those lifetime obligations.
Null data is allowed when unused; callback pointers are non-null.

Callback locals, fields, arrays, results, casts, indirect invocation, captures,
variadics and aggregate-by-value callback signatures remain closed. Raw pointer
operations retain their existing restrictions. Retained/asynchronous callback support
requires a separate lifetime and ownership contract.

## Representation and verification

The builtin type spelling carries the complete bounded signature. Parsing accepts only
canonical supported spellings, at most nine total parameters, and a bounded text length.
Stable interfaces and package bodies preserve that identity without nominal rebinding;
target FunctionID rebinding and dependency tracking reuse the existing opcode contract.
MIR verification checks target/result/parameter identity, no effects, final data pairing,
unique non-null origin and exactly one use without overwrite in its foreign call block.
Emission rejects failed verification. Typedefs have deterministic signature-based names;
one adapter definition is shared across partitioned sources.

## Validation

Focused compiler, semantic, HIR/MIR, stable artifact, process/disk callback-cache and
C-backend checks cover the extension. Native Windows observations are recorded separately
in [the receipt](../ops/windows-ordinary-callbacks-20261101.md). Native Linux execution is
batched in the ordered work list and is not inferred from Linux-profile emission.
Grammar, SDK, DB, dependency, runtime, opcode/codec envelope and CI/runner policy are unchanged.
