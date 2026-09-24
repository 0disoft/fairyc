# Fairy C

- Status: Stage 0 implementation prototype
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

- License: [Apache-2.0](LICENSE) for project-owned components; see
  [licensing scope](LICENSING.md) and [third-party notices](THIRD_PARTY_NOTICES.md).

Fairy C is an experimental systems programming language for new native software that needs C-like control, predictable costs, explicit C interoperability, and a substantially safer default resource model without a garbage collector or a heavy runtime.

The project is not trying to beat every hand-tuned C program or make every C ownership graph safe. Its intended advantage is that ordinary Fairy C code carries enough ownership, aliasing, bounds, and effect information for the compiler to reject common failures and generate competitive machine code without relying on build-mode-dependent undefined behavior.

## Human quickstart

### Get the compiler

There is no approved public installer or binary release yet. For a source checkout,
follow the [Windows compiler build guide](DEVELOPMENT.md#local-windows-compiler-executable).
If you received the separately verified experimental ZIP, extract it and keep its
license files together. Go and Bun are compiler-build tools, not prerequisites for
using that executable.

The verified starter path is native Windows/AMD64 with x64 MSVC and a Windows SDK.
Install Microsoft's [C++ Build Tools](https://learn.microsoft.com/en-us/cpp/overview/acquire-msvc)
with the Desktop development with C++ workload for native tests and execution.
WSL2, Linux and a virtual machine are not required. Source checking and semantic
tests do not require the C compiler. Other target mechanisms and CI probes are not
a promise of equivalent end-user support.

### Create, test and run

With a built `fairyc` executable, run these product commands manually from a scratch parent
directory to create one explicit-identity starter. The commands below assume the
executable is on PATH; in PowerShell you can instead use its absolute path with `&`.

```text
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
fairyc check
fairyc check --locked
fairyc test --list --json --locked
fairyc test --json --locked
fairyc test --locked
fairyc test --native --locked
fairyc test --native --release --locked
fairyc run --locked
fairyc run --release --locked
```

The generated `src/main.fy` is a small complete program:

```text
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

Its test reports one passed, zero failed. Both run modes exit zero without output.

The first check validates the generated source and publishes `fairy.lock`. On Windows/amd64 with
local x64 MSVC, `fairyc test --native --locked` compiles each generated test into an isolated
temporary image, while `fairyc run --locked` compiles the `.fy` program through generated C and runs
its temporary native image with the caller's stdin attached after successful compilation. The
generated `README.md` contains the exact next steps and a safe
`std::io::write` example for direct stable execution. `fairyc new` never merges, upgrades, or
overwrites an existing path.

### Experimental limits and reporting

This is the Go Stage 0 compiler, not the experimental Fairy-written parser or a
self-hosted release. No production-readiness, stable compatibility or complete
language/backend coverage is promised. See the [readiness evidence](docs/ops/parser-public-readiness.md)
for local and second-laptop evidence, exact candidate hashes, and the
[successful CI run on `34fe041c`](https://github.com/0disoft/fairyc/actions/runs/35103049037).

For non-sensitive bugs, use [GitHub Issues](https://github.com/0disoft/fairyc/issues)
with the compiler commit or executable hash, OS/toolchain versions, a minimal
reproducer, expected/actual behavior and redacted diagnostics. Issues are enabled;
while the repository is private, repository access is required.

A private vulnerability-reporting channel has not yet been confirmed. Do not post
credentials, private code, personal data or vulnerability details in public issues.
Confirm a private reporting channel with the maintainer before sending sensitive
details. Establishing that channel remains a public-launch prerequisite; no
response-time or supported-version commitment is implied.

## Current phase

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

ADR 0294 adds directory-only no-replace rename, ADR 0296 adds recursive parent creation with
pre-effect lexical validation, ADR 0297 adds no-replace byte copy, and ADR 0298 adds
`std::fs::PathKind` plus `path_kind`. ADR 0299 moves the SDK to `0.0.0-stage0.36` with
`std::bytes::append_byte`, a single-byte wrapper over the existing fallible Buffer append primitive.
ADR 0302 moves the SDK to `0.0.0-stage0.37` with
`std::text::as_bytes(ref str) -> ref u8[]`, an allocation-free immutable byte view that preserves
the exact input borrow origin and UTF-8 bytes without exposing raw storage.
ADR 0303 moves the SDK to `0.0.0-stage0.38` with allocation-free
`std::io::write_stderr_i32`/`write_stderr_u32`/`write_stderr_usize` and ASCII
`write_stderr_bool`, preserving exact completed-prefix failures and adding no bool-specific symbol.
ADR 0305 moves the SDK to `0.0.0-stage0.39` with bounded
`std::fs::seek_current(ref mut File, i32)` and `std::fs::seek_end(ref mut File, i32)`. Both reuse
the live Windows handle, return the resulting absolute `usize` position, preserve the cursor when a
negative delta would cross byte zero, and do not extend the file merely by moving beyond EOF.
ADR 0306 moves the SDK to `0.0.0-stage0.40` with
`std::fs::is_read_only(ref File) -> Result<bool, FileError>`. It observes
`FILE_ATTRIBUTE_READONLY` through the same live handle without path lookup, allocation, cursor
movement, or length change. The boolean is observed metadata rather than a guarantee that a write
will succeed; handle access, ACLs, sharing, filesystem policy, and later external changes remain
independent.
ADR 0307 moves the SDK to `0.0.0-stage0.41` with
`std::fs::hard_link_count(ref File) -> Result<usize, FileError>`. It observes
`FILE_STANDARD_INFO.NumberOfLinks` through the same live handle without path lookup, allocation,
cursor movement, or length change. The value is a momentary count rather than path enumeration,
authorization, or persistent identity.
ADR 0308 moves the SDK to `0.0.0-stage0.42` with
`std::fs::hard_link(ref CStr, ref CStr) -> Result<i32, FileError>`. It performs one bounded,
no-replace `CreateHardLinkW` operation, copies no bytes, and preserves exact collision and missing
source errors. Both names share identity and bytes; removing one name leaves the other alive.
ADR 0309 moves the SDK to `0.0.0-stage0.43` with
`std::fs::allocated_size(ref File) -> Result<usize, FileError>`. It observes the allocation length
reported through the same live handle without allocation, cursor movement, or length change. The
momentary value is not a portable physical-disk, compression, sparse-extent, quota, or billing
promise.
ADR 0310 moves the SDK to `0.0.0-stage0.44` with
`std::fs::set_read_only(ref mut File, bool) -> Result<bool, FileError>`. It changes only the
read-only attribute through the same live handle, preserves unrelated attributes and timestamps,
and exposes insufficient handle access as an exact target error. Read-only metadata remains
distinct from ACL, sharing, filesystem policy, and later write authorization.
ADR 0311 moves the SDK to `0.0.0-stage0.45` with
`std::fs::remove_dir_all(ref CStr) -> Result<usize, FileError>`. It rejects filesystem roots and
ambiguous bounded paths before effects, removes observed directory reparse points as leaves, and
reports exact removed-object progress on success or failure. Traversal order is filesystem-defined;
the operation is not atomic, rollback-capable, containment proof, or race-free authorization.
ADR 0317 moves the SDK to `0.0.0-stage0.46` with effect-free
`path_kind_is_file(PathKind)` and `path_kind_is_directory(PathKind)`. Both are exhaustive and keep
`.Other` false without turning a path snapshot into authorization or a race-free precheck.
ADR 0318 moves the SDK to `0.0.0-stage0.47` with allocation-free `std::bytes::empty()` and
`std::text::into_buffer(String) -> bytes::Buffer`. The latter transfers the exact String allocation
without copying, returns mutable bytes, and requires `from_buffer` to validate UTF-8 again after any
mutation.

ADR 0319 moves the SDK to `0.0.0-stage0.48` with public
`std::fs::metadata(ref File) -> Result<FileMetadata, FileError>`. It collects size, allocated size,
hard-link count, kind, and read-only state sequentially through the existing same-handle operations;
the compiler now emits enum-containing private structs in aggregate dependency order without a new
runtime symbol or atomic-snapshot promise.

ADR 0331 moves the SDK to `0.0.0-stage0.49` and makes the File open/read/write/sync/close core
target-neutral across Linux GNU and Windows MSVC. The closed linker selects a descriptor or HANDLE
adapter behind seven private common symbols; Windows-only path, seek, exact-read, Buffer-read, and
metadata extensions remain explicitly target-bound.

ADR 0332 moves the SDK to `0.0.0-stage0.50` and makes `std::fs::position` plus absolute
`std::fs::seek_start` target-neutral. Both operate on the already-open descriptor or HANDLE without
reopening a path; relative/end seek, size, metadata, exact-read, Buffer-read, and path mutation
remain target-bound.

ADR 0333 moves the SDK to `0.0.0-stage0.51` and makes `std::fs::size` target-neutral. Linux uses
`fstat` on the already-open descriptor and Windows delegates to `GetFileSizeEx`; both preserve the
cursor and ownership. Relative/end seek, other metadata, exact-read, Buffer-read, and path mutation
remain target-bound.

ADR 0334 moves the SDK to `0.0.0-stage0.52` and makes bounded `std::fs::seek_current` plus
`std::fs::seek_end` target-neutral. Linux uses `lseek` with `SEEK_CUR` or `SEEK_END`; Windows keeps
the existing `SetFilePointerEx` behavior. Other metadata, exact-read, Buffer-read, and path mutation
remain target-bound.

ADR 0342 moves the SDK to `0.0.0-stage0.53` and language `fairyc-language/11`. Stable workspace
analysis admits canonical resource abilities and target-neutral on-chain effects, with
public-interface v4 and cache fixture v6 preserving those facts. The attributes add no ambient
chain API or runtime symbol; `examples/contract_capabilities_quickstart/` is the native result-42
admission proof.

ADR 0343 moves the SDK to `0.0.0-stage0.54` and language `fairyc-language/12`. `burn local;`
discharges one declaration-owned inert no-drop resource through a distinct verified MIR operation;
it is neither ordinary drop nor a generic forget primitive. `examples/asset_burn_quickstart/`
returns native 42 under low and optimized MSVC while its unreachable cleanup body would trap.

ADR 0377 introduced language `fairyc-language/13` and the compiler-bundled SDK identity
`0.0.0-stage0.56`. One `@contract_uid("...") state { ... }` declaration per package now lowers from
Fairy source into the existing typed HIR layout, public-interface v5 hash, and cache-key v4 identity.
The first fields are explicit `cell Value name;` and `map Key Value name;` declarations over a closed
fixed codec set. Source state operations, native persistence, and deployable targets remain closed.

ADR 0385 introduced stable explicit scalar generics in language `/14`. ADR 0395 advanced
the language to `fairyc-language/15` and SDK `0.0.0-stage0.60`, allowing record initializers
such as `Token { .kind, .start, .end }` as shorthand for same-name lexical values.
ADR 0396 is now current at `fairyc-language/16` and SDK `0.0.0-stage0.61`, adding
position-preserving argument labels such as `f(.minimum=1, .depth=0, .allow_record=true)`.
Labels are permitted only for same-module private non-generic direct calls and do not
change source evaluation order. Existing workspaces must update the language pin and refresh locks.
ADR 0399 advances language to `fairyc-language/17` and SDK to `0.0.0-stage0.62`.
ADR 0400 adds checked mutable `Vec.at_mut` access in SDK `0.0.0-stage0.63` without changing the language revision.
SDK `0.0.0-stage0.64` preserves that API and lowers its checked slot address directly in generated C.
ADR 0402 advances language to `fairyc-language/18` and SDK to `0.0.0-stage0.65` so a Result switch case ending in a direct `return` can coexist with a value-producing case.
ADR 0403 advances language to `fairyc-language/19` and SDK to `0.0.0-stage0.66` for narrow read-only `ref u8[]` iteration; its C-relative cost gate is still pending.
ADR 0405 advances only the SDK to `0.0.0-stage0.67`, lowering checked shared `Vec.at<T>` access through a typed element address while retaining owner-bound borrowing.
ADR 0407 advances only the SDK to `0.0.0-stage0.68` for strict UTF-8 Windows file paths; a portable path type remains future work.
An owned Result now supports `value.is_ok` as the existing two-arm success test,
including active-payload cleanup and its ability/effect checks. Existing workspaces
must update the language pin and refresh locks.
Stable workspace `check`, `build`, `run`, and `test` accept explicit scalar generic functions such
as `fn T identity<T>(T value) { return value; }` and `identity<i32>(42)`, including public functions
across imports. Type inference, nominal generic arguments, and user-defined static interfaces
remain unsupported for user-defined templates. Update existing manifest language pins and refresh their locks.

ADR 0390 adds the on-demand `std::vec::Vec<T>` module in SDK `0.0.0-stage0.58`,
without changing language `/14` or runtime `/0`. Stable Windows check/build/run
and native tests support scalar and resource elements, fallible push/reserve,
pop/clear and immutable owner-bound access. User generic aggregates, mutable
element access, iteration and nested vectors remain unadmitted. See
[`std_vec_quickstart`](examples/std_vec_quickstart/README.md). ADR 0393 adds
capacity-preserving `truncate<T>` in SDK `0.0.0-stage0.59`; refresh existing SDK locks.

ADR 0344 adds an experimental compiler-private contract-state transaction MIR beside the stable
whole-program MIR. It reconstructs exact layouts, verifies canonical cell/map operations and
commit/reject/trap exits, then executes through the ADR 0338 atomic oracle. Source syntax, HIR,
stable artifacts, caches, backends, targets, and public version identities remain unchanged.

ADR 0345 adds source-independent package-owned state layouts to HIR, lowers them through the
transaction MIR, and carries the exact layout hash in public-interface v5. Whole-package cache key
v3 additionally consumes the owning package's current interface, so generated HIR layout changes
invalidate both the owner and its dependents. Source syntax and stable execution surfaces remain
closed.

ADR 0346 gives each compiler-private state program one opaque transaction capability and requires
every operation to use that exact identity. The verifier also derives `storage_read` and
`storage_write` requirements before the oracle begins.

ADR 0347 adds source-independent HIR transaction plans owned by an existing package function. The
function remains the sole effect authority; lowering resolves field names through its exact layout,
creates one fresh capability, verifies the result, and executes the generated program through the
failure-atomic oracle. Source syntax, stable/cached body transport, and target execution remain
closed.

ADR 0348 gives those verified HIR transaction bodies an exact
`fairyc.contract-state-package-body/v1` SHA-256 identity. Stable package coordinates, layout hash,
function signatures, effects, exact operation bytes/order, and exits participate; build-local
numbers, spans, and fresh capability tokens do not. Cache-key participation and payload transport
were separate follow-up gates.

ADR 0349 advances whole-package cache keys to v4 and requires that exact body identity for every
function-owning package before lookup. A body-only change invalidates its owner without widening
dependency entries beyond public-interface v5.

ADR 0350 adds an exact-version, caller-bounded canonical payload for those bodies. It excludes
build-local identities and capability tokens, reconstructs each layout, creates fresh private
capabilities, re-enters the verifier, and recomputes the ADR 0348 hash. Stable package-artifact
integration and cache storage were still closed at that boundary.

ADR 0351 checks the cache-selected expected hash, current coordinate/layout, and byte-exact current
HIR function descriptors before mapping that payload to fresh package/function symbols. Every
transaction receives another fresh capability and re-enters verification without requiring the
current HIR source-operation side table. Combined cached reassembly remained closed.

ADR 0352 adds `fairyc-stable-package-compilation/1`, a bounded canonical outer artifact that pairs
stable package MIR v3 with the exact state-body payload only when both independently validate and
share one package coordinate. ADR 0353 advances persistent cache identity to `/2`, stores exactly
that compilation payload, requires the key-selected body hash, and quarantines contradictory or
legacy MIR-only generations. ADR 0354 reassembles both members under one fresh coordinate-owned
function-symbol graph and returns cached state transactions that the compiler-private oracle can
execute after exact current-body identity selection without consulting the current operation side
table during reassembly. ADR 0355 publishes ordinary and contract-state MIR atomically through one
verified `compiler.Result` for cold and cached compilation paths. ADR 0356 admits only
`storage_read` and `storage_write` in the transaction program until block-context, event, and
external-call operations have their own verifier contracts. ADR 0357 rejects calls without a
transaction representation. ADR 0379 now admits only statically resolved same-package no-argument
`void` helpers whose exact call sequence, transitive storage effects, capability, costs, bodies,
and rollback behavior are verified. Indirect, foreign, cross-package, argument-bearing,
value-returning, recursive, and reentrant calls remain closed. ADR 0358 additionally rejects conditionals, loops, assertions, Result
propagation, and short-circuit operators so every current owner remains straight-line under the
exact compiler-private HIR boundary. ADR 0359 rejects unsafe declarations/scopes and raw-pointer
types through signatures, locals, expressions, and nested aggregates while preserving ordinary
safe-borrow and pointer-free resource types. ADR 0360 rejects receiver-bound methods and automatic
drop glue as state owners while preserving explicit static associated functions. ADR 0361 rejects
ordinary checked arithmetic and bounds operations whose conditional traps are absent from the
linear state side model while retaining its explicit static trap exit. ADR 0362 rejects owned
resource values and lexical regions whose drop, burn, and reset lifetimes are absent from that side
model while preserving safe resource borrows and static resource namespaces. ADR 0363 then freezes
the remaining pure owner HIR as a positive allowlist so future statement, expression, operator, or
expression-effect additions fail closed. ADR 0364 adds a versioned five-case transaction fixture
corpus that executes commit, reject, trap, delete, and write-limit behavior through HIR lowering,
verified transaction MIR, and the semantic oracle without admitting source or production storage.
ADR 0365 preserves that v0 baseline and adds a current v1 corpus for map write/read, map delete,
exact domain-separated storage keys, and map-key-limit rollback.
ADR 0366 routes both corpora through `compiler.LowerAnalyzed`, requires one atomic verified
ordinary/state MIR result, and proves malformed state bytes publish neither executable member.
ADR 0367 separately versions three overlapping semantic-oracle schedules for stale-writer conflict,
stable start snapshots, and fresh-version observation. A conflict publishes no stale state and does
not revoke already verified compiler IR.
ADR 0368 preserves that cell baseline and adds three map schedules proving that distinct keys still
share one optimistic store version, with exact physical-key and snapshot evidence.

ADR 0378 adds a bounded compiler-private SSA graph for data-dependent state work. Cell and map reads
define codec-typed values plus explicit `bool/v1` presence, branches consume bool values, and block
parameters make every join dependency explicit. Dense reachable acyclic blocks, capability identity,
storage effects, and unit operation/terminator costs are verified before the oracle begins. Current
v2 fixtures execute cell-read branch/join and map-found branch paths through atomic compiler
publication and reject cycle, join-type, missing-cost, and unknown-value mutations. Graph bodies use
the v2 stable envelope and existing body-hash cache invalidation while legacy v1 payload bytes remain
unchanged. Source state access, loops, native lowering, and production persistence stay closed.

ADR 0379 adds compiler-private bounded helper calls. A root can reach at most 32 sorted
module-qualified `void fn()` helpers across at most 1024 total operations. Calls cost one unit and
execute inside the root snapshot and overlay; helper return resumes, while helper reject or trap
rolls back the complete attempt. The v3 stable body includes helper order, calls, bodies, effects,
exits, and costs while preserving legacy v1 and graph v2 bytes. Manifest v3 executes a
pure-to-state helper chain and rejection rollback and rejects recursion and transitive-effect drift.

ADR 0323 reuses that aggregate graph for acyclic inert enums nested inside user-enum payloads. The
46th C-emission fixture and low/optimized generated-Fairy MSVC executions preserve nested
construction, by-value transport, extraction, exhaustive switching, and result 42 without changing
language, SDK, runtime ABI, or public ABI identities.

ADR 0320 closes the verbatim C-name false-acceptance gap without changing language, SDK, or runtime
ABI identities. `@repr(C)` tags and fields plus external symbols now reject C implementation-reserved
forms, generated-header macros, and backend-owned names before any invalid or colliding C11 is emitted.
The path-kind query distinguishes ordinary files, directories, and other observed attributes without opening
an owner, but its snapshot is not stable identity, authorization, containment, or a race-free
precheck. `std::fs::copy_file_bytes` holds exclusive source and `CREATE_NEW` destination handles,
never replaces an existing destination, returns exact copied bytes, and preserves any completed
destination prefix on a later failure without implying metadata copy, atomicity, or durability.
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
Those C scalar types remain distinct from coincidentally represented Fairy scalars and emit their
exact C spellings with target and provider assertions. Signed 32-bit `c::int` alone has explicit
bidirectional conversion with `i32` and pure interpreter support; other C scalars have no source
literal, operator, cast, raw-pointer, or interpreter support. Nonempty `@repr(C)` records may use exact C-scalar fields,
opaque `void` pointers, and acyclic nested `@repr(C)` records; generated definitions are ordered by
dependency and guarded by target-derived nested layout assertions. One pointer-to-opaque-pointer
output-slot shape is admitted for Arena allocation. A generated FY caller now receives both an
8-byte register result and a 20-byte hidden-memory result from hand-written C in the four-way native
matrix. Raw dereference or arithmetic, arbitrary casts, nested enums in public C records,
built-in-tag fields, array
fields or foreign array signatures,
callbacks, by-value aggregate parameters, exports, and public standard-library wrappers remain
unimplemented.
The C emitter records only foreign link names used by actual calls. `experimental build` uses one
exact-symbol contract to link the minimal provisional standard-library dependency closure; unused
declarations pull in nothing, while arbitrary foreign libraries and linker arguments remain closed.
These raw calls remain closed implementation details; public `std::bytes`, `std::text`, and
`std::io` wrappers expose only their audited safe subsets.
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
safe owned-text profile products, 13 package-workspace build products including public `std::bytes`
and `std::text` development/release quickstarts, and nine package-workspace run
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
One exact generated object now adds two storage-class-105 weak `SEARCH_ALIAS` records: the missing
name selects its object-local fallback, while a strong `target.obj` definition overrides the other
fallback. Both final REL32 values and native results are checked at low and optimized profiles.
Other weak search modes and a versioned weak-object fixture remain open. This is structural and
closed link evidence, not a public object format or source-level weak declaration.
The linked-image inspector now also follows the exception data directory into `.pdata` and its
referenced unwind records. It validates sorted nonoverlapping executable `RUNTIME_FUNCTION` ranges,
DWORD-aligned file-backed unwind RVAs, version/flag/frame/prolog structure, bounded v1 unwind-code
slots, version-2 epilog descriptor bounds, and minimum handler or chained tails. The local MSVC
product currently exposes both version 1 and version 2 records. The native unwind probe separately
proves its closed exception payload, captured-context virtual walk, and handler-directed termination
path. It also binds that frame's exact raw-object handler-tail `ADDR32NB` relocation through the
MSVC map into the final PE: low uses `__GSHandlerCheck_SEH`, while optimized uses
`__C_specific_handler`. Version-2 epilog offset semantics, language-specific handler data, recursive
chains, Fairy/foreign-frame crossings, chained-tail object relocations, and a complete versioned unwind
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
`fairyc run [--release] [--locked]`, and the semantic-by-default or Windows-native
`fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` and the execution-free
`fairyc test --list [--json] [--locations] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]`, plus source-independent
`fairyc explain <diagnostic-code>` and `fairyc targets`. Compiler developers can inspect the private
target-explicit C11 bootstrap through
`fairyc experimental emit-workspace-c --target <target-id> [--locked]`. Stable run forwards caller stdin, bounded Windows
program output, arguments, and status without pre-reading input. Native tests compile each selected
entry into a separate temporary Windows image, forward values after `--` to every isolated test,
and explicitly attach caller stdin to exactly one selected test with `--stdin`, and retain exact
binary stdout/stderr in native JSON v4 with `--capture-output`; shared or replayed multi-test input,
live native-test output streaming, and input forwarding for other native commands,
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
Push-Location examples/regions
go run ../../cmd/fairyc check
Pop-Location
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
go run ../../cmd/fairyc test --native --locked
go run ../../cmd/fairyc test --native --release --locked
go run ../../cmd/fairyc experimental emit-workspace-c --target x86_64-windows-msvc --locked
go run ../../cmd/fairyc experimental emit-workspace-c --target x86_64-linux-gnu --locked
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
process, requires it to create a no-overwrite starter, check it, run its semantic and native tests, and compile
its generated source through MSVC with conventional program status zero. The same executable runs
`examples/windows_quickstart/main.fy` through MSVC with
native result 42 by iterating two bytes with checked `usize` arithmetic and safely converting each
indexed `u8` value to `i32`,
then execute `examples/windows_output/main.fy` with exact stdout and process status
42 while explicitly supplying its separate `windows_console.fy` module, and finally run the
two-package `examples/workspace_quickstart` graph through an exact generated lock with native result
42, and runs the safe `std::io` workspace with exact text and byte-view lines on separated
stdout/stderr and status zero. The same standalone
compiler runs the isolated semantic test, a real native trap followed by a passing native test, and
the checked-in native test at both MSVC profiles without changing the retained executable. It also
emits locked Windows and Linux C11 application units, then measures nine fresh-cache cold/warm pairs with one new compiler process per sample,
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

The private `fairyc experimental emit-workspace-c --target <target-id> [--locked]` command consumes that same manifest-owned
workspace and root `main::main fn() -> i32`, then writes only the selected Linux GNU or Windows MSVC
ISO C11 application translation unit to stdout. Target selection is mandatory because LP64 and
LLP64 are different contracts; the command never guesses from this host. It invokes no C compiler,
linker, SDK, WSL, subprocess, or network and creates no build artifact. The separate runtime and
standard-library support unit is intentionally absent, so redirected output is inspection/transport
input rather than a promised standalone C package. Ordinary mode maintains `fairy.lock`;
`--locked` verifies it without writing. Generated C is compiler bootstrap transport rather than a
cross-release source product; the removed stable `fairyc emit c ...` spelling returns code 2 with
the experimental migration direction.

The stable `fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` command keeps tests out of ordinary compilation. It overlays
the root package's optional lowercase `tests/` directory after the production lock is handled, or
exactly one root-reachable local member when its manifest UID is selected with `--package`;
`tests/answer.fy` therefore declares `module tests::answer;`. Mark a private safe zero-argument
`void` function with argument-free `@test`. Without `--native`, each discovered test runs in
canonical name order through a fresh verified-MIR interpreter state without a toolchain or artifact.
With `--native`, the compiler appends one private verified `fn() -> i32` harness per test, emits its
reachable closure, compiles and inspects one temporary Windows MSVC image, and executes it with EOF
stdin by default. Explicit `--stdin` requires exactly one selected test after complete compiler
admission and then attaches caller input directly without pre-reading, buffering, or replay; values
after the first `--` become that process's arguments, and `--release` selects
optimized MSVC. Repeatable `--exact` values select
canonical `<module>::<function>` identities byte-for-byte while retaining canonical report order;
repeatable `--file` instead selects exact portable workspace-relative test-source paths, including
member workspace prefixes and known files with zero tests. Unknown, duplicate, host-separated, or
case-mismatched paths fail; `--file` and `--exact` do not combine. Repeatable `--dir` recursively
selects exact admitted directory descendants using the same portable path spelling. Known directories
come only from admitted test-source ancestors, so empty filesystem directories are not selection
targets; a directory containing only zero-test sources is valid. Unknown, duplicate, host-separated,
case-mismatched, dot-segment, and trailing-slash spellings fail, and `--dir` does not combine with
`--file` or `--exact`. The complete selected-package test overlay is still compiled.
`fairyc test --list [--package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]` performs that complete
compiler admission and prints the canonical identities without running the interpreter, backend,
toolchain, or test process. Listing with `--json` emits the stable `fairyc.test-list/v1` schema with
the canonical identity array, count, and lock state. Execution with `--json` emits the stable
`fairyc.test-report/v1` schema with canonical outcomes, engine/target/profile identity, totals,
aggregate status, and lock state. A trap fails an executed test and later selected
tests still run in fresh processes. `--locations` adds an exact-current source anchor to text and
selects `fairyc.test-list/v3` or `fairyc.test-report/v3` for JSON. Each anchor contains package UID,
portable workspace-relative path, 1-based line and byte column, exact name byte span, and matching
`fyfile1`/`fylm1` identities without exposing an absolute path or source text. Native JSON
`--capture-output` uses `fairyc.test-report/v4` and adds required canonical base64 stdout/stderr to
every outcome, preserving binary and pre-trap bytes while the private trap control stays off-stream.
Without that flag, native output remains rejected and legacy report bytes do not change. Shared
multi-test input, live output streaming,
portable targets, benchmarks, fuzzy/tag/glob filters, catalog-wide unreachable aggregation, and
feature-instance selection remain separate contracts.

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
map omissions. A separate bounded `WorkspaceCallIndex` now revalidates the complete caller-supplied
package set, joins exact compiler-observed calls to dependency SymbolIds, and exposes detached local
and cross-package outgoing/incoming ledgers without changing those query results. The default-free
`experimental atlas-calls list` command selects one exact package-qualified visible/full SymbolId
and direction under explicit package, call, and output limits, then emits the separate checksummed
`fairyc.workspace-call-catalog-json/v0` direct-call catalog. It adds no traversal or impact claim.
`internal/contextpack` now runs the package-local queries directly and emits caller-bounded,
checksummed canonical `fairyc.context-pack-json/v1` with exact package/compatibility/query/budget/
usage/completeness/fact/relation/omission fields and visibly empty text sections. The first
`experimental atlas-pack overview` command exposes one explicitly selected package UID with explicit
record/selection/output limits and no map-store default. Exact SymbolIds obtained from the separate
catalog command can now drive `experimental atlas-pack show` and `around` with explicit direction,
relation, record, relation, depth, selection-byte, and output-byte policy. Those commands add no
default store policy, workspace call-ledger consumption, source-excerpt output, pack decoder, or text
importer. The internal
query index now also provides exact full-SymbolId-ordered catalog pages with exact module/name/kind
filters, explicit symbol/byte limits, and full-ID-only continuation. Exact-name matches retain every
same-name declaration across modules rather than guessing one winner. Symbol Catalog v1 exposes
those pages through `experimental atlas-symbols list` as checksummed
`fairyc.symbol-catalog-json/v1`, with every filter and output limit explicit. The Windows
host smoke now also records nine raw whole-package canonical rewrite samples after planning, checks
byte identity, and reports fixed source/record/encoded-byte denominators without choosing chunking,
compression, incremental updates, or a performance budget.

The Windows host smoke now also drives that exact root `main` call catalog through nine fresh locked
standalone compiler processes after one untimed warmup. Its path-free
`fairyc.atlas-call-process-measurement/v0` report preserves raw caller wall time, whole-process peak
working-set and pagefile bytes, fixed source/call/output denominators, and the explicit construction
bounds. This is a tiny local control with uncontrolled OS caches, not a reference budget, speedup,
Atlas-only heap measurement, representative repository result, traversal decision, or impact claim.

A separate in-process scaling control generates fixed star workspaces with 2 packages and 8 calls,
8 packages and 64 calls, and 32 packages and 512 calls. After generation, loading, compilation, map
planning, and symbol discovery, it records nine raw `WorkspaceCallIndex` construction plus outgoing
catalog-encode observations per shape, including process-wide Go `TotalAlloc` and `Mallocs` deltas
and canonical output bytes. The three shapes and 64 MiB fail-closed output ceiling are harness
policy, not representative workloads, complexity evidence, retained-memory attribution, product
defaults, traversal policy, impact evidence, or regression thresholds.

The Go correctness suite separately generates complete chain, diamond, and fan-in `.fy` workspaces.
It compiles them to verified MIR, prepares exact-current maps, checks complete cross-package call
usage plus outgoing/incoming/both selections, and requires repeated canonical catalog bytes. These
synthetic non-star controls remove one structural blind spot from the star measurement, but they are
still not representative repositories, indirect-call recovery, traversal, impact, or performance
evidence.

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
- Public-interface hash v5 extraction with resource abilities and package-owned state layout identity, reusable package-set computation, fixtures, and runner: `internal/moduleinterface/`, `spec/fixtures/module-interface/manifest-v5.json`, `internal/moduleinterfacefixture/`
- Package-source digest v1 encoding, fixtures, and runner: `internal/packagecontent/`, `spec/fixtures/package-source/manifest-v1.json`, `internal/packagecontentfixture/`
- Whole-package cache-key v4, bounded memory store, current-interface/body invalidation, current v8 fixtures, and runner: `internal/packagecache/`, `spec/fixtures/package-cache/manifest-v8.json`, `internal/packagecachefixture/`; fixture manifests v1 through v7 remain historical
- Exact-version verified whole-program MIR artifact codec: `internal/mircodec/`
- Strict local-exact `fairy.toml` v1 parser, fixtures, and runner: `internal/projectmanifest/`, `spec/fixtures/project-manifest/manifest-v1.json`, `internal/projectmanifestfixture/`
- Filesystem-safe local workspace loader, fixtures, and runner: `internal/workspaceloader/`, `spec/fixtures/workspace-loader/manifest-v1.json`, `internal/workspaceloaderfixture/`
- Canonical generated `fairy.lock` v1 with strict parsing, stale comparison, and atomic replacement: `internal/lockfile/`
- MIR and C backend contracts: `docs/mir/`
- Package identity, visibility, source/interface hashing, per-file integrity, stable function identity, process-local MIR slices, and bounded whole-package cache: `docs/package/package-graph-v0.md`, `docs/package/package-instance-id-v1.md`, `docs/package/package-source-digest-v1.md`, `docs/package/source-file-identities-v1.md`, current `docs/package/public-interface-hash-v3.md`, `docs/package/package-function-id-v1.md`, `docs/package/process-local-mir-artifact-v0.md`, and current `docs/package/whole-package-cache-v2.md`; v1/v2 interface and v1 cache documents remain historical
- Manifest and lockfile readiness audit: `docs/package/manifest-lockfile-readiness.md`
- Project manifest v1: `docs/package/project-manifest-v1.md`
- Local workspace loader v1: `docs/package/workspace-loader-v1.md`
- Generated lockfile v1: `docs/package/lockfile-v1.md`
- Target and C ABI: `docs/abi/c-abi-v0.md`
- Runtime and standard library: `docs/runtime/runtime-stdlib-v0.md`
- Compatibility and versioning: `docs/compatibility/versioning-v0.md`
- Diagnostic catalog: `docs/diagnostics/catalog-v0.md`
- Atlas semantic snapshot and documentation sidecar, provisional SymbolId, logical planning, canonical transport, contained current-source-validated store, bounded package-local queries, exact workspace-call catalog, canonical compiler-fact Context Packs, canonical exact retained-source/documentation attributed-text output, single-symbol Evidence Bundle v0, frontend-owned exact `///` attachment, exact catalog plus overview/show/around/calls/text/evidence CLI, whole-package rewrite measurement, fresh-process workspace-call control, and fixed-shape join-and-encode scaling control: `docs/atlas/`, `internal/lexer/documentation.go`, `internal/semanticsnapshot/`, `internal/atlasmap/`, `internal/workspacecallcatalog/`, `internal/contextpack/`, `internal/attributedtext/`, `internal/evidencebundle/`, `internal/workspaceloader/exact_source.go`, `internal/cli/atlas_pack*.go`, `internal/cli/atlas_calls.go`, `internal/cli/atlas_text.go`, `internal/cli/atlas_evidence.go`, `internal/buildmeasure/atlas_rewrite.go`, `internal/buildmeasure/atlas_call_process.go`, `internal/buildmeasure/atlas_call_scale.go`
- CLI contract: `docs/cli/command-contract.md`
- Workspace ownership: `docs/monorepo/package-ownership.md`

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not committed
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
