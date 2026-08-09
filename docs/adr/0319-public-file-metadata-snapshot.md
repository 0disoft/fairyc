# ADR 0319: Public FileMetadata snapshot

- Status: Accepted
- Date: 2026-08-08
- SDK: `0.0.0-stage0.48`
- Depends on: ADR 0270, ADR 0277, ADR 0306, ADR 0307, ADR 0309, ADR 0317

## Decision

1. Add public `std::fs::FileMetadata` with `usize size`, `usize allocated_size`,
   `usize hard_link_count`, `FileKind kind`, and `bool read_only` in that order.
2. Add filesystem-effectful
   `std::fs::metadata(ref File) -> Result<FileMetadata, FileError>`. It observes the existing live
   handle sequentially through `size`, `allocated_size`, `hard_link_count`, `kind`, and
   `is_read_only`, returning the first error in that order. The result is not an atomic filesystem
   snapshot.
3. Extend constrained C emission so private structs and inert enums participate in one acyclic
   by-value dependency graph. Complete aggregate definitions are emitted topologically while prior
   root declaration order remains stable when no dependency requires movement. Enum-containing
   `@repr(C)` structs and, at this decision, enums nested inside enums remain closed. ADR 0323 later
   admits acyclic private inert-enum nesting without opening public C records.
4. Add no target C symbol and preserve language `fairyc-language/10` and runtime ABI
   `fairyc-runtime/0`. Advance the compiler-bundled SDK to `0.0.0-stage0.48` because the public
   metadata type and function are additive.

## Evidence

The 44th constrained C-emission fixture constructs an enum-containing private struct, reads its
fields, and switches on the contained enum while requiring the enum definition before the struct
definition. The FileMetadata quickstart uses the existing five raw metadata symbols and carries the
public aggregate through clean verified MIR and C emission. Low and optimized MSVC executions cover
the generated-Fairy metadata path while preserving cursor and logical size.

## Consequences

Callers can collect the common same-handle file facts with one public SDK call and no allocation,
new ABI symbol, cursor movement, or ownership transfer. Because the fields are sequential
observations, concurrent file changes may produce a mixed-time result. Atomic snapshots, timestamps,
ACLs, writable-access guarantees, Linux public linkage, enum payload nesting, and enum fields in
public C records remain separate gates.
