# Go closure value structural types, 2026-11-08

This is the syntax/type-retention stage of accepted ADR 0551. Work was performed
in `.cache/closure-values-20261108` at baseline
`b4ac33904430281b81f366db21154570496f5165`. This stage grants no executable closure
value support and makes no native runtime claim.

## API for the semantic owner

- `syntax.TypeFormClosureFunction` has value `"closure-function"`. A safe
  `fn result(parameters)` type is a parser-owned `KindType` node with this form.
  Its type children contain the written parameters in order, then the result,
  matching the existing foreign-function type child convention. A zero-parameter
  signature therefore has one child, its result. Existing parser budgets apply.
- `ast.TypeKindClosureFunction` has value `"closure-function"`.
  `ast.TypeReference.Result` and `.Parameters` preserve the structural signature;
  `.ABI` stays empty. `.Text` and `.Span` retain the original spelling and byte
  interval, including comments inside the signature. Lowering follows the node
  form and children rather than parsing `.Text`.
- `hir.TypeClosureI32` is the interned builtin leaf spelled `fn i32(i32)`.
  `hir.IsClosureI32(hir.Type) bool` recognizes exactly that leaf. It does not
  admit a wrapper or inspect arbitrary signature text. `hir.ParseType` recognizes
  this exact canonical spelling through the builtin table; other safe function
  spellings stay opaque. Existing text/JSON transport round trips the leaf.

The next semantic owner must check the AST kind, a direct named `i32` result,
and exactly one direct named `i32` parameter before mapping to
`hir.TypeClosureI32`. Source whitespace/comments must not decide admission;
grouped, generic, wrapper or other signatures must not gain admission merely
because their text resembles the canonical spelling. The new leaf supplies no
capture, lifetime, construction, call or verification rules by itself.

The C callback remains `TypeKindForeignFunction` with ABI `C`, and its HIR
callback identity stays separate. Existing untyped closure literals and callback
parsing retain their original paths. The previous language revision continues to
reject the new safe function type syntax.

## Scope and closed boundaries

The grammar now retains general structural safe function signatures, including
nested result/parameter types. This is structural retention, not executable
admission. Current semantic analysis still rejects the canonical factory and
value-call examples, noncanonical signatures, wrappers, borrows, raw pointers,
arrays and foreign declarations carrying this type. Source generic expansion
continues to reject a closure type argument without mutating its input AST.
The exact HIR helper and existing C callback signature checks also reject
wrapper/foreign admission independently.

No semantic, Fairy, MIR, backend, runtime, cache, ADR, queue or lock files changed.
The only normative source-of-truth change is `spec/grammar/stage0.ebnf`.
There are no database, runner, dependency or repository-hygiene changes.
No commit or remote action was performed.

Changed files:

- `spec/grammar/stage0.ebnf`
- `internal/syntax/tree.go`
- `internal/syntax/declaration.go`
- `internal/syntax/closure_type_test.go` (new)
- `internal/ast/type.go`
- `internal/ast/closure_type_test.go` (new)
- `internal/hir/type.go`
- `internal/hir/closure_type.go` (new)
- `internal/hir/closure_type_test.go` (new)
- `docs/development/go-closure-value-types-20261108.md` (new)

## Focused validation receipt

Windows, `GOMAXPROCS=1`, 2026-11-08:

```powershell
go test -mod=readonly -count=1 ./internal/syntax ./internal/ast ./internal/hir -run '^(TestClosureValue|TestType|TestOrdinaryCallback|TestParsedType|TestLowerPreservesNestedTypeStructure$|TestLowerPreservesCShapedTypeBinding$|TestCopiedClosureSyntaxRetainsSource$|TestParseBudget|TestNestedTypeClose)' -v
go vet -mod=readonly ./internal/syntax ./internal/ast ./internal/hir
git diff --check
```

All passed. The final focused test run reported syntax 0.046s, AST 0.044s and
HIR 0.070s. Vet completed successfully with no output.

The new tests cover exact UTF-8/CRLF byte origins and signature slices;
parameter/result order and nested signatures; typed declarations and bindings;
malformed signatures, missing-close repair origin and bounded recovery; existing
depth/node/work limits; structural lowering after raw type text is deliberately
replaced; distinct safe/C callback AST and HIR identities; canonical HIR text,
JSON and map-key round trips; noncanonical/malformed callback spellings; and
unchanged representative scalar/wrapper/C callback serialization bytes.
Existing parser budget, nested type close, copied closure syntax, AST type
lowering and HIR identity/transport regressions passed in the same run.

An initial test harness build expected two results from `semantic.Analyze`; its
single-result API was used before the passing runs. No unresolved test failures
remain. Full-suite, native C11/C17 and runtime/cache checks were not run because
this assignment changes structural types only. Executable admission and
Go/Fairy runtime parity remain work for later ADR 0551 stages.
