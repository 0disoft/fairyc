# Testing Standard

- Status: Draft
- Owner: UNASSIGNED
- Semantic sources: `spec/grammar/stage0.ebnf`, `spec/fixtures/parser/manifest-v0.json`, `spec/fixtures/semantic/manifest-v0.json`, `spec/fixtures/resource/manifest-v0.json`, `spec/fixtures/region/manifest-v1.json`, `spec/fixtures/result/manifest-v0.json`, `spec/fixtures/mir/manifest-v0.json`, `spec/fixtures/mir-verifier/manifest-v0.json`, `spec/fixtures/interpreter/manifest-v0.json`, `spec/fixtures/package/manifest-v1.json`, `docs/language/`, `docs/mir/`, `docs/package/`, `docs/abi/`, `docs/runtime/`, `docs/compatibility/`, `docs/diagnostics/catalog-v0.md`, `docs/atlas/`
- Merge-blocking validation: `VALIDATION.md`

The Go frontend executes every case in `spec/fixtures/parser/manifest-v0.json` through
`internal/parserfixture`. Source, lexer, syntax, AST, budget, containment, and manifest behavior
also have focused unit tests. Documentation-trivia tests require exact lossless subtype retention,
LF/CRLF attachment through attributes and modifiers, conservative detachment, and AST/HIR span
preservation without compiler-private inheritance. Lexer and parser fuzz targets run their deterministic seed corpus
during ordinary Go tests and assert lossless reconstruction, stable output, valid spans, bounded
  exhaustion, and error-free AST lowering. Stable project-creation tests freeze the exact five-file
  starter bytes, manifest reparsing, portable identity rejection, no-overwrite behavior, path-clean
  errors, the starter-v4 semantic/native guide distinction, and a generated project's ordinary check, locked check, lock publication, semantic test,
  and semantic checking of the actual README console source fence.
The Windows smoke repeats new/duplicate/check/test/run through a separately built compiler process,
checks the package-specific guide paths, safe source fence, and stable run warning, and requires
program status zero from the generated source.
Stable workspace-check tests cover lock creation, exact
no-write locked verification, rejected source arguments, missing manifests, parser diagnostics with
source exit code 3, and deterministic success output. Experimental CLI tests cover argument routing,
deterministic content-safe JSON, schema synchronization, human diagnostics,
bounded file reads, source-root escape rejection, output failure, clean syntax checks, semantic
success, semantic diagnostics, unsupported-subset rejection, closed native-build options,
cross-platform artifact-path rejection, no-overwrite promotion, symbolic-link escape rejection,
and ineligible-host failure before artifact creation. Stable build tests additionally cover closed
boolean flags, source/package failure separation, missing root entry, package-derived output naming,
replace-on-success publication, empty-image rejection, and staging cleanup. The configured Windows
smoke runs stable `check --locked`, a locked release build, and a locked release run through a
separately built `fairyc.exe`, requiring output-free program status 42 and an unchanged retained build
artifact. Direct stable-run proof additionally covers development and release status propagation,
normal-result/trap collision separation at status 131, division-by-zero trap classification,
canonical-lock restoration, artifact immutability, exact safe stdout with status zero, and generated
starter status zero. Unit tests cover stream identity, preceding program stderr, mismatched receipt-like
text, and stdout/stderr writer failures.
Stable explain tests execute every catalog entry through the exact three-line CLI shape after the
diagnostic package proves canonical unique lookup and exact parity with both owning Markdown tables;
they also cover all namespaces, malformed and unknown codes, missing roots, help, and output failure.
Stable targets tests freeze the current complete human report, canonical target and missing-evidence
ordering, indentation, separation, final newline, missing-root independence, argument rejection,
help, output failure, and absence of host-local paths while target tests own maturity invariants.
Stable emit tests run one manifest-owned workspace through both declared targets, require repeated
byte identity, LP64/LLP64 `long` assertions, raw C11 stdout, canonical lock update/verification,
stable source/package/capability/output failure separation, no experimental wording, and no build
directory. Backend and fixture tests continue to own the translation-unit contents and safety rules.
Stable workspace-test coverage separates its oracles: loader tests own production/test source
isolation, exact `tests/` path identity, overlap/case/symlink containment, rebuilt dependency lookup,
and ephemeral digest behavior; declaration tests own `E9001`; compiler tests own verified-MIR
admission; interpreter tests own fresh zero-argument `void` execution, comparison result typing,
trap categories, and entry rejection; CLI tests own canonical ordering, continued execution after a
trap, lock behavior, report grammar, exit codes, output failure, zero-test success, and absence of
retained build artifacts. Listing tests separately require canonical identity output, no execution,
complete-overlay compiler admission, incompatible-option rejection, zero-test output, and writer
failure. Native harness tests additionally require a collision-free private symbol,
MIR re-verification, one reachable i32 C entry, preserved order, and continuation after an exact trap
receipt. The standalone Windows smoke runs the checked-in semantic test, an actual failing-then-
passing native pair selected by reverse-ordered exact identities, the checked-in compiler-admitted
test list, and the checked-in exact selected
native test at low and optimized profiles through a
separately built compiler; each native image passes PE and unwind inspection before execution.
Target-description tests separately cover structural validation, power-of-two layouts, closed and
sorted scalar and feature sets, immutable lookup copies, unknown-target rejection, deterministic
host-independent JSON, false maturity claims, and top-level JSON-schema synchronization.
`internal/layoutfixture` additionally executes all 25 v0 layout cases across the 16-case Linux GNU
and nine-case Windows MSVC manifests with strict JSON decoding, exact target binding, checked
size/alignment arithmetic, declaration-order offsets and padding, closed error codes, sorted unique
case IDs, and pass/fail expectation-shape validation.

Experimental semantic CLI coverage also verifies explicit multi-file module sets, input-order
determinism, cross-file visibility diagnostics, duplicate rejection, bounded file counts, and a
repository-owned qualified-import example.

`internal/semanticfixture` executes all 61 cases in `spec/fixtures/semantic/manifest-v0.json`
against name resolution, typed HIR, and local safe-borrow Flow. It verifies exact failure phase,
complete diagnostic set, related spans, selected resolution coordinates, selected canonical type
facts, public inert-struct construction, explicit public resource factories, cross-module resource
literal `E0103`, borrow origins, last uses, escape facts, loop-context diagnostics, handled annotated returns,
ignored-call `E4202`, and invalid-declaration `E4201`. `internal/resourcefixture` executes all 12 cases
in `spec/fixtures/resource/manifest-v0.json`, comparing complete-root states, cleanup obligations,
and the complete deterministic `E11xx` set after parser, resolver, and type setup.
`internal/packagefixture` executes all 9 cases in `spec/fixtures/package/manifest-v1.json` through
strict duplicate-key and unknown-field rejection, contained complete-source loading, package-local
module parsing, exact instance-class comparison, direct-binding validation, and deterministic
lookup or rejection comparison. The reusable `internal/packagegraph` engine owns exact-instance
deduplication and current-plus-direct-only visibility. `internal/regionfixture` consumes that same
validated package case rather than re-decoding a partial graph contract.
`internal/regionfixture` executes all 6 cases in `spec/fixtures/region/manifest-v1.json`, compares
hidden allocation origins and complete `E31xx` diagnostics, and admits the lexical Arena case only
after proving its root, dependency, lookup, instance class, provider role, source root, and exported
`std::alloc::Arena` against the referenced package graph. `internal/resultfixture` executes all 24
cases in `spec/fixtures/result/manifest-v0.json`, comparing contextual leading-dot construction,
exhaustive shallow switching, handling modes, canonical payloads,
single-evaluation facts, and complete `E41xx` diagnostics. `internal/mirfixture` executes all 13
cases in `spec/fixtures/mir/manifest-v0.json` after Flow, mutable-local MIR lowering, and MIR
verification, comparing source-linked evaluation, call, move, Region, loop-control cleanup, Result-branch, trap, and
eliminated-form facts. `internal/mirverifyfixture` constructs all 10 named verifier seeds, requires
each baseline to pass, clones it, applies exactly one typed mutation, and proves all 22 cases return
  the complete expected one-code result. `internal/interpreterfixture` executes all 94 interpreter
v0 cases through source loading, Flow, verified MIR, and the bounded interpreter, comparing exact
`i32` returns, scalar and resource-bearing `Option` and `Result` construction, every built-in switch
variant, both Result `?` paths, canonical trap categories, resource construction, whole-root call
moves, ordered resource reassignment, consuming-close ownership, active-variant cleanup,
Region fallthrough, direct-return, Result-propagation, nested, break, and continue reset ordering, source-linked drop traces, checked arithmetic, matching-type bitwise operations, and left/right-shift outcomes, exact fixture-only output bytes and order, or the stable interpreter step/output limits
category. `internal/cbackendfixture` executes all 50 cases in
`spec/fixtures/c-emission/manifest-v0.json` through source loading, Flow, verified MIR, interpreter
outcome comparison, and deterministic C11 emission, comparing target, dialect, canonical function
names, feature families, LF-only path-clean text, and repeated byte identity. It does not invoke a C
parser, compiler, assembler, linker, executable, sanitizer, layout probe, or ABI probe. Native C
behavior belongs to `internal/cbackendnative`, which consumes the same emitted units through the
shared C driver and scalar runtime. ABI and complete-runtime runners remain separate evidence
requirements. `internal/architecturecontract` parses every repository Go import and owns the
mutation-sensitive one-way compiler-to-Atlas dependency oracle. `internal/semanticsnapshot` owns the
in-memory Atlas producer tests; publication, current-source acceptance, and package-local query
evidence are owned downstream. `internal/symbolidentity` owns
the provisional identity byte-vector, stability/sensitivity, path validation, strict text form,
package-UID nominal normalization, and visible/full collision regressions.
`internal/atlasmap` owns real-workspace logical planning tests for exact compiler identity,
one-interface-per-package closure, algorithm-rich target-independent headers, complete package
partitioning, namespace references, stable omissions, deterministic ordering, and detached
accessors. Its transport tests additionally own exact canonical bytes, the frozen checksum vector,
bounds, strict fields, corruption rejection, semantic cross-record validation, and declaration-to-
SymbolId rederivation. Store tests own opened-root containment, immutable convergence, current-source
matching, corruption/conflict rejection, retention bounds, temporary recovery, and process-kill
outcomes. Query tests build a real three-function workspace, require exact-current construction,
visible/full SymbolId equivalence, deterministic detached `overview`/`show`/`around` results, local
breadth-first traversal, record/relation/depth/selection-byte truncation, explicit dependency and
incoming-map omissions, and malformed request rejection. Workspace-call-join tests independently
require caller-order-invariant exact dependency SymbolId resolution, bidirectional root/dependency
lookup, local-call classification, detached results, complete partition/expectation/dependency sets,
stale-map rejection, missing-target rejection, and explicit package/call budgets. Package-local
query tests themselves do not consume that ledger, and neither lane exercises a public workspace
graph traversal, source excerpt, or rewrite cost. `internal/workspacecallcatalog` separately owns
canonical no-newline output, independent checksum recomputation, package-qualified visible/full
subject identity, outgoing/incoming/both selection, exact empty results, construction budget/usage
echo, final byte bounds, and host-path/source-body exclusion. `internal/cli` tests additionally drive
that writer from one compiled workspace and require ordinary/locked byte identity, exact root/member
call direction, closed package/symbol/direction/budget flags, atomic join-budget failure, unknown
package/symbol rejection, and output-failure classification.
`internal/contextpack` separately owns
canonical output bytes, checksum recomputation, exact query payload selection, final byte bounds,
query-error propagation, empty text/warning sections, and host-path/source-body exclusion. It does
not imply a decoder or untrusted-pack acceptance boundary. `internal/cli` tests now discover a real
function through the exact catalog and require visible-ID `show`, full-ID `around`, ordinary/locked
byte identity, deterministic local traversal, closed direction/relation/depth/budget flags, unknown
symbol rejection, and output-failure classification.

`internal/cstdlib` separately embeds the provisional SystemHeap implementation and its closed C
probe. Ordinary Go tests prove deterministic, LF-only source ownership and forbid raw `realloc` or
output. The Linux native runner compiles and executes the probe with Clang and GCC at low and
optimized profiles, checking zero-size normalization, fundamental alignment, growth and shrink
prefix preservation, unsupported-alignment rollback, release, and invalid output rejection. These
four configured executions are not source-level standard-library fixtures, deterministic OOM
injection, or hosted evidence until a Linux receipt is observed.

The Buffer probe adds ordinary append, no-op reserve identity, relocating initialized-prefix
self-append, length-overflow rollback, zeroed growth, truncate, clear, malformed state, and drop.
Its probe-only SystemHeap adapter forces nonempty initialization, reserve, self-overlap append
growth, and zeroed-resize growth failure. It verifies unchanged pointer, capacity, alignment,
length, initialized bytes, and live-allocation count, then proves recovery bytes, zero-fill, and
final cleanup without freezing the geometric capacity choice or reading spare storage. The raw
caller and target-specific callee execute as separate MSVC translation units at both profiles.

A separate generated-Fairy Buffer source probe protects the ABI connection rather than duplicating
the C-only behavior matrix. Its success case requires dependency-ordered nested `@repr(C)`
definitions, exact 24-byte child and 32-byte containing layouts, real initialize and zeroed-resize
mutation, observable zero bytes, and normalized drop. Backend unit boundaries reject recursive
value dependencies and a non-`@repr(C)` nested field.

The Arena probe adds single-active-Region rejection, zero-initialized and copied byte placement,
unsupported-alignment rollback, zero-size normalization, active-drop rejection, reset generation
advance, stale-use rejection, and final drop. Its probe-only SystemHeap adapter separately fails the
payload allocation and tracking-node allocation, requires unchanged outputs, observes rollback to
zero live allocations, and proves normal recovery still exposes zeroed bytes. It does not freeze
node size, allocation grouping, or an Arena growth policy.

A separate generated-Fairy Arena source probe protects the exact cross-translation-unit record
contract. It requires self-linked node pointers, the nested SystemHeap block, provider-proven
`uint64_t`/`unsigned long` identity, 32/24/32-byte node/Arena/Region layouts, a `void **` output slot,
observable zeroed bytes, released nodes, invalidated Region state, and reset/drop generations.

The Windows target product smoke covers a narrower source-level Arena lane. Its private FY records
use target-width `usize` for the C unit's eight-byte generation storage, call the five exact lifecycle
symbols, require generation 1 after initialization, one active Region and non-null allocation/node,
generation 2 plus cleared Region state after reset, and generation 3 after drop at low and optimized
MSVC profiles. It does not dereference the output pointer. A separate raw Arena caller and
target-specific callee now supply zeroed/copy byte observation plus the two-stage allocation-failure
matrix through separate MSVC translation units at both profiles.

The text-view probe validates mixed-width text and rejects stray continuation bytes, two overlong
families, surrogate encodings, values above U+10FFFF, truncation, and obsolete leaders. It checks
valid and invalid byte-boundary slices, empty normalization, invalid pointer/length pairs, and
unchanged outputs on failure. It is byte-level UTF-8 evidence, not Unicode normalization or
grapheme behavior.

The owned-text probe checks String copy, immutable view, ordinary and self-source append, drop,
failed Buffer conversion with unchanged ownership and destination, successful consuming move, and
CString construction, bounded CStr view, exact trailing terminator, interior-NUL rejection,
unchanged failure output, and empty CString. It does not assert source Result layout or locale
conversion.

A separate generated-Fairy CStr source probe protects the exact cross-translation-unit view record
and real bounded validator call. Its success case requires a nonempty terminated byte sequence and a
logical length excluding the terminator. Its two failure cases distinguish interior NUL from a
missing terminator and require the previously valid output view to remain byte-for-byte observable.

A compiler-owned C-string source probe protects the literal path separately from that record-based
validator. The Fairy caller must preserve exact `fairy\0` storage through semantic analysis, MIR,
and C emission, perform the dedicated unsafe immutable-byte-pointer extraction, and let a separate C
callee read all six bytes. A second product-build smoke carries nonempty `ref CStr` storage through the
ordinary CLI without a companion source. These are execution and artifact oracles, not proof of a
public safe CStr API.

A compiler-owned byte-string source probe separately protects exact non-terminated `fairy` bytes,
the private non-null `ref u8[]` `{ data, length }` view, length five, typed-local unsafe data-address
extraction, and a separate C callee reading all five bytes. The emitter test also covers embedded
NUL, duplicate storage, and the length-zero non-null sentinel. A second product-build smoke carries
the view through the ordinary CLI. These are bounded-view transport oracles, not source indexing,
slicing, a public slice ABI, or safe foreign-borrow evidence.

The generated-Fairy owned-text source probe protects the success lifecycle through the exact nested
records and by-value operation result. String must copy `Fairy`, expose it, append it to
`FairyFairy`, expose the new bytes, and drop to canonical empty storage. CString must copy the text,
expose a five-byte CStr view followed immediately by one NUL, and drop to canonical empty storage.
Every high-level and allocation status must be zero. Existing C-only cases retain ownership of the
negative and rollback matrix rather than duplicating it in the source probe.

The generated-Fairy Linux File source probe protects exact result, File, and open-option records
plus by-value operation-result returns. It writes and synchronizes `FairyFile`, consumes close,
requires repeated close to report closed, reopens, performs three three-byte partial reads, observes
EOF with count zero, closes, and compares the reconstructed bytes and normalized handle. The C-only
Linux File probe retains direct partial-write, invalid-option, path, no-progress, `EINTR`, and forced
close-failure ownership rather than duplicating those negatives in the source probe.

The target-neutral Linux console probe links its production POSIX adapter as a separate translation
unit under Clang/GCC and both profiles. It proves empty and invalid input, exact progress through
redirected stdout and stderr, and unavailable status after closing each descriptor without emitting
probe bytes. The
safe SDK wrapper and package integration are proved separately by Go tests and the Windows native
workspace smoke; ADR 0214 records the corresponding Linux hosted execution.

The Windows File probe keeps its production component in a separate translation unit and executes
both local MSVC profiles. It creates and truncates a file, writes a five-byte prefix once and the
remaining four bytes through write-all, rejects an
oversized length without mutation, shrinks to five and extends to nine without moving the cursor,
rewrites the unspecified extended bytes, observes size and
cursor, rejects an overflowing absolute seek without moving it, seeks to byte 5 and back to zero,
syncs, consumes close, rejects repeated close, reopens, seeks and reads the final four bytes, returns
to zero, reads in three-byte chunks, observes EOF, and compares the bytes. It then steals and closes
the underlying handle so size, kind, identity, exact read, and explicit close must return `ERROR_INVALID_HANDLE` while close
still normalizes ownership, and finally checks best-effort and
repeated drop. Both objects, the linked PE, and its unwind records must pass the same structural
inspectors before execution. A distinct two-module FY example then exercises typed open errors,
safe resource borrows and partial/exact reads over fixed arrays, shared size, kind, and identity observation,
one-call write success and read-only failure, write-all/position/absolute-seek, open-or-create creation/preservation, exclusive create/collision/unchanged reopen, seek-resistant append, existing-file read/write update, cursor-preserving length change,
sync/read/EOF, consuming close, and implicit drop at both profiles.
ADR 0264 promotes the ownership and I/O slice to compiler-bundled public `std::fs`; ADRs 0269 and
0270 add positioning and size, ADR 0271 adds update-open, ADR 0272 adds length change, ADR 0274
adds non-truncating open-or-create, and ADR 0275 adds exclusive create with exact collision and
unchanged-reopen evidence in the same raw and public generated-Fairy proofs. ADR 0276 adds
missing-file append creation and repeated exact EOF writes after explicit repositioning. ADR 0277
adds same-handle disk/character classification, including `NUL`, plus closed/invalid-handle raw
failures. ADR 0278 adds complete fill, unexpected-EOF completed-prefix and untouched-suffix
evidence, empty success, and exact-read closed/invalid-handle failures. ADR 0279 adds complete
volume-plus-128-bit identity equality for two live handles to the same disk file, inequality for a
second live file, null-output rejection, and unchanged closed/invalid-handle failure output. Portable
paths, relative/end-based seek, Linux public linkage, general
mutable buffers, and hosted evidence remain open.

The standard-library product-build smoke is distinct from source-call companion probes. Its `.fy`
source calls the real empty-input UTF-8 predicate without companion C, so success requires emitted
used-symbol metadata, minimal component resolution, the ordinary scalar-runtime build path, native
link, and exit 42 under Clang/GCC at low/optimized. Uncalled declaration filtering and component
dependency closure remain host-independent Go tests.

## Oracle order

1. Product specification and accepted ADRs define the intended boundary.
2. Versioned language and MIR fixtures become the executable contract.
3. The MIR verifier rejects structurally invalid compiler output.
4. The MIR interpreter provides the first executable semantics oracle.
5. Generated C is compared against the interpreter under the C compilers and profiles declared by the target fixture.
6. Later backends are compared against the same interpreter and C evidence rather than defining new behavior.

## Fixture classes

| Class | Proves |
| --- | --- |
| Parse pass and fail | `spec/fixtures/parser/` currently proves grammar acceptance, precedence trees, literal cooking, lossless reconstruction, recovery provenance, byte spans, and deterministic diagnostics through the Go parser runner |
| Semantic pass and fail | `spec/fixtures/semantic/` currently proves all 61 v0 cases: public inert-struct literals plus explicit resource factories, cross-module resource-literal `E0103`, explicit source-location expansion and E4401/E4402 boundaries, bounded scalar compile-time evaluation and E4301 through E4305, narrow callable `@must_use` handling plus `E4201`/`E4202`, classic scalar `for` typing, bounded lexical `defer`, loop-local initializer scope, and bounded copy fixed-array Option/Result construction, switching, and propagation, resolution coordinates, canonical types, radix-correct integer typing, `E1013` range rejection before MIR, contextual shared and mutable fixed-array byte views, local and branch-reachable last-use loans, straight-line, branch, and loop-carried safe-borrow reassignment, header/`continue`/`break` origin joins, exact interprocedural borrow-return parameter remapping, selected-call-owner protection, borrow return origins, loop-control context rejection, admitted scalar foreign calls with distinct source/link identities, producer-to-consumer transport for all sixteen exact TargetSpec C scalars, explicit exact `c::int <-> i32` casts, `E5101` unsafe boundaries, `E5102` safe-reference exclusion, `E5106` malformed or duplicate link-name rejection, canonical callable and resource effect sets, caller-superset enforcement, implicit-drop effect propagation, whole-move exemption, `E7101` declaration rejection, `E7102` call and cleanup rejection, phase ownership, related spans, and deterministic cascade suppression |
| Module pass and fail | File-to-module identity, explicit aliases, forbidden import forms, cycles, and source visibility |
| Type and initialization fail | Inert/resource separation and initialized-local rules |
| Resource flow pass and fail | `spec/fixtures/resource/` currently proves all 12 v0 cases: whole-root moves, consuming tagged switches, live replacement, dead-slot reinstallation, scope obligations, branch, normal-loop, break-edge, and continue-edge states, and deterministic `E11xx` spans; conditional close ownership stays a runtime/MIR concern |
| Borrow-origin pass and fail | Local, branch, and loop-edge last-use loans, canonical reassignment generations, exact internal-call return summaries, selected-owner conflicts, single-origin return, and storage bans |
| Region pass and fail | `spec/fixtures/region/manifest-v1.json` currently proves all 6 v1 cases: parameter and package-resolved local origins, permitted parameter-origin returns, `E3101` local escape, capability restrictions, and no `NeedsDrop` placement |
| Result pass and fail | `spec/fixtures/result/` currently proves all 24 v0 cases: contextual leading-dot scalar construction, exhaustive shallow switching, return, discard, propagation, local and parameter transfer, handled reassignment, all-branch handling, partial-branch rejection, loop back-edge, break, continue, and overwrite rejection, early-return obligations, exactly-once evaluation, and error compatibility; active resource payload traces are covered by MIR and interpreter fixtures |
| MIR normalization | `spec/fixtures/mir/` currently proves all 13 v0 cases through the Go MIR runner: source-linked evaluation order, call successors, classic scalar `for` and bounded lexical `defer` elimination, ownership transfer including a resource-bearing Option switch, RHS-before-drop resource reassignment with live/dead destinations, Region placement and reset ordering on fallthrough, direct return, Result propagation, nested, break, and continue exits, cleanup order, Result branching, division and checked-arithmetic trap paths, and eliminated source forms; graph identity remains excluded |
| Resolved package graph | `internal/packagefixture` currently executes all 9 v1 cases in `spec/fixtures/package/`: the eight historical instance-equality, feature-split, direct-visibility, module-isolation, ambiguity, transitive-hiding, duplicate-binding, and explicit-standard-library cases plus post-deduplication dependency-cycle rejection. The isolation case additionally converts the validated graph to semantic input and completes the shared compiler coordinator through verified MIR with distinct build-local HIR package identities; the v0 manifest remains directly readable. |
| Built-in tagged resources | `.ok`, `.err`, `.some`, discard, switch, propagation, early return, and nested active-payload drop traces |
| MIR verifier mutation | `spec/fixtures/mir-verifier/` currently proves all 10 named baselines and 22 isolated mutation cases, including missing resource definition, tagged-operation type, and internal call target `V1001`, fallthrough and break Region reset `V1403`, and missing checked-arithmetic `V1501`, through the Go verifier runner without accepting arbitrary MIR as source truth |
| MIR interpreter run, trap, and limit | `spec/fixtures/interpreter/` currently proves all 94 v0 cases: literal return, classic scalar `for` including continue-through-post, bounded lexical `defer` LIFO execution and immediate capture, direct `i32` minimum folding, scalar fixed-array construction/calls/value-copy isolation/contextual immutable and mutable byte-view borrowing/checked reads and stores, scalar call arithmetic, always-checked assertion success and `assertion-failed`, borrowed-view length and emptiness metadata, checked end-exclusive byte slicing with explicit, omitted-start, omitted-end, omitted-both, empty-at-end, reversed, and oversized ranges, checked `i32`, `u8`, `u32`, and `usize` addition/subtraction/multiplication/division/remainder across trapping and non-trapping boundaries, checked negation, matching-type bitwise operations, signed arithmetic and unsigned logical right shift, checked matching-type left shift with invalid-amount-before-overflow precedence, exact width boundaries, typed scalar equality and integer ordering, boolean branching and logical not, source `&&` and `||` short-circuiting, scalar `while`, nearest-loop break and continue, scalar and resource-bearing `Option` and `Result` construction, every built-in switch variant, scalar and resource-bearing Result `?` paths, checked byte-view indexing and indexed-store `bounds` traps, value-preserving integer casts, resource construction and field reads, whole-root and tagged-payload moves, resource returns, ordered live/dead/self reassignment, consuming-close success/failure ownership, nested active-variant cleanup, deterministic drop traces, lexical Region placement and reset on fallthrough, conditional fallthrough, direct return, Result propagation, nested, break, and continue exits, exact manifest-authorized in-memory stdout/stderr bytes, order, and retained limit prefix, exact division-by-zero, signed-division-overflow, integer-overflow, invalid-shift, and assertion-failed trap identity, and bounded step/output-limit termination; user cleanup-body effects and host I/O remain deferred |
| Drop elaboration | Partial construction and cleanup-suffix merging; loop `break` and `continue` are fixture-backed |
| C emission | `spec/fixtures/c-emission/` currently proves 50 verified scalar, classic-`for`, lexical-`defer`, scalar-fixed-array, always-checked-assert, borrowed-view-metadata/call/index/store/slice, checked-unsigned-arithmetic/division, checked left/right shift, matching-type bitwise, typed-comparison, restricted inert-enum, flat-resource, copy-tagged, and resource-tagged source-to-C11 cases, including exact explicit `c::int <-> i32` casts, private assignable array wrappers, contextual zero-copy shared and mutable fixed-`u8`-array byte-view borrowing, internal array calls, checked reads and fixed-array or mutable-view stores, checked pointer derivation for explicit and omitted-bound end-exclusive byte subviews, private user-enum tag/payload/switch transport, inert-enum fields inside private structs, acyclic inert enums inside user-enum payloads, direct copy-scalar or acyclic aggregate-bearing `Option`/`Result` fields inside private aggregates with dependency-ordered declarations, and copy-only tags over compact scalar tags, namespace-safe public `@repr(C)` tags with `_lowercase` members, private scalar/inert-struct, user-enum, copy fixed-array, and resource-bearing `Option`/`Result` transport including fixed-array success/error propagation and user-enum `Option` Some and None, direct move-only resources over Fairy copy scalars or raw pointers, shared and mutable borrowed-resource field reads, one verifier-linked pure nonempty-drop body, scalar, resource, fixed-array, and user-enum-error typed propagation, UTF-8 and byte-string metadata reads, borrowed-view internal parameters, deterministic target/dialect/function/feature metadata, target-layout assertions, LF-only path-clean output, repeated byte identity, and schema/manifest feature agreement. Unit regressions additionally reject C implementation-reserved identifiers, generated-header macros, backend-owned tag or macro names, unsupported tagged fields, recursive graphs, and public tagged fields before emission, and require payload aggregate, tagged, immutable byte, mutable byte, and UTF-8 view transport declarations to precede their first by-value or prototype use; fixtures alone do not prove C syntax acceptance or native behavior. |
| C differential | The configured Linux runner covers all 50 C-emission cases across Clang/GCC and low/optimized profiles, ASan+UBSan production-unit checks, exact TargetSpec scalar crossings, aggregate-result ABI paths, SystemHeap, Buffer, Arena/Region, text, owned String/CString, Linux File, and compiler-owned literal storage. Report v32 additionally configures four separate-translation-unit executions of the target-neutral POSIX console adapter. Each execution covers empty and invalid stdin input, an exact five-byte pipe read with untouched suffix, EOF, unavailable stdin, and the existing empty, invalid, redirected-write, and unavailable-stream behavior on stdout and stderr. Generated callers and ordinary product builds retain separate oracle classes. ADR 0214 records the first hosted Linux success; broader output, wider returns, effect-bearing user cleanup bodies, public ABI, aggregate parameters, broader safe standard-library APIs, and the complete product runtime remain deferred. |
| Target description | `internal/target` proves host-independent Linux GNU and fail-closed Windows MSVC facts without promoting target maturity. Explicit target commands retain bounded source closures and distinct result-versus-program-output contracts. The legacy two-module Windows example confines its unsafe target symbol; the manifest-owned `examples/std_io_quickstart` instead imports compiler-bundled `std::io`, contains no unsafe or platform name, and stable run/build emit exact text, byte-view, and decimal `i32` output on separated stdout/stderr with status zero. Direct execution of the retained product proves `INT32_MIN`, stdin EOF, a short exact echo, and one eight-byte prefix from longer input; stable run forwards the short input through both in-process and standalone compiler boundaries. Raw storage, File, text, layout, ABI, COFF, PE, and unwind probes remain separate evidence layers. ADR 0214 records the first generic hosted native receipts; broader Fairy FFI, general formatting/text-or-line-input/terminal APIs, input routing for other native commands, and complete independent ABI/runtime evidence remain absent. |
| Target layout | `spec/fixtures/layout/` currently proves 16 host-independent Linux GNU v0 cases for language and C scalars plus nine Windows MSVC cases for LLP64-sensitive scalars, data and function pointers, explicit C-enum bases, a nested aggregate, and absent POSIX `c.ssize_t`; the 11 valid Linux cases generate one recursive C11 plus explicit POSIX-typedef probe executed in 4 Clang/GCC low/optimized hosted runs recorded by ADR 0214, while the eight Windows pass cases generate a separate non-POSIX C11 unit compiled and executed twice by x64 MSVC through the hosted smoke; full Windows scalar coverage remains deferred evidence |
| C ABI classification and scalar source FFI | `spec/fixtures/c-abi/` proves 17 Linux SysV v0 cases for INTEGER/SSE/MEMORY eightbytes, register exhaustion, whole-aggregate rollback, and hidden results plus 15 Windows Microsoft-x64 cases for shared ordinal slots, fifth-argument stack transport, small-record integer transport, aggregate indirection, and hidden-return slot shifting; the 14 Linux pass cases generate 56 separate Clang/GCC crossings, while the 12 Windows pass cases generate separate caller/callee C11 units that x64 MSVC crosses at low and optimized profiles. Distinct `@link_name`-mapped `i32 -> i32`, four individual C-scalar, one remaining twelve-type family, one register-plus-hidden struct-result, and one exact SystemHeap block source fixture each cross generated Linux callers and hand-written or project-owned C callees in both directions and profiles for 4 executions. The shared struct-result fixture also emits a Windows-profile Fairy caller and crosses its 8-byte register-sized and 20-byte hidden result from a hand-written C callee in 4 same-MSVC executions across C11 and C17 consumer modes. ADR 0214 records the first hosted execution of both lanes; the Windows source probe remains same-compiler rather than independent/register-level evidence, and by-value aggregate parameters, retained callbacks, exports, C-scalar value semantics, and C-scalar raw-pointer lowering remain absent. |
| Package instance identity | `spec/fixtures/package-identity/manifest-v1.json` fixes four exact canonical byte and full SHA-256 vectors across empty and nonempty features plus ambiguous component boundaries, then rejects unsorted, duplicate, and empty feature sets with stable reasons. Unit tests additionally reject invalid UTF-8 and noncanonical serialized IDs. A semantic integration test proves source `pub` survives in typed HIR across all five current item categories while generated cleanup glue remains private. These fixtures do not prove a manifest, lockfile, solver, source archive, cache, registry, or interface hash. |
| Public interface invalidation | `spec/fixtures/module-interface/manifest-v3.json` compiles twelve complete source variants and freezes one full SHA-256 baseline, with v1 and v2 retained historically. Private implementation, ordinary public body, declaration order, private resource representation, and private must-use, private module-constant, and private source-location state preserve it; public signature, public struct field order, foreign link-name, and public must-use changes invalidate it. Unit tests additionally prove build-local package renumbering is inert, dependency coordinate changes invalidate, effects participate, and private local types cannot leak. Inline-body hashing, serialized module interfaces, manifests, lockfiles, and cache records remain absent. |
| Package source digest | `spec/fixtures/package-source/manifest-v1.json` freezes one full SHA-256 baseline and proves file input order is inert while content, canonical path, and line-ending changes invalidate. Unit vectors fix exact length-prefixed bytes and prove the streaming compute path matches them without reordering caller input; negative tests reject empty or duplicate graphs and unsafe, unclean, non-FY, or invalid-UTF-8 paths. Discovery, archives, manifests, lockfiles, and cache storage remain absent. |
| Source-file identities | Unit vectors freeze domain-separated exact-content and line-map bytes, LF-derived zero-based starts, CRLF and final-LF behavior, empty input, strict lowercase full-digest text, and content-only versus byte-geometry invalidation. Workspace and snapshot tests require canonical detached source/test records and complete file/module coverage. Current-source map validation remains future evidence. |
| Whole-package cache and artifact prerequisites | `spec/fixtures/package-cache/manifest-v5.json` freezes one full v2 `fycache2` compatibility-vector key using current `fyif3` dependency inputs and proves compiler-build, source, dependency-interface, target, feature, and profile invalidation; fixture manifests v1 through v4 remain historical vectors. Unit tests reject noncanonical sets and duplicate dependency identities, then prove mandatory byte/entry bounds, LRU eviction, deep-copy isolation, hit/miss accounting, nondestructive oversize rejection, and exact-record quarantine. `DiskStore` tests add opened-root traversal and external-symlink rejection, immutable concurrent publication, digest and canonical-record checking, corrupt/conflicting-generation safe misses, stale-temp recovery, deterministic entry/byte eviction, and compare-and-delete preservation. Platform publication uses handle-relative Windows rename plus final-file flush or non-Windows directory sync. Separate processes prove identical-writer convergence, conflicting-writer rejection, cleanup of a temporary left by process exit, a pre-publication kill yielding only a miss, and a post-publication kill yielding one verified generation. `internal/mircodec` proves bounded deterministic whole-program MIR encoding, exact decode, and verifier re-entry. `internal/packageartifact` proves stable function identities and type context across package renumbering, explicit owned/import bindings, independently verified process-local slices, package-scoped context closure, equality-checked mixed-context merge, unreferenced-private-declaration byte stability, bounded canonical stable body and package codecs, internal/foreign call rebinding, fresh package/function/foreign/local identity reconstruction, verifier re-entry, result-42 workspace execution, byte-identical re-encoding, conflicting-definition rejection, corrupt body/ID/import rejection, missing-provider rejection, exact-context drift rejection, process-codec isolation, expected-interface and package-instance checking, verified-hit accounting, and rejected-entry quarantine. Independent workspace Sessions and separate writer/reader test processes over one disk root require complete stable hits, skip a second lowering, rebind and verify fresh MIR, execute result `42`, and miss after compiler-build identity drift. `internal/buildidentity` proves byte-sensitive executable identity and parent/child equality. CLI tests prove cold/warm persistent invocations, exact JSON/schema parity, explicit policy rejection, and corruption recovery. `internal/buildmeasure` rejects incomplete or misleading no-cache, cache-edit, standalone-session, paired persistent-process, and retention-growth reports. The configured Windows smoke drives eight editor-shaped requests, nine fresh-cache cold/warm pairs, and one nine-process single-cache retention staircase through one separately built compiler. The current staircase requires zero record quarantine, three evictions under a six-entry ceiling, and three complete no-op all-hit observations; it still blocks a default-policy claim because private dependency edits over-invalidate both packages and representative growth evidence is absent. Physical device power-cut evidence, representative workspace growth, default limits, a declared reference machine, and an accepted disk speed or memory budget remain absent. |
| Project manifest | `spec/fixtures/project-manifest/manifest-v1.json` executes eight complete `fairy.toml` files: minimal and explicit-workspace passes plus unknown field, duplicate key, missing feature set, path escape, feature ordering, and schema rejection. Unit tests additionally cover semantic output, deterministic dependency sorting, required fields, UID/name/exact-SemVer/language/path validation, invalid UTF-8, and the 1 MiB input bound. |
| Local workspace loader | `spec/fixtures/workspace-loader/manifest-v1.json` executes complete directory trees for feature-distinct graph success plus version, UID, exact-case, and cycle rejection. Unit tests additionally cover nested/overlapping members, unknown paths, language mismatch, empty and parser-invalid sources, symlinks, source digests, deterministic graph shape, exact conditional SDK injection, reserved SDK UID/binding/module-namespace rejection, missing `std::*` modules, and package-aware verified MIR. The explicit semantic-test overlay separately proves ordinary-load isolation, absent/empty `tests/`, exact lowercase and disjoint containment, path-to-`tests::` module identity, dependency lookup rebuilding, and ephemeral digest separation. Lockfile I/O and target C linkage remain separate. |
| Generated lockfile and workspace commands | `internal/lockfile` round-trips feature-distinct and SDK-importing workspaces with exact identity/source-origin revalidation and stale comparison. Stable check/build/run/test preserve their separate lock, artifact, stream, and interpreter contracts. The Windows smoke covers ordinary and locked workspace paths through both in-process compiler logic and a separately built `fairyc.exe`; it additionally runs `examples/std_io_quickstart` with forwarded short stdin through both compiler boundaries, builds it in ordinary and locked modes with exact separated stdout/stderr and status zero, then directly executes the artifact with EOF, short input, and an over-capacity input. A unit regression requires pre-launch source failure to consume zero input reads. Persistent build-cache defaults remain absent. |
| Windows PE executable and COFF object/archive structure | `internal/pecoff` validates bounded synthetic positive and malformed AMD64 PE32+ images, raw COFF objects, and closed Microsoft archives without host loading. `spec/fixtures/coff-object/manifest-v0.json` adds two exact raw-object vectors whose strict runner freezes ordered primary symbols, REL32 targets and types, ANY COMDAT, and associative COMDAT parent selection; mutations independently kill section, symbol, and relocation expectations. Relocations to auxiliary records, unknown AMD64 types, missing or duplicate COMDAT definitions, invalid selections, and invalid associative parents fail closed. ADR 0232 adds a separate synthetic object/image pair whose mutations kill relocation type, target, opcode, addend, displacement, receipt, and source-range drift; the configured smoke repeats the direct `CALL rel32` formula proof across separately compiled MSVC units at low and optimized profiles. ADR 0233 validates archive signature, alignment, both linker members, preferred indexes and symbol bounds, exact object members and raw candidate values, demanded/unused map presence, pick-any metadata, map owners, final addresses and value, and read-only receipt semantics. It repeats four actual MSVC input-order executions across low and optimized profiles. ADR 0234 preserves relocation section indexes and adds synthetic mutations for exact `ADDR64` and all three unwind `.pdata` `ADDR32NB` formulas, internal-label and `.xdata` symbol identity, map ownership, independent addends, and final values; the same two relocation-probe executions prove them at low and optimized profiles. ADR 0235 binds the selected archive function root to its direct associative `.pdata` and `.xdata` children, all three object addends, exact archive-member map owners, and final runtime-function entry in the existing four archive links. ADR 0236 proves two exact external ANY-COMDAT function parents reached the linker, binds the winner, requires the losing `.xdata` payload to be absent from every final runtime-function reference and file-backed section, and requires zero non-directory `.pdata` bytes. ADR 0240 adds a profile-specific handler-tail `ADDR32NB` object-to-map-to-image proof without adding an executable. ADR 0241 parses exact weak auxiliaries and proves one missing-alias fallback plus one strong override through final REL32 values and native execution in the existing two images. ADR 0242 normalizes ordinary and extended relocation tables, rejects malformed sentinel/count combinations and excess allocation, and proves one exact 65,535-record object whose final semantic REL32 controls the native result, with exact map ownership and low/optimized final displacement. ADR 0243 separately parses a bounded version-2 AMD64 bigobj header and 20-byte symbols, then proves its internal REL32, exact map owner, final displacement, and native result in the same two images. ADR 0244 freezes four generators by explicit inputs, deterministic replay, exact byte length and SHA-256, complete section and symbol receipts, and a contiguous relocation-run partition covering every parsed record. ADR 0245 proves NOLIBRARY suppresses an available member and LIBRARY extracts its separate member through archive indexes, an indivisible sentinel, exact map ownership, both final displacements, and native execution. The lane now inspects fifty-nine target or probe PE images and every intermediate object before cleanup. ADR 0214 is only the historical hosted smoke; a fresh same-commit receipt, whole compiler-object identity, remaining relocation formulas, weak chains/cycles and COMDAT/default-library interactions, non-unwind and broader archive/COMDAT policies, complete CodeView/PDB semantics, high-section and bigobj auxiliary/COMDAT evidence, and raw stored duplicates of the generated objects remain absent. |
| Windows x64 unwind structure and execution | `internal/pecoff` builds synthetic valid and malformed exception directories and applies the same parser to all fifty-nine MSVC target or probe PE images. `.pdata` ordering, executable ranges, aligned unwind data, bounded prolog/epilog structures, and the separate real exception-dispatch/virtual-unwind probe ran in the hosted smoke recorded by ADR 0214. ADR 0234 additionally binds one section-indexed object's complete three-field `.pdata` entry through exact internal-label and `.xdata` symbols plus independent ADDR32NB addends to its final runtime-function entry. ADR 0240 locates the aligned handler tail in `frame.obj`, requires the low `__GSHandlerCheck_SEH` or optimized `__C_specific_handler` ADDR32NB target, binds its raw addend and map owner to an executable final RVA, and then runs the unchanged dispatcher path. Version-2 epilog semantics, language-specific handler payload parsing, recursive chain equivalence and chained-tail relocations, Fairy/foreign-frame crossings, and a complete versioned unwind-fixture contract remain absent. |
| C safety | Overflow, division, shifts, evaluation order, aliasing, alignment, inactive payload, and sanitizer-assisted checks |
| ABI | Target layout, calls, callbacks, raw bindings, and boundary shims for candidate target profiles |
| Runtime and stdlib | Zero-size normalization, allocation failure, reserve rollback, UTF-8 invariants, partial I/O, Linux descriptor and Windows handle close consumption, and retained-pointer wrappers |
| Compatibility | Exact pre-1 matching, cache misses, runtime mismatch, diagnostic identity, stale maps, target mismatch, and relation-specific completeness |
| Atlas one-way package boundary | `internal/architecturecontract` parses all `.go` imports below `cmd/` and `internal/`, requires zero current violations, and requires exact deterministic failures for injected compiler and MIR imports while preserving only the explicit Atlas/export, CLI, and Windows-evidence consumers. It includes tests and inactive build-tag files but proves direct dependency structure only, not process isolation, package initialization, or downstream model safety. |
| Atlas semantic snapshot | Current unit and pipeline tests require Flow-plus-verified-MIR admission, complete package and file metadata, file/module bijection, build-local package-renumbering invariance, deep input/output detachment, stable ordering, structured path-free type identity, exact local calls/traps/unsafe/ownership facts, foreign-call-only partial completeness, unavailable relation disclosure, and fail-closed invalid MIR or package mappings. These tests alone imply no encoded map, current-source validation, or query evidence; those are separate `internal/atlasmap` lanes. |
| Atlas SymbolId | Current unit and compiler-pipeline tests freeze tagged length-delimited bytes, visible/full digest derivation, strict text forms, path/kind validation, tuple boundaries, body/span/version/build-number stability, coordinate/signature sensitivity, nominal package-UID normalization, deterministic ordering, pair integrity, and synthetic compact-ID collision rejection. No encoded map or released cross-version promise is implied. |
| Atlas logical map plan | Current real-workspace and unit tests require exact compiler identity, a complete public-interface-hash set, all independent algorithm identifiers, package-stable headers, exact namespace references, complete file/module/declaration/SymbolId/function assignment, deterministic package order, immutable accessors, and explicit omissions. Current-source validation, workspace call joining, queries, and whole-package rewrite measurements are owned by separate evidence lanes. |
| Atlas transport | Current real-workspace and mutation tests require repeated canonical JSON identity, exact schema/encoding/checksum algorithms, a frozen checksum vector, positive caller bounds, unknown/trailing/noncanonical rejection, canonical identities and record order, exact counts and ownership, bounded type depth, checksummed-payload revalidation, and declaration-to-SymbolId rederivation. Authentication, alternate encodings, compression, and representative rewrite performance remain future evidence; the current quickstart-only rewrite control and storage evidence are owned by separate rows. |
| Atlas store | Current real-workspace, storage, containment, and process-interruption tests require caller limits, opened-root relative paths, checksum-derived immutable names, identical convergence, close/reopen loading, exact compiler/interface/package/file/line-map acceptance, stale/incompatible/corrupt/conflict separation, corrupt quarantine, bounded eviction, expired temporary recovery, external-symlink rejection, and pre/post-publication miss-or-verified-hit outcomes. Default policy, public CLI behavior, physical power-cut retention, dependency joins, and query performance remain future evidence. |
| Atlas package-local queries | Current real-workspace tests require exact-current constructor admission, canonical visible/full SymbolId lookup, deterministic and detached overview/show/around output, explicit relation filters and directions, local breadth-first traversal, record/relation/depth/selection-byte limits, stable budget/filter/join omissions, and fail-closed malformed partitions, selectors, requests, and compatibility inputs. Experimental CLI tests additionally exercise exact catalog discovery into visible-ID show and full-ID around. Excerpts, dependency joins, representative latency, and workspace-wide graph claims remain future evidence; Context Pack output is owned by the separate pack row. |
| Atlas exact-current workspace call join and catalog | Current quickstart and local-call tests require every package to re-enter exact-current query validation, complete and equal partition/expectation sets, canonical sorted direct-dependency admission, exact observed call provenance, target declaration and public-visibility agreement, caller-order independence, package-qualified visible/full selector equivalence, bidirectional dependency callers, local-call classification, detached call and package-context results, and fail-closed package/call budgets, stale maps, missing maps, and missing targets. The join reuses the validated sorted dependency list for logarithmic target admission instead of rescanning every dependency for every call, then retains compact package context and SymbolId lookup state instead of complete package-local query indexes. Workspace-call catalog tests additionally require deterministic canonical no-newline bytes, independent checksum recomputation, visible/full identity, outgoing/incoming/both selection, successful empty selection, explicit package/call/output bounds, ordinary/locked identity, source-body and host-path exclusion, and closed CLI failure behavior. Package-local query output still does not consume the ledger; other relation families, traversal, persistence, stable commands, indirect calls, impact, and representative time or memory measurements remain absent. |
| Atlas exact symbol catalog | Current real-workspace tests require full-SymbolId ordering, complete header-count agreement, repeated result identity, exact module/declaration-name/kind filtering, same-name preservation across modules, successful zero matches, full-ID-only exclusive cursors inside the complete filtered set, deterministic pagination, known remaining counts, positive symbol/selection-byte limits, and fail-closed invalid filters, foreign cursors, or unpageable budgets. Symbol Catalog v1 output tests additionally require domain-separated checksum integrity, v0-prefix rejection, canonical no-newline JSON, echoed exact-name metadata, final-byte bounds, path/source exclusion, ordinary/locked identity, closed CLI filters and name limits, and output-failure classification. Fuzzy or normalized search, dependency catalogs, decoder input, and representative latency remain absent. |
| Atlas compiler-fact Context Pack | Current real-workspace tests require repeated canonical JSON bytes, independently recomputed domain-separated checksums, exact overview/show/around payload exclusivity, package and compatibility identity, query-budget adherence, a separate final byte limit, empty documentation/source-excerpt/warning arrays, no absolute path or source-body leakage, and exact query-error propagation. Experimental CLI tests additionally require exact root/member UID selection, catalog-to-query SymbolId flow, visible/full selector equivalence across show/around, ordinary/locked byte identity, canonical no-newline stdout, explicit bounded options, empty text sections, path/source-body exclusion, closed flags and failure behavior, and output-failure classification. Decoder strictness, untrusted input, dependency joins, and representative performance remain future evidence; attributed text is owned by a separate internal lane and does not change this pack. |
| Atlas attributed text | ADR 0195 owns the separate untrusted authority/origin, exact identity/span, no-reopen, UTF-8/control, atomic budget, omission, checksum, and unchanged `.fymap`/Context Pack v1 policy. ADR 0196 tests freeze the model/fragment checksum and closed vocabulary; validate SymbolId/package/file/line-map/span linkage, mutation detection, prompt-like printable text, and guard budgets; and compile a BOM-prefixed CRLF/non-ASCII workspace to prove retained-byte extraction, stale/wrong-workspace rejection, and atomic safety omissions. ADR 0197 output tests require explicit selection, visible/full and cross-root byte identity, compact field order, no path/newline, independently recomputed and frozen document integrity, one-byte-too-small failure, compiler-bundled actual-origin omission, and rejection of mutated subject/selection/usage/record/cardinality. ADR 0198 CLI tests additionally require catalog-discovered visible/full and locked byte identity, explicit origin/count/text/output limits, prompt-like text retained as untrusted, successful checksummed size omission without source bytes, closed flag/action policy, unknown package/symbol and final-bound failures, and output-failure classification. ADR 0199 frontend tests require exact `///` subtype and byte-span attachment, conservative detachment, and CST/AST/HIR retention. ADR 0200 tests require immutable snapshot sidecars, byte-identical `.fymap` v1 with or without that sidecar, exact retained marker-bearing documentation, visible/full/locked and cross-root identity, unavailable/size/UTF-8/control omissions, stale workspace and mismatched snapshot rejection, final-output bounds, and a distinct experimental documentation action. Multi-record count/remaining-aggregate behavior, representative performance, decoder, dependency/generated text, rendered documentation, and model-facing security claims remain absent. |
| Atlas Evidence Bundle | ADR 0202 package tests require visible/full byte identity, one retained-workspace state after disk mutation, independently valid/checksummed Context Pack and two attributed-text children, independently recomputed wrapper integrity, explicit `none`/`compiler-fact`/`untrusted-text` authority separation, no host path, exact source-versus-documentation selection, successful unavailable and atomic text-budget omissions, and independent facts/text-document/final byte-limit failures. CLI tests additionally require ordinary/locked identity, canonical no-newline stdout, every query/text/component/final option, closed action/flag policy, unknown package/symbol failures, and output-writer classification. Decoder, untrusted input, cross-version child acceptance, persistence, authentication, multi-symbol or neighborhood composition, dependency/generated text, token budgeting, model calls, and representative latency or payload distributions remain absent. |
| Atlas whole-package rewrite measurement | Schema tests require the exact encode-only scope, one warmup plus nine raw samples, measured clock classification, fixed package/source/record denominators, invariant encoded bytes, explicit per-package bounds, path/source exclusion, and no format or incremental-update conclusion. The Windows smoke prepares real verified quickstart partitions outside the timer, then requires every complete canonical rewrite to reproduce the warmup bytes. Representative repository distributions, peak memory, planning cost, query latency, disk-publication cost, reference-machine evidence, and regression thresholds remain absent. |
| Atlas workspace-call process measurement | Schema tests require the exact command-start-through-exit scope, excluded compiler build/discovery/warmup, nine first-attempt samples, clock classification, fixed source and direct-call shape, invariant canonical output size, explicit construction/output bounds, per-process Windows working-set/pagefile peaks, path/source exclusion, and no budget/speedup/traversal/impact conclusion. The Windows smoke uses one standalone compiler, exact root-main discovery, one untimed warmup, and nine fresh locked processes whose catalogs must remain byte-identical. The tiny fixture, uncontrolled OS caches, whole-process memory attribution, absent live-heap/index retention, representative repositories, reference environment, and regression threshold remain explicit gaps. |
| Atlas workspace-call scaling measurement | Schema tests require detached closed definitions for 2-package/8-call, 8-package/64-call, and 32-package/512-call generated star workspaces; exact join-plus-outgoing-encode scope; one warmup and nine raw samples per shape; increasing source/map/output denominators; invariant canonical bytes; process-wide `TotalAlloc`/`Mallocs` deltas; and no complexity/default/traversal/impact/regression conclusion. Generator tests compile and prepare every shape before sampling. Non-star repositories, indirect calls, retained or peak memory, controlled GC, reference-machine evidence, cross-machine comparison, and regression thresholds remain absent. |
| Atlas workspace-call topology corpus | The Go correctness suite generates ordinary manifest-backed chain, diamond, and fan-in `.fy` workspaces, compiles them to verified MIR, prepares exact-current partitions, requires complete cross-package usage, checks root outgoing/intermediate both/sink incoming selections, and repeats canonical catalog encoding. The closed synthetic shapes cover multi-level, shared-leaf, and converging-caller structure only; representative repositories, indirect or foreign calls, other relation families, traversal, impact, and performance conclusions remain absent. |
| Atlas workspace-call retained-heap control | Schema tests freeze a separate path-free report over ADR 0228's three generated star shapes, five first-observation samples, 16 retained indexes per sample, prepared-input liveness, forced GC before baseline and after construction, and positive process `HeapAlloc`/`HeapObjects` deltas. The Windows harness compiles and prepares each shape before sampling. The deltas include Go runtime, allocator, garbage-collector, and other process state; exact Atlas ownership, per-index division, peak memory, representative repositories, reference-machine evidence, budgets, defaults, traversal, impact, and regression thresholds remain absent. |

## Diagnostic assertions

Compile-fail fixtures assert the stable diagnostic code, primary failure category, and required related spans. They do not freeze full prose unless wording itself is the contract. A single malformed input may produce recovery diagnostics, but the fixture names which diagnostic is primary.

Parser fail fixtures are stricter: they list every `E00xx` result, use half-open byte spans, and record whether recovery inserted, skipped, synchronized, or exited. Parser pass fixtures assert syntax only and may intentionally fail later name or type analysis. Operator fixtures compare normalized tree shape rather than only evaluation results.

Semantic fail fixtures list every independent diagnostic after cascade suppression, identify the owning resolver, type, or Flow phase, and use role-labelled related spans. Semantic pass fixtures may assert declaration coordinates, canonical types, and borrow origins, but never whole AST or HIR dumps or build-local IDs.

Resource fixtures sample complete-root state immediately before or after a source span. They assert that `Live` owns one cleanup obligation, `Dead` owns zero, and a source-visible `MaybeLive` join is rejected. MIR drop flags and executable drop order remain separate fixture contracts.

Region v1 resolves its lexical escape source through the explicit standard-library package instance and opaque fixture `Arena`; it does not pretend a user-declared type is the standard-library arena. Result fixtures retain parameter-driven cases to isolate handling, use the current contextual leading-dot constructors and shallow switches, and exercise the ADR 0019 pending-obligation dataflow separately. ADR 0017 remains historical evidence for the superseded qualified spelling. Each manifest lists its deferred coverage explicitly.

MIR fixtures compare source-linked observations after verification. They do not compare textual dumps, block or local numbers, temporary names, or generated C. MIR-verifier fixtures separately require each named baseline to verify, apply one typed mutation, and return the complete one-code result; arbitrary malformed graph input remains deferred.

The MIR, interpreter, C-emission, and native source-probe fixture producers obtain executable MIR
through `internal/compiler`. A compiler-boundary architecture test parses those runner sources and
fails if they call semantic analysis, MIR lowering, or MIR verification directly. Direct verifier
calls remain intentional in mutation fixtures and as defensive checks at interpreter and backend
entry points.

Package fixtures start from an already-resolved graph. Build-local package IDs may be asserted only
inside compiler tests to prove isolation; they are not stable fixture output. Package-aware tests
must assert that source paths remain unqualified while span identity still distinguishes packages.
Single-package fixture span matchers require the default package ID. Those resolved-graph fixtures
do not prove the separately tested project-manifest parser, version solving, registry behavior,
source acquisition, lockfile encoding, diagnostic path mapping, or a concrete package-instance hash.

C-emission fixtures compare structured unit facts and deterministic text hygiene without freezing the
entire generated translation unit. `internal/cbackendnative` separately requires Linux/amd64,
glibc, Clang, and GCC, then compiles and executes every emitted case at `-O0` and `-O2`. The command
fails rather than skips when a required toolchain component is absent. Its current semantic oracle compares
reserved process status and exact source-linked compiler-owned drop sequences through a closed
instrumentation profile. A distinct `address-undefined-o1` mode runs the uninstrumented production
unit under ASan+UBSan with both compilers and treats every diagnostic as failure; this is an
executed-path backend check, not another language oracle. The lane does not prove output, negative
or larger `i32` returns, user drop-body effects, layout, ABI, the complete product runtime, or target conformance. The
experimental build command shares this driver and scalar runtime but emits and links only the
uninstrumented explicit FY source closure; it does not inherit the fixture observer or broaden target
evidence.

## Differential matrix

The first planned matrix targets `x86_64-linux-gnu`, uses the MIR interpreter plus generated C, and compares Clang with GCC at low and optimized profiles. Exact compiler and glibc baselines remain UNDECIDED. Fairy-to-C and C-to-Fairy fixture directions are both mandatory. A new target or backend cannot replace the interpreter and reference-C oracles merely because its outputs agree with itself.

The host-independent Linux C ABI fixtures cover scalar and pointer classification, fixed arrays,
register exhaustion, mixed integer and floating arguments, by-value aggregate plans, and hidden
return storage. The Windows fixtures separately cover shared ordinal slots, aggregate size gates,
indirect arguments, and hidden-result slot shifting. A separate-object Clang/GCC probe exercises all 14 Linux pass plans in both compiler
directions. A Fairy-source probe additionally crosses an 8-byte register struct result and a
20-byte hidden-memory struct result, while another crosses typed null pointers plus `usize` and `u8` in
both directions. Another four-way probe links the empty-input UTF-8 call against the actual
standard-library text-view C unit instead of a mock. A CStr probe additionally links the generated
caller against the actual owned-text C unit and supplies nonempty success and failure storage.
Buffer, Arena, and Linux File source probes supply non-null nested records, a self-pointer, one
pointer-output crossing, partial counts, and consuming state transitions, but native source fixtures still need independently padded
`@repr(C)` parameters, explicit C enums, narrow extension rules, mixed-class aggregate results,
and the permitted synchronous callback. Every excluded C ABI feature eventually needs a negative
source fixture as its semantic surface is implemented.

## Fuzzing and sanitizers

Lexer and parser fuzz entry points now exist, and their fixed seed corpus runs under the ordinary
test command on Windows and Linux CI. A timed or continuous fuzz campaign is not configured yet.
Type, Flow CFG, MIR verifier, and schema fuzzing remain later validation surfaces. Generated-C
sanitizer runs now cover every admitted emission case with Clang and GCC in the hosted native
command, and ADR 0214 records the first hosted result. They are backend evidence, not proof of
language semantics. Fuzz duration, retained corpus policy, and coverage thresholds remain
uncommitted until reproducible campaign evidence exists.

## Review blockers

- A semantic rule has no positive or negative observable fixture class.
- A grammar, literal, precedence, or recovery change updates prose without changing `spec/grammar/stage0.ebnf` and the parser manifest in the same review.
- A resolution, typing, or Flow rule changes without a schema-compatible semantic pass or fail fixture and synchronized diagnostic meaning.
- A whole-root ownership rule changes without synchronized resource-state assertions, `E11xx` spans, and drop-elaboration review.
- A Region or Result rule changes without its dedicated manifest, deferred-coverage review, `E31xx` or `E41xx` spans, and MIR-lowering review.
- A MIR rule changes without synchronized source-linked observations, deferred-coverage review, verifier impact, and interpreter or enabled-backend impact.
- A MIR verifier rule changes without synchronized seed entities, one-mutation isolation, complete code coverage, and harness-failure review.
- A package identity or visibility rule changes without synchronized resolved-graph cases, module lookup review, compatibility review, and Atlas package-edge review.
- A backend test compares only generated text instead of executable behavior where execution is available.
- A diagnostic code changes meaning without the catalog and compile-fail fixtures changing together.
- A trap, move, or cleanup result differs by optimization profile.
- A foreign nonlocal-control test crosses a Fairy C frame instead of using a boundary shim.
- An Atlas test treats missing or opaque edges as a complete graph.
- A target is called conformant or supported while its declared fixture subset contains a skip.
- A runtime wrapper normalizes Linux-specific behavior and then documents it as portable POSIX behavior.
- Exact test counts, durations, or coverage numbers are promised without an executable harness and source of truth.
