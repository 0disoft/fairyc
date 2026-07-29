# Fairy C

- Status: Stage 0 implementation prototype
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

Fairy C is an experimental systems programming language for new native software that needs C-like control, predictable costs, explicit C interoperability, and a substantially safer default resource model without a garbage collector or a heavy runtime.

The project is not trying to beat every hand-tuned C program or make every C ownership graph safe. Its intended advantage is that ordinary Fairy C code carries enough ownership, aliasing, bounds, and effect information for the compiler to reject common failures and generate competitive machine code without relying on build-mode-dependent undefined behavior.

## Human quickstart

With a built `fairyc` executable, run these product commands manually from a scratch parent
directory to create one explicit-identity starter:

```text
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
fairyc check
fairyc test --locked
```

The first check validates the generated source and publishes `fairy.lock`. On Windows/amd64 with
local x64 MSVC, `fairyc run --locked` then compiles the `.fy` program through generated C and runs
the temporary native image with the caller's stdin attached after successful compilation. The
generated `README.md` contains the exact next steps and a safe
`std::io::write` example for direct stable execution. `fairyc new` never merges, upgrades, or
overwrites an existing path.

## Current phase

The Go Stage 0 compiler now implements immutable source bytes, a lossless lexer, compact recoverable
syntax, error-free AST lowering, typed HIR, resource and borrow Flow, verified MIR, a bounded
interpreter, deterministic C11 transport, strict local workspaces, stable new/check/build/run/test/C
  emission commands, and experimental Atlas fact export. The semantic runner executes all 42 v0 resolve, type,
and local Flow cases, including integer-literal range rejection, checked compound assignment, branch- and loop-aware safe-borrow reassignment,
exact interprocedural borrow-return summaries, last-use loans, borrow-return origins, canonical
`@effects(...)` sets, and caller-superset enforcement. Whole-root resource Flow
  also executes all 12 v0 cases for moves, cleanup obligations, branch joins, normal, break, and continue loop edges,
consuming tagged switches, and live/dead resource reassignment. The
Region runner executes all 6 v1 cases, including package-proven `std::alloc::Arena`, hidden
allocation origins, lexical escape, capability storage, and plain-region resource rejection. The
  Result runner executes all 24 v0 cases for contextual leading-dot construction, exhaustive shallow switching,
  must-use return/discard/propagation, local and parameter transfer, branch joins, loop back-edges and control exits,
overwrite and early-return rejection, exactly-once classification, non-Result `?`, and propagated
error compatibility. The MIR runner executes all
  12 v0 normalization cases through mutable-local block lowering and the MIR verifier, covering
left-to-right call evaluation, complete-root moves, normal-exit cleanup order, `Result` error
  cleanup, nearest-loop control, explicit checked arithmetic, and non-unwinding trap paths. The verifier mutation runner constructs all 10 named
  baselines and proves all 22 isolated `V1001` through `V1701` rejection recipes, including a
  missing internal call target rejected before artifact or backend handoff. The first MIR
  interpreter slice executes all 93 v0 cases for literal return, direct `i32` minimum constants, scalar fixed-array calls, contextual immutable and mutable byte-view borrowing, exact lengths, checked reads and indexed stores, function-call arithmetic,
boolean branches and logical not, source `&&` and `||` short-circuiting, scalar `while`, classic
scalar `for`, scalar
`Option` construction/switching, scalar `Result` construction/switching/propagation,
resource construction, field reads, call moves, resource-bearing `Option` and `Result` payload moves,
resource returns and propagation, ordered reassignment, consuming-close success/failure,
nested active-variant cleanup, deterministic drop traces,
  normal lexical Region placement plus direct-return, Result-propagation, nested, break, and continue reset traces,
division/remainder zero and signed-overflow, ordinary integer-overflow, matching-type bitwise and
right-shift behavior, exact invalid-shift traps, and bounded step-limit
termination. Fixture-only, manifest-authorized byte sinks additionally prove exact in-memory
stdout/stderr bytes, cross-stream order, and a shared output limit without granting host I/O.
Resource declarations now retain ordered, resolved fields and drop-body classification from AST
through typed HIR and executable MIR. A nonempty body becomes one verifier-linked private
function, executes in the interpreter's `dropping` state, and is called by generated C before field
cleanup. A nonempty resource may declare drop effects, and Resource Flow requires them only on
actual cleanup edges; whole-root moves transfer the obligation without charging the sender.
The interpreter constructs, reads, and recursively drops resources from the verified MIR definition;
host effect implementations remain target-runtime work rather than interpreter emulation.
The constrained C backend emits deterministic C11 for 37 verified fixtures across the scalar
payload surface, restricted inert user enums, flat nonempty move-only resources over Fairy copy
scalars or raw pointers with empty or pure nonempty drop bodies, copy-scalar/inert-struct and resource-bearing
built-in `Option` and `Result` values,
borrowed-view byte lengths, checked
immutable byte-slice indexing and end-exclusive slicing, the target-independent value-preserving integer casts `u8 as i32`,
`u8 as u32`, `u8 as usize`, and `u32 as usize`, checked `u8`, `u32`, and `usize` addition,
subtraction, multiplication, division, remainder, bitwise operations, and checked right shift,
typed equality across `bool` and the four Fairy integer scalars,
typed ordering across those integer scalars, borrowed UTF-8/byte-slice internal calls, and private
value-semantic scalar fixed arrays with exact lengths, contextual immutable and mutable byte-view borrowing,
and checked indexed reads and mutable stores. The cases cover
resource construction, field reads, whole-root call moves, live reassignment, calls and control
flow, active-payload moves, Result propagation, bitwise operations, checked division and shift, overflow
trapping, scalar `Option` Some and None, and scalar `Result` propagation success and failure. This
fixture lane only produces C text; no C compiler, linker, native executable, complete runtime, target
conformance, or self-hosting is implied.

Source `assert(condition);` now requires one `bool` and remains active in low and optimized builds.
False traps as `assertion-failed`; it is not lowered to the build-mode-sensitive C assert macro.
Executable `fairyc.target-spec/v0` descriptions now own host-independent target facts. Linux GNU
records LP64, SysV AMD64, ELF, and DWARF. Windows MSVC records LLP64, four-byte C `long`, 32-byte
shadow space, Microsoft x64, COFF, and Windows unwind/CodeView identities. Neither contains a local
compiler, SDK, sysroot, registry, or filesystem path; both still report `emittable`, `conformant`,
and `supported` as false.
The host-independent layout engine executes all 16 Linux target-layout v0 cases for language and C
scalars, data and function pointers, explicitly based C enums, fixed arrays, nested structs, field
offsets, internal and tail padding, and fail-closed malformed or overflowing inputs. These are
target-mechanics fixtures, not source-language aggregate support or native C ABI proof.
A separate nine-case Windows ledger proves LLP64-sensitive C scalars, pointer-width values,
unavailable POSIX `ssize_t`, and exact mixed-record padding inside the same engine. Its eight pass
cases generate one private C11 assertion unit that the configured Windows smoke compiles and
executes with local x64 MSVC at low and optimized profiles. This is a local probe path, not a hosted
receipt, complete scalar coverage, call/return ABI proof, or target-emission claim.
A host-independent SysV AMD64 classifier now executes 17 versioned C ABI plan cases over the same
type and layout vocabulary, covering INTEGER/SSE/MEMORY classes, mixed and homogeneous returns,
hidden result pointers, register exhaustion, and whole-aggregate rollback. Those plans are not yet
native evidence by themselves, and the target's C ABI marker stays fail-closed.
The same target-bound engine now selects a separate Microsoft x64 algorithm for Windows. Its 15
versioned cases cover shared ordinal RCX/RDX/R8/R9 or XMM0-3 slots, fifth-argument stack transport,
small aggregate integer transport, odd-size and 16-byte aggregate indirection, and hidden-return
slot shifting. The 12 pass plans additionally generate separate caller and callee C11 translation
units that x64 MSVC compiles, links, and executes at low and optimized profiles. ADR 0214 records
the first hosted same-MSVC execution; independent compiler or assembly/register inspection remains
missing. A shared
generated-Fairy caller now also crosses one 8-byte register-sized and one 20-byte hidden-result
`@repr(C)` return from an independently compiled C callee at both MSVC profiles. The identical C11
units repeat those crossings under MSVC's C17 consumer mode without creating a C17 emitter. Broader and
bidirectional source-level Fairy crossings remain missing;
stack/shadow-store observations, callbacks, exports, and hosted evidence remain absent.
The 14 valid plans now generate separate caller and callee objects for Clang-to-GCC and
GCC-to-Clang at `-O0` and `-O2`, for 56 configured crossings. This Windows host cannot execute
them. Separate scalar source fixtures now carry explicitly mapped `@link_name` declarations for
`unsafe extern C fn(i32) -> i32`, individual exact `c::int`, negative `c::long`, `c::size_t`, and
negative `c::ssize_t` flows, plus one family fixture for the remaining twelve TargetSpec C scalars.
Every value travels from producer result through a typed local to a matching consumer argument
across AST, typed HIR, verified MIR, and C emission, with four planned cross-compiler object
crossings per fixture; none of those native executions has been observed on this Windows host.
Separate WSL-free `experimental run-host` and `experimental build-host` commands can now compile a
target-neutral verified `.fy` program through generated C into an x64 MSVC PE. The first executes and
removes a temporary image; the second retains one no-overwrite `.exe` under `build/`. Both reject
FFI, `@repr(C)`, exact `c::*` values, and provisional standard-library calls, so they provide local
source-to-native use without pretending that the host-smoke artifact is a Windows target result.
The distinct `experimental build-target` command now consumes the declared
`x86_64-windows-msvc` profile, emits LLP64-aware C11, and retains one target-labelled PE. Its
checked-in source proves four-byte `c::long` emission and calls the real project-owned SystemHeap,
Buffer, Arena/Region, String/CString/CStr, and UTF-8 C implementations before returning 42. Buffer
selection pulls in SystemHeap exactly once, then construction, length-four resize metadata, and
normalized drop state execute at both MSVC profiles. Arena selection independently closes over the
same SystemHeap component once, then initialization, one active Region, an eight-byte allocation,
reset invalidation, and drop generations execute at both profiles. The Windows x64 heap path uses
the documented 16-byte `malloc` alignment ceiling instead of the MSVC C header's missing
`max_align_t`. The same checked-in target constructs and reads a move-only resource containing an
exact pointer-width raw handle plus `u32` state before returning 42. The Windows target commands now
accept a bounded explicit FY module closure through
repeated `--source`; every foreign companion input, unproven standard-library component, user linker
option, distribution promise, and self-hosting path remains closed.
The sibling `experimental run-target` command uses the same declared target, emission profile,
closed support linkage, MSVC resolution, and PE/unwind validation, but executes a temporary image
instead of retaining it. It reports signed `i32` results outside the runtime's reserved trap-status
range or the corresponding canonical product-trap category and rejects program output. The
reserved-result collision remains an experimental-command limitation; stable workspace run uses a
separate run-only trap receipt and has no such collision. This direct target command remains
experimental and does not promote target maturity.
The package-aware `experimental build-workspace` and `run-workspace` commands now start from the
current directory's strict `fairy.toml` instead of a positional source list. They resolve local
members, exact direct imports, the compiler-bundled SDK, and `fairy.lock`, then select only the root
package's qualified `main::main` for the same closed Windows target pipeline. The checked-in
`examples/workspace_quickstart` imports a public function from one local dependency and returns 42
under both MSVC profiles, a direct locked run, and a locked run through a separately built
`fairyc.exe`. This is real package-aware native compilation without WSL, not Go interpretation.
Those target/profile/output-heavy spellings remain experimental; stable check/build/run reuse the
same root-entry convention without promoting target maturity. Stable test instead overlays the
checked-in `tests/answer.fy` and runs its `@test fn void` through the verified semantic interpreter.
`experimental exec-target` is the program-facing sibling: it forwards bounded stdout and stderr
without a compiler banner and preserves the program's full Windows process status. The checked-in
`examples/windows_output/main.fy` uses a support module containing the one exact unsafe Windows
stdout write-all symbol inside a checked local helper. The raw declaration, safe text wrapper, and
`main` propagate `@effects(console)` while the byte-view helper remains pure. The wrapper returns
`Result<usize, ConsoleError>`, preserving either the exact byte count or the runtime status plus
completed prefix. Its `main` passes `ref str` and `ref u8[]` through safe internal calls; the
byte helper observes the exact NUL and `A` bytes through checked read-only indexing while the entry contains no raw pointer or `unsafe`,
prints `Fairy C says hi`, and exits 42. That is real
FY-to-MSVC process output, unsafe containment, and source-effect enforcement for the legacy example.
The compiler-bundled SDK `0.0.0-stage0.6` exposes safe text
`std::io::write(ref str)` and `std::io::write_stderr(ref str)` operations plus exact binary
`write_bytes(ref u8[])` and `write_stderr_bytes(ref u8[])` operations. All four return
`Result<usize, ConsoleError>`. The same module now exposes allocation-free
`write_i32(i32) -> Result<usize, ConsoleError>` for canonical locale-independent decimal stdout and
`read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>`; one call performs at most one bounded
read, and success with zero bytes means EOF. `examples/std_io_quickstart`
contains no raw pointer, unsafe block, foreign declaration, or platform name; stable run forwards
its exact text and byte-view output on separated stdout and stderr and preserves success status zero, while
stable build produces the same inspected Windows PE for direct execution. Direct execution proves
EOF, a short input echo, and an eight-byte bounded prefix. Stable `fairyc run` forwards the short
input after successful compilation, and a separately built `fairyc.exe` repeats that path plus the
locked build and artifact execution.
Linux owns a POSIX adapter and four native probes covering all three process streams. The first
hosted Linux receipt in ADR 0214 executed that configured lane successfully. Byte input and output perform no UTF-8
validation. General formatting, implicit newlines, text or line input, terminal encoding, live output
interleaving, and a final symbolic error taxonomy remain outside the API.
A provisional SystemHeap implementation now lives in the target standard-library
boundary rather than the thin runtime. It normalizes empty allocation, admits current fundamental
alignments, preserves bytes across copy-and-release growth or shrink, and leaves the old block
unchanged when validation or allocation fails. A closed C-only probe and an internal generated FY
caller using the exact `void *`/`size_t` block each have four Clang/GCC low/optimized executions in
the hosted receipt recorded by ADR 0214. `experimental build` now selects this helper
when an actual generated call requires it, but no public `std::alloc` API exists. Arena, text
ownership, and Linux File now have their own C slices below; each now also has a raw generated-Fairy
source crossing.
A second provisional slice implements Buffer-owned initialized bytes, failure-atomic reserve,
relocating self-overlap append, length-overflow rollback, zeroed growth, truncate, clear, and drop.
Its closed C-only probe and an internal generated FY caller using the exact nested SystemHeap-block
plus `size_t` record each have four Clang/GCC low/optimized executions in that hosted receipt.
Three exact Buffer helpers are now admitted by the constrained source
backend and exercised by the closed native fixture. Used-symbol product linkage selects the Buffer
plus SystemHeap dependency closure; the public standard-library package remains absent. An
independent raw caller now links the target-specific Buffer closure as a separate MSVC translation
unit at both profiles. Its probe-only allocator forces initialization, reserve, relocating
self-overlap append, and zeroed-resize allocation failures, requires exact pointer/metadata/byte
preservation, proves normal recovery and zeroed growth, and returns live allocations to zero after
drop. Product linkage retains the ordinary allocator.
A third slice implements Arena-owned byte nodes with one active generation-checked Region,
failure-atomic zeroed or copied placement, reset invalidation, stale-use rejection, and active-drop
rejection. Its closed C-only probe and an internal generated FY caller using the exact self-linked
node, Arena, and Region records each have four configured native executions. The generated caller
crosses real initialization, begin, zeroed allocation through a `void **` output slot, reset, and
drop. Used-symbol product linkage selects Arena plus SystemHeap, but lexical Region lowering and a
public `std::alloc` API remain absent. Windows target resolution now admits the six exact Arena
symbols, including copied placement, while the checked-in `build-target` example uses five lifecycle
symbols used by that example and observes non-null allocation plus node, active-state, and generation
transitions at both MSVC profiles. Its private generation transport is target-width `usize`, which
matches the selected Windows x64 C layout without becoming a public representation promise. This
Windows path does not dereference the returned storage or independently inspect its zeroed bytes;
an independent raw caller now supplies that byte-level evidence at both MSVC profiles. It also
forces payload-allocation and tracking-node-allocation failure separately, requires unchanged Arena
and caller output, proves payload rollback with zero leaked live allocations, and rechecks stale
Region plus active-drop rejection. The shared probe adapter remains absent from product linkage.
The first text slice now validates canonical UTF-8 without allocation and rejects slicing through a
multibyte sequence. Its native probe includes overlong forms, surrogates, values above U+10FFFF,
truncation, and obsolete leaders and remains unobserved; raw source transports follow below.
Its empty-input validator now has exact internal `fy_std_*_v0` symbols and two real FY-to-stdlib-C
four-way crossings: direct validation and `@repr(C)` output-record initialization through a verified
raw address. The self-contained empty validator call is also the first configured product-build
smoke that requires automatic standard-library linkage. Compiler-owned `c"..."` storage now reaches
the C backend as immutable static bytes, and an explicit unsafe direct-literal conversion can expose
its read-only `ptr u8` address to raw FFI. Compiler-owned `b"..."` storage now independently reaches a
private non-null `{ data, length }` byte-slice transport, including an empty length-zero sentinel;
typed locals may expose only the immutable data address through explicit unsafe conversion. Plain
`"..."` literals now independently reach a private non-null `fy_str` view after complete decoded
UTF-8 validation, with deduplicated immutable storage and the same empty length-zero sentinel rule;
typed locals can expose only immutable data through explicit unsafe conversion. Safe borrowed
`ref str` and `ref u8[]` views expose `.len` as their `usize` byte count and `.is_empty` as its exact
zero-length `bool` without scanning or allocation.
Immutable `ref u8[]` additionally supports read-only `usize` indexing to `u8` and end-exclusive
`bytes[start..end]`, `bytes[..end]`, `bytes[start..]`, and `bytes[..]` subviews behind verified
`bounds` traps. Omitted bounds normalize to zero or the retained receiver length; written operands
evaluate once left to right and preserve the receiver's borrow origin. Mutable `ref mut u8[]` also
supports checked `u8` reads and indexed stores; the place is checked before the right-hand side is
evaluated. Mutable integer locals and admitted indexed scalar places support statement-only exact-type
`+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and `|=` with the existing traps and no
promotions. Text indexing or slicing, a stable public string ABI, and the general safe text API still
do not exist.
Fixed arrays of `bool`, `i32`, `u8`, `u32`, or `usize` with `1 <= N <= 4096` now construct from
nonempty list literals, cross internal calls and returns as inert copy values, expose exact
zero-cost `usize` `.len`, and support checked read-only `usize` indexing. A direct shared borrow of
`u8[N]` contextually becomes `ref u8[]` where that type is expected, so local arrays can use the
existing immutable byte-view calls and operations without unsafe code. A direct exclusive borrow
of a mutable `u8[N]` likewise becomes `ref mut u8[]` only where that internal parameter type is
expected. It carries exact length and may expose `ptr mut u8` only inside unsafe wrapper code; safe
checked reads and indexed `u8` stores are available, while mutable slicing remains absent. Mutable
fixed scalar-array locals and `mut` parameters likewise support checked element assignment and
checked integer compound assignment. The ceiling is a provisional Stage 0
implementation guard, not a stable language or ABI limit. A mutable safe-borrow local may
be reassigned through straight-line, branch, and `while` control flow; Flow replaces its provenance
after evaluating the right-hand side, joins reachable branch/header/`continue`/`break` origins
without guessing, and releases old owners only after their last reachable loan use. Exact whole-
array `ref T[N]` and `ref mut T[N]` types may now cross internal calls without becoming copyable
aliases. Empty and repeated literals, fields, nesting, general mutable projections and slicing,
and public or foreign ABI use remain closed.
Internal borrow-returning calls now infer one exact formal borrow parameter, region parameter, or
static return source before ordinary Flow checks. The selected actual argument alone carries its
loan past the call, including across argument reordering and acyclic wrapper chains; recursive
results without a complete proof remain fail-closed.
Buffer-backed String and CString C helpers now preserve ownership on failed UTF-8 conversion,
support overlap-safe String append, expose immutable bounded views, reject interior NUL, and append
exactly one C terminator. Their C-only native probe remains configured for the four Linux
compiler/profile executions and now also runs as two separate-translation-unit x64 MSVC
executions. A separate
generated FY caller now crosses the exact CStr view record into the real bounded validator for valid,
interior-NUL, and unterminated storage while checking failure leaves the prior view unchanged. A
second generated FY caller uses exact nested heap/buffer/String/CString/result records to call the
real String copy, view, append, and drop helpers plus CString creation, CStr view, and drop. Its C
observers require exact bytes, repeated append, one terminator, and normalized release. These are raw
internal transports. A separate checked-in example now wraps opaque storage bridges in move-only
Fairy `String` and `CString` resources, maps operation failures into source `Result`, exposes scalar
metadata plus owner-bound `ref str` and `ref CStr` views, and runs its C release through verified resource
drop glue. Exact unsafe raw-parts built-ins remain private to the support module, and Flow rejects
moving an owner before the derived view's last use. This is an example-level safe boundary, not a
stable public `std::text` module, layout, or foreign borrowed-view ABI.
Windows `build-target` now admits the exact owned-text, CStr, and opaque storage-bridge entry points. A compiler-owned
five-byte UTF-8 literal is copied into String, viewed at length five, appended to length ten, and
dropped to normalized storage; the same input becomes a six-byte terminated CString whose view has
logical length five before normalized drop. A direct compiler-owned `c"Fairy"` storage block also
passes bounded CStr validation. The FY path observes statuses, lengths, non-null pointers, alignment,
and cleanup but cannot dereference returned storage. The independent raw C probe now supplies
Windows evidence for exact copied bytes, the terminator, null and malformed storage rejection,
failure preservation, self-overlap, size overflow, and repeated drop at both MSVC profiles. A
probe-only one-shot allocator also forces String and CString initial allocation failure plus String
append reallocation failure, verifies exact storage/byte preservation and live-allocation cleanup,
and remains absent from product linkage.
The separate `examples/owned_text/` program exercises only the safe wrapper surface. Low and
optimized x64 MSVC builds prove String copy and append, CString creation for nonempty and empty
text, interior-NUL rejection, safe internal borrowed-view calls, scalar lengths, and implicit
cleanup while keeping raw pointers and foreign calls out of its entry module.
The Linux File C slice now exposes partial I/O, write-all progress, sync, and consuming close that
is never retried, even after failure. ADR 0214 records its C-only four-way probe's first hosted run.
A separate generated FY caller now crosses exact result, File, and open-option records into the real
open, write-all, sync, consuming-close, reopen, partial-read, EOF, and final-close helpers. It also
requires a repeated close to report the closed status and both completed closes to normalize the
descriptor record. This is raw internal transport; safe File ownership, source effects, typed
errors, and hosted execution evidence remain absent. Product builds now select the Linux File C
unit only when one of the five admitted calls is actually present.
Windows now has an independent File C slice over inline `HANDLE` state with no hidden allocation.
It accepts a bounded ASCII bootstrap path, preserves partial read and write-all progress, syncs
explicitly, consumes state before one `CloseHandle` attempt on both success and failure, and makes
drop a silent one-attempt fallback. A separate-translation-unit probe creates, writes, syncs,
closes, reopens, reads in three-byte chunks, observes EOF, forces `ERROR_INVALID_HANDLE`, and checks
repeated close and drop at low and optimized local MSVC profiles. The six raw calls are target-bound
through the closed resolver. A checked-in two-module FY example now wraps that exact component in a
move-only `File`, preserves status, Win32 code, and completed bytes in `FileError`, carries
`filesystem` effects, writes and synchronizes `FairyFile`, reads it through `read(ref mut u8[])`
backed by a mutable fixed array, observes EOF, consumes explicit close, and proves implicit drop by
successfully reopening a path whose live write handle would deny sharing. Low and optimized MSVC builds both
return 42 after object, PE, and unwind inspection. This is usable Stage 0 source evidence, not yet a
stable `std::fs`, portable Unicode path, general mutable-buffer, hosted, or public-layout contract.
The source FFI slice accepts `i32`, `u8`, `u32`, `usize`, all sixteen exact transport-only TargetSpec
C scalars, admitted raw-pointer parameters or results, and nonempty acyclic `@repr(C)` struct
results. A declaration may keep its Fairy name or use one nonempty `@link_name("external_symbol")`;
AST, typed HIR, MIR, prototypes, and calls preserve the source/link identity split.
Calls require `unsafe {}` or `unsafe fn`; missing unsafe provenance is `E5101` and safe references in
raw declarations are `E5102`. The interpreter fails closed with a `foreign-call` capability error,
while C emission produces a plain external prototype with no inferred alias, purity, retention, or
unwind attributes. Typed `null` crosses a closed pointer/target-scalar native probe. An exact or
opaque safe-borrow-to-raw-pointer cast may form the address of a direct local place inside `unsafe`.
An existing resource or fixed-array safe borrow may erase only to `void` and forwards its referent
pointer unchanged; neither path permits mutability strengthening.
Those C scalar types remain distinct from coincidentally represented Fairy scalars, emit their exact
C spellings with target and provider assertions, and currently have no source literal, operator,
cast, raw-pointer, or interpreter support. Nonempty `@repr(C)` records may use exact C-scalar fields,
opaque `void` pointers, and acyclic nested `@repr(C)` records; generated definitions are ordered by
dependency and guarded by target-derived nested layout assertions. One pointer-to-opaque-pointer
output-slot shape is admitted for Arena allocation. A generated FY caller now receives both an
8-byte register result and a 20-byte hidden-memory result from hand-written C in the four-way native
matrix. Raw dereference or arithmetic, arbitrary casts, nested enum or built-in-tag fields, array
fields or foreign array signatures,
callbacks, by-value aggregate parameters, exports, and public standard-library wrappers remain
unimplemented.
The C emitter records only foreign link names used by actual calls. `experimental build` uses one
exact-symbol contract to link the minimal provisional standard-library dependency closure; unused
declarations pull in nothing, while arbitrary foreign libraries and linker arguments remain closed.
This enables raw internal calls, not yet a public `std::*` programming surface.
A deterministic native C11 plus explicit POSIX-typedef probe now translates the eleven valid layout cases into private numeric
types for Clang/GCC at `-O0` and `-O2`. This Windows host cannot execute those four probe runs, but
ADR 0214 records their first hosted success, so `c-layout-probes` is no longer missing for Linux.
A separate Windows C11 probe translates the eight valid LLP64 cases into private numeric types and
compiles and executes them twice with local x64 MSVC through the configured host smoke. The same
smoke separately crosses all 12 validated Microsoft x64 call cases between caller and callee
translation units at both profiles. A third three-object probe raises and catches a private Windows
exception, virtual-walks raiser, termination, and catcher frames, and requires one abnormal
`__finally` execution under both profiles. The target marker remains missing until broader layout
and independent/source-level/hosted ABI and unwind evidence exists. A host-independent PE parser
now validates the host-smoke product, both declared-target profile products, both safe File and
safe owned-text profile products, nine package-workspace build products including four measured
experimental, three stable quickstart builds, and two safe `std::io` builds, nine package-workspace run
products, two direct run-target temporary products, two exec-target output products, and all
eighteen raw or generated-Fairy Windows File, Buffer, owned-text, Arena, layout,
ABI, struct-result, and unwind probe images as bounded
AMD64 PE32+ console executables
with aligned headers and sections, an executable `.text` entry point, and bounded Microsoft x64
unwind records. That linked-image path does not inspect object symbols or relocations or establish
hosted API compatibility, signing, installation, or distribution. The same package separately parses every
intermediate MSVC `.obj` before its temporary build root is removed. It requires raw AMD64 COFF,
bounded section and relocation ranges, readable executable `.text` or `.text$...` code, valid
relocation symbol indexes, and a nonempty symbol table while admitting raw-storage-free `.bss`.
This is structural live-object evidence, not a versioned symbol/relocation fixture or unwind proof.
The linked-image inspector now also follows the exception data directory into `.pdata` and its
referenced unwind records. It validates sorted nonoverlapping executable `RUNTIME_FUNCTION` ranges,
DWORD-aligned file-backed unwind RVAs, version/flag/frame/prolog structure, bounded v1 unwind-code
slots, version-2 epilog descriptor bounds, and minimum handler or chained tails. The local MSVC
product currently exposes both version 1 and version 2 records. The native unwind probe separately
proves its closed exception payload, captured-context virtual walk, and handler-directed termination
path. Version-2 epilog offset semantics, language-specific handler data, recursive chains,
Fairy/foreign-frame crossings, object-level unwind relocations, and a complete versioned unwind
fixture contract remain open despite the observed generic Windows hosted receipt.
A separate fail-closed Linux/amd64 command compiles those cases with Clang and GCC
at `-O0` and `-O2`, link a closed conformance runtime, and compare process outcomes plus exact
source-linked resource-drop sequences. Product emission and product builds contain no trace observer.
This Windows host still has no Linux Clang/GCC execution environment, but GitHub Actions run
`30442310794` passed the complete hosted Linux lane for commit `0bb4199` as recorded by ADR 0214.
The remaining runtime, standard-library, ABI, baseline, and provisioning gaps keep the target unsupported.
A stable source-independent `fairyc explain <diagnostic-code>` lookup reports reserved Stage 0
meanings and categories without reading source or invoking a toolchain while still marking the
catalog provisional. A stable `fairyc targets` human report lists declared target maturity and
missing evidence without probing this Windows host. An unstable
`fairyc experimental` CLI can inspect syntax, syntax-check, semantic-check, emit the
scalar, flat-resource, and resource-tagged C11 translation unit, build that subset into a Linux executable on an eligible host,
inspect the declared target specification, or interpret an explicit source-root-relative FY module
closure. Its bounded foreground `check-workspace-session` command can keep one process-local
verified-MIR cache across sequential caller-triggered checks; it is not a daemon, watcher,
persistent cache, or stable editor protocol. Program commands use one entry plus repeated
`--source`; semantic checking uses its explicit positional module set. Stable project bootstrap is
`fairyc new <package-name> --uid <package-uid>`. The stable workspace commands are
`fairyc check [--locked]` and the Windows/amd64-local
`fairyc build [--release] [--locked]`, and the Windows/amd64-local
`fairyc run [--release] [--locked]`, the target-independent semantic
`fairyc test [--locked]`, plus source-independent
`fairyc explain <diagnostic-code>` and `fairyc targets`, and target-explicit
`fairyc emit c --target <target-id> [--locked]`. Stable run forwards caller stdin, bounded Windows
program output, and status without pre-reading input; native test execution, program arguments,
input forwarding for other native commands,
a complete runtime, safe standard library, package manager, supported
target, release-stable CLI as a whole, and a stable language revision do not exist yet.

## Try the current prototype

The following commands are manual developer examples, not agent command authority:

```text
go run ./cmd/fairyc explain E1101
go run ./cmd/fairyc targets
go run ./cmd/fairyc experimental run-host examples/run/modules/main.fy --source examples/run/modules/math.fy --profile optimized
go run ./cmd/fairyc experimental build-host examples/run/modules/main.fy --source examples/run/modules/math.fy --profile optimized --output build/fairyc-windows-smoke.exe
go run ./cmd/fairyc experimental build-target examples/windows_target/main.fy --target x86_64-windows-msvc --profile optimized --output build/fairyc-windows-target.exe
go run ./cmd/fairyc experimental run-target examples/windows_target/main.fy --target x86_64-windows-msvc --profile optimized
go run ./cmd/fairyc experimental exec-target examples/windows_output/main.fy --source examples/windows_output/windows_console.fy --target x86_64-windows-msvc --profile optimized
go run ./cmd/fairyc experimental check-syntax examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/semantic/main.fy examples/semantic/math.fy
go run ./cmd/fairyc experimental check-semantics examples/resources/main.fy
go run ./cmd/fairyc experimental check-semantics examples/regions/main.fy
go run ./cmd/fairyc experimental check-semantics examples/results/main.fy
go run ./cmd/fairyc experimental check examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental check-workspace
go run ./cmd/fairyc experimental check-workspace --locked
Push-Location examples/workspace_quickstart
go run ../../cmd/fairyc check
go run ../../cmd/fairyc check --locked
go run ../../cmd/fairyc build
go run ../../cmd/fairyc build --release --locked
& ./build/workspace-quickstart.exe
go run ../../cmd/fairyc run --release --locked
go run ../../cmd/fairyc test --locked
go run ../../cmd/fairyc emit c --target x86_64-windows-msvc --locked
go run ../../cmd/fairyc emit c --target x86_64-linux-gnu --locked
go run ../../cmd/fairyc experimental check-workspace
go run ../../cmd/fairyc experimental run-workspace --locked --target x86_64-windows-msvc --profile optimized
Pop-Location
go run ./cmd/fairyc experimental target-spec x86_64-linux-gnu
go run ./cmd/fairyc experimental target-spec x86_64-windows-msvc
go run ./cmd/fairyc experimental emit-c examples/c/main.fy
go run ./cmd/fairyc experimental emit-c examples/c/tagged.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource_tagged.fy
go run ./cmd/fairyc experimental emit-c examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental build examples/c/tagged.fy --cc clang --profile optimized --output build/fairyc-example
go run ./cmd/fairyc experimental build-target examples/owned_text/main.fy --source examples/owned_text/owned_text.fy --target x86_64-windows-msvc --profile optimized --output build/owned-text.exe
go run ./cmd/fairyc experimental run examples/run/main.fy
go run ./cmd/fairyc experimental run examples/run/resource_option.fy
go run ./cmd/fairyc experimental run examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental parse examples/syntax/main.fy
```

To exercise the bounded process-local workspace cache from PowerShell without WSL:

```powershell
Push-Location examples/workspace_quickstart
@(
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":1}'
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":2}'
) | go run ../../cmd/fairyc experimental check-workspace-session --locked --target x86_64-windows-msvc --max-requests 2 --cache-max-entries 8 --cache-max-bytes 33554432 --cache-max-artifact-bytes 8388608
Pop-Location
```

The first response must report `fallback` and the second `all-hit`. These numeric limits are only
explicit example inputs; they are not product defaults or tuning recommendations.

For repeated Windows use, `DEVELOPMENT.md` gives the WSL-free manual bootstrap that creates ignored
`build/fairyc.exe`. The configured Windows smoke separately builds a temporary standalone compiler
process, requires it to create a no-overwrite starter, check it, run its semantic test, and compile
its generated source through MSVC with conventional program status zero. The same executable runs
`examples/windows_quickstart/main.fy` through MSVC with
native result 42 by iterating two bytes with checked `usize` arithmetic and safely converting each
indexed `u8` value to `i32`,
then execute `examples/windows_output/main.fy` with exact stdout and process status
42 while explicitly supplying its separate `windows_console.fy` module, and finally run the
two-package `examples/workspace_quickstart` graph through an exact generated lock with native result
42, and runs the safe `std::io` workspace with exact text and byte-view lines on separated
stdout/stderr and status zero. The same standalone
compiler runs the isolated semantic test and emits locked Windows and Linux C11 application units without
changing the retained executable. It then measures nine fresh-cache cold/warm pairs with one new compiler process per sample,
sends eight editor-shaped workspace checks to one compiler process, validates the cold, no-op,
private-root, private-dependency, and public-dependency cache paths, and records Windows
process-memory peaks. Raw paired timings are local observations with zero retries, not a speedup or
default-cache claim. The compiler binary is
still built from Go; only the input programs and resulting temporary
target executables are Fairy C.

The stable `explain` and `targets` commands do not read the current directory or source tree. The
former marks its diagnostic catalog as provisional; the latter reports declared TargetSpec evidence
without probing local MSVC, SDK, WSL, or any other toolchain. The experimental source commands treat the current directory as the temporary source
root. `parse` emits versioned,
deterministic JSON containing token and CST kinds and byte spans, but not source text, token text,
literal values, or absolute paths. `check-semantics` accepts an explicit bounded list of source files
for the currently implemented scalar, safe-borrow, whole-resource, Region-parameter, and Result subset. It rejects duplicate paths, root
escapes, oversized input sets, and unsupported syntax rather than claiming success. The official
`experimental check` continues that explicit closure through MIR lowering and verification without
requiring `main`, running code, generating C, probing MSVC or WSL, or writing an artifact. It can
therefore validate an admitted FFI wrapper that the interpreter cannot execute. The official
`experimental run` accepts one entry plus a bounded explicit repeated-`--source` module closure whose
entry module contains `main() -> i32`, then executes its
verified MIR in the bounded interpreter, including boolean `if`, short-circuit control, scalar
`while`, classic scalar `for`, checked compound assignment,
resource construction, field reads, whole-root call moves, resource-bearing built-in tagged values,
ordered resource reassignment, consuming-close ownership patterns, cleanup drop events, Region
fallthrough, direct-return, Result-propagation, nested, break, and continue reset paths, checked
addition, subtraction, multiplication, negation, matching-type bitwise operations, and checked
signed or unsigned right shift. It performs no source
discovery and needs no C toolchain, MSVC, WSL, or host subprocess, and reports either `result: <i32>` or a canonical
trap. The official
stable `fairyc check [--locked]` command now checks the current `fairy.toml` workspace through
verified MIR and maintains or verifies `fairy.lock`. `experimental check` remains the explicit-file
developer surface; stable machine-readable target JSON and broader configuration are still separate work.

The stable `fairyc emit c --target <target-id> [--locked]` command consumes that same manifest-owned
workspace and root `main::main fn() -> i32`, then writes only the selected Linux GNU or Windows MSVC
ISO C11 application translation unit to stdout. Target selection is mandatory because LP64 and
LLP64 are different contracts; the command never guesses from this host. It invokes no C compiler,
linker, SDK, WSL, subprocess, or network and creates no build artifact. The separate runtime and
standard-library support unit is intentionally absent, so redirected output is inspection/transport
input rather than a promised standalone C package. Ordinary mode maintains `fairy.lock`;
`--locked` verifies it without writing.

The stable `fairyc test [--locked]` command keeps tests out of ordinary compilation. It overlays
only the root package's optional lowercase `tests/` directory after the production lock is handled;
`tests/answer.fy` therefore declares `module tests::answer;`. Mark a private safe zero-argument
`void` function with argument-free `@test`. Each discovered test runs in canonical name order through
a fresh verified-MIR interpreter state, so it needs no C compiler, MSVC, WSL, subprocess, or build
artifact. A trap fails that test and later tests still run. This is the fast semantic lane, not a
claim that native ABI, foreign calls, host I/O, benchmarks, filters, or workspace-member test
discovery exist.

Phase 4 now has its first executable boundary: `internal/semanticsnapshot` turns successful typed
HIR, semantic facts, verified MIR, and exact package metadata into detached, path-free compiler
facts. It is invariant under build-local package renumbering, keeps external C calls explicitly
opaque without degrading unrelated local facts, and publishes no source text or authority-bearing
documentation. `internal/symbolidentity` now turns those declarations into provisional
source-instance-independent SymbolIds with canonical signature hashing and mandatory visible/full
collision checks. Exact per-file content and LF-derived line-map identities now travel from the
validated workspace into the immutable snapshot without source bodies or host paths, closing the
stale-span metadata prerequisite. `internal/atlasmap` now validates the complete workspace input and
builds one immutable, deterministic logical partition per exact package. Each header carries the
exact compiler build, language, package, source, interface, file, line-map, and SymbolId algorithms,
and incomplete relations become explicit stable omissions. The same package now encodes each
partition as caller-bounded canonical `fairyc.fymap-json/v1`, binds it with the domain-separated
`fymap1-sha256` checksum, rejects unknown or noncanonical JSON, revalidates every identity and
record relationship, and rederives every SymbolId from the stored declaration and exact package
namespace references. The opt-in `MapStore` now publishes checksum-named immutable generations under
an opened root with mandatory retention bounds, recovers abandoned temporaries, rejects corrupt or
conflicting generations, and requires exact current workspace package/file/line-map plus compiler
and interface identities before returning spans. The internal query index repeats that freshness
check, accepts only exact SymbolIds, and implements deterministic caller-bounded `overview`, `show`,
and package-local `around` queries with explicit filter, depth, byte, dependency-join, and incoming-
map omissions. `internal/contextpack` now runs those queries directly and emits caller-bounded,
checksummed canonical `fairyc.context-pack-json/v1` with exact package/compatibility/query/budget/
usage/completeness/fact/relation/omission fields and visibly empty text sections. The first
`experimental atlas-pack overview` command exposes one explicitly selected package UID with explicit
record/selection/output limits and no map-store default. Exact SymbolIds obtained from the separate
catalog command can now drive `experimental atlas-pack show` and `around` with explicit direction,
relation, record, relation, depth, selection-byte, and output-byte policy. Those commands add no
default store policy, dependency-map join, source-excerpt output, pack decoder, or text importer. The internal
query index now also provides exact full-SymbolId-ordered catalog pages with exact module/name/kind
filters, explicit symbol/byte limits, and full-ID-only continuation. Exact-name matches retain every
same-name declaration across modules rather than guessing one winner. Symbol Catalog v1 exposes
those pages through `experimental atlas-symbols list` as checksummed
`fairyc.symbol-catalog-json/v1`, with every filter and output limit explicit. The Windows
host smoke now also records nine raw whole-package canonical rewrite samples after planning, checks
byte identity, and reports fixed source/record/encoded-byte denominators without choosing chunking,
compression, incremental updates, or a performance budget.

ADR 0195 closes the attributed-text policy, ADR 0196 implements its exact retained-source gate, and
ADR 0197 adds canonical bounded `fairyc.attributed-text-json/v0` output with explicit selection,
exact subject identity, independent document integrity, and no caller-result forgery. ADR 0198
exposes it as `experimental atlas-text source` with exact package/SymbolId selection, explicit
workspace origin and count/text/output budgets, and successful checksummed size omissions. `.fymap`
v1 stays body-free and Context Pack v1 keeps its text arrays empty. ADR 0199 separately classifies
physical-line-leading `///` trivia and carries the exact attached span for supported top-level
declarations through HIR. ADR 0200 preserves that fact in an in-memory-only snapshot sidecar and
exposes exact marker-bearing bytes as `experimental atlas-text documentation`, including explicit
`kind-unavailable`, safety, and size omissions. Existing source output and `.fymap` v1 bytes remain
unchanged. No decoder, text store, stable attributed-text CLI, rendered documentation model,
dependency text, file reopening, network fetch, model call, or agent authority exists.

ADR 0202 adds `experimental atlas-evidence show` for one exact SymbolId. It prepares the workspace
once and nests an unchanged compiler-fact Context Pack v1 plus unchanged source and documentation
attributed-text v0 documents in canonical checksummed `fairyc.atlas-evidence-json/v0`. The wrapper
has authority `none`, text remains `untrusted-text`, component and final limits are mandatory, and no
decoder, default, store, token estimate, model call, or Context Pack text population is introduced.

`experimental target-spec` accepts exactly one known Linux GNU or Windows MSVC ID and writes
deterministic `fairyc.target-spec/v0` JSON. The result separates target semantics from local
toolchain discovery and lists the profile-specific missing evidence that keeps all three target
maturity claims false.

`experimental emit-c` accepts the same explicit entry-plus-`--source` closure and prints the verified
scalar plus copy-scalar/inert-struct tags, flat all-`i32` resources, and resource-bearing built-in
tags as a C11 translation unit directly to stdout.
Every used C scalar receives target size/alignment assertions; signed and unsigned integer families
also receive signedness assertions. Target `c::ssize_t` alone adds the declared POSIX.1-2008 feature
macro and provider header. The command does not write a `.c` file or invoke a native toolchain.

On Linux/amd64 with glibc and the explicitly selected Clang or GCC installed, `experimental build`
uses that same verified unit and the embedded runtime to produce a no-overwrite executable
below the ignored `build/` directory. Compiler, `low` or `optimized` profile, and output path are
mandatory. This Windows host cannot execute that command, and the command remains narrower than the
interpreter: scalar tags, direct flat nonempty resources with empty or pure nonempty drop bodies, and
`Option<R>` or `Result<S, E>` where each payload is exactly `i32` or such a direct resource work.
Nested tags, partial initialization, user drop effects, general unsafe operations, arbitrary
aggregate parameters, callbacks, and public standard-library programs still fail closed before
native compilation. `unsafe extern C fn` calls, optionally mapped through one validated
`@link_name`, admit only the closed scalar, raw-pointer, and selected `@repr(C)` result transports;
`experimental build` succeeds only when the selected C driver can resolve the
external symbol from its ordinary link environment because the product CLI has no companion-source
or arbitrary link-input option.

## Design anchors

- Product contract: `docs/product/02-spec.md`
- Compiler and repository boundary: `docs/architecture/00-system-boundary.md`
- Language concepts: `docs/architecture/01-domain-model.md`
- Compilation flow: `docs/architecture/02-runtime-flow.md`
- Quality targets: `docs/architecture/03-quality-attributes.md`
- Initial compiler decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Contract ownership decision: `docs/adr/0002-contract-source-of-truth.md`
- Stage 0 safety boundary: `docs/adr/0003-stage-0-safety-boundary.md`
- Language semantics: `docs/language/`
- Normative Stage 0 grammar: `spec/grammar/stage0.ebnf`
- Parser acceptance fixtures: `spec/fixtures/parser/manifest-v0.json`
- Semantic acceptance fixtures: `spec/fixtures/semantic/manifest-v0.json`
- Resource Flow fixtures: `spec/fixtures/resource/manifest-v0.json`
- Region fixtures: `spec/fixtures/region/manifest-v1.json`
- Result fixtures: `spec/fixtures/result/manifest-v0.json`
- MIR normalization fixtures: `spec/fixtures/mir/manifest-v0.json`
- MIR verifier mutation fixtures: `spec/fixtures/mir-verifier/manifest-v0.json`
- MIR interpreter fixtures: `spec/fixtures/interpreter/manifest-v0.json`
- Constrained C emission fixtures: `spec/fixtures/c-emission/manifest-v0.json`
- Shared constrained-C driver and embedded scalar runtime: `internal/cdriver/`, `internal/cruntime/`
- Provisional target standard-library C sources: `internal/cstdlib/`
- Linux scalar C differential runner: `internal/cbackendnative/`
- Executable target description: `internal/target/`
- Experimental target JSON schema: `spec/cli/experimental-target-spec-v0.schema.json`
- Experimental bounded workspace-session schema: `spec/cli/experimental-workspace-check-session-v0.schema.json`
- Experimental persistent workspace-cache schema: `spec/cli/experimental-workspace-cache-check-v0.schema.json`
- Target layout engine and fixtures: `internal/layout/`, `spec/fixtures/layout/manifest-v0.json`, `spec/fixtures/layout/manifest-windows-msvc-v0.json`
- Shared Windows MSVC C probe type emitter: `internal/msvccprobe/`
- Shared generated-Fairy/C aggregate-result probe: `internal/foreignstructprobe/`
- Windows MSVC native layout probe generator: `internal/windowslayoutprobe/`
- Target C ABI classifiers and fixtures: `internal/cabi/`, `spec/fixtures/c-abi/manifest-v0.json`, `spec/fixtures/c-abi/manifest-windows-msvc-v0.json`
- Windows MSVC native ABI probe generator: `internal/windowsabiprobe/`
- Windows x64 native exception and virtual-unwind probe generator: `internal/windowsunwindprobe/`
- Windows AMD64 PE, COFF object, and x64 unwind structure inspector: `internal/pecoff/`
- Resolved acyclic package graph engine, fixture runner, and v1/v0 fixtures: `internal/packagegraph/`, `internal/packagefixture/`, `spec/fixtures/package/manifest-v1.json`
- PackageInstanceId v1 encoding, vectors, and runner: `internal/packagegraph/identity.go`, `spec/fixtures/package-identity/manifest-v1.json`, `internal/packageidentityfixture/`
- Public-interface hash v2 extraction, reusable package-set computation, fixtures, and runner: `internal/moduleinterface/`, `spec/fixtures/module-interface/manifest-v2.json`, `internal/moduleinterfacefixture/`
- Package-source digest v1 encoding, fixtures, and runner: `internal/packagecontent/`, `spec/fixtures/package-source/manifest-v1.json`, `internal/packagecontentfixture/`
- Whole-package cache-key v1, bounded memory store, fixtures, and runner: `internal/packagecache/`, `spec/fixtures/package-cache/manifest-v1.json`, `internal/packagecachefixture/`
- Exact-version verified whole-program MIR artifact codec: `internal/mircodec/`
- Strict local-exact `fairy.toml` v1 parser, fixtures, and runner: `internal/projectmanifest/`, `spec/fixtures/project-manifest/manifest-v1.json`, `internal/projectmanifestfixture/`
- Filesystem-safe local workspace loader, fixtures, and runner: `internal/workspaceloader/`, `spec/fixtures/workspace-loader/manifest-v1.json`, `internal/workspaceloaderfixture/`
- Canonical generated `fairy.lock` v1 with strict parsing, stale comparison, and atomic replacement: `internal/lockfile/`
- MIR and C backend contracts: `docs/mir/`
- Package identity, visibility, source/interface hashing, per-file integrity, stable function identity, process-local MIR slices, and bounded whole-package cache: `docs/package/package-graph-v0.md`, `docs/package/package-instance-id-v1.md`, `docs/package/package-source-digest-v1.md`, `docs/package/source-file-identities-v1.md`, `docs/package/public-interface-hash-v1.md`, `docs/package/package-function-id-v1.md`, `docs/package/process-local-mir-artifact-v0.md`, `docs/package/whole-package-cache-v1.md`
- Manifest and lockfile readiness audit: `docs/package/manifest-lockfile-readiness.md`
- Project manifest v1: `docs/package/project-manifest-v1.md`
- Local workspace loader v1: `docs/package/workspace-loader-v1.md`
- Generated lockfile v1: `docs/package/lockfile-v1.md`
- Target and C ABI: `docs/abi/c-abi-v0.md`
- Runtime and standard library: `docs/runtime/runtime-stdlib-v0.md`
- Compatibility and versioning: `docs/compatibility/versioning-v0.md`
- Diagnostic catalog: `docs/diagnostics/catalog-v0.md`
- Atlas semantic snapshot and documentation sidecar, provisional SymbolId, logical planning, canonical transport, contained current-source-validated store, bounded package-local queries, canonical compiler-fact Context Packs, canonical exact retained-source/documentation attributed-text output, single-symbol Evidence Bundle v0, frontend-owned exact `///` attachment, exact catalog plus overview/show/around/text/evidence CLI, and whole-package rewrite measurement: `docs/atlas/`, `internal/lexer/documentation.go`, `internal/semanticsnapshot/`, `internal/atlasmap/`, `internal/contextpack/`, `internal/attributedtext/`, `internal/evidencebundle/`, `internal/workspaceloader/exact_source.go`, `internal/cli/atlas_pack*.go`, `internal/cli/atlas_text.go`, `internal/cli/atlas_evidence.go`, `internal/buildmeasure/atlas_rewrite.go`
- CLI contract: `docs/cli/command-contract.md`
- Workspace ownership: `docs/monorepo/package-ownership.md`

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not committed
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
