# ADR 0264: Publish the Windows Stage 0 file surface

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0122, ADR 0124, ADR 0147, ADR 0258

## Context

The Windows target already owns a closed C component for bounded path validation, partial reads,
write-all progress, synchronization, consuming close, and best-effort resource drop. ADR 0124 also
proved a move-only Fairy `File` wrapper with typed `Result` failures and `filesystem` effects under
low and optimized MSVC. That safe wrapper remains in `examples/windows_file`, however, so ordinary
workspace programs cannot import it from the compiler-bundled SDK. Programs either duplicate the
unsafe declarations or cannot perform native file I/O through a public standard-library surface.

The raw Windows path accepts only a nonempty NUL-terminated ASCII bootstrap path and converts at
most 4,095 bytes to a stack-owned UTF-16 buffer. Treating that target-specific boundary as a
portable or Unicode-complete path design would freeze a false promise. The first public slice must
therefore expose the already proved ownership and I/O behavior while retaining an explicit Windows
x86-64 MSVC maturity boundary.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.13` to `0.0.0-stage0.14` because it
   adds the public module `std::fs`. Language remains `fairyc-language/8` and runtime ABI remains
   `fairyc-runtime/0`.
2. `std::fs` exposes the move-only resource `File`, inert `FileError`, `create_truncate`,
   `open_read`, `write_all`, `read`, `sync`, consuming `close`, and the scalar accessors
   `error_status`, `error_os_code`, and `error_completed`. Every operation that can touch a file
   declares `filesystem`; the error accessors are effect-free.
3. `File` layout stays private. Its cleanup invokes the closed
   `fy_std_windows_file_drop_v0` bridge through an opaque mutable address. External safe code can
   obtain an owner only from a factory, borrow it only through the declared operations, move it by
   value, or consume it with `close`.
4. Explicit close consumes and normalizes the owner before one `CloseHandle` attempt. Both `.ok`
   and `.err` leave the resource closed, so automatic cleanup cannot retry a failed close. Drop is
   the silent one-attempt fallback for a still-live handle.
5. `create_truncate` and `open_read` accept `ref CStr`, but the Windows component continues to
   admit only nonempty ASCII paths terminated within its 4,096-byte scan bound. Empty, non-ASCII,
   overlong, unterminated, invalid-option, and nonempty-output states fail before ownership changes.
   This is a Windows bootstrap path contract, not the future portable path type.
6. `read` performs one bounded call and returns the exact completed count, including zero at EOF.
   `write_all` either completes the whole immutable byte view or returns the completed prefix.
   `sync` remains explicit. No implicit buffering, text decoding, line I/O, seek, metadata, remove,
   rename, directory, permission, or asynchronous API is added.

## Compatibility

The Fairy surface is source-additive, but exact SDK selection is intentionally not lock-compatible:
the SDK source digest, package instance, public-interface hash, dependent identities, generated
locks, and whole-package cache keys change. No C symbol is added or renamed; the public module uses
the six existing pre-1 Windows target-support symbols. The module can be checked target-neutrally,
but native linkage is currently admitted only for `x86_64-windows-msvc`. Linux File has a separate
raw layout and path/options contract and is not silently claimed as the same public implementation.

## Evidence

- SDK and workspace-loader tests require exactly six embedded sources including `std/fs.fy`, module
  `std::fs`, SDK version `0.0.0-stage0.14`, and changed source identity.
- Compiler integration tests compile the checked-in quickstart through package loading, effects,
  Resource Flow, verified MIR, and entry-reachable Windows C emission. The used program observes
  exactly the six sorted `fy_std_windows_file_*_v0` foreign symbols; an unused program emits none.
- Effect-negative integration proves a caller without `@effects(filesystem)` receives `E7102` with
  the effect declaration as its related source.
- `examples/std_fs_quickstart` executes empty-path `.err` status/accessors, create/truncate,
  write-all, sync, consuming close, reopen, one bounded read, EOF, and implicit drop followed by
  successful reopen. Development and optimized MSVC builds each return 42 after PE32+ and x64
  unwind inspection.
- The complete Windows host smoke passed with 23 package-workspace builds and 89 inspected PE32+
  target or probe images. The independent raw component probe continues to execute partial I/O,
  forced close failure, repeated close, and repeated drop at both profiles.

## Consequences

Ordinary Windows Stage 0 programs can perform safe owned file I/O without declaring raw pointers,
foreign symbols, or resource layout. Portable Unicode paths, Linux public linkage, seek, metadata,
directory traversal, file deletion, general dynamic mutable buffers, retained foreign pointers,
public resource layout, and a supported Windows target or complete standard library remain separate
work.
