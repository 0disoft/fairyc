# ADR 0331: Target-neutral public File core

- Status: Accepted
- Date: 2026-08-15
- Depends on: ADR 0045, ADR 0063, ADR 0064, ADR 0264

## Context

The compiler-bundled `std::fs::File` API had safe ownership, typed errors, effects, and extensive
Windows execution evidence, while Linux product linkage exposed only private raw records. Source
using the common open, byte-I/O, sync, and close operations therefore compiled for Windows but
selected Windows-named symbols and failed before a Linux product build could link it.

## Decision

SDK `0.0.0-stage0.49` changes the private File storage field from a raw pointer to one pointer-width
`usize` token plus the existing open byte. The representation remains private and is accessed by C
through bytewise load/store. It does not expose a pointer, descriptor, HANDLE, public layout, or
portable serialized value.

The SDK's six open functions and public `read`, `write`, `write_all`, `sync`, consuming `close`, and
automatic drop call seven target-neutral private symbols. The closed source resolver maps those
symbols to one Linux File component or one Windows File component. The Windows adapter delegates to
the existing implementation without changing Win32 behavior.

The Linux adapter:

- accepts one nonempty NUL-terminated `CStr` whose terminator occurs within 4096 storage bytes;
- validates the same six one-byte open options as the Windows adapter;
- maps read/write, create, truncate, exclusive-create, and append to `open(2)` flags;
- creates with mode `0666`, still restricted by the process umask;
- stores `descriptor + 1` in the private token so zero remains the canonical closed value;
- preserves partial counts, retries interrupted open/read/write/sync calls, and consumes ownership
  before exactly one close attempt.

Ordinary POSIX path lookup is used. This slice does not claim directory containment, no-follow
resolution, race-resistant authorization, Unicode normalization, path mutation, metadata, seek,
exact-read, Buffer growth, or Windows-only path APIs on Linux.

## Evidence

Host-independent tests require the seven names to select exactly one target-owned component, require
both C adapters in the embedded source closure, and compile the public generated-Fairy program
through verified MIR and C emission. Native report v33 adds four `linuxPublicFileExecutions`: the
ordinary product build must compile with Clang and GCC at low and optimized profiles, execute in an
isolated temporary working directory, create and sync `Fairy C`, reopen it, read the exact seven
bytes, consume both owners, and return 42 with no output. Existing low and optimized Windows smoke
continues to execute the larger public `std::fs` example through the same target-neutral core names.

## Compatibility

The SDK advances to `0.0.0-stage0.49`, and its source digest and dependent package/cache identities
change. Language `fairyc-language/10`, runtime ABI `fairyc-runtime/0`, target identities, MIR,
lockfile schema, public function signatures, error records, and public ABI policy remain unchanged.
