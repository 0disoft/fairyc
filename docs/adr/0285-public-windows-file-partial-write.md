# ADR 0285: Expose one-call partial Windows file writes

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0122, ADR 0264, ADR 0276

## Context

The Windows File component already used one bounded `WriteFile` helper underneath public
`std::fs::write_all`, but the helper remained component-private. Safe callers could therefore ask
only for complete output, unlike the public one-call `read` operation and the minimum File contract
that distinguishes partial I/O from retry-to-completion convenience.

Exposing the existing target helper directly is useful for bounded streaming, device, and pipe
callers that must control retry, cancellation, framing, or fairness themselves. It requires no new
allocation, ownership transition, retained pointer, buffering, or language feature.

## Decision

1. Add `std::fs::write(ref mut File, ref u8[]) -> Result<usize, FileError>` to the compiler-bundled
   SDK. The operation carries the existing `filesystem` effect.
2. Perform at most one synchronous `WriteFile` call, capped to one Windows `DWORD` request. Success
   returns the exact completed count, including a possible short count. Empty input on an open File
   succeeds with zero without calling the OS.
3. A closed File, invalid nonempty buffer, or failed OS call returns the existing typed `FileError`.
   This one-call v0 adapter reports completed zero for a failed `WriteFile`; successful short writes
   remain ordinary `.ok(count)` results for the caller to handle.
4. Retain `write_all` as the explicit retry-to-completion operation. It continues to call the same
   primitive, accumulate prior successful progress, and reject a successful zero-count write as
   no progress. Neither operation is transactional or record-framed.
5. Admit `fy_std_windows_file_write_v0` in the closed Windows component symbol contract. The symbol
   and its raw record remain target-private pre-1 implementation compatibility data, not a stable
   Fairy or foreign ABI.
6. Prove the raw primitive followed by `write_all` in the separate C probe. Prove the public safe
   operation in generated Fairy by writing a five-byte prefix once, completing the four-byte suffix,
   reading the exact nine-byte result, and observing a read-only-handle error with completed zero
   under low and optimized MSVC profiles.
7. Move the SDK from `0.0.0-stage0.25` to `0.0.0-stage0.26`. Keep language revision
   `fairyc-language/9` and runtime ABI `fairyc-runtime/0` unchanged.

## Consequences

- Callers can own bounded retry and backpressure policy without dropping to unsafe raw symbols.
- Existing callers remain source-compatible; the public SDK surface, source digest, interface hash,
  package instance, dependent locks, and cache identities change.
- Append handles retain their existing rule: every individual underlying write starts at the
  then-current EOF even after an explicit seek.
- Overlapped I/O, cancellation, timeouts, vectored writes, concurrent-writer sharing, buffering,
  record framing, portable paths, and Linux public linkage remain outside this slice.

## Evidence

- `internal/sdkstdlib/sources/std/fs.fy` owns the safe source contract.
- `internal/cstdlib/stage0_windows_file.c` owns the one-call target adapter.
- `internal/cstdlibcontract/contract.go` owns target symbol admission and component selection.
- `internal/cstdlib/stage0_windows_file_probe.c` owns the raw direct-call observation.
- `examples/std_fs_quickstart/src/main.fy` owns generated-Fairy success, failure, and final-byte
  observations through real Windows MSVC images.
