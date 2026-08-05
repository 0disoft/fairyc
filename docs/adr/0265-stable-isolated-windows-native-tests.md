# ADR 0265: Stabilize isolated Windows native tests

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Depends on: ADR 0176, ADR 0181, ADR 0212, ADR 0258

## Context

ADR 0181 stabilizes root-only `@test fn void` discovery, validation, ordering, isolation, and report
grammar, but executes only the semantic-interpreter subset. That lane is fast and host-independent;
it cannot prove target C emission, closed standard-library linkage, MSVC compilation, PE structure,
unwind metadata, or native trap behavior. Users must currently turn a test into a production
`fn i32 main()` and run it manually to obtain those signals, which bypasses the test harness and
cannot continue after a trap.

One image containing every test would be cheaper to compile, but a language trap terminates the
process before the harness can identify later outcomes. A stable native lane therefore needs one
bounded process per selected test while reusing the existing source-overlay and report contracts.

## Decision

1. Extend the stable spelling to `fairyc test [--native [--release]] [--locked]`. Omitting
   `--native` preserves ADR 0181's interpreter behavior exactly. `--release` is valid only with
   `--native`; native development uses the low MSVC profile and native release uses optimized.
2. Loading, lock handling, root-only `tests/` overlay, `@test` validation, canonical identity
   ordering, semantic analysis, Flow, MIR lowering, and initial MIR verification happen once and
   are shared by both engines. Tests remain excluded from production locks, builds, caches,
   interfaces, and emitted applications.
3. For each native test, clone the verified program and append one compiler-private, collision-
   checked `fn() -> i32` MIR harness. The harness calls exactly that test's verified zero-argument
   `void` symbol and returns zero. Re-run the MIR verifier before target emission; neither the
   harness name nor symbol is source-visible or stable compatibility data.
4. Emit only the harness-reachable closure for `x86_64-windows-msvc`, select the existing closed
   target-support components from used foreign symbols, compile one temporary PE, and inspect its
   PE32+ and AMD64 unwind structures before execution. Each test receives a fresh process and
   temporary working directory, EOF stdin, no program arguments, and no retained artifact.
5. A validated runtime trap receipt produces the existing `FAILED (<trap-category>)` line and does
   not stop later tests. A normal zero exit produces `ok`. A nonzero exit without the exact receipt,
   toolchain/build/inspection/execution failure, or internal harness failure aborts without a test
   report rather than inventing a language outcome.
6. Native stdout or ordinary stderr is outside this report-only contract. Any such output rejects
   the requested native subset with code 2 and no report. Program-stream testing, snapshots,
   inherited environment policy, arguments, shared fixtures, and output attachment require later
   contracts.
7. Native success and trap reports retain ADR 0181's exact text and codes 0/5. Invocation and
   unsupported backend/output capabilities use 2, source or `E9001` diagnostics use 3, internal or
   report-writer failure uses 4, native build/toolchain/inspection/execution failure uses 6, and
   workspace/test-source/lock failure uses 7. Ordinary lock maintenance may precede later failure;
   `--locked` writes nothing.

## Compatibility

This is an additive stable CLI change. Existing `fairyc test [--locked]` invocations remain
byte-for-byte interpreter commands with the same report and exit codes. The new flags do not change
the language revision, compiler-bundled SDK, runtime ABI, manifest, lockfile, package identity,
public interface, or production executable. Native test harness names, symbols, images, and working
directories are compiler-private ephemeral state.

## Evidence

- CLI unit tests preserve interpreter behavior and validate new flag combinations, duplicate
  rejection, and the requirement that `--release` accompany `--native`.
- MIR/backend tests construct the private harness, re-run the verifier, and require a reachable
  Windows C entry that calls the selected void test.
- Dependency-injected execution tests prove canonical order and continued compilation/execution
  after a validated assertion trap.
- The configured Windows smoke builds a separate `fairyc.exe`, runs a real failing native test
  followed by a passing test, and requires both report lines plus code 5. It then runs the checked-in
  dependency test successfully in development and release native profiles. Every generated image
  passes the stable PE and unwind inspections and no retained production build artifact changes.

## Consequences

Fairy C projects can now choose a fast deterministic semantic lane or an actual Windows MSVC lane
without changing their test sources. Native compilation currently occurs once per test and is
deliberately optimized for isolation and trustworthy attribution, not throughput. Filters,
workspace-member recursion, parallel execution, persistent native-test caches, shared fixtures,
program streams, arguments, benchmarks, fuzzing, machine-readable reports, portable targets, and
hosted native-test receipts remain separate work.
