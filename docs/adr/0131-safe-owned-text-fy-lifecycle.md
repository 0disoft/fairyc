# ADR 0131: Add an example-level safe owned-text lifecycle

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`

## Context

The provisional C runtime already implemented Buffer-backed String and CString operations, exact
failure statuses, rollback, termination, and normalized drop. Earlier generated-Fairy probes crossed
private record layouts directly, but source programs still had no move-only wrapper that confined raw
pointers, mapped failures into `Result`, and ran cleanup from a resource drop body.

Treating Fairy resource storage as a C struct through an incompatible typed pointer would also rely
on C aliasing behavior that the runtime contract does not grant. A bridge must copy the flat storage
representation without claiming that generated Fairy storage has a C struct effective type.

## Decision

1. `examples/owned_text/owned_text.fy` owns an example-level safe module with public move-only
   `String` and `CString` resources plus a typed `TextError`. Raw records, foreign declarations, and
   opaque storage casts remain private to that module.
2. `copy`, `append`, and `cstring_from_text` return source `Result` values. `string_len`,
   `cstring_len`, and error accessors expose only scalar metadata. The example exposes no raw data
   pointer and no mutable bytes.
3. Five project-owned storage bridge symbols accept `void *` resource storage. They use `memcpy` to
   load and store the existing private C records, avoiding incompatible-struct pointer dereferences.
4. Constructors require canonical empty output storage. Failure preserves that state. Append leaves
   the prior String value owned and unchanged on failure. Successful drop writes canonical empty
   storage back before the source resource finishes cleanup.
5. The resource drop bodies call only the matching private storage bridge. Ordinary returns,
   `Result` branches, and implicit scope cleanup therefore share the compiler's verified nonempty
   drop-glue path.
6. A field read whose receiver is `&Resource` or `&mut Resource` retains the borrow type in MIR while
   resolving its field definition through the underlying resource type. The C backend emits pointer
   member access for that borrowed receiver and value member access for an owned receiver.
7. This slice does not create `std::text`, stable public layouts, a public ABI, safe `&str` or
   `&CStr` reconstruction from owned storage, Unicode algorithms, allocator selection, or a general
   collection API. Those require additional borrow-construction and package-surface decisions.

## Consequences

- A checked-in safe Fairy program can create, append, inspect lengths, reject interior NUL, and
  release owned text without exposing unsafe operations in its entry module.
- The storage bridge is deliberately tied to the example-private flat field order and the existing
  target-laid-out resource transport. Any field or layout change must update the bridge, tests, and
  this contract together.
- Exact copied bytes, terminator contents, rollback under allocation failure, invalid UTF-8,
  self-overlap, and malformed-state rejection remain owned by the independent native C probes.
- Safe borrowed views remain deferred instead of being faked as raw pointers or copied descriptors
  with unproven lifetime provenance.

## Verification

- compiler integration reads both checked-in example modules and requires verified MIR;
- MIR tests cover shared and mutable borrowed-resource field reads;
- C-backend tests require pointer member access for those borrowed receivers;
- standard-library tests require all five bridges, their closed symbol mapping, and byte-copy
  storage transport;
- the WSL-free Windows host smoke builds, inspects, and executes the safe example with local x64
  MSVC at low and optimized profiles, including normal, empty, interior-NUL, and implicit-drop paths.
