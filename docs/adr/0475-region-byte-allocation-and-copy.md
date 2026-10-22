# ADR 0475: Region byte allocation and copy

- Status: Accepted; locally implemented; Linux hosted verification pending
- Date: 2026-10-22
- Depends on: ADR 0042, ADR 0251, ADR 0252
- Contract: `docs/runtime/runtime-stdlib-v0.md`

## Context

The lexical `Region` surface already declares zeroed-byte allocation and byte copying, and the
provisional C Arena already provides both operations. The source-level result types, failure
behavior, borrow origin, and MIR representation were not stated precisely enough to implement them
consistently.

`Region.place(T)` already uses a fallible tagged result and a `region-place` operation. Reusing
that transport keeps the three Region allocation forms on one verified path.

## Decision

1. A lexical Region exposes these fallible byte operations:

   ```text
   alloc_zeroed_bytes(usize) -> Result<ref mut u8[], std::alloc::AllocError>
   copy_bytes(ref u8[]) -> Result<ref mut u8[], std::alloc::AllocError>
   ```

   They implement the operations already declared by the Region contract. No new source syntax or
   user-facing type is introduced.

2. Each operation evaluates its argument exactly once. `alloc_zeroed_bytes` returns a fresh,
   independently owned allocation whose bytes are all initialized to zero. `copy_bytes` returns a
   fresh, independently owned allocation initialized with the exact source bytes. The `.ok` mutable
   view has Region provenance and its lifetime is the Region, not the input view. The `.err` payload
   contains only `AllocError` and carries no borrow or view.

3. An empty size or source length succeeds with a zero-length view. It performs no platform
   allocation, and the returned pointer identity is unspecified. The zero-length view retains the
   Region origin.

4. Allocation failure preserves every existing Region allocation and every source byte. It
   publishes no view or success borrow, and a temporary allocation or tracking node is released
   before returning the error. Failure remains ordinary `Result` data rather than a trap.

5. Lexical reset releases every successful byte allocation made in that Region. The bytes are inert;
   reset visits no elements and runs no resource cleanup. Neither operation admits resource
   placement or resource-aware byte storage.

6. Compiler MIR reuses the existing `region-place` operation instead of adding another operation
   kind. Its `Operator` selects `alloc_zeroed_bytes` or `copy_bytes`. `OwnerType` is the argument
   type, namely `usize` or `ref u8[]`, and `Type` retains the complete tagged Result type. An empty
   `Operator` retains the legacy `place` meaning and its existing owner-type rule. The verifier must
   reject an operand, owner type, operator, or result type that does not match the selected
   operation.

7. The bounded interpreter models the abstract success case only. It returns a zeroed byte view or a
   copied byte view from bounded per-test memory and does not claim to exercise native allocator
   failure.

8. The constrained C backend maps the operations to the existing
   `fy_std_region_allocate_zeroed_v0` and `fy_std_region_copy_bytes_v0` helpers with alignment
   `_Alignof(u8)`, which is one. It constructs `.ok` only after the helper succeeds and constructs
   `.err` from a nonzero status. The existing helper behavior supplies atomic storage and tracking-
   node allocation, source preservation, and empty normalization.

9. This implements an operation already declared under language `/20`; the language revision does
   not change. The runtime ABI remains `/0`, the embedded SDK source bytes remain unchanged, and no
   target maturity flag is promoted. Rebuilding the compiler changes its executable bytes for
   persistent-cache identity under ADR 0165, so the new lowering cannot collide with an older cache
   artifact.

## Consequences

- No C runtime ABI, SDK source bundle, public C header, grammar, or package artifact schema changes.
- Existing source-level `place` behavior and generated C remain unchanged unless byte allocation is
  used.
- A fresh Linux runtime and hosted receipt are still required. The latest hosted v38 receipt predates
  these Region byte cases and does not establish them for the current source.
- Related Go/fixture suites and vet pass locally. The full repository Go suite was not rerun.

## Implementation evidence

The contract is committed at `ca44226b`, and zeroed byte allocation is committed at `e035e79c`.
Byte-copy lowering is committed at `0ed33081`. Source recognition and typing are in
[`region_types.go`](../../internal/semantic/region_types.go); MIR lowering and verification are in
[`lower_regions.go`](../../internal/mir/lower_regions.go) and
[`verify_operations.go`](../../internal/mir/verify_operations.go); C emission is in
[`region_bytes.go`](../../internal/cbackend/region_bytes.go).

Targeted `workspacecompiler` tests pass for both operations. They cover one-time argument
evaluation, zeroed and copied bytes, subviews followed by independent writes, and using a copy after its source local leaves scope. They also check `E3101` Region escape, `E2101` exclusive-alias
rejection, and `E1006` wrong-input rejection for both names. See
[`region_copy_test.go`](../../internal/workspacecompiler/region_copy_test.go) and
[`region_bytes_test.go`](../../internal/workspacecompiler/region_bytes_test.go). The interpreter
path is [`region_bytes.go`](../../internal/interpreter/region_bytes.go).

`TestSDKStorageProbe` passed Linux C emission on Windows and low/optimized MSVC execution. The
fixture forces both payload and tracking-node allocation failure for each new operation, checks that
empty requests consume no allocation, and verifies recovery, prior-data preservation, and zero live
allocations. See [`sdk_storage_source.go`](../../internal/cbackendnative/sdk_storage_source.go) and
[`sdk_storage_test.go`](../../internal/cbackendnative/sdk_storage_test.go).

`TestRegionByteCopyPersistentWorkspace` passed a cross-session disk-cache hit and execution. It
executes all three Region modes before and after disk-cache restoration. These are focused local
results, not a replacement for Linux runtime execution or hosted CI.
