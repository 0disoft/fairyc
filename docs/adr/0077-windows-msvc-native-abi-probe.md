# ADR 0077: Windows MSVC native ABI probe

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Classifier source: `internal/cabi/`
- Fixture source: `spec/fixtures/c-abi/manifest-windows-msvc-v0.json`
- Shared C type source: `internal/msvccprobe/`
- Probe generator source: `internal/windowsabiprobe/`
- Driver source: `internal/cdriver/`
- Local runner source: `cmd/fairyc-windows-host-smoke/`

## Context

ADR 0076 made Microsoft x64 classification executable as host-independent Go data, but a correct
table can still disagree with the C compiler that will actually carry values. The Windows machine
has local x64 MSVC and no WSL. A useful next gate must therefore be a bounded native Windows test,
not a pretend Linux proxy and not a second copy of the ABI rules encoded only in Go assertions.

## Decision

1. `internal/msvccprobe` owns one deterministic private C spelling and recursive type emitter for
   the closed Windows LLP64 fixture vocabulary. The layout and ABI probes share it so their C
   records, scalar spellings, size, alignment, and offset assertions cannot drift independently.
2. `internal/windowsabiprobe` consumes only the twelve pass cases exposed after strict execution of
   `manifest-windows-msvc-v0.json`. It replans every signature through `internal/cabi` and rejects
   any fixture/implementation disagreement before emitting C.
3. One caller translation unit initializes deterministic values, calls every dense target symbol,
   and checks returned values. One callee translation unit checks every received value and returns
   deterministic results. Fixture IDs and local paths do not enter generated C.
4. The driver writes `caller.c` and `callee.c` separately, then invokes local x64 MSVC once per low
   and optimized profile to compile, link, and execute the pair. The run must exit zero with no
   output. All temporary sources, objects, and executables are removed.
5. The configured Windows host smoke now proves 24 case/profile value crossings through two linked
   probe executions: twelve cases at low optimization and the same twelve at optimized settings.
6. This is a same-version MSVC C-to-C probe. It does not clear `bidirectional-c-abi-fixtures` because
   it lacks source-level Fairy callers or callees, an independent compiler or register/assembly
   oracle, exact stack and shadow-store observations, and a hosted Windows receipt. It also makes no
   PE/COFF, unwind, runtime, standard-library, distribution, conformance, or support claim.

## Consequences

- Regressions in the admitted scalar, mixed-slot, fifth-stack, small-record, indirect-record, and
  hidden-result value paths now fail on the actual Windows host under both compiler profiles.
- The probe uses positive deterministic scalar values. Narrow signed/unsigned extension boundaries,
  variadic floating mirroring, vector and intrinsic exceptions, retained callbacks, exports, and
  foreign unwinding still need separate fixtures.
- A later PE/COFF inspection gate can analyze the retained in-memory artifact bytes without turning
  temporary object files into product or conformance artifacts.

## External basis

- Microsoft x64 parameter, return, shadow-store, and stack rules:
  <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170>
- Microsoft x64 scalar sizes and alignment:
  <https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions?view=msvc-170>
