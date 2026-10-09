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

## Direct return-origin checking

After building the native candidate, run:

```powershell
.\build\lexer-candidate.exe --check-return-origins-file sources.txt
```

The manifest uses the same one-relative-path-per-line format as
`--check-modules-file`, relative to the command's working directory. LF and
CRLF are accepted. The command checks direct borrow return origins in one
bounded primary unit and prints original file/byte locations for E2102,
E2103 and their related origins. Status 3 means a semantic origin error;
status 2 means unknown/unsupported analysis, a phase/input failure or a limit.

Borrowed Option/Result payload returns are currently unsupported and return 2.
Inert tagged returns remain admitted. Loan, lifetime, conflict, move and unsafe
pointer provenance checking, and split-unit execution are still pending.

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
