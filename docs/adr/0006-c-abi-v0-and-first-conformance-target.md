# C ABI v0 and First Conformance Target

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

The compiler host and generated-program target are separate decisions. Stage 0 needs one concrete target environment to verify layout, C calls, runtime entry, file semantics, and generated-C behavior, but a successful hello-world build is not enough to claim platform support.

## Decision

1. The first planned conformance target is `x86_64-linux-gnu`. The Windows development host is not a target-layout authority.
2. The target remains planned, not supported, until its complete ABI, runtime, standard-library, toolchain, and CI fixtures pass without skips.
3. `x86_64-linux-musl` is the next static-distribution research profile. `x86_64-windows-msvc` follows unless Windows embedding becomes the primary product need.
4. Stage 0 exposes no stable Fairy native ABI, mangling, binary package format, or dynamic-library contract.
5. The only external binary contract is explicit C ABI v0 through `unsafe extern C fn`, `@repr(C)`, and `@link_name`.
6. `TargetSpec` owns semantic layout and calling-convention facts. `ToolchainSpec` owns compiler driver, sysroot, libc or CRT, startup objects, linker driver, flags, and capability fingerprints.
7. The C compiler driver performs compilation and linking. Stage 0 does not invoke a native linker directly.
8. Debug support is limited to generated-C `#line` mapping and toolchain-produced debug information.

## Rationale

The SysV AMD64 environment provides a documented aggregate classification and permits independent Clang and GCC transport checks. Linux also gives the initial `File`, `CStr`, allocator, and process-entry contracts one explicit operating environment. Choosing it creates a real WSL or Linux-CI dependency; it does not make cross-compilation free.

## Consequences

- Windows-native execution is not a Stage 0 conformance claim.
- The first file and path APIs are Linux-specific and cannot be advertised as portable.
- C ABI fixtures are required before a target advances from emittable to conformant.
- Toolchain versions and glibc baseline remain UNDECIDED until the executable harness exists.

## Revisit conditions

Change the first target only if Linux CI or toolchain provisioning is unavailable, Windows embedding becomes the primary product slice, or ABI fixtures expose a blocker that a different target resolves without widening the language.
