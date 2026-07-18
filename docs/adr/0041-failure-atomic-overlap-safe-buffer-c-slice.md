# Failure-Atomic Overlap-Safe Buffer C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation source: `internal/cstdlib/`

## Context

`Buffer` is the storage base for owned text and later I/O. Its dangerous case is not ordinary append;
it is appending an initialized slice borrowed from the same buffer when growth relocates storage.
Calling reserve first would invalidate the source pointer, while copying the full allocation would
read and propagate spare uninitialized capacity.

## Decision

1. The provisional Buffer record owns one SystemHeap block plus an initialized byte length. Its
   invariant is `length <= capacity`, with the heap block itself canonical.
2. Construction, reserve, append, zeroed resize, truncate, clear, and drop are internal C helpers.
   Safe access to spare capacity and raw-parts construction remain absent.
3. Reserve grows geometrically but does not expose the selected capacity as a stable promise. It
   allocates replacement storage, copies exactly the initialized prefix, and releases the old block
   only after allocation succeeds.
4. Append checks `length + count` before allocation. A size-overflow result leaves pointer, capacity,
   length, and initialized bytes unchanged.
5. An append source inside the initialized prefix is represented by an offset before reserve and
   rebound into replacement storage after reserve. A source inside spare capacity or extending past
   initialized bytes is rejected. The final copy uses `memmove` for overlap.
6. Growing resize initializes every newly visible byte to zero. Shrink, truncate, and clear do not
   expose or promise the contents of spare capacity.
7. The closed native probe covers empty construction, append, no-op reserve identity, relocating
   overlap append, overflow rollback, zeroed growth, truncate, clear, malformed-state rejection, and
   drop under Clang/GCC at low and optimized profiles.
8. `fairyc.native-c-report/v7` adds `bufferExecutions`, expected to be four after a successful hosted
   run. No hosted receipt has been observed.
9. The helper remains unavailable to Fairy C source and is not product-linked. `String`, `CString`,
   and file I/O therefore cannot claim Buffer-backed execution yet.

## Consequences

- Growth copies initialized bytes instead of using raw `realloc`; this favors semantic proof over an
  unmeasured fast path.
- The uintptr-based internal-source classification is target-transport code for
  `x86_64-linux-gnu`, not a portable language rule.
- Deterministic allocation failure injection and source-level borrow exclusion remain later fixture
  layers.
