# ADR 0076: Microsoft x64 classifier fixtures

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Layout source: `internal/layout/`
- Classifier source: `internal/cabi/`
- Fixture source: `spec/fixtures/c-abi/manifest-windows-msvc-v0.json`
- Runner source: `internal/cabifixture/`

## Context

The Windows TargetSpec names Microsoft x64, but a string identity is not an executable call
contract. SysV keeps independent integer and SSE register pools and may split a small aggregate
across them. Microsoft x64 instead gives the first four parameter positions shared ordinal slots:
each position chooses either its integer register or its same-numbered XMM register. Records that
are not exactly 1, 2, 4, or 8 bytes travel through a pointer, and a large result's hidden pointer
occupies slot zero. Reusing the SysV planner would therefore produce plausible-looking but wrong
register plans.

## Decision

1. `cabi.New` validates the target identities once and selects either the existing SysV AMD64
   profile or a distinct Microsoft x64 profile. No classification depends on the Go host, local
   MSVC, SDK paths, registry state, or WSL.
2. Integer, enum, pointer, and function-pointer scalars classify as `INTEGER`; scalar `f32`, `f64`,
   `c.float`, and `c.double` classify as `SSE`. Naturally laid out arrays or records of exactly 1,
   2, 4, or 8 bytes classify as one integer value regardless of their field classes. Other admitted
   aggregates classify as `MEMORY` and travel indirectly.
3. Parameter positions zero through three select RCX/RDX/R8/R9 for integer or pointer transport and
   XMM0-3 for scalar floating transport. Position four and later use the stack. An indirect
   aggregate uses the integer register for its position or a pointer stack slot after position
   three.
4. Direct scalar floating results use XMM0. Other direct scalars and admitted small records use
   RAX. A `MEMORY` result receives its destination pointer in RCX, returns that pointer in RAX, and
   shifts every explicit parameter by one shared position.
5. Direct array parameters and results remain rejected because C adjusts array parameters and
   forbids array returns. Arrays may participate only inside an admitted record.
6. The separate Windows v0 manifest contains 15 sorted cases: three closed failures and twelve pass
   plans covering scalar, pointer, enum, function-pointer, mixed-position slots, the fifth stack
   argument, small integer and float-containing records, odd-size and 16-byte indirection, indirect
   stack transport, and hidden-return shifting.
7. Implementing and executing this host-independent ledger removes
   `windows-x64-abi-classifier` from missing target evidence. `bidirectional-c-abi-fixtures`, hosted
   receipt, exact stack/shadow-store, PE/COFF, unwind, target emission, runtime, and standard-library
   evidence remain missing.

## Consequences

- SysV behavior and its 17-case ledger remain unchanged and continue to own Linux native probes.
- A future MSVC probe can consume the twelve validated Windows pass plans without redefining the
  register contract.
- Variadic floating duplication, vector/intrinsic exceptions, by-reference temporary alignment and
  lifetime, callbacks, exports, and foreign unwinding require separate fixtures before admission.

## External basis

- Microsoft x64 parameter, aggregate, return, shadow-store, and stack rules:
  <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention>
- Microsoft x64 scalar sizes and alignment:
  <https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions>
