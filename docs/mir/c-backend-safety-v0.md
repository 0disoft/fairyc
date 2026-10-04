# C Backend Safety v0

- Status: Draft Stage 0 contract
- MIR source: `docs/mir/mir-v0.md`
- Backend decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Target and ABI source: `docs/abi/c-abi-v0.md`

## Role

Generated C is a reference and differential-test transport for a declared `TargetSpec` and `ToolchainSpec`. It is not universally portable C and it is not the Fairy C semantic model. The backend must preserve verified MIR without inheriting C expression-order, signed-overflow, shift, aliasing, layout, foreign-control-flow, compiler-family, libc, or host accidents.

ADR 0137 fixes ISO C11 as the only normative transport dialect. The Windows struct-result probe now
compiles identical C11 units under distinct C11 and C17 MSVC consumer modes as compatibility
evidence, but this is not a new emitter profile and cannot authorize newer-dialect syntax. C23
consumer validation remains disabled.

The first planned profile is `x86_64-linux-gnu` compiled and linked through declared Clang and GCC drivers. Emission alone is not conformance or support evidence.

Current Fairy source spells borrows, raw pointers, and arrays as `ref T`, `ref mut T`, `ptr T`,
`ptr mut T`, and `T[N]`. This backend document names those source forms unless it explicitly says
that a spelling is an internal HIR/MIR type string or generated C syntax.

The profile ID now comes from the validated `fairyc.target-spec/v0` object in `internal/target`.
The backend does not read Go `unsafe.Sizeof`, local C compiler probes, or SDK paths to decide target
semantics. The target object reports the existing scalar, flat-resource, and resource-tagged emission subset while leaving all three
maturity claims false.

## Current scalar, borrowed-view, flat-resource, and resource-tagged emission slice

`internal/cbackend` currently emits one deterministic C11 translation unit from verified MIR
for a deliberately narrow subset. It accepts `i32`, `bool`, `Option<i32>`, and
`Result<i32, i32>` parameters and internal results, `void` internal results, a zero-argument
`fn i32 main()` entry, calls and control flow, scalar tag construction, lowered switch and Result `?`,
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

The same internal-value subset now admits compiler-owned immutable `ref u8[]` literals. MIR retains
their exact bytes and logical length, including empty and embedded-NUL values. Generated C uses a
private `fy_byte_slice { const uint8_t *data; uintptr_t length; }`, exact deduplicated immutable
storage for nonempty values, and a one-byte immutable sentinel with length zero for empty values.
This private shape is neither a public Fairy layout nor a foreign-call ABI.

A contextual exclusive borrow of a mutable fixed `u8[N]` uses a distinct private
`fy_mut_byte_slice { uint8_t *data; uintptr_t length; }`. The descriptor takes the verified array
local's data address and canonical `N` without allocation or copying. It never originates from
immutable literal storage, and generated C does not add `restrict` or infer non-retention.

The same private view transport admits verified `view-length` and `view-empty` operations for safe
borrows of `str` and shared or mutable `u8[]`. Generated C either copies the descriptor's `length` member into a
`uintptr_t`-backed Fairy `usize` or compares it with zero and normalizes a Fairy `bool`; it does not
call a helper, scan storage, allocate, trap, or expose the descriptor layout.
The verifier fixes the owner family and result type before emission.

Verified shared or mutable `ref u8[]` indexing compares the `uintptr_t`-backed index against the descriptor's
byte length, routes failure to the canonical `FY_TRAP_BOUNDS` path, and only then emits one
`data[index]` load into a `uint8_t`-backed Fairy `u8`. Receiver and index operands are evaluated once
before the check and reused unchanged by the load. The C backend rejects missing or mismatched MIR
check, owner, index, result, operator, or trap facts instead of emitting unchecked access.

ADR 0430 extends these checked operations to shared `ref i64[]` backed by a
fixed i64 array. Its private descriptor contains `const int64_t *data` and a
`uintptr_t` element count. A checked load produces `int64_t`; checked slicing
adjusts a typed element pointer and count without allocation or array copying.
Actual receiver definitions prevent byte/i64 reinterpretation. The descriptor
may be passed or returned internally, including as an Option/Result payload;
it adds no foreign ABI, mutable i64 access or general raw-parts constructor.

Verified indexed assignment reuses the exact checked receiver and frozen `uintptr_t` index. The
complete bounds edge is emitted before any right-hand-side blocks, and only its success region emits
one typed fixed-array or mutable-view `data[index] = value`. Shared descriptors and immutable array
places never reach a store operation. Generated C adds no allocation, helper call, `restrict`,
retained pointer, or public ABI, and the verifier rejects a store not dominated by its matching
non-returning `FY_TRAP_BOUNDS` edge.

Internal generated functions may accept those exact safe borrowed views by value. Calls pass the
private descriptor unchanged, preserving data and byte length without granting ownership or
creating a foreign ABI. Function declarations, call operands, and inferred local types must agree;
foreign declarations continue to reject safe borrows before emission.

Distinct verified `str-from-raw-parts` and `cstr-from-raw-parts` operations accept one shared safe
owner borrow, one raw byte or opaque pointer, and one `usize` extent. Their unsafe provenance must
name the exact source span and a containing unsafe scope. The UTF-8 path builds the private `fy_str`
descriptor without allocation or copying and substitutes a private non-null empty sentinel when
the data pointer is null with a zero extent. The CStr path forwards the immutable pointer while the
extent operand remains already evaluated. Neither operation creates a foreign safe-borrow ABI or
derives lifetime from the raw pointer.

The foreign-call transport additionally accepts all sixteen exact TargetSpec C scalars as results
and parameters. HIR and MIR keep those types distinct from coincidentally represented Fairy
scalars; one ordered backend table maps them to exact C spellings, TargetSpec keys, features, and
optional providers. Generated C asserts target size and alignment for every used scalar and
signedness for signed and unsigned integer families. `ssize_t` selects `_POSIX_C_SOURCE 200809L`
before every header rather than silently relying on the host's default feature namespace. A foreign
producer result local may be passed to a matching foreign consumer. The exact signed 32-bit
`c::int` additionally crosses an explicit `scalar-cast` with `i32` in either direction; verified
MIR keeps the source and result identities, the pure interpreter round-trips the value, and emitted
C uses explicit `int32_t` or `int` casts. Other literals, operators, comparisons, casts,
interpreter execution, and raw pointers to C scalars remain closed. Exact C-scalar fields are
accepted only in nonempty `@repr(C)` structs and share the same transport table and assertions.
An admitted struct may contain another acyclic nonempty `@repr(C)` struct by value. The backend
orders complete definitions by dependency and reuses the child target layout when computing the
containing field; recursive value dependencies and non-`@repr(C)` nested structs fail closed.
Private inert structs may also contain admitted inert enums by value. Struct and enum layouts share
one acyclic aggregate dependency graph, and complete definitions are emitted topologically while
preserving prior root order when no dependency requires movement. Enum fields in `@repr(C)` structs,
and recursive aggregate dependencies fail closed. Private restricted inert enums may contain other
restricted inert enums by value; the same graph orders and lays out both complete definitions.
The same admitted records may be foreign results by value. MIR retains the nominal destination type
and generated C uses the exact public tag plus ordinary assignment. Foreign aggregate parameters
remain closed. The native matrix includes an 8-byte register result and a 20-byte hidden-memory
result from a separately compiled C callee.

The emitter additionally accepts direct resource values only when every program-level definition is
nonempty and every declaration-ordered field is a Fairy copy scalar or raw pointer. An empty body
needs no user call; a nonempty body must name one verifier-matched compiler-private drop function.
Resources may be locals, parameters, private function returns, and by-value call arguments. Exact
MIR field types select target-laid-out private C fields; raw pointers remain non-owning unless the
typed body explicitly acts on them. Numeric translation-unit types and fields prevent source
identifiers from becoming C symbols.

Verified field reads may retain a shared or mutable safe-resource borrow as their receiver. MIR
resolves the field against the borrow's underlying resource definition while preserving the borrow
type for code generation. The C backend therefore emits `pointer->field` for a borrowed receiver and
`value.field` for an owned receiver, and rejects a mismatched definition, receiver type, or field
type. This opens no foreign safe-borrow ABI and does not permit pointer retention or ownership
transfer.

Generic `Option` and `Result` values are accepted when each payload is a canonical Fairy copy scalar
(`bool`, `u8`, `i32`, `u32`, or `usize`), a positive bounded fixed array of those scalars, an
admitted inert struct, a restricted inert user enum, a direct admitted resource, or an admitted safe borrow. A user enum's
private tag-plus-union representation and the outer built-in tagged record are both computed by the target
layout engine and guarded by generated C size, alignment, and offset assertions. Scalar,
inert-struct, resource, and fixed-array borrows use pointer transport;
`ref str` retains its bounded data-and-length view transport.
The compact `Option<i32>` and `Result<i32, i32>` transports remain distinct. Other concrete types
receive numeric private tagged structs. `Option` has one payload field and `Result` has separate
success and error fields; inactive fields are aggregate-zero-initialized rather than represented
with an uncontracted C union. Size, alignment, and offsets come from the target layout engine.
Copy-only tags do not need drop; a generic tag needs drop only when a direct payload is a resource.
Direct copy-scalar tags and acyclic tags whose payloads are admitted inert structs or restricted
user enums may be fields of private inert structs or user enums. One combined graph places complete
payload aggregates before their tagged records and tagged records before containing aggregates.
Copy-only tags may directly contain other acyclic copy-only tags whose leaves are admitted Fairy
copy scalars, inert structs, or restricted user enums. Those tags may also be fields of private
inert structs or restricted user enums. Array-, borrow-, raw-pointer/C-scalar-, recursive, and
resource-bearing nested tags remain outside aggregate fields, and all tagged fields remain outside
`@repr(C)`.
Payload extraction must agree with the MIR move marker. Result propagation reconstructs the
enclosing function's declared Result type on the error path, including restricted inert user-enum
errors without converting the enum into a scalar or target constant.

Verified `region-enter` retains its exact Arena owner local and mutable-borrow type. The emitter
allocates one private Region record per MIR Region symbol, passes the retained Arena owner address to
the closed target runtime, obtains zeroed storage with the placed type's target `sizeof` and
`_Alignof`, initializes the value exactly once, and constructs either the owner-bound `.ok` borrow or
an `AllocError` `.err`. Every verified reset edge calls the target reset helper. Unexpected begin or
reset rejection is an internal-invariant trap; ordinary placement failure remains Result data. The
current Region C subset admits canonical scalar, supported inert-struct, and scalar fixed-array
placement.

ADR 0392 additionally admits acyclic directly nested resource fields, with dependency-first
target layouts, owner cleanup followed by reverse field cleanup, and retention of nested drop
symbols. A nested owner's private C cleanup returns its post-cleanup receiver so field destruction
observes body mutations. Production output remains uninstrumented; the closed Windows drop-trace
profile compares source-linked cleanup events with the interpreter.

The emitter rejects empty, array/C-scalar/Region or partially
initialized resources, and nonempty drops without verifier-matched glue;
unsupported or recursive nested built-in tags, unsupported built-in tag fields, raw-pointer/C-scalar tagged payloads,
recursive or resource-bearing user enums, and other tagged payload
combinations; returning a safe borrow through a foreign C function boundary;
safe-borrow operations other than exact verified local borrowing and dereference, array-view construction, owner-bound text
raw-parts construction, view metadata, borrowed resource field reads, shared field-address projections and ADR 0445 exclusive plain-struct field projections, immutable
byte indexing, and immutable byte slicing; unsafe operations outside the admitted raw-address,
owner-bound text construction, and scalar foreign-call forms; complete runtime and
standard-library lowering; malformed
destinations; and non-canonical block shapes. Empty-drop discharge produces no invented cleanup
call. Nonempty direct and admitted active tagged payloads invoke verified private glue before field
cleanup. This is private transport, not public Fairy ABI.
The runner proves every source-to-emission case named by
`spec/fixtures/c-emission/manifest-v0.json` plus repeated byte identity. No C parser, compiler,
object writer, assembler, linker, executable,
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
| Value-preserving integer cast | Verify one admitted source/result pair, then emit an explicit C cast to `int32_t`, `uint32_t`, `uintptr_t`, or the target-asserted exact `int` spelling for `c::int`. The `i32 <-> c::int` pairs are admitted only because every current target declares signed 32-bit `c.int`; every admitted source value is representable, so no trap, truncation, wrapping, or implementation-defined signed conversion occurs. |
| Typed scalar and raw-pointer comparison | Verify two matching operands and the exact `bool` result before emission. Scalar equality is limited to `bool`, `i32`, `u8`, `u32`, and `usize`; scalar ordering excludes `bool`. Matching raw pointers admit equality only, with no dereference or ordering. Generated C normalizes the result to `INT32_C(0)` or `INT32_C(1)`. |
| Scalar fixed-array construction, byte-view borrowing, length, indexing, and stores | Verify one canonical length inside the provisional `1..=4096` executable bound and the exact copy-scalar element type. Emit a private assignable struct wrapper, materialize list elements left to right, construct a `const` shared or writable mutable byte view only from the matching verified source `u8[N]` local using its generated `.data` address and canonical `N`, emit `.len` from canonical `N` rather than `sizeof` or storage, and check every read or store against that same length. Access or assignment to `.data[index]` occurs only on the success edge; store indices are materialized so a host C frontend cannot reject an always-trapping source before the checked edge runs. The borrow allocates and copies nothing; mutable slicing remains absent; the bound, wrapper, and descriptors are private implementation guards, not a public Fairy or C ABI. |
| Assert | Verify one exact `bool` operand and canonical failure edge, compare with zero in every profile, and dispatch `FY_TRAP_ASSERTION_FAILED` through the trap runtime. Never emit or include the C library `assert` macro. |

The backend cannot add C or LLVM overflow, exactness, alignment, or alias promises unless the verified MIR fact explicitly proves them.
The volatile divisor slot is private generated-C transport. It does not add source-visible volatile
semantics, and it is emitted only in functions containing checked division or remainder.

## Representation and access

- The `fairyc.target-spec/v0` description, not host `sizeof(long)` or Go layout, determines scalar,
  pointer, stack, classifier, and object-format facts.
- `internal/layout` now computes checked array and aggregate size, alignment, field offset, and
  padding from that description. The C emitter consumes those results for flat all-`i32` private
  resources, their permitted resource-bearing built-in tags, private enum-containing inert structs,
  and nonempty source `@repr(C)` structs
  over `u8`, `i32`, `u32`, `usize`, exact C scalars, admitted raw pointers, and acyclic nested
  `@repr(C)` records. It emits dependencies before their users plus `_Static_assert` checks for
  `sizeof`, `_Alignof`, and every `offsetof`; public C records preserve their declared tag and field
  names. Any emitted struct, enum, resource, or generic tagged record owns the required
  `<stddef.h>` include directly rather than depending on another feature. Acyclic directly nested
  resource fields use the same dependency-first layout checks under ADR 0392; unsupported tagged,
  array or borrowed resource fields and remaining contracted `@repr(C)` field families fail closed.
- Verbatim `@repr(C)` tags and fields plus external C symbols pass namespace-specific guards before
  emission. File-scope names reject every leading underscore; members reject `__*` and `_[A-Z]*`
  while retaining `_lowercase`. Header macros, the backend `FY_` macro prefix, generated scalar
  names, and exact private record tags fail closed. Target-admitted `fy_std_` symbols and existing
  public `fy_std_*` record tags retain their closed paths.
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
- The deterministic C unit emits only the functions reachable from the selected entry through
  internal calls and resource drop glue. The generated C `main` keeps `(void)&fy_fn_N` only for
  reachable non-entry functions so warnings-as-errors builds stay free of false unused-function
  diagnostics. Dead functions and dead foreign declarations contribute no functions, prototypes,
  macro guards, or scalar features to the emitted unit. See ADR 0258.
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
open/read/write-all/sync/close. The Windows target additionally admits the five opaque String/CString
storage bridges used by the safe owned-text example and public SDK wrapper, and separately admits File
open/read/write-all/sync/close/drop; its single-write helper retains internal C linkage. Direct
Linux File write/drop and the rest of the prefix remain closed.
Typed null, an exact or opaque direct-local borrow-to-raw-pointer cast, and opaque erasure of one
existing safe resource or fixed-array borrow are the only source-constructed raw pointers. Direct
place casts lower to verified MIR `raw-address` and emit `&local`. Existing-borrow erasure lowers to
verified `raw-borrow-pointer` and emits the referent pointer unchanged. Both require unsafe
provenance, retain the concrete owner type, permit only the exact referent or `void`, and forbid
mutability strengthening, `restrict`, and `noalias`. Pointer dereference, arithmetic, integer
conversion, and arbitrary casts remain rejected.
The two owner-bound text raw-parts operations are separate from pointer casts. They require matching
unsafe provenance plus a shared safe owner operand, and their result origin remains a semantic Flow
fact rather than a C pointer attribute.
A separate verified `byte-slice-data-address` operation may expose the immutable data pointer of a
compiler-owned source `ref u8[]` literal or typed shared local inside unsafe code. A typed source
`ref mut u8[]` local may instead expose `ptr mut u8`. The operation requires the matching owner and raw-pointer mutability,
cannot strengthen a shared view, cannot request mutable static storage, and cannot pass the safe
view itself through a foreign signature.
The corresponding CStr-data operation accepts static literal storage or a local borrowed CStr.
Static storage emits its compiler-owned address. A local source `ref CStr` is already transported as
`const uint8_t *` and is forwarded unchanged; emitting a nonexistent `.data` member is rejected by
the native C compiler and its regression test.
Successful emission also records the sorted unique external link names used by actual foreign call
terminators. Uncalled declarations remain prototypes but are absent from this link-requirement list.
The exact standard-library namespace and component dependency mapping live in one shared contract
consumed by both C-emission admission and product-build source selection.
Windows File names are rejected for Linux and select one independent inline-handle component only
for `x86_64-windows-msvc`. ADRs 0124 and 0125 established an example-level safe FY wrapper plus a
length-generic fixed-array byte view; ADR 0264 publishes that exact Windows slice as `std::fs`, and
ADR 0269 adds two target-bound absolute-position symbols, ADR 0270 adds one shared size symbol, and
ADR 0271 reuses the existing open symbol for existing-file read/write access without create or truncate,
while ADR 0272 adds one target-bound cursor-preserving length-change symbol, ADR 0274 reuses the
existing open symbol for non-truncating open-or-create, ADR 0275 reuses it with one
compiler-private exclusive flag for atomic no-clobber creation, and ADR 0276 reuses it with one
compiler-private append flag selecting `FILE_APPEND_DATA`, ADR 0277 adds one target-bound
same-handle file-kind symbol without exposing Win32 constants, and ADR 0278 adds one target-bound
exact-read symbol with completed-prefix failures. ADR 0279 adds one target-bound file-identity
symbol whose 24-byte caller output is written only after success. Portable paths, relative/end-based
seek, general metadata, Linux public linkage, general
mutable-buffer operations, retained pointers, and public layout remain outside this boundary.
The sole nested-pointer exception is exactly source `ptr mut ptr mut void`, so source
`ref mut ptr mut void` can become a C
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
`i32` returns, effect-bearing drop-body/native-runtime integration evidence, public layout, ABI, and
broader runtime behavior remain unimplemented. A clean sanitizer run is backend evidence, not the
language-semantics oracle.

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
`fairyc.native-c-report/v36` separately records four `foreignCStructResultCallExecutions`, four
`systemHeapSourceCallExecutions`, four
`bufferSourceCallExecutions`, four `arenaSourceCallExecutions`, and four
`cStrValidationSourceCallExecutions`, plus four `ownedTextSourceCallExecutions` and four
`linuxFileSourceCallExecutions`, four `linuxPublicFileExecutions` covering target-neutral current
position and absolute seek, and four
`linuxConsoleExecutions`. The console count executes the
target-neutral POSIX stdout and stderr adapter's empty, invalid, redirected-write, and unavailable-
stream states. The SystemHeap count
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
lexical Region, safe text, or stable safe File APIs. The Windows-only ADR 0124 example is a separate
source-to-native lifecycle proof.
The public File count separately compiles the bundled SDK through the ordinary product build, lets
used-symbol linkage select the target-neutral Linux File component, and executes exact create,
write-all, absolute/current/end seek, cursor-preserving logical size, sync, consuming close, reopen,
partial read, and final close behavior under both compilers and profiles in isolated temporary
working directories.
Four `standardLibraryBuildSmokeExecutions` additionally require the ordinary product CLI to select
the text-view C component from an actual called symbol, link it with the scalar runtime, and execute
the resulting uninstrumented product under both compilers and profiles.
Four `byteStringLiteralSourceCallExecutions` require a generated Fairy caller to construct a
private `ref u8[]` view over exact non-terminated `fairy` storage, preserve length five, extract the
typed local's immutable data address, and have a separate C callee verify all five bytes. Four
`byteStringLiteralBuildSmokeExecutions` separately require the ordinary product CLI to carry that
view into an executable returning 42 under both compilers and profiles. Empty literals instead use
the non-null one-byte sentinel with logical length zero.
Four `utf8StringLiteralSourceCallExecutions` require a generated Fairy caller to construct a private
`ref str` view over exact `Fairy λ` UTF-8 storage, preserve byte length eight, extract a typed local's
immutable data address, and have a separate C callee verify all eight bytes. Four
`utf8StringLiteralBuildSmokeExecutions` separately require the ordinary product CLI to carry that
view into an executable returning 42 under both compilers and profiles. Empty text uses a non-null
one-byte sentinel with logical length zero, and invalid decoded UTF-8 never reaches MIR.
Four `cStringLiteralSourceCallExecutions` require a generated Fairy caller to create exact
`fairy\0` storage, cross the dedicated unsafe immutable-byte-pointer operation, and have a separate C
callee verify all six bytes. Four `cStringLiteralBuildSmokeExecutions` separately require the
ordinary product CLI to carry nonempty compiler-owned `ref CStr` storage into an executable returning
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

ADR 0446's shared fields from mutable plain struct references use an explicit
verified shared root before ordinary shared field-address emission. The backend
still receives matching owner/result reference kinds and never copies a stored
struct to obtain a projected address. No additional alias-optimization promise,
runtime symbol or target admission is implied.
