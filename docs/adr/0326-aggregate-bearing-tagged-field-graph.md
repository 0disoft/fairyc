# ADR 0326: Aggregate-bearing tagged field graph

- Status: Accepted
- Date: 2026-08-10
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0319, ADR 0323, ADR 0325

## Decision

1. Admit private `Option<T>` and `Result<T, E>` fields in inert structs and restricted user enums
   when every payload is a canonical Fairy copy scalar, an admitted inert struct, or a restricted
   inert user enum and the complete dependency graph is acyclic.
2. Extend aggregate layout resolution to traverse tagged nodes and their aggregate payloads before
   computing each containing aggregate's target-owned size, alignment, and offsets.
3. Emit one combined topological graph containing structs, user enums, and aggregate-bearing tagged
   records. Every complete payload aggregate precedes its tag, and every tag precedes an aggregate
   that stores it by value.
4. Keep nested built-in tags, fixed-array-, borrow-, raw-pointer/C-scalar-, and resource-bearing
   tagged fields closed. Keep all built-in tagged fields out of `@repr(C)` records and reject cycles.
5. Keep language, SDK, runtime ABI, target schema, MIR, cache, lockfile, and public API identities
   unchanged. This closes a private backend gap for already-valid Stage 0 source.

## Evidence

- The 49th constrained C-emission fixture constructs `Inner` and `Failure`, stores
  `Option<Inner>`, `Result<Inner, Failure>`, and `Option<Failure>` across `Outer` and `Envelope`,
  reads and switches every layer, and returns 42.
- The C-backend unit regression requires `Inner` and `Failure` definitions before their tagged
  records and those tagged records before `Outer` and `Envelope` by-value fields.
- A Windows-amd64 Go regression invokes the real `experimental run-host` path at low and optimized
  profiles. Both generated-Fairy MSVC executables return 42 with empty stderr.

## Consequences

Ordinary private product models can compose named product types with `Option` and `Result` without
flattening their fields. The combined graph remains finite and compiler-private; no generated C
name or layout is a public ABI promise, and unsupported payload classes still fail closed.
