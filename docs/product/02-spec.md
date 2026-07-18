# Product Specification

- Status: Draft
- Owner: UNASSIGNED
- Related ADRs: `docs/adr/0001-initial-architecture-boundaries.md` through `docs/adr/0022-controlled-interpreter-io-harness.md`

This file owns the intended Fairy C language and toolchain behavior at the product level. The detailed Stage 0 contracts under `docs/language/`, `docs/mir/`, `docs/abi/`, `docs/runtime/`, `docs/compatibility/`, `docs/diagnostics/`, and `docs/atlas/` refine this file and cannot silently broaden it. Their v0 forms are prototype contracts rather than released compatibility promises.

## Goals

Fairy C is a native systems language with explicit costs, no garbage collector, a small runtime surface, and deliberate C interoperability. Safe code should prevent or define the most expensive routine failures without requiring a general lifetime algebra.

The implementation should optimize for understandable rules, deterministic diagnostics, fast development builds, bounded build storage, and backend-independent semantics. Stage 0 favors a small rule set that can be explained locally over a more expressive feature that needs hidden lifetime or cleanup machinery.

## Stage 0 semantic boundary

Stage 0 uses four compiler-known type properties: copyability, whether a value needs drop, whether it contains a safe borrow, and the origin of that borrow. The source language does not expose lifetime parameters.

The following restrictions are part of the prototype contract:

- source-level locals are initialized by their declarations; `MaybeUninit`, unions, and general uninitialized storage are deferred;
- resources use whole-value affine ownership, and moving an individual resource field is rejected;
- safe borrows may be used locally or returned from exactly one statically known parameter or region origin;
- named aggregates cannot store safe borrows, and inert structs cannot contain resource fields;
- borrow-returning functions cannot be converted to function values, interface methods, or foreign callbacks;
- self-referential aggregates, user-defined interior mutability, captured closures, async or coroutine state, and thread capability traits are not Stage 0 features;
- a plain region cannot own values that need drop;
- generics, interfaces, dynamic dispatch, metaprogramming, and user macros begin only after the first executable semantic slice proves the non-generic core.

The detailed rules are in `docs/language/0001-values-and-resources.md`, `docs/language/0002-borrows-and-return-origins.md`, and `docs/language/0003-regions-and-arenas.md`.

Name resolution, canonical type selection, phase ownership, cascade suppression, and semantic fixture assertions are defined in `docs/language/0007-name-resolution-and-typing-v0.md` and `spec/fixtures/semantic/manifest-v0.json`.

## Stage 0 surface

- Source files use `.fy`, source is UTF-8, and Stage 0 identifiers are ASCII.
- Declarations use `name: Type`; C declarators, C casts, preprocessing, and textual includes are absent.
- Statements use explicit semicolons. Assignment is a statement, and integer values are not implicit booleans.
- Every file declares one module. Imports require aliases and cannot be wildcard, selective, cyclic, or re-exported.
- `let` prevents reassignment, `var` permits reassignment and mutable borrowing, and both require initializers.
- A by-value position consumes a resource. There is no general `move` keyword.
- Functions declare return types. Methods live in `impl` blocks and make shared, mutable, or consuming receivers explicit.
- `if` is an expression. `match` is limited to shallow variants and inert fields.
- `discard` explicitly ignores a must-use value and drops its active payload.
- Unsafe operations and C declarations use `unsafe`, `unsafe extern C fn`, `@repr(C)`, and `@link_name`.
- Decimal, hexadecimal, binary, floating, UTF-8 string, byte-string, C-string, boolean, and raw-pointer null literals use the exact spellings and escape rules in the parser contract.
- Expressions follow one fixed Pratt binding-power table; comparisons are non-associative and assignment is never an expression.
- File identity maps source-root-relative `path/name.fy` directly to `module path::name;` with no special root filename.
- Parser recovery preserves byte spans and error nodes but never permits a recovered file to reach HIR.

The explanatory surface is in `docs/language/0005-surface-syntax-v0.md`; exact spelling is owned by `spec/grammar/stage0.ebnf`, and lexing, precedence, module identity, and recovery are defined in `docs/language/0006-lexing-parsing-and-recovery-v0.md`.

## Value categories

### Inert values

`struct` represents passive data. It has no implicit constructor or destructor. A struct is copyable only when all fields are copyable. It cannot contain a resource or a safe borrow in Stage 0. Hidden heap allocation and user-defined implicit conversion are not allowed.

A restricted user `enum` may contain only inert copyable payload fields. Built-in `Option` and `Result` may contain resources because the compiler owns active-variant move and drop behavior. General resource-bearing enums are deferred.

### Resources

`resource` represents a value that owns cleanup responsibility.

- Resources are move-only by default.
- Assignment, return, or by-value argument passing transfers the complete value.
- A moved value cannot be used again.
- Scope exit runs `drop` exactly once for every live resource on a normal control-flow exit.
- Resource fields drop in reverse initialization order.
- `drop` cannot return an error and cannot be invoked directly by safe source code.
- A fallible `close`-style operation consumes the resource or returns ownership explicitly inside its failure value.
- Resource partial moves, implicit forgetting, and source-level uninitialized resource slots are rejected.

Construction is performed by ordinary explicit functions such as `File.open`; there is no implicit constructor call. Lifecycle details are defined in `docs/language/0001-values-and-resources.md` and `docs/mir/drop-elaboration-v0.md`.

Whole-root `Live`, `Dead`, and rejected `MaybeLive` transitions, live replacement, dead-slot reinstallation, and cleanup-obligation assertions are versioned in `spec/fixtures/resource/manifest-v0.json`.

### Borrows and raw pointers

- `&T` is a non-null shared read-only borrow.
- `&mut T` is a non-null exclusive mutable borrow.
- `*T` is a raw pointer. Null values, pointer arithmetic, arbitrary casts, and dereference require an `unsafe` boundary.
- Nullable borrows use `Option<&T>` rather than a nullable reference representation.

The Stage 0 borrow model uses local last-use loans and whole-root overlap. It does not infer disjoint mutable field borrows or two-phase borrows. A returned safe borrow must have one unambiguous origin. Stored borrows, self-reference, and multiple-origin returns must use an index or generation handle, a redesigned API, a lexical region, or a narrow unsafe module.

### Regions and arenas

`Region` is a compiler-known lexical capability, not an ordinary mutable arena object. It cannot be stored or returned. Allocations derived from it carry the region origin, and a derived safe borrow cannot outlive the lexical region.

An ordinary region may contain only values that do not need drop. A resource-aware arena would require a drop list, partial-construction rules, reverse-order cleanup, and reset failure semantics; it is deferred until those contracts and tests exist.

Region-parameter origins, permitted returns, package-resolved local-region escape, resource-placement rejection, capability storage, and direct-construction rejection are versioned in `spec/fixtures/region/manifest-v1.json`. The lexical escape case names the explicit standard-library package instance that owns the opaque fixture `Arena` type.

Verified MIR and interpreter fixtures additionally execute lexical Region entry, inert placement,
inner-resource cleanup, and reset on fallthrough, direct `return`, Result `?` propagation, and
nested Region exits. `break` and `continue` remain deferred with their unimplemented control-flow
forms.

## Errors and control flow

- `Option<T>` represents presence or absence.
- `Result<T, E>` represents success or a typed failure.
- Both types inherit move and drop behavior from their payloads.
- Error variants may carry payloads.
- `?` is defined only for `Result` in Stage 0 and lowers to an explicit result branch.
- Results must be handled, returned, or explicitly discarded.
- Result variants are constructed only as `Result::Ok(expression)` and
  `Result::Err(expression)` in a context with one known `Result<T, E>` type.
- A shallow Result match contains exactly one `Result::Ok(binding)` arm and one
  `Result::Err(binding)` arm; both arms produce one compatible type.
- Stage 0 constructs Option with `Option::Some(expression)` or bare `Option::None`. A shallow
  Option match contains exactly one `Option::Some(binding)` arm and one bare `Option::None` arm;
  both arms produce one compatible type.
- Exceptions and stack unwinding are not part of the language.

Changing a value from `T` to `Option<T>` must not silently turn a normal function call into conditional execution. Stage 0 return types contain at most one safe-borrow leaf. Full control-flow behavior is defined in `docs/language/0004-results-traps-and-control-flow.md`.

Result return, discard, propagation, local and parameter transfer, branch and loop must-use dataflow, non-Result `?`, and error-compatibility assertions are versioned in `spec/fixtures/result/manifest-v0.json`. Qualified Result syntax is fixed by ADR 0017, qualified Option syntax by ADR 0018, and complete Stage 0 Result obligations by ADR 0019. Scalar and resource-bearing built-in constructors and shallow matches execute through the MIR interpreter. Whole resources execute construction, field reads, call moves, tagged-payload ownership transfer, active-variant cleanup, and source-linked drop events. User-defined resource-bearing enums remain excluded.

## Safe-code semantics

Optimization level and backend must not change the meaning of safe code. Expression operands and call arguments evaluate from left to right.

- Out-of-bounds indexing traps unless safety is proven statically.
- Division by zero, signed minimum divided by negative one, invalid shifts, and ordinary integer overflow trap.
- Null cannot inhabit ordinary references.
- Wrapping, saturating, and checked arithmetic use distinct operations.
- `assert` always traps when false; a debug-only assertion is not an optimizer assumption.
- A trap is non-unwinding and does not run cleanup.
- Optimizer assumptions and unchecked access are explicit unsafe operations.

Raw pointers, inline assembly, foreign code, and incorrect unsafe libraries can still introduce undefined behavior. The promise applies to the safe subset, not to the entire program.

## C interoperability

C is a boundary contract rather than the default internal representation.

- Stage 0 has no stable Fairy native ABI, mangling, binary package ABI, or dynamic-library contract.
- The first planned conformance target is `x86_64-linux-gnu`; it is not supported until all declared target fixtures and CI paths pass.
- C-layout types use `@repr(C)` and exported or imported symbols use explicit `unsafe extern C fn` declarations.
- Platform-dependent C types use a distinct `c.*` namespace.
- Foreign calls and generated raw bindings are unsafe by default and use raw pointers rather than safe references.
- Safe wrappers own length, nullability, lifetime, error, and ownership conversion.
- Stage 0 callbacks are non-capturing, synchronous, same-thread raw `unsafe extern C` function pointers plus explicit user data.
- A C `longjmp` or foreign exception crossing a live Fairy C frame is forbidden; boundary shims must translate it before control reaches Fairy C.
- Variadics, bit fields, atomics, packed layout, unions, TLS symbols, retained callbacks, and C++ ABI are excluded from C ABI v0.
- `TargetSpec` owns semantic layout; `ToolchainSpec` owns C driver, sysroot, libc, startup, linker, and capability fingerprints.
- The C compiler driver performs linking. Header import and export remain later toolchain goals.

The target contract is in `docs/abi/c-abi-v0.md`; generated-C safety obligations are in `docs/mir/c-backend-safety-v0.md`.

## Runtime and standard library

Compiler intrinsics, the thin runtime, and standard-library APIs are separate boundaries. The runtime owns traps, process entry, an empty-storage sentinel, and only the bounded memory helpers required by generated C. It does not own allocation policy, files, strings, logging, unwinding, scheduling, or collections.

Stage 0 allocation supports only explicit `SystemHeap` and lexical `Arena` or `Region`. Allocation is fallible, zero-size behavior is normalized by Fairy C, and no hidden default allocator or user allocator protocol exists.

`Buffer` is the only growable owned collection. `Vec<T>` is deferred with user generics. `str` and `CStr` are compiler-known unsized views; `String` and `CString` are owned resources. The initial Linux `File` API accepts `&CStr`, preserves partial I/O, and uses a consuming non-retried Linux close contract. Portable paths, embedded support, and resource-aware arenas remain deferred.

The complete prototype boundary is in `docs/runtime/runtime-stdlib-v0.md`.

## Build and tooling contract

- The Stage 0 compiler is implemented in Go.
- Modules follow the one-file, explicit-alias import contract and do not use textual include expansion.
- Parsing starts from original source bytes, preserves trivia and error provenance in compact lossless syntax, and lowers to AST only after an error-free batch parse. An immutable green tree and incremental reparse are deferred until formatter or IDE work requires them.
- Semantic checking proceeds through resolver, type, and Flow phases. Fixtures compare stable source coordinates and canonical facts rather than build-local AST or HIR dumps.
- Typed HIR lowers through an analysis-only Flow CFG into mutable-local, backend-independent MIR.
- The C backend is the only committed Stage 0 product backend and emits a constrained target/toolchain profile rather than universally portable C.
- A small MIR interpreter is the semantic oracle before any second native backend.
- Its fixture-only output hooks require explicit manifest bindings, capture bytes in memory under a
  shared limit, and never authorize source names, the CLI, or host streams.
- QBE or another fast backend and LLVM require separate measured adoption decisions after MIR stability.
- Early caching is whole-package and bounded. Function-level caching requires measured invalidation or latency pressure.
- Development output contains intentional deliverables and small state, not an unbounded copy of every intermediate.

Source-linked MIR normalization assertions are versioned in `spec/fixtures/mir/manifest-v0.json`. They freeze ordered evaluation, move and cleanup behavior, `Result` branching, call successors, trap paths, and eliminated source-only forms without making textual MIR or internal graph identity public.

MIR verifier rejection recipes are versioned in `spec/fixtures/mir-verifier/manifest-v0.json`. They require a verified named seed, exactly one typed mutation, and exactly one expected `Vxxxx` result without creating a general serialized MIR or plugin format.

Resolved package graph assertions are versioned in `spec/fixtures/package/manifest-v0.json`. They fix package-instance equality, one source root per instance, direct-only dependency visibility, module-provider ambiguity, and explicit standard-library injection without choosing public manifest, lockfile, coordinate, registry, or solver syntax.

Formatting, checking, testing, documentation, vendoring, binding generation, and ABI inspection are planned tool surfaces, not claims of current implementation.

## Compatibility

Compiler release, language revision, internal MIR, target schema, ABI profile, diagnostic catalog and JSON, runtime ABI, standard library, package schemas, `.fymap`, `SymbolId`, and interface hashes are independently versioned contracts. Pre-1 source and runtime contracts use exact compatibility. MIR and caches are disposable and are regenerated rather than migrated.

No v0 document creates a stable language, native ABI, MIR exchange, binary package, diagnostic-span, debugger, anchor, or `SymbolId` promise. The compatibility vector and failure behavior are defined in `docs/compatibility/versioning-v0.md`.

## Fairy Atlas

The compiler analysis layer produces an immutable `SemanticSnapshot` from typed HIR and analyzed MIR. Atlas consumes that snapshot through a one-way boundary. The compiler frontend cannot import Atlas, and Atlas failure cannot change `fairyc check` results.

Atlas exports deterministic, model-neutral facts such as symbols, calls, types, ownership transfers, borrows, errors, effects, allocations, unsafe and FFI reachability, source spans, and related tests. Compiler facts and free-text documentation are physically separated and carry provenance. Unknown or incomplete edges remain explicit rather than being guessed.

Stage 0 Atlas work starts with a versioned `.fymap` schema and the `overview`, `show`, and `around` queries. `impact`, semantic `diff`, stable anchor syntax, LSP, SCIP, MCP, structural edits, embeddings, model-generated summaries, and a vector database are deferred behind usage and completeness evidence.

Source text and documentation cannot grant an LLM, IDE, or agent filesystem, process, network, credential, or approval authority. Detailed contracts are in `docs/atlas/fymap-v1.md` and `docs/atlas/context-pack-v1.md`.

## Explicit non-goals

- universal source compatibility with C;
- universal memory safety or safe expression of every alias graph;
- garbage collection, exceptions, class inheritance, or implicit heap allocation;
- a built-in application framework or async runtime;
- arbitrary user-defined implicit conversion;
- mandatory full-program optimization for normal development builds;
- guaranteed performance superiority over all C implementations;
- early self-hosting.

## Open decisions

Package coordinates and manifest syntax, public `Arena` construction and capacity APIs, exact parser resource limits, glibc and toolchain baselines, portable path model, additional targets, package registry, `.fymap` binary encoding, and the concrete `SymbolId` hash algorithm remain UNDECIDED.
