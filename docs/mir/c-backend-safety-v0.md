# C Backend Safety v0

- Status: Draft Stage 0 contract
- MIR source: `docs/mir/mir-v0.md`
- Backend decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Target and ABI source: `docs/abi/c-abi-v0.md`

## Role

Generated C is a reference and differential-test transport for a declared `TargetSpec` and `ToolchainSpec`. It is not universally portable C and it is not the Fairy C semantic model. The backend must preserve verified MIR without inheriting C expression-order, signed-overflow, shift, aliasing, layout, foreign-control-flow, compiler-family, libc, or host accidents.

The first planned profile is `x86_64-linux-gnu` compiled and linked through declared Clang and GCC drivers. Emission alone is not conformance or support evidence.

## Current scalar emission slice

`internal/cbackend` currently emits one deterministic ISO C11 translation unit from verified MIR
for a deliberately narrow scalar subset. It accepts `i32` and `bool` parameters, `i32`, `bool`, or
`void` internal results, a zero-argument `main() -> i32` entry, scalar calls and control flow,
comparisons, bitwise operations, and the implemented checked arithmetic families. Private numeric
C function names prevent source identifiers from becoming ABI or injection surfaces.

The emitter rejects resources, tagged values, Region operations, safe borrows, unsafe operations,
FFI, runtime and standard-library lowering, non-scalar signatures, malformed scalar destinations,
and non-canonical block shapes. `spec/fixtures/c-emission/manifest-v0.json` currently proves four
source-to-emission cases and repeated byte identity. No C parser, compiler, object writer, assembler,
linker, executable, sanitizer, layout probe, or ABI probe runs in that fixture lane.

## Ordered evaluation

Nested MIR effects are emitted as separate C statements and temporaries. Function designators and arguments are fully materialized in Fairy C's left-to-right order before the C call expression. Generated code never relies on the C implementation to choose a compatible evaluation order.

## Arithmetic

| Fairy C operation | C transport rule |
| --- | --- |
| Ordinary signed arithmetic | Check with a defined helper or widened/unsigned method, trap on overflow, then construct the result. |
| Wrapping arithmetic | Perform corresponding-width unsigned arithmetic and convert by a specified bit-preserving path. |
| Saturating arithmetic | Use explicit comparisons or helpers with no overflowing intermediate. |
| Checked arithmetic | Return an explicit tagged result without invoking C undefined behavior. |
| Division | Check zero and signed-minimum divided by negative one before the C operation. |
| Shift | Check the amount against the operand width before the C shift. Signed left shift is not a Stage 0 source operation. |

The backend cannot add C or LLVM overflow, exactness, alignment, or alias promises unless the verified MIR fact explicitly proves them.

## Representation and access

- Target descriptions, not host `sizeof(long)` or Go layout, determine widths and alignment.
- Bit reinterpretation and potentially unaligned access use a `memcpy`-style transport rather than incompatible lvalue access.
- `Option` and `Result` use a tag plus active payload representation; generated C never reads an inactive payload.
- Type punning through incompatible C lvalues is forbidden.
- Stage 0 emits no `restrict` and makes no speculative `noalias` promise.
- Padding bytes are never treated as initialized semantic data.

## Traps and cleanup

Checked MIR branches reach the non-returning internal `fy_runtime_trap(uint32_t)` symbol with a fixed
canonical category code. The emitter does not define the product termination mechanism. The native
fixture runtime maps the code to a reserved process status solely for differential observation. A
trap does not unwind through generated C and does not run Fairy C cleanup. Normal cleanup is already
explicit in MIR.

## Foreign calls

Raw foreign signatures contain C ABI scalars, explicitly represented aggregates, and raw pointers. Safe borrows are constructed only in Fairy C wrappers after validation.

Foreign functions are assumed capable of reading, writing, retaining pointers, and invoking unknown code unless a versioned binding contract says otherwise. The backend does not infer restrictive attributes from a declaration name or header annotation alone.

A C `longjmp` or C++ exception crossing generated Fairy C frames is forbidden. Integrations use boundary shims that catch or translate nonlocal control before returning a normal tagged status.

## Differential verification

`internal/cbackendnative` implements the first scalar-only matrix shape for Linux/amd64 with glibc,
Clang and GCC, and low and optimized profiles. Its current process-status oracle compares small
non-negative returns and canonical trap category. Output, negative or larger `i32` returns, drop
traces, sanitizers, layout, ABI, and broader runtime behavior remain unimplemented. Sanitizers are
useful backend checks but are not the language-semantics oracle.

`internal/cdriver` and `internal/cruntime` are the one implementation path for both that matrix and
the experimental scalar executable builder. The driver fixes the C dialect, warning policy,
optimization profiles, time and output bounds, and linked-artifact size limit; the embedded runtime
fixes trap dispatch before binding a caller-specific reserved process-status base. Sharing this
plumbing prevents test and product compiler flags from silently diverging, but does not promote the
target beyond the evidence supplied by the differential fixtures.

## External references

- WG14 C11 draft N1570: <https://www.open-std.org/jtc1/sc22/wg14/www/docs/n1570.pdf>
- LLVM Language Reference: <https://llvm.org/docs/LangRef.html>
- LLVM IR undefined-behavior guidance: <https://llvm.org/docs/UndefinedBehavior.html>

These references constrain backend transports; they do not become Fairy C's language specification.
