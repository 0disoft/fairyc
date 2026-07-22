# Compilation and Runtime Flow

- Status: Draft
- Architecture decision: `docs/adr/0001-initial-architecture-boundaries.md`
- MIR contract: `docs/mir/mir-v0.md`

## Frontend and analysis pipeline

```text
source bytes
  -> tokens and trivia
  -> compact syntax
  -> module and name resolution
  -> type and constant checking
  -> typed HIR
  -> target-independent layout needs and target selection
  -> left-to-right expression normalization
  -> analysis-only Flow CFG
  -> move, borrow-origin, region, result, effect, and drop analysis
  -> cleanup and partial-initialization elaboration
  -> mutable-local MIR
  -> MIR verifier
```

`internal/compiler` owns the executable handoff across the final semantic, MIR-lowering, and MIR-
verification stages. It accepts either one explicit single-package file closure or one already-
resolved package input. Ordinary source diagnostics remain semantic results, while invalid resolved
input, lowering failure, and verifier rejection retain distinct compiler-failure stages. A consumer
receives executable MIR only after the shared coordinator marks it verified.

The experimental product commands and all source-driven MIR, interpreter, C-emission, and native-
probe fixture producers enter that boundary through `compiler.Check` or `compiler.CheckPackages`.
Verifier-mutation fixtures call `mir.Verify` directly because rejection is their oracle, while the
interpreter and C backend re-verify caller-provided MIR as a downstream defense.

Parsing consumes original UTF-8 source bytes, preserves trivia and invalid-input provenance in compact lossless syntax, and recovers enough structure for deterministic diagnostics without accepting malformed programs as valid. Canonical spans remain half-open byte ranges. Only an error-free batch parse lowers to AST and HIR. Type checking operates on Fairy C target types, not Go host types. Arbitrary-size integer and exact-decimal literals remain exact until their target type and overflow behavior are known.

In a resolved multi-package compilation, every semantic and MIR span carries the build-local package
discriminator separately from its original relative source path. Sorting, compaction, unsafe
provenance, and source-linked instrumentation use both fields. Source-visible paths never contain
the private `$pN` qualification used by compiler entities.

Semantic acceptance is split into resolver, type, and Flow phases. The versioned semantic fixtures assert declaration coordinates, canonical types, borrow origins, and complete independent diagnostic sets without freezing internal arena IDs or whole-HIR serialization.

An immutable green tree and incremental reparse are deferred. The parser boundary preserves trivia so those features can be added without making them a prerequisite for executable semantics.

## Semantic lowering

Typed HIR preserves source concepts. Resource definitions retain canonical identity, ordered resolved
fields, source spans, and drop-body classification instead of being reconstructed from use sites.
Flow CFG computes fixed-point state without becoming a backend contract. MIR removes source
conveniences and records their behavior explicitly.

`?` becomes a discriminant branch. Resource state becomes moves, flags where genuinely needed, and cleanup blocks. Resource construction and field reads refer to a verified MIR resource definition; a nonempty drop body becomes a compiler-private typed function linked by symbol and exact resource type. Aggregate construction carries field progress. Calls receive already ordered argument temporaries. Bounds, arithmetic, shift, and assertion failures become explicit checked branches and traps.

Cleanup chains are created directly per normal exiting edge, verified, then identical suffixes may be merged. Traps have no cleanup edge. Calls have no unwind edge in Stage 0.

## Execution and backend flow

```text
semantic oracle:       verified MIR -> MIR interpreter
Stage 0 product path:  verified MIR -> constrained C -> TargetSpec + resolved ToolchainSpec -> object -> C-driver link
later fast path:       verified MIR -> measured fast backend -> object -> linker
later optimized path:  verified/optimized MIR -> LLVM IR -> object -> linker
```

Package-aware workspace checks always perform loading and semantic analysis. Memory-cache mode can
reuse process-bound package MIR inside one Session. Explicit persistent mode instead keys stable
package MIR by the exact compiler build and complete compatibility vector, accepts only complete
verified disk hits, rebuilds fresh MIR identities, and reruns the ordinary verifier before the
interpreter or backend can observe the program. Any cache failure returns to ordinary lowering.

The interpreter, `x86_64-linux-gnu` C path, and first provisional
`x86_64-windows-msvc` scalar/LLP64 C-emission path are committed. Later target and backend
expansions remain separate capability decisions.
The interpreter CLI and Windows target CLIs accept the same bounded explicit source closure: one
entry module plus repeated source files, with no directory, manifest, or import-driven discovery.

Generated C uses a declared dialect, compiler family, target profile, statements, ordered temporaries, and helpers to preserve evaluation order, overflow, division, shifts, bit casts, aliasing, and trap semantics. It is a constrained reference transport, not universally portable C or the Fairy C semantic model.

`internal/target` is the host-independent source for declared target IDs, scalar and pointer
layouts, per-profile calling-convention identities, stack facts, object/debug/unwind contracts, and
maturity evidence.
The C driver adds local compiler, libc, and path resolution only after selecting that target. A
target can therefore be inspected on Windows without borrowing Windows layout or claiming that the
Linux toolchain is locally executable.
`Stage0WindowsMSVCProfile` is the first backend consumer of the Windows description. It shares the
semantic C emitter but selects Windows scalar providers and layout facts, identifies its output as
`x86_64-windows-msvc`, rejects unavailable POSIX `c::ssize_t`, and leaves validation-only drop
instrumentation unavailable. `experimental build-target` and `experimental run-target` resolve MSVC only after that profile has
been selected. Its target-specific support resolver admits only exact project-owned SystemHeap
allocate/deallocate, Buffer initialize/zeroed-resize/drop, Arena initialize/Region begin/zeroed
allocation/copied placement/reset/Arena drop, CStr validation, String copy/view/append/drop, CString
from-text/view/drop, the five opaque String/CString storage bridges, and
`fy_std_text_utf8_valid_v0` calls. Buffer and Arena selection each
close over SystemHeap in deterministic component order; the resolver combines SystemHeap, Buffer,
Arena, text-view, owned-text, and scalar runtime at most once each. It also owns independent
target-bound Windows console and Windows File components; the File slice is selected only by one of
its six admitted raw calls and stores the handle inline without a hidden allocation. The ADR 0124
example module contains the unsafe boundary and exposes effect-checked move-only FY wrappers over
only those calls; it does not widen component selection or linker input. Every other
foreign or unproven standard-library symbol is rejected before toolchain discovery. The Windows SystemHeap source overrides only its fundamental-alignment
ceiling with the x64 `malloc` guarantee of 16 bytes; allocation semantics stay shared. Companion
inputs and arbitrary linker arguments remain absent.
The three target commands share one compilation function. Its frontend accepts one entry plus a
bounded explicit repeated-source set, analyzes all listed modules together, and performs no ambient
source discovery. The commands validate the resulting PE and Microsoft x64
unwind structure before retaining or executing it. `build-target` promotes bytes to one explicit
no-overwrite output; `run-target` executes an independently temporary image, rejects output, and
reports either exact signed `i32` process bits or a canonical product trap. Neither path uses WSL.
The Windows execution driver captures stdout and stderr into separate byte sequences under one
shared 64 KiB ceiling. It preserves stream identity but makes no ordering claim between the two OS
pipes. `run-host` and `run-target` still require both streams to be empty. `exec-target` forwards the
separated bounded streams without a compiler receipt and preserves the program status; the current
source surface reaches only one unsafe Windows stdout write-all function. The checked-in program
contains it behind a safe local `ref str` helper; `main` uses only safe borrowed-view calls while the
private generated-C descriptors carry data and length between internal functions.
`internal/layout` is the next boundary: it turns those facts into checked array and aggregate size,
alignment, offset, and padding results before any backend chooses C syntax. Separate Linux GNU and
Windows MSVC manifests hold host-independent expectations. The generated native C probe checks
every valid Linux case through its configured runner. A separate Windows generator turns all eight
Windows pass cases into MSVC C11 assertions; the configured host smoke compiles and executes them at
low and optimized profiles. Neither local/configured path is a hosted receipt by itself.
`internal/cabi` then selects a target algorithm over the same type and layout facts. Linux produces
SysV AMD64 eightbyte/register plans; Windows produces Microsoft x64 ordinal-slot, direct-small-
aggregate, indirect-aggregate, and hidden-result plans. The Linux runner turns its validated pass plans into separate Clang/GCC
caller and callee objects in both directions. Generated Fairy callers now cross scalar, pointer,
register-result, and hidden-memory-result boundaries; an observed hosted result remains a later
evidence layer. The Windows host smoke turns all twelve validated pass plans into separate caller
and callee C11 translation units and crosses them through local x64 MSVC at both profiles. That
same-compiler probe remains below source-level Fairy FFI, independent ABI inspection, and hosted
evidence.
`internal/pecoff` is the next evidence boundary after local linking. It parses executable bytes
without using the host loader and rejects non-AMD64, non-PE32+, DLL, non-console, misaligned,
out-of-range, duplicate-section, missing-`.text`, and non-executable-entry images. The Windows smoke
applies it to the retained target-neutral product, both declared-target profile products, two direct
run-target temporary products (one driven through a separately built compiler process), both safe
File and safe owned-text profile products, and both profiles of the raw Windows File, raw Buffer,
raw owned-text, raw Arena, layout, ABI, and unwind probes before
execution. COFF object records and unwind/debug payload semantics remain outside this linked-image
branch.
The same package has a distinct raw-COFF path used inside `internal/cdriver` before local linking
results are returned. It admits MSVC `.text$...` subsections and storage-free uninitialized `.bss`,
but requires bounded section and relocation ranges, valid relocation symbol indexes, readable
executable code, and named symbols. Object bytes survive only in the in-memory build receipt; the
temporary source, object, and executable root is still removed. Exact symbol and relocation-type
contracts remain a later versioned fixture boundary.
The linked-image branch additionally walks the Microsoft x64 exception directory. It maps sorted
`.pdata` runtime-function entries to executable code and readable unwind data, validates version 1
prolog-code slot structure and bounded version 2 epilog descriptors, and checks handler or chained
tails without interpreting language-specific payloads. This remains static structure inspection;
the Windows loader and unwinder are not smuggled into that parser's structural oracle.
A distinct `internal/windowsunwindprobe` execution layer then uses those platform services
explicitly. Three separately compiled C units raise a private noncontinuable exception, inspect its
payload and captured `CONTEXT` in an outer filter, virtual-walk the raiser, termination frame, and
catcher, and require an abnormal `__finally` execution during the real handler-directed unwind.
This layer proves only those closed native C frames. It does not change Stage 0's non-unwinding MIR,
trap, cleanup, or foreign-call rules.

`internal/cstdlib` is a separate target-standard-library transport boundary. Its first provisional
SystemHeap helper normalizes empty allocation, rejects unsupported alignment, and preserves the old
block when copy-and-release resize fails. A closed C probe reaches the existing Linux compiler
matrix. Internal generated Fairy transports call the real allocate/deallocate symbols with exact
C-layout blocks on Linux and a private layout-equivalent `usize` block on Windows. The Windows path
uses the target's 16-byte x64 `malloc` alignment ceiling and executes at both MSVC profiles.
A distinct raw Buffer caller links against an instrumented target-specific dependency closure at
both profiles. It forces construction plus reserve, self-overlap append, and zeroed-resize growth
failures, requires exact storage preservation, observes recovery bytes and zero-fill, and requires
drop to return the live-allocation count to zero. Product and generated-Fairy source paths continue
to use the uninstrumented Buffer closure.
The same Windows target example now reaches the real Arena source through private self-linked node,
Arena, Region, and `void **` transport. It observes initialization, active attachment, a non-null
allocation/node, reset invalidation, and drop generation transitions at both profiles without
dereferencing the returned storage. This is lifecycle and dependency-closure evidence, not safe
lexical Region lowering, zeroed-byte observation, a public representation, or a public allocator API.
A distinct raw Arena caller links against an instrumented target-specific dependency closure at both
MSVC profiles. It observes zeroed and copied payloads, forces failure at the payload and tracking-node
allocation boundaries, requires rollback to zero live allocations, and verifies stale Region and
active-drop rejection. The adapter is prepended only to probe implementations; product resolution
continues to select the ordinary SystemHeap source.
The Windows target example also crosses compiler-owned static UTF-8 and C-string storage into the
real owned-text source. Private nested heap/Buffer/String/CString/view/result records carry String
copy, view, append, second view, and drop plus CString construction, view, drop, and direct CStr
validation. Status, length, alignment, pointer, and normalized-drop state execute at both profiles;
the FY subset cannot dereference the returned view. A separate raw C caller now crosses the
target-specific owned-text callee at both MSVC profiles and supplies byte, terminator, null/malformed
storage, failure-preservation, self-overlap, size-overflow, and repeated-drop evidence. A probe-only
adapter overrides the SystemHeap allocation macros in that callee, forces initial and append
reallocation failures, and observes live allocations returning to zero. The ordinary target support
source retains CRT allocation and no probe controls. This does not add safe source ownership or a
public text API.
An independent checked-in safe owned-text module now keeps those raw declarations and opaque casts
behind move-only FY resources. Its storage bridges use `memcpy` over `void *`, and low/optimized
source builds execute Result mapping, length reads, interior-NUL rejection, and verified cleanup.
It still cannot reconstruct safe borrowed text views and therefore remains an example boundary.
`internal/cstdlibcontract` owns the exact-symbol/component maps used by backend admission and
target-specific product linking. The emitted C unit records only actually called foreign names;
`experimental build` and `build-target` link only the permitted minimal dependency-closed C source
plus the scalar runtime. No public standard-library API exists. This does not
broaden the thin runtime or complete the standard library.

The implemented slice emits deterministic C11 for verified scalar-payload MIR, including
private `Option<i32>` and `Result<i32, i32>` transport, copy-only built-in tags over canonical Fairy
scalars or admitted inert structs, plus nonempty resource definitions over Fairy copy scalars or
  raw pointers with empty or pure nonempty drop bodies and resource-bearing built-in tags whose other payloads are
supported copy values or another
direct admitted resource. Resource and tagged layout comes from the declared target engine and is
rechecked with C11 size, alignment, and field-offset assertions. `?` error paths rebuild the
enclosing function's declared Result type instead of returning a differently typed operand
container. A separate
Linux/amd64 runner now drives Clang and GCC at low and optimized profiles, links the embedded scalar
runtime, executes each emitted fixture, compares its process outcome and source-linked drop trace,
and separately runs every production unit under a closed ASan+UBSan mode. The same bounded C driver
and runtime feed an explicit experimental source-closure build command whose artifact is confined to
the ignored source-root `build/` directory. One product smoke remains scalar/resource-only; a second
requires the ordinary build path to select and link the real text-view C component from an actual
Fairy call. The current Windows host cannot execute either native path, and no hosted run has been
observed yet. Complete runtime, public ABI, and supported-target gates remain unimplemented.

## Atlas flow

```text
typed HIR + verified MIR
  -> immutable SemanticSnapshot
     + exact file content/line-map identities
  -> provisional SymbolId derivation
  -> immutable per-package logical map plan
  -> bounded canonical Atlas transport encoder/decoder
  -> checksummed .fymap bytes
  -> contained immutable publication + current-source validation
  -> bounded overview/show/around queries
  -> optional canonical compiler-fact Context Pack
```

The compiler frontend cannot import Atlas. Atlas does not reparse source to invent semantic facts. Its failure does not change diagnostics or build results.

`internal/semanticsnapshot` implements the first arrow. It re-verifies MIR, consumes exact package
metadata, removes build-local identities, deep-detaches its records, and reports relation-specific
completeness. The validated workspace supplies exact source-file content and line-map identities;
snapshot construction requires a file/module bijection without retaining source bytes.
`internal/symbolidentity` implements the next arrow from stable semantic coordinates and canonical
signatures while retaining full collision evidence. `internal/atlasmap` then validates the complete
workspace package/interface set and assigns every snapshot file, module, declaration, SymbolId, and
function summary to one immutable package partition. Its target-independent header records all
independent identity algorithms, the exact compiler executable identity, package compatibility, and
explicit relation omissions. The same package now implements the caller-bounded canonical JSON
transport and domain-separated checksum, then strictly decodes, revalidates, and rederives SymbolIds
before returning immutable facts. `MapStore` now publishes content-addressed generations beneath an
opened root, applies caller retention bounds, and returns only one exact compiler/interface/current-
source match. The package-local query index repeats that current-state check and implements bounded
`overview`, exact-SymbolId `show`, and breadth-first `around` without source text or dependency
discovery. `internal/contextpack` runs those queries directly and emits a separately byte-bounded,
checksummed canonical compiler-fact envelope with empty documentation and source-excerpt sections.
Dependency-map loading, commands, source excerpts, text import, and pack decoding remain
unimplemented and cannot feed back into these boundaries.

## Runtime behavior

The executable contains only the selected Fairy C runtime and linked platform libraries. The Go runtime belongs to the compiler process and is never linked into compiled Fairy C programs.

Safe-code bounds, null, division, shift, and overflow behavior remains stable across profiles. Proven checks may be removed. Unproven checks remain unless source enters an explicit unsafe or unchecked operation. A trap terminates without unwinding or cleanup.

## Incremental build flow

Early versions cache whole-package source fingerprints, interfaces, typed HIR, generic-free MIR, and target objects as their implementations become stable. Public interface and private implementation identities are kept separate. The first executable cache boundary now provides a full compatibility-vector key and a caller-bounded process-local LRU store for opaque whole-package records. It does not yet persist across compiler processes. Persistent storage, default numeric limits, function-level reuse, packed storage, and remote caching require measurements and an explicit adoption decision.

## Failure and recovery

- Frontend errors produce diagnostics and no executable.
- MIR verifier failures are compiler defects and never reach a backend.
- Backend or linker errors identify the boundary and preserve intermediates only when requested.
- Cache entries are disposable and cannot be the only copy of source or specification data.
- Differential fixtures compare interpreter and C output, status, trap category, and drop trace.
- Drop comparison uses a closed native-conformance profile with deterministic source/type/span site
  IDs. Product C emission and product-built artifacts never contain the observer or its binary
  stderr records.
- The C reference backend remains the recovery path for a failed experimental native backend.
