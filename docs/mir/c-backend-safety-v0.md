# C Backend Safety v0

- Status: Draft Stage 0 contract
- MIR source: `docs/mir/mir-v0.md`
- Backend decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Target and ABI source: `docs/abi/c-abi-v0.md`

## Role

Generated C is a reference and differential-test transport for a declared `TargetSpec` and `ToolchainSpec`. It is not universally portable C and it is not the Fairy C semantic model. The backend must preserve verified MIR without inheriting C expression-order, signed-overflow, shift, aliasing, layout, foreign-control-flow, compiler-family, libc, or host accidents.

The first planned profile is `x86_64-linux-gnu` compiled and linked through declared Clang and GCC drivers. Emission alone is not conformance or support evidence.

The profile ID now comes from the validated `fairyc.target-spec/v0` object in `internal/target`.
The backend does not read Go `unsafe.Sizeof`, local C compiler probes, or SDK paths to decide target
semantics. The target object reports the existing scalar, flat-resource, and resource-tagged emission subset while leaving all three
maturity claims false.

## Current scalar, flat-resource, and resource-tagged emission slice

`internal/cbackend` currently emits one deterministic ISO C11 translation unit from verified MIR
for a deliberately narrow subset. It accepts `i32`, `bool`, `Option<i32>`, and
`Result<i32, i32>` parameters and internal results, `void` internal results, a zero-argument
`main() -> i32` entry, calls and control flow, scalar tag construction, match and Result `?`,
comparisons, bitwise operations, and the implemented checked arithmetic families. Private numeric C
function names prevent source identifiers from becoming ABI or injection surfaces.

The emitter additionally accepts direct resource values only when every program-level definition is
nonempty, every declaration-ordered field is exactly `i32`, and the verified user drop body is
empty. Resources may be locals, parameters, private function returns, and by-value call arguments.
Numeric translation-unit types and fields prevent source identifiers from becoming C symbols.

Resource-bearing `Option` and `Result` values are accepted when every payload is exactly `i32` or a
direct admitted resource and at least one payload is a resource. Concrete types receive numeric
private tagged structs. `Option` has one payload field and `Result` has separate success and error
fields; inactive fields are aggregate-zero-initialized rather than represented with an uncontracted
C union. Payload extraction must agree with the MIR move marker. Result propagation reconstructs
the enclosing function's declared Result type on the error path.

The emitter rejects empty, non-`i32`, nested, partially initialized, or nonempty-drop resources;
resource-bearing nested tags and other tagged payload combinations; Region operations;
safe borrows; unsafe operations outside the admitted scalar foreign-call form; complete runtime and
standard-library lowering; malformed
destinations; and non-canonical block shapes. Empty-drop discharge produces no invented cleanup
call. This is private transport, not public Fairy ABI or user drop-body execution.
`spec/fixtures/c-emission/manifest-v0.json` currently proves seventeen source-to-emission cases and
repeated byte identity. No C parser, compiler, object writer, assembler, linker, executable,
sanitizer, layout probe, or ABI probe runs in that fixture lane.

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

- The `fairyc.target-spec/v0` description, not host `sizeof(long)` or Go layout, determines scalar,
  pointer, stack, classifier, and object-format facts.
- `internal/layout` now computes checked array and aggregate size, alignment, field offset, and
  padding from that description. The C emitter consumes those results for flat all-`i32` private
  resources, their permitted resource-bearing built-in tags, and nonempty flat source `@repr(C)`
  structs over `u8`, `i32`, `u32`, `usize`, and raw pointers. It emits `_Static_assert` checks for
  `sizeof`, `_Alignof`, and every `offsetof`; public C records preserve their declared tag and field
  names. General nested resources and the remaining contracted `@repr(C)` field families fail closed.
- Bit reinterpretation and potentially unaligned access use a `memcpy`-style transport rather than incompatible lvalue access.
- `Option<i32>` and `Result<i32, i32>` currently use distinct private structs with a `uint32_t` tag
  and `int32_t` payload. Generated C extracts payload only in the selected variant block and traps
  on an invalid tag. This private shape is not `@repr(C)` or a public Fairy ABI.
- Resource-bearing built-in tags use distinct numeric private structs with explicit payload fields.
  Their larger inactive-field representation is a transport choice, not a stabilized language
  layout or ABI promise.
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

The implemented source slice preserves individual `unsafe extern C fn` declarations and
`unsafe {}` or `unsafe fn` call provenance through AST, typed HIR, and verified MIR. The C backend
currently admits exact-name `C` declarations using `i32`, `u8`, `u32`, `usize`, raw pointers to
that bounded scalar set, and raw pointers to admitted nonempty `@repr(C)` structs. It emits one plain prototype and an ordinary call without `pure`, `const`, `restrict`,
`nothrow`, alias, retention, or visibility promises. Compatible repeated declarations of the same
external name coalesce; signature conflicts fail closed. A local preprocessor guard undefines an
included-header macro with that name before the prototype. `main`, generated `fy_`/`FY_` names,
C11 keywords, and identifiers consumed by the generated preamble are rejected. The only current
exceptions to the generated `fy_` namespace are the exact allowlisted internal standard-library
symbols `fy_std_text_utf8_valid_v0` and `fy_std_text_validate_utf8_v0`; the rest of the prefix
remains closed. Typed null and an exact direct-local borrow-to-raw-pointer cast are the only
source-constructed raw pointers. The cast requires unsafe provenance, preserves the pointee type,
forbids mutability strengthening, lowers to a verified MIR raw-address operation, and emits plain
`&local` without `restrict` or `noalias`. Pointer dereference, arithmetic, integer conversion, and
arbitrary casts remain rejected.
Source `@repr(C)` construction, field reads, exact public names, layout assertions, pointer
prototype spelling, and direct-local address passing are implemented, but by-value aggregate ABI,
nested/array/enum fields, `@link_name`, `c.*` scalars, and callbacks remain outside this slice.

The MIR interpreter deliberately returns a `foreign-call` capability error instead of guessing a
host binding. A closed native fixture compiles the generated Fairy caller and a hand-written C
callee as separate Clang/GCC objects in both directions and both profiles. Its four configured
executions are source-to-link-to-process evidence only after the Linux lane actually runs.

Foreign functions are assumed capable of reading, writing, retaining pointers, and invoking unknown code unless a versioned binding contract says otherwise. The backend does not infer restrictive attributes from a declaration name or header annotation alone.

A C `longjmp` or C++ exception crossing generated Fairy C frames is forbidden. Integrations use boundary shims that catch or translate nonlocal control before returning a normal tagged status.

## Differential verification

`internal/cbackendnative` implements the first scalar, flat-resource, and resource-tagged matrix
shape for Linux/amd64 with glibc, Clang and GCC, and low and optimized profiles. Its seventeen cases plan 68 process-status
and exact source-linked drop-trace comparisons for small non-negative returns and canonical trap
category, including both scalar tag families, flat resource transport, and resource-bearing
Option/Result paths. A closed internal instrumentation profile emits conditional active-payload
observer calls keyed by canonical resource type and binding span; production C contains none of
those declarations, calls, or site metadata. Each production unit also has a separate configured
Clang and GCC execution at `-O1` with address and undefined-behavior sanitizers, for 34 planned
sanitizer executions. That path overrides ambient sanitizer options, forbids recovery, requires
empty stdout/stderr, and is not selectable through the product CLI. Output, negative or larger
`i32` returns, user drop-body effects, public layout, ABI, and broader runtime behavior remain
unimplemented. A clean sanitizer run is backend evidence, not the language-semantics oracle.

The same runner additionally plans four scalar and four typed-null pointer/target-scalar
Fairy-source foreign-call crossings: Clang-generated Fairy caller to GCC C callee and the reverse
at low and optimized profiles. Those crossings use the existing separate-object driver and are
counted separately as `foreignCallExecutions` and `foreignPointerCallExecutions` in
`fairyc.native-c-report/v13`; they do not expand the seventeen interpreter-owned differential cases.

`internal/cdriver` and `internal/cruntime` are the one implementation path for both that matrix and
the experimental constrained-C executable builder. The driver fixes the C dialect, warning policy,
optimization profiles, a separate closed sanitizer build mode, time and output bounds, and linked-artifact size limit; the embedded runtime
fixes trap dispatch before binding a caller-specific reserved process-status base. The native lane
adds a Linux-only four-byte drop recorder whose stderr is decoded as conformance data rather than
language output. Sharing this
plumbing prevents test and product compiler flags from silently diverging, but does not promote the
target beyond the evidence supplied by the differential fixtures.

## External references

- WG14 C11 draft N1570: <https://www.open-std.org/jtc1/sc22/wg14/www/docs/n1570.pdf>
- LLVM Language Reference: <https://llvm.org/docs/LangRef.html>
- LLVM IR undefined-behavior guidance: <https://llvm.org/docs/UndefinedBehavior.html>

These references constrain backend transports; they do not become Fairy C's language specification.
