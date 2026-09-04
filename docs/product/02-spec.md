# Product Specification

- Status: Draft
- Owner: UNASSIGNED
- Related ADRs: `docs/adr/0001-initial-architecture-boundaries.md` through `docs/adr/0137-c11-transport-baseline.md`

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
- a mutable safe-borrow local may be reassigned through straight-line, branch, and `while` control
  flow; header, `continue`, and `break` origins are joined without guessing;
- internal borrow-returning calls use inferred formal-parameter, region-parameter, or static return
  summaries; a recursive result without a complete proof remains closed;
- named aggregates cannot store safe borrows, and inert structs cannot contain resource fields;
- borrow-returning functions cannot be converted to function values, interface methods, or foreign callbacks;
- self-referential aggregates, user-defined interior mutability, captured closures, async or coroutine state, and thread capability traits are not Stage 0 features;
- a plain region cannot own values that need drop;
- generics, interfaces, dynamic dispatch, metaprogramming, and user macros begin only after the first executable semantic slice proves the non-generic core.

The detailed rules are in `docs/language/0001-values-and-resources.md`, `docs/language/0002-borrows-and-return-origins.md`, and `docs/language/0003-regions-and-arenas.md`.

Name resolution, canonical type selection, phase ownership, cascade suppression, and semantic fixture assertions are defined in `docs/language/0007-name-resolution-and-typing-v0.md` and `spec/fixtures/semantic/manifest-v0.json`.

## Stage 0 surface

- Source files use `.fy`, source is UTF-8, and Stage 0 identifiers are ASCII.
- Written declarations put one complete type before one name. C name-splitting declarators and C
  casts are absent. Source bytes reach the lexer without macro expansion, token pasting,
  conditional token removal, ambient defines, or textual includes; `#` is not a directive prefix.
- Simple statements use explicit semicolons; compound `if`, standalone blocks, and standalone
  `unsafe` blocks do not. Assignment is a statement, and integer values are not implicit booleans.
- Every file declares one module. Imports require aliases and cannot be wildcard, selective, cyclic, or re-exported.
- Target or feature variation must select explicit package/module graph inputs. It cannot rewrite or
  hide tokens inside a `.fy` file. Manifest v1 feature lists identify dependency instances only and
  do not enable source-level conditional compilation.
- `let` prevents reassignment, `var` permits reassignment and mutable borrowing, and both require initializers.
- Parameters use `Type name`. A callee that needs an assignable place initializes a `var` local from
  the parameter without changing the call type or ownership transfer.
- A by-value position consumes a resource. There is no general `move` keyword.
- Functions use `fn Result name(Parameters)` and always declare a result type. Associated functions
  and methods use owner-qualified names; a method's first parameter is an explicit shared, mutable,
  or consuming `this` receiver.
- `if` remains an expression. Exhaustive `switch` handles shallow variants and inert fields without
  fallthrough.
- `discard` explicitly ignores a must-use value and drops its active payload.
- Unsafe operations and C declarations use `unsafe`, `unsafe extern C fn`, `@repr(C)`, and `@link_name`.
- Decimal, hexadecimal, binary, floating, UTF-8 string, byte-string, C-string, boolean, and raw-pointer null literals use the exact spellings and escape rules in the parser contract. Implemented integers reject out-of-range magnitudes as `E1013` before MIR; direct unary minus folds the one exact `i32` minimum magnitude without changing runtime checked-negation behavior.
- Expressions follow one fixed Pratt binding-power table; comparisons are non-associative and ordinary or compound assignment is never an expression.
- File identity maps source-root-relative `path/name.fy` directly to `module path::name;` with no special root filename.
- Parser recovery preserves byte spans and error nodes but never permits a recovered file to reach HIR.

The explanatory surface is in `docs/language/0005-surface-syntax-v0.md`; exact spelling is owned by `spec/grammar/stage0.ebnf`, and lexing, precedence, module identity, and recovery are defined in `docs/language/0006-lexing-parsing-and-recovery-v0.md`.

### Current C-shaped pre-1 surface

ADR 0136 defines the current coordinated breaking source revision aimed at C programmers without
adopting C declarators. It uses explicit declaration introducers, type-before-name declarations,
`let name = value;` inference without `auto`, `T[N]` and `T[]` postfix type constructors,
read-only-by-default `ref T` and `ptr T` handles with explicit `mut` write capability, C99-shaped
`.field = value` record designators, parenthesized control headers, explicit function returns,
owner-qualified functions with explicit receivers, non-callable resource cleanup blocks,
contextual leading-dot variants, exhaustive no-fallthrough `switch`, and unchanged exact-error `?`
propagation. The grammar and examples named above are the only product-compiler surface. The
immediately preceding revision is accepted only by the separate migration tool; dual parsing and
compile-time source rewrites remain forbidden.

Classic scalar `for` and bounded lexical `defer` are the separate decisions in ADR 0213 and ADR
0217. ADR 0224 defers range `for`, checked narrowing conversions, and expression-bodied function
shorthand after a checked source audit found insufficient demand. Provider-qualified package imports,
generic iterators, and formatter behavior retain separate follow-up gates.

ADR 0218 owns the admission boundary for concepts borrowed from modern C++, Java after 17, Go, and
other languages. ADR 0219 implements a narrow callable `@must_use` contract with `E4201`/`E4202`,
direct-call Flow, typed-HIR retention, public-interface invalidation, and no runtime marker.
ADR 0220 implements private `bool`/`u8`/`i32`/`u32` module constants, private
`@const_eval` straight-line functions, and module-level `static_assert` under deterministic depth
and step limits. It deliberately avoids the Rust-shaped `const fn` spelling, target-dependent
execution, public evaluator transport, and any runtime marker. Its first semantic facts also stay
outside the closed five-kind Atlas v0 declaration transport rather than forcing an unrelated map
version. ADR 0221 implements exact zero-argument `source_location()` as a static relative-path and
byte-coordinate string under `E4401`/`E4402`. It explicitly excludes hidden default arguments,
callee propagation, runtime stack inspection, absolute paths, and new MIR or ABI surface. ADR 0222
retains ordinary named functions returning `T` or `Result<T, E>` as the complete construction model
after auditing the SDK/example factory and rollback patterns. ADR 0223 records that the current
29-file SDK/example corpus has no cross-type generic-algorithm duplication and that the grammar has
no user-generic declaration model. Static interfaces therefore remain absent until implemented
bounded user generics and measured duplication satisfy that ADR's reopening gate. Class inheritance,
exceptions, implicit conversions, overload sets, unrestricted template metaprogramming, runtime
reflection, garbage-collected object roots, and language-owned async or virtual-thread scheduling
remain non-goals. The current modern-concept admission queue is closed. ADR 0224 also keeps range
iteration, checked narrowing, and expression bodies outside `/6` until their exact evidence gates
reopen. ADR 0221 owns the current implemented `/6` cutover; later candidate statements do
not change the current language or Stage 0 grammar.

ADR 0328 records production-workload admission gates for Linux target promotion, cancellation and
structured task ownership, bounded data interchange and generated declarations, and vendor-neutral
operational context. Those gates are candidates only. They reserve no syntax or public API, accept
no server or database framework, scheduler, trait, macro system, registry, deployment platform, or
telemetry protocol, and change no current target maturity, SDK, runtime ABI, or language identity.

ADR 0342 releases resource abilities in `fairyc-language/11`. Direct and resolved-package analysis
share the same Flow rules, HIR preserves canonical ability metadata, and public-interface v4 plus
cache fixture v6 transport and invalidate every public effective ability set. The runtime ABI stays
unchanged because abilities add no runtime representation.

ADR 0342 also releases the five target-neutral on-chain effects in language 11 and stable package
analysis. No ambient chain context, storage API, event/call transport, rollback, metering, runtime,
backend, ABI, or target support is implied; each operation still requires an explicit capability
and its own vertical slice.

ADR 0343 advances the current language to `fairyc-language/12` and admits `burn local;` for a bare
local or parameter whose direct inert no-drop resource is declared in the current package and
module. Flow discharges one obligation, verified MIR preserves a distinct Burn operation, and the
interpreter and C backend destroy no fields or drop glue. Droppable, temporary, projected, wrapped,
resource-bearing, and foreign-module operands fail closed with `E1109`. SDK
`0.0.0-stage0.55` carries the exact language identity; public-interface v4 and runtime ABI remain
unchanged.

ADR 0377 advances the current language to `fairyc-language/13` and admits one explicit package-owned
`@contract_uid("...") state { ... }` declaration. Ordered `cell Value name;` and
`map Key Value name;` fields lower through the existing HIR layout and contract-state builder with
the closed `bool`, `u32`, `u64`, `bytes20`, and `bytes32` codec vocabulary. Public-interface v5 and
cache key v4 remain authoritative; no source access operation, runtime symbol, persistence provider,
or target support is added. SDK `0.0.0-stage0.56` carries the new language identity and runtime ABI
remains unchanged.

ADR 0338 adds a process-local typed state-layout and failure-atomic transaction oracle for future
contract profiles. ADR 0344 adds a separate experimental compiler-private layout and transaction MIR
that verifies canonical cell/map operations plus explicit commit, reject, and trap exits before
oracle execution. ADR 0345 adds compiler-constructible package-owned HIR layout declarations,
public-interface v5 layout hashes, and whole-package cache-key v3 current-interface invalidation.
ADR 0346 adds one opaque transaction capability plus verifier-derived `storage_read` and
`storage_write` requirements to the compiler-private submodel. ADR 0347 adds process-local
source-independent HIR transaction plans owned by existing package functions and lowers exact field
names, encoded values, effects, capabilities, and static exits through that verifier into oracle
execution. ADR 0377 now admits source layout declarations only; it adds no source state access or
stable/cached transaction-body artifacts, SDK,
runtime, backend, ABI, database, production Merkle storage, or a supported chain target. Production persistence must
preserve the oracle's observable layout, limit, rollback, conflict, and append-only upgrade rules
through separate evidence.
ADR 0374 adds a compiler-internal SHA-256 commitment over an exact in-memory layout/version snapshot,
with bounded inclusion and sorted-boundary absence proofs. It adds no persistence, serialization,
network proof service, consensus, chain target, or public Fairy API.
ADR 0376 defines a compiler-internal vendor-neutral persistence port for exact-layout snapshots,
atomic version/layout compare-and-publish, and guarded append-only layout publication. The reference
adapter is process-local executable evidence; no database, durable provider, native runtime path,
migration product, replication, or deployment target is selected.

ADR 0348 assigns each verified package transaction set an exact
`fairyc.contract-state-package-body/v1` identity over stable package/function/type facts, layout,
effects, operation bytes and order, and exit. It excludes build-local numbering and capability
tokens. ADR 0349 makes package-cache key v4 consume the exact owning-package body hash while
dependency entries remain public-interface-only. ADR 0350 adds a separate exact-version, bounded
canonical payload that reconstructs the layout, creates fresh private capability identity, runs the
verifier, and recomputes that hash. ADR 0351 additionally requires the expected hash and exact
current coordinate/layout/function descriptors before mapping the payload to fresh package/function
symbols and capabilities without current source operation bodies. ADR 0352 pairs the payload with
stable package MIR v3 in a coordinate-bound outer artifact. ADR 0353 makes persistent cache records
carry that bundle and requires the key-selected body hash before a hit. ADR 0354 reassembles both
members under the MIR program's fresh coordinate-owned function symbols and returns verified cached
state transactions for explicit compiler-private oracle execution. ADR 0355 makes ordinary and
cached compilation publish those programs through one atomic verified compiler result. ADR 0356
admits only the `storage_read` and `storage_write` effects represented by current transaction MIR.
ADR 0357 rejects calls without a transaction representation. ADR 0379 admits only statically
resolved same-package no-argument void helper calls with a closed acyclic graph, exact current-HIR
call sequence, transitive effects, stable v3 body identity, and one shared transaction overlay.
Indirect, foreign, cross-package, argument-bearing, value-returning, and recursive calls stay
closed. ADR 0358 also rejects conditionals, loops, assertions, Result propagation, and
short-circuit operators. ADR 0359 rejects unsafe code and recursively reachable raw-pointer types
while preserving ordinary safe-borrow and pointer-free resource types. ADR 0360 rejects
receiver-bound methods and automatic drop glue while preserving static associated entries; none
admits owner-bound state borrows or pointer provenance. ADR 0378 separately
admits bounded acyclic conditional state blocks with typed reads, explicit joins and represented
costs without lowering ordinary HIR conditionals. ADR 0361 also rejects ordinary potentially trapping operations while preserving the state
model's explicit static trap exit. ADR 0362 rejects owned resource values and lexical regions while
preserving safe resource borrows and static resource namespaces. ADR 0363 freezes the residual pure
owner HIR as a positive allowlist so future forms and expression-local effects fail closed.

ADR 0339 supplies a bounded compiler-internal specialization planner with exact provider coherence,
deterministic transitive identities, cycle rejection, and explicit expansion limits. It reopens the
generic evidence question without admitting source syntax, inference, HIR type variables, MIR,
runtime dispatch, reflection, ABI, or backend emission.

ADR 0340 adds a standalone bounded parser for a separate experimental contract-interface profile,
canonical chain-neutral IDL v0, and typed Go/TypeScript clients over a consumer-supplied transport.
It changes neither Stage 0 grammar nor the stable `fairyc` CLI and claims no source-level contract
call, codec, signing, reentrancy proof, HIR/MIR lowering, backend, deployment, or supported target.

ADR 0369 adds a standalone experimental deterministic-contract profile over the ordinary verified
MIR pipeline. Its profile-only `@max_iter` overlay is removed before core semantic analysis and does
not change the language revision or public interface. Profile v0 rejects ambient effects, FFI,
unsafe/raw-pointer operations, target-dependent types, recursion, unmetered Regions, and the
compiler result's contract-state transactions; persistent storage and target selection remain closed.

ADR 0370 adds a compiler-private fixed-width unsigned arithmetic oracle for possible future
deterministic workloads. It defines checked, wrapping, and saturating `u64`, `Uint128`, and `Uint256`
reference behavior without adding source types, MIR operations, backend lowering, ABI promises, or
target semantics. Address, signature, cryptographic, fixed-point, gas, and target choices remain
`UNDECIDED`.

## Value categories

### Inert values

`struct` represents passive data. It has no implicit constructor or destructor. A struct is copyable only when all fields are copyable. It cannot contain a resource or a safe borrow in Stage 0. Hidden heap allocation and user-defined implicit conversion are not allowed.

A restricted user `enum` may contain only inert copyable payload fields. A payloadless variant is
constructed contextually as `.Variant`; a payload variant uses `.Variant { .field = value }`.
Shallow switching names every payload field and covers every declared variant exactly once. The
expected enum type resolves the leading-dot form; global best-fit variant lookup is forbidden.
Built-in `Option` and `Result` may
contain resources because the compiler owns active-variant move and drop behavior. General
  resource-bearing enums are deferred. Public-interface v4 transports the canonical effective
  `drop`, `store`, and `key` abilities of public resources without yet opening stable package
  admission for the experimental source attribute.

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

Construction is performed by ordinary explicit functions such as `open_read`; there is no implicit
constructor call. A public resource type does not expose its representation: only its declaring
module may use a resource record literal, while external modules must call a source-visible factory.
Public inert structs with source-visible fields remain directly constructible. Lifecycle details are
defined in `docs/language/0001-values-and-resources.md` and `docs/mir/drop-elaboration-v0.md`.

Whole-root `Live`, `Dead`, and rejected `MaybeLive` transitions, live replacement, dead-slot reinstallation, and cleanup-obligation assertions are versioned in `spec/fixtures/resource/manifest-v0.json`.

### Borrows and raw pointers

- `ref T` is a non-null shared read-only borrow.
- `ref mut T` is a non-null exclusive mutable borrow.
- `ptr T` is a raw pointer. Inside `unsafe`, a direct borrow may erase its pointee to `ptr void` without
  strengthening mutability while MIR retains the concrete owner type. Null values, pointer
  arithmetic, every other pointee-changing cast, and dereference remain outside safe code.
- Nullable borrows use `Option<ref T>` rather than a nullable reference representation.

The Stage 0 borrow model uses local last-use loans and whole-root overlap. It does not infer disjoint mutable field borrows or two-phase borrows. A returned safe borrow must have one unambiguous origin. Stored borrows, self-reference, and multiple-origin returns must use an index or generation handle, a redesigned API, a lexical region, or a narrow unsafe module.

### Regions and arenas

`Region` is a compiler-known lexical capability, not an ordinary mutable arena object. It cannot be
stored or returned. `Region.place(T)` returns `Result<ref T, std::alloc::AllocError>`; the `.ok`
payload carries the region origin through tagged flow, and a derived safe borrow cannot outlive the
lexical region. Allocation failure is handled or propagated as data rather than converted into a
trap or implicit unwrap.

An ordinary region may contain only values that do not need drop. A resource-aware arena would require a drop list, partial-construction rules, reverse-order cleanup, and reset failure semantics; it is deferred until those contracts and tests exist.

Region-parameter origins, permitted returns, package-resolved local-region escape, resource-placement rejection, capability storage, and direct-construction rejection are versioned in `spec/fixtures/region/manifest-v1.json`. The lexical escape case names the explicit standard-library package instance that owns the opaque fixture `Arena` type.

Verified MIR and interpreter fixtures additionally execute lexical Region entry, inert placement,
inner-resource cleanup, and reset on fallthrough, direct `return`, Result `?` propagation, and
nested Region exits. `break` and `continue` target the nearest `while` or classic `for` and execute
scope-indexed resource cleanup and Region reset before their MIR goto; use outside a loop is
`E1012`. In a classic `for`, `continue` transfers through the post clause while `break` skips it.
The constrained C backend lowers the same verified Region lifecycle through the closed target
runtime. It retains the exact Arena owner, uses private lexical Region storage, derives placement
size and alignment from the target C type, initializes only successful storage, preserves the Region
origin on the `.ok` borrow, constructs `AllocError` on `.err`, and resets on every verified exit.
This backend slice admits canonical scalar, supported inert-struct, and scalar fixed-array
placement. The checked-in safe workspace executes all three placement shapes under both Windows
MSVC profiles.

## Effects and external capabilities

Functions declare external authority with `@effects(...)`. The Stage 0 vocabulary is exactly
`console`, `filesystem`, `environment`, `process`, `network`, `clock`, and `random`; omission is the
empty set. A direct call is accepted only when the caller declares every callee effect. `main` has
no ambient authority, and raw foreign calls must satisfy both this rule and their independent
`unsafe` boundary.

An effect-bearing nonempty resource drop body declares the authority used by its private glue.
Resource Flow requires that set only where a live value is actually cleaned up; a whole-root move
transfers the obligation without charging the sender. Effects are canonical typed-HIR metadata and
API compatibility facts, not proof that a runtime or
backend operation exists. `unsafe`, explicit allocation, Region operations, traps, and cleanup are
not external effects. Full rules are in `docs/language/0008-effects-and-capabilities-v0.md`.

## Errors and control flow

- `Option<T>` represents presence or absence.
- `Result<T, E>` represents success or a typed failure.
- Both types inherit move and drop behavior from their payloads.
- Error variants may carry payloads.
- `?` is defined only for `Result` in Stage 0 and lowers to an explicit result branch.
- Results must be handled, returned, or explicitly discarded.
- Result variants are constructed only as contextual `.ok(expression)` and `.err(expression)`
  where one `Result<T, E>` type is known.
- A shallow Result switch contains exactly one `case .ok(binding): { ... }` and one
  `case .err(binding): { ... }`; both cases produce one compatible type.
- Stage 0 constructs Option with contextual `.some(expression)` or bare `.none`. A shallow Option
  switch contains exactly one `case .some(binding): { ... }` and one bare
  `case .none: { ... }`; both cases produce one compatible type.
- Exceptions and stack unwinding are not part of the language.
- `break` and `continue` target only the nearest enclosing loop; labelled or value-producing loop
  control is not part of Stage 0.

Changing a value from `T` to `Option<T>` must not silently turn a normal function call into conditional execution. Stage 0 return types contain at most one safe-borrow leaf. Full control-flow behavior is defined in `docs/language/0004-results-traps-and-control-flow.md`.

Result return, discard, propagation, local and parameter transfer, branch and loop must-use dataflow, `break` and `continue` edge obligations, non-Result `?`, and error-compatibility assertions are versioned in `spec/fixtures/result/manifest-v0.json`. ADR 0017 and ADR 0018 record the superseded qualified spellings; ADR 0136 makes leading-dot Result, Option, and user-enum variants current. Complete Stage 0 Result obligations are fixed by ADR 0019 and loop-control cleanup by ADR 0024. Scalar and resource-bearing built-in constructors and shallow switches execute through the MIR interpreter. Restricted user-enum construction, named inert payload extraction, and exhaustive multi-way switches execute through verified MIR and the interpreter; one C-emission fixture carries the same behavior into the native matrix. Whole resources execute construction, field reads, call moves, tagged-payload ownership transfer, active-variant cleanup, and source-linked drop events. User-defined resource-bearing enums remain excluded.

## Safe-code semantics

Optimization level and backend must not change the meaning of safe code. Expression operands and call arguments evaluate from left to right.

- Out-of-bounds indexing traps unless safety is proven statically.
- Fixed arrays `T[N]` currently admit only `bool`, `i32`, `u8`, `u32`, or `usize`, nonempty list
  literals, and `1 <= N <= 4096`. The ceiling is a provisional implementation guard, not a stable
  language or ABI limit. Elements evaluate once left to right, arrays copy across internal calls
  and results, `.len` returns exact canonical `N` as `usize` without a scan or trap, and a `usize`
  index returns `T` or traps with `bounds`. A direct shared borrow of an admitted `u8[N]` place
  contextually becomes an immutable `ref u8[]` with the same local origin and exact `N`, so existing
  byte-view calls, metadata, indexing, and slicing can consume locally assembled bytes. A mutable
  safe-borrow local may switch owners after right-hand-side evaluation; Flow joins reachable
  branch and loop-edge origins and keeps every reachable loan generation live. Empty or repeated
  literals, other fields, nesting, mutation, general slicing, escaping local views, and foreign ABI
  use remain deferred. Exact whole-array `ref T[N]` and `ref mut T[N]` internal parameters are
  admitted without creating a slice or public representation.
- Division by zero, signed minimum divided by negative one, invalid shifts, and ordinary integer overflow trap.
- Matching-type shifts accept `i32`, `u8`, `u32`, or `usize`. Right shift is arithmetic for `i32`
  and logical for unsigned types. Negative `i32` amounts and amounts at or above the exact operand
  width trap as `invalid-shift`. Left shift then traps as `integer-overflow` when its mathematical
  result is not representable; invalid amount takes precedence over overflow.
- Null cannot inhabit ordinary references.
- Wrapping, saturating, and checked arithmetic use distinct operations.
- Ordinary `+`, `-`, and `*` accept matching `i32`, `u8`, `u32`, or `usize` operands and trap with
  `integer-overflow` when the mathematical result is outside that exact type. They never promote,
  change signedness, or wrap according to the backend or build profile.
- Ordinary `/` and `%` accept those same matching types. A zero divisor always traps as
  `division-by-zero`; only `i32` minimum with negative one additionally traps as
  `signed-division-overflow`.
- Unary `~` and binary `&`, `^`, and `|` accept the same four integer types and return that exact
  type without promotion, signedness conversion, or a trap. Binary operands must match.
- Source `as` integer conversion admits the target-independent, value-preserving pairs `u8 -> i32`,
  `u8 -> u32`, `u8 -> usize`, and `u32 -> usize`, plus explicit `i32 <-> c::int` when the selected
  target declares signed 32-bit `c.int`. Narrowing, signedness-changing, other C-scalar,
  pointer/integer, and other numeric conversions require a later explicit contract.
- `==` and `!=` accept matching `bool`, `i32`, `u8`, `u32`, or `usize` operands. `<`, `<=`, `>`, and
  `>=` accept matching operands from the four Fairy integer types. Matching raw-pointer equality is
  also retained for null/state checks without dereference. No implicit numeric promotion,
  signed/unsigned mixing, aggregate equality, pointer ordering, or boolean ordering is admitted.
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
- The executable source C-scalar slice transports all sixteen exact TargetSpec C scalars. They are
  distinct from coincidentally represented Fairy scalars; only signed 32-bit `c::int` has explicit
  bidirectional conversion with `i32`. C-scalar literals, operators, other conversions, and raw
  pointers remain deferred. Exact C-scalar fields are admitted only in
  nonempty `@repr(C)` structs; an acyclic nested field is admitted only when its type is also
  `@repr(C)`. Opaque `ptr void` and `ptr mut void` are admitted without dereference or conversion
  semantics. One pointer-to-opaque-pointer output-slot shape is admitted without opening arbitrary
  raw-pointer nesting.
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

`Buffer` is the only growable owned collection. `Vec<T>` is deferred with user generics. Positive scalar fixed arrays are compiler-owned inert values but are not growable collections or public ABI records. `u8[]`, `str`, and `CStr` are compiler-known unsized views; `String` and `CString` are owned resources. Compiler-owned `b"..."` literals implement immutable static `ref u8[]` construction and private generated-C transport. Safe borrowed `ref u8[]` and `ref str` views expose `.len` as a `usize` byte count and `.is_empty` as the exact zero-length `bool` by reading existing metadata; neither access scans, allocates, calls, traps, or counts Unicode characters. Immutable `ref u8[]` additionally supports read-only `usize` indexing with a `u8` result and end-exclusive slicing with an immutable `ref u8[]` result. Slice starts and ends may be omitted, normalizing to zero and the retained receiver length without re-evaluating the receiver. Written operands evaluate once left to right and trap with `bounds` before access or pointer derivation when their bounds fail; slicing preserves the receiver's borrow origin. Mutable fixed scalar-array places support checked exact-type element assignment. A direct contextual `ref mut` borrow of a mutable fixed `u8[N]` place may produce `ref mut u8[]` for an exact internal parameter. It preserves the exclusive origin, exposes canonical `.len`, supports checked `u8` reads and stores, and may derive `ptr mut u8` only inside unsafe code. Indexed assignment checks the receiver and `usize` index before evaluating its right-hand side, so an invalid place traps without value-side effects. Mutable integer locals and admitted indexed scalar places support statement-only checked `+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and `|=` with one place evaluation and no promotions. It does not add assignment expressions, increment operators, mutable slicing, resizing, temporary borrowing, retained foreign pointers, or public ABI. Language revision `/9` additionally permits those positive bounded copy-scalar fixed arrays as direct `Option` and `Result` payloads with the same value-copy, indexing, return, switch, and compatible Result-propagation meaning; it does not make an array growable, resource-bearing, nested, public ABI, or foreign ABI data. The public File core accepts `ref CStr`, preserves partial I/O, syncs explicitly, and uses a consuming non-retried close contract on Linux and Windows. Private pointer-width storage selects a descriptor-plus-one Linux adapter or the existing Windows HANDLE adapter without exposing either value. Same-handle position observation, absolute and bounded relative/end seek, and logical size observation are target-neutral; exact reads, Buffer reads, path mutation, and other metadata remain Windows-only. Portable Unicode path semantics, race-resistant containment, embedded support, and resource-aware arenas remain deferred.

An independent example-level owned-text wrapper exposes move-only Fairy `String` and `CString`
resources, source `Result` errors, scalar length access, owner-bound `ref str` and `ref CStr` views, and
verified drop cleanup over alias-safe opaque C storage bridges. Two exact unsafe compiler built-ins
accept a shared owner borrow, raw storage pointer, and `usize` extent; the returned view inherits
only the owner's origin. Safe wrappers confine those operations, and Flow rejects a by-value owner
move before the derived view's last use with `E2105`. Low and optimized Windows target builds
execute String copy/append, nonempty and empty CString construction, internal borrowed-view calls,
interior-NUL rejection, and implicit cleanup. This does not stabilize `std::text`, public resource
layout, the private generated-C view transport, a foreign safe-borrow ABI, Unicode algorithms, or
allocator selection.

The complete prototype boundary is in `docs/runtime/runtime-stdlib-v0.md`.

## Build and tooling contract

- The Stage 0 compiler is implemented in Go.
- The Windows smoke must also build `cmd/fairyc` as a standalone temporary `fairyc.exe`, execute its
  help surface, verify the checked-in workspace with stable `check --locked`, replace and execute its
  default artifact with stable `build --release --locked`, execute the workspace through stable
  `run --release --locked` without changing that artifact, and use that separate process to compile
  and run both the checked-in scalar quickstart through `run-target` and exact-output example through
  `exec-target` for `x86_64-windows-msvc`. Stable run must preserve the program's ordinary result
  42 as process status 42; the latter experimental output command must expose exact stdout and the
  same process status to the parent.
  This proves the bootstrap process boundary only: the compiler is still Go-built, the artifacts
  are removed, and no self-hosting, installation, distribution, or WSL contract follows.
- The same smoke must build `examples/owned_text/main.fy` with its explicit safe support module at
  low and optimized `x86_64-windows-msvc` profiles, inspect both PE/unwind images, execute native
  result 42, and remove the products. This is source-level lifecycle evidence, not a stable library
  surface or hosted-Windows receipt. It must separately compile the raw owned-text probe and
  target-specific implementation as caller and callee translation units at both profiles. Those
  executions must preserve opaque
  storage on rejected input, accept self-overlap, reject null and malformed storage, distinguish
  interior NUL and size overflow, publish exact bytes and one terminator, and normalize normal plus
  repeated drop. A probe-only allocator must force String/CString initial allocation failure and
  String append reallocation failure, preserve canonical or prior storage and bytes, and return the
  live-allocation count to zero after cleanup. Product linkage must not include that adapter.
- The same Windows smoke must separately compile the raw Buffer probe and target-specific Buffer
  plus SystemHeap implementation as caller and callee translation units at low and optimized
  profiles. It must force nonempty initialization, reserve growth, relocating self-overlap append
  growth, and zeroed-resize growth allocation failure. Construction failure must preserve caller
  output; every growth failure must preserve the exact pointer, capacity, alignment, initialized
  length, and bytes with one live allocation. Normal recovery must produce the expected overlap
  append bytes, expose zero-filled growth, and return the live count to zero after drop. Product
  linkage must retain the ordinary allocator and no probe controls.
- The same Windows smoke must separately compile the raw Arena probe and target-specific Arena plus
  SystemHeap implementation as caller and callee translation units at low and optimized profiles.
  It must observe zeroed and copied bytes, simultaneous and stale Region rejection, active-drop
  preservation, and final cleanup. Probe control must force failure before payload allocation and
  after payload allocation but before tracking-node publication; both failures must preserve the
  Arena head and caller output, the second must release its payload, and the live-allocation count
  must return to zero. Product linkage must retain the ordinary allocator and no probe controls.
- The constrained C unit records sorted unique link names used by actual foreign calls. The
  experimental product build resolves exact project-owned standard-library names through one shared
  symbol/component contract and links only their dependency closure plus the scalar runtime; unused
  declarations and unrelated library components are not linked.
- Modules follow the one-file, explicit-alias import contract and do not use textual include expansion.
- Parsing starts from original source bytes, preserves typed trivia and error provenance in compact
  lossless syntax, and lowers to AST only after an error-free batch parse. Physical-line-leading
  consecutive `///` comments attach by exact byte span only to the selected top-level declaration
  families through typed HIR; they remain grammar trivia and carry no rendered or semantic meaning.
  ADR 0375 adds an editor-internal immutable green tree and a deterministic full-file formatter that
  changes only lexer-owned whitespace after proving identical parser shape and significant tokens.
  Incremental reparse, LSP, SCIP, semantic rename, and structural edits remain deferred until
  measured editor work requires them.
- Semantic checking proceeds through resolver, type, and Flow phases. Fixtures compare stable source coordinates and canonical facts rather than build-local AST or HIR dumps.
- Already-resolved multi-package input carries a deterministic build-local package discriminator and
  exact import-provider bindings. Semantic module identity is package plus source-visible module
  path, and typed HIR and MIR retain the discriminator for internal type and function identity.
  Every source import in that input requires exactly one validated provider binding; missing,
  duplicate, unused, or absent-package bindings reject before semantic phases.
  Semantic and MIR spans store the build-local package discriminator separately from the original
  relative source path. Source-visible paths never contain the private numeric qualification.
  The private numeric ID and qualification spelling are disposable and cannot define source syntax,
  public diagnostics, lockfiles, caches, Atlas identity, or ABI names.
- Typed HIR lowers through an analysis-only Flow CFG into mutable-local, backend-independent MIR.
- One compiler-owned coordinator runs semantic analysis, MIR lowering, and MIR verification for
  both explicit single-package closures and already-resolved package input. Source diagnostics and
  unsupported forms stop before MIR; invalid package input, lowering failure, and verifier rejection
  retain distinct failure ownership. Only a verified result may reach execution or a backend.
- Resource declarations retain canonical type identity, declaration-ordered resolved fields, source
  spans, and cleanup-body classification through typed HIR into MIR. The current executable prototype
  accepts an empty or effect-declared nonempty user `cleanup` body, lowers the latter to verifier-linked
  private glue, and checks actual implicit cleanup edges in Resource Flow.
- Inert struct declarations retain canonical identity, declaration-ordered fields, and `@repr(C)`
  classification through AST, typed HIR, verified MIR, the executable interpreter subset, and the
  constrained C backend. The current C-layout implementation is nonempty and acyclic over `u8`,
  `i32`, `u32`, `usize`, all sixteen exact TargetSpec C scalars, admitted raw pointers including
  opaque `void`, and nested types that are themselves `@repr(C)`. Dependencies are emitted before
  their users; recursive value types and broader contracted field families remain fail-closed.
- The executable raw-address slice accepts only an exact direct-local `ref T -> ptr T`,
  `ref mut T -> ptr T`, or `ref mut T -> ptr mut T` cast inside `unsafe`. AST keeps the target type separate, typed HIR keeps
  unsafe scope, MIR verifies pointee/mutability/provenance, generated C emits `&local`, and the
  interpreter fails closed instead of inventing an address. The admitted `T` surface now includes
  opaque raw pointers, enabling only the exact `ref mut ptr mut void -> ptr mut ptr mut void` output slot needed
  by the Arena source probe.
- An `unsafe extern C fn` may keep its Fairy name as the external symbol or use exactly one
  nonempty `@link_name("symbol")`. AST preserves raw and cooked attribute evidence, typed HIR and
  MIR preserve distinct source/link identities, and generated C validates and emits only the link
  identity. Compatible declarations of one external symbol coalesce; conflicting signatures fail
  closed. The attribute grants no interpreter or host-binding authority.
- A foreign result may produce any of the sixteen exact TargetSpec C scalars, occupy a typed MIR
  local, and feed a later matching foreign parameter. One drift-checked backend table owns each
  exact C spelling, TargetSpec key, feature, and optional provider. Generated C asserts target
  size/alignment for every used scalar and signedness for signed and unsigned integer families.
  Target-specific `ssize_t` consumes declared `sys/types.h`/`posix.1-2008` metadata. Plain `char`
  follows target signedness. Floating transport is limited to exact finite probe values and makes no
  NaN, infinity, signed-zero, rounding, or exception-mode promise.
- A foreign result may also be a nonempty acyclic `@repr(C)` struct from the admitted field subset.
  Generated C uses the exact public tag and ordinary C by-value assignment; target-derived size,
  alignment, and field offsets remain asserted. Foreign aggregate parameters remain rejected. The
  source probe crosses one 8-byte INTEGER-class result and one 20-byte MEMORY-class
  hidden result between separate Clang/GCC objects in both directions and profiles.
- The C backend is the only committed Stage 0 product backend and emits a constrained target/toolchain profile rather than universally portable C.
- ISO C11 is the only normative constrained-C transport dialect. C17 or C23 compiler modes, if
  added, validate the identical C11 source as consumers and do not create product emission profiles
  or authorize C23-only output. The Windows ABI evidence lane now repeats its generated-Fairy
  small/large struct-result crossing under explicit C11 and C17 MSVC consumer identities; C23
  remains disabled. Target ABI, runtime capabilities, and instrumentation remain separate
  compatibility axes as fixed by ADR 0137.
- `internal/target` owns executable `fairyc.target-spec/v0` descriptions. The first Linux target
  records host-independent LP64, scalar, pointer, SysV AMD64, ELF, unwind, debug, and C-layout facts.
  The provisional Windows target records LLP64, four-byte C `long`, 32-byte shadow space, Microsoft
  x64 identities, COFF, and Windows unwind/debug identities. Both exclude every local compiler,
  SDK, sysroot, registry, and filesystem path.
- `internal/layout` consumes that description and computes checked scalar, pointer, explicitly
  based C-enum, positive fixed-array, and nonempty nested-struct size, natural alignment, field
  offset, and top-level padding facts. `fairyc.layout-fixtures/v0` currently executes 16 pass/fail
  cases. This target-mechanics path does not by itself admit corresponding source syntax.
- `internal/cabi` consumes those same target and layout facts for the closed SysV AMD64 Stage 0
  type vocabulary. `fairyc.c-abi-fixtures/v0` executes 17 exact class and register-plan cases,
  including hidden returns and whole-aggregate rollback under integer or SSE register exhaustion.
  Direct array signatures fail because fixed arrays cross C only inside a representable struct.
  The 14 pass plans generate separate Clang/GCC caller and callee objects in both directions and
  profiles for 56 native crossings. ADR 0214 records their first hosted success. This does not
  implement source FFI.
- For Windows the same engine selects a distinct Microsoft x64 planner. A separate 15-case
  manifest covers RCX/RDX/R8/R9 and XMM0-3 selection by shared declaration-order slot, later stack
  parameters, 1/2/4/8-byte record integer transport, larger or odd-size record indirection, and the
  hidden-return RCX slot. Its 12 pass plans generate separate caller and callee C11 translation
  units that x64 MSVC compiles, links, and executes at low and optimized profiles. ADR 0214 records
  the first hosted execution of that same-compiler crossing. It is not source-level Fairy FFI or
  independent ABI inspection.
- `internal/pecoff` parses every retained or temporary executable produced by the Windows host
  smoke. It requires a bounded AMD64 PE32+ non-DLL console image, large-address awareness, aligned
  image/header/section ranges, a readable executable `.text` section, and an entry point inside
  executable code. It records import and exception-directory presence without treating either as
  validated content. Complete `.pdata`/`.xdata` semantics, CodeView/PDB, and broad source-level
  Windows FFI remain separate gates. The generic
  hosted-receipt marker is closed by ADR 0214; complete object and unwind fixture gates are not.
- The Windows C driver retains each compiler-produced `.obj` only in its bounded temporary root,
  parses it as raw AMD64 COFF, and fails before returning the linked image when the object has an
  optional PE header, linked-image flags, no `.text`/`.text$...` code, invalid raw or relocation
  ranges, relocation indexes outside the raw primary-symbol set, auxiliary relocation targets,
  unknown AMD64 relocation types, malformed COMDAT definitions, or no named symbols. Uninitialized
  `.bss` may have a nonzero logical size and zero raw pointer. ADR 0231 adds a versioned internal
  raw-object manifest for exact primary symbols, REL32, ANY COMDAT, and associative COMDAT parent
  receipts. ADR 0232 adds a separate two-translation-unit MSVC probe that reads the exact object
  addend and preferred-image function addresses, recomputes `S + A - (P + 4)`, compares the linked
  `CALL rel32` displacement, and executes at low and optimized profiles. ADR 0234 preserves exact
  relocation section indexes and reuses those runs to bind one receipt `ADDR64` relocation to its
  map-owned target address and all three `.pdata` `ADDR32NB` entry relocations to independently
  mapped source and unwind-info RVAs and the validated runtime-function entry. ADR 0233 adds a real
  two-member Microsoft archive plus one direct pick-any COMDAT competitor. The bounded archive
  inspector validates both linker members and every contained object; the final proof requires the
  preferred index, demanded and unused member identities, a non-COMDAT unused sentinel absent from
  the final map, raw candidate values, exact MSVC map owners, and `.fysel` preferred-image addresses and candidate value to
  agree across four low/optimized input-order executions. ADR 0235 also binds the selected function
  COMDAT to its direct associative `.pdata` and `.xdata` children, all three raw `.pdata` addends,
  exact archive-member map owners, and unique final runtime-function entry without adding another
  image. ADR 0236 adds a second contributed ANY-COMDAT function parent with a distinct unwind child;
  the selected owner and bytes must agree while the losing payload occurs zero times in final
  runtime-function references and file-backed sections. ADR 0237 adds one exact generated object;
  each of its `REL32` through `REL32_5` records must have one exact target and satisfy
  `S + A - (P + 4 + n)` in the low and optimized final images while its trailing marker bytes stay
  unchanged. ADR 0241 adds a second exact generated object with two weak `SEARCH_ALIAS` records.
  One absent name must select its object-local fallback and one present name must select the strong
  definition in `target.obj`; both final REL32 values and both native return values must agree at
  low and optimized profiles. ADR 0242 adds a third exact generated object with the smallest
  extended relocation table: one count sentinel, 65,534 semantic ABSOLUTE records, and one final
  REL32 record required by its executable result. Its exact map owner, final displacement, and
  native result must agree at both profiles. ADR 0243 adds a fourth exact generated object with a
  version-2 bigobj header and 20-byte symbols. Its internal REL32, exact map owner, final
  displacement, and native result must agree at both profiles while the 4,096-section inspection
  budget stays unchanged. ADR 0238 compiles the two C inputs with `/Z7`, requires exact function-targeted
  `.debug$S` `SECREL`/`SECTION` pairs and object-level formulas, and binds the final PE RSDS GUID/age
  to the bounded MSF 7 PDB information stream at both profiles. ADR 0239 then requires system
  DbgHelp to resolve the exact source-function start to its name and `caller.c` line with zero
  displacement. ADR 0240 separately locates the exact handler-bearing `frame.obj` unwind record,
  requires the profile-specific tail relocation (`__GSHandlerCheck_SEH` for low and
  `__C_specific_handler` for optimized) to be `ADDR32NB`, and binds its addend and MSVC map row to
  the executable final handler RVA. The driver still removes every temporary
  object, archive, map, PDB, and image. This is not
  a stable product object or library format, proof of high-section bigobj or auxiliary/COMDAT
  layouts, remaining relocation formulas or chained
  unwind tails, non-unwind or broader archive/COMDAT policies, complete CodeView/PDB semantics, broader
  symbolication, unwind fixture, or fresh hosted same-commit receipt.
- `internal/pecoff` additionally maps the exception directory through `.pdata` and every referenced
  file-backed unwind record. `RUNTIME_FUNCTION` entries must be 12-byte, sorted, nonoverlapping,
  executable-code ranges with DWORD-aligned unwind RVAs. Version 1 prolog operations use their
  closed slot counts and descending offsets. Version 2 admits bounded short and long epilog
  descriptors plus spare slots because local MSVC 14.51 emits them despite the public Learn page
  still describing version 1 as current. Handler and chained tails receive minimum structural
  checks only.
- A separate three-translation-unit Windows probe deliberately raises one private noncontinuable
  software exception below a stack-owning `__try`/`__finally` frame and catches it in an outer
  `__try`/`__except` frame. Its filter validates both exception arguments and uses
  `RtlLookupFunctionEntry` plus `RtlVirtualUnwind` over the captured `CONTEXT` to observe the
  raiser, termination frame, and catcher in order. The selected handler then requires the
  `__finally` block to have executed exactly once with `AbnormalTermination()` true. Local x64
  MSVC compiles and executes this closed C probe at low and optimized profiles after COFF and
  linked-image structure validation. This is explicit Windows dispatcher and runtime-walk evidence,
  not a Fairy C language unwind feature: Stage 0 traps and calls remain non-unwinding. ADR 0240
  additionally binds the frame's exact profile-specific handler-tail object relocation to the final
  executable target before these executions. Version-2 epilog offset semantics, language-specific
  handler payload parsing, recursive chain equivalence and chained-tail relocation, source-level
  Fairy or foreign-frame crossings, and hosted
  fixtures remain fail-closed.
- The eleven valid layout cases additionally generate one deterministic private C11 plus explicit
  POSIX-typedef probe. The Linux
  native runner compiles, links, and executes it with Clang/GCC and low/optimized profiles, checking
  recursive size, alignment, field size, and offset assertions. Failure cases remain Go-owned input
  rejections. ADR 0214 records the first hosted success of the valid probes.
- A small MIR interpreter is the semantic oracle before any second native backend.
- Its unstable `experimental run` path accepts one entry plus an explicit bounded repeated-`--source`
  closure, selects the entry module's `main() -> i32`, and performs no source discovery.
- Its unstable `experimental check` path accepts one primary source plus the same bounded repeated
  `--source` closure, requires no executable entry, and stops after MIR verification. It performs no
  backend, target, toolchain, process, network, or artifact action. It remains distinct from stable
  package-aware `fairyc check`.
- Stable `fairyc new <package-name> --uid <package-uid>` creates one no-overwrite direct-child
  starter workspace with explicit durable UID, manifest v1, public `answer()` returning 42, a private
  entry returning zero when that answer is correct, one semantic test, a build-only ignore rule, and
  one compiler-owned README. The README labels its commands as manual, explains lock refresh and
  stable run v2's stream/process-status boundary, distinguishes the default semantic test from the
  explicit isolated Windows native-test command, and carries
  one complete safe `std::io::write` source replacement plus the package-specific retained Windows
  executable path. The command never guesses identity, merges or upgrades an existing path,
  generates a lockfile, accepts an arbitrary destination, or uses a selectable template, prompt,
  stdin, environment, config, process, toolchain, registry, or network. Ordinary creation failures
  attempt contained rollback; crash-left partial directories remain user-owned recovery.
- Stable `fairyc check [--locked]` loads the current directory's strict manifest-owned local
  workspace, selects the compiler-bundled SDK only when imported, maintains canonical `fairy.lock`
  in ordinary mode or requires an exact no-write lock in `--locked` mode, and runs the complete
  package-aware frontend through verified MIR without requiring `main`. Its first version accepts no
  source paths, target, backend, toolchain, profile, cache, JSON, network, stdin, or configuration
  override.
- Stable `fairyc build [--release] [--locked]` uses that workspace and lock boundary, fixes the
  executable entry to root `main::main` with type `fn() -> i32`, and emits the closed C11 transport
  through local x64 MSVC on Windows/amd64. The root manifest's validated package name selects the
  sole retained path `build/<package-name>.exe`. Default mode is the low-optimization development
  profile; `--release` is the ordinary optimized profile without LTO. The complete new PE and unwind
  image is inspected and staged before replacing the previous artifact; failure before replacement
  preserves the prior successful executable. No user output/target/compiler/linker/profile-value,
  cache, JSON, configuration, network, or stdin surface is admitted. Command stability does not
  change the target's independent maturity state.
- Stable `fairyc run [--release] [--locked]` uses the same workspace, root entry, lock policy,
  development/release profiles, and closed local Windows target pipeline. It executes an inspected
  compiler-temporary image and retains no executable. After successful compilation, linking, and
  inspection, caller stdin is attached directly without compiler-side pre-reading, retention, or a
  separate input byte ceiling; pre-launch failures consume no program input. Bounded captured stdout and stderr are
  forwarded to matching streams without a compiler success banner; an ordinary `i32` return becomes
  the complete process status. A recognized language trap returns 5 even when its reserved OS status
  equals a normal result. A compiler-private run-only suffix provides that separation; only the suffix
  is removed, while preceding program stderr and a canonical trap line remain visible. The two OS
  pipes share a 64 KiB ceiling without a cross-stream ordering promise. The command accepts no program
  arguments, `--` delimiter, retained artifact, target, toolchain, profile-value, cache, JSON,
  configuration, or network surface. It never creates, replaces, reads, or executes the stable build
  artifact. Command stability does not change target maturity or provide hostile-code isolation.
- Its fixture-only output hooks require explicit manifest bindings, capture bytes in memory under a
  shared limit, and never authorize source names, the CLI, or host streams.
- The current C backend emits deterministic C11 only for verified scalar, inert-struct,
  restricted inert-enum, enum-containing private inert-struct, flat-resource, and resource-tagged MIR
  under the provisional `x86_64-linux-gnu` profile. It includes private `Option<i32>`,
  `Result<i32, i32>`, copy-only built-in tags over canonical Fairy scalars, positive bounded
  copy-scalar fixed arrays, admitted inert structs, or restricted user enums,
  nonempty empty-drop resource transport over Fairy copy scalars or raw pointers, and built-in
  resource-bearing tags whose payloads are exactly `i32` or direct admitted resources, plus private
  restricted user-enum tags
  whose payload fields are admitted scalars, raw pointers, admitted inert structs, acyclic
  restricted inert user enums, or direct `Option<T>` and `Result<T, E>` transports whose payloads
  are canonical Fairy copy scalars, admitted inert structs, or restricted inert user enums. Those
  acyclic copy-only tag graphs whose leaves are admitted Fairy copy scalars, inert structs, or
  restricted user enums are also admitted as private inert-struct or user-enum fields; one
  combined graph emits every
  payload aggregate or inner tag before its dependent tag and every tag before its by-value
  containing aggregate. Fixed-array-, borrow-, raw-pointer/C-scalar-, recursive, or
  resource-bearing built-in tags inside
  aggregates, recursive dependencies, and resource-bearing user enums remain C-backend capability
  boundaries.
  `Result<T, UserEnum>` propagation preserves and reconstructs the exact
  error enum on the enclosing return path. No public
  tagged or resource ABI is claimed.
  Nonempty admitted `@repr(C)` structs preserve public tags and field names after namespace-aware
  rejection of C implementation-reserved, generated-header macro, and backend-owned collisions; their layout and private
  resource layout are computed from the target description and guarded by generated C11 size,
  alignment, and offset assertions. Its emission fixtures do not compile, link, or execute C and do
  not establish a target support level.
- A separate Linux/amd64 command requires glibc, Clang, and GCC and is configured to compare the
  current scalar, flat inert-struct, restricted inert-enum, flat-resource, and resource-tagged cases at low and optimized
  profiles. It separately links generated FY callers to hand-written individual `c::int`,
  negative-preserving `c::long`, `c::size_t`, and negative-preserving `c::ssize_t`
  producer/consumer C units plus one twelve-type C-scalar family unit, and to the actual text-view C
  unit for both the empty predicate and `@repr(C)` output-record mutation paths. An exact generated
  `void *`/`size_t` block additionally crosses the real SystemHeap allocate and deallocate helpers.
  A nested generated Buffer record then crosses the real initialize, zeroed-resize, and drop
  helpers. Exact generated node, Arena, and Region records additionally cross real initialization,
  begin, zeroed allocation through `void **`, reset, and drop. An exact generated CStr view record
  crosses the real bounded validator with valid, interior-NUL, and unterminated storage and observes
  unchanged output on both failures. Exact nested String/CString records additionally cross the real
  copy/view/append/drop and from-text/view/drop helpers and observe bytes, termination, statuses, and
  normalized release. Exact Linux File result, handle, and open-option records additionally cross
  the real open, write-all, sync, consuming-close, reopen, partial-read, EOF, and final-close helpers
  while observing counts, repeated-close status, normalized state, and bytes. Report v36 retains four
  `linuxConsoleExecutions` for the target-neutral stdout and stderr adapter's empty, invalid,
  redirected-write, and unavailable-stream cases. It retains
  `linuxFileSourceCallExecutions` alongside the earlier standard-library source-call counters and
  four `linuxPublicFileExecutions` for an ordinary generated-Fairy product that selects the public
  File component, writes seven bytes, observes position and size seven, seeks to positions two and
  one, seeks relative to end to positions five and ten while size remains seven, resets to two,
  rejects `usize` maximum without moving the cursor, restores position seven, syncs, reopens, reads
  the exact bytes, and returns 42 under
  Clang/GCC low/optimized execution in isolated temporary working directories. It retains
  `foreignCStructResultCallExecutions` for the register and hidden-memory result pair, plus
  `standardLibraryBuildSmokeExecutions` for a self-contained product build that calls the real UTF-8
  validator through automatic used-symbol linkage. It additionally records
  `cStringLiteralSourceCallExecutions` when generated source-owned `fairy\0` storage is read by a C
  callee and `cStringLiteralBuildSmokeExecutions` when the ordinary product build carries nonempty
  compiler-owned `ref CStr` storage into an executed binary. It additionally records
  `byteStringLiteralSourceCallExecutions` when a generated private `ref u8[]` view preserves five exact
  `fairy` bytes and its length before C reads the data, and `byteStringLiteralBuildSmokeExecutions`
  when the ordinary product build carries the same view path into an executed binary. It additionally
  records `utf8StringLiteralSourceCallExecutions` when a generated private `ref str` view preserves the
  eight UTF-8 bytes and length of `Fairy λ` before C reads the data, and
  `utf8StringLiteralBuildSmokeExecutions` when the ordinary product build carries the same text-view
  path into an executed binary.
  The compiler-bundled SDK `0.0.0-stage0.56` exposes
  `std::io::write(ref str) -> Result<usize, ConsoleError>` and a matching `write_stderr` operation
  for text, plus `write_bytes(ref u8[])` and `write_stderr_bytes(ref u8[])` for exact binary output.
  All four declare `console`, keep raw pointers and target linkage private, append no newline,
  preserve completed-prefix byte counts, and map one internal symbol per
  output stream to a shared target-owned Linux POSIX or Windows Win32 adapter. The module additionally
  exposes `read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>`. It performs one
  allocation-free bounded read, treats a successful zero count as EOF, and leaves text decoding,
  line handling, buffering, and retry-to-fill to later APIs. It also exposes canonical decimal
  `write_i32(i32) -> Result<usize, ConsoleError>` output without allocation, locale, or an implicit
  newline. It also exposes canonical decimal `write_u32(u32) -> Result<usize, ConsoleError>` and
  `write_usize(usize) -> Result<usize, ConsoleError>` output covering the full unsigned domain in
  at most 10/20 bytes and ASCII `write_bool(bool) -> Result<usize, ConsoleError>` output. Matching
  `write_stderr_i32`, `write_stderr_u32`, `write_stderr_usize`, and `write_stderr_bool` operations
  select stderr under the same canonical formatting, allocation, newline, and completed-prefix
  error contracts. The checked-in safe workspace has exact
  text, byte-view, and `INT32_MIN` Windows output evidence through stable build/run plus direct-artifact EOF,
  short-input, and eight-byte-prefix evidence through a separately built compiler. Stable run now
  forwards the short input after successful compilation; Linux native execution remains configured
  but unobserved on this host.
  ADR 0261 additionally exposes the effect-free `std::process` surface:
  `arg_count() -> usize` includes the executable name at index 0, and
  `arg(usize index, ref mut u8[] buffer) -> Result<usize, ProcessArgError>` copies one argument's
  exact UTF-8 bytes with statuses 1 (out of range), 2 (invalid encoding, or argv that could not be
  materialized as valid UTF-16), and 3 (buffer too small, whose `needed` also serves as the length
  query). Stable `fairyc run` forwards arguments after `--` to the temporary program; `build`
  rejects the delimiter. The generated C entry remains `int main(void)`; because narrow startup
  leaves the UCRT wide argv NULL, the Windows adapter materializes it once with
  `_configure_wide_argv(_crt_argv_unexpanded_arguments)` before strict two-pass
  `WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, ...)`, and the checked-in quickstart proves
  argv[0], empty, flag-looking, quoted, trailing-backslash, and non-ASCII round-trips plus both
  error paths under Windows development and release profiles.
  ADR 0246 additionally exposes a move-only `std::bytes::Buffer` with explicit fallible
  construction, exact copy, reserve, append, zeroed resize, truncate, clear, scalar metadata, and
  owner-bound shared/exclusive views of only the initialized prefix. The SDK contains every raw
  pointer and runtime symbol. The checked-in safe quickstart executes that full lifecycle under
  Windows development and release builds; private layout, spare-capacity access, and generic
  element storage remain excluded.
  ADR 0373 additionally exposes `append_i32`, `append_u32`, `append_usize`, and `append_bool` on
  `std::bytes`. They append canonical locale-independent ASCII to a caller-owned Buffer through one
  existing bulk append, return the new initialized length, and preserve data, length, capacity, and
  ownership on allocation failure. Format literals, templates, reflection, varargs, floating point,
  user-defined display, and implicit Buffer creation remain excluded.
  ADR 0247 additionally exposes move-only `std::text::String` and `CString`, explicit fallible
  construction and append, byte length and emptiness metadata, owner-bound immutable `ref str` and
  `ref CStr` views, and separate text/allocation status fields. The checked-in safe quickstart
  executes nonempty and empty owners, append, interior-NUL rejection, views, and cleanup under both
  Windows profiles. Private storage, capacity, mutable text bytes, implicit conversion, formatting,
  locale, and Unicode algorithms remain excluded.
  ADR 0248 additionally exposes
  `std::text::validate_utf8(ref u8[]) -> Result<ref str, TextError>`. Success borrows the exact
  input bytes without allocation, copy, or normalization and retains the input origin. Invalid
  input returns text status `2` and allocation status `0`. The checked-in safe quickstart covers
  nonempty multibyte success and invalid-byte failure under both Windows profiles. Implicit
  byte-to-text conversion, lossy decoding, text indexing or slicing, and a public view ABI remain
  excluded.
  ADR 0263 additionally exposes `std::text::from_buffer(bytes::Buffer) ->
  Result<String, bytes::Buffer>` zero-copy ownership transfer. Valid UTF-8 moves the same
  allocation into the String without allocation or copy; invalid UTF-8 returns the original Buffer
  unchanged as the `Result` err payload; an empty Buffer becomes an empty String. The checked-in
  safe quickstart executes the valid, invalid, and empty arms under both Windows profiles.
  `text::copy` remains the copy-required borrowed-text path.
  ADR 0302 additionally exposes
  `std::text::as_bytes(ref str) -> ref u8[]`. It constructs an immutable byte view in constant time
  without allocation, copy, validation, or a runtime call, and retains the exact input borrow
  origin. The checked-in safe quickstart covers ASCII, multibyte UTF-8, appended String, and empty
  views under both Windows profiles; package-aware Flow rejects moving the owning String before a
  derived view's last use. Mutable bytes, capacity, raw parts, implicit conversion, and a public
  view ABI remain excluded.
  ADR 0264 additionally exposes the Windows MSVC `std::fs` slice. `File` is move-only, carries
  `filesystem` cleanup authority, and is constructed only through `create_truncate(ref CStr)`,
  `create_new(ref CStr)`, `open_or_create(ref CStr)`, `open_append(ref CStr)`,
  `open_read(ref CStr)`, or
  `open_update(ref CStr)`. The update
  constructor opens only an existing file with read/write access, does not create or truncate it,
  and starts at byte zero. The open-or-create constructor atomically opens or creates through one
  target operation, preserves existing contents, and also starts at byte zero.
  The create-new constructor atomically creates only a missing path as an empty write-only file at
  byte zero; an existing path returns status `1`, Windows code `80`, and completed bytes `0`
  without changing that file.
  The append constructor creates a missing file or preserves an existing file, grants write-only
  `FILE_APPEND_DATA`, and makes every underlying write begin at the then-current EOF regardless of
  a caller-selected position. `write(ref mut File, ref u8[])` performs at most one underlying
  write and returns its exact successful count, including a short count. `write_all` may contain
  multiple underlying writes and is not a transactional record append. The current share policy
  does not admit a concurrent writer.
  `remove_file(ref CStr)` removes one existing file path through exactly one target operation and
  returns zero on success. It treats a missing path as `FileError`, does not close or mutate a live
  File owner, and therefore reports the current Windows sharing violation until that owner is
  explicitly closed. It is not recursive and does not remove directories.
  `rename_file(ref CStr source, ref CStr destination)` changes one existing file path without
  replacing an existing destination or falling back to cross-volume copy-and-delete. It rejects an
  observed directory source, returns exact target failures, and does not close a live File owner.
  The attribute observation is not a handle-pinned identity or authorization check against external
  concurrent path replacement. Success does not claim a final-directory durability barrier.
  `copy_file_bytes(ref CStr source, ref CStr destination)` holds one exclusive source handle and
  one exclusive newly created destination handle, copies only bytes through a fixed 65,536-byte
  stack buffer, and returns the exact copied count. It validates both paths before effects and never
  replaces an existing destination. A later read, write, or close failure preserves the completed
  destination prefix reported in `FileError.completed`. It does not copy metadata, flush storage,
  publish atomically, follow a portable link policy, or claim directory-entry durability.
  `create_dir(ref CStr path)` creates exactly one final directory component without creating missing
  parents. `remove_dir(ref CStr path)` removes exactly one existing empty directory without
  enumerating or deleting children. Both return zero on success, preserve exact target failures,
  use the bounded ASCII path contract, and do not change the process current directory. Recursive
  deletion, directory iteration, explicit permission policy, and portable link semantics remain
  outside this slice.
  `create_dir_all(ref CStr path)` validates the complete lexical path before effects, rejects
  parent/self traversal plus ambiguous drive-relative and device-namespace paths, and creates each
  missing directory prefix. Existing directories succeed only when their observed kind is a
  directory. A later target failure preserves every prefix already created; no rollback,
  enumeration, deletion, ACL choice, or metadata-durability promise is implied.
  `rename_dir(ref CStr source, ref CStr destination)` changes one existing directory path without
  replacing an existing destination or falling back to cross-volume copy-and-delete. It rejects an
  observed non-directory source, preserves exact target failures, and moves a nonempty directory
  with its children without enumerating, copying, or deleting them. The source-kind observation is
  not a handle-pinned identity check, and success claims no final-directory durability barrier.
  `read(ref mut File, ref mut u8[])` returns one exact partial count including zero at EOF.
  `read_append(ref mut File, ref mut bytes::Buffer, usize maximum)` performs at most one read into
  Buffer spare capacity and increases its initialized length only by the successful completed
  count. EOF returns zero without changing length. Allocation failure precedes file I/O and
  preserves cursor, allocation, bytes, capacity, and length. File failure preserves bytes and
  length but may retain capacity grown by the successful reserve. `ReadAppendError` separates exact
  file status and Windows code from allocation status; only one failure domain is nonzero.
  `read_to_end(ref mut File, ref mut bytes::Buffer)` starts at the current cursor, appends through
  EOF in bounded 65,536-byte growth windows, and returns only the newly appended byte count. It
  preserves the preexisting Buffer prefix and every completed read if a later operation fails.
  Allocation for a new window occurs before its corresponding read, so allocation failure consumes
  no additional file byte. The helper adds no target symbol or runtime ABI surface; callers needing
  a total-size limit, cancellation, or incremental backpressure retain `read_append`.
  `read_exact(ref mut File, ref mut u8[])` loops on that same handle until it fills the complete
  buffer or fails. Unexpected EOF is status `7`, OS code `0`, and the exact initialized prefix in
  `completed`; the unread suffix keeps its prior bytes. Empty input on an open File succeeds with
  zero. `write_all`, explicit `sync`, consuming `close`, and automatic drop preserve the
  closed target component's ownership rules. `position(ref mut File)` observes the absolute byte
  cursor and `seek_start(ref mut File, usize)` moves it from the file start; both return the resulting
  position and reject offsets above `INT64_MAX` without moving the cursor. Bounded
  `seek_current(ref mut File, i32)` and `seek_end(ref mut File, i32)` reuse the same live handle and
  return an absolute `usize`; a negative resulting position fails without cursor movement, and an
  EOF-relative position beyond current size does not extend the file by itself. `size(ref File)` observes
  the exact byte length through a shared owner borrow without changing position.
  `kind(ref File)` observes the already-open object without path lookup, allocation, or cursor
  movement and returns target-neutral public enum `FileKind` with exhaustive `.Unknown`, `.Disk`,
  `.Character`, and `.Pipe` variants. `kind_is_disk`, `kind_is_character`, `kind_is_pipe`, and
  `kind_is_unknown` remain compatibility helpers; neither surface exposes Win32 constants.
  `is_read_only(ref File)` observes `FILE_ATTRIBUTE_READONLY` through the same live handle and
  returns target-neutral `bool` without moving the cursor, changing length, allocating, retaining a
  pointer, or reopening a path. It reports metadata rather than writable-access authority: handle
  access, ACLs, sharing, filesystem policy, and later external changes remain independent.
  `path_kind(ref CStr)` observes one bounded path and returns exhaustive public `PathKind` variants
  `.File`, `.Directory`, or `.Other`. Device and reparse attributes map to `.Other`; missing or
  inaccessible names retain exact `FileError`. The result is a momentary name snapshot, not an
  opened-object identity, authorization or containment decision, or race-free precondition for a
  later operation.
  `file_id(ref File)` observes the same open handle and returns opaque `FileId` containing the
  complete Windows volume serial and 128-bit target identifier. `file_id_equal(ref FileId,
  ref FileId)` compares every bit without consuming either value. The identity is bounded to one
  computer and observation interval; it is not serialized, ordered, hashed, persisted, or accepted
  as authorization, and filesystem providers may reject or later reuse/change identifiers.
  `set_len(ref mut File, usize)` truncates or extends through an exclusive borrow, returns the new
  length, rejects values above `INT64_MAX`, and does not move the cursor. Bytes added by extension
  have unspecified contents until explicitly written. `FileError` exposes
  stable status, Windows OS code, and completed-byte accessors. The checked-in safe quickstart
  executes invalid-path and missing-update failure, missing-file open-or-create plus preserving
  second open, exclusive create plus exact collision and unchanged reopen, missing-file append plus
  repeated seek-resistant exact concatenation, write/read/EOF, complete and unexpected-EOF exact reads,
  same-file live-handle identity equality and distinct-live-file inequality,
  position/seek including overflow preservation, existing-file in-place
  update, shrink/extend with cursor preservation and explicit rewrite,
  oversized and read-only length-change failures, nonempty and empty size observation, disk and
  `NUL` character kinds, ordinary/read-only observation and same-handle read-only mutation plus host-created two-name hard-link
  counts and filesystem-reported allocation length with cursor/size preservation, no-replace hard-link creation with collision and
  missing-source preservation, shared identity and bytes, and a surviving destination alias,
  owned-Buffer prefix append, short read, EOF, write-only failure,
  forced growth-allocation failure with zero retained allocations, explicit close, implicit-drop
  reopen, exact file removal, no-replace file rename, one-level directory creation and removal,
  recursive parent creation with idempotency and partial-prefix preservation, no-replace nonempty
  directory rename, old-path disappearance, and moved-child preservation under both Windows
  profiles. The path is
  still bounded ASCII bootstrap input; portable Unicode paths and Linux path/metadata extensions
  remain excluded.
  ADR 0133 strengthens the existing `arenaExecutions` success condition with deterministic
  first-stage and second-stage allocation failure plus live-allocation cleanup. ADR 0134 strengthens
  `bufferExecutions` with deterministic construction and three growth-allocation failures, exact
  storage preservation, recovery zeroing, and live-allocation cleanup. Neither adds a report field
  or counter, so they create no report revision.
  Its closed
  `drop-trace-v0` emission profile assigns source/type/span identities to dense site IDs and compares
  exact four-byte native records with the MIR interpreter trace. This instrumentation is absent from
  product `emit-c` and `build`. ADR 0214 records the first hosted success; broader target
  conformance does not follow from either the command or that bounded receipt.
- The experimental target-inspection command emits that target description without probing the
  host. It reports `scalar-and-flat-copy-resource-tagged-c11` separately while keeping `emittable`,
  `conformant`, and `supported` false and listing the evidence still missing.
- The stable top-level `targets` human command renders every validated TargetSpec in canonical ID
  order with its schema, admitted emission subset, independent maturity states, and sorted missing
  evidence. It performs no host or toolchain probe. The Windows entry reports emission subset
  `scalar-and-llp64-c-scalar-c11` and remains independent from the target-neutral MSVC host smoke.
- The experimental build command accepts the shared bounded explicit FY module closure and uses the same bounded C driver and embedded scalar
  runtime to produce a no-overwrite executable below `build/` on an eligible host. This artifact
  path does not expand the constrained backend subset, establish stable build syntax, or change target
  support status.
- The Windows/amd64 `run-host` and `build-host` smokes separately compile one explicit target-neutral
  verified FY module closure with a locally discovered x64 MSVC toolchain. `run-host` executes a temporary PE and reports
  the `i32` result or canonical trap; `build-host` retains one no-overwrite `.exe` below `build/`.
  They use no WSL and reject foreign symbols, standard-library linkage, `@repr(C)`, exact `c::*`
  scalars, and foreign calls or results. They do not establish Windows target emission, ABI,
  supported distribution artifacts, conformance, or self-hosting.
- The distinct `experimental build-target` path selects `x86_64-windows-msvc` before C emission.
  It accepts one entry FY file plus a bounded explicit repeated `--source` module list. All paths
  remain under the current source root, duplicate normalized paths and module/path drift fail, and
  no source discovery or manifest lookup occurs.
  Its backend profile consumes Windows LLP64 scalar providers, emits the target ID in the C unit,
  admits `c::long` as four-byte signed C `long`, and rejects unavailable POSIX `c::ssize_t`.
  `examples/windows_target/main.fy` carries an unused `c::long` declaration so local x64 MSVC must
  compile the target assertion, constructs and reads one move-only resource containing an exact
  pointer-width raw handle plus `u32` state, then calls exact project-owned SystemHeap allocate/deallocate,
  Buffer initialize/zeroed-resize/drop, Arena initialize/Region begin/zeroed allocation/reset/Arena
  drop, CStr validation, String copy/view/append/drop, CString from-text/view/drop, and
  `fy_std_text_utf8_valid_v0(ptr u8, usize) -> u32` Fairy foreign declaration backed by the C
  implementations before the generated PE can return 42. The heap crossing requires an eight-byte
  block, observes its size and alignment, frees it, and observes normalized zero state. The Buffer
  crossing selects the Buffer-to-SystemHeap dependency closure, observes capacity eight and length
  four, drops it, and observes normalized zero state. The Arena crossing selects Arena-to-SystemHeap
  closure, observes initialization generation 1, one active Region and non-null allocation/node,
  reset generation 2 with invalidated Region state, and drop generation 3. Private Windows generation
  fields use target-width `usize` only as a layout-compatible transport, not a public representation.
  Neither Buffer nor Arena reads the zeroed bytes on this FY path. Separate raw Buffer and Arena
  probes supply Windows byte-level evidence, while ADR 0255, ADR 0256, and ADR 0257 execute
  generated-Fairy Arena, Buffer, and String.append allocation-failure `.err` arms at both MSVC
  profiles. Owned text copies a compiler-owned
  five-byte UTF-8 literal, observes String length five then ten after append, normalized drop,
  six-byte CString storage with logical CStr length five, normalized drop, and direct validation of
  compiler-owned terminated storage. It observes only status, length, alignment, and pointer state,
  not returned bytes. The checked-in example links only SystemHeap, Buffer, Arena, text view, owned
  text, and the scalar runtime. The closed Windows resolver additionally admits six target-bound
  raw File operations: open, partial read, write-all, sync, consuming close, and best-effort drop.
  Their independent C component accepts a bounded ASCII bootstrap path, captures Win32 errors in
  its result record, uses no hidden allocation, and is selected only when an admitted call is
  present. A separate-translation-unit native probe exercises normal lifecycle, three-byte reads,
  EOF, repeated close, forced invalid-handle close failure, and repeated drop at both local MSVC
  profiles after COFF, PE, and unwind inspection. A second checked-in two-module FY example now
  wraps this raw component in an effect-checked move-only File, typed errors, `read(ref mut u8[])`
  backed by fixed-array storage, consuming close, and implicit drop, then returns 42 at both MSVC profiles. This is an
  example-level Stage 0 boundary rather than a stable public standard-library package. The
  SystemHeap path fixes its fundamental-alignment ceiling at the documented
  Windows x64 `malloc` guarantee of 16 bytes because MSVC C does not provide `max_align_t` in
  `<stddef.h>`. Every other standard-library or foreign symbol fails before toolchain resolution;
  foreign companion inputs and user linker arguments remain absent. Copied byte placement,
  source-level FY observation of owned-text byte/terminator contents, portable `std::fs` paths and
  linkage, general mutable buffers, and stable layouts remain absent. This clears only the missing target
  C-emission-profile evidence; `emittable`, `conformant`, and `supported` remain false.
- `experimental run-target` reuses that exact Windows target compilation and closed linkage, validates
  the temporary PE plus Microsoft x64 unwind structure, executes it without WSL, rejects output, and
  reports either exact signed `i32` result bits or the canonical product-trap category. It accepts no
  output path, compiler selection, foreign companion input, or linker option and retains no target artifact.
  Direct local execution is additional product-path evidence, not hosted conformance or support.
- `experimental exec-target` uses the same validated temporary Windows target but forwards its
  separated stdout and stderr byte sequences without a compiler receipt and preserves the full
  32-bit program status. One target-only unsafe
  `fy_std_windows_stdout_write_all_v0(ptr u8, usize) -> { status: u32, written: usize }` call loops over
  `WriteFile`, reports partial progress on failure, normalizes empty input, rejects nonempty null
  input, and is admitted only by the Windows support resolver. The raw declaration, safe local
  `ref str` helper, and `main` each carry `@effects(console)`, while the independent `unsafe` boundary
  remains confined to the support module. The wrapper validates status and progress and keeps raw
  pointers, foreign calls, and `unsafe` out of `main`. A second pure `ref u8[]` helper proves byte-view parameter
  transport. This is executable process I/O and local unsafe containment, not
  a safe or portable `std::io` package, Unicode console policy, stdin, stderr writing, arguments, or
  target maturity evidence.
- QBE or another fast backend and LLVM require separate measured adoption decisions after MIR stability.
- Early caching is whole-package and bounded. Function-level caching requires measured invalidation or latency pressure.
- Development output contains intentional deliverables and small state, not an unbounded copy of every intermediate.

Source-linked MIR normalization assertions are versioned in `spec/fixtures/mir/manifest-v0.json`. They freeze ordered evaluation, move and cleanup behavior, `Result` branching, call successors, trap paths, and eliminated source-only forms without making textual MIR or internal graph identity public.

MIR verifier rejection recipes are versioned in `spec/fixtures/mir-verifier/manifest-v0.json`. They require a verified named seed, exactly one typed mutation, and exactly one expected `Vxxxx` result without creating a general serialized MIR or plugin format.

Resolved package graph assertions are versioned in `spec/fixtures/package/manifest-v1.json`, which includes all eight historical v0 cases and adds deterministic package-cycle rejection. They fix package-instance equality, one source root per instance, acyclic direct dependency structure, direct-only dependency visibility, module-provider ambiguity, and explicit standard-library injection without choosing public manifest, lockfile, coordinate, registry, or solver syntax. `internal/packagefixture` executes all nine cases through the reusable `internal/packagegraph` validator and resolver, converts passing graphs into build-local semantic package input, and proves isolated equal module paths complete semantic Flow. Region v1 consumes the same validated graph result. The independently versioned `spec/fixtures/package-identity/manifest-v1.json` freezes the canonical resolved-tuple byte encoding, full SHA-256 `PackageInstanceId`, text form, component boundaries, and feature-set rejection reasons without choosing any package-manager format. `PackageCoordinateId` v1 separately identifies the source-independent UID/version/feature coordinate used by public cross-package contracts. Typed HIR retains source `pub` state for every current item category and may carry one source-independent contract-state layout per package. `spec/fixtures/module-interface/manifest-v5.json` compiles thirteen current source variants and freezes the coordinate-based public-interface v5 digest: it preserves every v4 item relation and adds the exact package-owned state-layout hash. V1 through v4 manifests remain frozen historical vectors. `spec/fixtures/package-source/manifest-v1.json` separately freezes the exact path-and-source-byte digest used for the current package's own implementation invalidation. Historical `spec/fixtures/package-cache/manifest-v1.json` records the first caller-bounded process-local LRU key baseline.

ADR 0349's v4 key encoding uses current vectors in
`spec/fixtures/package-cache/manifest-v8.json`. The current package keeps exact source identity,
its complete public-interface v5 hash, and exact contract-state body identity; dependency pairs use source-independent coordinates plus
public-interface v5, including public must-use, resource abilities, and contract-state layout
identity. Stable function,
nominal type, foreign, body, and package artifacts use the same coordinate boundary and still
re-enter MIR verification, so a private dependency edit can reuse the caller without accepting an
old provider body. Historical cache fixture manifests v1 through v7 remain frozen.

ADR 0350 separately defines a bounded canonical contract-state body payload over the same coordinate
and body hash. Its decoder reconstructs layouts and operations, creates fresh private capabilities,
re-enters the transaction-MIR verifier, and recomputes the hash. ADR 0351 binds it to an exact
expected hash, current HIR layout, and fresh package/function symbols. ADR 0352 then pairs it with
stable package MIR v3 in a separate outer artifact. ADR 0353 adds cache storage, expected-body
lookup, and exact-generation quarantine. ADR 0354 adds state-body rebind, combined fresh-symbol
reassembly, and explicit cached oracle execution. ADR 0355 unifies cold and cached transaction
publication in `compiler.Result`, and ADR 0356 closes accepted authority to represented storage
effects. ADR 0357 closes unrepresented owner-body calls. ADR 0379 admits only the bounded
same-package no-argument void helper graph while public source state operations, general call
admission, and native/runtime admission remain closed. ADR 0358 closes current ordinary owner-body dynamic control flow. ADR 0378
admits compiler-private typed state values and bounded acyclic conditional blocks with v2 stable
transport while source state access and ordinary conditional lowering remain absent. ADR 0359 closes unsafe
and raw-pointer owner transport while safe borrows remain ordinary HIR facts rather than state
permissions. ADR 0360 closes receiver and drop-glue ownership while retaining explicit static
associated functions. ADR 0361 closes checked-arithmetic and bounds trap edges in current owners.
ADR 0362 closes owned-resource drop/burn flow and lexical region cleanup while preserving safe
borrows and static resource namespaces.
ADR 0363 closes the owner HIR world to an explicit positive statement, expression, and operator
allowlist.
ADR 0364 versions the first five transaction execution cases and requires them to pass through
source-independent HIR lowering, verified contract-state MIR, and the failure-atomic oracle. The
manifest is an admission corpus, not a source, runtime, or persistence format.
ADR 0365 preserves the v0 cell corpus and extends the current v1 manifest with map write/read,
delete, exact domain-separated storage-key, and encoded-key-limit rollback evidence.
ADR 0366 requires both fixture generations to pass through `compiler.LowerAnalyzed` and treats only
one verified atomic ordinary/state MIR result as executable. Direct HIR fixture construction remains
compiler test input, not a source-language contract.
ADR 0367 separately versions overlapping semantic-oracle schedules. A stale commit returns conflict,
publishes none of its overlay, and leaves the winning store version unchanged; a transaction reads
the snapshot captured at begin, while a later begin observes the latest committed version. These
runtime facts do not erase verified compiler IR.
ADR 0368 applies the same version boundary to maps. Overlapping transactions that write distinct map
keys still conflict when their base store version is stale; open readers retain their captured map
snapshot, and a fresh transaction observes the latest committed keys.

ADR 0143 fixes the first human project file as strict `fairy.toml` schema
`fairyc.manifest/v1`. It requires explicit package UID, kebab name, exact SemVer, exact language
revision, and source root; optional workspaces list sorted explicit members; local path dependencies
require exact versions and explicit sorted per-edge features. Eight complete-file fixtures cover
minimal/workspace success and closed-field, duplicate-key, missing-field, path, ordering, and schema
rejection. The parser performs no filesystem access and does not yet construct the resolved graph or
write a lockfile.

ADR 0144 separately fixes the first filesystem consumer of that manifest. It loads only one
caller-selected root plus explicit nonoverlapping members, rejects nested workspaces, symlinks,
nonexact path case, duplicate package UIDs, language/version mismatches, source collisions, invalid
sources, and dependency cycles, and uses package-source digest v1 as local source identity. Per-edge
features remain distinct and only root-reachable instances enter the resolved graph.

ADR 0145 fixes generated `fairy.lock` as canonical JSON. It records root-reachable local and SDK
instance tuples and edges, recomputes every PackageInstanceId and source digest, detects workspace
staleness, and atomically replaces only canonical validated bytes.

`experimental check-workspace` was the first end-to-end consumer: it assigns build-local package
IDs, binds resolved imports, runs package-aware semantic, Flow, MIR lowering, and verification, and
either refreshes the canonical lock or requires an exact no-write `--locked` match. ADR 0175 exposes
the same current-workspace boundary as stable `fairyc check [--locked]` with a versioned narrow
argument, output, and exit-code contract. It emits no native code.

ADR 0147 fixes the first real SDK selection: `std` and `std::*` imports receive one direct
compiler-bundled `fairyc.std` package with exact SDK identity, language revision, runtime ABI,
version, source digest, and lock fields. SDK `0.0.0-stage0.56` exports ADR 0250's move-only,
fallibly created `std::alloc::Arena` with automatic cleanup and exact allocation status,
while ADR 0251 makes compiler-known placement return `Result<ref T, AllocError>` under language `/8`.
ADR 0246's public safe `std::bytes::Buffer`, ADR 0299's fallible single-byte append, and ADR 0373's
canonical `i32`, `u32`, `usize`, and `bool` text append operations,
ADR 0247's public safe `std::text::String` and `CString`, ADR 0248's allocation-free
`std::text::validate_utf8`, ADR 0302's allocation-free immutable `std::text::as_bytes`, ADR 0263's
zero-copy `std::text::from_buffer`, ADR 0264's Windows MSVC
`std::fs::File` ownership and byte-I/O surface, ADR 0331's target-neutral File core, ADR 0269's
original absolute position/seek surface and ADR 0332's target-neutral linkage for it, ADR 0270's
logical size surface and ADR 0333's target-neutral linkage for it, ADR 0305's bounded current- and
end-relative seek surface and ADR 0334's target-neutral linkage for it, ADR 0306's same-handle
read-only metadata, ADR 0307's
same-handle hard-link count, ADR 0308's no-replace hard-link creation, ADR 0309's same-handle
allocated size, ADR 0310's same-handle read-only mutation, ADR 0311's bounded recursive directory
removal with exact completed-object progress, ADR 0270's shared file-size
observation, ADR 0271's existing-file read/write open, ADR 0272's cursor-preserving
file-length change, ADR 0274's non-truncating open-or-create, ADR 0275's exclusive create,
ADR 0276's append-only open, ADR 0277's same-handle file-kind observation, ADR 0278's exact read,
ADR 0279's bounded same-handle file identity, ADR 0281's exhaustive public file-kind enum,
ADR 0285's one-call partial file write, ADR 0288's bounded owned-Buffer file read append, ADR 0290's
source-level read from the current cursor through EOF, ADR 0291's exact one-file removal, ADR 0292's
file-only no-replace rename without cross-volume copy fallback, ADR 0293's one-level directory
creation and exact empty-directory removal, ADR 0294's directory-only no-replace rename, ADR 0296's
pre-effect validation and recursive parent creation with explicit partial-prefix preservation,
ADR 0297's exclusive no-replace file byte copy with explicit completed-prefix preservation, ADR
0298's exhaustive path-kind snapshot with explicit TOCTOU limits,
ADR 0205's `std::io::write(ref str) -> Result<usize, ConsoleError>`, and ADR 0209's matching
`write_stderr` operation, ADR 0210's matching immutable byte-view output operations, and ADR 0211's
`read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>`, plus ADR 0215's allocation-free
canonical decimal `write_i32`, ADR 0262's allocation-free decimal `write_u32`/`write_usize` and
ASCII `write_bool`, ADR 0303's matching scalar stderr output, and ADR 0261's effect-free
`std::process` argument surface; it
performs no host installation lookup. The I/O wrapper keeps raw
pointers and target linkage private while the remaining planned
standard-library API remains unsettled.

ADR 0305 adds allocation-free `std::fs::seek_current(ref mut File, i32)` and
`std::fs::seek_end(ref mut File, i32)` on the existing Windows File owner. Each operation returns
the resulting absolute `usize` position through the existing result layout and performs no path
lookup or ownership transfer. A negative resulting position fails with exact target error storage
and an unchanged cursor. Moving beyond EOF does not change file size until a later write or
`set_len`. The `i32` delta is intentionally bounded; wider signed offsets, sparse-gap promises,
portable offsets, and Linux public linkage remain separate gates.

ADR 0306 adds allocation-free
`std::fs::is_read_only(ref File) -> Result<bool, FileError>` on the existing Windows File owner.
It calls `GetFileInformationByHandleEx(FileBasicInfo)` through the same live handle and maps only
`FILE_ATTRIBUTE_READONLY` to public boolean meaning. Low and optimized raw plus generated-Fairy
paths cover ordinary/read-only files, closed and invalid storage, and cursor/size preservation.
Timestamps, atomic metadata records, attribute mutation, writable-access guarantees, ACLs, and
Linux public linkage remain separate gates.

ADR 0307 adds allocation-free
`std::fs::hard_link_count(ref File) -> Result<usize, FileError>` on the existing Windows File
owner. It calls `GetFileInformationByHandleEx(FileStandardInfo)` through the same live handle and
maps `NumberOfLinks` to target-neutral `usize`. Low and optimized raw plus generated-Fairy paths
cover a real one-to-two hard-link transition, both linked names, closed and invalid storage, and
cursor/size preservation. The result is a momentary count, not path enumeration, authorization, or
persistent identity. Hard-link mutation, allocation size, timestamps, atomic metadata records,
and Linux public linkage remain separate gates.

ADR 0308 adds
`std::fs::hard_link(ref CStr source, ref CStr destination) -> Result<i32, FileError>` for one
bounded Windows no-replace hard-link operation. Both bounded paths are validated before the target
effect, and `CreateHardLinkW` is called once without allocation, byte copying, destination
prechecks, or replacement. Low and optimized raw plus generated-Fairy paths cover success, exact
collision and missing-source failures, shared file identity and bytes, link-count transition, and
destination survival after source-name removal. Cross-volume behavior, filesystem support,
authorization, reparse-point containment, race freedom, durability, and Linux public linkage remain
separate gates.

ADR 0309 adds allocation-free
`std::fs::allocated_size(ref File) -> Result<usize, FileError>` on the existing Windows File owner.
It queries nonnegative `FileStandardInfo.AllocationSize` through the same live handle. Low and
optimized raw plus generated-Fairy paths cover ordinary, read-only, and two-link files, closed and
invalid storage, and cursor/logical-size preservation. The value is not portable physical usage,
block shape, compression, sparse-extent, quota, billing, or concurrent-stability evidence.

ADR 0310 adds
`std::fs::set_read_only(ref mut File, bool) -> Result<bool, FileError>` on the existing Windows File
owner. It reads `FileBasicInfo`, preserves unrelated attribute bits, passes zero timestamps, and
calls `SetFileInformationByHandle(FileBasicInfo)` through the same handle. Low and optimized raw
plus generated-Fairy paths cover true/false transitions, cursor/size preservation, closed and
invalid storage, invalid boolean rejection, and exact read-only-handle access denial with unchanged
metadata. The attribute is not an ACL, sharing, filesystem-policy, or future-write guarantee.

ADR 0311 adds
`std::fs::remove_dir_all(ref CStr) -> Result<usize, FileError>` for one bounded Windows recursive
removal. It rejects roots, trailing separators, dot components, non-ASCII input, and extended
namespaces before effects; observed directory reparse points are removed as leaves instead of
traversed. Low and optimized raw plus generated-Fairy paths cover a four-object tree, exact missing
and sharing failures, a read-only child, partial progress, preserved failure storage, and retry
cleanup. Sibling order is filesystem-defined, and the operation does not promise rollback,
atomicity, race-free containment, general Unicode paths, or Linux public linkage.

ADR 0148 adds the first package-aware native project path. `experimental build-workspace` and
`run-workspace` load the current `fairy.toml`, apply ordinary or exact no-write `--locked` policy,
select the root instance's build-local `main::main`, require `fn() -> i32`, and reuse the closed
`x86_64-windows-msvc` C emission, support-link, MSVC, PE, and unwind pipeline. The checked-in
two-package quickstart returns 42 under both build profiles, direct temporary execution, and a
separately built compiler process. The commands add no registry, solver, source download,
configurable entry, new backend capability, or persistent artifact format.

ADR 0176 promotes only the ordinary Windows-local build workflow to
`fairyc build [--release] [--locked]`. It derives one replace-on-success artifact path from the root
package name and hides prototype target, profile-value, and output flags. Experimental workspace
build/run remain available for explicit evidence and compatibility.

ADR 0177 independently promotes the ordinary temporary execution workflow to
`fairyc run [--release] [--locked]`. It separates the program's `i32` result from compiler status,
retains no image, preserves an existing stable-build artifact, and keeps program I/O, arguments,
target choice, cache policy, and target support outside the stable contract.

ADR 0207 supersedes that v1 result-receipt boundary with program-facing run v2. It forwards bounded
stdout/stderr, preserves ordinary 32-bit program status, classifies traps through a separate private control pipe,
keeps canonical trap reporting, and still admits no program arguments or stdin forwarding.

ADR 0212 supersedes run v2 with stable run v3. It attaches caller stdin only after compilation,
linking, and PE/unwind inspection succeed, performs no compiler-side pre-read or retention, preserves
the existing timeout/output/status/trap rules, and still admits no program arguments or input
forwarding for other native commands. ADR 0261 later adds program arguments after `--` to stable
run without changing the generated C entry.

ADR 0178 stabilizes the source-independent `fairyc explain <diagnostic-code>` lookup, its exact
three-line human shape, and codes 0, 2, and 4. It keeps catalog lifecycle, wording improvement,
diagnostic implementation and fixture status, JSON, and span compatibility as separate contracts.

ADR 0179 stabilizes the source-independent `fairyc targets` human report grammar, canonical ordering,
atomic failure, and codes 0, 2, and 4 without freezing TargetSpec rows, evidence, or maturity. The
report remains distinct from local toolchain availability and experimental target JSON.

ADR 0180 historically stabilized `fairyc emit c --target <target-id> [--locked]`. ADR 0335
supersedes that product-surface decision: the stable spelling now returns code 2 with a migration
direction, while `fairyc experimental emit-workspace-c --target <target-id> [--locked]` retains
explicit Linux GNU or Windows MSVC inspection of the private C11 bootstrap unit. Generated C text,
its private names, formatting, and whole-unit bytes are not cross-release product contracts.

ADR 0181 stabilizes `fairyc test [--locked]` as a root-package semantic lane. Ordinary production
loading and locking exclude `tests/`; the command overlays that exact contained directory in memory,
maps paths into the `tests::` module namespace, validates argument-free `@test` on private safe
zero-argument `void` functions, checks the combined package closure through verified MIR, and runs
each selected entry in a fresh deterministic interpreter state. ADR 0265 extends the spelling to
`fairyc test [--native [--release]] [--locked]`: native mode preserves the same discovery and report,
but appends one private verified i32 harness per test, compiles its reachable closure into an
inspected temporary Windows MSVC image, and runs every test in a fresh process and directory. Test
traps continue to later tests in either engine. ADR 0267 adds repeatable exact canonical test-
identity selection to both engines without hiding syntax or compiler failures in the complete
overlay. ADR 0268 adds `fairyc test --list [--locked]`, which exposes those same canonical
identities after complete compiler admission without executing a test or target toolchain. ADR 0273
adds `--json` for the versioned identity array, count, and lock state without changing discovery or
execution. ADR 0287 additionally admits execution-mode `--json` and freezes
`fairyc.test-report/v1` with canonical semantic/native outcomes, engine/target/profile identity,
totals, status, and lock state while preserving text bytes and exit codes. Catalog-wide member recursion,
fuzzy/tag/glob filters, setup/teardown, portable native targets,
shared or replayed multi-test input, live native-test output streaming, benchmarks, and host-environment
policy remain separate contracts.

ADR 0289 adds singular `--package <package-uid>` selection to execution and listing. The UID must
name one local package with exactly one root-reachable resolved instance; only that package's
contained `tests/` tree is overlaid. Existing invocations remain root-only and byte-for-byte
unchanged. ADR 0295 adds mutually exclusive `--workspace` aggregation over every unambiguous
root-reachable local package in the production graph. Only that mode qualifies canonical identities
as `<package-uid>::<module>::<function>`, accepts those exact strings through `--exact`, and emits
`fairyc.test-list/v2` or `fairyc.test-report/v2` for JSON while preserving the v1 field sets.
Catalog-only unreachable members, ambiguous feature instances, parallel scheduling, caching,
durations, retries, live output streaming, and richer attachments remain separate.

ADR 0300 adds explicit `--locations` to listing and execution. It preserves every existing
identity, selector, text byte, and v1/v2 JSON byte when omitted. When requested, text reports a
portable workspace-relative path with 1-based line and byte column; JSON uses
`fairyc.test-list/v3` or `fairyc.test-report/v3` and binds each function-name span to package UID,
exact byte offsets, `fairyc.source-file/v1`, and `fairyc.source-line-map/v1`. These anchors come
from the validated in-memory test source and disclose neither source text nor an absolute host path.
Consumers must verify both identities before trusting a retained span as current.

ADR 0301 adds repeatable `--file <portable-test-path>` to execution and listing. The selector uses
the same exact portable workspace-relative path domain as `--locations`; member paths include their
workspace prefix and matching is case- and separator-sensitive. Distinct values form a union;
missing, duplicate, or unknown values fail as code 2, while an admitted file with zero tests
succeeds. `--file` is mutually exclusive with `--exact`. Selection limits only listing or execution:
the complete selected test overlay still passes declaration, semantic, Flow, MIR, and verifier
admission. Existing text and v1/v2/v3 JSON bytes remain unchanged.

ADR 0304 adds repeatable `--dir <portable-test-directory>` to execution and listing. Selectable
directories are derived only from admitted portable test-source paths and extend from each file's
parent to that package's exact `tests` root, never above it. Matching is byte-exact, case-sensitive,
and `/`-only; recursive membership requires the selected directory plus one `/` boundary. Distinct
values form a union, an admitted directory containing zero tests succeeds, and missing, duplicate,
host-separated, dot-segment, trailing-slash, or otherwise uncataloged values fail as code 2.
`--dir` is mutually exclusive with `--file` and `--exact`. Selection performs no filesystem walk,
does not weaken complete-overlay compiler admission, and leaves text and v1/v2/v3 JSON bytes
unchanged.

ADR 0312 advances only native execution to
`fairyc test --native [--release] ... -- <program-args...>`. The first delimiter is excluded and the
remaining values are forwarded unchanged to every selected isolated test process. Flag-looking,
empty, and non-ASCII UTF-8 arguments are preserved; embedded NUL is rejected before workspace
loading. Semantic execution and listing reject the delimiter. Discovery, selection, lock behavior,
text and JSON report bytes, schemas, SDK, language, runtime ABI, and generated C remain unchanged.

ADR 0313 advances native execution to explicit
`fairyc test --native [--release] --stdin ...` only when selection resolves to exactly one test after
complete overlay compiler admission. The caller-owned reader is attached directly after compilation,
linking, and PE/unwind inspection without compiler-side pre-reading, buffering, replay, retention,
decoding, or a separate byte limit. Native tests retain EOF by default; semantic/list execution
rejects the flag, and zero or multiple selections fail as code 2 without consuming input or launching
a process. Program arguments, output rejection, text/JSON report bytes and schemas, SDK, language,
runtime ABI, and generated C remain unchanged.

ADR 0314 advances native JSON execution to explicit `--capture-output` and
`fairyc.test-report/v4`. Each canonical outcome requires padded base64 stdout/stderr so arbitrary
binary and pre-trap bytes survive without UTF-8 decoding; recognized trap classification uses a
separate private control pipe and leaves stderr unchanged. The flag requires `--native --json`, composes with optional
`--locations`, and preserves the shared bounded-capture ceiling without claiming cross-stream order.
Without it, native program output remains code-2 rejection and all text/v1-v3 JSON bytes remain
unchanged. SDK, language, runtime ABI, and generated C remain unchanged.

ADR 0315 adds native-only `--stdin-file <portable-workspace-file>`. After complete compiler
admission, the compiler opens one canonical workspace-contained regular file, rejects inputs above
65,536 bytes, and gives every selected isolated process a fresh byte-zero reader. It does not combine
with caller-owned `--stdin` or listing; file failures launch no process and reports remain unchanged.

ADR 0316 adds native-only canonical `--timeout-seconds <1..60>`. The selected duration bounds each
test process after compilation and PE inspection; omission preserves 60 seconds. Timeout is code 6,
emits no completed report, stops later execution, and retains no artifact.

ADR 0317 advances the compiler-bundled SDK to `0.0.0-stage0.46` with effect-free
`path_kind_is_file` and `path_kind_is_directory`. Both exhaust `PathKind`, return false for `.Other`,
add no C symbol, and preserve the path snapshot's race and authorization limits.

ADR 0318 advances the compiler-bundled SDK to `0.0.0-stage0.47` with allocation-free
`std::bytes::empty()` and `std::text::into_buffer(String) -> bytes::Buffer`. The transfer preserves
the exact allocation and initialized bytes without copying, canonicalizes the consumed String, and
returns mutable bytes whose UTF-8 validity must be re-established before conversion back to String.

ADR 0319 advances the compiler-bundled SDK to `0.0.0-stage0.48` with public
`std::fs::FileMetadata` and `std::fs::metadata(ref File)`. The function observes the existing live
handle sequentially through size, allocated size, hard-link count, kind, and read-only operations,
returning the first error and adding no target symbol. The C backend now lays out struct and enum
by-value dependencies together and emits their complete definitions topologically, preserving
enum-containing private-struct construction, field reads, and switches. The result is not an atomic
filesystem snapshot, and enum fields in `@repr(C)` structs remain closed.

ADR 0331 advances the compiler-bundled SDK to `0.0.0-stage0.49` while retaining the public File
signatures. The create/open family, partial read/write, write-all, sync, consuming close, and drop
now call target-neutral private symbols selected by the closed product linker. Linux uses bounded
NUL-terminated CStr bytes, ordinary POSIX lookup, `0666` creation subject to umask, kernel append,
exclusive create, and a descriptor-plus-one private token. Windows delegates to its existing HANDLE
implementation. Linux path mutation, metadata, seeking, exact-read, Buffer-read, Unicode path, and
containment promises remain absent.

ADR 0332 advances the compiler-bundled SDK to `0.0.0-stage0.50` while retaining the public
`position` and `seek_start` signatures. Both now select target-neutral private symbols. Linux uses
`lseek` on the already-open descriptor, retries `EINTR`, and rejects an absolute offset above
`INT64_MAX` before effects; Windows delegates to the existing same-HANDLE operations. Relative/end
seek, size, metadata, exact-read, Buffer-read, path mutation, Unicode path, and containment promises
remain absent on Linux.

ADR 0333 advances the compiler-bundled SDK to `0.0.0-stage0.51` while retaining the public `size`
signature. It now selects a target-neutral private symbol. Linux calls `fstat` on the already-open
descriptor, retries `EINTR`, and rejects a negative kernel size as `EOVERFLOW`; Windows delegates to
the existing same-HANDLE operation. Success returns the current logical byte length without moving
the cursor. Relative/end seek, other metadata, exact-read, Buffer-read, path mutation, Unicode path,
containment, and atomic external-mutation snapshot promises remain absent on Linux.

ADR 0334 advances the compiler-bundled SDK to `0.0.0-stage0.52` while retaining the public
`seek_current` and `seek_end` signatures. Both now select target-neutral private symbols. Linux uses
`lseek` with `SEEK_CUR` or `SEEK_END` on the already-open descriptor and retries `EINTR`; Windows
delegates to the existing same-HANDLE operations. The `i32` delta remains deliberately bounded, and
movement beyond EOF changes only the cursor. Other metadata, exact-read, Buffer-read, path mutation,
Unicode path, containment, and atomic external-mutation snapshot promises remain absent on Linux.

ADR 0320 keeps language `fairyc-language/10`, SDK `0.0.0-stage0.48`, and runtime ABI
`fairyc-runtime/0` while fixing the constrained C emitter's verbatim-name boundary. Public C tags,
members, and external symbols now reject their applicable implementation-reserved forms,
generated-header macros, and backend-owned names before emission. `_lowercase` remains valid for a
C member but not for a file-scope tag or symbol; the closed `fy_std_` paths remain intact.

ADR 0146 rejects speculative persistent caching at this point. ADR 0149 supplies deterministic,
bounded, verifier-checked bytes for a complete MIR program, but the workspace path still produces
no independently reusable per-package MIR/object artifact and has no configured comparable
cold/warm measurement. Disk format, defaults, and cleanup policy therefore remain `UNDECIDED`.

The resolved-graph paragraph's separation from public manifest syntax describes fixture ownership,
not current product status: ADR 0143 owns syntax, ADR 0144 owns local workspace I/O, ADR 0145 owns
generated lock bytes, ADR 0147 owns current SDK selection, and ADR 0148 owns the experimental
Windows project commands. Stable public command policy, registry, and solver behavior remain
outside those fixture contracts.

Target layout assertions are versioned in `spec/fixtures/layout/manifest-v0.json` for Linux and
`spec/fixtures/layout/manifest-windows-msvc-v0.json` for Windows. They fix the closed layout input
vocabulary, size/alignment arithmetic, declaration-order offsets, top-level padding, and failure
codes without creating a serialized compiler IR, source-level aggregate promise, or native ABI
receipt. Linux pass cases feed the configured Clang/GCC native probe. The eight Windows pass cases
feed a separate x64 MSVC C11 probe at low and optimized profiles. ADR 0214 records its first hosted
execution, but this is not complete Windows layout coverage.

Formatting, rendered/general documentation generation, vendoring, binding generation, and ABI
inspection remain planned tool surfaces rather than claims of current implementation. The narrow
Atlas exact declaration-documentation extractor described below is not a documentation generator.
Testing has the bounded stable root semantic
surface above but no native, recursive, filtered, benchmark, or fixture runner. Checking has the bounded stable
current-workspace `fairyc check [--locked]` surface; diagnostic explanation has its independent
stable source-free surface; private target-explicit workspace C inspection is experimental.

## Compatibility

Compiler release, language revision, internal MIR, target schema, ABI profile, diagnostic catalog and JSON, runtime ABI, standard library, package schemas, `PackageInstanceId`, `.fymap`, `SymbolId`, interface hashes, Context Pack, attributed text, Symbol Catalog, and Evidence Bundle are independently versioned contracts. Pre-1 source and runtime contracts use exact compatibility. MIR and caches are disposable and are regenerated rather than migrated.

No v0 document creates a stable language, native ABI, MIR exchange, binary package, diagnostic-span, debugger, anchor, or `SymbolId` promise. The compatibility vector and failure behavior are defined in `docs/compatibility/versioning-v0.md`.

## Fairy Atlas

The compiler analysis layer now produces an immutable `SemanticSnapshot` from typed HIR, semantic
facts, verified MIR, and exact package metadata. It removes disposable compiler identities, retains
structured types and source-relative spans, and reports relation-specific completeness. Atlas
consumes that snapshot through a one-way boundary. The compiler frontend cannot import Atlas, and
Atlas failure cannot change `fairyc check` results.

Validated workspace source bytes now produce independent `fairyc.source-file/v1` exact-content and
`fairyc.source-line-map/v1` byte-geometry identities. Snapshot file records retain those identities,
package instance, canonical relative path, and source/test kind while excluding source text and host
paths. Construction requires a one-to-one file/module match. The implemented map-store reader
recomputes them through the validated workspace boundary before trusting stored spans; source
and documentation extraction instead uses those exact identities to revalidate already retained
workspace bytes without reopening a path.

The first snapshot consumer now derives provisional `SymbolId` values with the independently
versioned `fairyc.symbol-id/v1-sha256-128+256` algorithm. Package UID, module, declaration kind and
path, zero generic arity, and a canonical structured signature form the hash input. Package
instances, versions, source positions, bodies, build-local numbers, compiler identity, target, and
language revision do not. Nominal signature types use package UID and logical path; exact instance
compatibility now lives in the map header. ADR 0225 joins exact resolved calls separately, while
other dependency graph-edge families and query traversal remain future work. Each visible 128-bit prefix is
stored with its full 256-bit digest, and a mismatch or collision rejects the symbol set.

The next in-memory boundary now produces one immutable `fairyc.fymap-logical/v1` partition for every
exact workspace package. It requires the caller-provided compiler executable identity and one
independently computed public-interface hash per package, records each identity algorithm separately,
assigns every snapshot file, module, declaration, SymbolId, and function summary to its owning
package, and exposes partial or unavailable relations as stable omissions. The implemented plan is
target-independent and performs no filesystem, executable discovery, transport encoding, or map
publication.

Transport schema `fairyc.fymap/v1` now encodes each logical partition as one caller-bounded compact
canonical `fairyc.fymap-json/v1` object. `fairyc.fymap-checksum/v1-sha256` binds the exact schema,
encoding, and logical document. Readers reject unknown fields, trailing or noncanonical JSON,
malformed or zero identities, contradictory counts and ownership, excessive type nesting, invalid
completeness, and declarations whose stored SymbolIds do not rederive through the exact self/direct-
dependency package namespace references. The checksum detects corruption; it does not authenticate
the author or grant authority.

The internal opt-in `MapStore` now publishes canonical maps as checksum-named immutable generations
under an opened filesystem root. Every store requires caller-selected generation, total-byte,
per-map, and temporary-age bounds. Readers quarantine corrupt maps, reject multiple compatible
generations, and require the exact compiler build, public-interface hash, current package identity,
aggregate source digest, and every file content/line-map identity before returning spans.
Interruption before or after publication yields a miss or one fully verified hit. No default path,
ordinary-command writer, public CLI, watcher, daemon, or background cleanup is selected.

The internal query layer now requires that same exact current workspace/compiler/interface vector
before indexing a partition. It accepts only canonical visible or full SymbolId v1 selectors and
implements package-local `overview`, `show`, and deterministic breadth-first `around`. Every request
has explicit record, relation, traversal-depth, and deterministic selection-byte limits. Local
targets may extend traversal; dependency targets retain exact declaration descriptors without an
invented SymbolId, and unavailable cross-package incoming calls remain explicit omissions. Results
contain compiler facts only and are detached from the index. ADR 0225 adds a separate bounded
in-memory `WorkspaceCallIndex`: it revalidates a complete caller-supplied exact-current partition
set, joins exact compiler-observed calls by complete declaration key, and exposes detached
package-qualified outgoing and incoming call ledgers. Package-local query results do not consume
that ledger. ADR 0226 exposes only that direct-call ledger through default-free
`experimental atlas-calls list`: one exact package UID, one visible/full SymbolId, one explicit
direction, atomic package/call construction limits, and a final output bound produce the separate
checksummed `fairyc.workspace-call-catalog-json/v0` value. A zero-call selection is complete and
successful. No source excerpt, documentation text, workspace graph traversal, transitive impact,
or stable standalone query transport is enabled.

The first Context Pack encoder now runs those exact-current queries itself and emits one separately
byte-bounded compact `fairyc.context-pack-json/v1` envelope. It retains exact package, map,
compiler/interface, query, budget, usage, completeness, typed fact, relation, and omission data;
`fairyc.context-pack-checksum/v1-sha256` binds the version, encoding, and canonical logical pack.
Documentation, source-excerpt, and warning arrays are explicitly empty, so no caller prose can be
relabelled as compiler truth. The checksum is integrity rather than authentication. No decoder,
untrusted pack input, text import, model call, token estimate, or permission is enabled.

Atlas exports deterministic, model-neutral facts such as symbols, calls, types, ownership transfers,
borrows, errors, effects, allocations, unsafe and FFI reachability, source spans, and related tests.
Unknown or incomplete edges remain explicit rather than being guessed.

ADR 0203 makes the one-way semantic boundary executable. The eight current Atlas/export package
roots may be imported only inside that layer, by `internal/cli` orchestration, or by the Windows
evidence harness. A repository test parses all `cmd/` and `internal/` Go files, including tests and
inactive build-tag files, so a new compiler, backend, runtime, or library package cannot consume
Atlas without an explicit allowlist and architecture update. This is a direct-import boundary, not a
plugin, process-isolation, package-initialization, or separately buildable-Atlas promise.

ADR 0195 accepts a separate `fairyc.attributed-text/v0` logical policy, ADR 0196 implements its first
internal source-excerpt gate, ADR 0197 implements separately bounded canonical
`fairyc.attributed-text-json/v0` output, ADR 0198 exposes its source selection through default-free
`experimental atlas-text source`, and ADR 0200 exposes exact frontend-attached text through the
parallel `documentation` action. `.fymap` v1 remains body-free and
Context Pack v1 retains empty text arrays. Every text record is
`untrusted-text` and binds both SymbolId forms, exact package identity, canonical relative
file, content and line-map identities, half-open byte span, origin, exact UTF-8 bytes, explicit
fragment/count/aggregate budgets, omissions, and a domain-separated fragment checksum. Text is
selected only from already validated retained bytes; file reopening, discovery, URL fetching,
silent normalization, phrase filtering, fact replacement, completeness changes, and authority
grants are forbidden. The implemented extractor admits one workspace-authored exact declaration
source excerpt from revalidated retained bytes. Its output builder requires explicit kind/origin and
text/final-byte budgets, binds exact subject identity even for an omission, runs extraction itself,
and includes independent fragment and document checksums. The command requires exact package and
visible/full SymbolId selection, `workspace-authored` origin, explicit count/text/final-output
limits, and optional locked no-write mode. ADR 0199 fixes the C-friendly `///` frontend attachment
contract and retains exact spans through typed HIR. ADR 0200 carries those spans through an
in-memory-only snapshot sidecar and emits exact marker-bearing retained bytes or an explicit
`kind-unavailable` omission. The current implementation still has no decoder, text store, stable
CLI, rendered documentation model, or dependency/generated text admission.

ADR 0202 adds independent output family `fairyc.atlas-evidence/v0` and experimental
`atlas-evidence show`. One exact-current workspace preparation produces an unchanged Context Pack v1
`show` child plus separate unchanged source and documentation attributed-text v0 children. The
wrapper repeats canonical symbol/package identity, labels itself `none`, labels facts
`compiler-fact`, labels both text children `untrusted-text`, checksums the exact nested values, and
requires explicit query, text, child-document, and final-output limits. It accepts no caller JSON or
text, does not populate Context Pack v1, and adds no decoder, authentication, model call, token
budget, default policy, or authority.

Stage 0 Atlas now exposes `experimental atlas-pack overview` for one exact manifest package UID with
mandatory record, selection-byte, and final-output-byte limits plus optional locked no-write mode.
It emits canonical pack bytes without a newline or map-store write.

The internal exact-current query layer now also provides full-SymbolId-ordered declaration catalog
pages with explicit kind filters, optional exact module and Stage 0 declaration-name selection,
positive symbol/byte limits, and full-ID-only continuation. Exact names use byte equality and retain
all same-name records across modules or kinds. ADR 0193 exposes the command, while ADR 0201 moves
the closed result to checksummed `fairyc.symbol-catalog-json/v1` through `experimental
atlas-symbols list`. It retains no fuzzy rank, normalization, guessed uniqueness or rename, source
text, map-store path, or hidden limit.

Those exact visible or full SymbolIds now drive `experimental atlas-pack show` and `around`.
Both require explicit direction, exact relation filters, record/relation/selection/output limits,
and package UID; `around` additionally requires explicit depth. They emit Context Pack v1 bytes,
retain incomplete dependency and incoming-package joins as omissions, and add no name search, text
import, default store, or safe-change inference. A separate map-publication policy, decoder,
additional text origin/kind support, and multi-symbol or neighborhood evidence composition remain
future work.

The workspace call ledger does not change those package-local command omissions. ADR 0226 exposes
only exact direct calls through `experimental atlas-calls list`: package UID, visible/full SymbolId,
direction, package/call construction budgets, and final output bytes are mandatory. The separate
canonical workspace call catalog v0 echoes the exact construction budget and usage, writes no
source body or host path, and succeeds with an explicit empty call list when appropriate. It does
not traverse across packages, persist a reverse index, join other relation families, or infer
indirect calls, impact, renames, or safe changes.

ADR 0227 adds a separate local process control for that exact command. The configured Windows smoke
builds one standalone compiler outside timing, discovers the root `main` SymbolId and performs one
untimed warmup, then preserves nine first-attempt fresh locked process observations. Each sample
includes process startup through exit and reports raw caller wall time, whole-process Windows peak
working-set/pagefile bytes, canonical output bytes, the fixed two-package/two-source/one-cross-call
shape, and explicit construction bounds. OS caches remain uncontrolled. These are not live-heap or
Atlas-only retained-memory values, and the tiny local fixture establishes no reference budget,
default, percentile, speedup, representative-use, traversal, or impact conclusion.

ADR 0228 separately generates closed 2-package/8-call, 8-package/64-call, and
32-package/512-call star workspaces. Generation, loading, compilation, map planning, and exact
root-symbol discovery happen before timing; one warmup and nine raw samples per shape cover only
exact workspace-call index construction plus outgoing catalog encoding. Samples carry raw monotonic
wall time, process-wide Go `TotalAlloc`/`Mallocs` deltas, and canonical output bytes. The fixed
shapes and 64 MiB output ceiling are harness controls, not representative workloads, retained-memory
evidence, complexity proof, defaults, traversal or impact policy, or a regression gate.

ADR 0229 separately adds test-only chain, diamond, and fan-in workspaces. They pass through ordinary
manifest loading, compilation, verified MIR, Atlas planning, exact-current call joining, and
canonical catalog encoding, then require exact outgoing, incoming, and both-direction selections.
They add no output schema, command, default, indirect-call recovery, traversal, impact policy, or
representative-use claim.

ADR 0230 separately records forced-GC process `HeapAlloc` and `HeapObjects` deltas for five
first-observation batches of 16 retained indexes across ADR 0228's three generated star shapes.
Prepared partitions and expectations stay live across each baseline and sample. The result is a
process live-heap control, not exact Atlas ownership, a per-index cost, a product budget, a
representative workload, traversal policy, or a regression threshold.

The configured Windows host smoke separately measures whole-package canonical rewrite cost after
workspace analysis and map planning. It performs one untimed warmup and nine raw rewrites of every
fixed quickstart package partition, requires exact byte identity, and reports path-free package,
source-file, source-byte, record, and encoded-byte denominators. This control fixture selects no
format replacement, chunking, compression, incremental-update policy, product limit, or regression
threshold.
`impact`, semantic
`diff`, stable anchor syntax, LSP, SCIP, MCP, structural edits, embeddings, model-generated
summaries, and a vector database are deferred behind usage and completeness evidence.

Source text and documentation cannot grant an LLM, IDE, or agent filesystem, process, network,
credential, or approval authority. The implemented snapshot, provisional identity, and logical map
plan are specified in `docs/atlas/semantic-snapshot-v0.md`, `docs/atlas/symbol-id-v1.md`, and
`docs/atlas/fymap-logical-plan-v1.md`. The implemented transport is specified in
`docs/atlas/fymap-transport-v1.md`; the implemented store is specified in
`docs/atlas/fymap-store-v1.md`; the implemented query layer is specified in
`docs/atlas/fymap-query-v1.md`; the internal exact workspace call prerequisite is specified in
`docs/atlas/workspace-call-join-v0.md`; its experimental direct-call catalog is specified in
`docs/atlas/workspace-call-catalog-v0.md` and ADR 0226; compiler-fact pack output is specified in
`docs/atlas/context-pack-v1.md`; the experimental overview command is fixed by ADR 0190, the
rewrite measurement boundary by ADR 0191, the internal exact symbol catalog by ADR 0192, its
experimental output command by ADR 0193, and exact-name Symbol Catalog v1 by ADR 0201. ADR 0194
fixes experimental show/around. Stable commands,
map publication and attributed-text decoding remain future work; ADR 0195 fixes the attributed-text
policy, ADR 0196 implements its internal exact retained-source gate, ADR 0197 fixes its canonical
output-only JSON, ADR 0198 fixes its first experimental source command, ADR 0199 fixes exact
frontend documentation attachment, and ADR 0200 fixes exact documentation extraction and its
experimental action without changing compiler-fact transports.
ADR 0227 separately fixes the path-free local workspace-call process-measurement evidence envelope;
it changes no compiler, catalog, map, query, or command compatibility surface.
ADR 0228 fixes the separate path-free local workspace-call scaling-measurement evidence envelope;
it likewise changes no compiler, catalog, map, query, or command compatibility surface.
ADR 0229 adds only a deterministic test corpus and changes no compatibility surface.
ADR 0230 fixes a separate path-free retained-heap batch evidence envelope and changes no compiler,
catalog, map, query, or command compatibility surface.

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

Registry coordinates, provider-qualified import spelling, range `for`, resource-aware or block-form
`defer`, checked narrowing conversion syntax, expression-bodied function shorthand,
public `Arena` construction and capacity APIs, exact parser resource limits, glibc and toolchain
baselines, portable path model, targets beyond the declared Linux GNU and Windows MSVC profiles,
package registry, and the `.fymap` default path, limits, and public command policy remain UNDECIDED.
Alternate map encodings, compression, and chunking remain evidence-gated. The
provisional pre-release SymbolId hash algorithm is now implemented by ADR 0183; its eventual stable
release and cross-version migration policy remain open.
