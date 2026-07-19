# ADR 0081: Windows x64 native unwind execution

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Local runner: `cmd/fairyc-windows-host-smoke/`
- Generator: `internal/windowsunwindprobe/`

## Context

ADR 0080 validates linked `.pdata` and referenced unwind-record structure, but a structurally
bounded record can still describe the wrong machine operation. Ordinary successful calls also do
not ask the Windows dispatcher to search frames or run termination handlers. The next evidence
layer therefore needs an intentional native exception path while preserving Stage 0's language
rule that Fairy C calls and traps do not unwind.

## Decision

1. A closed generator owns exactly three observable cases: exception-filter payload transport,
   captured-context virtual walking, and abnormal termination-handler execution.
2. Local x64 MSVC compiles three C translation units separately at low and optimized profiles:
   an outer catcher/context walker, a stack-owning `__try`/`__finally` frame, and a private software
   exception raiser.
3. The raiser calls `RaiseException` with a private noncontinuable code and two deterministic
   application arguments. The outer `__except` filter requires the exact code and arguments.
4. The filter copies the provided `CONTEXT`, then repeatedly combines
   `RtlLookupFunctionEntry` and `RtlVirtualUnwind(UNW_FLAG_NHANDLER, ...)`. The observed program
   counters must encounter the raiser, termination frame, and outer catcher in that order within a
   fixed 64-frame bound.
5. Returning `EXCEPTION_EXECUTE_HANDLER` triggers the actual unwind. The intervening `__finally`
   must execute exactly once, report `AbnormalTermination()` true, preserve its checked stack
   state, and never report a normal exit.
6. Each build must first pass raw COFF inspection for all three objects and linked PE32+/unwind
   structure inspection. The process must exit zero with no output under both profiles.
7. This fixture is explicit native Windows target evidence. It does not add source exceptions,
   unwind successors, or cleanup-on-trap to Fairy C. It does not prove version-2 epilog offsets,
   language-specific handler payload parsing, recursive chain equivalence, object-level unwind
   relocations, Fairy/foreign-frame crossings, hosted Windows execution, or debugger behavior.
   `windows-x64-unwind-fixtures` remains fail-closed.

## Consequences

- The local Windows lane now distinguishes static unwind-record admission, virtual context walking,
  and real handler-directed unwind execution instead of collapsing them into one Boolean.
- A missing or wrong native frame fails with a dedicated nonzero process result, while compile,
  object, link, image, and execution failures remain separately reported by the Go runner.
- The probe uses Windows APIs only inside temporary generated C; Fairy C compiler and product
  executables do not acquire a Windows exception runtime dependency.

## References

- Microsoft `RaiseException` and frame search:
  <https://learn.microsoft.com/en-us/windows/win32/api/errhandlingapi/nf-errhandlingapi-raiseexception>
- Microsoft `RtlVirtualUnwind`:
  <https://learn.microsoft.com/en-us/windows/win32/api/winnt/nf-winnt-rtlvirtualunwind>
- Microsoft structured exception handler syntax:
  <https://learn.microsoft.com/en-us/windows/win32/debug/handler-syntax>
- Microsoft `AbnormalTermination`:
  <https://learn.microsoft.com/en-us/windows/win32/debug/abnormaltermination>
