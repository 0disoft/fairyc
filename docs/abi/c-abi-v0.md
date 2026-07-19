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

The Windows Go compiler host does not define target layout. Proving the first target requires WSL or Linux CI, an explicit glibc baseline, a sysroot or runner image, Clang and GCC, and a target runtime bundle.

The current Go prototype can emit deterministic ISO C11 for a verified scalar, inert-struct,
flat-resource, and resource-tagged profile carrying this target identity. The inert-struct slice
includes nonempty source `@repr(C)` records whose fields are currently limited to `u8`, `i32`,
`u32`, `usize`, and raw pointers. It preserves the public C tag and field names and emits target-
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
`i32`, `u8`, `u32`, `usize`, the distinct transport-only `c::int` and `c::size_t`, raw pointers to the fixed-width
Fairy scalar set, and raw pointers to an
admitted nonempty `@repr(C)` struct. AST, typed HIR, MIR, unsafe provenance, deterministic C
source/link identity, prototype emission, typed null, separate-object cross-compiler linking, and process-status
observation are wired. Source struct construction and field reads reach verified MIR, the MIR
interpreter for its executable scalar subset, and generated C. Inside an explicit unsafe boundary,
an exact `&T -> *T`, `&mut T -> *T`, or `&mut T -> *mut T` cast may take the address of a direct
local place without inventing alias promises. Raw dereference, pointer arithmetic, integer-pointer
conversion, mutability strengthening, and other casts remain unsupported. Source `c::int` maps to
target key `c.int`, emits exact C `int`, and is guarded by target-derived size and alignment
assertions. Exact `c::size_t` maps separately to target key `c.size_t`, includes `<stddef.h>`, emits
C `size_t`, and asserts target size, alignment, and unsignedness instead of aliasing `usize` or
`uintptr_t`. Their only current value path is foreign return to typed local to foreign argument; they have
no source literal, arithmetic, comparison, cast, interpreter behavior, or aggregate-field support.
Raw pointers to C scalars also remain outside the admitted backend slice. This is not implementation of exports, callbacks, other `c.*` scalars,
nested/array/enum `@repr(C)` fields, or by-value aggregate calls and returns. The interpreter
remains fail-closed on every foreign call.

## C-representable types

`@repr(C)` permits fixed-width integers, `c.*` scalars, `f32`, `f64`, raw pointers, fixed arrays, nested `@repr(C)` structs, and an explicitly based C enum. It rejects resources, safe borrows, slices, `str`, `CStr`, `Option`, `Result`, `Region`, `String`, `Buffer`, payload enums, empty structs, packed or custom alignment, unions, bit fields, long double, atomics, and vector types.

The implemented source/backend slice is deliberately smaller than that language contract: it
accepts only nonempty, flat structs over `u8`, `i32`, `u32`, `usize`, and raw pointers. `c::int` and
`c::size_t`
does not broaden that aggregate slice. Other
contracted field families remain fail-closed until their source typing, layout, ABI, and native
crossing tests are connected together.

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

Semantic target data includes target identity, architecture, OS and environment, endianness, pointer and function-pointer layouts, scalar layouts, `usize` and `isize`, C scalar layouts and signedness, natural alignment, aggregate layout and C ABI classifier identities, stack and return rules, object format, symbol rules, unwind model, debug contract level, and supported `@repr(C)` features.

It contains no local executable or SDK paths.

`internal/target` now implements the first host-independent description under
`fairyc.target-spec/v0`. For `x86_64-linux-gnu` it fixes little-endian LP64, eight-byte pointers and
function pointers, the Stage 0 language and permitted C scalar size/alignment table, 16-byte SysV
call alignment, the 128-byte red zone, SysV AMD64 aggregate and return-classifier identities,
ELF64 symbols, DWARF CFI unwind, generated-C DWARF debugging, and the permitted `@repr(C)` feature
vocabulary. The scalar tables are ordered by canonical type name and every alignment is validated
as a nonzero power of two.

These entries define target facts for later compiler stages. They do not claim that every listed
source scalar, `c.*` name, aggregate form, or call class is accepted by the current semantic
frontend or C backend.

`internal/layout` now executes those facts for scalars, pointers, explicitly based C enums,
positive fixed arrays, and nonempty nested structs. Its versioned v0 manifest records exact size,
natural alignment, declaration-order field offsets, top-level internal padding, tail padding, and
closed rejection codes across 15 cases. This proves the internal target-layout algorithm against
repository expectations only. The ten valid cases now also generate a private recursive C11 probe
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

This executable description is target data, not conformance evidence. Its evidence block names the
existing `scalar-and-flat-i32-resource-tagged-c11` subset, keeps `emittable`, `conformant`, and `supported` false, and
lists the missing hosted native C layout-probe receipt, bidirectional ABI, runtime, standard-library, hosted-Linux, toolchain
baseline, and provisioning gates. `fairyc experimental target-spec x86_64-linux-gnu` exposes that
same immutable data without probing the host.

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
form before the target can claim that source-level gate. The current cross-compiler C-object probe
checks both compiler directions for the existing plans but does not substitute for Fairy lowering.

The planned transport matrix uses Clang as the primary compiler and GCC as an independent compiler, each at low and optimized profiles. Sanitizers supplement but do not define conformance.

## Later profiles

`x86_64-linux-musl` needs a separate libc, startup, sysroot, dynamic-loader, and static-distribution profile even though its CPU ABI is SysV-derived. `x86_64-windows-msvc` needs its own LLP64 layouts, four-register and shadow-space calling rules, PE/COFF, UCRT/VCRuntime, and unwind metadata contract.

## External references

- x86-64 psABI project: <https://gitlab.com/x86-psABIs/x86-64-ABI>
- Microsoft x64 calling convention: <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- musl reference manual: <https://musl.libc.org/doc/1.0.0/manual.txt>

These sources constrain target profiles; they do not define Fairy C source semantics.
