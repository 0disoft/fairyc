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

Classic and range `for`, lexical `defer`, provider-qualified package imports, and other new semantic
features do not ride this gate merely because they appeared in the same design review. Expression-
bodied function shorthand is also deferred until real source justifies a second body form. Each
addition needs a separate accepted contract and executable fixture matrix.

## Phase 1: first interpreted semantic slice

Current implementation status: the source, lexer, recoverable-syntax, parser-diagnostic, AST
lowering, and parser-fixture-runner portion of the first item exists. Name resolution and typed HIR
now cover the semantic fixture's scalar and safe-borrow subset. Local last-use loans, borrow
  conflicts, single-origin borrow returns, checked matching-type left shift, loop-context diagnostics,
  scalar foreign-call unsafe/signature boundaries, and `@link_name` shape diagnostics execute across
  all 38 semantic v0 cases, including branch- and loop-aware safe-borrow reassignment, exact
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
  execute all 11 MIR normalization v0 cases for the current scalar, resource, tagged-payload, Result, Region, loop-control cleanup, call,
  division, and checked-arithmetic trap subset. All 10 verifier baselines and 21 isolated mutation cases execute through
  the same MIR verifier. The first MIR interpreter executes all 92 interpreter v0 cases, including direct `i32` minimum constants, scalar fixed-array calls, contextual immutable and mutable byte-view borrowing, exact lengths, checked reads and indexed stores, and
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
release-stable `fairyc check` semantics remain a later gate.

Verified resource definitions now preserve ordered fields and
  drop classification from AST through HIR into MIR; nonempty user drop bodies now lower to
  verifier-linked private functions and execute before reverse field cleanup. Resource-declared
  effects are preserved and required only at actual cleanup edges. Restricted inert enums now preserve named payloads,
  construct through contextual leading-dot variants, require exhaustive shallow switches, and execute through
  verified MIR, the interpreter, and the constrained C backend. The C backend now emits deterministic C11 for 36
  verified scalar, borrowed-view, typed-comparison, inert-enum, flat-resource, and resource-tagged cases under the provisional `x86_64-linux-gnu` profile,
  including `Option<i32>` and `Result<i32, i32>` construction, switch, calls, and propagation,
  copy-scalar/inert-struct tagged transport, private
  positive scalar fixed-array construction, contextual zero-copy shared and mutable `u8[N]` byte-view borrowing, exact length observation, internal value calls, and checked reads and indexed stores, plus
  flat copy-resource construction, field reads, moves, reassignment, and resource-bearing built-in
  tag variants and propagation. The unstable
  `experimental emit-c` command exposes that exact verified translation unit on stdout
  without writing an artifact. On an eligible Linux/amd64 glibc host, `experimental build` can now
  compile and link that subset with explicit Clang or GCC and low or optimized profiles into
  a no-overwrite `build/` artifact. Stable portable file and console packages, effect-bearing function values,
  and every later Phase 1
  capability remain unimplemented.

- Implement source management, lexer, trivia-preserving tokens and spans, compact recoverable syntax, deterministic diagnostics, one-file modules, alias imports, and non-generic typed HIR in Go.
- Implement restricted inert enums, whole-resource moves, local last-use loans, call-bounded mutable reborrows, single-origin borrow returns, plain regions, `Option`, `Result`, shallow `switch`, and explicit `?` and `discard`.
- Implement Flow CFG, mutable-local MIR, drop elaboration, the verifier, target-independent layouts, and the compatibility-vector metadata model.
- Build a small MIR interpreter covering scalar and tagged behavior, resource payloads, drop traces, region allocation, a controlled in-memory I/O test harness, and traps.

Exit gate: positive and negative fixtures agree with the source contracts, resource-bearing built-in variants have correct active-payload traces, and every named verifier seed passes before its isolated mutation demonstrates the intended single `Vxxxx` rejection.

## Phase 2: first hosted executable slice

Current implementation status: a fail-closed Linux/amd64 constrained-C differential command now
requires glibc, Clang, and GCC and is wired into a separate hosted Ubuntu CI job. It is designed to
run all 36 emission cases at low and optimized profiles for 144 process-status and exact
source-linked drop-trace comparisons, then drive
the actual experimental build CLI with resource-bearing Option and Result plus a flat all-`i32` resource
through 4 compiler/profile executable
smokes. It now adds four-way product builds containing compiler-owned nonempty `ref CStr` storage and a
private compiler-owned `ref u8[]` view, plus separate generated-Fairy-to-C calls that verify every byte
of `fairy\0` and every non-terminated byte of `fairy` respectively. The same job
also compiles and executes each production, uninstrumented C unit with Clang
and GCC under the closed `address-undefined-o1` ASan+UBSan mode for 72 fail-closed sanitizer
executions. The current Windows
host has no eligible Linux toolchain, and no hosted run receipt has been observed, so this is a
configured lane rather than native conformance evidence. A shared bounded C driver and embedded
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
Its four C-only native probe executions and four generated-Fairy source-call executions are
configured but unobserved. The internal caller uses the exact nested SystemHeap-block record and
reaches initialize, zeroed resize, and drop. Used-symbol product linkage now supplies the exact
Buffer/SystemHeap closure, but public `std::bytes` and safe ownership methods remain missing. A raw
Buffer caller now executes against an instrumented target-specific closure at both MSVC profiles.
It forces initialization, reserve, relocating self-overlap append, and zeroed-resize allocation
failure, preserves exact storage state and initialized bytes, proves recovery zeroing, and returns
the live-allocation count to zero after drop.
The Arena storage slice now adds a single-active-Region generation check, failure-atomic zeroed and
copied byte placement, reset invalidation, stale-use rejection, and active-drop rejection. Its four
C-only executions and four generated-Fairy source-call executions are configured but unobserved.
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
including overlong, surrogate, out-of-range, truncated, and obsolete forms. Its four native runs
are configured but unobserved. A separate four-way Fairy-source crossing now calls the actual
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
close including forced close failure. Its four C-only native runs remain configured but unobserved.
A second four-way generated-Fairy crossing uses exact result, File, and open-option records to call
the real open, write-all, sync, close, reopen, partial-read, EOF, and final-close helpers. It checks
exact counts, repeated-close status, normalized closed state, and reconstructed bytes. Public safe
File ownership, source effects, typed errors, and hosted execution evidence remain missing.
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
products, and eighteen native probe executables for raw Windows File, raw Buffer, raw owned text,
raw Arena, layout, ABI, generated-Fairy struct results, and
unwind behavior for AMD64
PE32+ console identity, aligned
headers/sections, and an executable entry point, but does not inspect `.obj` symbols, relocations,
unwind payloads, or CodeView/PDB data.
Every local MSVC compile in the same lane now also parses its intermediate `.obj` as raw AMD64 COFF
before cleanup, including `.text$...`, `.bss`, symbol-table, and relocation-index structure. Exact
versioned symbol names, COMDAT selection, relocation types, unwind records, and a hosted object
receipt remain absent, so the target evidence marker stays fail-closed.
The twenty-nine local target or probe PE images now also cross a static Microsoft x64 unwind parser. It validates `.pdata`
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
packaging, and hosted receipt remains closed. The Windows TargetSpec therefore reports
`scalar-and-llp64-c-scalar-c11` and no longer lists `target-c-emission-profile`, while all three
maturity booleans remain false.
The same target compilation is now exposed through `experimental run-target`. It validates PE and
Microsoft x64 unwind structure, executes a temporary optimized image of the checked-in FY target
example, requires no output and native result 42, and removes the image. The minimal Windows
quickstart iterates two bytes with typed `usize` ordering and checked addition, performs checked byte
indexing, and safely converts each `u8` to `i32` through a separately built compiler process. The
host smoke also proves maximum-`usize` addition traps under both MSVC profiles. The configured Windows
smoke therefore covers two retained target builds, two safe File builds, two safe owned-text builds,
two direct target executions, two exec-target executions, and twenty-nine total target or probe PE
images. The second direct execution comes from a separately built temporary
`fairyc.exe` compiling the minimal Windows quickstart, so the standalone compiler process boundary
is exercised without retaining or distributing it. This improves the developer execution path
without changing target maturity or claiming self-hosting.
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
The GitHub Actions Windows matrix is now configured to execute that full smoke after `go test`,
using the hosted runner's local x64 MSVC and SDK with no WSL and retaining no artifacts. The
`hosted-windows-receipt` marker deliberately remains missing until an exit-zero workflow run is
observed for the exact commit; workflow text is not execution evidence.

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
private. A deterministic public-interface v1 projection now sorts modules and public items,
normalizes self and dependency nominal types independently from build-local numbering, hashes the
canonical bytes with full SHA-256, and executes eight source-level invalidation fixtures. This closes
package-instance encoding and the first source-interface hash. An independent exact package-source
digest now sorts canonical relative paths and hashes every source byte, with five versioned cases
proving order stability plus content, path, and line-ending invalidation. The first whole-package
cache key combines those identities with every declared compiler, language, MIR, target, ABI,
toolchain, backend, runtime, standard-library, feature, and profile compatibility member. A
caller-bounded process-local LRU store proves byte/entry bounds, deep-copy isolation, and safe
eviction without inventing an unmeasured default. Public manifests, lockfiles, coordinates, source
acquisition, serialized module interfaces, persistent cross-process storage, measurements, and
stable package commands remain Phase 3 work.

The manifest/lockfile readiness audit now separates the fixed compiler graph contract from ten
unresolved package-manager choices. Its recommended first profile is local-only, exact-version,
per-edge-feature, explicit-workspace, cycle-free, and network-free; it remains a recommendation
until an ADR accepts the wire format and parser strategy.

ADR 0142 and package fixture v1 now enforce the cycle-free part at the reusable resolved-graph
boundary, including self-cycles created only after exact-instance deduplication. ADR 0143 fixes the
strict `fairy.toml` v1 declaration surface through a pinned isolated TOML adapter. ADR 0144 adds
explicit filesystem-safe workspace/member/source loading and root-reachable feature-distinct graph
construction. ADR 0145 then fixes canonical generated `fairy.lock` v1, full identity revalidation,
stale-workspace comparison, and atomic replacement. SDK selection and public CLI lock modes remain.
The unstable `experimental check-workspace` now connects those contracts to package-aware semantic
analysis and verified MIR with ordinary refresh and no-write `--locked` behavior. Stable command
spelling and package-aware build/run still remain.

- Add exact SDK standard-library injection and public ordinary/`--locked` CLI policy on top of the
  accepted manifest, workspace loader, and lockfile library.
- Wire whole-package reuse into package compilation, then adopt persistent content-addressed storage
  and default limits only from measured clean/warm workload evidence.
- Stabilize the minimal `fairyc check`, `build`, `run`, `test`, `emit`, `explain`, and `targets` contracts.
- Move to function-level reuse only when measured package rebuild or storage evidence justifies the complexity.

Exit gate: clean and warm build evidence is reproducible on a declared machine and workload; invalidation fixtures distinguish public interface, inline body, and private implementation; incompatible caches are safe misses.

## Phase 4: Atlas fact export

- Produce an immutable `SemanticSnapshot` from analyzed HIR and verified MIR.
- Export `.fymap` with package UID and instance identity, compatibility vector, provenance, relation-specific completeness, and deterministic symbol identity.
- Implement bounded `overview`, `show`, and `around` queries and Context Pack output.
- Measure whole-package map rewrite cost before considering chunks or compression.

Exit gate: Atlas failure cannot change compilation; stale or incompatible maps fail closed; unknown graph edges remain explicit; text cannot grant authority.

## Later capability gates

- `x86_64-linux-musl`: requires a separate libc, sysroot, startup, loader, and static-distribution profile.
- `x86_64-windows-msvc`: the TargetSpec, scalar/LLP64 C-emission product builds at both profiles, closed FY-to-project-C SystemHeap, Buffer, Arena/Region lifecycle, and UTF-8 calls, an eight-case local layout probe, 15-case host-independent Microsoft x64 classifier, 12-case same-MSVC native crossing, one generated-Fairy small/large struct-result crossing under C11 and C17 consumer modes, bounded PE32+ executable inspection, structural live-COFF object inspection, static version-1/version-2 unwind-structure inspection, and a three-case native exception/virtual-walk probe exist, but complete hosted LLP64 layout evidence, broader and bidirectional source-level Fairy crossings, independent or inspected call evidence, versioned exact COFF symbol/COMDAT/relocation fixtures, complete version-2/handler/chain and Fairy/foreign-frame unwind fixtures, complete UCRT/VCRuntime and deployment evidence, distribution, and debug fixtures remain required.
- QBE or another fast backend: requires stable internal semantics, interpreter and C oracles, ABI coverage, and measured edit-build benefit.
- LLVM backend: requires representative workloads that justify optimization and distribution cost after simpler paths are exhausted.
- Generics and `Vec<T>`: require non-generic stability plus compile-time, code-size, active-payload, and diagnostics fixtures.
- Immutable green tree, incremental reparse, formatter, LSP, and SCIP: require an editor product slice and measured parser needs.
- `impact`, semantic `diff`, stable anchors, MCP, and structural edits: require `.fymap` identity and completeness evidence from real use.
- Custom allocators, portable paths, resource-aware arenas, closures, concurrency capabilities, async, sockets, and freestanding or embedded support each require a separate ownership and target contract.

## Self-hosting gate

Self-hosting starts only after modules, strings, containers, `Result`, resource moves, C FFI, package builds, source maps, and the test runner can support the compiler without freezing immature language decisions. The Go compiler remains a bootstrap and reference implementation.
