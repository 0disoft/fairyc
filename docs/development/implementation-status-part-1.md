# Implementation Status

- Status: Draft

Phase history moved from `README.md` unchanged. It records what the Stage 0 compiler
implemented at each step, including the command surface as it stood at that time. Older entries
describe states that later entries replace, so read it as a chronological record rather than a
current contract.

Current usage lives in [README.md](../../README.md); the manual developer examples live in
[prototype-usage.md](prototype-usage.md).
This page is part 1 of 2. Continue with [part 2](implementation-status-part-2.md).

## Current phase (part 1 of 2)

The Go Stage 0 compiler now implements immutable source bytes, a lossless lexer, compact recoverable
syntax, error-free AST lowering, typed HIR, resource and borrow Flow, verified MIR, a bounded
interpreter, deterministic C11 transport, strict local workspaces, stable new/check/build/run/test/C
  emission commands, and experimental Atlas fact export. The semantic runner executes all 61 v0 resolve, type,
and local Flow cases, including explicit resource-factory construction with cross-module resource
representation privacy, explicit `source_location()`, bounded scalar compile-time evaluation, narrow callable `@must_use`, integer-literal range rejection, checked compound assignment, branch- and loop-aware safe-borrow reassignment,
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
  13 v0 normalization cases through mutable-local block lowering and the MIR verifier, covering
left-to-right call evaluation, complete-root moves, normal-exit cleanup order, `Result` error
  cleanup, nearest-loop control, explicit checked arithmetic, and non-unwinding trap paths. The verifier mutation runner constructs all 11 named
  baselines and proves all 23 isolated `V1001` through `V1701` rejection recipes, including a
  missing internal call target rejected before artifact or backend handoff. The first MIR
  interpreter slice executes all 94 v0 cases for literal return, bounded lexical `defer`, direct `i32` minimum constants, scalar fixed-array calls, contextual immutable and mutable byte-view borrowing, exact lengths, checked reads and indexed stores, function-call arithmetic,
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
The constrained C backend emits deterministic C11 for 46 verified fixtures across the scalar
payload surface, restricted inert user enums, flat nonempty move-only resources over Fairy copy
scalars or raw pointers with empty or pure nonempty drop bodies, copy-scalar/inert-struct and resource-bearing
built-in `Option` and `Result` values,
borrowed-view byte lengths, checked
immutable byte-slice indexing and end-exclusive slicing, the target-independent value-preserving integer casts `u8 as i32`,
`u8 as u32`, `u8 as usize`, and `u32 as usize`, plus exact explicit `i32 <-> c::int` casts, checked `u8`, `u32`, and `usize` addition,
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
separate run-only trap control pipe and has no such collision. This direct target command remains
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
The compiler-bundled SDK `0.0.0-stage0.10` exposes safe text
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
The same SDK now exposes a move-only `std::bytes::Buffer`. Safe code can construct or copy one,
reserve, append, zero-grow, truncate, clear, inspect length/capacity/emptiness, and borrow the
initialized prefix as `ref u8[]` or `ref mut u8[]`; raw pointers and all seven runtime symbols stay
inside the SDK. `examples/std_bytes_quickstart` runs that lifecycle and automatic cleanup in both
Windows development and release builds without `unsafe` in user code.
The same SDK exposes move-only `std::alloc::Arena`, fallible `alloc::create()`, and stable
`AllocError` status inspection. `examples/std_alloc_quickstart` reaches verified MIR, then builds
and executes with automatic Arena cleanup under both Windows MSVC profiles; lexical Region C
lowering remains the next gate.
The same SDK now exposes move-only `std::text::String` and `std::text::CString`. Safe code can
explicitly validate any `ref u8[]` as UTF-8 and receive an allocation-free
`Result<ref str, TextError>` view tied to the input borrow. It can also copy UTF-8 text, append to
String, inspect byte lengths and emptiness, borrow owner-bound `ref str` or
`ref CStr`, and observe typed text/allocation status on failure. `examples/std_text_quickstart`
executes valid multibyte and invalid-byte validation, nonempty and empty construction, append,
interior-NUL rejection, views, and implicit cleanup
in both Windows development and release builds without raw pointers, foreign declarations, or
`unsafe` in user code.
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
All seven Buffer helpers are admitted by the constrained source backend. Used-symbol product
linkage selects the Buffer plus SystemHeap dependency closure, and ADR 0246's bundled
`std::bytes` module contains the raw boundary behind a move-only safe resource. An independent raw
caller still links the target-specific Buffer closure as a separate MSVC translation unit at both
profiles. Its probe-only allocator forces initialization, reserve, relocating self-overlap append,
and zeroed-resize allocation failures, requires exact pointer/metadata/byte preservation, proves
normal recovery and zeroed growth, and returns live allocations to zero after drop. Product linkage
retains the ordinary allocator.
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
promotions. Bounded `defer` captures copy-scalar arguments immediately and executes internal `void`
calls in LIFO order when their lexical block exits; traps do not unwind through those calls. Text
indexing or slicing, Unicode algorithms, lossy decoding, implicit byte-to-text conversion, and a
stable public string ABI still do not exist.
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
drop glue. Exact unsafe raw-parts built-ins remain inside the SDK, and Flow rejects moving an owner
before the derived view's last use. ADR 0247 promotes this boundary to public `std::text` while
keeping private layout, raw symbols, mutable bytes, and any foreign borrowed-view ABI excluded.
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
It accepts a bounded ASCII bootstrap path, preserves one-call partial read/write counts and write-all progress, syncs
explicitly, consumes state before one `CloseHandle` attempt on both success and failure, and makes
drop a silent one-attempt fallback. A separate-translation-unit probe creates, writes, syncs,
closes, reopens, reads in three-byte chunks, observes EOF, inspects size and absolute position,
rejects an overflowing offset without moving the cursor, reports `ERROR_INVALID_HANDLE` for size
and close after an externally closed handle, and checks repeated close and drop at low and optimized
local MSVC profiles. The closed raw call set is target-bound
through the closed resolver. A checked-in two-module FY example now wraps that exact component in a
move-only `File`, preserves status, Win32 code, and completed bytes in `FileError`, carries
`filesystem` effects, writes and synchronizes `FairyFile`, observes its size and resets its absolute cursor,
reads it through `read(ref mut u8[])`
backed by a mutable fixed array, observes EOF, consumes explicit close, and proves implicit drop by
successfully reopening a path whose live write handle would deny sharing. Low and optimized MSVC builds both
return 42 after object, PE, and unwind inspection. ADR 0264 now promotes that safe ownership and
I/O contract into compiler-bundled `std::fs`; ADR 0269 adds allocation-free `position` and absolute
`seek_start`, ADR 0270 adds shared `size`, ADR 0271 adds existing-file read/write `open_update`,
ADR 0272 adds cursor-preserving `set_len`, ADR 0274 adds atomic read/write `open_or_create` without
truncation, ADR 0275 adds atomic no-clobber `create_new`, ADR 0276 adds seek-resistant
write-only `open_append`, ADR 0277 adds same-handle target-neutral file-kind observation, ADR 0278
adds exact-fill reads with explicit unexpected-EOF prefixes, ADR 0279 adds opaque same-handle
file identity comparison, ADR 0281 promotes target-neutral file kinds to an exhaustive public
enum, ADR 0285 exposes one-call partial `write`, ADR 0288 appends one bounded read directly into
owned Buffer spare capacity, ADR 0290 reads from the current cursor through EOF into that Buffer,
ADR 0291 removes one closed file path with exact missing and sharing failures, ADR 0292 adds a
file-only no-replace rename without cross-volume copy, and ADR 0293 adds non-recursive one-level
directory creation plus empty-directory removal under SDK `0.0.0-stage0.31`. The public slice remains
Windows MSVC-only and does
not claim portable Unicode paths, general uninitialized-memory access, a supported target, or public layout.
