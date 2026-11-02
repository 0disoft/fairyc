# C callback interoperability

This development example exercises the checked-in synchronous `unsafe extern C fn` boundary. `main.fy` calls the independent C helper `legacy_filter` with the non-capturing Fairy predicate `keep_large` and an explicit local visit counter.

`legacy_filter` examines `[2, 12, 7, 20, 1]` and calls `keep_large` once per value. The predicate accepts values `>= 10` and calls `legacy_note_visit`, which increments the counter. After five callbacks, Fairy checks `accepted == 2` and `visits == 5`, then calls `legacy_report` to print:

```text
accepted=2 callbacks=5
```

On Windows, the program's stdout line ends with CRLF. The harness then prints an `interop ok: ...` line with the selected profile and MSVC version.

## Files

- [main.fy](main.fy) declares the C imports, provides the callback, owns the local counter, and checks the results.
- [legacy.c](legacy.c) filters the fixed values, invokes the callback, increments the visit counter, and reports the totals.
- [legacy.h](legacy.h) is an ordinary C-owned header included by `legacy.c`; Fairy does not auto-import it.
- [main.go](../../tools/dev/fairyc-c-interop/main.go) is the repository development harness.

## Run

Use Windows amd64 with Go, MSVC, and the Windows SDK. From the repository root:

```text
go run ./tools/dev/fairyc-c-interop --profile low
go run ./tools/dev/fairyc-c-interop --profile optimized
```

Both profiles must print the expected line and exit with status 0. The harness sends `main.fy` through the normal frontend, HIR, borrow Flow, and verified MIR, emits through the private C11 backend, compiles the Fairy output and original C source as separate MSVC objects, links them, and executes the result.

## Integration boundary

The C-to-Fairy direction is a synchronous, same-thread, call-bounded callback made while `legacy_filter` is running. It is not a Fairy source export or a retained or asynchronous callback. This is an `unsafe` boundary with a trusted contract: C must keep the context valid while calls are active, must not retain or use the callback or context after return, and must avoid unwinding or `longjmp` across Fairy frames.

This is a developer integration harness, not `fairyc build --c-file` and not a supported public export or library feature. Generated C11 is a compiler-private bootstrap transport under [ADR 0335](../../docs/adr/0335-private-c11-bootstrap-transport.md), not a stable generated-C interface or standalone distribution format. Native Linux execution remains queued, and this harness does not use WSL.
