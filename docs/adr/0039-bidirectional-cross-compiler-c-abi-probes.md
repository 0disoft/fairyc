# ADR 0039: Cross C ABI fixtures between Clang and GCC objects

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- ABI source: `docs/abi/c-abi-v0.md`
- Fixture source: `spec/fixtures/c-abi/manifest-v0.json`
- Driver source: `internal/cdriver/`
- Implementation source: `internal/cbackendnative/`

## Context

ADR 0038 executes the target's SysV AMD64 class and register plan, but both the layout engine and
classifier are repository-owned Go code. They can agree with each other and still encode the wrong
native boundary. Compiling caller and callee in one translation unit is also weak evidence because
one compiler can use private knowledge, inline the call, or make the same mistake on both sides.

The source-level Fairy C `extern C` surface is not implemented yet. The next honest gate is therefore
not “Fairy-to-C works”; it is “independent required C compilers agree on the exact target-laid-out
types and fixed signatures that future lowering will use.”

ADR 0046 later implements the first exact-name scalar source slice. This context remains the reason
the broader aggregate matrix was established before source lowering existed; it is no longer a
statement that every source `extern C` form is absent.

## Decision

1. The native runner consumes only the fourteen validated pass signatures exposed by
   `cabifixture.NativeProbeCases`. The three fail cases remain Go-owned rejections and never reach a
   C compiler.
2. Every case generates deterministic caller and callee ISO C11 sources with the same private
   numeric typedefs, numeric struct tags, field names, function declaration, and target-layout static assertions.
   Fixture IDs, source names, absolute paths, and user identifiers do not enter generated C.
3. The caller initializes every scalar, aggregate, array field, null pointer, and function pointer
   with bounded exact values, invokes one external target function, then checks both callee status
   and the returned value. The callee checks every received value and constructs the expected
   result. A single external integer failure cell reports void-return and argument-only failures.
4. Direct arrays remain forbidden signature types. Fixed arrays cross only as fields of a
   C-representable struct, matching C function declarator rules.
5. `cdriver.BuildCrossCompiler` compiles caller and callee into separate objects with the selected
   low or optimized profile. It requires distinct resolved Clang and GCC toolchains with the same
   target and glibc fingerprint, then links through the caller's driver.
6. Each signature runs in both directions—Clang caller/GCC callee and GCC caller/Clang callee—at
   `-O0` and `-O2`. Four executions per case produce 56 configured crossings. Missing compilers,
   compile/link failures, diagnostics promoted by the warning policy, output, timeouts, or nonzero
   process status fail the lane rather than skip.
7. The native report advances from `fairyc.native-c-report/v4` to
   `fairyc.native-c-report/v5`. It adds the exact sorted `abiCases`, exact two `abiDirections`, and
   `abiProbeExecutions` count. No object, executable, generated source, or temporary path is
   retained in the report.
8. A configured path is not a hosted receipt. The target's `bidirectional-c-abi-fixtures` and
   `hosted-linux-receipt` missing markers remain until a clean hosted v5 report is observed.
9. This probe proves cross-compiler C transport for the current type/signature set. It does not
   implement or prove Fairy source FFI lowering, narrow signed/unsigned extension diagnostics,
   non-null raw pointer dereference, callback invocation semantics, exact stack offsets,
   assembly/object metadata, foreign unwinding, or public ABI stability.

## Consequences

- Small mixed returns, homogeneous pairs, hidden aggregate returns, integer/SSE exhaustion, and
  whole-aggregate rollback now have an independent native execution path.
- Separate compiler objects prevent same-translation-unit optimization from masquerading as ABI
  evidence.
- The C driver gains one closed validation-only object/link path; product builds remain one
  compiler, one generated unit, and one scalar runtime.
- Real Fairy-to-C and C-to-Fairy fixtures remain necessary after source `@repr(C)` and `extern C`
  lowering exists.

## Source-level runtime closure correction (2026-07-22)

Later source-level foreign-call probes reuse the cross-compiler driver with generated Fairy caller
code. Checked scalar conversions in that caller can reference `fy_runtime_trap`; compiling only the
caller and foreign callee objects left that symbol unresolved. `BuildCrossCompilerWithRuntime` now
compiles the exact scalar runtime as a third caller-toolchain object and links all three objects.
The foreign callee remains an independent translation unit compiled by the other toolchain. Raw C
ABI probes that do not use generated Fairy support continue through the original two-object API.

## Rejected alternatives

- Compile caller and callee together: permits inlining and one-compiler self-agreement.
- Test only Clang caller to GCC callee: misses direction-specific return and argument lowering.
- Parse assembly as the primary oracle: couples the first gate to compiler text formatting and
  optimization details rather than observable cross-object behavior.
- Clear target evidence when code is merged: substitutes configured intent for an observed hosted
  run and implemented source FFI.
