# ADR 0312: Stable native-test program argument forwarding

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0261, ADR 0265, ADR 0267, ADR 0287

## Context

Stable `fairyc run` already separates compiler options from program arguments with the first `--`,
but stable native tests always launched each isolated executable with an empty program-argument
list. A CLI program could therefore exercise `std::process` through `run` but could not retain the
same argument-dependent behavior as an isolated native regression test. Interpreter tests cannot
model host process startup and must remain deterministic and host-independent.

## Decision

1. Advance stable workspace test behavior to v11 with
   `fairyc test --native [--release] ... -- <program-args...>`. The first `--` is excluded and every
   following value is forwarded, without reinterpretation, to each selected isolated native test
   process in canonical execution order.
2. Keep arguments before the delimiter in the existing compiler-option domain. Flag-looking,
   empty, space-containing, quoted, backslash-containing, and non-ASCII values after the delimiter
   are ordinary program arguments. A second `--` after the delimiter is also an ordinary argument.
3. Reject an embedded NUL before workspace loading or compilation. Reject the delimiter unless
   `--native` is present. Semantic interpreter execution and `--list` therefore retain no process
   argument or host-startup contract.
4. Reuse the stable run splitter's validation kernel while retaining command-specific diagnostics:
   NUL rejection names `test`, not `run`. Native execution passes the same validated argument
   sequence to each fresh temporary image.
5. Preserve discovery, exact directory/file/identity selection, package/workspace qualification,
   complete-overlay admission, isolated working directories, trap continuation, output rejection,
   lock behavior, exit codes, text report bytes, and JSON schemas. Program arguments are execution
   input and are not copied into reports.
6. Keep SDK `0.0.0-stage0.45`, language `fairyc-language/10`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, interface, cache, and generated-code versions unchanged.

## Evidence

- Go tests cover compiler/program separation, flag-looking, empty and non-ASCII values, semantic
  rejection, command-specific NUL rejection, and identical forwarding to every isolated executor.
- The checked-in `examples/native_test_args_quickstart` imports `std::process` from a real `@test`
  function and asserts argv count plus exact UTF-8 bytes for `--locked`, an empty argument, and
  `한글λ`.
- The configured Windows host smoke runs that exact test once in development and once in release,
  verifies ordinary lock publication followed by `--locked` verification, and structurally
  inspects both generated PE images.

## Consequences

CLI applications can keep argument-sensitive behavior in stable isolated native tests without a
test-only argv adapter. One invocation supplies the same values to every selected test, so tests
that require different argument sets remain separate command invocations. Stdin forwarding,
captured program output, environment mutation, per-test argument declarations, portable native
targets, parallel execution, retries, and report attachments remain separate contracts.
