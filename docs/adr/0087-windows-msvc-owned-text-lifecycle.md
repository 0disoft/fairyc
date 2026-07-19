# ADR 0087: Windows MSVC owned-text lifecycle

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Component: `owned-text` requiring `system-heap`, `buffer`, and `text-view`

## Context

The Windows target path already executes SystemHeap, Buffer, Arena/Region, and allocation-free UTF-8
validation. String and CString are the first Windows component closure that combines allocation,
Buffer growth, UTF-8 validation, compiler-owned static storage, nested private records, pointer
views, and a small by-value result returned from a separately compiled C unit.

The current FY subset can compare view pointers and lengths but cannot dereference returned raw
storage. The Windows lane can therefore prove real success lifecycles and internal validation, but
it cannot act as an independent byte-for-byte oracle. Existing Linux C-only and generated-source
probes retain exact bytes, rollback, overlap, terminator, and negative-case responsibility.

## Decision

1. The Windows allowlist adds exactly `fy_std_cstr_validate_v0`, `fy_std_string_copy_v0`,
   `fy_std_string_as_view_v0`, `fy_std_string_append_v0`, `fy_std_string_drop_v0`,
   `fy_std_cstring_from_text_v0`, `fy_std_cstring_as_view_v0`, and
   `fy_std_cstring_drop_v0`.
2. Selecting any admitted owned-text symbol closes over SystemHeap, Buffer, and text view. Each
   component is emitted once in deterministic order; Arena remains independent even when the same
   checked-in program also selects it.
3. The FY example uses private 16-byte text/CStr views, 32-byte String/CString records nested over
   Buffer, and an 8-byte two-`u32` operation result. These layouts are closed transport, not public
   ABI or source-resource representation.
4. At low and optimized profiles, compiler-owned `"Fairy"` storage is copied into a String with
   logical length five, exposed through a non-null view, appended to logical length ten, exposed
   again, and dropped to zeroed metadata. The same input creates six-byte CString storage, exposes
   a non-null logical length-five CStr view, and drops to zeroed metadata. Compiler-owned
   `c"Fairy"` storage separately passes bounded CStr validation with logical length five. Every
   operation and allocation status must be zero.
5. This decision does not claim independent output-byte or terminator inspection, invalid UTF-8 or
   interior-NUL rollback, self-overlap behavior, empty CString, deterministic OOM, malformed-state
   rejection, safe String/CString resources, source `Result` mapping, stable layouts, hosted
   execution, conformance, or support.

## Consequences

- A Windows target PE now executes the real String, CString, and CStr C implementation rather than
  only allocation-free text validation.
- The target path exercises an 8-byte aggregate result crossing and a three-component dependency
  closure through both MSVC profiles.
- Linux File remains the only admitted Stage 0 standard-library component intentionally excluded
  from the Windows source contract.

## References

- `docs/adr/0044-owned-string-and-cstring-c-slice.md`
- `docs/adr/0060-cstr-validation-source-transport.md`
- `docs/adr/0062-owned-text-source-transport.md`
- `docs/adr/0064-used-symbol-standard-library-auto-link.md`
- `docs/adr/0085-windows-msvc-buffer-dependency-closure.md`
- `docs/runtime/runtime-stdlib-v0.md`
