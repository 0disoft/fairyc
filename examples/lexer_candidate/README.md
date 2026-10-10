# Experimental Fairy compiler candidate

The candidate implements recognition, typing, owned HIR/MIR and C emission in
Fairy source. Go stage0 still builds this Windows AMD64 MSVC candidate; compiler
promotion and self-hosted stages are pending.

Read the [current module and SDK progress](../../docs/ops/candidate-module-progress-20261008.md)
and [numbered remaining work](../../docs/ops/candidate-remaining-work-20261007.md)
for maintained support and evidence. The guide below preserves the earlier
feature contracts and reproduction commands; version-specific milestone prose
is historical.

File input is bounded to 64 KiB, a joined program to 1 MiB (including separators),
and a source list to 256 files. AST, depth, binding and work limits remain separate.
`--c-modules-file` emits verified C for admitted multi-file programs; authenticated
`std::num` bodies execute from authenticated SDK source, including enum error
payloads. Authenticated `std::process` bodies call the original target argument
runtime through independently checked SDK source and signature ownership.
Other SDK bodies and project build/run/test integration remain in progress.

## Return-origin checking

After building the native candidate, run:

```powershell
.\build\lexer-candidate.exe --check-return-origins-file sources.txt
```

The manifest uses the same one-relative-path-per-line format as
`--check-modules-file`, relative to the command's working directory. LF and
CRLF are accepted. The command checks direct, single-leaf tagged and
single-safe-borrow-leaf Result `?` borrow return origins in one
bounded primary unit and prints original file/byte locations for E2102,
E2103 and their related origins. Local Region escapes emit E3101 with the
original allocation location, including provider and generic allocation sites.
Status 3 means a semantic origin error;
status 2 means unknown/unsupported analysis, a phase/input failure or a limit.

Single-borrow-leaf Option/Result payload construction, empty payloads, nested
wrappers and call forwarding are supported. Single-safe-borrow-leaf Result `?`
propagation preserves the success payload origin and records the implicit error
return at the full `?` expression when the failure carries a safe borrow.
Borrowing a wrapper itself retains its storage origin. Multiple borrowed leaves
remain unsupported. Match payload bindings preserve subject origins and lexical
scope. Unsupported return/binding/assignment/match locations are all reported
when no semantic origin error is present. `Region` input parameters are
admitted as a compiler-owned kind-19 capability and forward across modules,
direct and provider calls and scalar-generic instantiations. `Region.place`
checks fully inert values as `Result<ref T, AllocError>` and carries the
receiver Region's provenance through the success payload. Resource-containing
values are refused with code 3102; borrow-containing values remain unsupported.
Lexical Region bindings and their exact authenticated `ref mut std::alloc::Arena`
sources are typed, and the source expression is evaluated before its nested block.
A returned local Region borrow is rejected with E3101 at the return or `?`
expression and an `allocation` related location. Multiple allocations from one
Region keep one origin; different Region origins still produce E2103.
Region enter/reset and runtime execution, loan/subview/lifetime/conflict/move,
unsafe pointer provenance, split-unit
execution, the full project dispatcher, stage1/stage2, release and Linux are
pending.

## Feature guide

- [Experimental Fairy lexer candidate](guide/01-experimental-fairy-lexer-candidate.md)
- [Owned scalar HIR (`--hir-file`)](guide/02-owned-scalar-hir---hir-file.md)
- [Bounded syntax parser](guide/03-bounded-syntax-parser.md)
- [Generic declarations and SDK observations](guide/04-generic-declarations-and-sdk-observations.md)
- [Mutable byte views and native C execution](guide/05-mutable-byte-views-and-native-c-execution.md)
- [Scalar Result handling obligations](guide/06-scalar-result-handling-obligations.md)
- [Cleanup body ownership checks (unit 21)](guide/07-cleanup-body-ownership-checks-unit-21.md)

## Existing feature links

<a id="owned-consumer-api"></a>
[Owned consumer API](guide/01-experimental-fairy-lexer-candidate.md#owned-consumer-api)
<a id="regular-file-token-dump-cli---tokens-file"></a>
[Regular file token dump CLI (`--tokens-file`)](guide/01-experimental-fairy-lexer-candidate.md#regular-file-token-dump-cli---tokens-file)
<a id="regular-file-declarations-cli---declarations-file"></a>
[Regular file declarations CLI (`--declarations-file`)](guide/01-experimental-fairy-lexer-candidate.md#regular-file-declarations-cli---declarations-file)
<a id="regular-file-name-checking---check-names-file"></a>
[Regular file name checking (`--check-names-file`)](guide/01-experimental-fairy-lexer-candidate.md#regular-file-name-checking---check-names-file)
<a id="retained-declaration-use-links---bindings-file"></a>
[Retained declaration-use links (`--bindings-file`)](guide/01-experimental-fairy-lexer-candidate.md#retained-declaration-use-links---bindings-file)
<a id="scalar-type-checking---check-types-file"></a>
[Scalar type checking (`--check-types-file`)](guide/01-experimental-fairy-lexer-candidate.md#scalar-type-checking---check-types-file)
<a id="control-flow-name-scopes"></a>
[Control-flow name scopes](guide/01-experimental-fairy-lexer-candidate.md#control-flow-name-scopes)
<a id="control-flow-type-checking"></a>
[Control-flow type checking](guide/01-experimental-fairy-lexer-candidate.md#control-flow-type-checking)
<a id="multi-file-scalar-checking---check-modules-file"></a>
[Multi-file scalar checking (`--check-modules-file`)](guide/01-experimental-fairy-lexer-candidate.md#multi-file-scalar-checking---check-modules-file)
<a id="owned-scalar-hir---hir-file"></a>
[Owned scalar HIR (`--hir-file`)](guide/02-owned-scalar-hir---hir-file.md#owned-scalar-hir---hir-file)
<a id="owned-scalar-mir---mir-file"></a>
[Owned scalar MIR (`--mir-file`)](guide/02-owned-scalar-hir---hir-file.md#owned-scalar-mir---mir-file)
<a id="recognition-and-limits"></a>
[Recognition and limits](guide/02-owned-scalar-hir---hir-file.md#recognition-and-limits)
<a id="complete-checked-in-source-corpus"></a>
[Complete checked-in source corpus](guide/02-owned-scalar-hir---hir-file.md#complete-checked-in-source-corpus)
<a id="own-source-comparison-without-increasing-process-output-limits"></a>
[Own-source comparison without increasing process output limits](guide/02-owned-scalar-hir---hir-file.md#own-source-comparison-without-increasing-process-output-limits)
<a id="bounded-syntax-parser"></a>
[Bounded syntax parser](guide/03-bounded-syntax-parser.md#bounded-syntax-parser)
<a id="own-source-syntax-coverage"></a>
[Own-source syntax coverage](guide/03-bounded-syntax-parser.md#own-source-syntax-coverage)
<a id="package-syntax-and-file-identity"></a>
[Package syntax and file identity](guide/03-bounded-syntax-parser.md#package-syntax-and-file-identity)
<a id="generic-declarations-and-sdk-observations"></a>
[Generic declarations and SDK observations](guide/04-generic-declarations-and-sdk-observations.md#generic-declarations-and-sdk-observations)
<a id="verified-scalar-c-emission---c-file"></a>
[Verified scalar C emission (`--c-file`)](guide/04-generic-declarations-and-sdk-observations.md#verified-scalar-c-emission---c-file)
<a id="same-file-nominal-struct-type-checking-historical-0160"></a>
[Same-file nominal struct type checking (historical: 0.16.0)](guide/04-generic-declarations-and-sdk-observations.md#same-file-nominal-struct-type-checking-historical-0160)
<a id="owned-nominal-hir-historical-0170"></a>
[Owned nominal HIR (historical: 0.17.0)](guide/04-generic-declarations-and-sdk-observations.md#owned-nominal-hir-historical-0170)
<a id="owned-nominal-mir"></a>
[Owned nominal MIR](guide/04-generic-declarations-and-sdk-observations.md#owned-nominal-mir)
<a id="owned-nominal-c"></a>
[Owned nominal C](guide/04-generic-declarations-and-sdk-observations.md#owned-nominal-c)
<a id="owned-scalar-fixed-arrays"></a>
[Owned scalar fixed arrays](guide/04-generic-declarations-and-sdk-observations.md#owned-scalar-fixed-arrays)
<a id="owned-static-text-and-byte-literals"></a>
[Owned static text and byte literals](guide/04-generic-declarations-and-sdk-observations.md#owned-static-text-and-byte-literals)
<a id="static-literal-mir-and-native-c-execution"></a>
[Static literal MIR and native C execution](guide/04-generic-declarations-and-sdk-observations.md#static-literal-mir-and-native-c-execution)
<a id="shared-array-view-hir"></a>
[Shared array view HIR](guide/04-generic-declarations-and-sdk-observations.md#shared-array-view-hir)
<a id="shared-array-views-slices-and-native-c-execution"></a>
[Shared array views, slices and native C execution](guide/04-generic-declarations-and-sdk-observations.md#shared-array-views-slices-and-native-c-execution)
<a id="mutable-byte-view-hir"></a>
[Mutable byte-view HIR](guide/04-generic-declarations-and-sdk-observations.md#mutable-byte-view-hir)
<a id="mutable-byte-views-and-native-c-execution"></a>
[Mutable byte views and native C execution](guide/05-mutable-byte-views-and-native-c-execution.md#mutable-byte-views-and-native-c-execution)
<a id="cost-baseline"></a>
[Cost baseline](guide/05-mutable-byte-views-and-native-c-execution.md#cost-baseline)
<a id="array-fields-and-rank-two-array-hir"></a>
[Array fields and rank-two array HIR](guide/05-mutable-byte-views-and-native-c-execution.md#array-fields-and-rank-two-array-hir)
<a id="array-fields-and-rank-two-native-execution"></a>
[Array fields and rank-two native execution](guide/05-mutable-byte-views-and-native-c-execution.md#array-fields-and-rank-two-native-execution)
<a id="text-access-diagnostic"></a>
[Text access diagnostic](guide/05-mutable-byte-views-and-native-c-execution.md#text-access-diagnostic)
<a id="payloadless-enum-hir"></a>
[Payloadless enum HIR](guide/05-mutable-byte-views-and-native-c-execution.md#payloadless-enum-hir)
<a id="payloadless-enum-native-execution"></a>
[Payloadless enum native execution](guide/05-mutable-byte-views-and-native-c-execution.md#payloadless-enum-native-execution)
<a id="integer-and-bool-payload-enum-hir"></a>
[Integer and bool payload enum HIR](guide/05-mutable-byte-views-and-native-c-execution.md#integer-and-bool-payload-enum-hir)
<a id="integer-and-bool-payload-enum-execution"></a>
[Integer and bool payload enum execution](guide/05-mutable-byte-views-and-native-c-execution.md#integer-and-bool-payload-enum-execution)
<a id="scalar-option-type-hir-and-mir"></a>
[Scalar Option type, HIR and MIR](guide/05-mutable-byte-views-and-native-c-execution.md#scalar-option-type-hir-and-mir)
<a id="scalar-option-native-execution"></a>
[Scalar Option native execution](guide/05-mutable-byte-views-and-native-c-execution.md#scalar-option-native-execution)
<a id="option-control-flow"></a>
[Option control flow](guide/05-mutable-byte-views-and-native-c-execution.md#option-control-flow)
<a id="result-scalar-type-identity"></a>
[Result scalar type identity](guide/05-mutable-byte-views-and-native-c-execution.md#result-scalar-type-identity)
<a id="result-constructors-and-patterns"></a>
[Result constructors and patterns](guide/05-mutable-byte-views-and-native-c-execution.md#result-constructors-and-patterns)
<a id="owned-scalar-result-hir"></a>
[Owned scalar Result HIR](guide/05-mutable-byte-views-and-native-c-execution.md#owned-scalar-result-hir)
<a id="scalar-result-mir-and-native-execution"></a>
[Scalar Result MIR and native execution](guide/05-mutable-byte-views-and-native-c-execution.md#scalar-result-mir-and-native-execution)
<a id="result-propagation-type-checking"></a>
[Result propagation type checking](guide/05-mutable-byte-views-and-native-c-execution.md#result-propagation-type-checking)
<a id="result-propagation-execution"></a>
[Result propagation execution](guide/05-mutable-byte-views-and-native-c-execution.md#result-propagation-execution)
<a id="scalar-result-handling-obligations"></a>
[Scalar Result handling obligations](guide/06-scalar-result-handling-obligations.md#scalar-result-handling-obligations)
<a id="nested-option-and-result-values"></a>
[Nested Option and Result values](guide/06-scalar-result-handling-obligations.md#nested-option-and-result-values)
<a id="struct-and-diagnostic-optionresult-payloads"></a>
[Struct and diagnostic Option/Result payloads](guide/06-scalar-result-handling-obligations.md#struct-and-diagnostic-optionresult-payloads)
<a id="tagged-struct-fields"></a>
[Tagged struct fields](guide/06-scalar-result-handling-obligations.md#tagged-struct-fields)
<a id="composite-array-type-and-owned-hir-prerequisite"></a>
[Composite array type and owned HIR prerequisite](guide/06-scalar-result-handling-obligations.md#composite-array-type-and-owned-hir-prerequisite)
<a id="composite-array-mir-and-native-execution"></a>
[Composite array MIR and native execution](guide/06-scalar-result-handling-obligations.md#composite-array-mir-and-native-execution)
<a id="resource-type-identity-and-owned-hir"></a>
[Resource type identity and owned HIR](guide/06-scalar-result-handling-obligations.md#resource-type-identity-and-owned-hir)
<a id="explicit-resource-factory-and-construction-hir"></a>
[Explicit resource factory and construction HIR](guide/06-scalar-result-handling-obligations.md#explicit-resource-factory-and-construction-hir)
<a id="whole-root-resource-transfer-hir"></a>
[Whole-root resource transfer HIR](guide/06-scalar-result-handling-obligations.md#whole-root-resource-transfer-hir)
<a id="resource-use-after-move-and-transfer-provenance"></a>
[Resource use-after-move and transfer provenance](guide/06-scalar-result-handling-obligations.md#resource-use-after-move-and-transfer-provenance)
<a id="resource-branch-joins"></a>
[Resource branch joins](guide/06-scalar-result-handling-obligations.md#resource-branch-joins)
<a id="resource-loop-edges"></a>
[Resource loop edges](guide/06-scalar-result-handling-obligations.md#resource-loop-edges)
<a id="cleanup-body-ownership-checks-unit-21"></a>
[Cleanup body ownership checks (unit 21)](guide/07-cleanup-body-ownership-checks-unit-21.md#cleanup-body-ownership-checks-unit-21)

## Move-state checking

`--check-return-origins-file` also tracks per-lexical-local move state. Each
analysis local is Live, Dead or MaybeLive across consuming arguments, returns,
record constructors, Option/Result `switch` arms and `?`, short-circuit operands,
reachable branch joins, classic scalar `for`/`while` and iteration backedges with
`break`/`continue`, reassignment and scope transitions. It reports E1101 use after
move, E1102 second cleanup obligation, E1103 field move out of a resource, E1104
inconsistent states at a branch join and E1105 inconsistent states at a loop edge
beside its origin diagnostics. E1103 carries field and owner anchors across
provider modules and E1102 keeps the cleanup receiver declaration anchor.
Non-indexed field assignment reads its right-hand side before the store target, so
a move there rejects the target read with E1101, while indexed targets read the
receiver and index first. `discard` counts as a read, a classic `for` header fixes
the loop-header coordinates for E1105, and cast type syntax is not traversed as a
value. The private fact API returns an immutable owned plan validated after the
source and values are released.

Cleanup drop obligations and effects now propagate through owned resource fields and
their partially evaluated owners; MIR/C/runtime wiring, outstanding borrow and access
edges, unsafe provenance, the full dispatcher and stage1/stage2 remain pending; Release,
Linux and the full compiler suite are deferred.

## Cleanup and drop-effect checking

`--check-return-origins-file` also derives the cleanup obligations of live locals
from the same move state. A local that reaches a scope exit, `return`, `break`,
`continue`, reassignment of a live owner, a `?` error edge, a consumed `is_ok`, a
discarded temporary or a match arm raises one cleanup request; moved and MaybeLive
locals raise none, and the cleanup receiver `this` is exempt. The command rejects an
outer resource whose ability set omits `drop` with E1108, inspecting its ordered
fields only when the outer resource is droppable so an outer rejection supersedes
inner effects, and rejects a nominal owner whose cleanup effect mask exceeds the
caller with E7102. Both diagnostics carry the offending `ability-declaration` or
`effect-declaration` as a related location.

A private owned body IR now retains the checked bodies after the parsed and typed owners are released, and a private CFG lowers each body into one-operation blocks with shape and coverage checks. The graph lowers CFG in reverse from a terminal block, reverses a temporary's drops before lexical owners, normalizes equivalent repeated cleanup requests to one runtime drop site, supports normal records, and rejects switch, foreach, slices, the named-construct syntax, defer and Region with 1004. Its seven native graph tests pass and the ninth mutation mode is accepted while modes 1-7 and 9 reject with 1001 and mode 8 with 1002; generated C, the whole language, defer/Region and the source admission origins remain pending.

A private owned body IR now retains the checked bodies after the parsed and typed owners are released. `module_unit_body_ir_lower.fy` copies each frame's typed expression nodes with their syntax links, its statements with their assignment operators, and its locals, calls, arguments and Region scopes and exits, plus the raw and cooked literal text with scanner string cooking decoded to real bytes through `literals::decode`; at program level it resolves the layout, field and callable names into one bounded pool, and `module_unit_body_ir_verify.fy` revalidates the tables, body sizes, root blocks, syntax links and call/Region/operand shapes independently. It keeps semantic request order and does not attach drop sites to CFG blocks or connect the reverse-lexical, defer and Region ordering, so the C handoff, an executable split-unit body and any new C pipeline remain pending. The existing `module_program` C path still joins and is unchanged, no CLI route exercises the fragment, and the four native body-IR tests pass, with eleven malformed table modes rejected; Release, Linux and the full compiler suite stay deferred.
The command walks each owner's ordered fields, follows `Option` and `Result` payloads
and deduplicates canonical types, so cleanup effects propagate through nested fields:
an outer resource whose field cleanup declares an effect the caller does not hold is
rejected with E7102 anchored at that field's `effect-declaration`. A later field or call
initializer's abrupt exit now charges cleanup effects for the owners already evaluated
before it. A completed construction or invocation releases its pending frame;
`break`/`continue` charge only owners captured inside the target loop. Each cleanup
request keeps a source-qualified operand identity: an owned lexical root is identified
by its binding, a temporary by its original source expression ID, and generic clones
stay distinct through their evaluation frames. A private source-independent cleanup MIR
fragment now lowers those requests into owned canonical tables, per-type glues, ordered
fields, frame-qualified operands and drop sites with bounded exit reasons, and
revalidates them independently; it keeps semantic request order and is not executable
whole MIR. Split-unit cleanup event MIR, C and runtime elaboration, remaining borrow and
access edges, unsafe provenance, and the full dispatcher and stage1/stage2 remain
pending; Release, Linux and the full compiler suite are deferred.

## Bounded lexical defer

The CFG stage now lowers `defer call;` into immediate left-to-right scalar capture
slots (closed scalar set 1..6, including `i64`) with the target checked as a resolved
internal call returning `void` whose callable or instance frame passes the authority,
result, argument-count and effect-subset rules; foreign, template and value-returning
targets are rejected. Registered calls invoke in reverse registration order (LIFO) on
fallthrough, `return`, Result `?`, `break` and `continue`, with the return value frozen
before the deferred calls and both before implicit drops, and `break`/`continue` invoke
only the defer suffix registered inside the loop while the outer registrations wait for
the enclosing exit. Defer site IDs follow per-block preparation order and are not a
registration-order contract, so the nested case asserts the captured literal values in
execution order `3,2,4,1`. The five `tests/module_unit_cfg_defer.fy` tests (four
behavior paths plus one mutation case that accepts a valid clone and rejects nine
malformed defer modes with 1001) join the unchanged seven `tests/module_unit_cfg.fy`
graph tests for twelve unique native passes; the source check passed 252 files and 3205
MIR functions and the locked development build passed, with the receipt
`docs/ops/evidence/candidate-module-cfg-defer-20261010.json` recording the twelve passes
and nine new defer rejections. `Region` is pending for the next commit, and `switch`,
the C-graph evaluator and any generated or executed C remain unimplemented; this
supersedes the earlier defer-pending notes above.

## Region placement and reset staging

`module_unit_body_ir_lower.fy` copies and `module_unit_body_ir_verify.fy` revalidates a
`regions::Scope`/`regions::Placement` ledger that pairs every builtin `place` call
one-to-one, carries the canonical `ref mut std::alloc::Arena` source with its kind-19
capability local, and requires a nested Region to stay inside its parent body;
`module_unit_cfg_region.fy` lowers a Region checkpoint (block kind 15) and resets
(kind 16), emitting on an exiting edge all active defers first, then per crossed Region
its drops and reset innermost-first, then the outer drops, with the checks limited to
ledger and span facts rather than dominance, liveness or runtime ordering. Eight
`tests/module_unit_cfg_region.fy` outcomes pass (five new Region cases plus three reused
regressions) and nineteen new rejections are covered - eight invalid graph modes and
eleven invalid body-ledger modes, all 1001 - with the receipt at
`docs/ops/evidence/candidate-module-cfg-region-20261010.json`, reading the real
`internal/sdkstdlib/sources/std/alloc.fy` by byte identity and leaving the SDK ABI
unchanged; the locked source check passed 253 files and 3215 MIR functions and the locked
development build passed. The Region code is unchanged since the first four-pass run, and
this changes the private IR API while the stable CLI, SDK ABI, DB and CI stay unchanged
with no push; generated C, the whole language, the drop dominance-path proof and the
whole source admission and pipeline remain pending, and the tests exercise the `.fy`
compiler graph lowering only, so no Arena checkpoint runs and no C is emitted from or executed for this new CFG.

## Scalar C emitter unit (2026-10-10)

A private scalar C emitter now renders a standalone numerical C unit from the owned body IR without the original parsed or joined modules. `module_unit_c_emit.fy` renders from a validated zero-parameter `i32` entry frame, revalidating the body IR and freshly lowering each `Graph`, and `module_unit_c_scalar.fy` renders the closed scalar set of kinds 1..6 (i32, i64, u8, u32, usize, bool) with kind 7 as void and checked arithmetic. Boolean branches, `while` and classic `for` with `break`/`continue`, compound assignments, block/if value copies, frozen scalar `defer` captures and internal ordinary and generic calls are emitted; resources, `Result`/`Option`, views, references, arrays, casts, extern calls and Region execution are rejected with 1004, and output is transactional within a 2 MiB bound.

`tests/module_unit_c_emit.fy` holds six native cases, and its five emitted C programs pass under both MSVC `/Od` and `/O2`, ten executions in all, with the expected exits, including generic identity `h::id<i32>(answer)==11` and recursive `factorial(5)==120`; they are compiled by `tools/verify-candidate-module-c.py`, the render test runs as `/development`, and the generated C is not the candidate release build. The complete dominance and path proofs, whole source admission and the full dispatcher and stage1/stage2 remain pending, and the stable CLI, SDK ABI, Go, DB and CI are unchanged with no push; Release, Linux and the full compiler suite stay deferred. The scoped evidence receipt `docs/ops/evidence/candidate-module-c-execution-20261010.json` records the six native cases and the ten generated-C executions.

## Owned aggregate C unit (2026-10-10)

`module_unit_c_aggregate.fy` and the updated emitter now generate used canonical structs, resources and tagged Option/Result payload unions in dependency order, with retained field ID/type/module/declaration/source checks and header `fairyc.candidate-module-owned-c/v1`. Cleanup receives live storage by pointer; drop helpers run its body before nested fields in reverse declaration order and drop only the active tagged payload. Graph drops resolve frame-local bindings or materialized original expressions. Aggregate/internal-call, move and return copies follow the checked ownership ledger, and Result `?` runs existing exit defers and drops on error. Ordinary scalar extern C calls validate retained callable provenance, signatures, effect subsets and linker names, deduplicating compatible prototypes across providers. Pointer/aggregate/C-scalar extern ABI remains unsupported; registered defers retain internal scalar arguments only.

The combined captured native run passes 12/12 (six owned cases and six scalar regressions). `tools/verify-candidate-module-c.py --suite all` compiles ten programs at MSVC `/Od` and `/O2`, passing 20/20 executions (ten owned and ten scalar) with exact expected traces and exits. Generation tests use development; generated-C `/O2` is separate from candidate Release validation. The locked source check passed 256 files and 3252 MIR functions, and the locked development build passed. The receipt `docs/ops/evidence/candidate-module-owned-c-execution-20261010.json` records `passed=true`, 14/14 current source hashes and SHA-256 references to the full cached reports.

Arrays/references/pointers/views/casts/enums/Region execution, full path/dominance/initialization proofs, complete source admission and unsafe provenance, the whole dispatcher and stage1/stage2 remain pending. Admission revalidates the owned body IR and CFG; checked arithmetic uses bounded scalar helpers. The existing non-unwinding trap contract remains: traps run no defers, drops or Region resets and have no cleanup edge. The source cap stays 256. The private C API expands; public CLI, stable SDK ABI, Go, DB and CI are unchanged, with no push. Candidate Release, Linux and the full suite stay deferred; this bounded status supersedes earlier aggregate and scalar-extern C pending notes.

## Reference C unit (2026-10-10)

Canonical `ref T` and `ref mut T` now emit as C pointers, with const pointees for shared references. Dereference, field projection and address formation preserve owner storage; borrowed resources are not dropped, and generic reference returns are supported. CFG kind 17 freezes a scalar compound-assignment target's old value before its RHS, and the C store uses that snapshot. Five mutation modes reject malformed, missing or duplicate snapshots.

The combined three-suite run passes 18/18 native tests (six new reference cases and twelve regressions). Fifteen C programs pass at MSVC `/Od` and `/O2` (30/30 executions: ten reference and twenty regression), with exact reference traces: root/field `3,6`, borrowed resource `10,3`, indirect `5` and sequential `8`. The latest `--suite all` includes references; prior receipt counts remain historical. The locked check passed 256 sources and 3253 MIR functions, and the locked development build passed. Receipt: `docs/ops/evidence/candidate-module-reference-c-execution-20261010.json`, with all fifteen source identity hashes verified against current files (fourteen candidate source/test/lock files and one verifier).

Raw pointer/array/view/cast/enum transport, foreign pointer ABI/SDK calls, Region execution, complete source admission and unsafe provenance, dominance/path/initialization proofs, the whole pipeline and stage1/stage2 remain pending. The non-unwinding trap contract and public CLI/SDK ABI/Go/DB/CI are unchanged; no push. Candidate Release, Linux and the full suite remain deferred. This supersedes earlier references-pending notes only within the supported reference unit.

## Bounded fixed-array C unit (2026-10-10)

Canonical kind-16 rank-one arrays of scalars 1..6 or inert structs, length 1..4096, now emit as dependency-ordered numerical wrapper structs with `fy_items[N]`. List/repeated literals copy frozen children, evaluating a repeat operand once and checking retained repeat-count text with `arrays::count` against the canonical extent; arguments, returns and copies pass by value. Array `.len`, projected writes and element references preserve actual owner storage. Category-6 bounds checks precede reads and assignment RHS evaluation; kind-17 compound snapshots remain intact. Shared/mutable array views and whole-array borrow parameters remain unsupported. Array calls/returns are ordinary; generic arguments remain scalar-only (ADR 0384), and fixtures keep array fields in a plain Container with scalar resource cleanup separate (ADRs 0463/0468).

The final locked check passed 256 files and 3253 MIR functions, and the final locked development build passed; the source count and cap stay 256. One full seven-test native report on final production code and fixtures passes 7/7 with zero failures. Five generated C programs pass at MSVC `/Od` and `/O2` (10/10 executions), matching all stdout exactly. Receipt: `docs/ops/evidence/candidate-module-array-c-execution-20261010.json`, with twenty current source identity hashes verified (nineteen candidate source/test/lock files and one verifier).

The original loop-filled 4096-element `i64`-maximum repeat passed `/Od` but copied -1 at `/O2`, failing assertion category 5 (exit 105). Using the frozen operand in an aggregate initializer, as in `c_arrays`, fixes the observed case: `/O2` passes with stdout `4,1,3,4,9,1`; optimization remains enabled. The expanded `--suite all` (23 native/40 C) was not run, and unchanged baseline suites were not rerun on current sources. Prior receipt counts remain historical; generated-C `/O2` is separate from candidate Release validation.

Direct nested-array elements, raw pointer/view/cast/enum transport, authenticated SDK calls/foreign pointer ABI, Region execution, complete source admission and unsafe provenance, path/dominance/initialization proofs, the whole pipeline and stage1/stage2 remain pending. The non-unwinding trap contract stays intact. The private C API expands; public CLI/SDK ABI/Go/DB/CI/hygiene rules are unchanged, with no push. Candidate Release, Linux and the full suite stay deferred. This supersedes earlier fixed-array C pending notes within the stated scope.

## Array-backed view C unit (2026-10-10)

Canonical byte views (10/11), extended scalar views (15) and plain-struct views (17/18) now emit numerical descriptors with const or mutable `fy_data` pointers and `fy_length`. Array borrows and subranges alias original owner/field storage; left-value slots may still copy arrays. Array-backed descriptors support `.len/.is_empty`, reads/mutable writes, category-6 bounds checks, empty end and nested subranges, and omitted bounds. CFG evaluates owner/lower/upper in order without emitting a kind-27 wrapper. Calls freshly revalidate checked expected/actual types and shared flags; mutable-to-shared views use descriptor compound literals and reference pointers gain const pointees.

The first mutation program passed `/Od` but failed assertion category 5 (exit 105) at `/O2`, retaining scalar value 3 despite owner/copy value 5. Kind-2 aggregate owner reads now use `fy_unit_copy`, excluding scalar/reference types. Its MSVC-conditional `#pragma function(memcpy)` and `__declspec(noinline)` force CRT copying only in that helper, then `#pragma intrinsic(memcpy)` restores scalar-bit intrinsics. `/O2` remains enabled; descriptors alias original owner storage and materialized array copies remain.

Final native validation passes 11/11 (seven view programs, two existing aggregate programs and two text guards). Nine C programs pass at `/Od` and `/O2` (18/18 executions: fourteen view, four aggregate), matching all stdout/exits. View stdout: owner alias `7,4,5,`, struct-field data `6,7,`, bound order `11,22,2,33,`. Reversed/upper-bound slices exit 106 with `1,`/`2,`; empty-end indexing exits 106 with empty stdout. Locked check: 256 sources, 3264 MIR functions; locked development build passed. Receipt `docs/ops/evidence/candidate-module-view-c-execution-20261010.json` records the results and copy correction with twenty-one verified current hashes (twenty candidate files plus verifier). Expanded `--suite all` (30 native/54 C) and full scalar/reference/owned/array suites were not rerun; two selected existing aggregate programs passed. Prior counts stay historical. Linux was not run; generated-C `/O2` remains separate from candidate Release validation.

This supports array-backed borrowed views only. Byte/text/CStr literals remain unsupported (literal kind 1 is scalar-only), as do raw pointer/cast/enum/direct nested-array transport, authenticated SDK calls/foreign pointer ABI and Region execution. Complete source admission and unsafe provenance, dominance/path/initialization proofs, the whole pipeline and stage1/stage2 remain pending. Non-unwinding traps are unchanged. The private C API expands; normative contracts and public CLI/SDK ABI/Go/DB/CI/hygiene rules are unchanged, with no push. Candidate Release, Linux and the full suite stay deferred. This supersedes earlier array-view pending notes within this scope.

## Candidate owned scalar-cast C unit (2026-10-10)

The owned-body scalar C emitter now admits explicit scalar casts. `module_unit_c_scalar.fy` checks a cast node as an exact target/result/source pair: a missing or inconsistent target, missing operand or wrong `as` token is malformed (1001), while an unsupported pair such as a forged identity is 1004. `scalar_casts.fy` admits the seven value-preserving integer widening pairs (u8 to i32, i64, u32, usize; i32 to i64; u32 to i64, usize) plus the exact two-way i32/c::int conversion, and the signed-32 `_Static_assert` on `sizeof(int)`/`_Alignof(int)` is emitted only when c::int is admitted; c::int keeps the `int` spelling while i32 stays `int32_t` across locals, parameters, returns and ordinary scalar extern prototypes, and every other C scalar or pointer cast stays closed. Operands are frozen once before the widening, and the arithmetic that follows stays checked at the target type range. In the transactional rejection case a full c::uint render keeps the sentinel, and the malformed and forged-identity modes are refused by the direct scalar emitter before any write. Earlier sections are historical and this status supersedes them only within this cast scope; Linux, candidate Release, the full suite, whole source and path proofs, Region, pointers, enums, SDK authentication and stage1/stage2 remain pending, with public CLI, SDK ABI, Go, DB, CI and normative contracts unchanged and no push.

Final validation passes 7/7 native C-emitter cases (five new cast cases and two regressions) plus all 4/4 existing cast-typing cases, 11/11 native tests in total. Six generated C programs pass at MSVC /Od and /O2 (12/12 executions), matching every exit and stdout. Signed-32 ABI round-trip stdout is -2147483648,2147483647,42, and single-evaluation/defer stdout is 1,7,9,; widened overflow exits 101. The locked check passed 256 sources and 3264 MIR functions, and the locked development build passed. The receipt docs/ops/evidence/candidate-module-cast-c-execution-20261010.json records the results with 21 current source hashes verified. The full suite, Linux and candidate Release were not run; generated-C /O2 is separate from candidate Release validation.

## Candidate owned pointer-cast typing unit (2026-10-10)

Owned-body source typing now records a cast's own lexical unsafe boundary as a separate kind-4 reference and admits exact scalar or nominal reference-to-pointer, mutable-to-shared and byte-view-to-byte-pointer conversions while refusing strengthening, void erasure outside a direct named reference or named-local borrow, and raw-pointer or integer reinterpretation. This is source typing and unsafe provenance only; pointer C transport, the `void**` output slot and raw dereference/arithmetic stay pending, public CLI, SDK ABI, Go, DB and CI are unchanged with no push, and Linux, candidate Release and the full suite remain deferred.

Validation: 9/9 native tests, locked source check (256 sources, 3268 MIR functions) and locked development build passed. All 249 root source/cast-fixture hashes stayed unchanged. Evidence: docs/ops/evidence/candidate-module-pointer-typing-20261010.json. Pointer C execution is still the next unit.

## Candidate owned pointer C unit (2026-10-10)

The owned-body C emitter now transports canonical kind-12 raw pointers as their C spellings: encoded scalar, `void` and `void**` pointers, and nominal pointees as `T *` or `T const *`. It renders typed `null`, inert copies, and internal pointer arguments, returns and `==`/`!=` comparisons. Reference casts forward the referent pointer, including the original fixed-array wrapper address for an opaque `void` cast, while borrowed view casts extract `fy_data`; an opaque local resource or array/view address crosses as `void` without a carrier address. Foreign ABI stays limited to encoded scalar, `void` and `void**` pointers, so a nominal pointer foreign ABI is still rejected 1004 pending `repr(C)` layout assertions, while nominal pointers still pass and compare through internal calls.

A direct cast of a fixed-array borrow (`ref mut u8[2] as ptr mut u8`) is refused by the contract, matching the Go baseline, so the fixture binds it explicitly (`let ref mut u8[] data = ref mut bytes;`) and then casts `data as ptr mut u8`; the array opaque `void` cast verifies the original owner address unchanged.

The central `raw_cast` rule now also admits only the single mutable opaque pointer output slot per ADR 0059 (`ref mut ptr mut void` to `ptr mut ptr mut void`); pointer-integer and pointee reinterpretation, raw dereference and arithmetic stay closed. `check_casts` revalidates each cast from retained canonical source, target, state and the `as` token with its unsafe flag, requiring a retained unsafe expression or a matching non-foreign unsafe callable and retained direct local syntax for opaque conversions, a retained-IR gate rather than authentication of the original source or a full path proof. Emission stays transactional, and the focused programs cover `null` with a foreign returned pointer and a `void**` slot, mutable and shared scalar references with a generic provider unsafe call, opaque resource and array/view crossings with cleanup once, and nominal pointer identity.

Focused native validation passes all 8 selected identities: four pointer C programs, one pointer guard, one output-slot typing case and two C regressions. The first run passed 7/8; after correcting only the unsupported direct-array-to-byte-pointer fixture, its 1/1 rerun passed. Exact byte reconstruction verifies all other helper/test bytes and production/verifier inputs unchanged, so the seven earlier passes were reused. Six generated C programs pass MSVC /Od and /O2 (12/12 executions: eight pointer and four regression), matching every exit and stdout. Pointer traces are 42,77, for null/result/output-slot; 9,6, for scalar storage; and 8,5,9,9, for byte-view/array/resource storage and one cleanup. Locked check: 256 sources and 3271 MIR functions; locked development build passed. The receipt docs/ops/evidence/candidate-module-pointer-c-execution-20261010.json records the initial failure, correction, final results and 22 verified current source identities. Linux, candidate Release and the full suite were not run; generated-C /O2 is separate from candidate Release validation.

## Candidate Region C place (2026-10-10)

Receipt: `docs/ops/evidence/candidate-region-c-place-20261010.json`. This supersedes the earlier place-rejects-1004 status within the supported scope below; earlier validation records remain historical.

- Supported operands: plain, fully initialized no-drop/no-safe-borrow scalars, raw pointers, structs, enums, Option/Result and fixed arrays.
- Internal transport: Region parameters use a private bridge pointer; authenticated Arena and `AllocError.status:u32` are verified. Other SDK storage and foreign Region remain closed.
- Placement: the expression is evaluated once and its value copied into allocated storage. Storage and tracking malloc failures return `.err` status 1; `?` runs Region reset and Arena cleanup.
- Generated C: actual `stage0_system_heap.c` plus `stage0_arena.c` pass MSVC `/Od` and `/O2` (8/8 executions: placement success 2, malloc failures at calls 1/2 4, lifecycle regression 2). Live allocations end at 0; success records exactly 10 malloc calls, and failure cases exactly 1 or 2.
- Switch correction: a terminal early arm's expected type was set before the switch result was known. Every arm is now frozen after result inference.
- Validation: final locked check passes 266 source files and 3370 MIR functions. The complete selected SDK file native rerun passes 9/9 with zero failures and lock verified (`native-place-complete.json`, exit 0); together with the passing switch regression 1/1, final native outcomes total 10/10, superseding the earlier five focused merged outcomes.
- Reference boundary: the corrected success fixture passes Go `check-semantics` (2 files, 2 modules, 11 functions), but experimental verified-MIR rejects its Region parameter with `place uses inactive region22`; this reference limitation leaves success-fixture MIR parity unverified. The failure fixture passes experimental verified-MIR check (2 files, 10 functions, 3 foreign functions).
- Changes: private emitter and local harness expanded; normative specs, stable CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push. Candidate Release, Linux, the full suite and stage1/stage2 remain deferred; generated C `/O2` is separate from candidate Release validation.
- Next: root-only SDK ordinary body materialization. Other SDK storage, full provenance/path/dominance/initialization proofs, foreign ABI, the dispatcher, self-hosting and stage1/stage2 promotion remain.

## Candidate root-only SDK ordinary bodies (2026-10-10)

Receipt: `docs/ops/evidence/candidate-root-sdk-bodies-20261010.json`, written by `record-imports.py` with 14 current source identity hashes, 11 final native outcomes plus 2 reused regressions and 18 generated-C executions. This implementation supersedes earlier root-only SDK-body deferrals within the stated scope; historical results remain unchanged.

- Collector: `module_unit_generics.extend_imports` gathers exact outside-range ordinary declarations from bound call targets and closure seeds into owned concrete bodies with zero type arguments. It preserves signatures, effects, unsafe flags, parameters and source IDs, forwards generic dependencies and deduplicates instances. The retained table is bounded at 256 ordinary plus generic bodies; generic specializations remain limited to 64 total and 16 per declaration with the unchanged 1M-work budget; failures roll back the appended graph.
- Closure and origins: existing call closure adds transitive ordinary/private/foreign targets; C targets require a non-template ordinary callable. Origin input lookup includes the owned ordinary lane without retyping the entire provider.
- SDK projection: the C plan skips absent raw-record shapes only when unused; active signatures, locals and expressions still require their shapes. The SDK test helper now binds only root 0..1 instead of root plus SDK 0..2. The initial console failure from an unused raw-read shape requirement was corrected.
- Transitive fixture: the root borrows `n` before `go(*borrowed)`. Four original sources include an unused module with an invalid body, a provider private helper and borrow return, and a leaf ordinary `add` calling generic `id<i32>`. The final test confirms five owned instances (four ordinary, one generic), six IR frames and exclusion of the unused source, then emits after the original sources are dropped.
- Rollback: a borrowed generic argument rejects 4502 after a borrow type has been appended; the original type count is restored.
- Validation: final locked check passes 266 source files and 3375 MIR functions. `native-imports-final.json` passes 10/10 (SDK 9 plus borrowed transitive 1), and `native-import-rollback.json` passes 1/1: 11/11 new/final identities. Two unchanged generic regressions pass 2/2 and are reused from `native-transitive-imports.json`, excluding its old transitive fixture.
- Generated C: the local helper's eight programs pass MSVC `/Od` and `/O2` (18/18 executions, including extra allocation-failure cases at malloc calls 1 and 2). Generated C `/O2` remains separate from candidate Release validation.
- Changes: private implementation and local verifier expanded with a transitive C case. Normative specs, public CLI, SDK ABI, Go, DB, CI and hygiene are unchanged; no push.
- Next: other SDK storage bridges; full provenance/path/dominance/initialization proofs, foreign ABI, dispatcher/self-hosting and stage1/stage2 promotion remain. Linux, candidate Release and the full repository suite are deferred.

## Candidate Buffer C execution (2026-10-10)

Receipt: `docs/ops/evidence/candidate-buffer-c-execution-20261010.json`, recorded against parent `466d0864` with 13 current source identity hashes and generated-C source-byte hashes matching the nine native outcomes. Final results are 9/9 native, 10/10 generated-C executions and 3/3 Go verified-MIR checks. This adds bounded Buffer and raw-parts C execution; earlier records remain historical.

- Bridge: new Fairy modules `module_unit_c_buffer.fy` and `module_unit_c_raw_views.fy` verify the exact pinned `std::bytes::Buffer` resource (size 32, alignment 8, field offsets 0/8/16/24), including its `ptr mut void` and `usize` fields. SDK codes 9..15 use real typed `fy_std_buffer_*` runtime prototypes. Wire/live-owner copies use `fy_unit_copy`; initialize copies back only on success, preserving the error owner.
- Raw parts: shared/mutable raw byte views and `str` raw-parts C constructors are admitted; CStr remains closed. Verification rechecks exact builtin name, lexical unsafe scope, three known arguments, safe-owner mutability, raw u8/void pointer, usize extent, result kind and projected-owner origin under bounded work.
- Buffer execution: exact `A NUL λ Z` copy, mutable first byte `B`, append 33, reserve 32, resize 9 with zero fill, invalid truncate without change, clear/empty and two cleanups execute. Initial allocation failure at malloc 1 and growth failure at malloc 2 return `.err` status 1, preserve data/capacity/length and each clean up once. Success/initial-failure/growth-failure heap attempts are 3/1/2 and drops 2/1/1; every case ends with `heap_live=0`.
- Raw-view proof: local u8 view writes 9, extent evaluates once and null with zero extent is supported; five metadata mutations and work-budget exhaustion reject.
- Validation: final locked source check passes 268 source files and 3380 MIR functions. Native cases pass 9/9 (three Buffer cases, three raw-view/guard cases and three Arena/SDK-budget/placement-budget regressions); all nine were executed for this unit, with no reused outcomes. All three Buffer programs pass Go experimental verified-MIR checks, each with 28 functions and 7 foreign functions.
- Generated C: four programs pass MSVC `/Od` and `/O2` against actual heap and Buffer runtime inputs (8/8 executions), and the freshly executed Arena regression passes 2/2: final 10/10 executions, with no reused outcomes. Generated C `/O2` is separate from candidate Release validation.
- Changes: private API and local C verifier expanded; normative specs, public CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push.
- Next: Vec storage intrinsics and runtime bridge, then String/File. Whole-SDK coverage, self-hosting, full provenance/path/dominance/initialization proofs, foreign ABI, dispatcher and stage1/stage2 remain pending. Linux, candidate Release and the full repository suite are deferred.

## Candidate Vec C storage execution (2026-10-10)

Receipt: `docs/ops/evidence/candidate-vector-c-execution-20261010.json`. This extends the private owned-body C emitter to authenticated SDK Vec storage; earlier Vec and Buffer records remain historical.

- Storage: all seven typed operations (size, alignment, take, put, element address, shared borrow and mutable borrow) emit C from retained Fairy IR. The pinned `std::vec::Vec<T>` descriptor is checked against size 48, alignment 8 and field offsets 0/8/16/24/32/40. Runtime codes 3..8 use the real `stage0_vector_storage.c` void-pointer boundary, whose memcpy transport preserves C aliasing rules.
- Ownership: push/pop, reserve, truncate and automatic reverse cleanup execute for scalar and resource elements. Initial malloc failure returns the original donor; growth failure at malloc 2 returns donor 9 and preserves the eight-element prefix and capacity. Actual runtime counters confirm one successful release per program and zero live allocations after every run; cleanup traces are exactly `3,2,1,`, `7,` and `9,8,7,6,5,4,3,2,1,`.
- Views: shared/mutable i64 slices, single-element references and nominal struct slices execute against the actual allocation. Mutable slice and element writes are observed after the borrow ends; Pair values keep their nominal layout through push, whole-element replacement and pop.
- Admission: storage calls require the retained authentic SDK module, exact callee/source span, argument and result types, lexical unsafe authority and owner-derived borrow provenance. The existing destructor-only folded `take_slot<T>` leaf additionally requires its exact Vec receiver element identity and inherited unsafe flag. Six forged metadata modes and work exhaustion reject. Unused SDK-only Vec signature instantiations need no storage/destructor; a retained body using that owner still requires a materialized shape.
- Validation: locked source check passes 270 source files and 3389 MIR functions. Native generation/guard tests pass 6/6; five generated C programs pass MSVC `/Od` and `/O2` (10/10 runs) against actual SystemHeap and vector runtime inputs. All five programs pass the Go workspace verified-MIR check with its authenticated, on-demand Vec SDK. Generated C `/O2` is separate from candidate Release validation.
- Changes: private Fairy implementation, tests, local verifier and source lock expanded. Normative specs, stable CLI, SDK ABI, Go compiler, DB and CI are unchanged; no push.
- Remaining: nested-resource Vec execution coverage, String/File storage, whole-SDK coverage, full provenance/path/dominance/initialization proofs, foreign ABI, dispatcher and stage1/stage2 promotion. Linux, standalone candidate Release and the full repository suite remain deferred.

## Candidate nested-resource Vec C execution (2026-10-10)

Receipt: `docs/ops/evidence/candidate-nested-vector-c-20261010.json`. This closes the nested-resource Vec execution coverage left by the preceding Vec storage unit. Implementation sources and source lock are unchanged; their passing source check is reused after exact hash comparison.

Two new native emitter cases pass 2/2, Go workspace verified-MIR checks pass 2/2, and actual heap/vector C runtime execution passes MSVC `/Od` and `/O2` (4/4 runs). A Vec of Items containing two Leaf resources moves through a function return into a Holder resource. Pop destroys Item 2 and its right/left leaves; Holder cleanup then destroys Item 1 and its right/left leaves: `102,22,21,900,101,12,11,`. Initial allocation failure returns Item 7 with both original leaves intact and records `107,72,71,900,`. Every run attempts one allocation, releases Vec storage exactly once and finishes with zero live allocations.

The initial fixtures tried to mutably borrow a resource field. Candidate and Go both rejected E1008 at matching spans, consistent with the existing language boundary. Corrected fixtures mutate the local Vec and then transfer it into the Holder; no field-borrow contract was changed. Only tests, the local verifier and progress/evidence documents changed; public API, DB, repository hygiene and CI are unchanged, with no push. String/File, whole-SDK and full proof coverage, dispatcher/self-hosting, stage1/stage2, Linux and standalone candidate Release remain.

## Candidate String/CString C storage execution (2026-10-10)

Receipt: `docs/ops/evidence/candidate-owned-text-c-20261010.json`. Authenticated `std::text::String` and `CString` storage is now admitted through existing real runtime storage functions and raw record bridges. Both 32-byte, 8-aligned descriptors have exact pointer/usize fields and offsets 0/8/16/24; the runtime harness independently asserts the real C layouts. The private CStr raw-parts descriptor retains its NUL-inclusive storage extent and original safe owner; only explicit unsafe data-pointer casts cross the scalar foreign boundary.

- Native generation/guard tests pass 7/7. Six programs pass Go workspace verified-MIR checks and generated C execution against real heap, Buffer, UTF-8 and owned-text runtime components under MSVC `/Od` and `/O2` (12/12 runs). The locked candidate source check passes 271 files and 3390 MIR functions. Generated C `/O2` remains separate from standalone candidate Release validation.
- String copy, append, UTF-8 views, empty text, invalid UTF-8 and CString interior-NUL rejection execute. CString payload and terminating NUL bytes are inspected through an explicit pointer boundary, including empty CString. Initial allocation errors expose text status 1/allocation status 1; append malloc failure at call 2 keeps the original byte sequence and length.
- Buffer-to-String-to-Buffer preserves the original allocation address with no transfer allocation. Invalid UTF-8 returns the original Buffer address and bytes. Every actual C run verifies exact allocation attempts and external SDK String/CString/Buffer drops, successful drop statuses and zero live allocations. Three forged CStr authority/kind/owner modes and work exhaustion reject.
- Unused resource types appearing only in signatures no longer demand storage/destructor materialization; retained-body uses still require their actual shapes. This extends the existing dormant-Vec rule without permitting live owners to lose their layouts.
- Reference compiler fix `c68d3ca6` removes a MIR invariant failure for raw pointer casts of returned byte/text/CStr views. The view producer executes once and unsafe provenance remains explicit; four returned-view cases and three literal/path regressions pass. The core candidate storage implementation remains Fairy code.
- The private emitter, tests, local verifier and source lock changed. Normative specs, stable CLI commands, SDK ABI, DB, repository hygiene and CI are unchanged; the separate Go fix is named above. No push. File storage, whole-SDK and full proof coverage, foreign ABI, dispatcher/self-hosting and stage1/stage2 remain; Linux, standalone candidate Release and the full repository suite are deferred.

## Candidate File C execution (2026-10-11)

Receipt: `docs/ops/evidence/candidate-file-c-20261011.json`. The private owned-body emitter now bridges authenticated File storage and the File result/options records through actual Windows runtime components. The File descriptor is checked at size 16, alignment 8 and offsets 0/8; runtime storage uses memcpy. `read_append` passes a local real Buffer structure and copies its fields back after the synchronous call. Directory iteration and other filesystem operations remain closed.

- Locked source check: 272 files and 3397 MIR functions. Six programs pass Go workspace checks, candidate native C generation and actual MSVC `/Od` plus `/O2` execution (12/12 runs). Final native results combine the first report with the corrected EOF and UTF-8 reruns; each execution's C bytes match its final native output hash.
- UTF-8 file paths, embedded NUL payload, write/write_all, sync, consuming close, read/read_exact, size, position and start/current/end seek execute. Partial read_exact reports status 7 and completed 2 while preserving the untouched suffix. Open OS errors, embedded-NUL path rejection and exclusive creation preserve the original file.
- Scope cleanup closes two real handles exactly once; GetHandleInformation confirms they are invalid after cleanup. Sync/close/replace consumes the replacement path, then append and reading recover the exact new contents. Buffer read_append/read_to_end keep the initialized prefix; malloc failure at attempt 2 preserves both the prefix and file cursor. All fixtures remove their paths; buffered runs verify exact heap attempts, one successful Buffer drop and zero live allocations.
- Fixture fixes made close calls unconditional outside short-circuit conditions, gave helper functions parameters for the existing entry selector, and restored lambda literals after a temporary script read UTF-8 as cp949. Final checks use corrected inputs; initial reports remain identified in the receipt. The shared SDK verifier now reports compact failures for selected cases and runs each file program in an isolated temporary folder.
- Private File/record bridges, tests, local verifier and source lock changed. Normative specs, stable CLI/SDK ABI, Go, DB, repository hygiene and CI are unchanged; no push. `set_len` has its shared ABI bridge but was not individually executed. Other filesystem operations, whole-SDK and complete proof coverage, foreign ABI, dispatcher/self-hosting and stage1/stage2 remain. Linux, standalone candidate Release and the full repository suite are deferred; generated C `/O2` is separate from candidate Release.

## Original file loader in module-unit C (2026-10-11)

Receipt: `docs/ops/evidence/candidate-original-file-loader-c-20261011.json`. The actual `file_source.fy` and `candidate_limits.fy` are now parsed with authenticated text/File/Buffer SDK inputs by the module-unit pipeline and emitted as executable C. The source implementation is unchanged; the wrappers only select and inspect its public load/manifest calls.

Two candidate native generation cases and two Go reference workspace checks pass. Generated C runs under MSVC `/Od` and `/O2` (4/4). The exact NUL/UTF-8 payload, empty manifest, 65536-byte boundary, 65537-byte rejection (code 4), missing/invalid UTF-8/empty/NUL path errors (code 1), and initial Buffer allocation failure (code 3) execute. Successful paths perform three explicit closes; overflow cleans up one open File; failed opens also clean up closed descriptors. The allocation-failure path closes its actual handle. Every run checks File/Buffer cleanup counts, exact heap attempts, zero live allocations and unchanged fixture SHA256 contents.

The prior locked source check is reused after checking unchanged compiler inputs, source lock and tracked compiler tree against `df5d0baf`. Only the integration test, local verifier and evidence/progress records changed; public CLI/API, SDK ABI, normative specs, Go, DB, repository hygiene and CI are unchanged; no push. The original scanner/token path, full dispatcher, native driver and stage1/stage2 remain. Linux, standalone candidate Release and the full suite are deferred; emitted C `/O2` is independently verified.

## Copied byte iteration in module-unit C (2026-10-11)

Receipt: `docs/ops/evidence/candidate-byte-iteration-c-20261011.json`. Named immutable shared byte views and their direct slices now run copied `u8` loops through owned body IR, CFG and private C emission. Capture evaluates the source and bounds once. Each step checks the saved length before copying a byte. Empty input, nested loops, continue, break and return execute.

The retained IR verifier now recognizes loop header patterns and checks iteration variable uses against the loop body scope. CFG verification requires the exact capture/step pair and rejects entry paths that reach the step before capture. Six forged graphs and exhausted work are rejected. The cleanup oracle produces `101,1,102,2,103,3,9,`, with each iteration's defer preceding its local resource cleanup and the outer resource last.

Source check and `check --locked` pass (274 files, 3405 MIR functions). Three candidate native tests and two Go reference workspace checks pass. Generated C runs under actual MSVC `/Od` and `/O2` (4/4); its source hashes match the native report bytes. This expands item 59's module-unit execution coverage. Other iteration carriers and reference items remain closed in this C slice. Original scanner/token, full dispatcher/driver and Stage1/Stage2 work remain. Standalone candidate Release, Linux and the full suite are deferred. Normative contracts, stable CLI/API, SDK ABI, Go, DB, CI and repository hygiene are unchanged; no push.

## Imported body budget separation (2026-10-11)

Receipt: `docs/ops/evidence/candidate-import-body-budget-20261011.json`. Imported ordinary functions previously consumed the same 64 slots as generic specializations, rejecting the original scanner at `diagnostics::report`'s `v::at<Entry>` request. The retained-body table is now bounded at 256 entries, while generic admission still limits 64 total specializations and 16 per declaration. All forwarding and cleanup traversal mark tables use the same retained-body limit. Depth 32, work 1048576, per-file declaration admission and downstream AST/type/body limits are unchanged.

One native boundary test admits 256 ordinary bodies and rejects 257 with 4503, admits 64 generic specializations and rejects 65 with 4503. Ordinary fixtures are split across three provider files to respect the unchanged per-file declaration cap. `check --locked` passes (274 source files, 3406 MIR functions). The original scanner now retains 73 bodies and reaches C emission; its native C execution is recorded separately. No normative contracts, stable CLI/API, SDK ABI, Go, DB, CI or repository hygiene changed; no push. Standalone candidate Release, Linux and the full suite remain deferred.

## Original scanner report in module-unit C (2026-10-11)

Receipt: `docs/ops/evidence/candidate-original-scanner-c-20261011.json`. The unchanged original `scanner.fy` and its six private dependencies are parsed with authentic Vec/Buffer/IO SDK inputs. The owned pipeline retains 73 bodies and emits 469905 C bytes after temporary parsed sources are released. An explicit filesystem test artifact carries the C output; length and eight SHA256 words bind its bytes without changing the native test runner's 64KiB stdout limit.

One candidate native generation test and the Go reference workspace check/run pass. Actual emitted C executes under MSVC `/Od` and `/O2` (2/2). `report_work` on `let n=1;` matches the Go reference token kinds, trivia, spans, cooked number byte, EOF and zero-diagnostic output exactly. Both profiles verify three actual heap attempts, one Buffer drop, three vector releases, zero live allocations and zero cleanup status errors. The external C wrapper only counts actual allocation/cleanup calls; scanner recognition stays in the original Fairy source.

The locked source check from `693e6fc3` is reused after verifying unchanged compiler/SDK/runtime source, image and source lock (274 files, 3406 MIR functions). Only the integration test, local verifier and evidence/progress records changed; normative contracts, stable CLI/API, SDK ABI, Go, DB, CI and repository hygiene are unchanged; no push. Scanner error/UTF8/string/comment/budget/failure matrices, the file-to-scanner path, full dispatcher/driver and Stage1/Stage2 remain. Standalone candidate Release, Linux and the full suite are deferred; emitted C `/O2` is independently verified.

## Original file-token path in module-unit C (2026-10-11)

Receipt: `docs/ops/evidence/candidate-original-file-tokens-c-20261011.json`. The unchanged `file_tokens::run` now executes original file loading, scanner collection, TSV formatting and diagnostics through the owned module pipeline. Sixteen exact input files (ten private sources and six authentic SDK sources) retain 105 bodies and emit 584990 C bytes. Native C artifact length/SHA256 is verified after temporary parsed owners are released.

A focused Release native generation test passes with a fresh validation-only bootstrap whose C build deadline is 120 seconds. The default development API execution exceeded 60 seconds, and the first Release compilation exceeded its original 60-second build deadline; neither is counted as a pass. The repository's Go sources and the native test's explicit 60-second process deadline are unchanged. Standalone compiler candidate Release is still deferred.

Actual emitted C executes under MSVC `/Od` and `/O2`. Sixteen cases per profile match the Go-built authored file-token component's exact exit/stdout/stderr: empty and Korean/lambda paths, numeric/string/byte/cstring literals, BOM/comments, malformed tokens/UTF8, CR/LF diagnostics, 65536-byte acceptance, 65537-byte rejection, missing files and wrapper argument errors. All 24 actual SDK heap allocation positions in the literal fixture are then failed independently per profile. Every injected failure returns 2 with no partial token stdout and the original allocation error. All 80 executions verify zero live allocations and cleanup errors, balanced explicit/automatic File closure, invalid handles after automatic cleanup, and unchanged fixture content hashes.

The prior locked source check (274 files, 3406 MIR functions) is reused after verifying unchanged compiler/SDK/runtime sources and lock. Source implementation, normative contracts, stable CLI/API, SDK ABI, DB, CI and repository hygiene are unchanged; only the integration fixture and local observer/evidence records changed. No push. Main/full dispatcher and native driver integration, explicit scanner budget matrices, compiler timeout/performance work, Linux, the full suite and Stage1/Stage2 remain.

## Original scanner budget matrix in module-unit C (2026-10-11)

Receipt: `docs/ops/evidence/candidate-scanner-budget-c-20261011.json`.

- Original `scanner::report_work` and file loading: 14 authentic input modules, 88 retained bodies, 551981 C bytes. Native Release artifact generation passes 1/1; artifact length and SHA256 match its report. The validation-only host compile timeout is 120 seconds; native execution keeps its 60-second limit.
- Exact authored Fairy sources built with the Go Stage0 compiler: locked reference check passes (17 source files, 264 MIR functions). This reference executes the same Fairy scanner, rather than independently implementing recognition.
- Byte/token/diagnostic limits, empty/BOM/UTF8 inputs, revision-specific `defer`, invalid arguments and work exhaustion: 38 cases, 86 actual C executions across MSVC `/Od` and `/O2`, including five allocation-failure positions per profile. Exact exit/stdout/stderr and resource counts agree. Contiguous spans cover every source byte through EOF; budget recovery retains E0010. Heap live count and cleanup errors stay zero; file closures balance and scratch inputs remain unchanged.
- Exact work completion thresholds: ordinary `let n=1;` 8, escaped Unicode string 17, nested comment 27. Zero, below, exact and above bounds execute in both profiles.
- Shared verifier default-mode regression: 16 reference cases and 80 file-token C executions pass. Compiler/SDK/runtime sources, bootstrap hashes and source lock are unchanged; the prior compiler source check is reused. Stable CLI/API, SDK ABI, normative contracts, Go, DB, CI and repository hygiene are unchanged; no push.
- Remaining: full original dispatcher/driver integration, default compilation timeout/performance, standalone candidate development/Release, Linux, full repository suite and Stage1/Stage2.

## Complete original main source-set admission (2026-10-11)

Receipt: `docs/ops/evidence/candidate-main-source-set-admission-20261011.json`.

- Exact original main import closure: 228 private and six authentic SDK modules, 1646578 raw bytes, 1646811 bytes including separators. Native parsing, exact catalog source snapshots and bounded grouping pass for every file, producing 23 units with contiguous complete coverage.
- The prior native attempt failed at original `module_unit_generics.fy` with parser code 4. Parser expression storage now shares the existing 16384-expression joined-unit limit. All other parser limits retain their prior values.
- Aggregate input admission is separately bounded at 2 MiB. Individual files stay at 64 KiB and individual joined owners at 1 MiB. Catalog name storage, file count and downstream unit arenas retain their existing limits.
- Six focused native cases pass: complete original source set, source/file/separator boundaries, large-span HIR/MIR generation, parser expression boundary and two existing grouping regressions. The first expression fixture hit the unchanged statement limit; only that case was corrected and rerun. The other passing results are reused.
- Fresh source check and locked check pass (274 files, 3407 MIR functions). Architecture contracts and the source lock are updated. Go, stable CLI/API, SDK ABI, DB, CI and repository hygiene are unchanged; no push. Native generation uses the previously recorded validation-only 120-second host compile overlay; each test execution retains its 60-second cap.
- This admits the complete dispatcher input. Whole-dispatcher signatures/body closure/typing/C generation and execution, native driver integration, default compilation timeout/performance, standalone candidate development/Release, Linux, the full suite and Stage1/Stage2 remain.

## Complete original main signatures and root binding (2026-10-11)

Receipt: `docs/ops/evidence/candidate-main-signatures-and-binding-20261011.json`.

- Complete original source closure: 234 files, 1646708 raw bytes, 1646941 separator-inclusive bytes, 23 units. Native signature construction retains 2070 function signatures, 757 canonical types and 107 SDK Vec descriptors. Every source snapshot matches; original `main.fy` binding and independent coordinate validation pass with 4890 work.
- The initial full signature build rejected with 4503 because the source-wide Vec descriptor pool still used 64 slots. Its nominal metadata pool now has a separate 256-entry limit. Canonical types remain limited to 4096; executable generic instances remain 64 per unit and 16 per declaration, and each unit's vector layout limit remains 64.
- Six focused native cases pass: original source admission, complete signatures/root binding, three existing Vec identity/ownership/mutation regressions, and isolated 256/257 descriptor admission. The boundary includes five SDK scalar-view descriptors and 251 user nominal descriptors spread across five providers to respect the existing 128-declaration file cap. Initial fixture count/file-cap errors were corrected; the three unchanged passing regression cases are reused.
- Fresh source check and locked check pass (274 files, 3407 MIR functions). The architecture contract and source lock are updated. Go, stable CLI/API, SDK ABI, DB, CI and repository hygiene remain unchanged; no push. Native execution retains its 60-second limit with the previously recorded validation-only 120-second host compile overlay.
- Remaining: concrete dispatcher call closure and body typing, whole-dispatcher C generation/execution, native driver integration, default compilation timeout/performance, standalone candidate development/Release, Linux, full suite and Stage1/Stage2.

## Original main unit closure and external call contracts (2026-10-11)

Receipt: `docs/ops/evidence/candidate-main-unit-closure-20261011.json`.

- `module_unit_call_closure::build_unit` checks external ordinary function signatures while retaining their bodies in their provider units. The existing transitive `build` keeps its previous ownership behavior and executable specialization limits. Source AST call checking retains effect, must-use and unsafe requirements, including calls inside loops, region bodies and conditional branches.
- Native Release execution consumes the exact 235-file original main import closure. Root binding, two closure passes and four root value contexts pass with 19928 work and no copied external ordinary bodies. This removes the root's transitive specialization overflow by assigning ordinary provider bodies to their own units.
- Focused native cases type both sides of an ordinary external call, retain exact target coordinates and parameter signatures, reject bad argument types, missing effects, ignored must-use and unguarded unsafe calls, and preserve the existing transitive call regressions. Explicit discard, returned must-use values, matching effect authority, unsafe blocks and unsafe functions are accepted. An initial nested-loop fixture used invalid syntax. A legacy nested-cleanup fixture also used a struct-owned resource field, which correctly rejects with 1002 under the existing field contract; its outer type is now a resource. Only these two cases are rerun; six unchanged passing cases are reused.
- Caller MIR lowering succeeds, but standalone C rendering rejects a missing provider body. Project linking still needs to type all selected provider units, reconcile their canonical type/specialization identities and supply executable bodies before whole-dispatcher C generation can proceed.
- Source and locked checks pass (275 files, 3417 MIR functions). Native tests use the existing validation-only 120-second host compilation overlay and 60-second execution cap. No Go, stable CLI/API, SDK ABI, DB or CI changes and no push. Whole-dispatcher native execution, native driver integration, default compilation timeout/performance, standalone candidate development/Release, Linux, the full suite and Stage1/Stage2 remain.
