# ADR 0313: Stable single-native-test stdin forwarding

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0211, ADR 0212, ADR 0265, ADR 0267, ADR 0312

## Context

Stable run can attach caller stdin and native tests can receive program arguments, but every isolated
native test still receives EOF. Generated Fairy tests therefore cannot exercise the public
`std::io::read_stdin` path through the stable test runner. Attaching one caller stream to several
sequential test processes would make consumption depend on canonical test order. Replaying it would
require compiler-owned buffering, a byte limit, and failure semantics that the current command does
not otherwise need.

## Decision

1. Advance stable workspace test behavior to v12 with explicit
   `fairyc test --native [--release] [--stdin] ...`. Without `--stdin`, every native test retains the
   existing EOF input.
2. Admit `--stdin` only for native execution. Semantic execution and `--list` reject it. A literal
   `--stdin` after the first program-argument delimiter remains an ordinary process argument.
3. Require exactly one selected test after complete overlay declaration, semantic, Flow, MIR, and
   verifier admission. Zero or multiple selected tests fail as code 2 without launching a process or
   consuming caller input. Ordinary lock maintenance may already have occurred; `--locked` remains
   the no-write form.
4. After compilation, linking, PE/unwind inspection, and immediately before the selected process
   starts, attach the caller-owned reader directly. The compiler does not pre-read, copy, buffer,
   replay, retain, decode, or impose a separate byte ceiling. The caller owns content and EOF.
5. Preserve process arguments, working-directory isolation, trap classification, program-output
   rejection, report ordering and bytes, JSON schemas, exit codes, and artifact non-retention.
   Program stdin is execution input and is absent from reports.
6. Keep SDK `0.0.0-stage0.45`, language `fairyc-language/10`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, interface, cache, and generated-code versions unchanged.

## Evidence

- Go tests cover native-only parsing, list rejection, exact-one selection, unchanged default EOF,
  and direct caller-reader plus program-argument forwarding to the selected executor.
- The checked-in `examples/native_test_stdin_quickstart` calls `std::io::read_stdin` from a real
  effectful `@test`, checks exact binary bytes including an embedded NUL, preserves an untouched
  caller-storage sentinel, and observes EOF on the next read.
- The configured Windows host smoke runs that exact test once in development and once in release,
  verifies ordinary lock publication followed by `--locked` verification, and structurally inspects
  both generated PE images.

## Consequences

An argument- and input-sensitive CLI behavior can now be tested through one stable generated-Fairy
native invocation without a test-only adapter. Several tests cannot implicitly share or replay one
caller stream; callers select one exact test or invoke the command separately. Native program-output
attachments, fixture-file input, per-test input declarations, portable native targets, environment
mutation, parallel execution, and retries remain separate contracts.
