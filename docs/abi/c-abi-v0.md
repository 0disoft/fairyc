# C ABI v0 and Target Model

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0006-c-abi-v0-and-first-conformance-target.md`

## Target status

The first planned conformance target is `x86_64-linux-gnu`. It is not currently supported because
the complete runtime, sysroot policy, public layout and ABI fixtures, maintained distribution, and
observed clean hosted evidence do not exist.

Target claims use three levels:

- `emittable`: the compiler can emit constrained C for the target profile;
- `conformant`: all declared language, layout, bidirectional C ABI, runtime, and toolchain fixtures pass without skips;
- `supported`: conformance plus maintained CI, distribution, diagnostics, and documented toolchain provisioning.

The Windows Go compiler host does not define Linux target layout. Proving the first target requires native Linux CI or a dedicated Linux runner, an explicit glibc baseline, a sysroot or runner image, Clang and GCC, and a target runtime bundle. WSL is not a project prerequisite.

The current Go prototype can emit deterministic C11 for a verified scalar, inert-struct,
flat-resource, and resource-tagged profile carrying this target identity. The inert-struct slice
includes nonempty source `@repr(C)` records whose fields may use `u8`, `i32`, `u32`, `usize`, all
sixteen exact TargetSpec C scalars, admitted raw pointers, and acyclic nested `@repr(C)` records. It
preserves the public C tag and field names, emits dependencies before their users, and emits target-
computed size, alignment, and offset assertions. Private `Option<i32>`, `Result<i32, i32>`,
all-`i32` resources, and resource-bearing tags remain compiler-private; resources are still
forbidden from `@repr(C)` and the public C ABI.
A separate Linux/amd64 runner is configured to require glibc, Clang, and GCC, link the embedded
scalar runtime, compare process status at low and optimized profiles, probe valid target layouts,
and run each production unit under a closed ASan+UBSan mode. No hosted success receipt has been
observed, and the runner does not inspect objects, exercise the public C ABI,
provide the complete product runtime, or establish the complete target profile. That narrow
capability does not advance the target itself to `emittable`.

## No native Fairy ABI

Stage 0 does not stabilize Fairy function mangling, ordinary Fairy calling conventions, `Option` or `Result` layout, resource layout, slice layout, drop symbols, or dynamic-library exports. Calls within one compilation are compiler-private.

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
the fixed-width Fairy scalar set or opaque `void`, and raw pointers to an
admitted nonempty `@repr(C)` struct. One pointer-to-opaque-pointer output-slot shape is additionally
admitted for exact `void **` calls; it does not open arbitrary pointer nesting. AST, typed HIR, MIR, unsafe provenance, deterministic C
source/link identity, prototype emission, typed null, separate-object cross-compiler linking, and process-status
observation are wired. Source struct construction and field reads reach verified MIR, the MIR
interpreter for its executable scalar subset, and generated C. Inside an explicit unsafe boundary,
an exact `&T -> *T`, `&mut T -> *T`, or `&mut T -> *mut T` cast may take the address of a direct
local place without inventing alias promises. Raw dereference, pointer arithmetic, integer-pointer
conversion, mutability strengthening, and other casts remain unsupported. The exact source-to-C
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
`_POSIX_C_SOURCE 200809L`. Their only current value path is foreign return to typed local to matching
foreign argument; they have no source literal, arithmetic, comparison, cast, interpreter behavior,
or raw-pointer support. Exact C-scalar fields may appear directly or through admitted nested
`@repr(C)` records. Raw pointers to C scalars remain outside the admitted backend slice. This is not
implementation of exports, callbacks, array or enum `@repr(C)` fields, or by-value aggregate
parameters. The interpreter remains fail-closed on every foreign call.

## C-representable types

`@repr(C)` permits fixed-width integers, `c.*` scalars, `f32`, `f64`, raw pointers, fixed arrays, nested `@repr(C)` structs, and an explicitly based C enum. It rejects resources, safe borrows, slices, `str`, `CStr`, `Option`, `Result`, `Region`, `String`, `Buffer`, payload enums, empty structs, packed or custom alignment, unions, bit fields, long double, atomics, and vector types.

The implemented source/backend slice is deliberately smaller than that language contract: it
accepts only acyclic nonempty structs over `u8`, `i32`, `u32`, `usize`, all sixteen exact TargetSpec
C scalars, admitted raw pointers including opaque `void`, and nested structs that are themselves
`@repr(C)`. C-scalar fields use the same exact transport table and provider assertions as foreign
scalar signatures. One flat record crosses the real SystemHeap C unit; one nested record crosses
the real Buffer C unit; exact self-linked node, Arena, and Region records cross the real Arena C
unit. The Arena crossing requires that the selected provider's `uint64_t` is exactly
`unsigned long` before using `c::ulong` for the generation fields. Arrays, explicit C enums,
floating Fairy scalars, and other contracted field
families remain fail-closed until their source typing, layout, ABI, and native crossing tests are
connected together.

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
conformance, byte-content proof, or permission for by-value aggregate parameters.

The implemented foreign-result slice admits those same nonempty acyclic `@repr(C)` records by value.
Generated C emits the exact public struct return type and ordinary assignment. A generated Fairy
caller is configured to receive both an 8-byte INTEGER-class result and a 20-byte MEMORY-class
hidden result from a separately compiled C callee in both compiler directions and profiles.
Aggregate parameters remain fail-closed.

C enums specify their underlying `c.*` integer. The backend may transport them as a fixed integer type plus constants rather than relying on a host compiler's default enum size.

By-value `@repr(C)` struct arguments and returns are included. The target layout engine must agree with the C compiler's actual ABI classification.

## Functions and callbacks

C variadics are rejected and require a fixed-signature shim.

The only Stage 0 callback form is non-capturing, synchronous, same-thread, call-bounded `unsafe extern C fn` with explicit user data. Nullable, retained, asynchronous, cross-thread, and exception- or `longjmp`-crossing callbacks remain unsafe raw integrations.

Fairy source TLS, C TLS symbol import or export, C++ ABI, and foreign unwinding are excluded.

## Startup, linking, and debugging

The C compiler driver owns compilation, CRT startup objects, default libraries, and linking. The generated program uses a small C entry shim that calls `fn main() -> i32`. Stage 0 does not directly invoke `ld`, `lld`, or a native object writer.

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

The Windows smoke additionally parses all nine linked executable images without invoking the host
loader as the parser. The target-neutral product, both declared-target profile products, plus both profiles of the layout, ABI, and unwind probes must be
bounded AMD64 PE32+ large-address-aware non-DLL console images with aligned headers and sections, a
readable executable `.text` section, and an entry point inside executable code. Import and exception
directory presence is recorded but their payloads are not interpreted. This is executable-image
structure evidence only, not COFF object symbol/relocation, `.pdata`/`.xdata`, CodeView/PDB, or
distribution conformance.

Before returning any linked Windows artifact, the shared C driver now parses every intermediate
MSVC `.obj` as raw AMD64 COFF. It rejects optional PE headers and linked-image flags; requires
bounded raw and relocation ranges, readable executable `.text` or `.text$...` code, valid
relocation symbol indexes, and named symbols; and permits storage-free uninitialized `.bss` only
when its section characteristic says so. This covers every `build-host`, `build-target`, `run-host`,
layout-probe, ABI-probe, and unwind-probe compilation in the local smoke. It still does not define exact exported/imported
symbols, COMDAT selection, permitted relocation types, `.pdata`/`.xdata` content, or a hosted
object-fixture receipt, so `pe-coff-object-fixtures` remains missing.

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
target evidence and does not add source exceptions or unwind successors to Fairy C. Version-2
epilog offset meaning, language-specific handler payload parsing, recursive chain equivalence,
source-level Fairy or foreign-frame crossings, object-level unwind relocations, and hosted fixtures
remain absent; `windows-x64-unwind-fixtures` therefore remains missing.

This executable description is target data, not conformance evidence. Its evidence block names the
existing `scalar-and-flat-i32-resource-tagged-c11` subset, keeps `emittable`, `conformant`, and `supported` false, and
lists the missing hosted native C layout-probe receipt, bidirectional ABI, runtime, standard-library, hosted-Linux, toolchain
baseline, and provisioning gates. `fairyc experimental target-spec x86_64-linux-gnu` exposes that
same immutable data without probing the host.

The Windows evidence block reports `scalar-and-llp64-c-scalar-c11` while keeping all maturity
booleans false. `Stage0WindowsMSVCProfile` binds generated C to the Windows TargetSpec, emits
four-byte `c::long` assertions, rejects unavailable `c::ssize_t`, and is executed through the
checked-in `build-target` example. That example crosses SystemHeap allocate/deallocate with a
24-byte pointer/`usize`/`usize` record, crosses Buffer initialize/zeroed-resize/drop with a nested
32-byte record, crosses private Arena/Region records, crosses String/CString/CStr lifecycle with
private views and 8-byte result returns, then calls `fy_std_text_utf8_valid_v0` using a null `*u8`
and zero `usize`. Buffer, Arena, and owned text close over their declared dependencies; the Windows
support resolver links only SystemHeap, Buffer, Arena, text
view, owned text, and scalar runtime, while every other foreign symbol fails closed. The SystemHeap C source receives a target-only
16-byte maximum-fundamental-alignment definition matching the Windows x64 `malloc` contract. It
lists the still-incomplete LLP64 layout-probe evidence, complete bidirectional source-level and
independently inspected Microsoft x64 crossings,
PE/COFF object and unwind fixtures, complete runtime and standard library, hosted
Windows receipt, toolchain baseline, and provisioning gates. The local
eight-case layout probe, twelve-case same-MSVC C crossing, and three-case native unwind probe are
deliberately insufficient to clear those markers.

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
form before the target can claim that source-level gate. The current generated-Fairy struct-result
probe covers one register and one hidden return in both compiler directions; it does not yet cover
aggregate parameters, mixed INTEGER/SSE source records, or nested-result interaction.

The planned transport matrix uses Clang as the primary compiler and GCC as an independent compiler, each at low and optimized profiles. Sanitizers supplement but do not define conformance.

## Later profiles

`x86_64-linux-musl` needs a separate libc, startup, sysroot, dynamic-loader, and static-distribution profile even though its CPU ABI is SysV-derived. The declared `x86_64-windows-msvc` TargetSpec still needs complete executable LLP64 layout coverage plus a hosted receipt, independent or register-inspected four-slot call evidence, broader and bidirectional source-level Fairy crossings, versioned exact COFF symbol/COMDAT/relocation fixtures, complete version-2/handler/chain and Fairy/foreign-frame unwind fixtures, complete UCRT/VCRuntime and deployment evidence, and debug fixtures before any maturity promotion.

## External references

- x86-64 psABI project: <https://gitlab.com/x86-psABIs/x86-64-ABI>
- Microsoft x64 calling convention: <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- Microsoft `RaiseException`: <https://learn.microsoft.com/en-us/windows/win32/api/errhandlingapi/nf-errhandlingapi-raiseexception>
- Microsoft `RtlVirtualUnwind`: <https://learn.microsoft.com/en-us/windows/win32/api/winnt/nf-winnt-rtlvirtualunwind>
- Microsoft structured exception handler syntax: <https://learn.microsoft.com/en-us/windows/win32/debug/handler-syntax>
- musl reference manual: <https://musl.libc.org/doc/1.0.0/manual.txt>

These sources constrain target profiles; they do not define Fairy C source semantics.
