# ADR 0022: Controlled interpreter I/O harness

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Fixture source: `spec/fixtures/interpreter/manifest-v0.json`

## Context

The interpreter needs deterministic output evidence for later differential backend tests, but the
runtime contract deliberately keeps console access out of compiler intrinsics. Recognizing a
source function by name would also let ordinary code accidentally acquire host authority.

## Decision

The interpreter exposes an options-only test harness. A fixture manifest may bind an existing
`fn(i32) -> i32` by canonical function name to one of two closed hooks:
`write-stdout-byte` or `write-stderr-byte`. The harness intercepts only the bound MIR call symbol,
accepts one value in `0..255`, appends it to an in-memory stream and ordered I/O trace, and returns
`1`.

Both streams share one explicit output byte limit. Crossing it returns the interpreter capability
category `output-limit` before appending the next byte while preserving the already captured
stdout, stderr, and ordered trace prefix. The maximum configurable limit is
1,048,576 bytes. Fixture bindings must be sorted, unique, signature-compatible, and accompanied by
a positive limit.

Default interpreter execution configures no hooks. Function names alone never activate the
harness, and `fairyc experimental run` has no option that grants it. No byte reaches a host file,
terminal, process stream, environment, or network endpoint.

## Consequences

- Interpreter fixtures can compare exact stdout bytes, stderr bytes, cross-stream order, and the
  retained partial prefix at the output limit without adding text formatting or operating-system
  emulation.
- A future generated-C harness can bind the same declared test functions and compare outputs to
  the interpreter without making them language intrinsics.
- Real console APIs, Linux `File`, partial writes, arguments, environment access, and process
  status remain standard-library/runtime work.

## Rejected alternatives

- Reserve magic function names: grants behavior by spelling and can change ordinary programs.
- Write directly to Go process streams: makes tests host-coupled and risks unbounded output.
- Add console operations to MIR: contradicts the runtime/standard-library boundary.
- Use strings as the first output payload: couples I/O evidence to unfinished text semantics and
  allocation.
