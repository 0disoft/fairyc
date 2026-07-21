# Fairy C

- Status: Stage 0 implementation prototype
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

Fairy C is an experimental systems programming language for new native software that needs C-like control, predictable costs, explicit C interoperability, and a substantially safer default resource model without a garbage collector or a heavy runtime.

The project is not trying to beat every hand-tuned C program or make every C ownership graph safe. Its intended advantage is that ordinary Fairy C code carries enough ownership, aliasing, bounds, and effect information for the compiler to reject common failures and generate competitive machine code without relying on build-mode-dependent undefined behavior.

## Current phase

The first Stage 0 frontend slice now exists in Go: immutable source bytes, a lossless lexer,
compact recoverable syntax, error-free AST lowering, deterministic `E00xx` diagnostics, and a
runner for all parser v0 fixtures. The next slice resolves names and produces typed HIR for the
  fixture-backed scalar, safe-borrow, scalar foreign-call, and declared-effect subset. Its semantic runner executes all 38 v0 resolve, type,
and local Flow cases, including integer-literal range rejection, branch- and loop-aware safe-borrow reassignment,
exact interprocedural borrow-return summaries, last-use loans, borrow-return origins, canonical
`@effects(...)` sets, and caller-superset enforcement. Whole-root resource Flow
  also executes all 12 v0 cases for moves, cleanup obligations, branch joins, normal, break, and continue loop edges,
consuming tagged matches, and live/dead resource reassignment. The
Region runner executes all 6 v1 cases, including package-proven `std::alloc::Arena`, hidden
allocation origins, lexical escape, capability storage, and plain-region resource rejection. The
  Result runner executes all 24 v0 cases for qualified construction, exhaustive shallow matching,
  must-use return/discard/propagation, local and parameter transfer, branch joins, loop back-edges and control exits,
overwrite and early-return rejection, exactly-once classification, non-Result `?`, and propagated
error compatibility. The MIR runner executes all
  11 v0 normalization cases through mutable-local block lowering and the MIR verifier, covering
left-to-right call evaluation, complete-root moves, normal-exit cleanup order, `Result` error
  cleanup, nearest-loop control, explicit checked arithmetic, and non-unwinding trap paths. The verifier mutation runner constructs all 10 named
  baselines and proves all 21 isolated `V1001` through `V1701` rejection recipes. The first MIR
  interpreter slice executes all 92 v0 cases for literal return, direct `i32` minimum constants, scalar fixed-array calls, contextual immutable and mutable byte-view borrowing, exact lengths, checked reads and indexed stores, function-call arithmetic,
boolean branches and logical not, source `&&` and `||` short-circuiting, scalar `while`, scalar
`Option` construction/matching, scalar `Result` construction/matching/propagation,
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
The constrained C backend emits deterministic C11 for 36 verified fixtures across the scalar
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
units that local x64 MSVC compiles, links, and executes at low and optimized profiles. Independent
compiler or assembly/register inspection, broader and bidirectional source-level Fairy crossings,
and a hosted receipt remain missing;
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
instead of retaining it. It reports the exact signed `i32` result or the same canonical product-trap
category and rejects program output. This is the first direct Windows target execution command; it
is still experimental and does not promote target maturity.
`experimental exec-target` is the program-facing sibling: it forwards bounded stdout and stderr
without a compiler banner and preserves the program's full Windows process status. The checked-in
`examples/windows_output/main.fy` uses a support module containing the one exact unsafe Windows
stdout write-all symbol inside a checked local helper. The raw declaration, safe text wrapper, and
`main` propagate `@effects(console)` while the byte-view helper remains pure. The wrapper returns
`Result<usize, ConsoleError>`, preserving either the exact byte count or the runtime status plus
completed prefix. Its `main` passes `&str` and `&[u8]` through safe internal calls; the
byte helper observes the exact NUL and `A` bytes through checked read-only indexing while the entry contains no raw pointer or `unsafe`,
prints `Fairy C says hi`, and exits 42. That is real
FY-to-MSVC process output, unsafe containment, and source-effect enforcement, but it is not yet a
portable `std::io` API or final symbolic error taxonomy.
A provisional SystemHeap implementation now lives in the target standard-library
boundary rather than the thin runtime. It normalizes empty allocation, admits current fundamental
alignments, preserves bytes across copy-and-release growth or shrink, and leaves the old block
unchanged when validation or allocation fails. A closed C-only probe and an internal generated FY
caller using the exact `void *`/`size_t` block each have four configured Clang/GCC low/optimized
executions. Neither has an observed hosted receipt. `experimental build` now selects this helper
when an actual generated call requires it, but no public `std::alloc` API exists. Arena, text
ownership, and Linux File now have their own C slices below; each now also has a raw generated-Fairy
source crossing.
A second provisional slice implements Buffer-owned initialized bytes, failure-atomic reserve,
relocating self-overlap append, length-overflow rollback, zeroed growth, truncate, clear, and drop.
Its closed C-only probe and an internal generated FY caller using the exact nested SystemHeap-block
plus `size_t` record each have four configured Clang/GCC low/optimized executions. Neither has an
observed hosted receipt. Three exact Buffer helpers are now admitted by the constrained source
backend and exercised by the closed native fixture. Used-symbol product linkage selects the Buffer
plus SystemHeap dependency closure; the public standard-library package remains absent.
A third slice implements Arena-owned byte nodes with one active generation-checked Region,
failure-atomic zeroed or copied placement, reset invalidation, stale-use rejection, and active-drop
rejection. Its closed C-only probe and an internal generated FY caller using the exact self-linked
node, Arena, and Region records each have four configured native executions. The generated caller
crosses real initialization, begin, zeroed allocation through a `void **` output slot, reset, and
drop. Used-symbol product linkage selects Arena plus SystemHeap, but lexical Region lowering and a
public `std::alloc` API remain absent. Windows `build-target` now admits the five exact lifecycle
symbols used by that example and observes non-null allocation plus node, active-state, and generation
transitions at both MSVC profiles. Its private generation transport is target-width `usize`, which
matches the selected Windows x64 C layout without becoming a public representation promise. This
Windows path does not dereference the returned storage or independently inspect its zeroed bytes;
the Linux probes retain byte-level evidence.
The first text slice now validates canonical UTF-8 without allocation and rejects slicing through a
multibyte sequence. Its native probe includes overlong forms, surrogates, values above U+10FFFF,
truncation, and obsolete leaders and remains unobserved; raw source transports follow below.
Its empty-input validator now has exact internal `fy_std_*_v0` symbols and two real FY-to-stdlib-C
four-way crossings: direct validation and `@repr(C)` output-record initialization through a verified
raw address. The self-contained empty validator call is also the first configured product-build
smoke that requires automatic standard-library linkage. Compiler-owned `c"..."` storage now reaches
the C backend as immutable static bytes, and an explicit unsafe direct-literal conversion can expose
its read-only `*u8` address to raw FFI. Compiler-owned `b"..."` storage now independently reaches a
private non-null `{ data, length }` byte-slice transport, including an empty length-zero sentinel;
typed locals may expose only the immutable data address through explicit unsafe conversion. Plain
`"..."` literals now independently reach a private non-null `fy_str` view after complete decoded
UTF-8 validation, with deduplicated immutable storage and the same empty length-zero sentinel rule;
typed locals can expose only immutable data through explicit unsafe conversion. Safe borrowed
`&str` and `&[u8]` views expose `.len` as their `usize` byte count and `.is_empty` as its exact
zero-length `bool` without scanning or allocation.
Immutable `&[u8]` additionally supports read-only `usize` indexing to `u8` and end-exclusive
`bytes[start..end]`, `bytes[..end]`, `bytes[start..]`, and `bytes[..]` subviews behind verified
`bounds` traps. Omitted bounds normalize to zero or the retained receiver length; written operands
evaluate once left to right and preserve the receiver's borrow origin. Mutable `&mut [u8]` also
supports checked `u8` reads and indexed stores; the place is checked before the right-hand side is
evaluated. Text indexing or slicing, a stable public string ABI, and the general safe text API still
do not exist.
Fixed arrays of `bool`, `i32`, `u8`, `u32`, or `usize` with `1 <= N <= 4096` now construct from
nonempty list literals, cross internal calls and returns as inert copy values, expose exact
zero-cost `usize` `.len`, and support checked read-only `usize` indexing. A direct shared borrow of
`[u8; N]` contextually becomes `&[u8]` where that type is expected, so local arrays can use the
existing immutable byte-view calls and operations without unsafe code. A direct exclusive borrow
of a mutable `[u8; N]` likewise becomes `&mut [u8]` only where that internal parameter type is
expected. It carries exact length and may expose `*mut u8` only inside unsafe wrapper code; safe
checked reads and indexed `u8` stores are available, while mutable slicing remains absent. Mutable
fixed scalar-array locals and `mut` parameters likewise support checked element assignment. The ceiling is a provisional Stage 0
implementation guard, not a stable language or ABI limit. A mutable safe-borrow local may
be reassigned through straight-line, branch, and `while` control flow; Flow replaces its provenance
after evaluating the right-hand side, joins reachable branch/header/`continue`/`break` origins
without guessing, and releases old owners only after their last reachable loan use. Exact whole-
array `&[T; N]` and `&mut [T; N]` types may now cross internal calls without becoming copyable
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
Fairy `String` and `CString` resources, maps operation failures into source `Result`, exposes only
length and typed error metadata, and runs its C release through verified resource drop glue. This is
an example-level safe boundary, not a stable public `std::text` module, ABI, or borrowed-view API.
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
text, interior-NUL rejection, scalar lengths, and implicit cleanup while keeping raw pointers and
foreign calls out of its entry module.
The Linux File C slice now exposes partial I/O, write-all progress, sync, and consuming close that
is never retried, even after failure. Its C-only four-way probe remains configured but unobserved.
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
`filesystem` effects, writes and synchronizes `FairyFile`, reads it through `read(&mut [u8])`
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
types and is configured for Clang/GCC at `-O0` and `-O2`. This Windows host cannot execute those
four probe runs, and no hosted success receipt has been observed, so `c-layout-probes` remains a
missing target evidence marker.
A separate Windows C11 probe translates the eight valid LLP64 cases into private numeric types and
compiles and executes them twice with local x64 MSVC through the configured host smoke. The same
smoke separately crosses all 12 validated Microsoft x64 call cases between caller and callee
translation units at both profiles. A third three-object probe raises and catches a private Windows
exception, virtual-walks raiser, termination, and catcher frames, and requires one abnormal
`__finally` execution under both profiles. The target marker remains missing until broader layout
and independent/source-level/hosted ABI and unwind evidence exists. A host-independent PE parser
now validates the host-smoke product, both declared-target profile products, both safe File and
safe owned-text profile products, two direct run-target temporary products, two exec-target output
products, and all ten raw Windows File, owned-text, layout, ABI, and unwind probe images as bounded
AMD64 PE32+ console executables
with aligned headers and sections and an executable `.text` entry point. That linked-image path does not inspect
object symbols, relocations, or Windows unwind records. The same package separately parses every
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
Fairy/foreign-frame crossings, object-level unwind relocations, and hosted fixtures remain open.
A separate fail-closed Linux/amd64 command is now present to compile those cases with Clang and GCC
at `-O0` and `-O2`, link a closed conformance runtime, and compare process outcomes plus exact
source-linked resource-drop sequences. Product emission and product builds contain no trace observer. It cannot run on
this Windows host because no Linux Clang/GCC execution environment is installed, and its hosted CI result
has not yet been observed. The target therefore remains unsupported.
A provisional source-independent `fairyc explain <diagnostic-code>` lookup reports reserved Stage 0
meanings and categories without reading source or invoking a toolchain. A provisional `fairyc targets`
report lists declared target maturity and missing evidence without probing this Windows host. An unstable
`fairyc experimental` CLI can inspect syntax, syntax-check, semantic-check, emit the
scalar, flat-resource, and resource-tagged C11 translation unit, build that subset into a Linux executable on an eligible host,
inspect the declared target specification, or interpret an explicit source-root-relative FY module
closure. Program commands use one entry plus repeated `--source`; semantic checking uses its explicit
positional module set. There is no stable CLI, user drop-body execution, complete runtime,
safe standard library, package manager, supported target, release-stable CLI, or stable language
revision yet.

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

For repeated Windows use, `DEVELOPMENT.md` gives the WSL-free manual bootstrap that creates ignored
`build/fairyc.exe`. The configured Windows smoke separately builds a temporary standalone compiler
process and requires that executable to run `examples/windows_quickstart/main.fy` through MSVC with
native result 42 by iterating two bytes with checked `usize` arithmetic and safely converting each
indexed `u8` value to `i32`,
then execute `examples/windows_output/main.fy` with exact stdout and process status
42 while explicitly supplying its separate `windows_console.fy` module. The compiler binary is
still built from Go; only the input programs and resulting temporary
target executables are Fairy C.

`explain` and `targets` do not read the current directory or source tree. The former marks its
diagnostic catalog as provisional; the latter reports declared TargetSpec evidence without probing
local MSVC, SDK, WSL, or any other toolchain. The experimental source commands treat the current directory as the temporary source
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
verified MIR in the bounded interpreter, including boolean `if`, short-circuit control, scalar `while`,
resource construction, field reads, whole-root call moves, resource-bearing built-in tagged values,
ordered resource reassignment, consuming-close ownership patterns, cleanup drop events, Region
fallthrough, direct-return, Result-propagation, nested, break, and continue reset paths, checked
addition, subtraction, multiplication, negation, matching-type bitwise operations, and checked
signed or unsigned right shift. It performs no source
discovery and needs no C toolchain, MSVC, WSL, or host subprocess, and reports either `result: <i32>` or a canonical
trap. The official
release-stable `fairyc check` command remains unavailable; `experimental check` does not settle
package discovery, targets, configuration, JSON, or stable output and exit codes.

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
- Target layout engine and fixtures: `internal/layout/`, `spec/fixtures/layout/manifest-v0.json`, `spec/fixtures/layout/manifest-windows-msvc-v0.json`
- Shared Windows MSVC C probe type emitter: `internal/msvccprobe/`
- Windows MSVC native layout probe generator: `internal/windowslayoutprobe/`
- Target C ABI classifiers and fixtures: `internal/cabi/`, `spec/fixtures/c-abi/manifest-v0.json`, `spec/fixtures/c-abi/manifest-windows-msvc-v0.json`
- Windows MSVC native ABI probe generator: `internal/windowsabiprobe/`
- Windows x64 native exception and virtual-unwind probe generator: `internal/windowsunwindprobe/`
- Windows AMD64 PE, COFF object, and x64 unwind structure inspector: `internal/pecoff/`
- Resolved package graph engine, fixture runner, and fixtures: `internal/packagegraph/`, `internal/packagefixture/`, `spec/fixtures/package/manifest-v0.json`
- MIR and C backend contracts: `docs/mir/`
- Package identity and visibility: `docs/package/package-graph-v0.md`
- Target and C ABI: `docs/abi/c-abi-v0.md`
- Runtime and standard library: `docs/runtime/runtime-stdlib-v0.md`
- Compatibility and versioning: `docs/compatibility/versioning-v0.md`
- Diagnostic catalog: `docs/diagnostics/catalog-v0.md`
- Atlas schemas: `docs/atlas/`
- CLI contract: `docs/cli/command-contract.md`
- Workspace ownership: `docs/monorepo/package-ownership.md`

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not committed
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
