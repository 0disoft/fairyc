# ADR 0324: Scalar tagged fields in private aggregates

- Status: Accepted
- Date: 2026-08-10
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0050, ADR 0280, ADR 0323

## Decision

1. Admit the existing compiler-private `Option<i32>` and `Result<i32, i32>` representations as
   by-value fields of private inert structs and restricted inert user enums in constrained C.
2. Reuse the established `{ u32 tag; i32 payload; }` target-derived layout and emit its private C
   typedef before every aggregate definition that embeds it.
3. Preserve construction, by-value aggregate transport, field reads, payload extraction, and
   exhaustive switching through verified MIR and generated C.
4. Keep these tagged fields out of `@repr(C)` records. General tagged aggregate payloads,
   resource-bearing tags, recursive aggregate dependencies, and public tagged ABI remain closed.
5. Keep language, SDK, runtime ABI, target schema, MIR, cache, lockfile, and public API identities
   unchanged. This closes a private backend gap for already-valid Stage 0 source.

## Evidence

- The 47th constrained C-emission fixture embeds both scalar tags in a private struct, embeds that
  struct and another scalar option in a user enum, then constructs, reads, switches, and returns 42.
- The C-backend unit regression requires the scalar tagged typedefs to precede their first by-value
  aggregate field and checks the interpreter result and feature ledger.
- A Windows-amd64 Go regression invokes the real `experimental run-host` path at low and optimized
  profiles. Both generated-Fairy MSVC executables return 42 with empty stderr.

## Consequences

Private data models can compose the smallest built-in optional and fallible scalar values without
flattening tags into parallel fields. The representation remains compiler-private and may change
with a future backend revision; no C consumer may depend on its spelling or layout.
