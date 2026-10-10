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
