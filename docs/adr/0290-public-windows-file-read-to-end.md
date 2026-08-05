# ADR 0290: Read a Windows file through EOF into an owned byte buffer

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0246, ADR 0264, ADR 0288

## Context

Public `std::fs::read_append` safely performs one bounded read into owned Buffer spare capacity.
Callers that need the rest of a file must still choose a growth bound, repeat the operation, track
the newly appended count, and stop only after an explicit zero-byte EOF read. That policy is common
enough to deserve one source-level helper, provided it remains bounded per allocation and preserves
the existing File and Buffer failure boundaries.

The helper can be expressed entirely in compiler-bundled Fairy source over `read_append`. It does
not need a new target symbol, raw record, native ABI, retained pointer, file-size query, or implicit
seek.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.27` to `0.0.0-stage0.28`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public
   `std::fs::read_to_end(ref mut File, ref mut bytes::Buffer) ->
   Result<usize, ReadAppendError>` with the existing `filesystem` effect. It starts at the File's
   current cursor, appends every byte through EOF, and returns only the number newly appended.
3. Implement the operation in Fairy source by repeatedly calling `read_append` with bounded
   65,536-byte growth windows. A short successful read reuses the unfilled part of its current
   window before another allocation is attempted. EOF is the final successful zero-byte read.
4. Preserve the Buffer's preexisting initialized prefix. Each completed read remains committed if
   a later file or allocation failure occurs. Allocation for a new window precedes its corresponding
   file read, so a failed second-window allocation consumes no byte beyond the already committed
   prefix and leaves the File cursor at that prefix boundary.
5. Reuse `ReadAppendError` and its exact file, Windows-code, and allocation-status accessors. Add no
   new C symbol, target component, raw layout, runtime ABI, implicit file-size trust, seek, timeout,
   cancellation, asynchronous I/O, or transactional rollback.
6. Prove generated-Fairy success from a nonzero cursor into a nonempty Buffer under low and
   optimized MSVC. Separately force the second growth allocation to fail after a 65,536-byte
   committed prefix and require exact statuses `0/0/1`, cursor and Buffer length 65,536, preserved
   boundary bytes, consuming close, automatic Buffer drop, and zero retained probe allocations at
   both profiles.

## Consequences

- Safe callers can consume the remaining bytes of an ordinary file without duplicating a growth
  loop or exposing uninitialized Buffer capacity.
- The operation is intentionally eager and unbounded in total by the API; callers that require an
  application limit, incremental processing, cancellation, or backpressure retain `read_append`.
- Files that grow continuously may keep the operation running and allocating until EOF, failure,
  or address-space exhaustion. Concurrent file mutation remains target-defined external behavior.
- The source-additive SDK change updates source digest, public-interface hash, package instance,
  generated locks, dependent identities, and whole-package cache keys.
- Portable paths, Linux public linkage, vectored I/O, and a cross-target file contract remain
  separate decisions.

## Evidence

- `internal/sdkstdlib/sources/std/fs.fy` owns the public source helper.
- `examples/std_fs_quickstart/src/main.fy` executes current-position append success through
  generated Fairy.
- `cmd/fairyc-windows-host-smoke/read_to_end_failure_probe.go` executes the forced second-window
  allocation failure and cleanup contract.
