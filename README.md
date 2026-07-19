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
  fixture-backed scalar, safe-borrow, and scalar foreign-call subset. Its semantic runner executes all 23 v0 resolve, type,
and local Flow cases, including last-use loans and borrow-return origins. Whole-root resource Flow
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
  interpreter slice executes all 56 v0 cases for literal return, function-call arithmetic,
boolean branches and logical not, source `&&` and `||` short-circuiting, scalar `while`, scalar
`Option` construction/matching, scalar `Result` construction/matching/propagation,
resource construction, field reads, call moves, resource-bearing `Option` and `Result` payload moves,
resource returns and propagation, ordered reassignment, consuming-close success/failure,
nested active-variant cleanup, deterministic drop traces,
  normal lexical Region placement plus direct-return, Result-propagation, nested, break, and continue reset traces,
division/remainder zero and signed-overflow, ordinary integer-overflow, and invalid-shift traps, and bounded step-limit
termination. Fixture-only, manifest-authorized byte sinks additionally prove exact in-memory
stdout/stderr bytes, cross-stream order, and a shared output limit without granting host I/O.
Resource declarations now retain ordered, resolved fields and drop-body classification from AST
through typed HIR and executable MIR. The current executable subset accepts only an empty user
`drop` body; any nonempty body fails explicitly before Flow or MIR instead of being silently ignored.
The interpreter constructs, reads, and recursively drops resources from the verified MIR definition,
but still does not execute user drop-body effects.
The constrained C backend emits deterministic C11 for 18 verified fixtures across the scalar
payload surface, restricted inert user enums, flat nonempty all-`i32` resources with empty drop
bodies, and resource-bearing built-in `Option` and `Result` values. The cases cover
resource construction, field reads, whole-root call moves, live reassignment, calls and control
flow, active-payload moves, Result propagation, bitwise operations, checked division and shift, overflow
trapping, scalar `Option` Some and None, and scalar `Result` propagation success and failure. This
fixture lane only produces C text; no C compiler, linker, native executable, complete runtime, target
conformance, or self-hosting is implied.
The first executable `fairyc.target-spec/v0` description now owns the host-independent
`x86_64-linux-gnu` identity, LP64 scalar and pointer layouts, SysV AMD64 calling and stack facts,
ELF/DWARF identities, and fail-closed maturity evidence. It deliberately contains no local
compiler, SDK, or sysroot path and still reports `emittable`, `conformant`, and `supported` as
false.
The host-independent layout engine now executes all 16 target-layout v0 cases for language and C
scalars, data and function pointers, explicitly based C enums, fixed arrays, nested structs, field
offsets, internal and tail padding, and fail-closed malformed or overflowing inputs. These are
target-mechanics fixtures, not source-language aggregate support or native C ABI proof.
A host-independent SysV AMD64 classifier now executes 17 versioned C ABI plan cases over the same
type and layout vocabulary, covering INTEGER/SSE/MEMORY classes, mixed and homogeneous returns,
hidden result pointers, register exhaustion, and whole-aggregate rollback. Those plans are not yet
native evidence by themselves, and the target's C ABI marker stays fail-closed.
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
source-to-native use without pretending that a Windows TargetSpec, ABI, distributable artifact, or
self-hosted compiler exists.
A provisional `x86_64-linux-gnu` SystemHeap implementation now lives in the target standard-library
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
public `std::alloc` API remain absent.
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
typed locals can expose only immutable data through explicit unsafe conversion. Indexing, slicing,
source-level length access, a stable public string ABI, and the general safe text API still do not exist.
Buffer-backed String and CString C helpers now preserve ownership on failed UTF-8 conversion,
support overlap-safe String append, expose immutable bounded views, reject interior NUL, and append
exactly one C terminator. Their C-only native probe is configured but unobserved. A separate
generated FY caller now crosses the exact CStr view record into the real bounded validator for valid,
interior-NUL, and unterminated storage while checking failure leaves the prior view unchanged. A
second generated FY caller uses exact nested heap/buffer/String/CString/result records to call the
real String copy, view, append, and drop helpers plus CString creation, CStr view, and drop. Its C
observers require exact bytes, repeated append, one terminator, and normalized release. These are raw
internal transports; public safe String/CString types and source `Result` mapping remain absent.
The Linux File C slice now exposes partial I/O, write-all progress, sync, and consuming close that
is never retried, even after failure. Its C-only four-way probe remains configured but unobserved.
A separate generated FY caller now crosses exact result, File, and open-option records into the real
open, write-all, sync, consuming-close, reopen, partial-read, EOF, and final-close helpers. It also
requires a repeated close to report the closed status and both completed closes to normalize the
descriptor record. This is raw internal transport; safe File ownership, source effects, typed
errors, and hosted execution evidence remain absent. Product builds now select the Linux File C
unit only when one of the five admitted calls is actually present.
The source FFI slice accepts `i32`, `u8`, `u32`, `usize`, all sixteen exact transport-only TargetSpec
C scalars, admitted raw-pointer parameters or results, and nonempty acyclic `@repr(C)` struct
results. A declaration may keep its Fairy name or use one nonempty `@link_name("external_symbol")`;
AST, typed HIR, MIR, prototypes, and calls preserve the source/link identity split.
Calls require `unsafe {}` or `unsafe fn`; missing unsafe provenance is `E5101` and safe references in
raw declarations are `E5102`. The interpreter fails closed with a `foreign-call` capability error,
while C emission produces a plain external prototype with no inferred alias, purity, retention, or
unwind attributes. Typed `null` crosses a closed pointer/target-scalar native probe. An exact
safe-borrow-to-raw-pointer cast now lowers only for a direct local place inside `unsafe`, allowing a
nonempty `@repr(C)` record to cross by pointer and receive standard-library output.
Those C scalar types remain distinct from coincidentally represented Fairy scalars, emit their exact
C spellings with target and provider assertions, and currently have no source literal, operator,
cast, raw-pointer, or interpreter support. Nonempty `@repr(C)` records may use exact C-scalar fields,
opaque `void` pointers, and acyclic nested `@repr(C)` records; generated definitions are ordered by
dependency and guarded by target-derived nested layout assertions. One pointer-to-opaque-pointer
output-slot shape is admitted for Arena allocation. A generated FY caller now receives both an
8-byte register result and a 20-byte hidden-memory result from hand-written C in the four-way native
matrix. Raw dereference or arithmetic, arbitrary casts, nested enum or built-in-tag fields, arrays,
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
inspect the declared target specification, or interpret one source-root-relative `.fy` file, while semantic checking also
accepts an explicit module set. There is no stable CLI, user drop-body execution, native execution
command, complete runtime, standard library, package manager, supported target, release-stable CLI, or stable language
revision yet.

## Try the current prototype

The following commands are manual developer examples, not agent command authority:

```text
go run ./cmd/fairyc explain E1101
go run ./cmd/fairyc targets
go run ./cmd/fairyc experimental run-host examples/windows_smoke/main.fy --profile optimized
go run ./cmd/fairyc experimental build-host examples/windows_smoke/main.fy --profile optimized --output build/fairyc-windows-smoke.exe
go run ./cmd/fairyc experimental check-syntax examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/semantic/main.fy examples/semantic/math.fy
go run ./cmd/fairyc experimental check-semantics examples/resources/main.fy
go run ./cmd/fairyc experimental check-semantics examples/regions/main.fy
go run ./cmd/fairyc experimental check-semantics examples/results/main.fy
go run ./cmd/fairyc experimental target-spec x86_64-linux-gnu
go run ./cmd/fairyc experimental emit-c examples/c/main.fy
go run ./cmd/fairyc experimental emit-c examples/c/tagged.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource_tagged.fy
go run ./cmd/fairyc experimental build examples/c/tagged.fy --cc clang --profile optimized --output build/fairyc-example
go run ./cmd/fairyc experimental run examples/run/main.fy
go run ./cmd/fairyc experimental run examples/run/resource_option.fy
go run ./cmd/fairyc experimental parse examples/syntax/main.fy
```

`explain` and `targets` do not read the current directory or source tree. The former marks its
diagnostic catalog as provisional; the latter reports declared TargetSpec evidence without probing
local MSVC, SDK, WSL, or any other toolchain. The experimental source commands treat the current directory as the temporary source
root. `parse` emits versioned,
deterministic JSON containing token and CST kinds and byte spans, but not source text, token text,
literal values, or absolute paths. `check-semantics` accepts an explicit bounded list of source files
for the currently implemented scalar, safe-borrow, whole-resource, Region-parameter, and Result subset. It rejects duplicate paths, root
escapes, oversized input sets, and unsupported syntax rather than claiming success. The official
`experimental run` accepts one file whose declared module contains `main() -> i32`, executes its
verified MIR in the bounded interpreter, including boolean `if`, short-circuit control, scalar `while`,
resource construction, field reads, whole-root call moves, resource-bearing built-in tagged values,
ordered resource reassignment, consuming-close ownership patterns, cleanup drop events,
  and Region fallthrough, direct-return, Result-propagation, nested, break, and continue reset paths,
checked addition, subtraction, multiplication, negation, and signed right shift, and reports either `result: <i32>` or a canonical
trap. The official
`fairyc check` command remains unavailable until effects and the
remaining semantic groups exist.

`experimental target-spec` accepts exactly the known `x86_64-linux-gnu` ID and writes deterministic
`fairyc.target-spec/v0` JSON. The result separates target semantics from local toolchain discovery
and lists the missing evidence that keeps all three target maturity claims false.

`experimental emit-c` accepts the same single-file entry shape and prints the verified scalar plus
flat all-`i32` resource and resource-bearing built-in tag C11 translation unit directly to stdout.
Every used C scalar receives target size/alignment assertions; signed and unsigned integer families
also receive signedness assertions. Target `c::ssize_t` alone adds the declared POSIX.1-2008 feature
macro and provider header. The command does not write a `.c` file or invoke a native toolchain.

On Linux/amd64 with glibc and the explicitly selected Clang or GCC installed, `experimental build`
uses that same verified unit and the embedded runtime to produce a no-overwrite executable
below the ignored `build/` directory. Compiler, `low` or `optimized` profile, and output path are
mandatory. This Windows host cannot execute that command, and the command remains narrower than the
interpreter: scalar tags, direct flat nonempty all-`i32` resources with empty drop bodies, and
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
- Target layout engine and fixtures: `internal/layout/`, `spec/fixtures/layout/manifest-v0.json`
- Resolved package graph fixtures: `spec/fixtures/package/manifest-v0.json`
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
