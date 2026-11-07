# Candidate closure value structural types, 2026-11-08

ADR 0551 Fairy **type-retention stage only**, in isolated checkout
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/closure-values-20261108`,
baseline `b4ac33904430281b81f366db21154570496f5165`.

## Changed files and representation

This worker owns exactly these five changed/new files:

- `examples/lexer_candidate/src/types.fy`
- `examples/lexer_candidate/src/generic_source_type_json.fy`
- `examples/lexer_candidate/tests/closure_value_types.fy` (new)
- `internal/cli/candidate_closure_value_types_test.go` (new)
- `docs/development/candidate-closure-value-types-20261108.md` (new)

The syntax type-node audit found tags 1 through 16 already occupied: paths,
borrows, slices, grouping, generic applications/links, arrays/lengths, raw
pointers, generic parameter names/links, C function signatures (15), and
ordered function parameter links (16). Tag **17** was unused in this syntax
node family and now represents a safe closure-function type. Expression,
statement, and resolved module-graph tags are separate families and were not
repurposed.

`fn result(parameters)` produces tag 17, with `left` holding the structural
result and `right` holding the existing kind-16 parameter tail. Written
parameter order, child structure and exact source byte intervals survive.
`unsafe extern C fn ...` remains tag 15. The serializer emits
`"Kind":"closure-function"`, structural `Parameters`/`Result`, null `Element`,
and empty `ABI`; C functions retain `"foreign-function"` and ABI `C`.
No old JSON field ordering, spelling, null representation or escaping changed.

No `bodies.fy` or syntax accessor change was required. Declaration signatures
and explicit typed bindings already reach `bodies::read_type`, which calls
`types::parse`; the native probe verifies both paths, including a comment
inside a binding's written type and its exact type start/end.

## Independent oracle and native coverage

The new Go test uses the real `frontend.AnalyzeSyntax` AST and `json.Marshal`
on its original structural `ast.TypeReference` records. It supplies a binary
`FYCLTYP1` oracle to the actual Fairy parser/serializer; it does not expand
source, replace a provider/body, or emulate Fairy behavior in Go. The oracle
contains 10 original source cases, 141 structural type records, 25 safe
function signatures, and eight malformed controls.

Coverage includes the ADR's returned factory/apply source, typed copies,
declaration results/parameters, struct fields, UTF-8 comments, CRLF byte
offsets, comments inside signatures, zero parameters, trailing commas,
noncanonical scalar signatures, nested function results and parameters,
generic types, grouping, arrays, Option/Result, references and raw pointers.
Every selected original type and recursive child is compared against the
entire Go canonical JSON byte string. Function signatures additionally check
the result interval and every parameter interval in written order.

Existing scalar/wrapper/array/pointer/C callback sources are included in the
same native comparison. Independently constructed Go AST expectations pin
representative legacy scalar and C callback JSON, including pointer
mutability, nil fields, ABI and byte spans.

Malformed controls reject a missing close, empty parameter, named parameter
inside a type, capture syntax inside a type, `unsafe fn`, `extern C fn`
without `unsafe`, missing parameter list, and missing result. The missing
close's Go E0007 byte origin is compared with Fairy's retained next-token
origin. Existing Go depth/node/work limits reject bounded controls. Fairy
checks depth 31 accepted versus 32 rejected, exact 4096-node capacity versus
one additional required node, exact JSON work exhaustion versus one unit
over, JSON depth rejection, sticky failure, and rejection of internal links
as standalone types. No bounds were changed.

Type-only declarations/bindings succeed through normal `syntax::parse`
without closure preparation. The original typed factory literal retains
its structure through `syntax_parse_driver::parse_diagnostics` but normal
`syntax::parse` still rejects it with code 4. The unchanged `module_types`
resolver rejects the new syntax type while preserving the seeded original
graph record and record count. This stage admits no executable closure value
and does not change the module type graph.

## Validation receipt

Windows AMD64, `GOMAXPROCS=1`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueTypeOracle$' -count=1 -timeout=3m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueTypes$' -count=1 -timeout=8m -v
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_closure_value_types_test.go
git diff --check -- examples/lexer_candidate/src/types.fy examples/lexer_candidate/src/generic_source_type_json.fy examples/lexer_candidate/tests/closure_value_types.fy internal/cli/candidate_closure_value_types_test.go docs/development/candidate-closure-value-types-20261108.md
```

- Go oracle/fixture syntax attempt 1: **PASS**, test 0.01s, package 0.159s.
- Native parser/type attempt 1: **PASS**, test 14.48s, package 14.605s.
  The test copies the authored import closure once into its temporary project,
  runs the real CLI `check`, then runs the same frozen source/oracle with both
  `run -- <oracle>` and `run --release --locked -- <oracle>`. Both actual native
  products exit 0 with empty stdout/stderr. All five fixture checks execute in
  each product; no full candidate CLI host is built.
- Scoped CLI vet and gofmt: **PASS**, no diagnostics or unformatted paths.
- Scoped whitespace check: **PASS**.

Logs are retained in this isolated checkout under
`.cache/closure-value-types-go-attempt1.log`,
`.cache/closure-value-types-native-attempt1.log`, and
`.cache/closure-value-types-vet.log`. Both first test attempts passed; no
failed native attempt was discarded or reported as a pass.

Final frozen source SHA-256 (receipt itself excluded):

| File | SHA-256 |
| --- | --- |
| `src/types.fy` | `6142EA5D80A67E2A28DEAA90FD28E0B70C74067342E56255FC42ED964FE8FE87` |
| `src/generic_source_type_json.fy` | `52CB3917890DDCB7F274609E290DC78BCDD9BADAAB1A4B7F75E219BE88B27168` |
| `tests/closure_value_types.fy` | `2D2255990B699F2C565836801F0C60448B65E25731D5AF7986F5DD357C69C91C` |
| `internal/cli/candidate_closure_value_types_test.go` | `123456B881917F3F83D76D60D6515B9453F5904A103469FE774B3A3ECCCB1E22` |

The first three paths are relative to `examples/lexer_candidate/`.

Whole-source bootstrap/self-build, full CLI rebuilding, runtime/MIR/artifact
admission, package/cache execution, Linux and remote checks were not run for
this structural stage. Future execution and serialization owners must review
their own format/admission gates. No source-of-truth contract, ADR, lock,
queue, shared syntax accessor, Go production file, runtime, database, runner,
dependency or repository-hygiene configuration was changed by this worker.
Other workers' changes in this checkout are outside this receipt. No main-tree
copy, commit, lock regeneration, Linux or remote action was performed.

## Coordinator integration

The five frozen files above were copied into main without changing their
tested bytes. A fresh copy of the combined authored tree passed `check`
(16.028s) and `check --locked` (19.401s), each with 758 compiled sources and
9,594 MIR functions. All 1,231 FY files matched both the snapshot and main
before the newly verified lock was copied. Evidence:
`.cache/closure-value-types-source-20261107-all-check.json` and snapshot
`.cache/closure-value-types-source-all-d609db5d`. This verifies source
integration and lock freshness; it does not admit executable closure values.
