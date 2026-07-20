# ADR 0122: Add a consuming-close Windows File C slice

- Status: Accepted
- Date: 2026-07-21
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Target source: `docs/product/02-spec.md`
- Ownership source: `docs/language/0001-values-and-resources.md`

## Context

The Windows target already links project-owned allocation, text, and console components, and the
language can carry a flat move-only resource with effect-bearing drop glue. It still has no
project-owned file-handle implementation to connect to that ownership model. Reusing the Linux
descriptor helper would be wrong: Windows owns a different handle representation, open contract,
error source, path boundary, and close operation.

The first slice must also remain usable from the current constrained C backend. It cannot rely on a
hidden heap allocation, a platform-neutral path claim, exceptions, thread-local error state after
return, or retrying a handle after a failed close.

## Decision

1. `internal/cstdlib/stage0_windows_file.c` owns one target-specific C11/MSVC component. Its raw
   state is `{ void *handle; uint8_t open; }`; the closed representation is exactly `{ NULL, 0 }`.
2. Open accepts a bounded NUL-terminated ASCII byte path and closed read/write/create/truncate
   options. It converts at most 4,095 non-NUL bytes into a stack-owned `WCHAR` buffer and calls
   `CreateFileW`. Empty, non-ASCII, unterminated, invalid-option, and already-open output states are
   distinct fail-closed outcomes. This is deliberately not a portable or complete Unicode path API.
3. Read performs at most one `ReadFile` call and reports the exact completed count, including zero
   at EOF. `write_all` repeatedly calls a private single-write helper, preserves the completed
   prefix on failure, and returns a no-progress status if a successful write completes zero bytes.
4. Sync maps to `FlushFileBuffers`. It is explicit; close does not imply durable storage.
5. Explicit close consumes ownership before calling `CloseHandle`. Both success and failure leave
   the state closed, and no caller or automatic drop may retry the handle. A repeated explicit
   close reports the closed status.
6. Automatic drop normalizes the state first, attempts `CloseHandle` at most once, discards its
   failure, allocates nothing, and emits no output.
7. The public raw result carries `{ status, os_code, count }`. Win32 error identity is captured at
   the failing operation and returned as `os_code`; successful and locally rejected operations use
   zero.
8. The closed standard-library resolver admits only open, read, write-all, sync, close, and drop for
   `x86_64-windows-msvc`. The single-write helper has internal linkage, and Linux resolution rejects
   every Windows File symbol.
9. A separate-translation-unit native probe exercises create/truncate, exact write-all, sync,
   consuming close, repeated close, three-byte partial reads, EOF, forced invalid-handle close
   failure, best-effort drop, and repeated drop at low and optimized MSVC profiles. Every object,
   linked PE, and unwind table crosses the existing structural inspectors before execution.

## Consequences

- Windows now has a real project-owned raw File component with an executable consuming-close
  contract and no hidden allocation.
- The component is target-specific and intentionally narrower than a future safe `std::fs` API.
- Exact Windows handle state can be stored inline in a future move-only FY `File` resource.
- Public FY construction, typed `FileError`, filesystem effect propagation through wrappers, safe
  byte-buffer access, and source-level implicit-drop execution remain separate work.
- The local probe is same-toolchain Windows evidence. It does not establish hosted CI, other Windows
  architectures, MinGW compatibility, portable paths, or self-hosting.

## Verification

- Go contract tests keep the admitted symbol set, target binding, dependency closure, and embedded
  source markers synchronized;
- the full Go suite protects the target-neutral resolver and compiler behavior;
- the Windows host smoke compiles, inspects, links, and executes both native probe profiles without
  WSL;
- Linux-only Clang/GCC, glibc, ASan, and UBSan lanes are not claimed on this Windows host.
