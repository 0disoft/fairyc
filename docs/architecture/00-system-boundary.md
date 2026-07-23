# System Boundary

- Status: Draft
- Product source: `docs/product/02-spec.md`
- Architecture decision: `docs/adr/0001-initial-architecture-boundaries.md`

## Repository-owned systems

The repository is planned to own:

- the Stage 0 `fairyc` compiler implemented in Go;
- the versioned Fairy C language specification and diagnostics contract;
- target-independent typed HIR and MIR definitions;
- analysis-only Flow CFG plus move, borrow-origin, region, drop, bounds, overflow, and effect analysis;
- a bounded MIR interpreter used as the first semantic oracle;
- reference and native backend adapters;
- the thin runtime, bounded Stage 0 standard library, target descriptions, and C interoperability layer;
- compile-pass, compile-fail, runtime, trap, differential, and ABI fixtures;
- deterministic Fairy Atlas semantic-map production;
- package, cache, and command contracts owned by the `fairyc` CLI.

## External systems consumed

- a supported Go toolchain for the Stage 0 compiler;
- a Windows Go host for the initial compiler development environment;
- WSL or Linux CI plus Clang, GCC, glibc, startup objects, and a C linker driver for the first planned `x86_64-linux-gnu` conformance target;
- QBE or another fast backend only after a measured adoption decision;
- LLVM tools only for the later production backend;
- platform SDKs and system libraries required by an explicitly supported target;
- Clang libraries or tools for future C-header import and ABI inspection.

Exact Go, Clang, GCC, glibc, sysroot, and runner baselines remain UNDECIDED. No target is currently supported; `x86_64-linux-gnu` is the first planned conformance target.

## Hard boundaries

- Go host types never define Fairy C target size, alignment, integer width, or ABI layout.
- Frontend analysis owns language semantics; backends translate explicit MIR and may not invent drop, trap, or evaluation behavior.
- The runtime does not hide general-purpose allocation or add a garbage collector.
- Compiler intrinsics, thin runtime helpers, and standard-library APIs have separate ownership.
- The standard library cannot broaden unsafe behavior without an explicit safe-wrapper contract.
- Stage 0 exposes no stable native Fairy ABI; only explicit C ABI v0 is external.
- Build cache and generated output are derived state, never specification authority.
- An immutable `SemanticSnapshot` is the only compiler-to-Atlas boundary. ADR 0203 makes the
  direction executable: only the Atlas/export layer, CLI orchestration, and Windows evidence harness
  may directly import Atlas/export packages; every new consumer fails closed pending review.
- Atlas facts come from analyzed compiler IR; free-text documentation remains separate untrusted metadata.

## Initial delivery boundary

Stage 0 ends at a command-line compiler, a MIR interpreter, a constrained C backend, an `x86_64-linux-gnu` conformant profile, a thin runtime, a bounded standard library, and executable semantic and ABI fixtures. A second native backend, stable native ABI, portable path layer, custom allocator, package registry, native IDE, debugger, self-hosted compiler, production optimizer, and public 1.0 compatibility promise are outside that boundary.

## Recovery boundary

The Go Stage 0 compiler remains able to rebuild the declared bootstrap subset. Backends are replaceable behind MIR. C output remains a readable reference transport for explicitly profiled targets even after a native backend exists.
