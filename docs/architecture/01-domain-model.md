# Domain Model

- Status: Draft
- Language source: `docs/product/02-spec.md`

## Language concepts

| Concept | Responsibility | Stage 0 restriction |
| --- | --- | --- |
| Inert value | Passive data with structural copyability | No resource or safe-borrow fields |
| Restricted enum | Tagged inert alternatives for errors and simple states | Payload fields are inert and copyable |
| Resource | Unique cleanup responsibility | Whole-root affine moves only |
| Shared borrow | Non-null read-only view | Local last-use loan or one permitted return origin |
| Mutable borrow | Non-null exclusive view | Whole-root overlap; no two-phase borrow |
| Raw pointer | Nullable low-level escape hatch | Dereference, arithmetic, and safe-wrapper proof are unsafe |
| Borrow origin | Hidden lifetime relation | `Static`, one parameter, one region parameter, local, or raw-unknown |
| Region | Compiler-known lexical allocation capability | Cannot be stored or returned; no `NeedsDrop` elements |
| Resource-aware arena | Arena with registered cleanup | Deferred as a separate abstraction |
| Option | Presence or absence | Inherits payload move and drop behavior |
| Result | Success or typed failure | Must-use; `?` only at explicit sites |
| Unsafe boundary | Searchable location where safe invariants may be assumed or broken | Carries source provenance into MIR |

Generics, interfaces, dynamic dispatch, captured closures, async, self-reference, user interior mutability, and cross-thread capability traits are later language concepts, not Stage 0 compiler obligations.

## Compiler concepts

| Concept | Responsibility |
| --- | --- |
| Source bytes and span | Original `.fy` bytes plus stable half-open byte ranges without host-path leakage |
| Trivia token stream | Preserved lexical, comment, invalid-input, and repair provenance |
| Compact lossless CST | Arena-backed structure accounting for every source byte and zero-width missing token |
| AST | Error-free semantic syntax lowered separately from CST recovery nodes |
| Resolver | Module, import, item, parameter, and lexical binding lookup with canonical declaration coordinates |
| Typed HIR | Resolved names, types, values, resources, borrows, regions, results, and effects |
| Flow CFG | Analysis-only graph for initialization, ownership, loan, origin, escape, and cleanup fixed points |
| MIR | Mutable-local backend-independent executable semantics |
| MIR interpreter | Small semantic oracle before a second native backend |
| Target description | Pointer width, endian, alignment, C widths, calling convention, and feature set |
| Layout engine | Target-derived size, alignment, field offset, and ABI classification |
| Backend | Translation from verified MIR to C or a later measured target |
| Diagnostic | Stable code, primary span, related spans, explanation, and repair direction |
| SemanticSnapshot | Immutable compiler-fact boundary consumed by Atlas |
| Semantic map | Versioned facts, uncertain edges, provenance, and completeness |
| TargetSpec | Semantic layout, calling, object, and representation facts | No local tool paths |
| ToolchainSpec | Compiler driver, sysroot, libc or CRT, startup, linker, and capabilities | Cannot redefine target semantics |
| Compatibility vector | Independent versions for source, MIR, ABI, runtime, diagnostics, packages, and maps | Compiler version is not universal compatibility |

## Identity model

Compiler arenas use typed integer IDs. These IDs are build-local unless a contract explicitly promotes them. Atlas omits node IDs, derives opaque symbol identities from versioned semantic coordinates, and reserves human-maintained anchors for a later schema.

## Invariants

- A resource root is `Live`, `Dead`, or `MaybeLive` at every Flow CFG point.
- Every live resource has exactly one cleanup obligation on a normal exit.
- Every source local has an initializer; partial construction is compiler-controlled.
- Every parser result consumes the complete source or reports a bounded failure; any lexer or parser diagnostic blocks HIR construction.
- Resolver, type, and Flow failures retain source-byte spans and do not expose build-local IDs as language contracts.
- Safe borrows are non-null and cannot outlive their one proven owner or region origin.
- Backends receive explicit evaluation, failure, and cleanup control flow.
- Target layout is never calculated from the host Go runtime or host C defaults.
- Public output order does not depend on Go map iteration or parallel scheduling.
- Atlas cannot feed authority or semantic decisions back into the frontend.
- `pub` changes source visibility only and does not create binary ABI stability.
