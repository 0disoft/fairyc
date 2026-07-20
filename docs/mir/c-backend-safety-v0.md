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

## Current scalar, borrowed-view, flat-resource, and resource-tagged emission slice

`internal/cbackend` currently emits one deterministic C11 translation unit from verified MIR
for a deliberately narrow subset. It accepts `i32`, `bool`, `Option<i32>`, and
`Result<i32, i32>` parameters and internal results, `void` internal results, a zero-argument
`main() -> i32` entry, calls and control flow, scalar tag construction, match and Result `?`,
comparisons, bitwise operations, and the implemented checked arithmetic families. Private numeric C
function names prevent source identifiers from becoming ABI or injection surfaces.

Checked `u8`, `u32`, and `usize` addition, subtraction, and multiplication are emitted only after
the verified typed trap edge. The backend tests preconditions against `UINT8_MAX`, `UINT32_MAX`, or
`UINTPTR_MAX` before evaluating the source operation: addition uses `left > max - right`, subtraction
uses `left < right`, and nonzero multiplication uses `left > max / right`. The continuation performs
the unsigned C operation only after those checks, so neither wraparound nor optimizer behavior can
erase the Fairy C `integer-overflow` meaning. Marker, switch, operand, and result type drift fails
closed before C text is produced.

Checked division and remainder use a function-local volatile divisor slot whose C type matches the
verified `i32`, `u8`, `u32`, or `usize` operand type. Every type checks zero before the operation;
only `i32` emits the signed-minimum/negative-one check. This prevents strict C compilers from
diagnosing an unreachable constant-zero expression without erasing typed MIR semantics.

Typed unary `~` and matching-type `&`, `^`, and `|` preserve the exact verified `i32`, `u8`, `u32`, or `usize` result.
The signed path uses explicit bit-preserving `uint32_t` transport; unsigned paths use their exact
unsigned representation directly. Right shift first checks the amount against 32 for `i32`, 8 for
`u8`, 32 for `u32`, or 64 for the admitted `usize` targets. Only the non-trapping continuation emits
the C shift. `i32` uses the defined arithmetic-shift helper; unsigned types use a direct logical shift.
Unsigned unary complement is explicitly cast back to the verified `uint8_t`, `uint32_t`, or
`uintptr_t` representation after C integer promotion; `i32` retains bit-preserving `uint32_t` transport.

The same internal-value subset now admits compiler-owned immutable `&[u8]` literals. MIR retains
their exact bytes and logical length, including empty and embedded-NUL values. Generated C uses a
private `fy_byte_slice { const uint8_t *data; uintptr_t length; }`, exact deduplicated immutable
storage for nonempty values, and a one-byte immutable sentinel with length zero for empty values.
This private shape is neither a public Fairy layout nor a foreign-call ABI.

The same private view transport admits verified `view-length` and `view-empty` operations for safe
borrows of `str` and `[u8]`. Generated C either copies the descriptor's `length` member into a
`uintptr_t`-backed Fairy `usize` or compares it with zero and normalizes a Fairy `bool`; it does not
call a helper, scan storage, allocate, trap, or expose the descriptor layout.
The verifier fixes the owner family and result type before emission.

Verified immutable `&[u8]` indexing compares the `uintptr_t`-backed index against the descriptor's
byte length, routes failure to the canonical `FY_TRAP_BOUNDS` path, and only then emits one
`data[index]` load into a `uint8_t`-backed Fairy `u8`. Receiver and index operands are evaluated once
before the check and reused unchanged by the load. The C backend rejects missing or mismatched MIR
check, owner, index, result, operator, or trap facts instead of emitting unchecked access.

Internal generated functions may accept those exact safe borrowed views by value. Calls pass the
private descriptor unchanged, preserving data and byte length without granting ownership or
creating a foreign ABI. Function declarations, call operands, and inferred local types must agree;
foreign declarations continue to reject safe borrows before emission.

The foreign-call transport additionally accepts all sixteen exact TargetSpec C scalars as results
and parameters. HIR and MIR keep those types distinct from coincidentally represented Fairy
scalars; one ordered backend table maps them to exact C spellings, TargetSpec keys, features, and
optional providers. Generated C asserts target size and alignment for every used scalar and
signedness for signed and unsigned integer families. `ssize_t` selects `_POSIX_C_SOURCE 200809L`
before every header rather than silently relying on the host's default feature namespace. This
slice deliberately creates no immediate C-scalar MIR operand: a foreign producer result local may
be passed to a matching foreign consumer, while literals, operators, comparisons, casts,
interpreter execution, and raw pointers to C scalars remain closed. Exact C-scalar fields are
accepted only in nonempty `@repr(C)` structs and share the same transport table and assertions.
An admitted struct may contain another acyclic nonempty `@repr(C)` struct by value. The backend
orders complete definitions by dependency and reuses the child target layout when computing the
containing field; recursive value dependencies and non-`@repr(C)` nested structs fail closed.
The same admitted records may be foreign results by value. MIR retains the nominal destination type
and generated C uses the exact public tag plus ordinary assignment. Foreign aggregate parameters
remain closed. The native matrix includes an 8-byte register result and a 20-byte hidden-memory
result from a separately compiled C callee.

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
safe-borrow operations other than the exact verified view metadata, immutable byte indexing, and
immutable byte slicing subset; unsafe operations
outside the admitted scalar foreign-call form; complete runtime and
standard-library lowering; malformed
destinations; and non-canonical block shapes. Empty-drop discharge produces no invented cleanup
call. This is private transport, not public Fairy ABI or user drop-body execution.
`spec/fixtures/c-emission/manifest-v0.json` currently proves 31 source-to-emission cases and
repeated byte identity. No C parser, compiler, object writer, assembler, linker, executable,
sanitizer, layout probe, or ABI probe runs in that fixture lane.

The C11 dialect is extension-free for programs using only ISO-owned type families. Selecting
target `c::ssize_t` deliberately changes the provider profile: generated C fixes
`_POSIX_C_SOURCE 200809L` before headers and includes `<sys/types.h>`. No other source construct may
silently enable that namespace.

## Ordered evaluation

Nested MIR effects are emitted as separate C statements and temporaries. Function designators and arguments are fully materialized in Fairy C's left-to-right order before the C call expression. Generated code never relies on the C implementation to choose a compatible evaluation order.

## Arithmetic

| Fairy C operation | C transport rule |
| --- | --- |
| Ordinary signed arithmetic | Check with a defined helper or widened method, trap on overflow, then construct the result through corresponding-width unsigned arithmetic and a bit-preserving conversion. Generated C never evaluates the signed operation itself. |
| Wrapping arithmetic | Perform corresponding-width unsigned arithmetic and convert by a specified bit-preserving path. |
| Saturating arithmetic | Use explicit comparisons or helpers with no overflowing intermediate. |
| Checked arithmetic | Return an explicit tagged result without invoking C undefined behavior. |
| Division and remainder | Check zero and signed-minimum divided by negative one before the C operation. Materialize the verified divisor through a function-local volatile C transport slot so a host C compiler cannot reject or fold an otherwise unreachable constant-undefined expression. |
| Shift | Check the amount against the exact operand width before any C shift. Right shift uses the bit-preserving arithmetic helper for `i32` and exact unsigned C operations otherwise. Left shift then checks mathematical representability, reports overflow only after a valid amount, performs signed transport through `uint32_t`, and reconstructs `i32` from bits. |
| Value-preserving integer cast | Verify one admitted source/result pair, then emit an explicit C cast to `int32_t`, `uint32_t`, or `uintptr_t`. Every source value is representable, so no trap, truncation, wrapping, or implementation-defined signed conversion occurs. |
| Typed scalar and raw-pointer comparison | Verify two matching operands and the exact `bool` result before emission. Scalar equality is limited to `bool`, `i32`, `u8`, `u32`, and `usize`; scalar ordering excludes `bool`. Matching raw pointers admit equality only, with no dereference or ordering. Generated C normalizes the result to `INT32_C(0)` or `INT32_C(1)`. |
| Scalar fixed-array construction and indexing | Verify one canonical length inside the provisional `1..=4096` executable bound and the exact copy-scalar element type. Emit a private assignable struct wrapper, materialize list elements left to right, check `usize` against the retained constant length, and access `.data[index]` only on the success edge. The bound and wrapper are implementation guards, not a public Fairy or C ABI. |
| Assert | Verify one exact `bool` operand and canonical failure edge, compare with zero in every profile, and dispatch `FY_TRAP_ASSERTION_FAILED` through the trap runtime. Never emit or include the C library `assert` macro. |

The backend cannot add C or LLVM overflow, exactness, alignment, or alias promises unless the verified MIR fact explicitly proves them.
The volatile divisor slot is private generated-C transport. It does not add source-visible volatile
semantics, and it is emitted only in functions containing checked division or remainder.

## Representation and access

- The `fairyc.target-spec/v0` description, not host `sizeof(long)` or Go layout, determines scalar,
  pointer, stack, classifier, and object-format facts.
- `internal/layout` now computes checked array and aggregate size, alignment, field offset, and
  padding from that description. The C emitter consumes those results for flat all-`i32` private
  resources, their permitted resource-bearing built-in tags, and nonempty source `@repr(C)` structs
  over `u8`, `i32`, `u32`, `usize`, exact C scalars, admitted raw pointers, and acyclic nested
  `@repr(C)` records. It emits dependencies before their users plus `_Static_assert` checks for
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
- A folded direct `i32` minimum literal emits `INT32_MIN`; generated C never relies on the
  implementation-defined type or conversion of a positive decimal `2147483648` token.
- A bounded scalar fixed array emits as a private struct containing one exact C array so ordinary
  internal assignments, parameters, and results preserve Fairy value semantics. Raw C arrays,
  field embedding, foreign signatures, and layout stability are not inferred from this wrapper.
- Stage 0 emits no `restrict` and makes no speculative `noalias` promise.
- Padding bytes are never treated as initialized semantic data.
- Immutable byte-view slicing emits both `start > end` and `end > length` checks before any C
  pointer addition. Only the success continuation constructs the private `{ data + start,
  end - start }` view, so neither unsigned subtraction nor pointer formation can precede the
  dominating `bounds` decision. An omitted start reaches this path as `usize` zero; an omitted end
  is a prior metadata read from the already-materialized receiver, never a second receiver
  evaluation.

## Traps and cleanup

Checked MIR branches reach the non-returning internal `fy_runtime_trap(uint32_t)` symbol with a fixed
canonical category code. The emitter does not define the product termination mechanism. The native
fixture runtime maps the code to a reserved process status solely for differential observation. A
trap does not unwind through generated C and does not run Fairy C cleanup. Normal cleanup is already
explicit in MIR.

## Foreign calls

Raw foreign signatures contain C ABI scalars, explicitly represented aggregates, and raw pointers. Safe borrows are constructed only in Fairy C wrappers after validation.

The implemented source slice preserves individual `unsafe extern C fn` declarations, one optional
nonempty `@link_name("symbol")`, and
`unsafe {}` or `unsafe fn` call provenance through AST, typed HIR, and verified MIR. The C backend
currently admits `C` declarations using `i32`, `u8`, `u32`, `usize`, every exact TargetSpec C
scalar, raw pointers to that bounded Fairy scalar set or opaque `void`, raw pointers to admitted
nonempty `@repr(C)` structs, and one pointer-to-opaque-pointer output-slot shape. It emits one plain prototype and an ordinary call without `pure`, `const`, `restrict`,
`nothrow`, alias, retention, or visibility promises. Compatible repeated declarations of the same
external name coalesce even when their Fairy names differ; signature conflicts fail closed. A local preprocessor guard undefines an
included-header macro with that name before the prototype. `main`, generated `fy_`/`FY_` names,
C11 keywords, and identifiers consumed by the generated preamble are rejected. The only current
exceptions to the generated `fy_` namespace are the exact allowlisted internal standard-library
symbols for SystemHeap allocate/deallocate, Buffer initialize/zeroed-resize/drop, Arena initialize,
Region begin/zeroed-allocation/reset, Arena drop, text validation/output initialization, bounded
CStr validation, String copy/view/append/drop, CString from-text/view/drop, and Linux File
open/read/write-all/sync/close; direct File write/drop and the rest of the prefix remain closed.
Typed null and an exact
direct-local borrow-to-raw-pointer cast are the only
source-constructed raw pointers. The cast requires unsafe provenance, preserves the pointee type,
forbids mutability strengthening, lowers to a verified MIR raw-address operation, and emits plain
`&local` without `restrict` or `noalias`. Pointer dereference, arithmetic, integer conversion, and
arbitrary casts remain rejected. A separate verified `byte-slice-data-address` operation may expose
the immutable data pointer of a compiler-owned `&[u8]` literal or typed local inside unsafe code;
it cannot strengthen mutability or pass the safe slice itself through a foreign signature.
Successful emission also records the sorted unique external link names used by actual foreign call
terminators. Uncalled declarations remain prototypes but are absent from this link-requirement list.
The exact standard-library namespace and component dependency mapping live in one shared contract
consumed by both C-emission admission and product-build source selection.
The sole nested-pointer exception is exactly `*mut *mut void`, so `&mut *mut void` can become a C
`void **` output slot. Immutable outer or inner layers fail closed instead of risking incorrect C
qualifier placement. Nested pointers to C scalars, structs, or other pointer types remain rejected.
Source `@repr(C)` construction, field reads, exact public names, layout assertions, pointer
prototype spelling, direct-local address passing, opaque `void` pointers, exact C-scalar fields,
and acyclic nested `@repr(C)` fields are implemented, but by-value aggregate ABI, nested enum or
built-in-tag payloads, array fields, C-scalar value operations, non-`@repr(C)` C-scalar fields, and
callbacks remain outside this slice.

The MIR interpreter deliberately returns a `foreign-call` capability error instead of guessing a
host binding. A closed native fixture compiles the generated Fairy caller and a hand-written C
callee as separate Clang/GCC objects in both directions and both profiles. Its four configured
executions are source-to-link-to-process evidence only after the Linux lane actually runs.

Foreign functions are assumed capable of reading, writing, retaining pointers, and invoking unknown code unless a versioned binding contract says otherwise. The backend does not infer restrictive attributes from a declaration name or header annotation alone.

A C `longjmp` or C++ exception crossing generated Fairy C frames is forbidden. Integrations use boundary shims that catch or translate nonlocal control before returning a normal tagged status.

## Differential verification

`internal/cbackendnative` implements the first scalar, borrowed-view, inert-enum, flat-resource, and resource-tagged matrix
shape for Linux/amd64 with glibc, Clang and GCC, and low and optimized profiles. Its twenty cases plan 80 process-status
and exact source-linked drop-trace comparisons for small non-negative returns and canonical trap
category, including both scalar tag families, flat resource transport, and resource-bearing
Option/Result paths. A closed internal instrumentation profile emits conditional active-payload
observer calls keyed by canonical resource type and binding span; production C contains none of
those declarations, calls, or site metadata. Each production unit also has a separate configured
Clang and GCC execution at `-O1` with address and undefined-behavior sanitizers, for 40 planned
sanitizer executions. That path overrides ambient sanitizer options, forbids recovery, requires
empty stdout/stderr, and is not selectable through the product CLI. Output, negative or larger
`i32` returns, user drop-body effects, public layout, ABI, and broader runtime behavior remain
unimplemented. A clean sanitizer run is backend evidence, not the language-semantics oracle.

The same runner additionally plans four scalar, four exact `c::int`, four negative-preserving
`c::long`, four exact `c::size_t`, four negative-preserving `c::ssize_t`, four complete twelve-type
C-scalar-family, and four typed-null pointer/target-scalar Fairy-source foreign-call crossings:
Clang-generated Fairy caller to GCC C callee and the reverse at low and optimized profiles. Those
crossings use the existing separate-object driver and are counted separately as
`foreignCallExecutions`, `foreignCIntCallExecutions`, `foreignCLongCallExecutions`,
`foreignCSizeTCallExecutions`, `foreignCSSizeTCallExecutions`,
`foreignCScalarFamilyCallExecutions`, and `foreignPointerCallExecutions` in the current native
report; they do not expand the twenty interpreter-owned differential cases.
Every C-scalar probe uses hand-written C producers and consumers so each value crosses a generated
typed local without requiring a Fairy C literal. The family probe uses exactly representable finite
floating values and makes no NaN, infinity, signed-zero, rounding, or floating-environment claim.
`fairyc.native-c-report/v31` separately records four `foreignCStructResultCallExecutions`, four
`systemHeapSourceCallExecutions`, four
`bufferSourceCallExecutions`, four `arenaSourceCallExecutions`, and four
`cStrValidationSourceCallExecutions`, plus four `ownedTextSourceCallExecutions` and four
`linuxFileSourceCallExecutions`. The SystemHeap count
crosses an exact generated `void *`/`size_t` block into the real SystemHeap C unit. The Buffer count
nests that block in the exact Buffer record, calls the real
initialize, zeroed-resize, and drop helpers, and observes initialized bytes plus normalized release.
The Arena count crosses exact self-linked node, Arena, and Region records into the real Arena unit,
observes zeroed allocation through `void **`, and verifies reset/drop generation transitions. The
CStr count crosses the exact view into the owned-text C unit, distinguishes valid, interior-NUL, and
unterminated storage, and observes failure-atomic output. The owned-text count crosses exact nested
String/CString and result records into real copy/view/append/drop and from-text/view/drop helpers,
then observes bytes, termination, success statuses, and normalized release. The Linux File count
crosses exact result, handle, and open-option records into real open, write-all, sync, close, read,
and close helpers; it observes exact counts, repeated-close status, EOF, reconstructed bytes, and
normalized closed state. These are internal transport probes, not public allocator, Buffer, Arena,
lexical Region, safe text, or safe File APIs.
Four `standardLibraryBuildSmokeExecutions` additionally require the ordinary product CLI to select
the text-view C component from an actual called symbol, link it with the scalar runtime, and execute
the resulting uninstrumented product under both compilers and profiles.
Four `byteStringLiteralSourceCallExecutions` require a generated Fairy caller to construct a
private `&[u8]` view over exact non-terminated `fairy` storage, preserve length five, extract the
typed local's immutable data address, and have a separate C callee verify all five bytes. Four
`byteStringLiteralBuildSmokeExecutions` separately require the ordinary product CLI to carry that
view into an executable returning 42 under both compilers and profiles. Empty literals instead use
the non-null one-byte sentinel with logical length zero.
Four `utf8StringLiteralSourceCallExecutions` require a generated Fairy caller to construct a private
`&str` view over exact `Fairy λ` UTF-8 storage, preserve byte length eight, extract a typed local's
immutable data address, and have a separate C callee verify all eight bytes. Four
`utf8StringLiteralBuildSmokeExecutions` separately require the ordinary product CLI to carry that
view into an executable returning 42 under both compilers and profiles. Empty text uses a non-null
one-byte sentinel with logical length zero, and invalid decoded UTF-8 never reaches MIR.
Four `cStringLiteralSourceCallExecutions` require a generated Fairy caller to create exact
`fairy\0` storage, cross the dedicated unsafe immutable-byte-pointer operation, and have a separate C
callee verify all six bytes. Four `cStringLiteralBuildSmokeExecutions` separately require the
ordinary product CLI to carry nonempty compiler-owned `&CStr` storage into an executable returning
42 under both compilers and profiles. The C emitter materializes each unique byte sequence once as
sorted translation-unit `static const uint8_t[]` storage; MIR never delegates literal meaning to C
string-token spelling.
The struct-result count requires a generated Fairy caller to receive both a register-sized and a
hidden-memory result from a separately compiled hand-written C callee in both compiler directions
and profiles; it does not admit aggregate parameters.

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
