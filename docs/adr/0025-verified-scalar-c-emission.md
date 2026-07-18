# ADR 0025: Verified scalar C emission

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- MIR sources: `docs/mir/mir-v0.md` and `docs/mir/c-backend-safety-v0.md`
- Target source: `docs/abi/c-abi-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`

## Context

The Go Stage 0 compiler and MIR interpreter could execute a bounded Fairy C subset, but there was
no backend handoff beyond verified MIR. Calling that interpreted execution a native Fairy C build
would be false. The first backend step must preserve the interpreter's scalar semantics while
remaining honest about the missing C parser, compiler driver, runtime, linker, and Linux execution
lane.

C is also a hostile semantic transport if the emitter copies MIR operations into convenient C
expressions. Signed overflow, invalid shifts, division edge cases, implementation-defined signed
right shift, C identifier injection, and host-dependent widths can silently change the program.
The first slice therefore needs a deliberately closed capability boundary rather than a general
"best effort" C printer.

## Decision

1. `internal/cbackend` accepts only MIR that passes `mir.Verify` and additionally rejects malformed
   C-profile shapes such as non-canonical block identifiers, missing scalar destinations, invalid
   operands, and out-of-range successors.
2. The only initial profile identity is the provisional `x86_64-linux-gnu` target with an
   extension-free ISO C11 dialect. This identity selects an emission contract; it does not make the
   target emittable, conformant, or supported.
3. The accepted value subset is `i32` and `bool` parameters, `i32`, `bool`, or `void` internal
   results, and one zero-argument `main() -> i32` entry. MIR stores parameter types aligned with
   parameter locals so the backend never guesses a signature from use sites.
4. Generated function symbols are private numeric identifiers assigned after sorting canonical
   Fairy C function names. User names cannot become C identifiers, linker symbols, or ABI promises.
5. Ordinary checked `i32` arithmetic uses defined widened checks before the C operation. Division
   and remainder reject zero and signed-minimum divided by negative one. Right-shift amounts are
   checked, and arithmetic signed right shift plus bitwise results use unsigned operations and
   `memcpy` bit preservation rather than implementation-defined signed conversions.
6. Canonical trap categories have fixed `uint32_t` codes and are passed to the non-returning internal
   `fy_runtime_trap` symbol. The emission layer does not choose a product termination mechanism;
   external observation belongs to the native harness and runtime boundary.
7. Emission is deterministic: functions, features, declarations, blocks, and helper categories have
   stable ordering; generated text uses LF and contains no absolute source path. Complete C text,
   private identifiers, block labels, and formatting are not compatibility contracts.
8. The initial slice rejects resources, tagged values, Region operations, safe borrows, unsafe
   operations, FFI, runtime and standard-library calls, target layout, and C ABI lowering. ADR 0029
   later admits only private `Option<i32>` and `Result<i32, i32>` transport while every broader
   tagged and resource case remains fail-closed at that decision point. ADR 0033 later admits only
   direct nonempty all-`i32` resources with empty drop bodies.
9. `internal/cbackendfixture` runs source through syntax, semantics, Flow, verified MIR, the MIR
   interpreter, and C emission twice. Its four initial cases prove scalar control and calls,
   bitwise bit preservation, checked division and shift, overflow trapping, target identity, feature
   identity, and byte-stable emission. They do not parse, compile, assemble, link, or execute the
   generated C. ADR 0029 expands the same v0 manifest to eight cases with scalar Option and Result;
   ADR 0033 expands it to eleven with flat-resource transport.

## Consequences

- Fairy C now has a real first backend boundary without pretending that a native program ran.
- The Go compiler remains the bootstrap and reference implementation. No self-hosting claim follows
  from generated C.
- The next evidence gate is the Linux toolchain lane in ADR 0026, which compiles and links the
  emitted translation unit and compares its process outcome with the MIR interpreter.
- The target status cannot advance until the broader target, runtime, layout, ABI, and maintained-CI
  gates in `docs/abi/c-abi-v0.md` pass.

## Rejected alternatives

- Transpile source syntax directly to C: duplicates semantic lowering and cleanup rules outside MIR.
- Emit unchecked signed C arithmetic: makes safe-code meaning depend on undefined behavior and
  optimization level.
- Reuse Fairy C names as C symbols: leaks source spelling into a native ABI that Stage 0 explicitly
  does not stabilize.
- Call the target emittable after text generation: confuses a narrow scalar capability with a
  complete target profile and skips the compiler, runtime, linker, and execution evidence.
