# Go copied closure values: source and semantic stage, 2026-11-08

This receipt covers the owned source-preparation and semantic implementation of
[ADR 0551](../adr/0551-copied-closure-values.md). It does not claim executable,
artifact, cache or native acceptance. Those stages belong to the parent/runtime
workers.

Checkout: `.cache/closure-values-20261108`, detached base
`b4ac33904430281b81f366db21154570496f5165`. Other workers concurrently supplied
syntax, structural types, MIR, backend and Fairy changes in this checkout. The
checks below used their then-present dependency files; the base commit alone
does not describe the tested working tree. No main-checkout writes, commits,
staging, remote operations, policy/limit changes or compiler-host builds were
performed by this source worker.

## Agreed construction and invocation API

* `ast.Function.LiftedClosure *ast.LiftedClosure`, with `CaptureCount int`.
* `ast.Expression.ClosureValue *ast.ClosureValue`, with `Target string`.
* Both AST fields have `json:"-"`. Original parsed AST JSON is compared before
  and after semantic preparation in the source fixtures.
* HIR construction: `ExpressionClosureValue`, `TypeClosureI32`, `Symbol` naming
  the actual lifted helper, and `Children` containing zero through four i32
  capture reads in written order.
* HIR invocation: `ExpressionClosureCall`, `TypeI32`, `Symbol=NoSymbol`, and
  `Children=[callablePath,valueArgument]`. Calls accept named locals/parameters.
* `hir.Function.ClosureTarget *hir.ClosureTarget` has `json:",omitempty"`; its
  `CaptureCount int` excludes the one value argument. Nonnil identifies even a
  zero-capture helper. Ordinary functions omit the metadata.
* Helper parameters are immutable i32 captures followed by the one immutable
  i32 value parameter; the result is i32. `closureprep.ValidateValueTarget`
  rechecks the real safe/pure AST declaration and bounded body. The parent MIR
  mapping is pointer presence to `ClosureTarget` and count to `ClosureCaptures`.
  Downstream verifiers must check actual target declarations/bodies independently.

`ast.Function.SourceGeneric bool`, also `json:"-"`, records the original
declaration's type-parameter presence during AST construction. Actual generic
preparation copies this fact while erasing `TypeParameters`. Semantic checking
rejects closure-bearing signatures and expression transport in those functions,
including transport through an ordinary closure-bearing struct. No generated
function-name prefix is used for this admission decision. This is transient
source provenance, not serialized artifact authority; any reconstruction of an
original generic source declaration must retain/rederive its original
type-parameter fact before specialization.

## Source behavior and bounds

Explicit outer immutable `let fn i32(i32)` literals produce separate real helper
functions and capture-value constructors. Ordinary monomorphic parameters,
results, immutable local copies and ordinary non-repr(C) struct fields retain the
closure leaf. A field must first be extracted to a local for invocation.

The existing untyped direct-closure path retains scalar snapshot locals and
ordinary direct helper calls, including its multiple-argument signature family.
It does not acquire closure-value target metadata.

Source refusals cover mutable/reassigned values, nested literal bindings, direct
literal return, untyped value escape, unsupported signatures/arity/labels,
direct field invocation, wrappers and pointer/reference transport, repr(C),
resources, foreign functions, closure/non-i32 captures, function paths masquerading
as constructed values, and impure/mutable helper bodies. Forward-declared nested
struct wrappers/repr(C) cannot evade the closure-bearing checks. The 0..4 capture,
16 immutable local and 64 expression-node limits remain unchanged; the next
capture/local/node boundary is refused.

## Executed checks

Windows PowerShell, `GOMAXPROCS=1`, unchanged source/product limits:

```powershell
go test -mod=readonly ./internal/closureprep ./internal/semantic -count=1 -timeout 2m
go vet -mod=readonly ./internal/ast ./internal/hir ./internal/closureprep ./internal/semantic
```

* Full closureprep PASS, 0.044s; full semantic PASS, 0.626s.
  Log: `.cache/closure-semantics/attempt5.log`.
* Vet PASS, exit 0; log: `.cache/closure-semantics/vet.log` (empty successful output).
* Owned tracked-file `git diff --check` PASS.
* Positive source fixtures cover factories returning distinct helper targets,
  common invocation parameters, struct storage/copies/extraction, all capture
  counts, admitted body boundaries and original AST JSON stability.
* Thirty-seven forbidden-shape cases pass. Three additional literal-free generic
  forwarding cases run original source through `genericsource.PrepareModule`,
  assert erased template parameters and retained source provenance, rename the
  actual specialized declaration to an ordinary name, and require the semantic
  monomorphic refusal. These cover direct closure parameter/result forwarding,
  parameter invocation, and closure-bearing nominal forwarding.
* The ordering fixture verifies one constructor read per capture in source
  order, construction before later scalar mutation, return of the constructed
  value, distinct helper-frame parameter symbols, and HIR callable-before-value
  argument operands. It retains an unused divide-by-zero initializer in the
  helper. These are source/HIR checks, not claims of observed runtime trap order
  or native frame-lifetime execution.
* The legacy fixture verifies an untyped two-argument closure still uses a
  direct call with its scalar snapshot and has no closure-value target fact.
* Forged target metadata outside the admitted capture/signature family is refused.

Retained failed attempts, all under `.cache/closure-semantics/`:

* `attempt1.log`: test harness used nonexistent `source.Span`; corrected to
  `diagnostic.Span`. Existing immutable-local tests passed in that run.
* `attempt2.log`: fixture module `main` was parsed using `closures.fy`; corrected
  to `main.fy`. `parser-probe.log` retains the investigation.
* `attempt3.log`: focused source/semantic suite passed before later guards.
* `attempt4.log`: new negative borrow fixture used unsupported `&action` syntax;
  corrected to the actual `ref action` syntax. Other new cases passed.

Native/CLI, MIR execution, interpreter/backend execution, artifact/cache replay,
and Linux were not run by this worker. Parent/runtime workers own these gates.
The separately owned AST test
`TestClosureValueTypeStageKeepsExecutionUnsupported` still expects admitted
`apply`/factory source to be rejected at the earlier syntax-only stage; its owner
must update that stage expectation during integration. This worker did not edit
that test or claim the entire repository suite passed.

## Exact owned files and SHA-256

The fifteen Go files below plus this receipt are the complete owned output.
The table records raw file-byte SHA-256 after the passing tests and formatting.

| Owned path | SHA-256 |
| --- | --- |
| `internal/ast/ast.go` | `9563ee72b306bb2f67ac70cf2d396989a882be45d62bd64104ba10d2ad643fcb` |
| `internal/ast/closure_value.go` | `779239fe3ad667735277512b01163bc6b91a113ec7794d1ee91be9eec480f375` |
| `internal/closureprep/prepare.go` | `a9f64de151aa6170f0148472ea6f142c79a5f067cd3efb46593a23d16f8cdec2` |
| `internal/closureprep/values.go` | `c5c2dd030f0ad136b88f335b2ba10d8a3ab72fd19b971f315d199056a585d14a` |
| `internal/hir/hir.go` | `2351c228ef613da66b8cb5886078c72856e3f6d56fc981dfb76fbcc64d1a9eff` |
| `internal/hir/kind.go` | `4afccf70725a84f7b45143a4e19c708e6b4c4f1c6586fbdedc891617d5959b3d` |
| `internal/hir/closure_value.go` | `3482f03bfd5366605067bf9b7a9f61830ad0dc2608b2bbe6d33d759518e03374` |
| `internal/semantic/check_blocks.go` | `00800c2fd98a870702c3f02363eb0e27aff6ade24f4a755f0ce9bd96d79c4c56` |
| `internal/semantic/check_calls.go` | `273ac60916a5381fdf66b47ce123afcc2300c0d22404bd6d752f98bdb7524ecd` |
| `internal/semantic/check_expressions.go` | `f0e9a6b67f410fc6241f15c3c05da21cdc0ab8908faca8f8fd008139a7335ffb` |
| `internal/semantic/check_function.go` | `72ea47efc5aa3fbe5f9b0e0e361c1cf045ac1a15f3f4eb6e4b133a452cf06562` |
| `internal/semantic/resource_types.go` | `88bab00351706544a8cfa952a72082f5389cf40b5a1b8e16f866b9f71badc8d3` |
| `internal/semantic/typecheck.go` | `4c29a1a32a11d2a89e0b2f4991c6f7e25a1ce06bdfc9605d3dd94a63fb087cea` |
| `internal/semantic/closure_values.go` | `eb1c5d4acc75c7114a4c062c6af219d0ab9173f857b01a7ed86395f15b432575` |
| `internal/semantic/closure_values_test.go` | `4e1e02afe594dd236fa18bcf53cc7802074794cb097225be5522dbc2686fecb9` |

Selected concurrent dependencies, read/used but not edited by this worker:

| Dependency | SHA-256 |
| --- | --- |
| `docs/adr/0551-copied-closure-values.md` | `30263ce979955428add524a8ae971d7f823fad2b9b516682a0ce0de587716d85` |
| `internal/ast/type.go` | `612a58bdda754b9dbcca07882c9059553b2098311a9a0c5facaa19ed48a9c9d4` |
| `internal/hir/type.go` | `c0f22e0236f34ec31b741089e0a62dcf443d7952701c56a2d9258cc1abb50d4f` |
| `internal/hir/closure_type.go` | `4672e15615a0459cfc8e3cda86ccd734bf99d6b35976fc5768a7c52d45501148` |
| `internal/genericsource/expand.go` | `4d6b1602d023c41e9e183a1d2154c798bb8627814009b5256184592b42819170` |
| `internal/genericsource/clone.go` | `15612ebf8358f664874d55d626ed74ad28d8c2cc79e5907061da308ee98b6a09` |
| `internal/genericsource/scan.go` | `842e892418d17768c5e0dcec2c088253fc54c64d5c1ded986750553ce1d8f83e` |

No source-of-truth contract, DB, CLI, workflow/runner or public ABI file was
changed by this worker. Internal AST/HIR APIs above are deliberate additions.
