# ADR 0124: Connect a safe FY Windows File lifecycle

- Status: Accepted
- Date: 2026-07-21
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Ownership source: `docs/language/0001-values-and-resources.md`
- C boundary source: `docs/mir/c-backend-safety-v0.md`
- Amended by: `docs/adr/0125-mutable-fixed-u8-array-byte-view.md`

## Context

ADR 0122 established a target-bound Win32 File C component with consuming close and best-effort
drop. ADR 0123 added mutable by-value parameters and the opaque address boundary needed by checked
wrappers. The remaining gap was source-level proof: a Fairy program still could not construct a
move-only File, propagate filesystem effects, map raw status into a typed error, use a safe buffer,
or execute verified drop glue against the real handle component.

The first connection must not pretend that Stage 0 already has general mutable-slice operations, a
portable path type, general standard-library package discovery, arbitrary FFI linkage, or a stable
public resource layout.

## Decision

1. `examples/windows_file/windows_file.fy` owns an example-level checked module, not a stable
   `std::fs` package. It declares the exact internal `@repr(C)` result and open-option records and a
   move-only `File { handle: *mut void, open: u8 }` resource.
2. `File` carries `@effects(filesystem)`. Its nonempty drop body passes the resource place through
   explicit unsafe opaque-address conversion to `fy_std_windows_file_drop_v0`; the raw fallback
   normalizes first, attempts release once, and discards any error.
3. `create_truncate` and `open_read` accept `&CStr`, construct closed inline state, call only the
   closed allowlisted raw open symbol, and return `Result<File, FileError>`. `FileError` preserves
   raw status, Win32 code, and completed-byte count. A failed open consumes and drops its still-
   closed temporary so branch ownership remains identical.
4. `write_all`, `sync`, and the initial bounded `read_16` accept safe borrows. Existing resource or fixed-
   array borrow values may erase only to an opaque raw pointer inside the wrapper's unsafe block;
   generated C forwards the referent pointer instead of taking the address of the borrow carrier.
5. ADR 0125 supersedes that initial length-specific choice with `read(&mut [u8])`. A mutable fixed
   `[u8; N]` place contextually supplies the private data pointer and canonical length; safe mutable
   indexing, slicing, temporary borrowing, retained foreign pointers, and public ABI remain closed.
6. `close(mut file: File)` consumes the resource by value. The raw close normalizes before its one
   kernel attempt; ordinary generated cleanup then observes closed state and becomes a no-op on
   both raw success and failure.
7. The C runtime loads and stores opaque caller state with `memcpy` rather than dereferencing it
   through an incompatible struct type. This keeps the internal layout crossing bytewise and avoids
   strict-aliasing assumptions. The layout remains target-derived and private.
8. `examples/windows_file/main.fy` writes and synchronizes the nine `FairyFile` bytes, closes,
   reopens, reads them through checked fixed-array indexing, observes EOF, and closes again. A
   separate path is opened and allowed to leave scope without explicit close; a second write-open
   on that path must succeed, proving generated implicit drop released the non-shareable handle.
9. The Windows host smoke builds this two-module source at low and optimized MSVC profiles, inspects
   each object, linked PE, and unwind data, runs from its temporary build directory, requires no
   output, and accepts only process result 42.

## Consequences

- A developer can now write and execute a real `.fy` Windows File program through the current
  compiler and checked example module without WSL.
- Unsafe code stays inside the support module; the entry module uses typed `Result`, safe borrows,
  consuming ownership, effects, and checked indexing.
- The proof is local Windows x64 MSVC evidence, not hosted CI, Unicode-complete paths, other Windows
  architectures, MinGW compatibility, a stable `std::fs` API, general mutable-buffer operations, or
  self-hosting.

## Verification

- semantic, MIR-verifier mutation, and C-emission tests cover safe resource/fixed-array borrow
  transport, opaque erasure, CStr pointer forwarding, and forbidden mutability strengthening;
- C source tests require bytewise opaque-state load/store and normalization before close;
- the complete Go suite passes;
- the Windows host smoke compiles, inspects, links, and executes both FY lifecycle profiles plus
  both raw separate-translation-unit probe profiles without WSL;
- Linux-only Clang/GCC, glibc, ASan, and UBSan lanes are not claimed on this Windows host.
