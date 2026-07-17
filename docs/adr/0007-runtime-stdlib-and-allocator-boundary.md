# Runtime, Standard Library, and Allocator Boundary

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Stage 0 needs traps, regions, bytes, text, files, and safe C wrappers to prove the language, but a generic allocator protocol, generic vector, formatting framework, or large runtime would introduce deferred generics and stored lifetime state through the back door.

## Decision

1. Compiler intrinsics own checked operations, bounds and traps, layout queries, built-in tagged values, move and drop glue, borrow markers, raw memory operations, slices, and region markers.
2. The runtime owns only trap dispatch, process entry, an empty-storage sentinel, and bounded memory helpers that generated C cannot express safely and portably within the selected profile.
3. The standard library owns `SystemHeap`, `Arena`, `Buffer`, UTF-8 validation, `String`, `CStr`, `CString`, `File`, and raw C wrapper helpers.
4. `SystemHeap` and lexical `Arena` or `Region` are the only Stage 0 allocation strategies. There is no user allocator interface or hidden default allocator.
5. Heap allocation is fallible. Zero-size allocation is normalized without calling the platform allocator. Public raw `realloc` is excluded.
6. `Buffer` is the only growable owned collection. `Vec<T>` arrives with user generics.
7. `str` and `CStr` are compiler-known unsized view types used behind safe borrows; they are not ordinary raw-pointer structs. `String` and `CString` are owned resources.
8. The initial `File` and path contract is Linux-specific and accepts `&CStr`. Portable `Path` is deferred.
9. A Linux `File.close(self)` consumes ownership on every reported outcome and is never automatically retried. Durability-sensitive code calls `sync` before `close`.
10. Retained foreign pointers require owned storage and a wrapper whose unregister contract is synchronous and infallible for memory safety. Otherwise the API remains unsafe.

## Rationale

This boundary proves resource cleanup and C interoperability without requiring generic protocols or storing borrowed allocator contexts. Explicit allocation failure and partial I/O prevent convenience APIs from silently changing ownership or error semantics.

## Consequences

- Stage 0 is a hosted Linux desktop or server prototype, not an embedded or freestanding claim.
- Byte and string APIs are less broad than mature standard libraries.
- Composite resources may use infallible reverse-order drop, but a general fallible composite-close protocol is deferred.
- Unsafe wrappers remain the soundness boundary for foreign code that retains pointers.

## Revisit conditions

Add custom allocation, generic collections, portable paths, sockets, or asynchronous I/O only after the non-generic core and first target fixtures are stable and the new ownership contract is written first.
