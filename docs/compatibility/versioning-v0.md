# Compatibility and Versioning v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0008-versioned-contract-boundaries.md`

## Compatibility vector

Build receipts, caches, emitted objects, diagnostics, and maps record the applicable members of this vector:

```text
compiler release and full build identity
language revision
internal MIR version
target schema
ABI profile
diagnostic catalog
diagnostic JSON schema
runtime ABI
standard-library version and package instance
manifest schema
lockfile schema
.fymap schema
SymbolId algorithm
interface-hash algorithm
```

The prototype now has concrete internal and manifest-family values documented below, including
`fairyc.manifest/v1`; there is still no released compiler compatibility promise. Examples such as
`L0.n`, `M0.n`, `RT0.n`, and `SID1` remain format illustrations rather than release claims.

## Compiler and language

The compiler release uses SemVer. A patch may fix crashes, performance, diagnostics wording, or implementation behavior that contradicted the current specification. It cannot intentionally add syntax, change overflow or ownership semantics, or make a valid program invalid without a language revision change.

Pre-1 language revisions are exact pins. A compiler may implement several revisions, but a newer pre-1 revision does not imply source compatibility.

Grammar spelling, keyword status, literal cooking, precedence, module-path identity, recovery acceptance, resolution, canonical typing, resource and Region Flow rejection, Result handling, declared effects, call-effect acceptance, and whether a diagnostic blocks HIR are language-revision semantics. Parser-, semantic-, resource-, Region-, and Result-fixture schemas version their expectation-file shapes; they do not make two different language revisions source-compatible.

ADR 0136's C-shaped surface is therefore a new pre-1 language revision, not a compiler patch. The
product compiler cuts over atomically and does not accept both source spellings, infer a source
revision, or rewrite source during compilation. A separate lossless-CST migrator may support only
the immediately preceding revision and must fail without modifying a file when it cannot preserve
that file completely. Its command spelling is not a public contract until implemented and declared
by the command contract. Creating the design record alone changes no released compiler or language
revision; the implemented source cutover requires a language-revision increment and at least a
compiler minor release. C ABI, runtime ABI, target schema, and MIR version change only if their own
observable contracts change.

ADR 0216 cut the exact pin from `fairyc-language/1` to `fairyc-language/2` because it added
ten compound-assignment tokens and statement forms. Valid `/1` source syntax remains valid `/2`
syntax, but exact manifests, SDK requirements, locks, and cache inputs still change pins; the
standalone previous-revision parser at that cutover accepted `/1` and rejected compound assignment. Runtime ABI,
target schema, C dialect, MIR version, diagnostic catalog, SDK version, and SDK source bytes remain
unchanged.

ADR 0217 cut the exact pin from `fairyc-language/2` to `fairyc-language/3` because it
reserves `defer` and adds a statement form with explicit cleanup-edge behavior. Valid `/2` source
remains valid `/3`; the previous-revision parser rejects `defer`, while source migration remains
byte-identical and manifests require a manual exact-pin update. Runtime ABI, target schemas, C11
transport, MIR version, diagnostic catalog, SDK version, and SDK source bytes remain unchanged.

ADR 0219 cut the exact pin from `fairyc-language/3` to `fairyc-language/4` because the
previously unsupported `@must_use` callable attribute gains semantic rejection behavior. Valid `/3`
source remains valid `/4`; the previous parser already preserves generic attribute syntax, while
the `/3` semantic contract rejects this attribute as unsupported. Migration therefore preserves
source bytes and exact manifests require a manual pin update. Public callable must-use state changes
the current public-interface algorithm from v2 to v3. Runtime ABI, target schemas, C11 transport,
MIR version, SDK version, and SDK source bytes remain unchanged.

ADR 0220 cut the exact pin from `fairyc-language/4` to `fairyc-language/5` because module
constants gain semantic values, `@const_eval` gains callable meaning, and contextual module-item
`static_assert` can reject source. Valid `/4` source without those forms remains valid `/5`.
Migration preserves existing constant and generic-attribute bytes and requires a manual pin update;
`static_assert` has no `/4` rewrite. Public-interface v3, runtime ABI, target schemas, C11 transport,
MIR version, SDK version, and SDK source bytes remain unchanged.

ADR 0221 cuts the current exact pin from `fairyc-language/5` to `fairyc-language/6` because the
previously unresolved `source_location()` call gains a static `ref str` meaning and its bare name
becomes reserved across value bindings. Valid `/5` source without that name remains valid `/6`.
The previous parser already preserves the generic call spelling, so migration preserves source
bytes and exact manifests require a manual pin update. Public-interface v3, runtime ABI, target
schemas, C11 transport, MIR version, SDK version, and SDK source bytes remain unchanged.

ADR 0246 cuts the exact pin from `fairyc-language/6` to `fairyc-language/7` because
`byte_slice::from_raw_parts` and `byte_slice::from_raw_parts_mut` gain compiler-known owner-bound
meaning. Valid `/6` source that does not claim those paths remains valid `/7`; migration preserves
source bytes and requires an exact manifest pin update. Public-interface v3, runtime ABI, target
schemas, C11 transport, and MIR version remain unchanged.

For one function contract, removing a declared effect preserves every existing caller accepted by
the effect checker. Adding an effect or replacing it with another can invalidate callers and is a
source-compatibility change. Reordering or deduplicating the same set is not a semantic change
because typed HIR stores canonical lexical order; duplicate source declarations are rejected.

## MIR and cache

Internal MIR accepts an exact version only. It is mutable compiler state, not a plugin or interchange ABI. Serialized MIR and caches are regenerated from source rather than migrated. The source-based MIR fixture manifest versions repository observations separately and deliberately excludes serialized graph identity. A future stable backend exchange format, if needed, is a separate contract.

`fairyc.mir-verifier-fixtures/v0` versions named seed vocabulary and mutation recipes. It does not serialize arbitrary MIR, stabilize compiler-local IDs, authorize external graph input, or make old MIR migratable.

A cache key includes the full compiler build identity, language and MIR versions, target schema and ABI profile, toolchain fingerprint, backend and profile flags, runtime ABI, standard-library instance, source digest, dependency interface digests, feature set, and interface-hash algorithm. Incompatibility is a cache miss followed by lazy collection. ADR 0165 defines the first executable build identity as `fairyc.compiler-executable-sha256/v1:` plus the full SHA-256 of the running executable bytes; no release string, branch, or filename fallback is compatible with that identity.

`fairyc.package-cache-fixtures/v1` fixes the domain-separated full SHA-256 key under the canonical
`fycache1-sha256-...` text form. The encoding includes the PackageInstanceId, source-digest, and
public-interface encoding/algorithm names in addition to their full digest bytes. A key-version or
component-algorithm mismatch is a miss. The first bounded store is process-local and the opaque
typed-HIR, MIR, and object payloads remain exact-compiler disposable state rather than serialized
compatibility formats. The bounded persistent storage format and one exact-version experimental
command now exist; ordinary-command policy and default limits remain undecided.

The current whole-package key encoding remains `fairyc.package-cache-key/v2` with
`fycache2-sha256-...` text. `fairyc.package-cache-fixtures/v5` supplies current vectors because its
dependency entries use `fyif3-sha256` public-interface hashes and `fairyc-language/10`; fixture
manifests v1 through v4 remain historical. A component algorithm change is an intentional cache miss,
not permission to reinterpret an older key.

## Target and ABI

ISO C11 is the only normative constrained-source transport dialect under ADR 0137. The Windows
struct-result evidence lane compiles the identical C11 units under explicit C11 and C17 MSVC
consumer identities; this is not a new emission profile, target, ABI, or runtime baseline. C23
remains disabled until its external-keyword guard and validation evidence exist. Emitting C23-only syntax or changing a C
representation requires a separate accepted decision and the corresponding compatibility change.

Target schema versions the structure of target descriptions. ABI profile versions the meaning of layout, calling convention, return classification, and target-visible representation. Fixing an implementation to match an unchanged profile changes the compiler build and invalidates cache; changing the specified meaning creates a new ABI profile major.

Local SDK or compiler paths never participate in target semantic identity. Toolchain family, baseline, and capability fingerprint do.

The prototype `fairyc.target-spec/v0` object is the first executable target-schema member. Its
identity, scalar and pointer layouts, call alignment, red-zone and shadow-space rules, classifier identities, object and symbol model,
unwind/debug contract, and C-layout feature vocabulary are compatibility data. Its maturity
evidence is current capability evidence, not an ABI promise; promoting a status requires the named
fixture and hosted receipts rather than only editing the booleans. The experimental CLI schema at
`spec/cli/experimental-target-spec-v0.schema.json` must remain synchronized with the Go model.
ADR 0214 records the first exact hosted Linux and Windows native run, removes the satisfied Linux
layout/bidirectional-ABI and Linux/Windows hosted-receipt markers, and leaves every maturity boolean
false. Runner labels and observed tool versions do not become compatibility baselines.
The target entry's `emissionSubset` is compatibility data: it names what the selected backend
profile may represent, not whether the target is conformant. The experimental `build-target`
command currently admits only `x86_64-windows-msvc`, a fixed low/optimized profile choice, one
positional FY entry plus a bounded explicit repeated-source closure, and one `.exe` output. Its
command spelling and text receipt remain unstable CLI surface;
the selected target identity, scalar layout, C dialect, and emitted static assertions still must
match the target-schema and ABI-profile contracts. Adding another called source-level FFI symbol or another target
requires a separate capability and compatibility decision rather than silently widening this path.
ADR 0083 admits exactly `fy_std_text_utf8_valid_v0` as the first Windows source-call linkage. That
symbol's signature, selected component, and fail-closed rejection of every other foreign symbol are
prototype compatibility data; this does not make the complete standard library portable or stable.
ADR 0084 adds exact SystemHeap allocate/deallocate linkage and a target-only 16-byte fundamental
alignment ceiling derived from the Windows x64 `malloc` contract. The ceiling, two signatures,
record layout, minimal component closure, and normalized release observations are prototype
compatibility data; allocator resize, forced OOM, over-aligned allocation, hosted CRT deployment,
and the public `std::alloc` API remain outside the admitted slice.
ADR 0085 adds exact Buffer initialize, zeroed-resize, and drop linkage. The nested 32-byte record,
Buffer-to-SystemHeap component closure, deterministic single emission, capacity/length
observations, and normalized drop state are prototype compatibility data. ADR 0246 admits
reserve/append/truncate/clear linkage and the public source `std::bytes::Buffer` contract on both
implemented target profiles. The four-field storage remains private implementation data rather
than a Fairy native or foreign ABI.
ADR 0086 adds exact Arena initialize, Region begin, zeroed allocation, Region reset, and Arena drop
linkage. The private 32/24/32-byte node/Arena/Region layouts, target-width generation transport,
Arena-to-SystemHeap closure, single emission, non-null allocation/node state, active flags, and
generation 1/2/3 transitions are prototype compatibility data. Returned-byte observation, copied
placement, simultaneous or stale Region failures, active-drop rejection, safe lexical lowering,
stable representation, and the public `std::alloc` API remain outside the Windows slice.
ADR 0087 adds exact CStr validation, String copy/view/append/drop, and CString from-text/view/drop
linkage. Private 16-byte views, 32-byte nested owned records, 8-byte result returns, the
owned-text-to-SystemHeap/Buffer/text-view closure, deterministic single emission, length/status/
pointer observations, and normalized drops are prototype compatibility data. Independent output-
byte and terminator observation, invalid-input and allocation-failure rollback, safe ownership,
source `Result` mapping, stable layouts, and the public `std::text` API remain outside the Windows
slice.
ADR 0088 adds `experimental run-target` over the same `x86_64-windows-msvc` compilation contract.
Its required target/profile flags, absence of an output path, target-labelled native-result receipt,
canonical product-trap mapping, zero-output rule, pre-execution PE/unwind validation, and temporary-
artifact cleanup are unstable CLI and validation policy, not a stable language or target ABI. The
command adds direct local evidence without changing `emittable`, `conformant`, or `supported`.
ADR 0091 adds the unstable `experimental exec-target` process contract and exact Windows-only
`fy_std_windows_stdout_write_all_v0` symbol. Required target/profile flags, separated bounded
streams, preserved 32-bit program status, the `{ status: u32, written: usize }` transport, status
meanings 0 through 4, and target-specific admission are prototype compatibility data. They do not
stabilize a public `std::io` API, console encoding, installation, or target maturity.
ADR 0117 adds `console` to the raw declaration, checked safe wrapper, and example entry contract.
Those annotations are language-level call-acceptance data, while the helper name, boolean result,
and example module paths remain experimental and do not become a standard-library compatibility
promise.
ADR 0092 adds `.len` to safe borrows of `str` and `[u8]`. The property name, `usize` result, byte-count
meaning, metadata-only behavior, and absence of scanning, allocation, calls, and traps are pre-1
language semantics. This does not stabilize private view layout, indexing, slicing, Unicode
counting, `CStr` operations, or a public string ABI.
ADR 0104 adds `.is_empty` to the same views as the exact zero-length `bool`, preserving the same
metadata-only and no-scan/no-call contract without stabilizing private representation.
ADR 0105 adds mandatory-bound, end-exclusive `&[u8]` slicing. The result type, left-to-right
single evaluation, `start <= end <= len` rule, `bounds` trap, and borrow-origin preservation are
pre-1 semantics; omitted bounds, text slicing, and private C layout remain unstabilized.
ADR 0106 adds `assert(condition);` as an always-checked statement. Exact `bool` input,
single evaluation, profile-independent execution, and the `assertion-failed` category are pre-1
semantics; messages, debug-only checks, compile-time assertions, and unsafe assumptions remain
separate and unstabilized.
ADR 0107 adds omitted start and end spellings to immutable byte-view slicing. Zero and retained
receiver length normalization, written-bound evaluation order, the unchanged checked MIR shape,
and the `bounds` outcome are pre-1 semantics; inclusive ranges, text slicing, and private C layout
remain unstabilized.
ADR 0108 assigns `E1013` to a well-formed integer literal outside its selected implemented integer
type. Radix-independent mathematical comparison, complete-literal spans, pre-MIR rejection, and the
four Stage 0 limits are pre-1 semantics; signed-literal folding, suffixes, wider integers, and C
scalar immediates remain unstabilized.
ADR 0109 admits the exact `i32` minimum magnitude only beneath direct unary minus. Equivalent radix
spellings, normalized literal HIR, unchanged runtime negation overflow, and `E1013` for larger or
independently grouped magnitudes are pre-1 semantics; broader constant folding remains unstabilized.
ADR 0110 admits fixed arrays over five copy-scalar types, nonempty list construction, internal
value transport, and checked read-only `usize` indexing. ADR 0111 adds exact `usize` `.len` while
preserving receiver evaluation and forbidding storage-based reflection. ADR 0112 adds the
contextual direct shared borrow from `[u8; N]` to immutable `&[u8]`, retaining the local origin and
whole-array loan. Element order, length, `bounds`, and this contextual conversion are pre-1
semantics. The current `1..=4096` element admission ceiling and private C wrappers are provisional
implementation guards rather than compatibility promises; empty/repeated literals, fields,
nesting, mutation, general slicing, and ABI use remain unstabilized or excluded. Exact whole-array
safe-borrow parameters are pre-1 internal-call semantics and do not stabilize a C ABI.
ADR 0125 adds the exact contextual mutable pair from a mutable `[u8; N]` place to `&mut [u8]`,
retaining one exclusive origin and canonical length through internal calls. The private writable C
descriptor and matching `*mut u8` extraction are pre-1 transport contracts. Safe mutable indexing,
assignment, slicing, resizing, temporary borrowing, retained foreign pointers, and a public slice
ABI remain excluded or unstabilized.
ADR 0126 adds checked mutable scalar-array and `&mut [u8]` indexed assignment, mutable-view reads,
bounds-before-value evaluation order, and the private `array-index-store`/`view-index-store` MIR
forms. These are pre-1 source and internal-IR additions; mutable slicing, public slice layout, foreign
safe references, and stable MIR serialization remain absent.
ADR 0127 adds the provisional `experimental check` spelling, explicit primary-plus-source closure,
and one human verified-MIR count line. These are prototype CLI contracts only. They do not stabilize
the later package-aware command, configuration, target selection, JSON, exit codes, or
function-count meaning. ADR 0175 separately stabilizes only `fairyc check [--locked]` as a
current-workspace command, its one count-and-lock-state success line, and exit codes 0, 2, 3, 4, and
7. Adding target/config/cache/JSON flags, changing lock writes, or changing those meanings requires a
new compatibility decision.
ADR 0131 adds example-level move-only String/CString wrappers, typed source errors, and five opaque
storage bridges plus borrowed-resource field-read transport. These are pre-1 source, MIR, and
runtime-example contracts. ADR 0135 adds the two exact owner-bound text raw-parts built-ins,
their unsafe provenance, `E2105` owner-move rejection, and private C transports as pre-1 language
and internal-MIR semantics. Neither decision stabilizes `std::text`, public layouts, a foreign
safe-borrow ABI, Unicode algorithms, allocator selection, bridge symbol compatibility, or a general
user-defined raw-parts protocol.
ADR 0113 admits branch-aware safe-borrow reassignment with right-hand-side-first replacement,
loan-generation preservation across aliases, reachable `if`/`match` origin joins, and path-aware
loan liveness. Those rules and fail-closed multi-origin returns are pre-1 semantics. ADR 0114 adds compiler-internal fixed-point return-origin
summaries for exact formal borrow parameters, region parameters, and static sources. Positional
remapping, unselected call-bounded arguments, and fail-closed unknown recursive results are pre-1
semantics; the summary representation is not a public function-type or ABI promise.
ADR 0115 extends safe-borrow reassignment through `while` with canonical static loan-site IDs,
entry/fallthrough/`continue` header joins, separate `break` exits, and a zero-iteration path. Those
rules are pre-1 semantics; scalar predicate correlation and at-least-once loop proofs remain
unstabilized.
ADR 0093 admits exact `&str` and `&[u8]` parameters on ordinary internal functions. Exact source
types, non-owning borrow behavior, matching call operands, and the absence of implicit raw-pointer
conversion are pre-1 language semantics. Interpreter values and generated-C by-value descriptors
remain private implementation transport; foreign safe-borrow signatures, exported ABI, retained
borrows, and public `std::io` remain excluded.
ADR 0094 adds repeatable `--source <file.fy>` to the three unstable Windows target commands. The
positional entry rule, explicit-only closure, 64-file ceiling, per-file and aggregate limits,
duplicate/path/module rejection, and lack of discovery are unstable CLI and validation policy.
They do not define a package manifest, lockfile, module search path, dependency solver, registry,
foreign companion input, linker interface, or supported distribution contract.
ADR 0095 applies the same explicit-only closure to unstable `experimental run`. It preserves the
interpreter subset and result/trap contract while adding no backend, package, discovery, cache, I/O,
or host-process compatibility promise.
ADR 0096 applies the closure to `emit-c`, Linux `build`, and target-neutral Windows `build-host` and
`run-host`. The repeated flag admits only FY files; compiler, linker, foreign-input, target, output,
and artifact contracts remain unchanged and unstable.

`fairyc.layout-fixtures/v0` separately versions the repository input and expectation vocabulary for
the target-layout engine; each manifest binds one declared target. It is not a public
type-description format, native ABI profile, cache
artifact, or C probe result. Adding a layout family or fixture-visible failure code requires a new
fixture-contract decision; changing target sizes, alignments, field placement, or classifier
meaning additionally requires the applicable target-schema or ABI-profile decision.
Native probes derived from a manifest are versioned repository evidence consumers, not new target
identities. A local compiler pass does not promote maturity without the declared coverage set and
hosted receipt.

`fairyc.c-abi-fixtures/v0` versions fixed-signature type inputs and target-bound class, register,
stack, indirect, and hidden-return plans. The Linux manifest records SysV AMD64 eightbyte and
register-family behavior; the Windows manifest records Microsoft x64 ordinal slots and aggregate
indirection. The Windows pass subset additionally owns the inputs for the same-version local MSVC
caller/callee probe; changing that subset or its initialization/check semantics requires an
applicable fixture and probe-contract decision. It reuses the layout fixture's recursive type vocabulary
but is not a public binding format, stable native Fairy ABI, object contract, or hosted conformance
receipt. Changing class merge, exhaustion rollback, hidden-result, or register identities requires
a new fixture-contract and applicable target/ABI-profile decision.

The PE executable inspector is internal validation policy rather than a serialized compatibility
format. Tightening its admitted image structure requires synchronized synthetic rejection tests and
Windows smoke evidence, but does not create a stable PE, COFF object, debug, or unwind contract.
The raw COFF object inspector follows the same internal-policy rule. ADR 0231's
`fairyc.coff-object-fixtures/v0` manifest versions internal test inputs and exact structural
expectations for primary symbols, AMD64 relocation types, and COMDAT selection; it is not a public
object format, serialized compiler receipt, or compiler-produced whole-object byte promise. Its
4,096-section and 64 MiB object budgets bound parser work while admitting current MSVC function-level
section fanout; the synthetic 154-section regression case is evidence, not a stable output count.
`fairyc.coff-generated-object-fixtures/v0` separately versions explicit inputs and exact byte
digests for the deterministic weak-external search-mode, relocation-overflow, and version-2 bigobj test
generators. Its complete symbol receipts and relocation-run partitions are compiler-private
regression oracles. They do not stabilize Fairy C object output, Microsoft COFF as a public API, or
arbitrary objects produced outside this closed test corpus. The absence of raw stored duplicates is
intentional and does not weaken the whole-byte SHA-256 oracle.
ADR 0232's REL32 link spec and report are likewise compiler-private validation policy. Its symbol
names, `.fyreloc` receipt, preferred-image addresses, raw addend, and displacement fields may change
with synchronized inspector, probe, mutation tests, and evidence docs; they do not define a public
object format, linker API, serialized receipt, or native Fairy calling convention.
ADR 0233 applies the same rule to the bounded archive report, preferred second-linker-member index,
MSVC map parser, `.fysel` receipt, input-order variants, and archive/direct candidate values. Those
identities may evolve with synchronized probes and mutations; they do not stabilize Microsoft
archive bytes, map syntax, a Fairy library format, or a linker-selection API.
ADR 0234 likewise keeps relocation section indexes, the private ADDR64/ADDR32NB report, MSVC map
rows, `$LN...` alias handling, and `.fyreloc` receipt compiler-private. They may change with the
inspector, probe, mutations, and evidence docs and do not version the target ABI, object format,
unwind ABI, or any serialized Fairy artifact.
ADR 0235 applies that internal-policy boundary to the selected-parent associative unwind report,
direct `.pdata`/`.xdata` section relationships, map-owner check, and final runtime-function binding.
Those receipts may evolve with synchronized inspectors, probes, mutations, and evidence docs; they
do not define a public COMDAT policy, unwind artifact, linker API, or serialized Fairy contract.
ADR 0236 applies the same internal-policy boundary to the two-candidate ANY-COMDAT elimination
report, candidate-specific unwind digests, raw-section absence scan, and mixed C11/C++20 probe
inputs. None of those validation identities admits C++ as Fairy C source or versions a public
linker, object, unwind, or security-setting contract.
ADR 0237 applies it to the generated six-relocation object, marker bytes, map-owner identities, and
private REL32-variant report. These may evolve with synchronized inspector, mutation, native-smoke,
and evidence updates; they neither admit arbitrary prebuilt-object input nor stabilize an object or
linker interface.
ADR 0238 applies it to private CodeView pair reports, `/Z7` probe inputs, MSF directory parsing, and
PE/PDB GUID-age association. This validates local ephemeral evidence only; it does not version or
publish a PDB format, public debug mode, symbol store, debugger contract, or source-retention policy.
ADR 0239 keeps the system-DbgHelp function-and-line receipt equally private and ephemeral. Its exact
probe name and file basename may evolve with synchronized canary and evidence updates; it does not
stabilize a debugger API, public symbol surface, source path, PDB retention, or product debug mode.
ADR 0240 keeps the profile-specific handler names, raw tail-addend report, MSVC map owners, and final
RVA comparison equally private. Those identities may evolve with synchronized probe, inspector,
mutation, and evidence updates; they do not stabilize a Fairy exception ABI, object format, runtime
library owner, linker interface, or security-setting contract.
ADR 0241 keeps the generated weak-object names and bytes, storage class, fallback indexes, search
characteristic, map owners, and selection report private. They may evolve with synchronized
generator, inspector, mutation, and native evidence; they do not admit source-level weak
declarations, arbitrary prebuilt objects, or a public object/linker compatibility promise.
ADR 0242 likewise keeps the generated overflow-object bytes, 1,048,576-record inspection budget,
map identity, and leaf code private. They may evolve with synchronized parser, mutation, and native
evidence updates; they do not stabilize a Fairy object writer or public prebuilt-object contract.
ADR 0243 keeps the bigobj header version/GUID, generated 161-byte object, 4,096-section and
262,144-symbol inspection budgets, map identity, and internal REL32 proof private. It does not
promise public bigobj emission, arbitrary prebuilt-object compatibility, or theoretical 2^32
section capacity.
Microsoft x64 unwind inspection is likewise an internal validation policy. Neither the observed
runtime-function count nor version-1/version-2 mix is stable. Future semantic epilog, handler,
or chain fixtures require an explicit unwind-contract decision before they can become target
evidence. ADR 0081 admits one closed native exception and virtual-walk probe as internal target
evidence; its case IDs, process result, and required ordering are test contracts, not a public Fairy
C exception ABI or serialized compatibility format.

## Diagnostics

The diagnostic catalog versions rule identities. Once a code appears in a released compiler, it is not reused for another rule. Wording, related notes, and repair hints may improve without changing the code; severity or rule meaning changes require a new compatibility decision.

Machine-readable diagnostic JSON has its own schema. Stable fields are code, severity, primary span, and rule identity. Exact prose and layout are not stable machine contracts.

The prototype in-memory span carries a build-local package discriminator separately from the
relative source path. That numeric value and compiler-private qualified entity spelling are not
stable diagnostic JSON fields, public package identity, or cross-build join keys. A future package-
aware diagnostic schema must use an explicitly versioned package reference and display policy.

Semantic, resource, Region, Result, MIR, MIR-verifier, and package fixture manifests are repository test contracts, not the released diagnostic JSON schema, a stable MIR exchange format, or a public package-manager format. Their related-span roles and canonical fact shapes may change only through a new fixture schema version plus synchronized cases. `fairyc.region-fixtures/v1` adds an optional resolved-package-graph reference while preserving v0 span and expectation shapes; v0 remains readable historical input.

The native differential report is also a repository evidence contract rather than released compiler
output. `fairyc.native-c-report/v2` supersedes v1 by adding the required
`dropTraceExecutions` count. V3 supersedes v2 by adding the required sorted `layoutCases` identity
list and `layoutProbeExecutions` count. V4 supersedes v3 by adding the exact `sanitizer` mode
identity and `sanitizerExecutions` count; the existing `cases` list owns sanitizer case identity.
V5 supersedes v4 by adding the exact sorted `abiCases`, exact `abiDirections`, and
`abiProbeExecutions` count. V6 supersedes v5 by adding `systemHeapExecutions` for the closed
Clang/GCC low/optimized provisional standard-library allocation probe. V7 supersedes v6 by adding
`bufferExecutions` for the failure-atomic overlap-safe Buffer probe. V8 supersedes v7 by adding
`arenaExecutions` for the single-active-Region Arena probe. V9 supersedes v8 by adding
`textViewExecutions` for canonical UTF-8 validation and boundary slicing. V10 supersedes v9 by
adding `ownedTextExecutions` for String ownership and CString termination. V11 adds
`linuxFileExecutions` for partial I/O and consuming close. V12 adds `foreignCallExecutions` for the
closed exact-name scalar Fairy-source caller to hand-written C callee crossings. V13 adds
`foreignPointerCallExecutions` for the typed-null raw-pointer plus `usize`/`u8` crossing. The runner
V14 adds `textValidationSourceCallExecutions` for a generated Fairy caller linked against the
actual provisional text-view C unit. V15 adds `textViewOutputSourceCallExecutions` for a generated
Fairy caller that passes a verified raw address to a flat `@repr(C)` output record, observes the
record mutation, and links the same real text-view C unit. ADR 0051 later strengthens the existing
source probes without changing report shape or counters: their Fairy declaration names now differ
from their emitted external `@link_name` symbols. V16 adds `foreignCIntCallExecutions` for the
exact `c::int` producer-result to typed-local to consumer-argument crossing. V17 adds
`foreignCSizeTCallExecutions` for the corresponding exact `<stddef.h>` `size_t` crossing and adds
target `c.size_t` to the independently versioned target data. V18 adds
`foreignCSSizeTCallExecutions`, target `c.ssize_t`, and optional scalar `header`/`availability`
metadata for the negative-preserving POSIX.1-2008 crossing. V19 adds
`foreignCLongCallExecutions` for the exact negative C `long` crossing without changing TargetSpec
or layout-fixture shape. V20 adds `foreignCScalarFamilyCallExecutions` for one generated Fairy caller
and one hand-written C companion crossing the remaining twelve exact TargetSpec C scalars in both
compiler directions and profiles. V21 adds `systemHeapSourceCallExecutions` for the generated Fairy
caller crossing the exact `void *`/`size_t` block into the real SystemHeap allocate and deallocate
helpers in both compiler directions and profiles. V22 adds `bufferSourceCallExecutions` for a
generated Fairy caller crossing the exact nested SystemHeap-block plus `size_t` Buffer record into
the real initialize, zeroed-resize, and drop helpers in both compiler directions and profiles. V23
adds `arenaSourceCallExecutions` for exact self-linked node, Arena, and Region records crossing real
initialization, begin, zeroed allocation, reset, and drop in both directions and profiles. V24 adds
`cStrValidationSourceCallExecutions` for an exact generated CStr view crossing the real bounded
validator with valid, interior-NUL, and unterminated storage in both directions and profiles. V25
adds `foreignCStructResultCallExecutions` for one generated Fairy caller receiving both an 8-byte
register result and a 20-byte hidden-memory result from C in both directions and profiles. V26 adds
`ownedTextSourceCallExecutions` for exact generated String/CString and result records crossing the
real owned-text lifecycle helpers in both directions and profiles. V27 adds
`linuxFileSourceCallExecutions` for exact generated result, File, and open-option records crossing
the real Linux open, write-all, sync, consuming-close, partial-read, EOF, and final-close lifecycle
in both directions and profiles. V28 adds `standardLibraryBuildSmokeExecutions` for an ordinary
product build that resolves a self-contained Fairy UTF-8 validator call through the shared
used-symbol standard-library contract under both compilers and profiles. V29 adds
`cStringLiteralSourceCallExecutions` for a generated Fairy caller whose compiler-owned terminated
storage is read by C and `cStringLiteralBuildSmokeExecutions` for the ordinary product-build path
carrying nonempty static `&CStr` storage, both under both compilers and profiles. V30 adds
`byteStringLiteralSourceCallExecutions` for a generated Fairy caller whose private `&[u8]` view
retains exact bytes and length before C reads its data, plus `byteStringLiteralBuildSmokeExecutions`
for the ordinary product-build path carrying that view under both compilers and profiles. V31 adds
`utf8StringLiteralSourceCallExecutions` for a generated Fairy caller whose distinct private `&str`
view retains exact valid UTF-8 bytes and byte length before C reads its data, plus
`utf8StringLiteralBuildSmokeExecutions` for the ordinary product-build path carrying that text view
under both compilers and profiles. V32 adds `linuxConsoleExecutions` for the target-neutral POSIX
console adapter under both compilers and profiles; ADR 0209 later expands each execution to exercise
both stdout and stderr without changing the report field or count. The
runner's existing `ownedTextExecutions` condition is strengthened by ADR 0132 to require
probe-only deterministic initial-allocation and append-reallocation failure plus live-allocation
cleanup. ADR 0133 similarly strengthens `arenaExecutions` with deterministic payload and
tracking-node allocation failure, rollback, zeroing, and live-allocation cleanup. It also admits the
existing internal `fy_std_region_copy_bytes_v0` entry point in the closed target resolvers without
stabilizing a public source or ABI contract. ADR 0134 strengthens `bufferExecutions` with
deterministic construction, reserve, self-overlap append growth, and zeroed-resize growth failure,
exact storage preservation, recovery zeroing, and live-allocation cleanup. None of these changes
adds a report field or counter, so they did not create a new report revision at the time. The
runner emits one exact version and does not pretend the report shapes are interchangeable.

## Runtime and standard library

Pre-1 runtime ABI uses an exact match. Runtime helper symbols carry an ABI-major suffix. The driver diagnoses an incompatibility before invoking the linker.

The standard library has an independent version and declares required language revision, runtime ABI, target profile, and package instance. Language and standard-library numbers do not move in lockstep.

ADR 0205 moves the compiler-bundled SDK from `0.0.0-stage0.1` to `0.0.0-stage0.2` because it adds
public `std::io::write` and `ConsoleError` source declarations and changes the exact SDK source
digest. The SDK and runtime ABI identity strings remain unchanged. The target-neutral internal C
symbol, numeric status meanings, and private result layout are pre-1 implementation compatibility
data, not a public C ABI or stable Fairy native ABI.

ADR 0209 moves the SDK to `0.0.0-stage0.3` because it adds public `std::io::write_stderr` and the
private target-neutral stderr symbol. Existing source that does not call the new function remains
valid, but the exact SDK digest, package instance, public interface hash, dependent identities, and
generated lock bytes change. The runtime ABI identity string and numeric result layout remain
unchanged; status 2 applies to whichever output stream was selected.

ADR 0210 moves the SDK to `0.0.0-stage0.4` because it adds public
`std::io::write_bytes` and `std::io::write_stderr_bytes` declarations. Existing text calls and the
two private raw console symbols are unchanged, but the exact SDK digest, package instance, public
interface hash, dependent identities, and generated lock bytes change. The runtime ABI identity,
numeric result layout, status meanings, and target C support-symbol set remain unchanged.

ADR 0211 moves the SDK to `0.0.0-stage0.5` because it adds public
`std::io::read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>` and the private
`fy_std_stdin_read_v0` target support symbol. Existing output calls and their raw result layout are
unchanged, but the exact SDK digest, package instance, public interface hash, dependent identities,
and generated lock bytes change. The runtime ABI identity remains unchanged. The input result uses
its own private numeric layout and treats a successful zero count as EOF; the new C symbol is pre-1
implementation compatibility data rather than a public C or Fairy native ABI.

ADR 0215 moves the SDK to `0.0.0-stage0.6` because it adds public
`std::io::write_i32(i32) -> Result<usize, ConsoleError>` and the private
`fy_std_stdout_write_i32_v0` target support symbol. The exact SDK digest, package instance, public
interface hash, dependent identities, and generated lock bytes change. The language revision,
runtime ABI identity, output result layout, and numeric status meanings remain unchanged.

ADR 0246 moves the SDK to `0.0.0-stage0.7` because it adds public `std::bytes::Buffer`,
`BufferError`, and the complete safe constructor, mutation, metadata, and borrow surface. The SDK
source digest, package instance, public-interface hash, dependent identities, generated locks, and
whole-package cache keys change. The runtime ABI stays `fairyc-runtime/0`; the previously private
Buffer C entry points retain their signatures while the target allowlist admits all seven.

ADR 0247 moves the SDK to `0.0.0-stage0.8` because it adds public `std::text::String`, `CString`,
`TextError`, explicit fallible construction and append, scalar byte metadata, and owner-bound
`ref str`/`ref CStr` views. The SDK source digest, package instance, public-interface hash,
dependent identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/7` and runtime ABI remains `fairyc-runtime/0`; the existing five opaque storage
bridges and their numeric status meanings are unchanged.

ADR 0248 moves the SDK to `0.0.0-stage0.9` because it adds public
`std::text::validate_utf8(ref u8[]) -> Result<ref str, TextError>`. Success is an allocation-free,
owner-bound view of the exact input bytes; invalid input reports text status `2` and allocation
status `0`. The SDK source digest, package instance, public-interface hash, dependent identities,
generated locks, and whole-package cache keys change. Language remains `fairyc-language/7` and
runtime ABI remains `fairyc-runtime/0`; the existing validator symbol and private `fy_str`
representation are unchanged.

ADR 0250 moves the SDK to `0.0.0-stage0.10` because it replaces the `std::alloc::Arena`
placeholder with a runtime-compatible move-only resource, adds fallible `create()`, typed
`AllocError`, exact status inspection, and automatic drop. Language remains `fairyc-language/7`
and runtime ABI remains `fairyc-runtime/0`; this does not yet stabilize Region placement or a
public Arena C ABI.

ADR 0251 cuts the language revision to `fairyc-language/8` because compiler-known
`Region.place(T)` now returns `Result<ref T, std::alloc::AllocError>` instead of a direct borrow.
The SDK remains `0.0.0-stage0.10` and runtime ABI remains `fairyc-runtime/0`: the embedded SDK
already owns `AllocError`, while the changed operation and tagged borrow provenance are language and
MIR contracts. Manifests, generated locks, and whole-package cache keys therefore change even though
the embedded SDK source digest does not.

ADR 0252 adds constrained-C lowering for the already-versioned language `/8` Region contract. This
is backend capability and native evidence, not another source-language, SDK, or runtime ABI change:
language remains `fairyc-language/8`, SDK remains `0.0.0-stage0.10`, and runtime remains
`fairyc-runtime/0`. A backend that lacks the closed Arena/Region target symbols still rejects the
program before toolchain invocation.

ADR 0261 moves the SDK to `0.0.0-stage0.11` because it adds the public `std::process` module with
`arg_count() -> usize` and
`arg(usize index, ref mut u8[] buffer) -> Result<usize, ProcessArgError>`, plus the private
`fy_std_process_arg_count_v0` and `fy_std_process_arg_v0` Windows target support symbols. The SDK
source digest, package instance, public-interface hash, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/8` and runtime ABI remains
`fairyc-runtime/0`; stable `fairyc run` forwards arguments after `--` without changing the generated
C entry, and the program-argument status/`needed` contract is pre-1 implementation compatibility
data rather than a public C or Fairy native ABI.

ADR 0262 moves the SDK to `0.0.0-stage0.12` because it adds the public
`write_u32(u32) -> Result<usize, ConsoleError>`, `write_usize(usize) -> Result<usize, ConsoleError>`,
and `write_bool(bool) -> Result<usize, ConsoleError>` functions to `std::io`, plus the private
`fy_std_stdout_write_u32_v0` and `fy_std_stdout_write_usize_v0` target support symbols. The SDK
source digest, package instance, public-interface hash, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/8` and runtime ABI remains
`fairyc-runtime/0`; `write_bool` reuses the existing stdout write-all symbol and adds no new C ABI.

ADR 0263 moves the SDK to `0.0.0-stage0.13` because it adds the public
`std::text::from_buffer(bytes::Buffer) -> Result<String, bytes::Buffer>` zero-copy ownership
transfer, plus the private `fy_std_string_from_buffer_storage_v0` target support symbol, and makes
`std::text` import `std::bytes`. The SDK source digest, package instance, public-interface hash,
dependent identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; the new storage symbol is pre-1
implementation compatibility data rather than a public C or Fairy native ABI.

ADR 0264 moves the SDK to `0.0.0-stage0.14` because it adds public `std::fs::File`, `FileError`,
bounded read/write-all, explicit sync, consuming close, automatic drop, and scalar error accessors.
The SDK source digest, package instance, public-interface hash, dependent identities, generated
locks, and whole-package cache keys change. Language remains `fairyc-language/8` and runtime ABI
remains `fairyc-runtime/0`; no C symbol changes because the module consumes the existing six
Windows File target-support symbols. Native linkage is currently Windows MSVC only and does not
promise portable paths or Linux public support.

ADR 0269 moves the SDK to `0.0.0-stage0.15` because it adds public
`std::fs::position(ref mut File) -> Result<usize, FileError>` and
`std::fs::seek_start(ref mut File, usize) -> Result<usize, FileError>`, plus the two private
Windows target-support symbols that implement them. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; the surface is
still Windows x86-64 MSVC-only and does not promise portable offsets or richer seek origins.

ADR 0270 moves the SDK to `0.0.0-stage0.16` because it adds public
`std::fs::size(ref File) -> Result<usize, FileError>` and the private
`fy_std_windows_file_size_v0` target-support symbol. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; the observation is
still Windows x86-64 MSVC-only and does not stabilize general metadata or portable file kinds.

ADR 0271 moves the SDK to `0.0.0-stage0.17` because it adds public
`std::fs::open_update(ref CStr) -> Result<File, FileError>`. It opens only an existing file with
read and write access and does not create or truncate it. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; no C symbol
changes because the wrapper reuses the existing Windows File open target-support operation.

ADR 0272 moves the SDK to `0.0.0-stage0.18` because it adds public
`std::fs::set_len(ref mut File, usize) -> Result<usize, FileError>` and the private
`fy_std_windows_file_set_len_v0` target-support symbol. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; the new symbol is
Windows x86-64 MSVC-only pre-1 implementation compatibility data rather than a stable native ABI.

ADR 0274 moves the SDK to `0.0.0-stage0.19` because it adds public
`std::fs::open_or_create(ref CStr) -> Result<File, FileError>`. The SDK source digest, package
instance, public-interface hash, dependent identities, generated locks, and whole-package cache
keys change. Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; no C
symbol or private open-options layout changes because the wrapper reuses the already admitted
Windows File open operation's `OPEN_ALWAYS` combination.

ADR 0275 moves the SDK to `0.0.0-stage0.20` because it adds public
`std::fs::create_new(ref CStr) -> Result<File, FileError>`. The SDK source digest, package instance,
public-interface hash, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; no C symbol
changes. The compiler-private open-options record grows from four bytes to five so the bundled
source can select the existing Windows File open operation's `CREATE_NEW` disposition. That pre-1
record is not a stable public native ABI.

ADR 0276 moves the SDK to `0.0.0-stage0.21` because it adds public
`std::fs::open_append(ref CStr) -> Result<File, FileError>`. The SDK source digest, package
instance, public-interface hash, dependent identities, generated locks, and whole-package cache
keys change. Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`; no C
symbol changes. The compiler-private open-options record grows from five bytes to six so the
existing Windows File open operation can select `FILE_APPEND_DATA`. That pre-1 record is not a
stable public native ABI.

ADR 0277 moves the SDK to `0.0.0-stage0.22` because it adds public
`std::fs::FileKind`, `std::fs::kind(ref File) -> Result<FileKind, FileError>`, and the four
target-neutral kind predicates, plus the private `fy_std_windows_file_kind_v0` target-support
symbol. The SDK source digest, package instance, public-interface hash, dependent identities,
generated locks, and whole-package cache keys change. Language remains `fairyc-language/8` and
runtime ABI remains `fairyc-runtime/0`; the fixed private C result and symbol are Windows x86-64
MSVC-only pre-1 implementation compatibility data rather than a stable native ABI.

ADR 0278 moves the SDK to `0.0.0-stage0.23` because it adds public
`std::fs::read_exact(ref mut File, ref mut u8[]) -> Result<usize, FileError>`, stable unexpected-EOF
status `7`, and the private `fy_std_windows_file_read_exact_v0` target-support symbol. The SDK
source digest, package instance, public-interface hash, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/8` and runtime ABI remains
`fairyc-runtime/0`; the new symbol and status are additive Windows x86-64 MSVC-only pre-1
compatibility data rather than a stable native ABI.

ADR 0279 moves the SDK to `0.0.0-stage0.24` because it adds opaque public `std::fs::FileId`,
`std::fs::file_id(ref File) -> Result<FileId, FileError>`, borrowed equality, and the private
`fy_std_windows_file_id_v0` target-support symbol plus 24-byte output record. The SDK source digest,
package instance, public-interface hash, dependent identities, generated locks, and whole-package
cache keys change. Language remains `fairyc-language/8` and runtime ABI remains
`fairyc-runtime/0`; the private output layout and symbol are Windows x86-64 MSVC-only pre-1
implementation compatibility data rather than a stable native ABI or serialized identity format.

ADR 0281 moves the SDK to `0.0.0-stage0.25` because it changes public `std::fs::FileKind` from an
opaque private-field struct to an exhaustive target-neutral enum with `.Unknown`, `.Disk`,
`.Character`, and `.Pipe` variants. Supported callers of `kind` and the four retained predicate
helpers remain source-compatible; callers gain direct exhaustive switching. The SDK source digest,
public-interface hash, package instance, dependent identities, generated locks, and whole-package
cache keys change. Language remains `fairyc-language/8`, runtime ABI remains `fairyc-runtime/0`, and
the private raw C result and target symbol remain unchanged.

ADR 0284 cuts the exact language pin from `fairyc-language/8` to `fairyc-language/9` because
previously unsupported positive bounded copy-scalar fixed arrays become valid direct `Option` and
`Result` payloads. Existing `/8` source remains valid, while migration preserves source bytes and
requires an exact manifest pin update. SDK source and public API remain `0.0.0-stage0.25`, but its
language requirement, package instance, dependent lock entries, and whole-package cache keys change.
Runtime ABI, target schemas, C dialect, MIR version, and public native ABI remain unchanged.

ADR 0285 moves the SDK to `0.0.0-stage0.26` because it adds public
`std::fs::write(ref mut File, ref u8[]) -> Result<usize, FileError>` and admits the private
`fy_std_windows_file_write_v0` target-support symbol. Existing public operations remain
source-compatible. The SDK source digest, public-interface hash, package instance, dependent
identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/9`, runtime ABI remains `fairyc-runtime/0`, and the raw result layout and target
symbol remain Windows x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0299 moves the SDK to `0.0.0-stage0.36` because it adds public
`std::bytes::append_byte(ref mut Buffer, u8) -> Result<usize, BufferError>`. Existing SDK operations
remain source-compatible. The wrapper reuses `fy_std_buffer_append_v0`, so the SDK source digest,
public-interface hash, package instance, dependent identities, generated locks, and whole-package
cache keys change without a new target symbol. Language remains `fairyc-language/10` and runtime
ABI remains `fairyc-runtime/0`.

ADR 0302 moves the SDK to `0.0.0-stage0.37` because it adds public
`std::text::as_bytes(ref str) -> ref u8[]`. Existing SDK operations remain source-compatible. The
wrapper reuses compiler-known immutable raw-parts construction and preserves the input borrow
origin, so the SDK source digest, public-interface hash, package instance, dependent identities,
generated locks, and whole-package cache keys change without a runtime symbol, target component, or
ABI change. Language remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0303 moves the SDK to `0.0.0-stage0.38` because it adds public
`write_stderr_i32(i32)`, `write_stderr_u32(u32)`, `write_stderr_usize(usize)`, and
`write_stderr_bool(bool)` functions to `std::io`. The three numeric functions add private
target-neutral `fy_std_stderr_write_*_v0` support symbols; the bool function reuses
`fy_std_stderr_write_all_v0`. Existing SDK calls remain source-compatible. The SDK source digest,
public-interface hash, package instance, dependent identities, generated locks, and whole-package
cache keys change. Language remains `fairyc-language/10` and runtime ABI remains
`fairyc-runtime/0`; the raw result layout and status meanings are unchanged.

ADR 0305 moves the SDK to `0.0.0-stage0.39` because it adds public
`std::fs::seek_current(ref mut File, i32)` and `std::fs::seek_end(ref mut File, i32)`. Existing SDK
calls remain source-compatible. The two functions add private Windows-only
`fy_std_windows_file_seek_current_v0` and `fy_std_windows_file_seek_end_v0` symbols while reusing
the existing position-result C layout. The SDK source digest, public-interface hash, package
instance, dependent identities, generated locks, and whole-package cache keys change. Language
remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0306 moves the SDK to `0.0.0-stage0.40` because it adds public
`std::fs::is_read_only(ref File) -> Result<bool, FileError>`. Existing SDK calls remain
source-compatible. The function adds private Windows-only
`fy_std_windows_file_read_only_v0` and its private attribute-result C layout. The SDK source digest,
public-interface hash, package instance, dependent identities, generated locks, and whole-package
cache keys change. Language remains `fairyc-language/10` and runtime ABI remains
`fairyc-runtime/0`.

ADR 0307 moves the SDK to `0.0.0-stage0.41` because it adds public
`std::fs::hard_link_count(ref File) -> Result<usize, FileError>`. Existing SDK calls remain
source-compatible. The function adds the private Windows-only
`fy_std_windows_file_link_count_v0` symbol and reuses the private size-result C layout. The SDK
source digest, public-interface hash, package instance, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/10` and runtime ABI remains
`fairyc-runtime/0`.

ADR 0308 moves the SDK to `0.0.0-stage0.42` because it adds public
`std::fs::hard_link(ref CStr, ref CStr) -> Result<i32, FileError>`. Existing SDK calls remain
source-compatible. The function adds the private Windows-only `fy_std_windows_file_hard_link_v0`
symbol and reuses the private file-result layout. The SDK source digest, public-interface hash,
package instance, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0309 moves the SDK to `0.0.0-stage0.43` because it adds public
`std::fs::allocated_size(ref File) -> Result<usize, FileError>`. Existing calls remain
source-compatible. The function adds private Windows symbol
`fy_std_windows_file_alloc_size_v0` and reuses the private size-result layout. SDK source
digest, public-interface hash, package instance, dependent identities, locks, and cache keys change.
Language remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0310 moves the SDK to `0.0.0-stage0.44` because it adds public
`std::fs::set_read_only(ref mut File, bool) -> Result<bool, FileError>`. Existing calls remain
source-compatible. The function adds private Windows symbol
`fy_std_windows_file_set_readonly_v0` and reuses the private attribute-result layout. SDK source
digest, public-interface hash, package instance, dependent identities, locks, and cache keys change.
Language remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0311 moves the SDK to `0.0.0-stage0.45` because it adds public
`std::fs::remove_dir_all(ref CStr) -> Result<usize, FileError>`. Existing calls remain
source-compatible. The function adds private Windows symbol `fy_std_windows_dir_remove_all_v0`
and reuses the private size-result layout, including exact completed-object progress on failure.
SDK source digest, public-interface hash, package instance, dependent identities, locks, and cache
keys change. Language remains `fairyc-language/10` and runtime ABI remains `fairyc-runtime/0`.

ADR 0286 cuts the exact language pin from `fairyc-language/9` to `fairyc-language/10` because the
previously unsupported explicit pairs `i32 -> c::int` and `c::int -> i32` become valid when the
selected target declares signed 32-bit `c.int`. Existing `/9` source remains valid; migration
preserves source bytes and updates the exact manifest pin. SDK source and public API remain
`0.0.0-stage0.26`, but the SDK language requirement, package instance, dependent lock entries, and
whole-package cache keys change. Runtime ABI, target schemas, C dialect, MIR version, and public
native ABI remain unchanged.

ADR 0288 moves the SDK to `0.0.0-stage0.27` because it adds public
`std::fs::ReadAppendError`, its three status accessors, and
`std::fs::read_append(ref mut File, ref mut bytes::Buffer, usize) ->
Result<usize, ReadAppendError>`. Existing operations remain source-compatible. The SDK source
digest, public-interface hash, package instance, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/10`, runtime ABI remains
`fairyc-runtime/0`, and the new dependency-closed target symbol and raw record remain Windows
x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0290 moves the SDK to `0.0.0-stage0.28` because it adds public
`std::fs::read_to_end(ref mut File, ref mut bytes::Buffer) ->
Result<usize, ReadAppendError>`. The helper is implemented entirely in compiler-bundled Fairy
source over the existing bounded `read_append` operation, so no target symbol, raw layout, or
runtime ABI changes. Existing operations remain source-compatible. The SDK source digest,
public-interface hash, package instance, dependent identities, generated locks, and whole-package
cache keys change. Language remains `fairyc-language/10` and runtime ABI remains
`fairyc-runtime/0`.

ADR 0291 moves the SDK to `0.0.0-stage0.29` because it adds public
`std::fs::remove_file(ref CStr) -> Result<i32, FileError>` and admits the private
`fy_std_windows_file_remove_v0` target-support symbol. Existing operations remain source-compatible.
The SDK source digest, public-interface hash, package instance, dependent identities, generated
locks, and whole-package cache keys change. Language remains `fairyc-language/10`, runtime ABI
remains `fairyc-runtime/0`, and the target symbol remains Windows x86-64 MSVC-only pre-1
implementation compatibility data.

ADR 0292 moves the SDK to `0.0.0-stage0.30` because it adds public
`std::fs::rename_file(ref CStr, ref CStr) -> Result<i32, FileError>` and admits the private
`fy_std_windows_file_rename_v0` target-support symbol. The helper rejects directory sources,
destination replacement, and cross-volume copy fallback. Existing operations remain
source-compatible. The SDK source digest, public-interface hash, package instance, dependent
identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/10`, runtime ABI remains `fairyc-runtime/0`, and the target symbol remains Windows
x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0293 moves the SDK to `0.0.0-stage0.31` because it adds public
`std::fs::create_dir(ref CStr) -> Result<i32, FileError>` and
`std::fs::remove_dir(ref CStr) -> Result<i32, FileError>` plus the private
`fy_std_windows_directory_create_v0` and `fy_std_windows_directory_remove_v0` target-support
symbols. Existing operations remain source-compatible. The SDK source digest, public-interface
hash, package instance, dependent identities, generated locks, and whole-package cache keys change.
Language remains `fairyc-language/10`, runtime ABI remains `fairyc-runtime/0`, and the target
symbols remain Windows x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0294 moves the SDK to `0.0.0-stage0.32` because it adds public
`std::fs::rename_dir(ref CStr, ref CStr) -> Result<i32, FileError>` and admits the private
`fy_std_windows_directory_rename_v0` target-support symbol. The helper rejects non-directory
sources, destination replacement, and cross-volume copy fallback. Existing operations remain
source-compatible. The SDK source digest, public-interface hash, package instance, dependent
identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/10`, runtime ABI remains `fairyc-runtime/0`, and the target symbol remains Windows
x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0296 moves the SDK to `0.0.0-stage0.33` because it adds public
`std::fs::create_dir_all(ref CStr) -> Result<i32, FileError>` and admits the private
`fy_std_windows_directory_ensure_v0` target-support symbol. Existing operations remain
source-compatible. Prefix creation is explicitly non-transactional: an error may leave directories
created earlier in the call. The SDK source digest, public-interface hash, package instance,
dependent identities, generated locks, and whole-package cache keys change. Language remains
`fairyc-language/10`, runtime ABI remains `fairyc-runtime/0`, and the target symbol remains Windows
x86-64 MSVC-only pre-1 implementation compatibility data.

ADR 0297 moves the SDK to `0.0.0-stage0.34` because it adds public
`std::fs::copy_file_bytes(ref CStr, ref CStr) -> Result<usize, FileError>` and admits the private
`fy_std_windows_file_copy_bytes_v0` target-support symbol. Existing operations remain
source-compatible. The operation opens the source and a newly created destination exclusively,
never replaces an existing destination, copies only bytes through a bounded fixed buffer, and
reports any retained partial destination prefix through `FileError.completed`. The SDK source
digest, public-interface hash, package instance, dependent identities, generated locks, and
whole-package cache keys change. Language remains `fairyc-language/10`, runtime ABI remains
`fairyc-runtime/0`, and the target symbol remains Windows x86-64 MSVC-only pre-1 implementation
compatibility data.

ADR 0298 moves the SDK to `0.0.0-stage0.35` because it adds public `std::fs::PathKind` and
`std::fs::path_kind(ref CStr) -> Result<PathKind, FileError>`, plus private Windows-only
`fy_std_windows_file_path_kind_v0`. Existing SDK operations and `FileKind` remain source-compatible.
The new public enum and function change the SDK source digest, package instance, public-interface
hash, dependent identities, generated locks, and whole-package cache keys. Language remains
`fairyc-language/10`, runtime ABI remains `fairyc-runtime/0`, and the private raw result layout and
symbol remain Windows x86-64 MSVC-only pre-1 implementation compatibility data.

## Package formats

Manifest schema, lockfile schema, source archive schema, and any future binary artifact schema are independent. Stage 0 needs manifest and lockfile schemas only. A stable binary package format would falsely imply a stable native Fairy ABI and is deferred.

`fairyc.lock/v1` is the first generated lock schema. ADR 0145 fixes canonical JSON bytes, complete
root-reachable package tuples and edges, closed local/SDK source-origin records,
identity-algorithm declarations, strict PackageInstanceId/source-digest recomputation, and
stale-workspace comparison. It is independently versioned from `fairyc.manifest/v1`; local paths
are refresh facts, while SDK identity and runtime ABI are exact host-path-free compatibility facts.

`fairyc.manifest/v1` is the first human project-manifest schema. It uses the exact `fairy.toml`
field set and local-only semantics in ADR 0143: required package UID/name/exact SemVer/language/source,
optional explicit sorted workspace members, and path dependencies with exact version plus explicit
sorted per-edge features. Unknown or missing fields, duplicate TOML keys, unsupported schema,
noncanonical paths/sets, and malformed identities reject. This schema does not imply lockfile,
workspace-loader, source-acquisition, registry, or solver compatibility.

ADR 0144 fixes the first loader behavior for this manifest schema: explicit root/member closure,
exact on-disk case, no symlinks, complete `.fy` source discovery, unique UIDs, one language revision,
exact member versions, package-source-digest local identity, and per-edge feature-distinct graph
instances. ADR 0147 adds one exact compiler-bundled SDK source origin with package UID
`fairyc.std`, SDK identity `fairyc-sdk-stdlib/0`, current version `0.0.0-stage0.45`, language
`fairyc-language/10`, and runtime ABI `fairyc-runtime/0`. Absolute host paths and implementation
guard values are not serialized compatibility data.

ADR 0148 consumes those independently versioned manifest, lock, package-instance, language, SDK,
runtime, target, and ABI identities without merging them. `experimental build-workspace` and
`run-workspace` are pre-stable command spellings; their root `main::main` convention and human
receipts are not a released compatibility promise.

ADR 0176 separately stabilizes `fairyc build [--release] [--locked]` for the current workspace on
Windows/amd64. Its root-package-derived `build/<name>.exe` path, development/release meanings,
replace-on-success rule, lock behavior, success receipt, and codes 0, 2, 3, 4, 6, and 7 are CLI
compatibility data. This does not stabilize the generated C, native ABI, MSVC version, distributable
runtime, target maturity, or persistent-cache policy.

ADR 0177 stabilizes `fairyc run [--release] [--locked]` over the same current-workspace and local
Windows boundary. Its temporary-image rule, development/release meanings, normal-result receipt,
trap receipt, and codes 0, 2, 3, 4, 5, 6, and 7 are CLI compatibility data. It does not stabilize
the generated C, native ABI, MSVC version, target maturity, program streams or arguments, runtime
distribution, or persistent-cache policy. Its compiler-private trap token is an implementation
detail rather than program stderr or serialized compatibility data; the observable obligation is
that all normal `i32` results remain distinct from traps.

ADR 0207 explicitly replaces the v1 normal-result receipt with stable run v2. After launch, exact
bounded stdout/stderr forwarding, no success banner, ordinary 32-bit program-status propagation,
private trap-suffix removal, preceding-stderr preservation, canonical trap status 5, the shared
64 KiB ceiling, and unspecified cross-stream order are compatibility data. Pre-launch compiler codes
remain unchanged, but equal post-launch values may be program-owned. Program arguments, stdin, live
streaming, Linux stable execution, runtime distribution, and target maturity remain outside v2.

ADR 0212 replaces guaranteed stdin EOF with stable run v3 caller-input forwarding. Attaching stdin
only after successful compilation and image inspection, consuming no input on pre-launch failure,
performing no compiler-side pre-read or retention, and keeping `cli.Run` as an EOF embedding helper
are compatibility data. The command spelling, flags, output ceiling, timeout, status/trap rules, and
pre-launch compiler codes remain unchanged. Program arguments, live output interleaving, input
forwarding for other native commands, Linux stable execution, runtime distribution, and target
maturity remain outside v3.

ADR 0178 stabilizes `fairyc explain <diagnostic-code>` independently from the provisional catalog
lifecycle. Exact lookup, the three-line shape, code/category/catalog identities, and codes 0, 2, and
4 are CLI compatibility data. Meaning prose may improve under the existing diagnostic policy, but
a code is never silently reassigned; implementation, fixture, release, JSON, and span claims remain
separate.

ADR 0179 stabilizes the source-independent `fairyc targets` human report grammar, canonical ordering,
atomic output, and codes 0, 2, and 4. Additive target rows, evidence removal, and maturity changes are
TargetSpec data evolution rather than CLI grammar changes; target removal or reidentification still
requires an explicit compatibility decision. Local availability and experimental target JSON remain
separate.

ADR 0180 stabilizes `fairyc emit c --target <target-id> [--locked]`, mandatory explicit target
selection, raw stdout publication, the C11 transport identity, lock behavior, and codes 0, 2, 3, 4,
and 7. Exact whole-unit bytes and private symbols are deterministic only for one exact compiler,
input, and target and remain disposable transport; public `@repr(C)` and `extern C` contracts,
TargetSpec data, runtime/support units, and generated-C consumer evidence remain independently
versioned.

ADR 0181 stabilizes `fairyc test [--locked]`, the root-only optional lowercase `tests/` overlay,
`tests/`-prefixed module identity, argument-free `@test` declaration shape, canonical test ordering,
fresh semantic-interpreter execution, human result grammar, production-lock isolation, `E9001`, and
codes 0, 2, 3, 4, 5, and 7. Target-native execution, member recursion, filters, setup/teardown,
benchmarks, host I/O, machine output, and test bytes in release package identities remain outside v1.

ADR 0265 additively extends that command to `fairyc test [--native [--release]] [--locked]` while
leaving existing interpreter invocations byte-for-byte unchanged. Native low/optimized profile
selection, one compiler-private verified harness and temporary Windows image per test, closed
support linkage, exact trap continuation, native program-output rejection, no-artifact behavior,
and code 6 build failures are stable CLI behavior. Harness names/symbols, generated C and PE bytes,
temporary paths, MSVC installation identity, language revision, SDK, runtime ABI, manifests, locks,
and package/interface identities are not changed or newly stabilized.

ADR 0267 additively extends both engines with repeatable
`--exact <test-identity>` selection. Values reuse the existing canonical report identity
byte-for-byte, retain canonical order independent of flag order, and reject missing, duplicate, or
unknown selectors with code 2. Existing all-test invocations, report grammar, lock behavior,
language, SDK, runtime ABI, manifest, and package/interface identities remain unchanged. No
substring, prefix, glob, regex, tag, file, line, skip, ignore, or random-order compatibility is
implied.

ADR 0268 adds the separate `fairyc test --list [--locked]` discovery mode. It reuses the canonical
identity and complete compiler-admission boundary, emits deterministic `test <identity>` lines plus
one count/lock summary, and rejects combination with execution or exact-selection flags. Existing
test execution bytes and statuses remain unchanged. The mode adds no language, SDK, runtime ABI,
manifest, lockfile, package, interface, target, or generated-code version change; machine-readable
listing and additional discovery semantics are not implied.

ADR 0273 adds `fairyc test --list --json [--locked]` with schema
`fairyc.test-list/v1`. The required fields are `schemaVersion`, canonical `tests`, matching
`count`, and `lock`; field order and compact serialization are deterministic output bytes, while
consumers must use the schema names rather than parse the text listing. Existing text listing and
test execution remain byte-for-byte unchanged. Language, SDK, runtime ABI, manifest, lockfile,
package, interface, target, and generated-code versions remain unchanged.

ADR 0287 additively admits `fairyc test --json [--native [--release]]
[--exact <test-identity>]... [--locked]` with schema `fairyc.test-report/v1`. Existing invocations
without execution-mode `--json` retain exact text bytes and codes. The JSON report freezes canonical
per-test outcomes, engine, nullable target/profile, totals, aggregate status, and lock state; a
complete trapped suite still returns code 5 after writing the report. Language, SDK, runtime ABI,
manifest, lockfile, package, interface, target, MIR, and generated-code versions remain unchanged.

ADR 0289 additively admits singular `--package <package-uid>` for stable test execution and listing.
The exact local manifest UID selects one root-reachable resolved package and its isolated test
overlay; omission preserves the existing root-only bytes. Canonical test identities and both JSON
v1 schemas remain unchanged because one invocation admits one package. Recursive aggregation,
multi-package selection, and feature-instance selection are not implied. Language, SDK, runtime
ABI, manifest, lockfile, package, interface, target, MIR, and generated-code versions remain
unchanged.

ADR 0295 additively admits mutually exclusive `--workspace` aggregation over every unambiguous
root-reachable local package in the production graph. Only this mode qualifies exact identities as
`<package-uid>::<module>::<function>` and uses `fairyc.test-list/v2` or
`fairyc.test-report/v2`; the v2 schemas retain their v1 field sets but change the identity domain.
Default root and singular `--package` discovery, text and JSON bytes, codes, locking, and artifacts
remain unchanged. Catalog-unreachable members, multiple feature instances, parallel scheduling,
and portable native targets are not implied. Language, SDK, runtime ABI, manifest, lockfile,
package, interface, target, MIR, and generated-code versions remain unchanged.

ADR 0300 additively admits `--locations` for stable test listing and execution. Omission preserves
all v1/v2 JSON and text bytes. Located JSON uses `fairyc.test-list/v3` or
`fairyc.test-report/v3`; the new source record freezes package UID, portable workspace-relative
path, 1-based line and byte column, half-open function-name byte span, and exact `fyfile1`/`fylm1`
identities. Canonical test identity, `--exact`, ordering, locking, exit codes, engine behavior, and
artifacts remain unchanged. Language, SDK, runtime ABI, manifest, lockfile, package, interface,
target, MIR, and generated-code versions remain unchanged.

ADR 0301 additively admits repeatable `--file <portable-test-path>` for stable test listing and
execution. Values match the `--locations` portable workspace-relative test-source path domain
byte-for-byte; unknown, duplicate, missing, host-separated, and case-mismatched values fail closed.
A known zero-test file succeeds. `--file` is mutually exclusive with `--exact`, preserves canonical
ordering and complete-overlay compiler admission, adds no output field, and leaves all v1/v2/v3
text and JSON bytes unchanged. Language, SDK, runtime ABI, manifest, lockfile, package, interface,
target, MIR, and generated-code versions remain unchanged.

ADR 0304 additively admits repeatable `--dir <portable-test-directory>` for stable test listing and
execution. Values match only the exact ancestor-directory catalog derived from admitted portable
test-source paths, recurse by a `/` segment boundary, and never trigger a second filesystem read.
Unknown, duplicate, missing, host-separated, case-mismatched, dot-segment, and trailing-slash values
fail closed. Directories backed only by admitted zero-test sources succeed, while empty filesystem
directories are not cataloged. `--dir` is mutually exclusive with `--file` and `--exact`, preserves
canonical ordering and complete-overlay compiler admission, adds no output field, and leaves all
v1/v2/v3 text and JSON bytes unchanged. Language, SDK, runtime ABI, manifest, lockfile, package,
interface, target, MIR, and generated-code versions remain unchanged.

ADR 0312 additively admits the first `--` delimiter only for native test execution and forwards
every following value unchanged to each selected isolated process. Semantic execution and listing
reject the delimiter; embedded NUL fails before workspace loading. Omission preserves program
arguments, text and v1/v2/v3 JSON bytes, schemas, codes, locking, and artifacts. Language, SDK,
runtime ABI, manifest, lockfile, package, interface, target, MIR, and generated-code versions remain
unchanged.

ADR 0313 additively admits native-only `--stdin` for exactly one selected test after complete
compiler admission. Omission preserves EOF. Zero or multiple selections fail before input
consumption or process launch; the successful path attaches caller input directly only after PE
inspection without compiler buffering or replay. Text and v1/v2/v3 JSON bytes, schemas, codes,
locking, and artifacts remain unchanged. Language, SDK, runtime ABI, manifest, lockfile, package,
interface, target, MIR, and generated-code versions remain unchanged.

ADR 0314 additively admits `--capture-output` only with native JSON execution and introduces
`fairyc.test-report/v4`. Every v4 outcome requires canonical padded base64 `stdoutBase64` and
`stderrBase64`; arbitrary binary and pre-trap bytes are exact, the private trap receipt is excluded,
and `source` remains conditional on `--locations`. Omission preserves native-output rejection and
all text plus v1/v2/v3 JSON bytes. V4 is a new opt-in schema rather than an in-place v1-v3 field
addition. Language, SDK, runtime ABI, manifest, lockfile, package, interface, target, MIR, and
generated-code versions remain unchanged.

ADR 0315 additively admits native-only `--stdin-file` with one canonical contained workspace path,
a 65,536-byte regular-file limit, and fresh byte-zero replay for every isolated selected test.
Omission preserves EOF or explicit caller stdin, and all report schemas and bytes remain unchanged.
Language, SDK, runtime ABI, manifest, lockfile, package, interface, target, MIR, and generated-code
versions remain unchanged.

ADR 0316 additively admits native-only canonical `--timeout-seconds` from 1 through 60 and applies
it only to each test process after build and inspection. Omission preserves 60 seconds; reports and
all SDK, language, runtime ABI, manifest, lockfile, package, target, MIR, and generated-code versions
remain unchanged.

ADR 0204 stabilizes `fairyc new <package-name> --uid <package-uid>`, its explicit non-derived UID,
single direct-child destination, exact four-file starter, no-overwrite behavior, absence of an
initial lockfile, success line, and codes 0, 2, 4, and 7. Adding templates, flags, merge behavior, an
arbitrary destination, lock generation, or different default bytes requires a new compatibility
decision; process-crash transactionality is not promised.

ADR 0206 makes that explicit compatibility decision for starter project v2. The command spelling,
identity, destination, no-overwrite, lock, success, exit-code, and crash boundaries stay unchanged;
the exact set adds compiler-owned `README.md`. Its package-name substitutions, manual command
sequence, stable run stream warning, and complete safe `std::io::write` source fence are now part of
the generated bytes. Existing projects are never upgraded. Another file-set or byte change still
requires a later compatibility decision.

ADR 0208 moves the same five-file set to starter project v3. Public `answer()` and its semantic test
stay at 42, private `main()` now returns `answer() - 42`, and the exact README describes run v2 plus
direct locked safe-output execution. Existing projects are not rewritten, and no template choice or
upgrade surface is added.

ADR 0266 moves the same five-file set to starter project v4 by changing only the generated README.
It labels `fairyc test --locked` as the fast semantic lane and adds the explicit Windows
`fairyc test --native --locked` step plus its one-temporary-image-per-test explanation. Manifest,
source/test bytes, package identity, lock behavior, project command, no-overwrite containment,
success line, exit codes, and existing projects remain unchanged.

ADR 0182 introduces the independently identified in-memory
`fairyc.semantic-snapshot/v0` model. It removes build-local package and symbol numbering from
detached compiler facts and fixes provenance plus relation-specific completeness behavior, but it
does not stabilize `.fymap`, `SymbolId`, a serialized type spelling, query output, or CLI behavior.
Those boundaries require their own versions rather than inheriting the snapshot model number.

ADR 0183 adds the independent provisional `fairyc.symbol-id/v1-sha256-128+256` algorithm and its
`fairyc.symbol-identity/v1` plus `fairyc.symbol-signature/v1` encodings. Its visible 128-bit prefix
must be accompanied by the full 256-bit digest and collisions reject. It deliberately excludes
package instances, versions, bodies, spans, language, compiler, and target identity; the implemented
map header records those compatibility dimensions separately. This executable pre-release contract does not
stabilize `.fymap`, map migration, anchors, query output, or a post-1 SymbolId promise.

ADR 0184 adds independent `fairyc.source-file/v1` exact-content and
`fairyc.source-line-map/v1` byte-geometry identities with canonical
`fyfile1-sha256-...` and `fylm1-sha256-...` text forms. They supplement, never replace, the exact
`fysrc1-sha256` package graph digest. Changing the domain, version byte, length/offset widths, LF
line-start rule, digest, or text prefix requires a new identity version. Their presence in a
snapshot does not by itself prove a future map is current; a map reader must recompute and compare
them against caller-authorized current source.

ADR 0159 adds the independent `fairyc.experimental.workspace-check-session/v0` NDJSON request and
response schema. The schema versions this experimental machine boundary and rejects unknown or
duplicate request fields, but it does not make the command spelling, numeric limits, process exit
policy, or cache behavior release-stable. An incompatible field, kind, sequence, status, or unit
change requires a new protocol schema even before stable CLI promotion.

ADR 0166 adds the independent `fairyc.experimental.workspace-cache-check/v0` one-shot response
schema. Its exact executable identity, explicit cache path, counters, and store outcomes are an
unstable experimental machine boundary; changing field meaning or units requires another schema
even though the command and cache policy are not release-stable.

`fairyc.project-manifest-fixtures/v1` versions the complete-file parser observations for that schema.
The pinned TOML library version is an implementation dependency, not the manifest schema version;
upgrading or replacing it requires proving the same accepted/rejected fixture surface or explicitly
versioning the manifest contract.

`fairyc.package-fixtures/v0` versions the eight historical repository cases for an already-resolved
graph. `fairyc.package-fixtures/v1` explicitly includes those cases and adds the closed
`package-dependency-cycle` rejection after exact-instance deduplication. The runner retains direct
v0 readability. Neither version is the public manifest schema, lockfile schema, solver protocol, or
concrete `PackageInstanceId` encoding, and compatibility with either does not imply compatibility
with any future package-manager format.

`fairyc.package-identity-fixtures/v1` independently fixes `PackageInstanceId` v1 as SHA-256 over
the domain-separated, tagged, network-order-length encoding in
`docs/package/package-instance-id-v1.md`. The only text form is the full lowercase
`fypi1-sha256-...` digest. A future manifest or lockfile declares this identity version rather than
silently inheriting it from its own schema. Changing the digest, canonical bytes, prefix, or
validation rules requires a new package-identity version. Caches using another version are safe
misses; contradictory lockfile or map records are corruption.

`fairyc.module-interface-fixtures/v3` independently fixes the current public-interface descriptor
and `fyif3-sha256-...` hash. It preserves v2 coordinate-based nominal identity and adds public
internal and foreign callable must-use state. It excludes ordinary bodies, private implementation,
and private must-use state while including public type shape, callable signatures, effects and
must-use bits, foreign link contracts, module paths, and dependency nominal identities. V1 and v2
fixtures remain frozen historical vectors. A cache records the current algorithm explicitly and
combines it with its own schema and compatibility vector. Changing interface inputs, canonical
encoding, stable type spelling, digest, or prefix requires a new interface-hash version.

`fairyc.package-source-fixtures/v1` independently fixes the exact package-source graph digest as
SHA-256 over sorted canonical relative paths and exact source bytes. Its text form is
`fysrc1-sha256-...`. The source digest invalidates the current package implementation; dependency
public-interface hashes decide whether dependents invalidate. Changing path validation, canonical
bytes, digest, or prefix requires a new source-digest version.

The per-file identities are independently versioned because content staleness, line-map staleness,
and package-graph staleness are different checks. `fairyc.source-file/v1` hashes exact bytes with a
domain and byte length. `fairyc.source-line-map/v1` hashes byte length and every LF-derived
zero-based line start, including EOF after a final LF. CR is not normalized. A mismatch invalidates
stored spans for that file; no consumer may guess a remapped position.

ADR 0185 adds the internal `fairyc.fymap-logical/v1` planning model. It requires the exact compiler
executable identity, exact current package set, one `fyif3-sha256` public-interface hash per package,
all independent identity-algorithm identifiers, and explicit relation omissions. The model is
target-independent and immutable. It is not the `.fymap` transport schema and therefore makes no
promise by itself about encoded bytes, file naming, publication, or reader compatibility.

ADR 0186 independently fixes `fairyc.fymap/v1`, `fairyc.fymap-json/v1`, and
`fairyc.fymap-checksum/v1-sha256`. The only accepted representation is one compact canonical JSON
value with strict fields and a domain-separated full SHA-256 checksum. Changing field meaning,
field order, JSON record shape, number representation, checksum input, digest, or text prefix
requires a new schema, encoding, or checksum algorithm identifier as appropriate.

## `.fymap` compatibility

The logical plan already records schema, snapshot model, compiler build, language revision, package
UID, package instance, package coordinate, source digest and kind, runtime ABI, target scope,
features, dependencies, each identity algorithm, exact interface hash, record counts, and explicit
omissions plus exact self/direct-dependency namespace references. Transport v1 preserves those
meanings and adds independently versioned encoding and integrity fields.

Transport v1 readers require an exact schema, encoding, checksum, logical model, and algorithm
match. They reject unknown fields, noncanonical bytes, contradictory records, a SymbolId that does
not rederive from its declaration, and any corrupt checksum. A future version may define optional
sections, but v1 never silently skips one. Atlas failure never fails compilation.

ADR 0187 adds the internal content-addressed `fymap-v1` store layout and exact current-source reader.
It requires caller-selected storage bounds, the exact compiler executable and public-interface
identities, and complete package plus per-file source identities. Stale, incompatible, corrupt, and
conflicting generations are distinct rejection outcomes. The layout is pre-release internal state;
no default path, CLI location, retention policy, or cross-version migration promise exists.

ADR 0188 adds the internal package-local query contract. Query construction requires the same exact
current source/compiler/interface vector, selectors require canonical SymbolId v1 text, and every
request fixes record, relation, depth, and selection-byte bounds. Relation kind spellings, omission
codes, and compact-JSON selection accounting are pre-release internal policy, not a stable wire or
CLI promise. Dependency joins, Context Pack encoding, source excerpts, and cross-version query
migration remain absent.

ADR 0189 independently fixes output version `fairyc.context-pack/v1`, encoding
`fairyc.context-pack-json/v1`, checksum `fairyc.context-pack-checksum/v1-sha256`, and the exact
compiler-fact envelope field order. The output retains package/map/query/budget/completeness and
omission identity while requiring empty documentation, source-excerpt, and warning arrays. It is
pre-release internal output: no decoder, cross-version acceptance, text import, default location, or
CLI stability is promised.

ADR 0190 adds only the unstable `experimental atlas-pack overview` spelling. Its exact package-UID
selector, mandatory record/selection/output limits, optional locked mode, canonical no-newline
stdout, and failure-code mapping are experimental CLI policy. It does not stabilize `.fymap` store
layout or policy, SymbolId discovery, text import, or any Atlas compatibility family.

ADR 0191 adds path-free measurement schema `fairyc.atlas-rewrite-measurement/v0`. It versions the
local evidence envelope and exact encode-only denominator, not `.fymap` compatibility, a product
byte or latency limit, a default path, a format decision, or an incremental-update promise.

ADR 0227 adds path-free measurement schema `fairyc.atlas-call-process-measurement/v0`. It versions
the local raw process-evidence envelope, exact command-start-through-exit scope, source/call/output
denominators, Windows peak-memory metric labels, and explicit harness-policy echo. It does not change
the workspace-call catalog, command behavior, `.fymap`, Context Pack, compiler release, or language;
nor does it promise a reference budget, percentile, speedup, default, traversal, impact, or
cross-machine comparability.

ADR 0228 adds path-free measurement schema `fairyc.atlas-call-scale-measurement/v0`. It versions the
three closed generated package/call shapes, exact prepared join-plus-outgoing-encode scope, raw
monotonic-time and process-wide allocation metric labels, nine-sample policy, fixed compatibility
identities, and harness bounds. It does not change the workspace-call catalog, command behavior,
`.fymap`, Context Pack, compiler release, or language; nor does it promise a complexity law,
representative workload, retained-memory attribution, reference budget, default, traversal, impact,
regression threshold, or cross-machine comparability.

ADR 0192 adds an internal exact-current symbol catalog ordered and paged by full SymbolId. Its kind
filters, cursor, budget accounting, and omission codes are pre-release query policy. No standalone
wire schema, CLI spelling, name-search behavior, stable anchor, or cross-version cursor is promised.

ADR 0193 independently fixes output version `fairyc.symbol-catalog/v0`, encoding
`fairyc.symbol-catalog-json/v0`, checksum `fairyc.symbol-catalog-checksum/v0-sha256`, and the
unstable `experimental atlas-symbols list` spelling. This does not stabilize the CLI, cross-version
cursors, exact-name or fuzzy search, dependency joins, source text, or a catalog decoder.

ADR 0201 supersedes that output-only writer with `fairyc.symbol-catalog/v1`, encoding
`fairyc.symbol-catalog-json/v1`, checksum `fairyc.symbol-catalog-checksum/v1-sha256`, and prefix
`fyscat1-sha256-`. V1 adds one echoed optional exact Stage 0 declaration-name filter to the closed
result shape, applies module/name/kind filters before totals and cursor validation, and retains all
same-name records. Current code emits only v1 and accepts neither version as input. This does not
promise v0 migration, cross-version cursors, fuzzy matching, name normalization, or stable CLI.

ADR 0194 adds the unstable `experimental atlas-pack show` and `around` spellings over the existing
Context Pack v1 output. Visible/full selectors, directions, exact relation filters, and mandatory
record/relation/depth/selection/output policy are experimental CLI behavior. This does not stabilize
SymbolId across a future identity family, a cross-version query migration, graph completeness,
dependency joins, text import, default map storage, or either command spelling.

ADR 0195 fixes logical policy `fairyc.attributed-text/v0` and fragment checksum family
`fairyc.attributed-text-fragment/v0-sha256` without defining an encoded artifact or command. It does
not change `.fymap` v1 or Context Pack v1, whose source/documentation bodies remain absent. A future
encoding, decoder, composite pack, CLI, documentation attachment rule, dependency origin, or
generated-text lineage requires an independent compatibility decision.

ADR 0196 implements those logical records and workspace-authored exact source excerpts as an
internal Go boundary. Its validation and omission spellings are pre-release internal policy, not a
wire schema or command promise. The implementation revalidates retained package/file/line-map
identities and adds no cross-version acceptance, persistent text artifact, default budget, or
public consumer contract.

ADR 0197 independently fixes output encoding `fairyc.attributed-text-json/v0`, document checksum
`fairyc.attributed-text-checksum/v0-sha256`, prefix `fytextdoc0-sha256-`, exact envelope/document
field order, explicit source kind/origin selection, exact subject linkage, and positive final-output
limit. It adds no decoder, untrusted input, default limit, storage, or composite-pack promise.
Changing field meaning/order, JSON spelling, checksum input, authority, subject identity, selection,
usage, or omission semantics requires an explicit compatibility decision.

ADR 0198 adds only the unstable `experimental atlas-text source` spelling. Its mandatory exact
package/SymbolId selectors, `workspace-authored` origin, count/text/output budgets, optional locked
mode, canonical no-newline stdout, successful omission behavior, and provisional failure mapping are
experimental CLI policy. It does not stabilize the command, SymbolId across identity families,
cross-version text acceptance, defaults, dependency or SDK text, decoding, storage, or permissions.

ADR 0199 assigns physical-line-leading `///` the pre-release documentation-trivia role and fixes
its conservative top-level attachment span through typed HIR. The comment remains grammar trivia;
existing token kind output, SymbolId, MIR, C emission, `.fymap` v1, Context Pack v1, and
`atlas-text source` encodings do not change. Alternative documentation spellings, additional
attachment targets, rendering, Atlas extraction, and post-1 stability require separate decisions.

ADR 0200 activates the already reserved `declaration-documentation` kind and
`fairyc-declaration-doc-utf8` format in canonical `fairyc.attributed-text-json/v0` without adding or
reordering fields. It adds the unstable `experimental atlas-text documentation` action and
`kind-unavailable` omission, while existing source-command bytes and the frozen checksum vector
remain unchanged. The in-memory snapshot sidecar is not part of `.fymap` v1, SymbolId, Context Pack
v1, or a persisted compatibility surface. Additional origins, attachment targets, rendering,
decoding, storage, and stable command behavior require later decisions.

ADR 0202 independently fixes output version `fairyc.atlas-evidence/v0`, encoding
`fairyc.atlas-evidence-json/v0`, checksum `fairyc.atlas-evidence-checksum/v0-sha256`, prefix
`fyevidence0-sha256-`, exact envelope/document field order, authority labels, canonical exact-symbol
subject, three nested child positions, and component/final limit meanings. Each child remains a
complete existing canonical document; the wrapper does not revise or decode Context Pack v1 or
attributed-text v0. It also adds only the unstable `experimental atlas-evidence show` spelling.
Changing wrapper field meaning/order, JSON spelling, authority labels, component set, checksum input,
or limit semantics requires an explicit compatibility decision. No decoder, input migration,
cross-version child acceptance, default limit, persistence, authentication, or stable CLI behavior
is promised.

ADR 0225 adds only an in-memory exact-current workspace call join. Its package-qualified selectors,
direct-call ledger, and construction budgets are internal query policy rather than a wire format.
ADR 0226 independently fixes output version `fairyc.workspace-call-catalog/v0`, encoding
`fairyc.workspace-call-catalog-json/v0`, checksum
`fairyc.workspace-call-catalog-checksum/v0-sha256`, prefix `fywcall0-sha256-`, exact logical field
order, direct-call ordering, direction vocabulary, construction budget/usage meanings, and the
unstable `experimental atlas-calls list` spelling. It changes neither `.fymap` v1 nor Context Pack
v1. Changing field meaning/order, JSON spelling, checksum input, completion meaning, or selector
canonicalization requires an explicit compatibility decision. No decoder, cross-version input,
default budget, persistence, traversal, indirect-call recovery, impact, or stable CLI behavior is
promised.

Dependency handling is relation-specific:

| Condition | Atlas behavior |
| --- | --- |
| Missing dependency map | Local facts remain; transitive relations are partial. |
| Newer optional schema in same compatible family | Known sections may be used; affected relations are partial. |
| Incompatible schema major | Dependency semantic facts are unavailable. |
| Incompatible language semantics | Names and spans may remain; ownership and effects are unavailable. |
| Incompatible `SymbolId` algorithm | Package-local display remains; cross-package joins are unavailable. |
| Target mismatch | Target-independent facts remain; layout and ABI facts are unavailable. |
| Stale source or corrupt map | Entire map is rejected. |

## `SymbolId`

Canonical identity includes package UID, module path, declaration kind, canonical declaration path, generic arity when it later exists, and a canonical signature fingerprint. It excludes absolute paths, source positions, compiler release, body digest, and language revision. Language compatibility is checked by the map header; a canonical type encoding change creates a new `SymbolId` algorithm.

Graph edges identify a package instance plus symbol. `PackageUid` names the long-lived project; `PackageInstanceId` distinguishes resolved source, version, and feature instances.

The implemented provisional v1 representation uses the first 128 SHA-256 bits as
`fysym1-sha256-128-...` plus the stored complete digest as `fysym1-sha256-256-...`. Different full
digests under the same visible identifier are a hard collision error. Random salt is forbidden
because it would destroy deterministic identity. Exact tuple and signature bytes are defined in
`docs/atlas/symbol-id-v1.md`.

Different `SymbolId` algorithms do not exact-join. Future cross-version matching may use a stable anchor, canonical declaration comparison, or explicitly heuristic similarity, never silent equality.

## Deferred stability

Language 1.0 compatibility, stable native ABI and mangling, stable MIR exchange, binary package ABI, stable `SymbolId`, stable anchors, exact diagnostic-span guarantees, stable debugger formats, and cross-version `.fymap` migration are not Stage 0 promises.
