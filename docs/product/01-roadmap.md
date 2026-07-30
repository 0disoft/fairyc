# Roadmap

- Status: Draft
- Owner: UNASSIGNED
- Scheduling: UNDECIDED

This roadmap defines capability gates, not calendar promises. Exact durations, fixture totals, latency targets, and storage limits remain absent until an executable harness and source of truth exist.

## Phase 0: freeze the executable contracts

- Freeze Stage 0 surface syntax through `spec/grammar/stage0.ebnf`, parser fixtures, semantic fixtures, resource, Region, Result Flow, verified-MIR, and resolved-package-graph fixtures, one-file modules, alias imports, restricted enums, ownership, borrow origins, regions, `Result`, evaluation order, traps, unsafe boundaries, and exclusions.
- Freeze the first planned conformance target, `TargetSpec`, `ToolchainSpec`, C ABI v0, runtime and standard-library layer ownership, and the compatibility vector.
- Reserve diagnostic categories and codes for rejected syntax, ownership, region, FFI, ABI, and compatibility paths.
- Define Flow CFG, MIR v0, drop elaboration, the MIR verifier, isolated verifier-mutation recipes, generated-C safety rules, and compile-pass, compile-fail, run, trap, drop-trace, runtime, and bidirectional C ABI fixture formats.

Exit gate: the written contracts have no contradictory syntax, initialization, ownership, cleanup, package identity, visibility, target, runtime, or compatibility rule, and every mandatory rule has an observable fixture class or an explicit dependency-gated deferral. Parser, semantic, resource, Region, Result, MIR, and package manifest cases must pass their full runners with no unlisted independent diagnostics or observation drift.

### Completed breaking surface-revision gate

ADR 0136's C-shaped pre-1 surface is now the only product parse path. The normative grammar,
diagnostics, fixtures, examples, embedded source literals, and current documentation use
return-type-first function headers, type-first declarations, `ref`/`ptr` types, parenthesized
control headers, `switch`/`case`, contextual leading-dot variants, and resource `cleanup` blocks.
The immediately preceding lossless grammar is isolated behind the fail-closed
`fairyc-migrate-prev` tool and cannot enter ordinary lexing, parsing, semantic analysis, or code
generation. The migration checker, embedded-source checker, and fixture-span checker all require a
zero-write result after the cutover.

This gate deliberately did not add package manifests, provider-qualified imports, `for`, `defer`,
or expression-bodied functions. Those remain separate capability decisions with their own
contracts and fixture requirements.

ADR 0213 now adds a separate classic scalar `for` contract and executable fixture matrix. Range
`for`, lexical `defer`, provider-qualified package imports, and other new semantic features remain
deferred. Expression-bodied function shorthand is also deferred until real source justifies a
second body form.

ADR 0217 now adds a separate bounded lexical `defer` contract. Immediate copy-scalar capture,
block-exit LIFO order, return and Result-propagation paths, loop-control exits, trap skipping, and
ordinary-call MIR elimination are executable. Resource or borrow capture, foreign calls, method
receivers, arbitrary deferred blocks, and abnormal-termination guarantees remain deferred.

### Post-Stage 0 language-concept admission queue

ADR 0218 separates ideas worth studying from accepted syntax and implemented behavior. No item in
this queue changes the current language revision until it receives its own ADR, grammar and
diagnostic contract, compatibility decision, and executable fixtures.

ADR 0219 has now implemented the first admitted item: narrow callable `@must_use` beyond the
existing mandatory `Result` and resource handling. It includes zero-argument internal and foreign
callable attributes, call-site-only explicit use, `discard`, direct-call-only function transport,
`E4201`/`E4202`, public-interface v3, exact language `/4`, and backend-identity evidence without
turning all return values into warning noise.

ADR 0220 has now implemented the second admitted item. It retains private scalar module constants,
uses `@const_eval` on private straight-line scalar functions rather than importing `const fn`, adds
module-level `static_assert`, closes the old AST constant-loss gap, rejects cycles and checked traps,
caps depth and evaluated nodes, stays frontend-owned, keeps public-interface v3 unchanged, and cuts
the exact language pin to `/5`.

ADR 0221 now implements the third admitted item: exact zero-argument `source_location()` expands to a
static `ref str` containing canonical relative path, byte-based line column, and start-byte
coordinates. It reserves no default-argument or stack-inspection machinery and cuts the exact
language pin to `/6` with semantic, HIR, interpreter, C11, migration, interface, and cache evidence.

ADR 0222 closes the construction candidate without adding syntax. The audited SDK/example corpus
contains four public fallible resource factories whose names expose copy, conversion, create, and
open policy; constructor syntax would not remove their explicit rollback work or improve ownership
evidence. Ordinary named `T`/`Result<T, E>` factories remain the complete construction model.

ADR 0223 closes the remaining static-interface candidate. The same 29-file corpus contains 82
function declarations but no algorithm duplicated across different production types, and the
language has no user-generic declaration or instantiation model for an interface to constrain.
Static interfaces may return to the queue only after implemented bounded user generics and the
cross-type duplication, build-cost, code-size, resolution, ownership, and backend evidence required
by that ADR exist.

The current modern-concept admission queue is closed. New candidates require a new measured problem
statement under ADR 0218 rather than being appended as syntax ideas.

ADR 0224 separately closes three loose surface follow-ups for the current corpus. One manual
length-indexed byte-view loop does not justify range iteration, no narrowing use justifies a checked
conversion matrix, and roughly twenty one-return blocks do not justify a second function-body
grammar. Classic scalar `for`, explicit `while`, lossless widening casts, and braced bodies remain
the complete implemented forms. Provider-qualified imports and formatter work retain their own
package and tooling gates.

Exceptions, class inheritance, implicit virtual dispatch, user-defined implicit conversions,
operator or function overloading, unrestricted template metaprogramming, runtime reflection,
garbage-collected object roots, and language-owned async, coroutines, virtual threads, or hidden
executors are outside this queue.

## Phase 1: first interpreted semantic slice

Current implementation status: the source, lexer, recoverable-syntax, parser-diagnostic, AST
lowering, and parser-fixture-runner portion of the first item exists. Name resolution and typed HIR
now cover the semantic fixture's scalar and safe-borrow subset. Local last-use loans, borrow
  conflicts, single-origin borrow returns, checked matching-type left shift, loop-context diagnostics,
  scalar foreign-call unsafe/signature boundaries, and `@link_name` shape diagnostics execute across
  all 59 semantic v0 cases, including explicit resource-factory construction and cross-module
  resource-literal rejection, explicit source-location expansion, bounded scalar compile-time evaluation, narrow callable `@must_use`, classic scalar `for`, bounded lexical `defer`, branch- and loop-aware safe-borrow reassignment, exact
  interprocedural borrow-return summaries, integer-literal range rejection before MIR, and exact
  transport for all sixteen TargetSpec C scalars. The type phase also canonicalizes the closed
  seven-name `@effects(...)` vocabulary and enforces caller supersets independently from unsafe. General
  whole-root resource Flow now executes all 12 resource v0 cases, including branch, normal-loop,
  break-edge, and continue-edge state rejection. Result Flow executes all 24 result v0 cases, including contextual leading-dot scalar construction,
exhaustive shallow switching, and local, parameter, branch, loop, overwrite, and early-return
must-use obligations. The resolved-package-graph engine and fixture runner execute all 8 package v0
cases, including exact-instance deduplication, feature splits, isolated equal module paths,
direct-only visibility, explicit standard-library injection, ambiguity, hidden transitive modules,
and duplicate bindings. Passing graphs now feed syntax-clean sources and exact import-provider
bindings into semantic analysis with deterministic build-local package identity; equal dependency
module and source paths remain distinct through typed HIR types and verified MIR names without
choosing a public package ID. Package-aware spans keep that identity separate from the original
relative source path so private numbering cannot leak into diagnostics. Mutable-local MIR lowering and verification
  execute all 13 MIR normalization v0 cases for the current scalar, resource, tagged-payload, Result, Region, classic-`for` and lexical-`defer` lowering, loop-control cleanup, call,
  division, and checked-arithmetic trap subset. All 10 verifier baselines and 22 isolated mutation cases execute through
  the same MIR verifier. The first MIR interpreter executes all 94 interpreter v0 cases, including classic scalar `for`, bounded lexical `defer`, direct `i32` minimum constants, scalar fixed-array calls, contextual immutable and mutable byte-view borrowing, exact lengths, checked reads and indexed stores, and
scalar `Option` and `Result` construction, exhaustive switching, and both Result `?` paths, and is
exposed by the unstable `experimental run` command, which now accepts a bounded explicit entry-plus-
repeated-`--source` module closure without discovery. Resource construction, field reads, whole-root
call moves, source-linked drop traces, resource-bearing `Option` and `Result` payload moves, ordered
resource reassignment, explicit consuming-close success/failure ownership, and normal lexical
Region placement, inner cleanup, normal reset, direct return, Result propagation, and nested
early-exit reset execute. Manifest-authorized in-memory stdout/stderr byte hooks prove output order
and bounded failure without granting host access. Checked `i32` addition, subtraction,
multiplication, negation, division, remainder, matching-type bitwise operations, and signed or
unsigned right shift execute across trapping and
  non-trapping boundaries with exact overflow, division-by-zero, or invalid-shift
  categories. Matching-type left shift additionally traps on an unrepresentable result, with
  invalid amount taking precedence over overflow. Nearest-loop `break` and `continue` now execute
  with resource and Region cleanup.

The sibling `experimental check` command accepts the same bounded explicit source closure but
requires no executable entry and stops after MIR verification. It validates admitted libraries and
FFI wrappers without interpreter, backend, target, toolchain, process, network, or artifact access;
ADR 0175 later promotes the package-aware current-workspace spelling to stable `fairyc check` while
retaining this explicit-source command as an experimental developer surface.

Verified resource definitions now preserve ordered fields and
  drop classification from AST through HIR into MIR; nonempty user drop bodies now lower to
  verifier-linked private functions and execute before reverse field cleanup. Resource-declared
  effects are preserved and required only at actual cleanup edges. Restricted inert enums now preserve named payloads,
  construct through contextual leading-dot variants, require exhaustive shallow switches, and execute through
  verified MIR, the interpreter, and the constrained C backend. The C backend now emits deterministic C11 for 38
  verified scalar, borrowed-view, typed-comparison, inert-enum, flat-resource, and resource-tagged cases under the provisional `x86_64-linux-gnu` profile,
  including `Option<i32>` and `Result<i32, i32>` construction, switch, calls, and propagation,
  copy-scalar/inert-struct tagged transport, private
  positive scalar fixed-array construction, contextual zero-copy shared and mutable `u8[N]` byte-view borrowing, exact length observation, internal value calls, and checked reads and indexed stores, plus
  flat copy-resource construction, field reads, moves, reassignment, and resource-bearing built-in
  tag variants and propagation. The unstable
  `experimental emit-c` command exposes that exact verified translation unit on stdout
  without writing an artifact. On an eligible Linux/amd64 glibc host, `experimental build` can now
  compile and link that subset with explicit Clang or GCC and low or optimized profiles into
  a no-overwrite `build/` artifact. Compiler-bundled `std::io::write`, `write_stderr`,
  `write_bytes`, and `write_stderr_bytes` now provide progress-preserving target-neutral text and
  binary output operations. `read_stdin` adds one allocation-free bounded mutable-byte-view input
  operation with EOF-as-success semantics. Windows has exact separated output plus direct-artifact
  EOF, short-input, and bounded-prefix evidence; ADR 0214 records the Linux probes' first hosted run.
  Input forwarding for other native commands, portable files, broader console operations, effect-bearing function values,
  and every later Phase 1
  capability remain unimplemented.

- Implement source management, lexer, trivia-preserving tokens and spans, compact recoverable syntax, deterministic diagnostics, one-file modules, alias imports, and non-generic typed HIR in Go.
- Implement restricted inert enums, whole-resource moves, local last-use loans, call-bounded mutable reborrows, single-origin borrow returns, plain regions, `Option`, `Result`, shallow `switch`, and explicit `?` and `discard`.
- Implement Flow CFG, mutable-local MIR, drop elaboration, the verifier, target-independent layouts, and the compatibility-vector metadata model.
- Build a small MIR interpreter covering scalar and tagged behavior, resource payloads, drop traces, region allocation, a controlled in-memory I/O test harness, and traps.

Exit gate: positive and negative fixtures agree with the source contracts, resource-bearing built-in variants have correct active-payload traces, and every named verifier seed passes before its isolated mutation demonstrates the intended single `Vxxxx` rejection.

## Phase 2: first hosted executable slice

Current implementation status: a fail-closed Linux/amd64 constrained-C differential command
requires glibc, Clang, and GCC and runs in a separate hosted Ubuntu CI job. It executes
run all 38 emission cases at low and optimized profiles for 152 process-status and exact
source-linked drop-trace comparisons, then drive
the actual experimental build CLI with resource-bearing Option and Result plus a flat all-`i32` resource
through 4 compiler/profile executable
smokes. It now adds four-way product builds containing compiler-owned nonempty `ref CStr` storage and a
private compiler-owned `ref u8[]` view, plus separate generated-Fairy-to-C calls that verify every byte
of `fairy\0` and every non-terminated byte of `fairy` respectively. The same job
also compiles and executes each production, uninstrumented C unit with Clang
  and GCC under the closed `address-undefined-o1` ASan+UBSan mode for 74 fail-closed sanitizer
executions. GitHub Actions run `30442310794` passed the complete lane for exact commit `0bb4199`;
ADR 0214 records that first hosted receipt. It closes the receipt marker only, not native
conformance. A shared bounded C driver and embedded
scalar runtime serve both that lane and the experimental explicit-source-closure build command, while a separate
closed conformance runtime records four-byte drop-site IDs only for the native matrix. Product C and
product-built artifacts contain no observer. This proves
the product wiring exists, not that runtime, ABI, or target-conformance gates pass.
The executable `fairyc.target-spec/v0` now centralizes host-independent Linux LP64/SysV/ELF/DWARF
facts and provisional Windows LLP64/Microsoft-x64/COFF/unwind facts, including an explicit zero or
32-byte shadow-space rule and per-profile missing maturity evidence. The C backend and driver still
use only the Linux typed target ID, while local toolchain paths remain outside both contracts.
Host-independent scalar,
array, explicit-C-enum, pointer, and nested-aggregate layout execution now runs through 16 versioned
Linux v0 cases. A separate nine-case Windows MSVC manifest covers LLP64-sensitive C `long`,
`size_t`, pointer, function-pointer, explicit-enum, and nested-record expectations while rejecting
the absent POSIX `ssize_t` identity. Its eight pass cases now generate one non-POSIX C11 assertion
unit compiled and executed by local x64 MSVC at low and optimized profiles. The eleven valid Linux
cases generate one deterministic C11 plus POSIX-typedef layout probe configured
for four Clang/GCC and low/optimized executions. Those native runs remain unobserved on this Windows
host. A host-independent SysV AMD64 classifier now executes 17 versioned cases for eightbyte
classes, registers, exhaustion rollback, and hidden returns, but bidirectional native ABI fixtures
and hosted evidence remain incomplete. Its 14 pass plans now generate 56 configured separate-object
Clang/GCC crossings in both compiler directions and profiles; these are not yet hosted evidence or
 source-level aggregate Fairy FFI fixtures. An explicitly `@link_name`-mapped scalar `i32 -> i32`
 source fixture now traverses AST, HIR, MIR, C emission, separate objects, and both compiler
 directions, but its four
 native executions remain unobserved on this Windows host.
The first standard-library-owned C storage slice now implements provisional fallible `SystemHeap`
allocation, copy-and-release resize, zero-size normalization, fundamental-alignment rejection, and
failure rollback. A closed probe is configured for Clang/GCC at low and optimized profiles. It is
not exposed through a public `std::alloc` API. One internal generated Fairy caller now crosses its
exact `void *`/`size_t` block into the real allocate and deallocate
helpers for four additional configured native executions. All eight native SystemHeap executions
remain unobserved on this Windows host; the later Arena, text ownership, and Linux File C slices now
exist and each has a raw generated-Fairy source crossing.
The next storage slice now adds provisional Buffer construction, initialized-prefix-only reserve,
overlap-safe append across relocation, overflow rollback, zeroed growth, truncate, clear, and drop.
Its four C-only native probe executions and four generated-Fairy source-call executions are included
in the hosted run recorded by ADR 0214. The internal caller uses the exact nested SystemHeap-block record and
reaches initialize, zeroed resize, and drop. Used-symbol product linkage now supplies the exact
Buffer/SystemHeap closure, but public `std::bytes` and safe ownership methods remain missing. A raw
Buffer caller now executes against an instrumented target-specific closure at both MSVC profiles.
It forces initialization, reserve, relocating self-overlap append, and zeroed-resize allocation
failure, preserves exact storage state and initialized bytes, proves recovery zeroing, and returns
the live-allocation count to zero after drop.
The Arena storage slice now adds a single-active-Region generation check, failure-atomic zeroed and
copied byte placement, reset invalidation, stale-use rejection, and active-drop rejection. Its four
C-only executions and four generated-Fairy source-call executions ran in ADR 0214's hosted receipt.
The internal caller crosses exact self-linked node, Arena, and Region records, obtains zeroed bytes
through `void **`, and verifies reset/drop generations; lexical Region lowering, source-level
`std::alloc` construction, and safe calls remain missing. Windows target resolution now admits six
exact Arena/Region calls, including copied placement. The declared target build executes five
lifecycle calls at both MSVC profiles, observes non-null
allocation plus node/active/generation state, reset invalidation, and final drop, and closes Arena
over SystemHeap exactly once. Its FY path cannot inspect returned bytes, so byte-level zeroing and
the broader failure matrix remain outside that source example. A separate raw Arena caller now
executes against the target-specific implementation at both MSVC profiles, observes zeroed and
copied bytes, forces payload and node allocation failure independently, proves rollback to zero live
allocations, and rechecks stale Region and active-drop rejection.
The allocation-free text-view slice now checks canonical UTF-8 and code-point byte boundaries,
including overlong, surrogate, out-of-range, truncated, and obsolete forms. ADR 0214 records its
four native runs. A separate four-way Fairy-source crossing now calls the actual
empty-input UTF-8 validator in the text-view C unit; the public borrowed text result, nonempty
source pointer path, and full wrapper remain missing.
Compiler-owned `c"..."` storage now crosses semantic typing with `Static` origin, MIR as exact
terminated bytes, and generated C as deduplicated immutable static storage. The direct literal can
expose only an immutable `ptr u8` address through an explicit unsafe operation. Compiler-owned
`b"..."` storage now crosses the same stages as exact non-terminated bytes and a private non-null
`{ data, length }` view; empty literals retain length zero through an immutable sentinel and typed
locals can expose only the immutable data address. Plain UTF-8 `"..."` literals now cross semantic
typing, a distinct MIR operand, and a private `fy_str` view with decoded UTF-8 validation,
deduplicated immutable bytes, an empty non-null sentinel, and typed-local immutable data extraction.
Borrowed `ref str` and `ref u8[]` views now expose their exact byte count through zero-cost `.len` metadata
reads and exact zero-length state through `.is_empty`, and cross internal function calls as private
non-owning descriptors. One new emission case
executes both borrowed-view parameter families through the interpreter and deterministic C backend.
Immutable `ref u8[]` now supports read-only `usize` indexing to `u8` and explicit or omitted-bound,
end-exclusive slicing through verified MIR bounds checks, interpreter `bounds` outcomes, and
generated-C checked loads or pointer derivation. Inclusive ranges, text
indexing or slicing, public CStr operations, Unicode counting, safe text operations, and stable
slice or string ABIs remain missing.
Mutable fixed `u8[N]` places now contextually borrow as `ref mut u8[]` for exact internal parameters.
The view preserves its exclusive origin, exposes canonical length, emits a private writable
descriptor, supports checked `u8` reads and indexed stores, and may derive `ptr mut u8` only inside
unsafe wrapper code. Mutable fixed scalar-array places also support checked exact-type element
assignment. Bounds are proven before right-hand-side evaluation. Mutable slicing, resizing, and
retained foreign pointers remain missing.
Owned String and CString now have provisional Buffer-backed C transport with immutable views,
successful-only Buffer moves, failure ownership preservation, self-source append, interior-NUL
rejection, and exact termination. Four Linux C-only native runs remain configured and locally
unobserved; the same probe now executes as two separate-translation-unit x64 MSVC runs. Separate
generated-Fairy crossings call the real bounded CStr validator and exact raw String/CString records.
An additional example-level safe FY module now confines five opaque storage bridges behind move-only
`String`/`CString` resources, source `Result` errors, scalar lengths, owner-bound `ref str`/`ref CStr`
views, and verified nonempty drop glue. Its low and optimized Windows target builds cover borrowed
view calls, append, empty CString, interior-NUL rejection, and implicit cleanup with result 42. The existing raw Windows target lifecycle still observes lengths,
pointers, alignment, statuses, and normalized drops. The raw Windows probe additionally observes
exact bytes, terminator contents, invalid UTF-8, self-overlap, null/malformed storage, failure
preservation, size overflow, and repeated drop. A probe-only allocator now deterministically forces
initial String/CString allocation and String append reallocation failures, proves preservation, and
returns its live-allocation count to zero. Stable `std::text`, public layouts, and general
user-defined raw-parts reconstruction remain later gates.
The Linux File C slice now also covers partial read/write, write-all progress, sync, and consuming
close including forced close failure. ADR 0214 records its four C-only native runs.
A second four-way generated-Fairy crossing uses exact result, File, and open-option records to call
the real open, write-all, sync, close, reopen, partial-read, EOF, and final-close helpers. It checks
exact counts, repeated-close status, normalized closed state, and reconstructed bytes. ADR 0214
records the hosted execution; public safe File ownership, source effects, and typed errors remain
missing.
Used-symbol product linkage now selects Linux File only for actual admitted calls.
The Windows target now owns a separate raw File C component over inline `HANDLE` state. It has
bounded ASCII-to-`WCHAR` open, one-call partial read, progress-preserving write-all, explicit sync,
consuming close, and silent best-effort drop without hidden allocation. Its low and optimized
separate-translation-unit MSVC probes cover create/truncate, exact data, three-byte reads, EOF,
repeated close, forced `ERROR_INVALID_HANDLE`, drop, and repeated drop after COFF, PE, and unwind
inspection. An example-level safe FY module now adds move-only ownership, typed errors, filesystem
effects, consuming close, implicit drop, and `read(ref mut u8[])`; low and optimized `.fy` builds fill
the caller's fixed array and return 42 through local MSVC without WSL. A stable `std::fs`, portable
Unicode path, hosted receipt, general mutable-buffer operations, and public layout remain later gates.

The product build path now records sorted unique foreign symbols used by MIR calls and resolves the
project-owned subset through one shared symbol/component contract. It links only the minimal
dependency closure: Buffer and Arena pull in SystemHeap, and owned text pulls in SystemHeap, Buffer,
and text view. A second four-way build smoke compiles and executes a self-contained FY call to the
real UTF-8 validator. Public `std::*` modules, effects, safe resources, arbitrary link inputs, and an
observed Linux receipt remain later gates.

The source FFI slice admits `unsafe extern C fn` declarations using `i32`, `u8`, `u32`, `usize`, all
sixteen exact transport-only TargetSpec C scalars, raw pointers to the bounded Fairy scalar set or
opaque `void`, and raw pointers to nonempty acyclic `@repr(C)` records whose fields may include the
exact C scalars or nested `@repr(C)` records. The same nonempty acyclic records may now be foreign
results by value, while aggregate parameters remain closed. A single
`@link_name("symbol")` may separate the Fairy lookup name from the external C symbol. Unsafe call,
link identity, and raw-address provenance survive AST, typed HIR, and MIR verification; the C
backend emits unannotated prototypes, exact record tags/fields/layout assertions, and `&local` only
for an exact unsafe borrow-to-pointer cast. The native lane plans four scalar, four typed-null
pointer/target-scalar, four individual exact C-scalar, four remaining twelve-type C-scalar-family,
four register/hidden-memory struct-result, four real SystemHeap block, four real nested Buffer, four real Arena/Region, four real UTF-8 validation, and four real text-view output-record
crossings. The interpreter refuses
raw addresses and foreign execution. Dereference, pointer arithmetic, arbitrary casts, callbacks,
C-scalar value semantics, nested enum or array fields, by-value aggregate parameters, exports, and public standard-library
wrappers remain Phase 2 work. The internal empty-input text transports are real source-to-stdlib-C
exceptions, not completed wrappers.

The same shared generated-Fairy struct-result vector now also emits with the declared Windows
profile and crosses an independently compiled C callee through local x64 MSVC at low and optimized
profiles. Its 8-byte two-`u32` result exercises register-sized return transport and its 20-byte
five-`u32` result exercises the hidden-result-pointer path; both return 42 and both objects, linked
images, and unwind data pass the bounded Windows inspectors. The identical emitted C11 caller and
callee also repeat both profiles under the distinct C17 consumer mode, for four source executions
without creating a C17 output profile. This is one same-MSVC source-level
crossing, not independent compiler, register-inspection, broad aggregate, callback, export, or
hosted evidence.

Separate Windows/amd64 host-smoke commands now discover local x64 MSVC without WSL and compile one
target-neutral generated-C unit. `run-host` executes a temporary PE and reports the `i32` result or
canonical trap; `build-host` retains a no-overwrite `.exe` below `build/`. Both deliberately reject
FFI, `@repr(C)`, exact `c::*` scalars, and standard-library linkage. A formal fail-closed
`x86_64-windows-msvc` TargetSpec now records LLP64 and platform identities, and its separate
nine-case host-independent layout ledger covers the first LLP64-sensitive expectations. All eight
pass cases now cross a local MSVC C11 layout probe twice, but full scalar coverage and a hosted
receipt remain absent. A separate 15-case Microsoft x64 classifier ledger now proves four shared
ordinal slots, small-record integer transport, aggregate indirection, and hidden-return shifting;
its 12 pass plans now cross separate caller and callee C11 translation units through local x64 MSVC
at both profiles. One generated-Fairy small/large struct-result crossing now also executes at both
profiles. Independent compiler or assembly/register inspection, broader and bidirectional
source-level Fairy crossings, and a hosted Windows receipt remain absent. PE/COFF objects, runtime, standard library,
supported distribution artifacts, and conformance remain later gates. A bounded parser now checks
the host-smoke product, both declared-target profile products, both safe File and safe owned-text
profile products, two direct run-target temporary products, two exec-target temporary output
products, and twenty-four native probe executables for raw Windows File, raw Buffer, raw owned text,
raw Arena, layout, ABI, REL32 linking, archive/COMDAT selection, generated-Fairy struct results, and
unwind behavior for AMD64
PE32+ console identity, aligned
headers/sections, and an executable entry point, but does not inspect CodeView/PDB data.
Every local MSVC compile in the same lane now also parses its intermediate `.obj` as raw AMD64 COFF
before cleanup, including `.text$...`, `.bss`, primary symbols, relocation targets and AMD64 types,
and exact COMDAT section-definition selection. ADR 0231 adds two versioned raw-object vectors for
REL32, ANY, and associative COMDAT behavior. ADR 0232 separately proves one direct cross-object
`CALL rel32` formula by reading the object addend and comparing the exact final-image displacement at
both profiles.
ADR 0234 reuses those objects and executions to bind one `ADDR64` receipt pointer and all three
unwind `.pdata` `ADDR32NB` fields from exact section-indexed object relocations and raw addends
through MSVC map identities to their final-image values.
ADR 0233 additionally parses a real two-member Microsoft archive, proves the preferred index maps
the demanded and unused functions to their exact members, requires the demanded function to be
owned by the selected archive member in the final map, and requires the unused function to be a
non-COMDAT sentinel absent from that map. It also reads both candidate object initializers and binds
one pick-any COMDAT map owner to its final PE address and value across direct-first and
archive-first links at both profiles. Whole compiler-produced object identity, remaining relocation
formulas and unwind tails, broader archive retention and COMDAT selection policies, and a fresh hosted object receipt remain
absent, so the target evidence marker stays fail-closed.
The fifty-three local target or probe PE images now also cross a static Microsoft x64 unwind parser. It validates `.pdata`
runtime-function ordering and code coverage plus referenced version 1 and version 2 unwind-record
bounds, including handler/chained minimum tails. A separate three-object probe now raises a private
noncontinuable exception at both profiles, validates its payload in an outer SEH filter, walks the
captured context through raiser, termination, and catcher functions with `RtlVirtualUnwind`, and
requires the intervening `__finally` block to report one abnormal termination. This is native
dispatcher/runtime-walk evidence for closed MSVC C frames. Version-2 epilog meaning,
language-specific handler data, recursive chains, source-level Fairy or foreign-frame crossings,
object-level relocation expectations, and hosted evidence remain absent, so the unwind marker also
stays fail-closed.
The first declared Windows target build now selects a distinct C-backend profile rather than
labelling Linux-profile output after the fact. A checked-in FY source carries an unused `c::long`
foreign declaration; the emitted unit asserts four-byte LLP64 `long`, rejects unavailable
`c::ssize_t`, allocates and frees an eight-byte block through the real project-owned SystemHeap,
selects the Buffer/SystemHeap closure for initialize, length-four resize metadata, and drop, selects
the Arena/SystemHeap closure for initialize, Region begin, eight-byte allocation, reset, and drop, calls
the owned-text closure for String copy/view/append/drop, CString from-text/view/drop, and direct CStr
validation, calls the exact empty-input UTF-8 validator, compiles its generated and standard-library C units through
local x64 MSVC at low and optimized profiles, passes COFF/PE/unwind inspection, and returns 42 in
both executions. The Arena checks require generation 1 after initialization, active/non-null
allocation state, generation 2 with Region invalidation after reset, and generation 3 after drop;
they do not dereference the allocated bytes.
The owned-text checks observe successful results, five-to-ten String length, six-byte CString storage,
logical CStr length five, non-null views, and normalized drops without dereferencing the views.
Windows x64 SystemHeap uses the documented 16-byte `malloc` alignment ceiling
because MSVC C does not expose `max_align_t` in `<stddef.h>`.
A separate raw probe now links the owned-text closure as a second translation unit and executes at
both profiles. It observes exact bytes and termination, invalid UTF-8 and interior NUL, null and
malformed opaque storage, failure preservation, self-overlap, deterministic size overflow, and
normal plus repeated drop. It also forces initial and append-reallocation failure with a probe-only
allocator, preserves prior storage and bytes, and observes zero leaked live allocations after cleanup.
A raw Buffer probe independently links its caller and target-specific dependency closure at both
profiles. It forces initialization, reserve, self-overlap append growth, and zeroed-resize growth
failure, preserves exact pointer/metadata/bytes, then proves recovery content and final cleanup.
A parallel raw Arena probe links its caller and target-specific dependency closure separately at both
profiles. It observes zeroed and copied payloads, forces failure before payload allocation and after
payload allocation but before node publication, requires unchanged outputs, and proves reset returns
the shared probe allocator's live count to zero.
The Windows target commands now admit one entry plus a bounded explicit repeated-`--source` FY
module closure; the driver performs no manifest or implicit source discovery. Every foreign
companion input, unproven Windows standard-library component, user linker argument, debug output,
and packaging remain closed. ADR 0214 separately records the first hosted native receipt. The
Windows TargetSpec therefore reports
`scalar-and-llp64-c-scalar-c11` and no longer lists `target-c-emission-profile`, while all three
maturity booleans remain false.
The same target compilation is now exposed through `experimental run-target`. It validates PE and
Microsoft x64 unwind structure, executes a temporary optimized image of the checked-in FY target
example, requires no output and native result 42, and removes the image. The minimal Windows
quickstart iterates two bytes with typed `usize` ordering and checked addition, performs checked byte
indexing, and safely converts each `u8` to `i32` through a separately built compiler process. The
host smoke also proves maximum-`usize` addition traps under both MSVC profiles. The configured Windows
smoke therefore covers two retained target builds, four measured experimental plus three stable
quickstart and two safe `std::io` package-workspace builds, nine package-workspace runs, two safe File builds, two safe owned-text
builds, two direct target executions, two exec-target executions, and fifty-three total target or
probe PE images. Stable build proof covers development output, failed-build preservation, release
replacement, a safe `std::io` product with exact separated stdout/stderr and status, and locked replacements from a
separately built temporary `fairyc.exe`. Direct execution of that product additionally proves EOF,
a short stdin echo, and an eight-byte prefix from longer input. Stable run forwards that short input
through both the in-process CLI boundary and a separately built compiler.
Stable run proof covers development and release result 42, a normal reserved-status result 131, a
division-by-zero trap using that same OS status, canonical-lock restoration, retained-build
immutability, and another locked release run from that compiler. The
second direct target and experimental workspace executions also come from that compiler, so the
standalone process boundary is exercised without retaining or distributing it. This improves the
developer execution path without changing target maturity or claiming self-hosting.
The additional `experimental exec-target` path uses the same inspected temporary Windows target but
forwards separated bounded stdout and stderr without a compiler receipt and preserves the program's
32-bit Windows status. Its FY example now splits the safe entry and unsafe support boundary into two
explicitly listed modules, observes empty and invalid raw-write cases, derives the
request and progress lengths from borrowed-view `.len`, confines the Windows-only unsafe C symbol
to a checked local `ref str` helper, passes both borrowed-view families through safe internal calls,
executes two checked byte-view loads that observe exact NUL and `A` values, and covers the `bounds`
trap at both MSVC profiles,
emits the exact 16 ASCII bytes `Fairy C says hi\n`, and exits 42. Its `main` contains no raw pointer,
foreign call, or `unsafe`. The same
output/status contract is observed both in-process and through the separately built `fairyc.exe`.
This does not establish a safe or portable `std::io` surface, target support, or self-hosting.
The GitHub Actions Windows matrix executes that full smoke after `go test`, using the hosted
runner's local x64 MSVC and SDK with no WSL and retaining no artifacts. Run `30442310794` passed the
Windows job for exact commit `0bb4199`; ADR 0214 records the receipt and removes only the
`hosted-windows-receipt` marker. Incomplete ABI, object, unwind, runtime, standard-library,
toolchain-baseline, and provisioning gates remain.

- Implement `x86_64-linux-gnu` `TargetSpec`, toolchain resolution, constrained generated-C transport, and C-driver linking.
- Implement the thin runtime and minimum `SystemHeap`, `Arena`, `Buffer`, text views, `String`, `CString`, and Linux `File` surface required by representative fixtures.
- Run interpreter-to-C differential fixtures and turn the validated C ABI plans into bidirectional
  Clang/GCC call fixtures at low and optimized profiles.
- Keep sanitizer-assisted backend checks mandatory for every admitted C-emission case without
  treating sanitizers as the language oracle or exposing the validation mode as a product profile.

Exit gate: the target reaches `conformant`: interpreter and generated-C runs agree on values, status, trap category, active-payload and drop traces; all declared layout and C ABI fixtures pass in clean Linux CI with no skips. This gate does not by itself make the target `supported`.

## Phase 3: modules, package builds, and bounded caching

Current implementation status: the first concrete `PackageInstanceId` is now executable. It hashes
the already-resolved package tuple with a domain-separated, length-prefixed v1 encoding and full
SHA-256, exposes one canonical lowercase text form, rejects malformed feature sets, and rechecks the
complete tuple before digest-based deduplication. Versioned vectors cover component boundaries and
feature-set failures. Typed HIR also preserves source `pub` visibility for structs, enums,
resources, ordinary functions, and foreign functions while keeping compiler-created cleanup glue
private. A deterministic public-interface v3 projection now sorts modules and public items,
normalizes self nominal types independently from build-local numbering, renders dependency nominal
types through source-independent package coordinates, retains public callable must-use state, hashes
the canonical bytes with full SHA-256, and executes twelve source-level invalidation fixtures,
including private-module-constant and private-source-location stability. This closes package-instance and coordinate
encoding plus the current source-interface hash. An independent exact package-source
digest now sorts canonical relative paths and hashes every source byte, with five versioned cases
proving order stability plus content, path, and line-ending invalidation. The first whole-package
cache key combines those identities with every declared compiler, language, MIR, target, ABI,
toolchain, backend, runtime, standard-library, feature, and profile compatibility member. A
caller-bounded process-local LRU store proves byte/entry bounds, deep-copy isolation, and safe
eviction without inventing an unmeasured default. Registry coordinates, source acquisition,
serialized module interfaces, persistent cross-process storage, measurements, and stable package
commands remain Phase 3 work.

The manifest/lockfile readiness audit separates the fixed compiler graph contract from broader
package-manager choices. Its local-only, exact-version, per-edge-feature, explicit-workspace,
cycle-free, and network-free profile is now implemented through the manifest, workspace, lock,
experimental check command, and compiler-bundled SDK boundaries.

ADR 0142 and package fixture v1 now enforce the cycle-free part at the reusable resolved-graph
boundary, including self-cycles created only after exact-instance deduplication. ADR 0143 fixes the
strict `fairy.toml` v1 declaration surface through a pinned isolated TOML adapter. ADR 0144 adds
explicit filesystem-safe workspace/member/source loading and root-reachable feature-distinct graph
construction. ADR 0145 then fixes canonical generated `fairy.lock` v1, full identity revalidation,
stale-workspace comparison, and atomic replacement. ADR 0147 adds exact compiler-bundled SDK
standard-library injection and closed SDK lock records.
The unstable `experimental check-workspace` first connected those contracts to package-aware
semantic analysis and verified MIR with ordinary refresh and no-write `--locked` behavior. ADR 0175
now exposes that bounded current-workspace policy as stable `fairyc check [--locked]`. ADR 0148 adds
`experimental build-workspace` and `run-workspace`: the root package's qualified `main::main`
crosses the existing Windows C/MSVC path, both profiles produce and execute the checked-in
two-package quickstart, and a separately built compiler performs a locked run. ADRs 0176 and 0177
stabilize the narrow Windows-local build and run spellings; ADR 0180 separately stabilizes
target-explicit C application-unit emission. ADR 0181 now adds the root-only semantic test lane:
production loading excludes `tests/`, the test overlay validates `@test fn void`, and each entry
runs in a fresh interpreter state without a target toolchain or artifact.
ADR 0204 closes the first-project bootstrap gap with stable
`fairyc new <package-name> --uid <package-uid>`: one explicit identity, one exclusive child
directory, exact manifest/source/test bytes, ordinary-failure rollback, and generated-project
check/locked-check/test evidence without adding a template engine or package manager.
ADR 0206 moves that exact generated set to starter v2 with one compiler-owned README. It keeps the
output-free source and stable run path unchanged while carrying manual check/test/build guidance,
the run-v1 stream boundary, and a checked safe `std::io::write` replacement into the created
project without adding template selection or upgrade behavior.
ADR 0207 then replaces stable run's compiler receipt with bounded program-stream forwarding and
ordinary 32-bit status propagation while retaining exact trap separation. ADR 0208 aligns starter
v3 by returning zero after checking answer 42 and using stable locked run directly for safe stdout.

ADR 0152 now removes disposable HIR package numbering from compiler-private function identities.
The exact package-instance map, logical name, canonical signature types, effects, and drop-glue type
produce one full-digest `FunctionID`; package renumbering does not. This is a prerequisite for
explicit package-artifact imports, not evidence that a reusable package MIR or object exists.

ADR 0153 adds the first independently verified package MIR boundary. Package-owned functions and
explicit cross-package signatures are sliced from verified MIR, checked with verifier-only import
stubs, rebound to fresh internal symbols, reassembled, verified again, and executed through the
checked-in workspace to result `42`. The boundary is process-local and still duplicates the exact
workspace type context; it is not a persistent or compiler-integrated cache hit.

ADR 0154 adds a mandatory-bounds process codec. Its random identity makes payloads unusable by a
different codec or compiler process, while canonical decode and verifier re-entry let the existing
byte-bounded LRU own the complete allocation. This is cache plumbing, not a persistent format or
cache-hit measurement.

ADR 0155 connects the codec to the bounded LRU with expected-interface validation, verified hit
accounting, rejected-entry quarantine, and compare-and-delete protection for concurrent replacement.
ADR 0156 adds a semantic-first workspace compiler session. It computes every package interface and
v1 cache key, skips MIR lowering only after all function-owning packages are verified hits and exact
reassembly succeeds, and otherwise lowers normally and refreshes the cache. Tests prove one lowering
call across a cold and immediate warm two-package compile, source invalidation, mixed-generation
fallback, self-healing, and result `42`. This is deterministic control-flow evidence, not an elapsed
speedup claim, persistent cache, default policy, or one-shot CLI reuse.

ADR 0157 adds a second path-free measurement schema to the configured Windows smoke. It records one
cold fallback and one immediate all-hit request over the same Session, including raw durations,
lowering decisions, package counts, retained bytes, and interpreter result `42`. Validation forbids
a speedup decision. One local receipt observed two cold misses followed by two verified hits with no
warm lowering; this closed the control-flow measurement gap but, at that stage, not the
reference-machine, memory, or long-lived command gates.

ADR 0158 adds nine independent fresh-Session samples for each of no-op warm, private root edit,
private dependency edit, and public dependency edit. It keeps edit/load setup outside the measured
semantic-through-verified-MIR phase, preserves clock-resolution-censored raw observations, and
validates result `42`. No-op requests reuse both packages; a private root edit reuses the dependency;
both dependency edit classes conservatively invalidate the root because its v1 key includes the
dependency's exact instance identity. This closes the first repeated edit-class gate without making
a speedup, reference-machine, memory, or regression-threshold claim.

ADR 0159 adds the first real process boundary for that reuse. The foreground
`experimental check-workspace-session` accepts a bounded versioned NDJSON request stream, reloads
the workspace for every request, keeps one verified-MIR Session, reports per-request cache decisions,
and terminates at EOF or its required request bound. Callers choose every cache limit explicitly;
there is no daemon, watcher, socket, concurrency, persistent storage, or default policy. In-process
tests prove no-op all-hit reuse, private-root partial invalidation, and recovery from transient source
diagnostics. Its initial configured Windows smoke drove two requests through a separately built
`fairyc.exe` and required fallback followed by all-hit.

ADR 0160 replaces that minimal process probe with one eight-request editor-shaped sequence over a
temporary workspace. It preserves caller-observed round-trip durations, validates cold, no-op,
private-root, private-dependency, and public-dependency cache paths, and records Windows
process-reported peak working-set and pagefile-use bytes before exit. The run uses no retries and
makes no speedup, memory-budget, or reference-machine claim.

ADR 0161 adds the first cross-process-stable type-context representation. Structs, enums, resources,
foreign declarations, nested types, effects, and drop links project through `PackageInstanceId` and
`FunctionID`; bounded canonical decoding rejects malformed or noncanonical data, and equivalent
contexts retain identical bytes and IDs after build-local package and symbol renumbering. Existing
package artifacts recompute and verify that projection.

ADR 0162 adds stable package MIR v1. Canonical package bytes carry complete stable context, owned
function bodies, `FunctionID` calls/imports, and context-indexed foreign calls. Decoded packages can
rebuild the context and all functions under fresh package, function, foreign, and local numbers,
pass `mir.Verify`, execute the workspace to `42`, and re-encode identically. The complete workspace
context is still duplicated per package.

ADR 0171 supersedes that package-envelope rule with stable package MIR v2. Each package now stores
only the type, resource, drop-link, and foreign declarations required by its owned bodies. Decoded
package contexts merge only when repeated stable definitions agree exactly; each body retains its
own context identity and foreign-index table before the final program re-enters `mir.Verify`.

ADR 0163 adds a caller-bounded immutable disk-record store with canonical digest-named generations,
safe conflicting-generation misses, corrupt-record rejection, stale-temp recovery, and deterministic
entry/byte collection. ADR 0164 connects it to stable package MIR through an internal opt-in Session.
The caller must supply an exact compiler-build identity and every bound; complete hits rebuild fresh
identities, rerun `mir.Verify`, and execute result `42` across independent Sessions and separate
writer/reader test processes. Disk tests also cover concurrent process publication and process-exit
temporary recovery. ADR 0165 adds exact executable-byte build identity, and ADR 0166 adds one
explicit experimental command with a versioned JSON receipt. A separately built Windows `fairyc.exe`
now proves cold fallback followed by two verified disk hits. ADR 0167 expands that probe into nine
fresh-cache cold/warm pairs with separate compiler processes, raw caller-observed durations, strict
state validation, and zero retries. ADR 0168 adds deterministic pre/post-publication process kills;
ADR 0169 replaces its pathname-based Windows primitive with opened-root containment, handle-relative
rename, Windows final-file flush, and non-Windows directory sync. Interruption may yield a verified
hit or safe miss, never accepted partial cache truth. No ordinary-command cache, default
path or quota, universal storage-retention guarantee, reference-machine comparison, or speed claim
exists yet.

ADR 0170 added a nine-process edit staircase over one six-entry cache and exposed valid-generation
conflicts caused by duplicated workspace context. ADR 0171 removes that defect without weakening
immutable conflict detection. The v1 measurement now follows `2, 3, 4, 6, 6, 6, 6, 6, 6`, performs
three evictions, and records zero quarantines. Private dependency edits still miss both packages,
showing that public-interface projection remains broader than the actual reusable boundary.

ADR 0172 identifies the deeper cause: exact source-sensitive `PackageInstanceId` is also used as the
public link namespace in dependency cache pairs and `FunctionID`. It adds a separate UID/version/
features `PackageCoordinateId` prerequisite while leaving cache behavior unchanged. Interface hash,
stable symbol identity, and cache keys must adopt that coordinate together before private dependency
reuse is truthful.

ADR 0173 completes the interface consumer. ADR 0174 then migrates stable MIR function/type/foreign
identity and whole-package dependency keys as one versioned boundary. Private dependency edits now
reuse the verified caller while rebuilding the provider; public interface edits still invalidate
both. The measured six-entry staircase is `2, 3, 4, 5, 6, 6, 6, 6, 6`, with two evictions and zero
quarantines, without selecting a default policy or making a speed claim.

- Repeat the bounded session on a declared reference machine with controlled distributions, private
  or live-heap evidence, and an accepted noise policy before selecting defaults or claiming
  developer-loop speedup. The current local run has process peaks, an editor-shaped workload, and
  an explicit zero-retry policy but is not reference evidence.
- Keep persistent storage experimental and opt-in until broader retained-byte observations, a
  declared reference environment, and an accepted decision rule justify an ordinary-command cache
  path and defaults. The crash/power-interruption safety boundary and configured local cold/warm
  distributions now exist; physical media retention remains storage-stack-specific.
- Keep the implemented no-overwrite `fairyc new <package-name> --uid <package-uid>` starter v3 bootstrap,
  `fairyc check [--locked]`, Windows-local
  `fairyc build [--release] [--locked]`, and Windows-local
  `fairyc run [--release] [--locked]` v3 contract narrow: caller stdin after successful compilation,
  bounded stdout/stderr, and program status, but no arguments, input routing for other native
  commands, target selection, retained artifact, or cache policy. Keep the source-independent
  `fairyc explain <diagnostic-code>` and `fairyc targets` v1 contracts separate from catalog and
  TargetSpec data lifecycle. Keep target-explicit `fairyc emit c --target <target-id> [--locked]`
  separate from private generated-C bytes and support units. Keep implemented semantic
  `fairyc test [--locked]` separate from future native tests, workspace-member recursion, filters,
  fixtures, benchmarks, and host I/O.
- Move to function-level reuse only when measured package rebuild or storage evidence justifies the complexity.

ADR 0209 extends the compiler-bundled SDK to `0.0.0-stage0.3` with safe exact-byte
`std::io::write_stderr`. Both process-owned output streams now share one progress/error model and one
target-owned loop per platform. The checked-in workspace has exact separated Windows stream evidence;
formatting, stdin, arguments, terminal policy, and symbolic error kinds remain separate decisions.

ADR 0210 extends the SDK to `0.0.0-stage0.4` with `write_bytes(ref u8[])` and
`write_stderr_bytes(ref u8[])`. These calls reuse the existing stream symbols and error model while
making the no-UTF-8-validation binary boundary explicit. Formatting, input, arguments, terminal
rendering policy, and owned stream handles remain separate decisions.

ADR 0211 extends the SDK to `0.0.0-stage0.5` with
`read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>`. One call performs at most one bounded
read, success with zero means EOF, and callers own retry or fill loops. Windows direct-artifact
execution proves EOF, short input, and truncation at an eight-byte buffer; ADR 0214 records the
Linux adapter probe's first hosted execution. Text decoding, line buffering, stable CLI stdin forwarding,
arguments, terminal policy, and owned stream handles remain separate decisions.

ADR 0212 promotes caller stdin forwarding into stable run v3 without changing its spelling. Input is
attached only after compilation, linking, and image inspection, is neither pre-read nor retained by
the compiler, and remains governed by the existing process timeout. Program arguments, input routing
for other native commands, live output interleaving, and terminal policy remain separate decisions.

ADR 0215 extends the SDK to `0.0.0-stage0.6` with
`write_i32(i32) -> Result<usize, ConsoleError>`. It emits canonical locale-independent decimal ASCII
through the existing stdout progress contract, uses one 11-byte stack buffer, and defines
`INT32_MIN` through unsigned magnitude arithmetic. General formatting, floating-point conversion,
other integer widths, padding, bases, locale, and implicit newlines remain separate decisions.

ADR 0216 cuts the exact language pin to `fairyc-language/2` and adds statement-only checked integer
compound assignment. All ten arithmetic, division, remainder, shift, and bitwise forms reuse the
existing exact-type trap semantics; mutable local and indexed places evaluate once and store only
after success. Classic `for` post clauses may now use `index += 1`. Assignment expressions,
increment operators, promotions, wrapping operations, resource targets, and general projections
remain absent.

ADR 0217 cuts the exact language pin to `fairyc-language/3` and adds bounded lexical `defer`.
Internal `void` calls with copy-scalar arguments capture those arguments immediately and execute in
LIFO order on fallthrough, return, Result propagation, break, or continue. Trap paths remain
non-unwinding; resource, borrow, aggregate, pointer, method, and foreign-call capture stay closed.

ADR 0219 cuts the exact language pin to `fairyc-language/4` and implements narrow callable
`@must_use`. Eligible internal, method, imported, and foreign call results require consumption or
explicit `discard`; malformed or ineligible declarations use `E4201` and ignored calls use `E4202`.
Typed HIR retains the bit, public-interface v3 hashes public changes while excluding private ones,
and MIR, generated C, and runtime behavior remain unchanged after semantic acceptance.

ADR 0220 cuts the exact language pin to `fairyc-language/5` and implements private
`bool`/`u8`/`i32`/`u32` constants, private straight-line `@const_eval` functions, and contextual
module `static_assert`. Checked compile-time traps use `E4303`, cycles and budgets use `E4304`,
runtime references enter HIR as folded scalar literals, and no evaluator state reaches MIR or C.

Exit gate: clean and warm build evidence is reproducible on a declared machine and workload; invalidation fixtures distinguish public interface, inline body, and private implementation; incompatible caches are safe misses.

## Phase 4: Atlas fact export

Current implementation status: ADR 0182 adds the first executable one-way compiler-fact boundary.
`internal/semanticsnapshot` accepts only semantic Flow success, reverified MIR, and complete exact
package metadata; removes disposable package/symbol qualification; emits detached canonically
ordered package, module, declaration, function-summary, and relation-completeness records; and
keeps foreign-call uncertainty local to the call relation. Package-renumbering, input/accessor
mutation, incomplete metadata, unverified MIR, and host/build-local identity leakage have direct
regressions. ADR 0183 and `internal/symbolidentity` now derive a provisional deterministic
`fairyc.symbol-id/v1-sha256-128+256` from package UID, semantic declaration coordinates, and a
canonical structured signature. The implementation deliberately ignores package instances,
versions, bodies, spans, and build numbering; normalizes nominal types through package UID; retains
both a 128-bit visible prefix and full 256-bit digest; and rejects malformed pairs or visible
collisions. Canonical encoding, component-boundary, rename/signature sensitivity,
body/span/version/build-number stability, path rejection, and collision behavior have direct
regressions. ADR 0184 now adds domain-separated `fyfile1-sha256` content and `fylm1-sha256`
line-map identities, projects them from validated workspace bytes, and requires canonical complete
file/module coverage in every snapshot without retaining source text. Exact LF/CRLF/final-newline,
content-versus-geometry, source/test classification, order, detachment, and missing/extra coverage
regressions exist. ADR 0185 and `internal/atlasmap` now close the next in-memory boundary: the
validated workspace, compiler result, exact executable identity, and reusable complete package
interface-hash set produce one immutable deterministic logical partition per package. Headers retain
every independent identity algorithm and exact interface hash; every file, module, declaration,
SymbolId, and function summary is assigned; relation gaps become stable omissions. ADR 0186 now
encodes those partitions as positive-caller-bounded canonical `fairyc.fymap-json/v1`, protects the
schema/encoding/document tuple with `fairyc.fymap-checksum/v1-sha256`, strictly rejects alternate
JSON and malformed records, and rederives every SymbolId from the declaration plus exact package
namespace references. ADR 0187 now adds the internal opened-root `MapStore`: checksum-named immutable
generations, mandatory caller retention bounds, corrupt/conflict rejection, stale temporary recovery,
process-interruption miss-or-verified-hit behavior, and exact current compiler/interface/package/file/
line-map acceptance all have executable regressions. ADR 0188 now adds exact-current package-local
`overview`, `show`, and `around`: canonical SymbolId selection, typed relation expansion,
deterministic breadth-first traversal, mandatory record/relation/depth/selection-byte limits, detached
results, and explicit filter/dependency/incoming omissions have executable regressions. ADR 0189 now
adds output-only canonical compiler-fact Context Packs for all three queries, with exact package and
compatibility identity, echoed budgets and usage, empty text sections, a separate final byte bound,
and a domain-separated checksum. ADR 0190 exposes `experimental atlas-pack overview` with
an exact package UID and explicit record/selection/output bounds; it writes canonical stdout bytes
and no map-store path. ADR 0191 now measures nine in-memory canonical rewrites of the complete fixed
quickstart package set after planning, with source/record/encoded-byte denominators and exact byte
identity. It makes no format, chunking, compression, incremental-update, or latency decision. No
default store path, ordinary-command publication, source-excerpt output in those commands,
documentation import, or pack decoder exists yet.
ADR 0192 supplies the internal exact discovery prerequisite: full-SymbolId ordering, exact module
and kind filters, explicit count/byte bounds, and full-ID stateless cursors all have real-workspace
regressions without fuzzy name inference. ADR 0193 exposes those pages through bounded canonical
`experimental atlas-symbols list` output with no store path, fuzzy search, or hidden default. ADR
0201 adds an optional exact Stage 0 declaration-name filter, retains every same-name match across
modules and kinds, rejects cursors outside the filtered set, and moves the closed output contract to
Symbol Catalog v1 instead of mutating v0.
ADR 0194 now exposes exact visible/full SymbolId `atlas-pack show` and `around` with mandatory
direction, relation, record, relation, depth, selection-byte, and output-byte policy. The commands
preserve package-local join omissions and add no text, decoder, store, or safe-change claim.
ADR 0225 now supplies the narrower cross-package prerequisite without changing those commands. Its
bounded in-memory `WorkspaceCallIndex` revalidates the complete caller-supplied package set, joins
exact compiler-observed calls to dependency SymbolIds, and builds detached outgoing and incoming
ledgers. Missing dependencies, stale maps, target contradictions, and package or call budget
exhaustion reject the join. It adds no workspace traversal, output schema, persistence, or impact
claim. ADR 0226 is its first deliberately narrow consumer: `experimental atlas-calls list` selects
one exact package-qualified visible/full SymbolId plus outgoing, incoming, or both direction under
mandatory package, call, and final-output bounds. It emits a separate canonical checksummed
workspace-call catalog v0 and leaves Context Pack v1 package-local. This direct-call slice adds no
transitive traversal, other relation family, default budget, persistence, or safe-change claim.
ADR 0227 adds the first process-level control without promoting that slice. The Windows harness runs
nine fresh locked standalone catalog processes after excluded discovery and warmup, preserves raw
wall time, per-process Windows working-set/pagefile peaks, exact source/call/output denominators, and
explicit construction bounds. The tiny quickstart, uncontrolled OS caches, and local machine support
no reference budget, speedup, Atlas-only heap attribution, representative-use conclusion, traversal,
or impact decision.
ADR 0228 adds a separate generated scaling control for exact join-plus-encode work at 2 packages and
8 calls, 8 packages and 64 calls, and 32 packages and 512 calls. It preserves nine raw time,
process-wide allocation, and output-size samples per shape after preparation. The star topology,
uncontrolled garbage collection, and local-only observations still support no complexity law,
retained-memory attribution, reference budget, default, traversal, impact, or regression decision.
ADR 0229 closes the immediate star-only correctness blind spot with generated chain, diamond, and
fan-in workspaces that pass the ordinary manifest, loader, compiler, verified-MIR, map, join, and
canonical catalog paths. Exact outgoing, incoming, and bidirectional counts are executable, but
synthetic topology coverage is still not representative repository evidence.
ADR 0230 adds a separate post-forced-GC live-heap control over batches of 16 retained workspace-call
indexes for the three generated star shapes. It keeps prepared inputs live and records raw process
`HeapAlloc`/`HeapObjects` deltas, but process attribution, synthetic inputs, and absent reference-
environment evidence still prohibit per-index budgets, defaults, traversal, impact, or regression
claims.
ADR 0195 closes the attributed-text origin, exact-current identity, UTF-8/control, atomic fragment
budget, integrity, omission, and non-authority policy. ADR 0196 implements one workspace-authored
exact source excerpt from revalidated retained bytes, and ADR 0197 adds its separately bounded
canonical output-only JSON with exact subject/selection evidence and document checksum. ADR 0198
exposes it through default-free `experimental atlas-text source` with exact SymbolId selection,
explicit origin/count/text/output policy, and checksummed omissions. There is no file reopen,
decoder, persistent text store, or change to body-free `.fymap` v1 and empty-text Context Pack v1.
ADR 0199 now establishes the prerequisite documentation fact: consecutive physical-line-leading
`///` trivia attaches conservatively to supported top-level declarations and survives CST, AST, and
typed HIR as an exact byte span without entering executable semantics or Atlas transports.
ADR 0200 now carries that fact through an immutable in-memory snapshot sidecar and exposes exact
retained marker-bearing bytes through default-free `experimental atlas-text documentation`.
Undocumented declarations produce `kind-unavailable`; map and Context Pack v1 bytes stay unchanged.
ADR 0202 now composes one exact-symbol Context Pack v1, source attributed-text v0, and documentation
attributed-text v0 document through `experimental atlas-evidence show`. Each child remains complete
and independently checksummed, the wrapper has no authority, facts and untrusted text remain visibly
separate, all query/text/component/final limits are explicit, and the existing formats stay
byte-unchanged.
ADR 0203 now parses every repository `cmd/` and `internal/` Go import and permits Atlas/export imports
only inside that layer, CLI orchestration, and the Windows evidence harness. Mutation-sensitive tests
require compiler and MIR imports to fail, closing the previously prose-only dependency direction.

- Preserve the implemented immutable `SemanticSnapshot` and per-package logical map plan.
- Preserve the implemented bounded canonical `.fymap` codec and semantic revalidation boundary.
- Preserve the implemented contained immutable store and current-source rejection boundary.
- Preserve the implemented bounded exact-current package-local query boundary.
- Preserve the implemented canonical compiler-fact-only Context Pack boundary.
- Preserve the output-only single-symbol Evidence Bundle v0 composition boundary; keep child authority and byte limits explicit and add no decoder or Context Pack text population.
- Preserve the explicit overview/show/around experimental commands and keep map publication policy separate.
- Preserve the raw whole-package rewrite harness; require representative failure before adding chunks or compression.
- Preserve the raw fresh-process, fixed-shape scaling, retained-heap batch, and non-star correctness controls; require representative real package/call distributions and a declared reference environment before selecting defaults, budgets, or traversal.
- Preserve the implemented exact module/name/kind SymbolId Catalog v1 output as the discovery path for experimental `show` and `around`; keep fuzzy ranking and guessed uniqueness out.
- Preserve the implemented attributed-text v0 model, exact retained source/documentation gates, canonical output-only JSON, explicit experimental source/documentation/evidence commands, and exact `///` frontend attachment sidecar; keep additional origins, multi-record composition, decoding, and rendering behind separate decisions.

Exit gate: satisfied for the current Stage 0 scope. `docs/atlas/phase4-exit-evidence.md` maps the
one-way import boundary, stale/incompatible-map rejection, explicit unknown edges, and text
non-authority to separate executable oracles and records what each still does not prove. Later
`impact`, semantic `diff`, anchors, MCP, and structural edits remain gated below.

## Later capability gates

- `x86_64-linux-musl`: requires a separate libc, sysroot, startup, loader, and static-distribution profile.
- `x86_64-windows-msvc`: the TargetSpec, scalar/LLP64 C-emission product builds at both profiles, closed FY-to-project-C SystemHeap, Buffer, Arena/Region lifecycle, and UTF-8 calls, an eight-case local layout probe, 15-case host-independent Microsoft x64 classifier, 12-case same-MSVC native crossing, one generated-Fairy small/large struct-result crossing under C11 and C17 consumer modes, bounded PE32+ executable inspection, structural live-COFF object/archive inspection, versioned exact synthetic COFF symbol/COMDAT/relocation fixtures, inspected cross-object `REL32`, `ADDR64`, and complete three-field unwind `.pdata` `ADDR32NB` formulas, one exact archive extraction and pick-any COMDAT final-selection proof, static version-1/version-2 unwind-structure inspection, and a three-case native exception/virtual-walk probe exist, but complete hosted LLP64 and same-commit COFF evidence, broader and bidirectional source-level Fairy crossings, independent or register-inspected calls, remaining relocation formulas and unwind tails, broader archive/COMDAT policies, complete version-2/handler/chain and Fairy/foreign-frame unwind fixtures, complete UCRT/VCRuntime and deployment evidence, distribution, and debug fixtures remain required.
- QBE or another fast backend: requires stable internal semantics, interpreter and C oracles, ABI coverage, and measured edit-build benefit.
- LLVM backend: requires representative workloads that justify optimization and distribution cost after simpler paths are exhausted.
- Generics and `Vec<T>`: require non-generic stability plus compile-time, code-size, active-payload, and diagnostics fixtures.
- Immutable green tree, incremental reparse, formatter, LSP, and SCIP: require an editor product slice and measured parser needs.
- `impact`, semantic `diff`, stable anchors, MCP, and structural edits: require `.fymap` identity and completeness evidence from real use.
- Custom allocators, portable paths, resource-aware arenas, closures, concurrency capabilities, async, sockets, and freestanding or embedded support each require a separate ownership and target contract.

## Self-hosting gate

Self-hosting starts only after modules, strings, containers, `Result`, resource moves, C FFI, package builds, source maps, and the test runner can support the compiler without freezing immature language decisions. The Go compiler remains a bootstrap and reference implementation.
