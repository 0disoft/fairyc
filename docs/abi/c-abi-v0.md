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

The current Go prototype can emit deterministic ISO C11 for a verified scalar-payload profile
carrying this target identity, including private `Option<i32>` and `Result<i32, i32>` transport.
A separate Linux/amd64 runner is configured to require glibc, Clang, and GCC, link the embedded
scalar runtime, and compare process status at low and optimized profiles. No hosted success receipt
has been observed, and the runner does not inspect layout or objects, exercise the public C ABI,
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

## C-representable types

`@repr(C)` permits fixed-width integers, `c.*` scalars, `f32`, `f64`, raw pointers, fixed arrays, nested `@repr(C)` structs, and an explicitly based C enum. It rejects resources, safe borrows, slices, `str`, `CStr`, `Option`, `Result`, `Region`, `String`, `Buffer`, payload enums, empty structs, packed or custom alignment, unions, bit fields, long double, atomics, and vector types.

C enums specify their underlying `c.*` integer. The backend may transport them as a fixed integer type plus constants rather than relying on a host compiler's default enum size.

By-value `@repr(C)` struct arguments and returns are included. The target layout engine must agree with the C compiler's actual ABI classification.

## Functions and callbacks

C variadics are rejected and require a fixed-signature shim.

The only Stage 0 callback form is non-capturing, synchronous, same-thread, call-bounded `unsafe extern C fn` with explicit user data. Nullable, retained, asynchronous, cross-thread, and exception- or `longjmp`-crossing callbacks remain unsafe raw integrations.

Fairy source TLS, C TLS symbol import or export, C++ ABI, and foreign unwinding are excluded.

## Startup, linking, and debugging

The C compiler driver owns compilation, CRT startup objects, default libraries, and linking. The generated program uses a small C entry shim that calls `fn main() -> i32`. Stage 0 does not directly invoke `ld`, `lld`, or a native object writer.

Debugging is delegated to generated-C `#line` directives and the C toolchain's DWARF output. Accurate Fairy local-variable and ownership-state debugging is not promised.

## `TargetSpec`

Semantic target data includes target identity, architecture, OS and environment, endianness, pointer and function-pointer layouts, scalar layouts, `usize` and `isize`, C scalar layouts and signedness, natural alignment, aggregate layout and C ABI classifier identities, stack and return rules, object format, symbol rules, unwind model, debug contract level, and supported `@repr(C)` features.

It contains no local executable or SDK paths.

## `ToolchainSpec`

Toolchain data includes C dialect, compiler and linker driver identities, version fingerprint, sysroot, libc family and baseline, startup objects, default libraries, dynamic loader, sanitizer capabilities, required flags, and local capability resolution. Local installation paths are resolution results, not target semantics.

## Conformance fixtures

Layout fixtures cover size, alignment, offsets, padding, nested structs, fixed arrays, explicit C enums, pointer fields, and rejected types. They include small register-sized aggregates and larger aggregates that exercise memory and hidden-return paths.

Call fixtures are bidirectional: Fairy-to-C and C-to-Fairy. They cover integer and floating register exhaustion, mixed arguments, narrow signed and unsigned values, pointers, by-value structs, hidden aggregate returns, nested aggregates, null raw pointers, and the permitted callback form.

The planned transport matrix uses Clang as the primary compiler and GCC as an independent compiler, each at low and optimized profiles. Sanitizers supplement but do not define conformance.

## Later profiles

`x86_64-linux-musl` needs a separate libc, startup, sysroot, dynamic-loader, and static-distribution profile even though its CPU ABI is SysV-derived. `x86_64-windows-msvc` needs its own LLP64 layouts, four-register and shadow-space calling rules, PE/COFF, UCRT/VCRuntime, and unwind metadata contract.

## External references

- x86-64 psABI project: <https://gitlab.com/x86-psABIs/x86-64-ABI>
- Microsoft x64 calling convention: <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- musl reference manual: <https://musl.libc.org/doc/1.0.0/manual.txt>

These sources constrain target profiles; they do not define Fairy C source semantics.
