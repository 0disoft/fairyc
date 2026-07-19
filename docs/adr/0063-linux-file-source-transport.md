# Linux File Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- ABI source: `docs/abi/c-abi-v0.md`

## Context

The provisional Linux File C unit already owns bounded path validation, partial I/O, write-all
progress, synchronization, and consuming-close behavior. Its native probe is C-only, so it cannot
detect drift in the generated Fairy records, foreign signatures, struct-result return ABI, raw
pointer transport, or calls emitted from actual `.fy` source.

Integer wrapper shims would hide those boundaries. The source probe must call the production C
symbols directly while remaining honest that the records are raw internal transport rather than
the public safe File resource.

## Decision

1. The source probe declares exact `@repr(C)` records for the operation result, Linux File, and open
   options. The records retain exact `c::int` and `c::size_t` fields and use the existing target
   layout assertions.
2. A bounded C companion creates one unique temporary path, supplies the immutable `FairyFile`
   payload and three mutable three-byte read segments, and observes results and final bytes. It does
   not implement open, read, write, sync, or close.
3. The generated Fairy caller opens the prepared file for truncating writes, invokes the real
   `fy_std_linux_file_write_all_v0`, synchronizes it, and invokes the real consuming close.
4. The caller invokes close a second time and requires the closed status. It then reopens for reads,
   performs three partial three-byte reads, requires a fourth read to report EOF with count zero,
   closes again, and verifies the descriptor and open flag are normalized after both successful
   closes.
5. Every successful operation must return zero status, zero error number, and the exact count. The
   existing C-only probe retains direct partial-write, no-progress, invalid-option, path-validation,
   `EINTR`, and forced-`EBADF` close-failure coverage.
6. Only the five exercised major-suffixed open, read, write-all, sync, and close symbols join the
   closed internal standard-library allowlist. Direct write and best-effort drop remain closed to
   source callers.
7. Caller and callee are separate objects. Clang-caller/GCC-callee and the reverse each run at low
   and optimized profiles and must exit 42 without output.
8. `fairyc.native-c-report/v27` adds `linuxFileSourceCallExecutions`, expected to be four only after
   every compiler direction and profile succeeds.
9. This transport does not add a public safe File resource, source effects, typed File errors,
   automatic standard-library linkage, portable paths, or target conformance.

## Consequences

- The configured native lane can now catch generated-record, struct-result, symbol, count, EOF, and
  consuming-close drift between actual Fairy source and the real Linux File C unit.
- The source probe deliberately covers the successful lifecycle plus repeated close; the lower C
  probe remains the owner of syscall-failure and rollback cases.
- Hosted Linux execution is still required before the four configured executions become observed
  evidence.
