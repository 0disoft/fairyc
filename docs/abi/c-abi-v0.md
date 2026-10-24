# C ABI v0 and Target Model

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0006-c-abi-v0-and-first-conformance-target.md`

## Target status

The first planned conformance target is `x86_64-linux-gnu`. Hosted layout, bidirectional C ABI,
runtime, and generated-source probes pass for the subset through F2-6. The [hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) at `9e1500e1` covers
the expanded floating-array probe and passed all nine jobs. The previous F2-5 receipt and the
historical F1–F3 receipt are linked from the [phase status](../development/phase-status.md). The
target evidence in
`internal/target/spec.go` retains `complete-runtime` and `complete-standard-library` as missing.
The Linux `emittable`, `conformant`, and `supported` flags remain false;
maintained distribution remains a separate support requirement.

[ADR 0474](../adr/0474-linux-toolchain-baseline-and-provisioning.md) now defines the
[reference toolchain and setup](linux-toolchain-v0.md). The implementation checks compiler/libc
version floors and the native target; the existing native CI job provisions that environment.
[Linux job 119130288187](https://github.com/0disoft/fairyc/actions/runs/39571134661/job/119130288187)
at `e2aba615a2063202398eb5b295bfdd24d3aad4e4` passed provisioning and native fixtures, fulfilling
both toolchain evidence markers for the bounded Ubuntu 24.04 amd64 profile.

Target claims use three levels:

- `emittable`: the compiler can emit constrained C for the target profile;
- `conformant`: all declared language, layout, bidirectional C ABI, runtime, and toolchain fixtures pass without skips;
- `supported`: conformance plus maintained CI, distribution, diagnostics, and documented toolchain provisioning.

The Windows Go compiler host does not define Linux target layout. Proving the first target requires native Linux CI or a dedicated Linux runner, an explicit glibc baseline, a sysroot or runner image, Clang and GCC, and a target runtime bundle. WSL is not a project prerequisite.

The current Go prototype can emit deterministic C11 for a verified scalar, inert-struct,
flat-resource, and resource-tagged profile carrying this target identity. The inert-struct slice
includes nonempty source `@repr(C)` records whose fields may use `i8`, `i16`, `i32`, `i64`, `isize`,
`u8`, `u16`, `u32`, `u64`, `usize`, `f32`, `f64`, all
sixteen exact TargetSpec C scalars, admitted raw pointers, bounded integer, floating and C-scalar arrays, and
acyclic nested `@repr(C)` records. It preserves the public C tag and field names, emits dependencies
before their users, and emits target-
computed size, alignment, and offset assertions. Private `Option<i32>`, `Result<i32, i32>`,
all-`i32` resources, and resource-bearing tags remain compiler-private; resources are still
forbidden from `@repr(C)` and the public C ABI.
ADR 0137 keeps that ISO C11 spelling as the transport baseline and separates any future newer-mode
consumer validation from target ABI, runtime capability, and product emission identity.
A separate Linux/amd64 runner requires glibc, Clang, and GCC, links the embedded
scalar runtime, compares process status at low and optimized profiles, probes valid target layouts,
and runs each production unit under a closed ASan+UBSan mode. ADR 0214 records its first hosted
success, including bidirectional Clang/GCC ABI probes. The runner does not provide a public C ABI,
the complete product runtime, or the complete target profile. That narrow
capability does not advance the target itself to `emittable`.

Verbatim C names are admitted only after namespace-aware validation. File-scope `@repr(C)` tags and
external symbols reject every leading underscore; members reject double underscore and underscore
followed by uppercase while retaining `_lowercase`. All three surfaces reject C11 keywords,
generated-header macro names, the backend `FY_` macro prefix, and conflicting generated names.
Public record tags additionally reject the exact private tags `fy_region`, `fy_byte_slice`,
`fy_mut_byte_slice`, and `fy_str`. The closed target-specific `fy_std_` symbol allowlist and existing
public `fy_std_*` record tags remain admitted; neither prefix is opened generally.

## No native Fairy ABI

Stage 0 does not stabilize Fairy function mangling, ordinary Fairy calling conventions, `Option` or `Result` layout, resource layout, slice layout, drop symbols, or dynamic-library exports. Calls within one compilation are compiler-private.

Private inert structs may contain acyclic nested inert structs, including
`@repr(C)` structs, using target-computed size, alignment and offset assertions
and dependency-ordered C definitions. This does not make the containing private
type a C ABI type. A `@repr(C)` parent still requires every nested struct to have
`@repr(C)`; resource fields remain excluded from this inert-struct path.

The external ABI consists only of:

```text
unsafe extern C fn
@repr(C)
@link_name
```

`pub` is not an ABI marker.

The current executable source subset implements `unsafe extern C fn` imports with an optional
single nonempty `@link_name("symbol")`, using
`i32`, `u8`, `u32`, `usize`, all sixteen distinct transport-only TargetSpec C scalars, raw pointers to
the previously admitted Fairy scalar set (`u8`, `i32`, `i64`, `u32`, `usize`) or opaque `void`, and raw pointers to an
admitted nonempty `@repr(C)` struct. One pointer-to-opaque-pointer output-slot shape is additionally
admitted for exact `void **` calls; it does not open arbitrary pointer nesting. AST, typed HIR, MIR, unsafe provenance, deterministic C
source/link identity, prototype emission, typed null, separate-object cross-compiler linking, and process-status
observation are wired. Source struct construction and field reads reach verified MIR, the MIR
interpreter for its executable scalar subset, and generated C. Inside an explicit unsafe boundary,
an exact `ref T -> ptr T`, `ref mut T -> ptr T`, or `ref mut T -> ptr mut T` cast may take the address of a direct
local place without inventing alias promises. Raw dereference, pointer arithmetic, integer-pointer
conversion, mutability strengthening, and other raw-pointer casts remain unsupported. ADR 0286
adds only explicit `i32 <-> c::int` conversion when the selected target declares signed 32-bit
`c.int`; it does not add implicit promotion or broaden any other C-scalar conversion. The exact source-to-C
mapping is closed:

| Fairy C | TargetSpec key | C spelling | Provider |
| --- | --- | --- | --- |
| `c::bool` | `c.bool` | `_Bool` | C11 keyword |
| `c::char` | `c.char` | `char` | C11 keyword; signed on the selected target |
| `c::double` | `c.double` | `double` | C11 keyword |
| `c::float` | `c.float` | `float` | C11 keyword |
| `c::int` | `c.int` | `int` | C11 keyword |
| `c::long` | `c.long` | `long` | C11 keyword |
| `c::longlong` | `c.longlong` | `long long` | C11 keyword |
| `c::schar` | `c.schar` | `signed char` | C11 keyword |
| `c::short` | `c.short` | `short` | C11 keyword |
| `c::size_t` | `c.size_t` | `size_t` | `<stddef.h>`, ISO C11 |
| `c::ssize_t` | `c.ssize_t` | `ssize_t` | `<sys/types.h>`, POSIX.1-2008 |
| `c::uchar` | `c.uchar` | `unsigned char` | C11 keyword |
| `c::uint` | `c.uint` | `unsigned int` | C11 keyword |
| `c::ulong` | `c.ulong` | `unsigned long` | C11 keyword |
| `c::ulonglong` | `c.ulonglong` | `unsigned long long` | C11 keyword |
| `c::ushort` | `c.ushort` | `unsigned short` | C11 keyword |

Generated C asserts target-derived size and alignment for every used C scalar and signedness for
signed and unsigned integer families. Target-specific `c::ssize_t` also defines
`_POSIX_C_SOURCE 200809L`. Every C scalar supports foreign return to typed local to matching foreign
argument. Exact `c::int` additionally supports explicit bidirectional conversion with `i32` and
pure interpreter execution without a foreign call. C scalars otherwise have no source literal,
arithmetic, comparison, cast, interpreter behavior, or raw-pointer support. Exact C-scalar fields may appear directly or through admitted nested
`@repr(C)` records. Raw pointers to C scalars remain outside the admitted backend slice. This is not
implementation of public exports. Bounded callbacks have the separate admission below.
Integer-array, floating-array and explicit C enum admissions are described below. By-value record
arguments are admitted only for the existing record shapes described below. The interpreter remains
fail-closed on every foreign call.

## C-representable types

Fieldless structs without `@repr(C)` use a private one-byte C11 transport with an
initialized dummy member. The member is not a Fairy field; constructors, value
copies, arguments, returns, nesting and plain-struct field replacement retain the nominal type. This representation
does not admit empty `@repr(C)` records or change the target layout engine's empty
aggregate rejection.

`@repr(C)` permits fixed-width integers, `c.*` scalars, `f32`, `f64`, raw pointers, fixed arrays, nested `@repr(C)` structs, and an explicitly based C enum. It rejects resources, safe borrows, slices, `str`, `CStr`, `Option`, `Result`, `Region`, `String`, `Buffer`, payload enums, empty structs, packed or custom alignment, unions, bit fields, long double, atomics, and vector types.

The implemented source/backend slice is deliberately smaller than that language contract: it
accepts only acyclic nonempty structs over `i8`, `i16`, `i32`, `i64`, `isize`, `u8`, `u16`, `u32`,
`u64`, `usize`, `f32`, `f64`, all sixteen exact TargetSpec
C scalars, admitted raw pointers including opaque `void`, bounded integer, floating and C-scalar arrays, and
nested structs that are themselves `@repr(C)`. C-scalar fields use the same exact transport table and provider
assertions as foreign scalar signatures. One flat record crosses the real SystemHeap C unit; one nested record crosses
the real Buffer C unit; exact self-linked node, Arena, and Region records cross the real Arena C
unit. The Arena crossing requires that the selected provider's `uint64_t` is exactly
`unsigned long` before using `c::ulong` for the generation fields. Other contracted field families
remain fail-closed until their source typing, layout,
ABI, and native crossing tests are connected together.

[ADR 0477](../adr/0477-repr-c-integer-array-fields.md) admits array fields with `u8`, `i32`, `i64`,
`u32` or `usize` elements and lengths `1..=4096`. [ADR 0482](../adr/0482-repr-c-integer-array-transport.md)
adds `i8`, `i16`, `u16`, `u64` and `isize` elements, using `int8_t`, `int16_t`, `uint16_t`,
`uint64_t` and `intptr_t` C element types. Together these are the ten Fairy integer families; each
uses actual C array members with target-derived layout assertions.

ADR 0477 arrays retain construction, whole-field reads, whole-field and nested replacement, and
record argument/result copy value semantics. Their raw-field references, views, element stores and
iteration remain closed; read indexing copies the field first. Bare foreign array parameters/results
remain closed for both admissions.

For the ADR 0482 families, C-produced values support whole-field reads, inferred and typed array
locals, internal by-value array arguments/results, record copies, explicit reconstruction and direct
whole-field replacement through a mutable record or record reference. Nested record reads and copies
are allowed. The `IntegerTransportArray` source helper distinguishes these fields from ordinary
executable arrays. Element literals, indexing, stores, views and iteration, scalar operations and
casts, private structs with these direct arrays, bare foreign array parameters/results, new scalar
pointers and direct arrays in resource, enum or tagged payloads remain outside executable admission.
New nested field-store paths are also excluded. ADR 0480's scalar restrictions remain unchanged.

[ADR 0483](../adr/0483-repr-c-floating-array-transport.md) extends those whole-field rules to
`f32` and `f64` arrays of lengths `1..=4096`, using actual `float[]` and `double[]` C members. The
`TransportOnlyArray` helper covers these fields together with the existing `IntegerTransportArray`
families. Inferred and typed locals, internal value arguments/results, nested record reads and
copies, reconstruction and direct referenced replacement are admitted. Floating literals,
arithmetic, comparisons, casts, indexing, element stores, iteration and views; bare foreign arrays;
pointer elements or pointer-based array transport; direct private arrays;
resource, enum, `Option` or `Result` payload arrays; and record-element arrays remain closed.

[ADR 0484](../adr/0484-repr-c-scalar-array-transport.md) extends the same bounded whole-array
transport to all sixteen exact `c::*` identities when the selected target supplies them.
`c::long` arrays follow target LP64/LLP64 layout; `c::ssize_t` remains POSIX-only. C `_Bool` arrays
do not admit Fairy `bool` arrays. C/Fairy arrays remain distinct even when their representations
match, and scalar conversions do not convert arrays. Existing wrappers and element copies carry
the values without additional runtime code or cache formats. Local source/MIR/interface checks,
both target emission profiles and all four MSVC C11/C17 low/optimized native crossings pass.
Native Linux execution for this addition remains pending.

The shared native record probe checks ADR 0477 array layout, all elements and copy isolation. Local
MSVC execution is recorded in the [F2-1 update](../development/phase2-target-scope.md#f2-1-integer-array-fields);
hosted Linux evidence passed in the [F1-F3 receipt](../development/phase2-target-scope.md#hosted-f1f3-receipt-2026-10-23).

The ADR 0482 implementation and native proof at `af181d65` use
[`integer_arrays.go`](../../internal/foreignstructprobe/integer_arrays.go) to check independent C
observation of every element, array value arguments/results, borrowed nested reads and whole-field
replacement. Focused compiler and backend checks pass, and MSVC 14.51.36231 passes all four
C11/C17 low/optimized executions with expected 42 and empty output. The [previous hosted F2-5
receipt](../development/phase2-target-scope.md#hosted-f2-5-receipt-2026-10-23) at `f79e55c6` passed
all nine jobs in [Frontend run 39849506646](https://github.com/0disoft/fairyc/actions/runs/39849506646)
and remains historical for F2-6. The historical `82b760cb` F1-F3 receipt remains historical and
does not cover this expansion.

ADR 0483 source, emission and focused tests are committed at `36dc9dcc`; its expanded shared probe
is committed at `2199a87d`. Local compiler, package and shared-emission checks pass, and local
Windows MSVC passes C11/C17 low/optimized with all four executions. Those local results are
historical pre-push validation. The [hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) at `9e1500e1` covers
Linux native and hosted execution. The previous `f79e55c6` hosted F2-5 receipt is historical for
this slice.

The Windows target product smoke uses a separate private `usize` generation transport. Under the
selected x64 MSVC TargetSpec, `usize` and the C unit's `uint64_t` are both eight-byte, eight-aligned
unsigned storage. That private same-layout crossing is sufficient for the closed initialization,
begin, allocation, reset, and drop calls, but it does not alias the types in the language, stabilize
the Arena representation, or admit arbitrary cross-translation-unit aggregate transport. The
Windows FY caller observes state and pointer transitions without dereferencing allocated bytes.

The same Windows product smoke now carries private 16-byte text and CStr views, 32-byte nested
String/CString records, and an 8-byte two-`u32` owned-text result. It executes small aggregate return
transport from the separately compiled C unit for String and CString operations at both profiles.
That is one closed same-toolchain source crossing, not a stable record ABI, broad aggregate-return
conformance, or byte-content proof. ADR 0476 separately admits by-value record parameters below.

The safe owned-text example and ADR 0247's public SDK wrapper use a separate opaque storage bridge
rather than passing those records as public Fairy types. C loads and stores the private record bytes
with `memcpy`; the bridge proves the current implementation-private layout crossing but does not
assign a C effective type to generated resource storage or stabilize that storage as ABI.

The implemented foreign-result slice admits those same nonempty acyclic `@repr(C)` records by value.
Generated C emits the exact public struct return type and ordinary assignment. A generated Fairy
caller is configured to receive both an 8-byte INTEGER-class result and a 20-byte MEMORY-class
hidden result from a separately compiled C callee in both compiler directions and profiles.
[ADR 0476](../adr/0476-repr-c-struct-foreign-arguments.md) additionally admits those same record
shapes as foreign arguments. The generated prototype and call retain the exact public struct type;
the C compiler handles register and stack transport without flattening or pointer substitution.
The shared source fixture checks small and large argument/result copies, unchanged caller values,
and a nested record with a `c::double` field after six integer arguments. The existing native lanes
compile caller and callee separately. The [current hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) covers these argument
observations in both compiler directions and profiles; the previous F2-5 receipt remains historical
for this expansion, and prior struct-result receipts attest results only historically. Other
aggregate parameter families remain closed.

C enums specify their underlying `c.*` integer. The backend may transport them as a fixed integer type plus constants rather than relying on a host compiler's default enum size.

[ADR 0479](../adr/0479-repr-c-floating-field-transport.md) admits distinct Fairy `f32`/`f64`
transport in repr(C) fields, including nested repr(C) records. C-produced records supply exact
floating values for local copies, internal value arguments/results, explicit record construction
and foreign record argument/results. C storage uses `float`/`double` with target-derived
size/alignment assertions plus binary32/binary64 precision and exponent checks. The source types
remain distinct from `c::float`/`c::double`. Floating literals, arithmetic, comparisons, casts,
array element operations and views, tagged/resource payloads, direct floating-field stores, private
direct floating fields and standalone foreign floating signatures remain closed. The pure interpreter rejects
foreign calls. The shared native record probe observes finite values, signed zero, infinities,
quiet NaNs and subnormals; it does not promise signaling NaN behavior or NaN payload identity.
The [F2-3 update](../development/phase2-target-scope.md#f2-3-floating-field-transport) records the
local checks; the [current hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) covers the hosted
compiler/profile combinations.

[ADR 0480](../adr/0480-repr-c-integer-field-transport.md) adds distinct Fairy `i8`, `i16`, `u16`,
`u64` and `isize` field transport using `int8_t`, `int16_t`, `uint16_t`, `uint64_t` and `intptr_t`.
C-produced values support typed locals, internal value calls, record copies, explicit reconstruction
and foreign record arguments/results, including nested repr(C) records. Generated assertions check
the target size, alignment and signedness. `isize` follows target pointer width rather than C `long`.
Literals, operators, comparisons, casts, arrays/views, direct scalar-field stores, direct private
fields, scalar resource/tagged/enum payloads, bare foreign signatures and pointers to these new
scalar families remain closed. These restrictions do not remove existing nested-record operations.
The shared probe observes signed minima/maxima, unsigned maxima, zero and ordinary values and
checks caller-copy isolation. See the [F2-4 update](../development/phase2-target-scope.md#f2-4-remaining-integer-field-transport)
for the local checks; the [current hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) covers their hosted
Linux execution.

[ADR 0478](../adr/0478-explicit-c-enum-fields.md) implements payload-free enums with an explicit
`@repr(c::integer)` base. Variants use zero-based declaration ordinals within the base's admitted
range. The enum retains its nominal type in Fairy; its C storage is a typedef of the exact selected
integer scalar with target size/alignment and provider assertions. Such enums may be fields of
repr(C) records, including nested records, and cross foreign calls inside record arguments/results.
Direct enum-field replacement preserves exact nominal typing and copy isolation. A switch traps on
an undeclared foreign ordinal rather than assuming it is valid. Plain private enums, payloads,
explicit discriminant values, enum-field borrows, nested enum leaf replacement and
standalone foreign enum arguments/results remain outside this admission. Target availability still
applies, including the absence of `c::ssize_t` on Windows. The
[F2-2 update](../development/phase2-target-scope.md#f2-2-explicit-c-enum-fields) records local native
execution; the [current hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) covers hosted Linux.

[ADR 0485](../adr/0485-repr-c-enum-array-transport.md) admits bounded `1..=4096` element arrays of
these explicit enums as repr(C) fields. Whole-value transport preserves nominal identity and exact
integer representation, including undeclared foreign ordinals. Typed/inferred locals, internal
array arguments/results, reconstruction, nested record reads/copies and direct whole-field
replacement follow the existing transport-only array rules. Array literals, indexing, length,
views/iteration, field borrowing, nested field stores, bare foreign arrays and direct private/
resource/tagged payload arrays remain closed. Local source/MIR/interface and both target-emission
checks pass; MSVC C11/C17 low/optimized native execution passes. Native Linux proof is pending.

By-value `@repr(C)` struct arguments and returns are included. The target layout engine must agree with the C compiler's actual ABI classification.

## Functions and callbacks

C variadics are rejected and require a fixed-signature shim.

The only Stage 0 callback form is non-capturing, synchronous, same-thread, call-bounded `unsafe extern C fn` with explicit user data. Nullable, retained, asynchronous, cross-thread, and exception- or `longjmp`-crossing callbacks remain unsafe raw integrations.

[ADR 0481](../adr/0481-bounded-c-callbacks.md) implements the first bounded source signature:

```fairy
unsafe extern C fn i32 visit(
    i32 value, unsafe extern C fn i32(i32, ptr mut void) callback, ptr mut void data);
unsafe fn i32 handle(i32 value, ptr mut void data) { return value + 1; }
fn i32 main() { return unsafe { visit(41, handle, null) }; }
```

The callback is the penultimate import parameter, followed by final `ptr mut void` user data.
The argument must name a non-capturing unsafe Fairy function with the exact `i32(i32, ptr mut void)`
signature and no declared effects. An internal C adapter supplies the function-pointer ABI;
it does not publish the Fairy function as a C symbol. The generated pointer size and alignment
are checked against TargetSpec. Null user data is allowed if the handler does not dereference it;
the callback itself cannot be null. Other signatures, callback locals/fields/arrays/results,
casts and indirect calls remain closed.

The unsafe import promises same-thread synchronous invocation before returning and no retention
of the callback or call-bounded data. C must not unwind or `longjmp` across Fairy frames. The
compiler verifies the source signature, direct argument placement and nonescaping MIR temporary;
it cannot prove a C implementation honors those promises. Raw user-data validity remains the
caller's responsibility, with existing borrow-to-pointer checks retained. The pure interpreter
rejects this integration. The [F3 update](../development/phase2-target-scope.md#f3-bounded-c-callbacks)
records the local checks; the [current hosted F2-6
receipt](../development/phase2-target-scope.md#hosted-f2-6-receipt-2026-10-23) covers hosted execution.

Fairy source TLS, C TLS symbol import or export, C++ ABI, and foreign unwinding are excluded.

## Startup, linking, and debugging

The C compiler driver owns compilation, CRT startup objects, default libraries, and linking. The generated program uses a small C entry shim that calls Fairy source `fn i32 main()`. Stage 0 does not directly invoke `ld`, `lld`, or a native object writer.

The native differential lane's `drop-trace-v0` observer and four-byte stderr records are
conformance instrumentation, not a Fairy ABI, C ABI, runtime ABI, source-visible hook, or product
artifact contract. Product C emission and product builds do not declare or link the observer.

Debugging is delegated to generated-C `#line` directives and the C toolchain's DWARF output. Accurate Fairy local-variable and ownership-state debugging is not promised.

## `TargetSpec`

Semantic target data includes target identity, architecture, OS and environment, endianness,
pointer and function-pointer layouts, scalar layouts, `usize` and `isize`, C scalar layouts,
signedness, optional typedef header/availability providers, natural alignment, aggregate layout and
C ABI classifier identities, stack and return rules, object format, symbol rules, unwind model,
debug contract level, and supported `@repr(C)` features. Language scalars cannot name C providers;
provider-qualified C typedefs must name both a header and a closed availability profile.

It contains no local executable or SDK paths.

`internal/target` now implements host-independent descriptions under `fairyc.target-spec/v0`.
For `x86_64-linux-gnu` it fixes little-endian LP64, eight-byte pointers and
function pointers, the Stage 0 language and permitted C scalar size/alignment table, 16-byte SysV
call alignment, the 128-byte red zone, zero shadow space, SysV AMD64 aggregate and return-classifier identities,
ELF64 symbols, DWARF CFI unwind, generated-C DWARF debugging, and the permitted `@repr(C)` feature
vocabulary. The scalar tables are ordered by canonical type name and every alignment is validated
as a nonzero power of two.

For `x86_64-windows-msvc` it fixes little-endian LLP64, eight-byte pointers and function pointers,
four-byte C `long`, eight-byte `size_t`, no POSIX `ssize_t`, 16-byte call alignment, zero red-zone
bytes, 32 shadow-space bytes, Microsoft x64 aggregate and return identities, COFF objects and
symbols, `.pdata`/`.xdata` unwind metadata, and generated-C CodeView/PDB debugging. Its target
profile assumes MSVC's default signed plain `char` and therefore excludes `/J`.

The shared layout engine now executes a separate nine-case Windows repository manifest. It proves
LLP64-sensitive leaf layouts, rejects POSIX `ssize_t`, and computes one exact 32-byte mixed record
with eight-byte aggregate alignment and explicit internal/trailing padding. Its eight pass cases
now generate a private C11 unit with `sizeof`, `_Alignof`, field-size, and `offsetof` assertions.
The configured Windows host smoke compiles and executes that unit with local x64 MSVC at low and
optimized profiles. This is direct local layout evidence for those eight cases, not a hosted
receipt, complete scalar table, call/return, object, or unwind observation.

These entries define target facts. Every listed C scalar now has exact source transport, but the
entries do not claim C-scalar value semantics, aggregate forms, or additional call classes in the
current semantic frontend or C backend.

`internal/layout` now executes those facts for scalars, pointers, explicitly based C enums,
positive fixed arrays, and nonempty nested structs. Its versioned v0 manifest records exact size,
natural alignment, declaration-order field offsets, top-level internal padding, tail padding, and
closed rejection codes across 16 cases. This proves the internal target-layout algorithm against
repository expectations only. The eleven valid cases now also generate a private recursive C11
plus POSIX typedef probe
with `sizeof`, `_Alignof`, field `sizeof`, and `offsetof` assertions configured for four
Clang/GCC low/optimized executions. Those executions are not yet observed on a hosted Linux runner;
call, return, object, and bidirectional ABI probes remain required native evidence.

`internal/cabi` now executes the closed Stage 0 SysV AMD64 classification model over those same
target-laid-out types. It assigns `INTEGER`, `SSE`, or `MEMORY` eightbyte classes; maps fixed
arguments to the six integer and eight SSE argument registers; maps returns to the two integer and
two SSE return registers; rolls an aggregate wholly to the stack when either register family is
insufficient; and reserves `rdi`/returns `rax` for a hidden memory-result pointer. The versioned v0
C ABI manifest records 17 exact cases, of which 14 pass cases feed the native probe. The Linux
    runner now generates separate caller and callee objects for those pass cases,
crosses Clang-to-GCC and GCC-to-Clang at low and optimized profiles, and plans 56 zero-output
executions. That configured cross-compiler path is not an observed hosted receipt or Fairy source
aggregate FFI lowering, does not expose exact stack-area offsets, and does not clear the
`bidirectional-c-abi-fixtures` target marker.

For `x86_64-windows-msvc`, the same engine selects a separate Microsoft x64 algorithm. Four
declaration-order parameter slots choose RCX/RDX/R8/R9 for integer, pointer, small-record, or
by-reference-pointer transport and XMM0-3 for scalar floating transport. Later parameters use the
stack. Naturally laid out records of exactly 1, 2, 4, or 8 bytes travel as integer values;
odd-sized and larger records travel indirectly. A memory result uses a hidden pointer in RCX,
returns that pointer in RAX, and moves the first explicit parameter to slot one. The separate
Windows v0 manifest executes 15 exact cases, of which 12 are pass plans. These facts clear only the
missing classifier capability. The twelve pass plans additionally generate separate caller and
callee C11 translation units; local x64 MSVC compiles, links, and executes the pair at low and
optimized profiles. That proves value transport across a same-version MSVC boundary for this
closed subset only. It does not prove source-level Fairy lowering, an independent compiler or
register-level oracle, exact stack/shadow-store placement, complete PE/COFF object or unwind
metadata, or hosted evidence.

A separate shared generated-Fairy probe does cover one source-level result boundary. The Windows
backend emits its `@repr(C)` 8-byte two-`u32` and 20-byte five-`u32` foreign results, then local x64
MSVC compiles that caller and a hand-written C callee as separate objects at low and optimized
profiles. The identical C11 units repeat under explicit C11 and C17 consumer identities. All four
executions return 42, and every object, linked image, and unwind structure passes the bounded
inspectors. This proves the same-MSVC source-to-C return path and focused C17 consumer compatibility
for those two shapes only; it
does not provide an independent compiler or register oracle, aggregate parameters, callbacks,
exports, nested results, or hosted evidence.

The Windows smoke additionally parses all fifty-nine linked target or probe executable images without
invoking the host loader as the parser. The target-neutral product, both declared-target profile
products, thirteen package-workspace build products including safe `std::io`, `std::bytes`, and
`std::text` products, nine package-workspace run products, both safe FY
File lifecycle products, both safe FY owned-text lifecycle products, two direct run-target temporary
products, two exec-target temporary output products, both profiles of the raw Windows File, raw Buffer, raw owned-text, raw Arena, layout, ABI, REL32-link, generated-Fairy
struct-result, and unwind probes, plus four archive/COMDAT input-order variants must be
bounded AMD64 PE32+ large-address-aware non-DLL console images with aligned headers and sections, a
readable executable `.text` section, and an entry point inside executable code. Import payloads are
not interpreted. The exception directory, `.pdata`, and referenced unwind records separately cross
the bounded Microsoft x64 structural inspector; language-specific handler data, recursive chain
equivalence, and chained-tail relocations remain unproven. ADR 0240 proves one profile-specific
handler-tail relocation. This is executable-image and
bounded unwind-structure evidence only, not complete COFF symbol/relocation, CodeView/PDB, or
distribution conformance.

Before returning any linked Windows artifact, the shared C driver now parses every intermediate
MSVC `.obj` as raw AMD64 COFF. It rejects optional PE headers and linked-image flags; requires
bounded raw and relocation ranges, readable executable `.text` or `.text$...` code, valid primary
relocation symbols and AMD64 types, exact COMDAT section definitions, and named symbols; and permits storage-free uninitialized `.bss` only
when its section characteristic says so. This covers every `build-host`, `build-target`, `run-host`,
Windows-File-probe, layout-probe, ABI-probe, REL32-link probe, archive-selection probe, generated-Fairy struct-result probe, and unwind-probe
compilation in the local smoke. ADR 0231 now freezes exact internal primary-symbol, REL32, ANY
COMDAT, and associative-COMDAT receipts over two raw AMD64 object vectors, while the same inspector
validates all local MSVC objects. ADR 0232 additionally reads one direct call's object addend and
preferred-image source and target addresses, recomputes `S + A - (P + 4)`, and compares the exact
linked displacement at low and optimized profiles before execution.
ADR 0234 preserves exact relocation section indexes and reuses those two executions to bind the
receipt target's `ADDR64` object addend to the MSVC map address and final pointer, then binds all
three internal-label and `.xdata`-symbol `.pdata` `ADDR32NB` addends to independently mapped RVAs
and the validated final runtime-function entry.
ADR 0233 now parses both Microsoft archive linker members and every contained object, requires the
preferred index to name the demanded and unused members, reads both candidate object initializers,
requires the unused sentinel to live in a non-COMDAT executable section and remain absent from the
final map, and binds the exact MSVC map owner for the demanded archive function and one pick-any
COMDAT to their final PE addresses and candidate value across four input-order executions. It
also carries ADR 0235's selected function root through its direct associative `.pdata` and `.xdata`
children, all three raw `.pdata` addends, exact archive-member map owners, and unique final
runtime-function entry in those same executions. ADR 0236 adds two contributed ANY-COMDAT function
parents with distinct unwind children, binds the selected owner, and requires the losing payload to
be absent from every runtime-function reference and file-backed final section. ADR 0237 adds one
exact generated object and binds `REL32` through `REL32_5` to their distinct
`S + A - (P + 4 + n)` final-image formulas at both profiles. ADR 0241 adds exact storage-class-105
weak aliases and proves both an object-local fallback and a `target.obj` strong override through
their final REL32 values and native results. ADR 0242 proves the COFF extended-relocation sentinel,
65,535 normalized semantic records, exact map ownership, final-record REL32 displacement, and
native result at both profiles. ADR 0238 adds exact `/Z7`
function-targeted CodeView `SECREL`/`SECTION` pairs and binds each final PE RSDS identity to its
ephemeral MSF 7 PDB. ADR 0239 independently resolves that exact source function and its `caller.c`
line through system DbgHelp at both profiles. ADR 0240 additionally binds the native unwind frame's
low `__GSHandlerCheck_SEH` and optimized `__C_specific_handler` tail relocations through exact map
rows to executable final RVAs.
ADR 0243 additionally proves one bounded version-2 bigobj header, 20-byte static-symbol REL32,
exact map owner, final displacement, and native result at both profiles. ADR 0245 proves the two
remaining weak search modes by suppressing one available archive member and extracting another.
It still does not define exported/imported product symbols, remaining relocation formulas,
high-section or bigobj auxiliary/COMDAT layouts, chained-tail relocations,
non-unwind or broader archive retention and COMDAT policies, complete `.pdata`/`.xdata`
content, complete CodeView/PDB record semantics or broader symbolication, or a fresh hosted same-commit object receipt, so
`pe-coff-object-fixtures` remains missing.

The executable-image path now follows the Microsoft x64 exception directory as well. It requires a
readable non-executable `.pdata` table of sorted nonoverlapping 12-byte `RUNTIME_FUNCTION` entries,
maps every function range to executable code, and maps every DWORD-aligned unwind RVA to readable
file-backed data. Version 1 prolog operations are checked for slot counts and descending code
offsets. Local MSVC 14.51 also emits version 2 records, so bounded short/long epilog descriptors and
spare slots are admitted structurally and reported separately. Minimum exception-handler and chained
tails are checked.

The native unwind probe adds a separate semantic layer above that parser. Local x64 MSVC compiles
the catcher, stack-owning termination frame, and exception raiser separately at low and optimized
profiles. `RaiseException` carries two checked application values into an outer `__except` filter;
the filter walks the captured `CONTEXT` with `RtlLookupFunctionEntry` and `RtlVirtualUnwind` and
requires the raiser, termination frame, and catcher in order. Selecting the handler must then run
the intervening `__finally` exactly once with `AbnormalTermination()` true. The probe is closed C
target evidence and does not add source exceptions or unwind successors to Fairy C. ADR 0240 binds
the frame's exact handler-tail `ADDR32NB` relocation to its final executable target at both
profiles. Mixed-stack walks and C-boundary exception containment now have
[bounded generated-Fairy evidence](../ops/evidence/windows-mixed-stack-20261023.json): separate
Fairy/C objects pass four local low/optimized executions, including callback trap termination
without unwinding. The later [live-handler record](../ops/evidence/windows-mixed-handler-20261024.json)
also checks the suspended C outer frame's exact handler/data addresses and independently saved
caller PC/SP, with six rejected expectation mutations across both profiles. Foreign exceptions
still must not cross Fairy frames. The [v2 epilog record](../ops/evidence/windows-v2-epilog-20261024.json)
checks bounded end-relative pop/ret descriptors against Windows virtual unwinding at actual linked
instruction boundaries with controlled stack/register contexts. The
[chained-unwind record](../ops/evidence/windows-chained-unwind-20261024.json) adds
metadata-derived controlled contexts for linked CRT fragments with nonempty prologs and no frame
register: 16 fragments and 32 contexts per profile include two-link restoration, exact caller PC/SP,
establisher frame, integer restoration and XMM preservation. The later
[zero-prolog record](../ops/evidence/windows-chained-zero-prolog-20261024.json) checks nine
add/pop/ret epilogs and 36 instruction-derived contexts per profile, including registers already
restored before the selected PC. It counts unadmitted CALL/INT3 and other instruction sequences
separately. General frame-register forms and XMM save operations remain untested. The final-image inspector follows
each chain to a valid primary record, rejecting cycles, excessive inspector depth, mismatched frame
fields and secondary stack changes. Live CRT capture, other epilog
instruction shapes, language-specific handler payload parsing, complete chain equivalence and chained-tail
relocation, and complete hosted fixtures remain unproven; `windows-x64-unwind-fixtures` therefore
remains missing. No runtime feature or dependency is added.

This executable description is target data, not conformance evidence. Its evidence block names the
existing `scalar-and-flat-copy-resource-tagged-c11` subset, keeps `emittable`, `conformant`, and `supported` false, and
retains the missing complete-runtime and complete-standard-library gates. ADR 0214 records the first
hosted Linux receipt; the current receipt above fulfills the later baseline/provisioning gates.
`fairyc experimental target-spec x86_64-linux-gnu` exposes that
same immutable data without probing the host.

The Windows evidence block reports `scalar-and-llp64-c-scalar-c11` while keeping all maturity
booleans false. `Stage0WindowsMSVCProfile` binds generated C to the Windows TargetSpec, emits
four-byte `c::long` assertions, rejects unavailable `c::ssize_t`, and is executed through the
checked-in `build-target` example. That example crosses SystemHeap allocate/deallocate with a
24-byte pointer/`usize`/`usize` record, crosses Buffer initialize/zeroed-resize/drop with a nested
32-byte record, crosses private Arena/Region records, crosses String/CString/CStr lifecycle with
private views and 8-byte result returns, then calls `fy_std_text_utf8_valid_v0` using a null `ptr u8`
and zero `usize`. Buffer, Arena, and owned text close over their declared dependencies; the Windows
support resolver links only SystemHeap, Buffer, Arena, text
view, owned text, and scalar runtime, while every other foreign symbol fails closed. The SystemHeap C source receives a target-only
16-byte maximum-fundamental-alignment definition matching the Windows x64 `malloc` contract. It
lists the still-incomplete LLP64 layout-probe evidence, complete bidirectional source-level and
independently inspected Microsoft x64 crossings,
PE/COFF object and unwind fixtures, complete runtime and standard library, hosted
Windows receipt, toolchain baseline, and provisioning gates. The local
eight-case layout probe, twelve-case same-MSVC C crossing, and three-case native unwind probe are
joined by one same-MSVC generated-Fairy small/large struct-result crossing under C11 and C17
consumer modes and remain deliberately
insufficient to clear those markers.

`experimental run-target` consumes the same Windows target unit and closed support closure, then
validates and executes a temporary PE. Its exact signed `i32` result and product-trap mapping are
process/CLI observations rather than a native Fairy calling-convention or stable ABI promise. The
command adds no export, callback, dynamic-library, object, or distribution surface and does not
change the false maturity flags.

## `ToolchainSpec`

Toolchain data includes C dialect, compiler and linker driver identities, version fingerprint, sysroot, libc family and baseline, startup objects, default libraries, dynamic loader, sanitizer capabilities, required flags, and local capability resolution. Local installation paths are resolution results, not target semantics.

The current constrained-C driver now binds each resolved Clang or GCC record to a known target ID
before host or libc probing. It still has no frozen glibc or compiler-version baseline, so those
requirements remain missing evidence rather than invented values in `TargetSpec`.

## Conformance fixtures

Layout fixtures cover size, alignment, offsets, padding, nested structs, fixed arrays, explicit C enums, pointer fields, and rejected types. They include small register-sized aggregates and larger aggregates that exercise memory and hidden-return paths.

Native source-level call fixtures are eventually bidirectional: Fairy-to-C and C-to-Fairy. The host-independent v0
classification fixtures already cover integer and floating register exhaustion, mixed arguments,
pointers, by-value structs, and hidden aggregate returns. Native crossings must additionally cover
narrow signed and unsigned values, nested aggregates, null raw pointers, and the permitted callback
form before the target can claim that source-level gate. The current shared generated-Fairy
struct-result probe covers one register and one hidden return in both Linux compiler directions and
in both local MSVC profiles under C11 and C17 consumer modes; it does not yet cover aggregate parameters, mixed INTEGER/SSE source
records, nested-result interaction, or a C-to-Fairy export direction.

The planned transport matrix uses Clang as the primary compiler and GCC as an independent compiler, each at low and optimized profiles. Sanitizers supplement but do not define conformance.

## Later profiles

`x86_64-linux-musl` needs a separate libc, startup, sysroot, dynamic-loader, and static-distribution profile even though its CPU ABI is SysV-derived. The declared `x86_64-windows-msvc` TargetSpec still needs complete executable LLP64 layout coverage plus hosted evidence, independent or register-inspected four-slot call evidence, broader and bidirectional source-level Fairy crossings, a fresh hosted receipt for the exact synthetic COFF fixture and strengthened live-object/archive inspectors, remaining relocation formulas and chained unwind tails, non-unwind and broader archive/COMDAT policies, complete version-2/handler-payload/chain and Fairy/foreign-frame unwind fixtures, complete UCRT/VCRuntime and deployment evidence, and debug fixtures before any maturity promotion.

## External references

- x86-64 psABI project: <https://gitlab.com/x86-psABIs/x86-64-ABI>
- Microsoft x64 calling convention: <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- Microsoft `RaiseException`: <https://learn.microsoft.com/en-us/windows/win32/api/errhandlingapi/nf-errhandlingapi-raiseexception>
- Microsoft `RtlVirtualUnwind`: <https://learn.microsoft.com/en-us/windows/win32/api/winnt/nf-winnt-rtlvirtualunwind>
- Microsoft x64 prolog and epilog: <https://learn.microsoft.com/en-us/cpp/build/prolog-and-epilog?view=msvc-170>
- Microsoft structured exception handler syntax: <https://learn.microsoft.com/en-us/windows/win32/debug/handler-syntax>
- musl reference manual: <https://musl.libc.org/doc/1.0.0/manual.txt>

These sources constrain target profiles; they do not define Fairy C source semantics.
