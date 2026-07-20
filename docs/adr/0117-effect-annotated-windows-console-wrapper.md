# ADR 0117: Effect-annotated Windows console wrapper

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0091 admitted one Windows-only stdout transport and ADR 0094 moved it behind a safe FY helper
module. That contained raw pointers and `unsafe`, but before ADR 0116 the helper had no way to tell
its safe callers that it could mutate a process stream. Leaving the existing example unannotated
after source effects exist would teach the wrong boundary: memory safety alone does not make I/O
pure.

The helper still returns a private boolean success test rather than a typed public error. It is
therefore suitable as executable effect evidence, not as the portable `std::io` design.

## Decision

1. The raw `fy_std_windows_stdout_write_all_v0` declaration carries `@effects(console)` in its FY
   support module in addition to its independent `unsafe extern C` boundary.
2. The safe `stdout_contract_edge_states` and `stdout_write_text` wrappers each declare
   `@effects(console)`. The pure `byte_view_ready` helper declares no effects.
3. The example `main` declares `@effects(console)` and may call both wrappers. Removing that
   declaration must produce `E7102` before MIR or native code generation.
4. The existing exact runtime contract is unchanged: success and status meanings, completed-prefix
   reporting, bounded capture, target-only support selection, output bytes, and process status all
   remain owned by ADR 0091.
5. This change does not create a portable public `std::io` module. Typed console errors, stderr,
   stdin, terminal encoding, redirection policy, Linux parity, formatting, and stable package
   identity remain later contracts.

## Consequences

- The first real host operation now exercises the same source-effect rule as ordinary semantic
  fixtures rather than bypassing it through an unsafe support module.
- `unsafe` proves the raw memory and FFI crossing; `console` separately propagates externally
  observable authority through safe calls.
- ADR 0118 later replaces the local boolean result with a progress-preserving typed record while
  retaining this effect boundary.

## Verification

- semantic tests cover foreign effects independently from unsafe and reject a missing caller set;
- the WSL-free Windows smoke compiles the two-module FY example with x64 MSVC, forwards exactly
  `Fairy C says hi\n`, observes no stderr, and preserves process status 42 both in-process and through
  a separately built Go-bootstrap `fairyc.exe`.
