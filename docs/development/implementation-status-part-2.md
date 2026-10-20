# Implementation Status - Part 2

Continues from [part 1](implementation-status-part-1.md); index at [implementation-status.md](implementation-status.md).

## Current phase (part 2 of 2)

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
ADR 0408 advances the SDK to `0.0.0-stage0.69` with stack-only UTF-8 text paths for cross-target file creation and reading.
ADR 0409 advances language to `fairyc-language/20` and SDK to `0.0.0-stage0.70` so a terminal `assert(false);` Result or Option case needs no fabricated value.
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
[`std_vec_quickstart`](../../examples/std_vec_quickstart/README.md). ADR 0393 adds
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
`fairyc check [--locked]`, the Windows/amd64-local
`fairyc build [--release] [--locked]` and
`fairyc run [--release] [--locked]`, the semantic-by-default or Windows-native
`fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]`, and the execution-free
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
