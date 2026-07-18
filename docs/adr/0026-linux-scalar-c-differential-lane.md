# ADR 0026: Linux scalar C differential lane

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- C transport source: `docs/mir/c-backend-safety-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`
- CI source: `.github/workflows/frontend.yml`

## Context

Deterministic C text is not native execution evidence. The next gate must prove that independent C
compilers accept the emitted translation unit, link it with a bounded runtime surface, and preserve
the MIR interpreter's process outcome at different optimization levels. Silently skipping this gate
on a Windows host with no Linux toolchain would turn a missing dependency into a false green check.

The current scalar-payload fixtures return small non-negative values and produce no host output. One
fixture traps. A temporary process-status protocol can therefore observe both return values and canonical
trap categories without pretending to be the final runtime ABI or console contract.

## Decision

1. `internal/cbackendnative` is a separate native differential runner. It requires Linux on amd64,
   glibc reported by `getconf`, and both `clang` and `gcc` in `PATH`. A missing requirement is an
   error, not a skip.
2. Every emitted case is compiled and linked with each compiler under a `low` `-O0` profile and an
   `optimized` `-O2` profile using ISO C11, pedantic errors, common warnings as errors, and
   `-fno-common`.
3. Generated C calls the internal symbol `fy_runtime_trap(uint32_t)`. Canonical trap identities use
   fixed codes 1 through 7 independent of which categories occur in a translation unit.
4. `internal/cruntime/stage0_scalar.c` is the embedded provisional implementation shared with the
   experimental product-build path. The runner supplies a reserved status base of 192; the runtime
   terminates with `_Exit(base + category)` and emits no output. It is not the complete product
   runtime or a stable runtime ABI.
5. The process-status oracle accepts ordinary returns only in the non-reserved range 0 through 191.
   Negative and larger `i32` returns require a later output or shared-memory observation protocol
   and fail closed in this lane.
6. Compiler and executable processes have bounded time and captured-output limits. Each C-driver
   invocation owns temporary generated C, runtime source, and linked output; the runner separately
   owns executable materialization. Ordinary cleanup removes both classes after the run. Compiler-
   internal scratch remains toolchain-owned, and no repository artifact is retained.
7. The initial four cases produced 16 fixture comparisons. ADR 0029 expands the same matrix to eight
   scalar-payload cases and 32 comparisons, including Option and Result success and failure paths.
   ADR 0033 expands the matrix to eleven cases and 44 comparisons with flat-resource transport.
   ADR 0034 expands it again to seventeen cases and 68 comparisons with resource-bearing tags.
   After those comparisons, the same command invokes the actual experimental build CLI for Clang
   and GCC under both profiles and executes each temporary product, requiring exit `42` and the
   exact relative-path build receipt.
8. The original command prints a deterministic `fairyc.native-c-report/v1` JSON report containing
   target, glibc fingerprint, first-line compiler fingerprints, profile names, case identifiers,
   fixture execution count, exact drop-trace execution count, and product-build smoke count. It contains no temporary or absolute
   path.
   ADR 0035 changes the report shape to `fairyc.native-c-report/v2` when it adds the separate
   `dropTraceExecutions` count.
9. `.github/workflows/frontend.yml` owns the hosted Ubuntu invocation. The ordinary cross-platform
   Go job still compiles and unit-tests the runner without claiming native execution; the separate
   native job invokes the fail-closed command.
10. Adding the runner and workflow does not itself prove a hosted run or advance the target to
   `emittable`, `conformant`, or `supported`. Current run evidence must be observed separately.

## Consequences

- The repository now has a concrete `.fy -> verified MIR -> C11 -> Clang/GCC -> ELF -> process`
  path for the constrained scalar and flat-resource fixtures on an eligible Linux host.
- The current Windows development host cannot run the native command because it has no registered
  WSL distribution or C compiler. Ordinary Go validation remains useful but is not substituted for
  native evidence.
- Trap category identity becomes externally testable through reserved process status. ADR 0035
  later adds a closed binary stderr record only to instrumented native fixture units for exact
  source-linked drop comparison; product emission and product builds remain marker-free.
- The differential lane and experimental build now share compiler profiles and runtime source, so
  their transport cannot drift through duplicated plumbing.
- Output comparison, negative and wide return values, user drop-body effects, sanitizers, layouts,
  bidirectional C ABI calls, and the product runtime remain later gates.

## Rejected alternatives

- Skip when Clang or GCC is missing: converts absent evidence into success.
- Test only one compiler or optimization profile: cannot expose common implementation or optimizer
  dependence.
- Parse generated text instead of executing it: proves formatting, not C semantics.
- Use `abort()` alone: proves termination but loses the canonical trap category.
- Install WSL, a compiler, or a container runtime as a side effect of validation: mutates the host
  and hides the declared toolchain prerequisite.
