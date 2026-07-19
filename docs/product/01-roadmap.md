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

Exit gate: the written contracts have no contradictory syntax, initialization, ownership, cleanup, package identity, visibility, target, runtime, or compatibility rule, and every mandatory rule has an observable fixture class or an explicit dependency-gated deferral. Parser, semantic, resource, Region, Result, MIR, and package manifest cases must pass their future full runners with no unlisted independent diagnostics or observation drift.

## Phase 1: first interpreted semantic slice

Current implementation status: the source, lexer, recoverable-syntax, parser-diagnostic, AST
lowering, and parser-fixture-runner portion of the first item exists. Name resolution and typed HIR
now cover the semantic fixture's scalar and safe-borrow subset. Local last-use loans, borrow
  conflicts, single-origin borrow returns, Stage 0 signed-left-shift rejection, loop-context diagnostics,
  scalar foreign-call unsafe/signature boundaries, and `@link_name` shape diagnostics execute across
  all 23 semantic v0 cases, including exact transport for all sixteen TargetSpec C scalars. General
  whole-root resource Flow now executes all 12 resource v0 cases, including branch, normal-loop,
  break-edge, and continue-edge state rejection. Result Flow executes all 24 result v0 cases, including qualified scalar construction,
exhaustive shallow matching, and local, parameter, branch, loop, overwrite, and early-return
must-use obligations. Mutable-local MIR lowering and verification
  execute all 11 MIR normalization v0 cases for the current scalar, resource, tagged-payload, Result, Region, loop-control cleanup, call,
  division, and checked-arithmetic trap subset. All 10 verifier baselines and 21 isolated mutation cases execute through
  the same MIR verifier. The first MIR interpreter executes all 56 interpreter v0 cases, including
scalar `Option` and `Result` construction, exhaustive matching, and both Result `?` paths, and is
exposed by the unstable `experimental run` command. Resource construction, field reads, whole-root
call moves, source-linked drop traces, resource-bearing `Option` and `Result` payload moves, ordered
resource reassignment, explicit consuming-close success/failure ownership, and normal lexical
Region placement, inner cleanup, normal reset, direct return, Result propagation, and nested
early-exit reset execute. Manifest-authorized in-memory stdout/stderr byte hooks prove output order
and bounded failure without granting host access. Checked `i32` addition, subtraction,
multiplication, negation, division, remainder, and signed right shift execute across trapping and
  non-trapping boundaries with exact overflow, division-by-zero, or invalid-shift
  categories; signed left shift is rejected as `E1011`. Nearest-loop `break` and `continue` now execute
  with resource and Region cleanup. Verified resource definitions now preserve ordered fields and
  empty-drop classification from AST through HIR into MIR; nonempty user drop bodies fail as
  unsupported before executable lowering. Restricted inert enums now preserve named payloads,
  construct through qualified variants, require exhaustive shallow matches, and execute through
  verified MIR, the interpreter, and the constrained C backend. The C backend now emits deterministic C11 for 18
  verified scalar, inert-enum, flat-resource, and resource-tagged cases under the provisional `x86_64-linux-gnu` profile,
  including `Option<i32>` and `Result<i32, i32>` construction, match, calls, and propagation plus
  all-`i32` resource construction, field reads, moves, reassignment, and resource-bearing built-in
  tag variants and propagation. The unstable
  `experimental emit-c` command exposes that exact verified translation unit on stdout
  without writing an artifact. On an eligible Linux/amd64 glibc host, `experimental build` can now
  compile and link that subset with explicit Clang or GCC and low or optimized profiles into
  a no-overwrite `build/` artifact. Real file and console I/O, effects, and every later Phase 1
  capability remain unimplemented.

- Implement source management, lexer, trivia-preserving tokens and spans, compact recoverable syntax, deterministic diagnostics, one-file modules, alias imports, and non-generic typed HIR in Go.
- Implement restricted inert enums, whole-resource moves, local last-use loans, call-bounded mutable reborrows, single-origin borrow returns, plain regions, `Option`, `Result`, shallow `match`, and explicit `?` and `discard`.
- Implement Flow CFG, mutable-local MIR, drop elaboration, the verifier, target-independent layouts, and the compatibility-vector metadata model.
- Build a small MIR interpreter covering scalar and tagged behavior, resource payloads, drop traces, region allocation, a controlled in-memory I/O test harness, and traps.

Exit gate: positive and negative fixtures agree with the source contracts, resource-bearing built-in variants have correct active-payload traces, and every named verifier seed passes before its isolated mutation demonstrates the intended single `Vxxxx` rejection.

## Phase 2: first hosted executable slice

Current implementation status: a fail-closed Linux/amd64 constrained-C differential command now
requires glibc, Clang, and GCC and is wired into a separate hosted Ubuntu CI job. It is designed to
run all 18 emission cases at low and optimized profiles for 72 process-status and exact
source-linked drop-trace comparisons, then drive
the actual experimental build CLI with resource-bearing Option and Result plus a flat all-`i32` resource
through 4 compiler/profile executable
smokes. It now adds four-way product builds containing compiler-owned nonempty `&CStr` storage and a
private compiler-owned `&[u8]` view, plus separate generated-Fairy-to-C calls that verify every byte
of `fairy\0` and every non-terminated byte of `fairy` respectively. The same job
also compiles and executes each production, uninstrumented C unit with Clang
and GCC under the closed `address-undefined-o1` ASan+UBSan mode for 36 fail-closed sanitizer
executions. The current Windows
host has no eligible Linux toolchain, and no hosted run receipt has been observed, so this is a
configured lane rather than native conformance evidence. A shared bounded C driver and embedded
scalar runtime serve both that lane and the experimental single-file build command, while a separate
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
Buffer/SystemHeap closure, but public `std::bytes` and safe ownership methods remain missing.
The Arena storage slice now adds a single-active-Region generation check, failure-atomic zeroed and
copied byte placement, reset invalidation, stale-use rejection, and active-drop rejection. Its four
C-only executions and four generated-Fairy source-call executions are configured but unobserved.
The internal caller crosses exact self-linked node, Arena, and Region records, obtains zeroed bytes
through `void **`, and verifies reset/drop generations; lexical Region lowering, source-level
`std::alloc` construction, and safe calls remain missing.
The allocation-free text-view slice now checks canonical UTF-8 and code-point byte boundaries,
including overlong, surrogate, out-of-range, truncated, and obsolete forms. Its four native runs
are configured but unobserved. A separate four-way Fairy-source crossing now calls the actual
empty-input UTF-8 validator in the text-view C unit; the public borrowed text result, nonempty
source pointer path, and full wrapper remain missing.
Compiler-owned `c"..."` storage now crosses semantic typing with `Static` origin, MIR as exact
terminated bytes, and generated C as deduplicated immutable static storage. The direct literal can
expose only an immutable `*u8` address through an explicit unsafe operation. Compiler-owned
`b"..."` storage now crosses the same stages as exact non-terminated bytes and a private non-null
`{ data, length }` view; empty literals retain length zero through an immutable sentinel and typed
locals can expose only the immutable data address. Plain UTF-8 `"..."` literals now cross semantic
typing, a distinct MIR operand, and a private `fy_str` view with decoded UTF-8 validation,
deduplicated immutable bytes, an empty non-null sentinel, and typed-local immutable data extraction.
Indexing, slicing, source-level length access, public CStr operations, safe text operations, and stable
slice or string ABIs remain missing.
Owned String and CString now have provisional Buffer-backed C transport with immutable views,
successful-only Buffer moves, failure ownership preservation, self-source append, interior-NUL
rejection, and exact termination. Four C-only native runs are configured but unobserved. A separate
four-way generated-Fairy crossing now calls the real bounded CStr validator with valid, interior-NUL,
and unterminated nonempty storage and requires failure to preserve the prior output view. Another
four-way generated-Fairy crossing calls the real String copy/view/append/drop and CString
from-text/view/drop helpers with exact nested records and by-value result records. Public safe
String/CString resources, Result error resources, and ownership-aware calls remain missing.
The Linux File C slice now also covers partial read/write, write-all progress, sync, and consuming
close including forced close failure. Its four C-only native runs remain configured but unobserved.
A second four-way generated-Fairy crossing uses exact result, File, and open-option records to call
the real open, write-all, sync, close, reopen, partial-read, EOF, and final-close helpers. It checks
exact counts, repeated-close status, normalized closed state, and reconstructed bytes. Public safe
File ownership, source effects, typed errors, and hosted execution evidence remain missing.
Used-symbol product linkage now selects Linux File only for actual admitted calls.

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
at both profiles. Independent compiler or assembly/register inspection, source-level Fairy
crossings, and a hosted Windows receipt remain absent. PE/COFF objects, runtime, standard library,
supported distribution artifacts, and conformance remain later gates. A bounded parser now checks
the product executable and four native probe executables for AMD64 PE32+ console identity, aligned
headers/sections, and an executable entry point, but does not inspect `.obj` symbols, relocations,
unwind payloads, or CodeView/PDB data.

- Implement `x86_64-linux-gnu` `TargetSpec`, toolchain resolution, constrained generated-C transport, and C-driver linking.
- Implement the thin runtime and minimum `SystemHeap`, `Arena`, `Buffer`, text views, `String`, `CString`, and Linux `File` surface required by representative fixtures.
- Run interpreter-to-C differential fixtures and turn the validated C ABI plans into bidirectional
  Clang/GCC call fixtures at low and optimized profiles.
- Keep sanitizer-assisted backend checks mandatory for every admitted C-emission case without
  treating sanitizers as the language oracle or exposing the validation mode as a product profile.

Exit gate: the target reaches `conformant`: interpreter and generated-C runs agree on values, status, trap category, active-payload and drop traces; all declared layout and C ABI fixtures pass in clean Linux CI with no skips. This gate does not by itself make the target `supported`.

## Phase 3: modules, package builds, and bounded caching

- Add public package manifests, lockfiles, module interfaces, concrete package-instance encoding, and public-interface hashing on top of the resolved-graph fixture contract.
- Start with whole-package reuse and a bounded content-addressed cache whose limits are derived from measurements.
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
- `x86_64-windows-msvc`: the TargetSpec, first eight-case local layout probe, 15-case host-independent Microsoft x64 classifier, 12-case same-MSVC native crossing, and bounded PE32+ executable-structure inspection exist, but complete hosted LLP64 layout evidence, independent or inspected call evidence, source-level Fairy crossings, COFF object/symbol/relocation fixtures, UCRT/VCRuntime, and unwind/debug fixtures remain required.
- QBE or another fast backend: requires stable internal semantics, interpreter and C oracles, ABI coverage, and measured edit-build benefit.
- LLVM backend: requires representative workloads that justify optimization and distribution cost after simpler paths are exhausted.
- Generics and `Vec<T>`: require non-generic stability plus compile-time, code-size, active-payload, and diagnostics fixtures.
- Immutable green tree, incremental reparse, formatter, LSP, and SCIP: require an editor product slice and measured parser needs.
- `impact`, semantic `diff`, stable anchors, MCP, and structural edits: require `.fymap` identity and completeness evidence from real use.
- Custom allocators, portable paths, resource-aware arenas, closures, concurrency capabilities, async, sockets, and freestanding or embedded support each require a separate ownership and target contract.

## Self-hosting gate

Self-hosting starts only after modules, strings, containers, `Result`, resource moves, C FFI, package builds, source maps, and the test runner can support the compiler without freezing immature language decisions. The Go compiler remains a bootstrap and reference implementation.
