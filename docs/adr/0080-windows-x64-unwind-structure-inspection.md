# ADR 0080: Windows x64 unwind structure inspection

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Inspector source: `internal/pecoff/`
- Local runner source: `cmd/fairyc-windows-host-smoke/`

## Context

The Windows TargetSpec names `.pdata`/`.xdata` unwind, while ADR 0078 stops at generic executable
structure and ADR 0079 stops at generic COFF object bounds. Merely observing an exception-directory
entry or section name does not prove that runtime-function entries are sorted, cover executable
code, reference file-backed unwind records, or contain bounded operation arrays. The public
Microsoft Learn x64 exception page still describes unwind version 1 as current, but local MSVC
14.51 produces version 2 records in the linked CRT image.

## Decision

1. `internal/pecoff` maps the PE32+ exception directory and requires a readable non-executable
   `.pdata` range whose size is an exact multiple of the 12-byte `RUNTIME_FUNCTION` structure.
2. Runtime-function entries must be sorted by begin RVA, nonoverlapping, nonempty, and wholly inside
   executable code. Their unwind RVAs must be nonzero, DWORD-aligned, and file-backed.
3. Every unique `UNWIND_INFO` record must be readable non-executable data. Its version, flags,
   prolog size, frame fields, code-slot count, and aligned optional tail must fit inside one
   file-backed section.
4. Version 1 records admit the documented prolog operations 0-5 and 8-10 with their exact one-,
   two-, or three-slot widths. Primary code offsets must be within the prolog and sorted descending.
   Handler and chain flags cannot conflict.
5. Version 2 records are admitted because the selected local MSVC emits them. Opcode 6 is treated as
   a bounded epilog descriptor: its first long form consumes two slots when the low OpInfo bit is
   clear; its short form and later descriptors consume one. Opcode 7 consumes one spare slot.
   Remaining prolog operations retain version-1 validation. This decision validates record bounds,
   not complete epilog offset meaning.
6. Exception or unwind handler tails must name executable code. Chained tails must contain a
   nonempty executable runtime-function range and aligned unwind RVA. Language-specific handler
   payloads and recursive chain equivalence remain opaque.
7. Host-independent tests cover one valid runtime function, unused nonzero `UWOP_SET_FPREG` OpInfo,
   version-2 short and long epilog descriptors, invalid directory size, empty functions, unmapped
   unwind records, invalid versions/flag combinations, truncated code arrays, and truncated v2
   descriptors.
8. The configured Windows smoke applies the inspector to the retained product and both profiles of
   the layout and ABI probes. Its text receipt reports the retained product's runtime-function and
   version-2 record counts dynamically; neither count is a stable contract.
9. This does not clear `windows-x64-unwind-fixtures`. Complete version-2 epilog semantics,
   language-specific handlers, recursive chain equivalence, object-level unwind relocations,
   deliberate exception/unwind execution, hosted receipts, and debugger behavior remain separate
   gates.

## Consequences

- Malformed or out-of-range Windows x64 unwind metadata now fails before any local probe is accepted.
- The project records the observed gap between public version-1 prose and current MSVC version-2
  output instead of rejecting current toolchains or silently treating v2 as v1.
- Static structure inspection remains a supporting oracle; it cannot substitute for an actual
  exception dispatch and stack-unwind fixture.

## External basis

- Microsoft x64 exception handling and documented version-1 unwind operations:
  <https://learn.microsoft.com/en-us/cpp/build/exception-handling-x64?view=msvc-170>
- Microsoft PE/COFF exception-directory and section format:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
