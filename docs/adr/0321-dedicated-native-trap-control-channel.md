# ADR 0321: Dedicated native trap control channel

- Status: Accepted
- Date: 2026-08-08
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0207, ADR 0265, ADR 0314

## Decision

1. Stop transporting compiler-private native trap receipts through program stderr. A run-only
   Windows runtime instead writes exactly one raw category byte to a dedicated inherited anonymous
   pipe immediately before its reserved trap exit.
2. Pass the write handle through the private `FAIRYC_TRAP_CONTROL_HANDLE_V1` process environment
   entry and an explicit Windows inherited-handle allowlist. Close the parent writer after process
   start, drain the control pipe concurrently under an eight-byte fail-closed bound, and retain
   stdout/stderr under their existing independent shared bound.
3. Attach the private handle only to artifacts compiled with run trap control. Ordinary retained
   builds, direct target commands, probes, and receipt-free scalar runtimes receive no control
   handle or private environment entry.
4. Classify a language trap only when the reserved process status and the exact single control byte
   agree. An empty channel preserves an ordinary colliding `i32` status. Unexpected, mismatched,
   repeated, or oversized control data is an execution-infrastructure failure.
5. Preserve every program stdout/stderr byte. Receipt-like stderr text is ordinary program data and
   cannot create, suppress, or alter a trap outcome. Native JSON v4 therefore encodes stderr
   directly without removing a suffix.
6. Keep CLI commands, report schemas, exit codes, SDK, language, runtime ABI, target, MIR, cache,
   lockfile, and generated-code identities unchanged. The control transport is compiler-private and
   Windows-run-only.

## Evidence

- Go tests cover reserved ordinary results, exact matching control, forged legacy stderr text,
  missing control, mismatched control, repeated bytes, and control on an ordinary status.
- The configured Windows host smoke compiles the control runtime with MSVC, runs stable workspace
  and isolated native-test traps, and preserves a forged legacy receipt suffix as program output
  while classifying the real trap from the inherited pipe.

## Consequences

Program output is no longer a control plane. Stable run and native-test capture keep byte-exact
stderr even when it ends in the former private receipt spelling. The inherited handle and its
environment entry exist only for the lifetime of one temporary controlled execution; they are not
a public runtime ABI, sandbox, authentication mechanism, or portable-target contract.
