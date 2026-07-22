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
- Written declarations put one complete type before one name. C name-splitting declarators, C casts,
  preprocessing, and textual includes are absent.
- Simple statements use explicit semicolons; compound `if`, standalone blocks, and standalone
  `unsafe` blocks do not. Assignment is a statement, and integer values are not implicit booleans.
- Every file declares one module. Imports require aliases and cannot be wildcard, selective, cyclic, or re-exported.
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
- Expressions follow one fixed Pratt binding-power table; comparisons are non-associative and assignment is never an expression.
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

Classic or range `for`, lexical `defer`, checked narrowing conversions, provider-qualified package
imports, generic iterators, expression-bodied function shorthand, and formatter behavior are
follow-up decisions rather than implicit parts of that spelling migration.

## Value categories

### Inert values

`struct` represents passive data. It has no implicit constructor or destructor. A struct is copyable only when all fields are copyable. It cannot contain a resource or a safe borrow in Stage 0. Hidden heap allocation and user-defined implicit conversion are not allowed.

A restricted user `enum` may contain only inert copyable payload fields. A payloadless variant is
constructed contextually as `.Variant`; a payload variant uses `.Variant { .field = value }`.
Shallow switching names every payload field and covers every declared variant exactly once. The
expected enum type resolves the leading-dot form; global best-fit variant lookup is forbidden.
Built-in `Option` and `Result` may
contain resources because the compiler owns active-variant move and drop behavior. General
resource-bearing enums are deferred.

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

- `ref T` is a non-null shared read-only borrow.
- `ref mut T` is a non-null exclusive mutable borrow.
- `ptr T` is a raw pointer. Inside `unsafe`, a direct borrow may erase its pointee to `ptr void` without
  strengthening mutability while MIR retains the concrete owner type. Null values, pointer
  arithmetic, every other pointee-changing cast, and dereference remain outside safe code.
- Nullable borrows use `Option<ref T>` rather than a nullable reference representation.

The Stage 0 borrow model uses local last-use loans and whole-root overlap. It does not infer disjoint mutable field borrows or two-phase borrows. A returned safe borrow must have one unambiguous origin. Stored borrows, self-reference, and multiple-origin returns must use an index or generation handle, a redesigned API, a lexical region, or a narrow unsafe module.

### Regions and arenas

`Region` is a compiler-known lexical capability, not an ordinary mutable arena object. It cannot be stored or returned. Allocations derived from it carry the region origin, and a derived safe borrow cannot outlive the lexical region.

An ordinary region may contain only values that do not need drop. A resource-aware arena would require a drop list, partial-construction rules, reverse-order cleanup, and reset failure semantics; it is deferred until those contracts and tests exist.

Region-parameter origins, permitted returns, package-resolved local-region escape, resource-placement rejection, capability storage, and direct-construction rejection are versioned in `spec/fixtures/region/manifest-v1.json`. The lexical escape case names the explicit standard-library package instance that owns the opaque fixture `Arena` type.

Verified MIR and interpreter fixtures additionally execute lexical Region entry, inert placement,
inner-resource cleanup, and reset on fallthrough, direct `return`, Result `?` propagation, and
nested Region exits. `break` and `continue` target the nearest `while` and execute scope-indexed
resource cleanup and Region reset before their MIR goto; use outside a loop is `E1012`.

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
- `break` and `continue` target only the nearest enclosing `while`; labelled or value-producing
  loop control is not part of Stage 0.

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
- Source `as` integer conversion is currently limited to the target-independent, value-preserving
  pairs `u8 -> i32`, `u8 -> u32`, `u8 -> usize`, and `u32 -> usize`. Narrowing, signedness-changing,
  C-scalar, pointer/integer, and other numeric conversions require a later explicit contract.
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
  distinct from coincidentally represented Fairy scalars; C-scalar literals, operators,
  conversions, and raw pointers remain deferred. Exact C-scalar fields are admitted only in
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

`Buffer` is the only growable owned collection. `Vec<T>` is deferred with user generics. Positive scalar fixed arrays are compiler-owned inert values but are not growable collections or public ABI records. `u8[]`, `str`, and `CStr` are compiler-known unsized views; `String` and `CString` are owned resources. Compiler-owned `b"..."` literals implement immutable static `ref u8[]` construction and private generated-C transport. Safe borrowed `ref u8[]` and `ref str` views expose `.len` as a `usize` byte count and `.is_empty` as the exact zero-length `bool` by reading existing metadata; neither access scans, allocates, calls, traps, or counts Unicode characters. Immutable `ref u8[]` additionally supports read-only `usize` indexing with a `u8` result and end-exclusive slicing with an immutable `ref u8[]` result. Slice starts and ends may be omitted, normalizing to zero and the retained receiver length without re-evaluating the receiver. Written operands evaluate once left to right and trap with `bounds` before access or pointer derivation when their bounds fail; slicing preserves the receiver's borrow origin. Mutable fixed scalar-array places support checked exact-type element assignment. A direct contextual `ref mut` borrow of a mutable fixed `u8[N]` place may produce `ref mut u8[]` for an exact internal parameter. It preserves the exclusive origin, exposes canonical `.len`, supports checked `u8` reads and stores, and may derive `ptr mut u8` only inside unsafe code. Indexed assignment checks the receiver and `usize` index before evaluating its right-hand side, so an invalid place traps without value-side effects. It does not add mutable slicing, resizing, temporary borrowing, retained foreign pointers, or public ABI. The initial Linux `File` API accepts `ref CStr`, preserves partial I/O, and uses a consuming non-retried Linux close contract. A separate Windows component plus example-level safe FY wrapper stores inline `HANDLE` state, maps typed errors and filesystem effects, preserves partial counts, syncs explicitly, consumes state before one `CloseHandle` attempt, and reads through the restricted mutable byte view. It is target-bound evidence rather than a stable `std::fs` API. Portable paths, embedded support, and resource-aware arenas remain deferred.

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
  default artifact with stable `build --release --locked`, and use that separate process to compile
  and run both the checked-in scalar quickstart through `run-target` and exact-output example through
  `exec-target` for `x86_64-windows-msvc`. The latter must expose exact stdout and process status 42
  to the parent.
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
- Parsing starts from original source bytes, preserves trivia and error provenance in compact lossless syntax, and lowers to AST only after an error-free batch parse. An immutable green tree and incremental reparse are deferred until formatter or IDE work requires them.
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
  configured source probe crosses one 8-byte INTEGER-class result and one 20-byte MEMORY-class
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
  profiles for 56 configured native crossings. This does not implement source FFI or establish a
  hosted result.
- For Windows the same engine selects a distinct Microsoft x64 planner. A separate 15-case
  manifest covers RCX/RDX/R8/R9 and XMM0-3 selection by shared declaration-order slot, later stack
  parameters, 1/2/4/8-byte record integer transport, larger or odd-size record indirection, and the
  hidden-return RCX slot. Its 12 pass plans generate separate caller and callee C11 translation
  units that local x64 MSVC compiles, links, and executes at low and optimized profiles. This is a
  same-compiler local crossing, not source-level Fairy FFI, independent ABI inspection, or hosted
  evidence.
- `internal/pecoff` parses every retained or temporary executable produced by the Windows host
  smoke. It requires a bounded AMD64 PE32+ non-DLL console image, large-address awareness, aligned
  image/header/section ranges, a readable executable `.text` section, and an entry point inside
  executable code. It records import and exception-directory presence without treating either as
  validated content. Exact COFF object symbols and relocation types, complete `.pdata`/`.xdata`
  semantics, CodeView/PDB, broad source-level Windows FFI, and a hosted receipt remain separate
  gates.
- The Windows C driver retains each compiler-produced `.obj` only in its bounded temporary root,
  parses it as raw AMD64 COFF, and fails before returning the linked image when the object has an
  optional PE header, linked-image flags, no `.text`/`.text$...` code, invalid raw or relocation
  ranges, relocation indexes outside the raw symbol table, or no named symbols. Uninitialized
  `.bss` may have a nonzero logical size and zero raw pointer. The driver returns isolated object
  bytes and dense names as an internal receipt, then removes every temporary file. This is not yet
  a versioned exact symbol, COMDAT, relocation-type, unwind, or hosted object fixture.
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
  not a Fairy C language unwind feature: Stage 0 traps and calls remain non-unwinding. Version-2
  epilog offset semantics, language-specific handler payload parsing, recursive chain equivalence,
  source-level Fairy or foreign-frame crossings, object-level unwind relocations, and hosted
  fixtures remain fail-closed.
- The eleven valid layout cases additionally generate one deterministic private C11 plus explicit
  POSIX-typedef probe. The Linux
  native runner compiles, links, and executes it with Clang/GCC and low/optimized profiles, checking
  recursive size, alignment, field size, and offset assertions. Failure cases remain Go-owned input
  rejections. No hosted success receipt follows from the configured path.
- A small MIR interpreter is the semantic oracle before any second native backend.
- Its unstable `experimental run` path accepts one entry plus an explicit bounded repeated-`--source`
  closure, selects the entry module's `main() -> i32`, and performs no source discovery.
- Its unstable `experimental check` path accepts one primary source plus the same bounded repeated
  `--source` closure, requires no executable entry, and stops after MIR verification. It performs no
  backend, target, toolchain, process, network, or artifact action. It remains distinct from stable
  package-aware `fairyc check`.
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
- Its fixture-only output hooks require explicit manifest bindings, capture bytes in memory under a
  shared limit, and never authorize source names, the CLI, or host streams.
- The current C backend emits deterministic C11 only for verified scalar, inert-struct,
  restricted inert-enum, flat-resource, and resource-tagged MIR
  under the provisional `x86_64-linux-gnu` profile. It includes private `Option<i32>`,
  `Result<i32, i32>`, copy-only built-in tags over canonical Fairy scalars or admitted inert structs,
  nonempty empty-drop resource transport over Fairy copy scalars or raw pointers, and built-in
  resource-bearing tags whose payloads are exactly `i32` or direct admitted resources, plus private
  restricted user-enum tags
  whose payload fields are admitted scalars, raw pointers, or admitted inert structs. Nested user enums
  and built-in tags inside a user-enum payload remain a C-backend capability boundary. No public
  tagged or resource ABI is claimed.
  Nonempty admitted `@repr(C)` structs preserve public tags and field names; their layout and private
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
  while observing counts, repeated-close status, normalized state, and bytes. Report v31 records
  `linuxFileSourceCallExecutions` alongside the earlier standard-library source-call counters and
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
  ADR 0133 strengthens the existing `arenaExecutions` success condition with deterministic
  first-stage and second-stage allocation failure plus live-allocation cleanup. ADR 0134 strengthens
  `bufferExecutions` with deterministic construction and three growth-allocation failures, exact
  storage preservation, recovery zeroing, and live-allocation cleanup. Neither adds a report field
  or counter, so report v32 is not created.
  Its closed
  `drop-trace-v0` emission profile assigns source/type/span identities to dense site IDs and compares
  exact four-byte native records with the MIR interpreter trace. This instrumentation is absent from
  product `emit-c` and `build`. No hosted success receipt or broader target
  conformance follows from the command's presence.
- The experimental target-inspection command emits that target description without probing the
  host. It reports `scalar-and-flat-copy-resource-tagged-c11` separately while keeping `emittable`,
  `conformant`, and `supported` false and listing the evidence still missing.
- The provisional top-level `targets` command renders every validated TargetSpec in canonical ID
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
  probes supply Windows byte-level and allocation-failure evidence. Owned text copies a compiler-owned
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
  foreign companion inputs and user linker arguments remain absent. Lexical `region` lowering, copied
  placement, source-level FY observation of owned-text byte/terminator contents, stable `std::fs` naming, portable
  paths, general mutable buffers, and stable layouts remain absent. This clears only the missing target
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

Resolved package graph assertions are versioned in `spec/fixtures/package/manifest-v1.json`, which includes all eight historical v0 cases and adds deterministic package-cycle rejection. They fix package-instance equality, one source root per instance, acyclic direct dependency structure, direct-only dependency visibility, module-provider ambiguity, and explicit standard-library injection without choosing public manifest, lockfile, coordinate, registry, or solver syntax. `internal/packagefixture` executes all nine cases through the reusable `internal/packagegraph` validator and resolver, converts passing graphs into build-local semantic package input, and proves isolated equal module paths complete semantic Flow. Region v1 consumes the same validated graph result. The independently versioned `spec/fixtures/package-identity/manifest-v1.json` freezes the canonical resolved-tuple byte encoding, full SHA-256 `PackageInstanceId`, text form, component boundaries, and feature-set rejection reasons without choosing any package-manager format. `PackageCoordinateId` v1 separately identifies the source-independent UID/version/feature coordinate used by public cross-package contracts. Typed HIR retains source `pub` state for every current item category. `spec/fixtures/module-interface/manifest-v2.json` compiles the current source variants and freezes the coordinate-based public-interface digest plus same/different invalidation behavior for bodies, private representation, declaration order, public signatures, public field order, foreign link names, dependency source changes, and dependency coordinate changes; historical v1 remains frozen. `spec/fixtures/package-source/manifest-v1.json` separately freezes the exact path-and-source-byte digest used for the current package's own implementation invalidation. `spec/fixtures/package-cache/manifest-v1.json` combines those identities with the exact compiler/language/MIR/target/ABI/toolchain/backend/runtime/standard-library/profile compatibility vector and the first caller-bounded process-local LRU store.

ADR 0174 advances the current whole-package cache contract to
`spec/fixtures/package-cache/manifest-v2.json`. The current package keeps exact source identity;
dependency pairs use source-independent coordinate plus public-interface v2. Stable function,
nominal type, foreign, body, and package artifacts use the same coordinate boundary and still
re-enter MIR verification, so a private dependency edit can reuse the caller without accepting an
old provider body. Historical cache-key v1 fixtures remain frozen.

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
version, source digest, and lock fields. The current source surface exports only opaque
`std::alloc::Arena`; it performs no host installation lookup and does not settle target C linkage or
the broader planned standard-library API.

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
build/run remain available for explicit evidence and compatibility; stable run remains separate.

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
feed a separate local x64 MSVC C11 probe at low and optimized profiles; this is not complete Windows
layout coverage or a hosted receipt.

Formatting, testing, documentation, vendoring, binding generation, and ABI inspection remain planned
tool surfaces rather than claims of current implementation. Checking has one implemented stable
surface: the bounded current-workspace `fairyc check [--locked]` contract above.

## Compatibility

Compiler release, language revision, internal MIR, target schema, ABI profile, diagnostic catalog and JSON, runtime ABI, standard library, package schemas, `PackageInstanceId`, `.fymap`, `SymbolId`, and interface hashes are independently versioned contracts. Pre-1 source and runtime contracts use exact compatibility. MIR and caches are disposable and are regenerated rather than migrated.

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

Registry coordinates, provider-qualified import spelling, classic and range
`for`, lexical `defer`, checked narrowing conversion syntax, expression-bodied function shorthand,
public `Arena` construction and capacity APIs, exact parser resource limits, glibc and toolchain
baselines, portable path model, targets beyond the declared Linux GNU and Windows MSVC profiles,
package registry, `.fymap` binary encoding, and the concrete `SymbolId` hash algorithm remain
UNDECIDED.
