# ADR 0315: Stable native-test stdin file replay

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0265, ADR 0267, ADR 0313

## Context

Caller-owned `--stdin` safely serves one native test, but cannot deterministically feed several
isolated processes. Reusing one stream makes later input depend on earlier consumption, while
unbounded compiler buffering would add an unsafe memory policy.

## Decision

1. Add native-only `--stdin-file <portable-workspace-file>` and keep it mutually exclusive with
   `--stdin` and listing. A spelling after `--` remains a program argument.
2. Admit only canonical `/`-separated workspace-relative paths. Open through a containment-enforcing
   root after complete compiler admission, require a regular file, and read at most 65,536 bytes.
3. Read the file once and attach a fresh byte-zero reader to every selected isolated test. Empty
   files replay EOF. File failures are code 7 and launch no test process.
4. Preserve arguments, reports, schemas, output capture, trap continuation, lock behavior, and all
   SDK, language, runtime ABI, MIR, target, package, and generated-code versions.

## Evidence

Go tests cover option composition, containment, regular-file and size guards, and fresh-reader
replay. The Windows host smoke runs two generated-Fairy tests at both low and optimized MSVC and
proves that both receive the same checked-in fixture from byte zero.

## Consequences

Native suites can share deterministic binary fixture input without caller-stream ambiguity.
Per-test fixture selection, inputs larger than 64 KiB, streaming files, and portable native targets
remain separate contracts.
