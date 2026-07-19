# Used-Symbol Standard-Library Auto-Link

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- CLI source: `docs/cli/command-contract.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

The constrained C backend can emit calls to a closed set of project-owned `fy_std_*_v0` symbols,
but `experimental build` previously linked only the scalar runtime. Native source probes supplied
the relevant C unit themselves, so they did not prove that an ordinary product build could resolve
the same call.

Linking every provisional C unit into every binary would hide unused-symbol mistakes and work
against the project's build-time and artifact-size goals. Maintaining separate symbol allowlists in
the backend and linker would also permit silent drift.

## Decision

1. A generated C unit records the sorted unique external link names used by actual MIR call
   terminators. Foreign declarations without a call are omitted because they create no undefined
   reference in the generated unit.
2. `internal/cstdlibcontract` is the single source for the exact admitted standard-library symbols
   and their implementation-component ownership. The C backend uses it for namespace admission and
   the standard-library source resolver uses it for linking.
3. Component dependencies are closed explicitly: Buffer requires SystemHeap; Arena requires
   SystemHeap; owned text requires SystemHeap, Buffer, and text view. Linux File remains independent.
4. The resolver emits each required C component once in deterministic order. Linux File is first
   when selected so `_POSIX_C_SOURCE` precedes every system header in the combined support unit.
5. `experimental build` combines only that dependency-closed standard-library source with the scalar
   runtime. A program with no called standard-library symbol retains the scalar-runtime-only path.
6. Non-standard-library foreign symbols are not guessed, downloaded, or supplied. They still must
   resolve through the selected driver's closed environment, and the CLI still accepts no companion
   source, object, library, search path, or arbitrary linker argument.
7. The Linux native lane builds and executes a self-contained generated-Fairy call to the real
   UTF-8 validator through the product CLI under Clang/GCC and low/optimized profiles.
8. `fairyc.native-c-report/v28` adds `standardLibraryBuildSmokeExecutions`, expected to be four only
   after all four product builds execute with exit 42 and no output.
9. This auto-link does not create public `std::*` source modules, safe ownership wrappers, effect
   declarations, typed errors, arbitrary FFI link configuration, target conformance, or self-hosting.

## Consequences

- Raw `.fy` programs can now use an admitted internal standard-library symbol through
  `experimental build` without a test-only companion C source.
- Unused declarations do not pull in a component; called owned-text operations pull in their exact
  storage and validation dependencies rather than every provisional library slice.
- Hosted Linux execution remains required before the configured product-build count becomes
  observed evidence.
