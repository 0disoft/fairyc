# Function-tree closure audit and integration boundary

## Scope and outcome

This audit starts at `d1306c72600ef956ec9dcd2e9e26f6c800788ea0`. Its subject is
one diagnostic-free ordinary function in the private comparison transport, not a
replacement for `frontend.AnalyzeSyntax`. The [encoding owner](body-tree-comparison.md)
and [body-family ledger](body-tree-coverage.md) remain authoritative for this test.
No missing current ordinary-function grammar-family writer route was found in
this bounded review. No parser or transport-tag change was needed. This is a
manual source/route inventory, not proof over all possible inputs or all AST fields.

## Function-level correspondence

| Source grammar / field | Go authority | Fairy authority | Comparison |
| --- | --- | --- | --- |
| `fairy_function_decl`, function/name envelope | Function Span/NameSpan; owning CST function | Declaration kind 3, start/end/name spans | 90, exact source-bound spans |
| `attribute*`, name and argument order | Attributes and Arguments, validated against owning CST paths/literals | Declaration attribute range; attribute argument range | 400 and 401/402/403 path/integer/string |
| `pub`, `unsafe` | Lowered flags checked against CST prefix after attributes and before first fn | Declaration modifier fields | Optional 92/93 leaves |
| `function_name` owner qualifier | OwnerSpan | owner_start/end | Optional 94 leaf; no receiver semantics |
| `type_parameters` | TypeParameters names/order plus CST list/name origins | generic spans and source-order accessors | Optional 95 list, 96 names |
| Result and ordinary parameter types | ResultType and Parameter.Type | result_type and parameter entry type roots | Existing 301-315 type routes |
| Parameter names/order/spans | Parameters, Span/NameSpan | Entry kind 1, filtered within declaration range | 91 plus type child; body entries are not parameters |
| `function_body` and nested syntax | Body, statements/expressions/types/patterns | Body statement and linked arenas | Existing body-family ledger |

Function children have a documented role order, not pure lexical order: attributes,
optional modifiers/owner/generics, result type, parameters, body. Reverse arena links
are normalized to source order. Raw source is emitted once; no cooked value is
inferred from a successful byte/span comparison. Empty attribute parentheses and
trailing commas remain distinguished by raw bytes/spans, not punctuation nodes.

## Exclusions versus gaps

| Surface | Classification and consequence |
| --- | --- |
| Function DocumentationSpan | Intentional exclusion. A documentation-comment witness retains bytes but does not prove declaration attachment or editor metadata parity. |
| Parameter.Mutable | Not a missing current ordinary-parameter production: current grammar is `type IDENT`; the separate legacy `mut name: type` branch sets this field. `ref mut` type mutability is compared. No previous-revision compatibility claim. |
| Expression/attribute Cooked, literal classes and normalized text | Intentional exclusion. Raw spelling/spans and selected syntax tags are not complete semantic values; a consuming adapter must verify its needed decoding and normalization. |
| All header/body combinations | Verification gap, not a missing route. Added a single full-header/complex-body composition and a documentation/attribute/C-function-result witness; combinations remain non-exhaustive. |
| Invalid-input diagnostics and recovery | Intentional exclusion. Both sides reject selected invalid inputs without partial output, but codes, recovery nodes, repair spans and ordering are not compared. Rejection agreement is not diagnostic parity. |
| Module/imports/multiple items and other declarations | Intentional exclusion. Module bytes are bound but module structure is not exported; selection admits exactly module plus one ordinary function. Function-mode rejection witnesses now directly exercise imports, extra functions and foreign declarations. |
| Source/record/depth limits | Deliberate closed boundary, not evidence of realistic-file scalability. Oversized source is rejected; parser and exporter limits differ and neither may silently truncate. |
| Public/default frontend, HIR, execution, bootstrap promotion | Not implemented by this transport. Local comparison cannot certify these surfaces or current hosted CI. |

## Conditions before integration

`internal/frontend/syntax.go` currently returns both a lossless `syntax.Tree` and
an AST only for diagnostic-free input; it verifies exact source reconstruction.
Replacing it requires more than reading this preorder stream:

1. **Name the consumer and supported subset.** Define which real AST fields it
   reads. Start with a private test-only scalar function adapter if an AST/HIR
consumer is the chosen next task; unsupported forms must stop explicitly, not
   be filled with guessed values or silently delegated to Go.
2. **Preserve source and value ownership.** Candidate-owned bytes must outlive all
   spans. Verify literal kind/cooked values, path spelling and type normalization
   needed by that consumer. Include documentation attachment only when the consumer
   requires it; do not claim a complete `SyntaxResult` without the lossless tree.
3. **Define the error boundary.** A clean-input-only adapter may reject diagnostics
   initially. A general frontend replacement must preserve structured diagnostic
   codes, ranges, order and recovery/lossless behavior, not merely nonzero exit.
4. **Define file/package selection.** One function is sufficient for a private
   single-function experiment, not for ordinary files. A file-facing adapter needs
   module/import identity, multiple declaration order, cross-item references and
   explicit handling of every unsupported declaration category.
5. **Verify at the intended integration level.** A private scalar adapter should
   compare field values and semantic results/diagnostics against Go on the same
   source. HIR/native results, target coverage and realistic resource limits are
   separate evidence, not consequences of tree equality.
6. **Keep promotion separate.** Default compiler wiring, stable API, package/cache
   compatibility and release changes require their own reviewed work. Required
   bootstrap promotion still follows the same-commit CI and stage1/stage2 gates in
   [self-hosting readiness](../../docs/architecture/self-hosting-readiness-v0.md).

The first integration unit is a **test-only adapter for one ordinary,
non-generic scalar function with explicit closed input support**, feeding the Go
semantic boundary for differential verification. The implemented private subset
and its evidence are recorded in [scalar semantic adapter](scalar-semantic-adapter.md).
This does not admit the general candidate into the default compiler.
The existing consumer is `semantic.Analyze([]semantic.File)`; each File holds an
`*ast.Module` and a source-relative Path. A prototype must populate and validate
the module identity as well as its function; function records alone are not a File.
It must not expose a decoder for arbitrary untrusted transport, replace the default
frontend, or require implementing all excluded language features first.

## Verification for this audit

`TestWindowsFairyParserBodyTree` runs 39 positive function inputs and 43 rejections
per low/optimized Windows MSVC profile, plus the unchanged 112 positive bodies and
57 rejections. The full composition combines attributes, owner, modifiers,
generics, parameters, nested record types/construction, for/if, region, defer,
discard, propagation/slice access and switch/unsafe/reference/cast syntax.
These programs are syntax witnesses, not promises of semantic validity.
Formatting and scoped strict Mustflow validation are separate checks. Full compiler,
release, other-host and hosted-CI validation are not claimed by this audit.
