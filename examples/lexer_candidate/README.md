# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

Example version 0.19.0 adds C11 emission and native execution verification for
same-file nominal structs (see [Owned nominal C](#owned-nominal-c)). The candidate
owns recognition, typing, HIR, MIR and C generation; the native C compiler and
runtime link the resulting unit. This remains an experimental compiler slice.

## Owned consumer API

`scanner::collect` consumes an input `bytes::Buffer` and the same explicit revision
and byte/token/diagnostic/work limits as recognition. It returns
`Result<scanner::Stream, i32>` without console output. Stream owns the original
source, token storage, cooked bytes and diagnostics; input is consumed on both
success and failure. Status 3/4 means allocation/storage failure and publishes no
Stream. Reference-budget exhaustion instead returns a complete lossless Stream
containing `E0010`.

`scanner::release(stream)` explicitly consumes and destroys the result. Ordinary
scope exit also destroys it; the API does not use `discard` as an early-release
operation.

`token_count`, `token_info`, `token_raw`, `token_cooked`, `source_view`,
`diagnostic_count`, `diagnostic_info` and `diagnostic_token` provide read-only
access. Metadata is copied; returned byte views borrow the Stream and cannot
outlive it or coexist with its move/drop. Indexed access retains the vector bounds
trap. Diagnostic access uses recognition order, while the legacy report sorts for
display. Numeric kinds/codes and cooked encoding remain experimental implementation
contracts, not a stable SDK or public ABI. Native consumers exercise transfer and
budget exhaustion; compile-fail tests cover local escape and use after/drop during
a live borrow.

## Regular file token dump CLI (`--tokens-file`)

A new front end adds a regular file mode without changing the binary stdin protocol:
the no-argument frame reader stays exactly as described below, and
`--tokens-file <path>`, `--declarations-file <path>`,
`--check-names-file <path>`, `--bindings-file <path>`, `--check-types-file <path>`,
`--check-modules-file <manifest>`, `--hir-file <path>`, `--mir-file <path>`, `--c-file <path>`, and `--help`
are the file-mode arguments.

```sh
lexer-candidate --tokens-file path/to/source.fy
```

A checked-in `samples/token-demo.fy` demonstrates it:

```powershell
.\build\lexer-candidate.exe --tokens-file samples/token-demo.fy
```

The supplied path is UTF-8. `src/file_cli.fy` opens it read-only and reads at most
32768 source bytes plus one sentinel byte, so any 32769th byte trips the existing
byte limit. It collects current-revision tokens only, through the same
`scanner::collect` entry point and the default byte/token/diagnostic/work budgets
used by the binary stdin driver; it does not accept the previous revision and does
not add scanner rules. It accepts any supplied regular file name, with no `.fy` suffix
requirement, and the path argument is bounded at 4095 bytes.

`src/token_output.fy` writes one TSV row per token, including trivia and the single
EOF token. Each row has six tab-separated columns: a human kind label, a human
trivia label, the half-open start and end byte offsets into the original source, and
the lowercase hexadecimal bytes of the raw source slice and of the scanner's cooked
representation. Dumping reuses the existing read-only Stream access
(`token_count`, `token_info`, `token_raw`, `token_cooked`, `source_view`) and
releases the Stream afterwards; no candidate tokens are precomputed and Go supplies
none. The dump is capped at 1 MiB: each row's conservative bound (96 plus twice the
raw length plus twice the cooked length) is checked before append.

Any scan diagnostic makes the dump fail closed: stdout stays empty, the process
exits 2, and stderr carries one line per diagnostic in recognition order:

```text
path:LINE:COLUMN: error: E000x (bytes START..END)
```

`path` is the path exactly as supplied, `LINE` and `COLUMN` are 1-based byte
positions over the original source, and the half-open `START..END` bytes are that
diagnostic's span. CLI I/O, usage, allocation and candidate-limit failures print
plain messages without the path/line/column prefix, and `--help` prints the usage.

Scanner rules, the compiler and the SDK are unchanged; this is a read-only reporting
front end over the existing candidate. The default diagnostic budget and the
`E0010` exhaustion contract described above are unchanged.

Portable verification uses a test-only Go reference fixture producer plus
`examples/lexer_candidate/verify.py` (Python standard library); both commands run from
the repository root:

```sh
go run ./cmd/fairyc-lexer-file-oracle .cache/lexer-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/lexer-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The `go run` form is the portable default; an optional workspace wrapper is:

```sh
bun tools/run-go.ts run ./cmd/fairyc-lexer-file-oracle .cache/lexer-file-cli/reference.json
```

The file CLI passes on Windows development and release: 29 Go reference files per
profile with their 29 original hashes preserved across 82 CLI executions, covering the
file, I/O, usage, `--help` and legacy-protocol cases. Accepted tokens are compared
exactly for kind, trivia, byte range, raw and cooked bytes, and byte reconstruction;
diagnostics are compared as a multiset of code, span and byte line/column, because the
candidate reports in recognition order while Go sorts. An output-limit fixture
alternates identifiers and spaces to 32768 source bytes, for which the Go oracle
expects more than 1 MiB of TSV while the native binary exits 2 with the output-limit
message on stderr and empty stdout. The narrow `TestLexerFileCLIWorkspace` test, the
CLI vet, the Go oracle vet, and the Linux and Windows C emission checks pass. Actual
Linux execution remains deferred to the owner's consolidated batch. This section makes
no performance claim, and allocation-failure or OS I/O fault injection was not run.
Evidence is `docs/ops/evidence/lexer-file-cli-20261005.json`.

## Regular file declarations CLI (`--declarations-file`)

`--declarations-file <path>` reads the same bounded regular file and reuses
`scanner::collect` plus the byte-only `syntax::parse(stream, 512)`. It is a standalone
structure dump, not a project check, so there is deliberately no basename, module or
path identity validation.

On success it writes one TSV row per declaration in source order with six columns:

```text
kind	name	start	end	name_start	name_end
```

`kind` is one of `module`, `import` (the alias is the name), `function`, `struct`,
`resource`, `const`, `static-assert`, `foreign-function`, `state`, or `enum`. An
unnamed `static-assert` or `state` has an empty `name` and `0..0` name spans; method
name spans keep the member spelling.

Lexical diagnostics keep the existing `--tokens-file` code format. Any parser problem
returns 2 with empty stdout and one located line per problem:

```text
path:LINE:COLUMN: error: expected TEXT (bytes START..END)
```

`syntax::Problem.code == 4` with `expected == 0` gives unsupported syntax; a parse
error 4 gives plain parser-limit. Go parser diagnostic IDs are not fabricated, and
allocation failures stay plain. The same 32768-byte file cap, 4095-byte path bound and
1 MiB output cap apply, and the 512-declaration parser limit counts module and import
items while the existing parser node, field and depth limits are retained.

Portable source invocation:

```sh
lexer-candidate --declarations-file samples/token-demo.fy
```

Optional Windows executable form:

```powershell
.\build\lexer-candidate.exe --declarations-file samples/token-demo.fy
```

`src/declaration_output.fy` writes the dump. Reference comparison runs from the
repository root and needs both references:

```sh
go run ./cmd/fairyc-lexer-file-oracle .cache/declarations-file-cli/tokens-reference.json
go run ./cmd/fairyc-lexer-file-oracle --declarations .cache/declarations-file-cli/declarations-reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/declarations-file-cli/tokens-reference.json --declarations-reference .cache/declarations-file-cli/declarations-reference.json --receipt /path/to/evidence/receipt.json
```

The Windows development and release profiles pass 156 CLI executions (78 per profile),
with 29 token fixtures and 29 declaration fixtures per profile; the per-profile
declaration set is 15 accepted, 9 syntax, 4 lexical and 1 limit. Declaration output is
compared exactly for kind, name, start, end, nameStart and nameEnd against Go
`syntax.Parse` for accepted inputs, and negative inputs are checked against authored
diagnostic goldens; original source hashes are preserved. `TestLexerFileCLIWorkspace`,
`go vet ./internal/cli ./cmd/fairyc-lexer-file-oracle`, and the Linux and Windows
`--locked` C emission checks pass, and the 619 compiler inputs match the manifest with
the compiler EXE hash unchanged. The example package is version 0.2.0 with no compiler
or SDK change. Evidence is `docs/ops/evidence/declarations-file-cli-20261005.json`; the
earlier token-dump receipt above remains the historical record for that mode.

## Regular file name checking (`--check-names-file`)

```sh
lexer-candidate --check-names-file samples/names-demo.fy
```

This private name-only candidate reads the same bounded file and reuses the scanner
and parser. `src/name_resolution.fy` collects ordinary function names before checking
parameters and source-ordered local bindings. The rules follow
`docs/language/0007-name-resolution-and-typing-v0.md` and the Go resolver in
`internal/semantic/resolve.go`. Functions can reference later functions. Parameters
occupy their own scope; the function body and each nested block open child scopes.
Same-scope duplicates fail, nested shadowing is allowed, and an initializer resolves
before its new local is registered. A local is invisible after its owning block ends.

The admitted subset is a module plus non-generic ordinary functions with optional
`pub`, scalar written types (`i32`, `i64`, `u8`, `u32`, `usize`, `bool`, result-only
`void`), inferred locals, integer/bool literals, simple names, arithmetic/comparison/
boolean expressions, ordinary positional calls, parenthesized expressions, nested
blocks, `let`/`var`, returns and simple-name assignment targets. Version 0.6.0
also admits the control-flow name scopes described below. This checks names only: call arity, types, mutability, return obligations, overflow and ownership are
unchecked. For example a named but mistyped return or assignment to a `let` can pass
this command. Those are later type/flow checks, not valid-program claims.

Imports, constants, composite types, fields, references, labeled calls, generics,
attributes, methods, unsafe forms, `switch` patterns and the `source_location` intrinsic
are rejected as unsupported. Output is `names ok` plus newline only after all checks
succeed. The first failure returns 2 with empty stdout and:

```text
path:LINE:COLUMN: error: MESSAGE (bytes START..END)
```

Messages are `unresolved name`, `duplicate name in scope`,
`unsupported name-check syntax`, and `name-check limit`. These are private candidate
messages, not new public Go diagnostic codes. Scanner/parser errors retain their
existing file-mode format. Limits are 128 module functions, 128 simultaneously active
parameters/locals, and 64 active walker levels, in addition to existing file/parser
limits. Sibling blocks release their active-name slots. One error is reported; this
candidate does not provide full diagnostic recovery or bound HIR/MIR.

From the repository root, generate the three test references and run the portable
harness against the desired native compiler:

```sh
go run ./cmd/fairyc-lexer-file-oracle .cache/names-file-cli/tokens-reference.json
go run ./cmd/fairyc-lexer-file-oracle --declarations .cache/names-file-cli/declarations-reference.json
go run ./cmd/fairyc-lexer-file-oracle --names .cache/names-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/names-file-cli/tokens-reference.json --declarations-reference .cache/names-file-cli/declarations-reference.json --names-reference .cache/names-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

Windows dev/release pass 276 CLI executions (138/profile). Each profile includes
50 name fixtures: 19 accepted, 8 unresolved, 3 duplicate, 17 unsupported and 3 limits;
accepted names and the exact unresolved/duplicate spans are independently checked
against Go semantic analysis. The 29 token and 29 declaration fixtures per profile,
malformed input, CLI/file errors and framed stdin smoke also pass. Source hashes are
unchanged before/after every fixture execution. Existing historical receipts remain
separate. Additional target/dependency checks are recorded in
`docs/ops/evidence/names-file-cli-20261005.json`. Linux native execution is deferred to
the owner's batch; allocation/I/O fault injection and performance testing were not
run. This changes the example package to 0.3.0; compiler, SDK and language contracts
are unchanged.

## Retained declaration-use links (`--bindings-file`)

```sh
lexer-candidate --bindings-file samples/names-demo.fy
```

The name resolver now retains its exact `syntax::Parsed` owner together with symbol
and reference vectors in `name_resolution::Resolved`. `resolve(parsed)` consumes
that syntax owner; `problem`, `parsed`, `source_view`, `symbol_count`, `symbol_at`,
`reference_count`, and `reference_at` are shared queries. A symbol ID is a one-based
index local to that result, not a durable cross-file/compiler identity. Every name
expression retains its expression ID, use span and actual target symbol ID.
Symbols retain kind, name span, unique owning scope, function owner, written type
root and syntax declaration/entry coordinates for the next type phase. Locals have
independent IDs even when sibling blocks reuse an active-name slot. Function symbols
are collected first, then parameter/local symbols in traversal order. Failed results
retain source for diagnostics, but table queries assert success and the CLI emits no
partial table. Builders own separate vectors while collecting; the completed resource
owns and cleans them alongside syntax, without mutable resource-field reborrows.

The command has the same admitted syntax and first-error behavior as name checking.
Success emits a TSV table with columns:

```text
record	id	kind	start	end	scope	owner	target
```

A `symbol` row uses `function`, `parameter`, `local-let` or `local-var`; its target is
zero. A `reference` row has kind `name`, its ID is the syntax expression ID, scope and
owner are zero, and target is the symbol ID. Spans identify source bytes; function
symbols use scope/owner zero. Output is fully buffered before publication and bounded
at 1 MiB. Parser bounds also cap storage at 128 module functions plus 1024 entries and
at most 4160 expression-use links.

The portable verification command in the preceding section accepts the regenerated
`--names` reference, now including Go AST symbol inventories and exact Go semantic
use-to-declaration resolutions. The Windows dev/release harness passes 396 CLI executions (198/profile).
Both name and binding file modes are checked for all 50 name fixtures, file/CLI errors, malformed syntax and invalid bytes. Per-profile
accepted inventories and 26 reference targets are compared independently with Go, including
forward calls, shadow initializer targets, restored outer names and sibling scopes.
`tests/bindings.fy::retained_symbol_links` reads the moved source owner and verifies
symbol/type/entry and expression links in both native profiles. Interpreter execution
is unavailable because this component uses the native-only `std::vec` SDK path.
Evidence is `docs/ops/evidence/bindings-file-cli-20261005.json`. The example version is
0.4.0; compiler, SDK and language contracts are unchanged. Linux native execution is
still deferred to the batch; no allocation-failure or OS I/O injection was run.

## Scalar type checking (`--check-types-file`)

```sh
lexer-candidate --check-types-file samples/types-demo.fy
```

Success emits `types ok` and exit 0 after lexing, parsing, name resolution and
checking the complete file. Failure emits one source/line/byte-range diagnostic
to stderr, exit 2 and no stdout. The input file is read only. This mode checks
`i32`, `i64`, `u8`, `u32`, `usize`, `bool` and `void` function results, typed or
inferred locals, arithmetic/bitwise/shift/comparison/boolean operators, immutable
assignment, direct function calls, argument count/types and return types. It
applies ADR 0417's fixed-peer contextual integer rules without implicit width
conversion, checks each literal's range, and preserves the direct-token signed
minimum exception in decimal, hexadecimal and binary.

The reusable `type_check::Typed` resource owns the resolved source and a bounded
SDK vector of scalar type slots. `problem`, `source_view`, `resolved`,
`symbol_type`, `expression_type` and `signed_minimum` expose the retained facts;
type-table queries require success. Private type codes are 0 absent/linkage,
1 i32, 2 i64, 3 u8, 4 u32, 5 usize, 6 bool, 7 void and 8 callable. Function
symbol types describe their results; a callee expression uses code 8. Postorder
linkage nodes carry no value type. Only a direct negative minimum sets the
signed-minimum flag, including its separately retained magnitude child type.

Slots use bounded heap storage, avoiding large fixed-array copies on the Windows
native stack. Allocation failure rejects the file. Additional limits are 4096
syntax expression nodes, 1024 entries, 1152 symbols and 64 active checker levels;
existing source/parser/name limits also apply. Statements after a return are
still checked. The return gate is extended by the control-flow section below;
ownership/initialization Flow remains subsequent work. Imports, composite types,
casts and generics are currently rejected by the preceding name phase.

Generate independent test data and run the portable harness from the repository
root (no PowerShell runtime is required):

```sh
go run ./cmd/fairyc-lexer-file-oracle .cache/types-file-cli/tokens-reference.json
go run ./cmd/fairyc-lexer-file-oracle --declarations .cache/types-file-cli/declarations-reference.json
go run ./cmd/fairyc-lexer-file-oracle --names .cache/types-file-cli/names-reference.json
go run ./cmd/fairyc-lexer-file-oracle --types .cache/types-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/types-file-cli/tokens-reference.json --declarations-reference .cache/types-file-cli/declarations-reference.json --names-reference .cache/types-file-cli/names-reference.json --types-reference .cache/types-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The type fixtures compare pass/fail and the selected error span against Go
semantic analysis, while unsupported syntax and the candidate's storage cap have
explicit rejection checks. CLI/file errors, malformed syntax/UTF-8, source
preservation, existing modes and the framed stdin protocol run in both Windows
native profiles. `tests/types.fy` checks retained symbol/expression types and
minimum flags. Evidence is `docs/ops/evidence/types-file-cli-20261005.json`;
the example is version 0.5.0, with compiler/SDK inputs unchanged. Linux C emission
is checked; Linux native execution remains deferred to the owner's batch.
Allocation/OS fault injection and performance experiments were not run.

## Control-flow name scopes

Version 0.6.0 extends `--check-names-file` and `--bindings-file` to `if/else`,
`else if`, `while`, classic `for`, `assert`, `discard`, `defer`, `break` and
`continue`. Branches and loop bodies have independent lexical scopes. A `for`
initializer has a separate scope visible to its condition, post clause and body;
the post clause is resolved before body-local declarations, and the initializer
binding disappears after the loop. Shadow initializers still see the outer name.

```sh
lexer-candidate --check-names-file samples/control-names-demo.fy
lexer-candidate --bindings-file samples/control-names-demo.fy
```

This is name resolution only: condition/operator types, loop-control validity,
return coverage and defer-call restrictions belong to the following type-phase
unit, connected in version 0.7.0 below. `switch` patterns require the later enum/Option/
Result type component. File, name-slot, recursion, first-error and no-partial-
output policies remain the same. Native `tests/name_control.fy` checks the exact
use targets across outer parameters, loop initializers, body shadows and branches.
Evidence: `docs/ops/evidence/control-names-cli-20261005.json`.

## Control-flow type checking

Version 0.7.0 connects `--check-types-file` to the control-flow name scopes above.
`if/else` and `else if` require bool conditions, propagate the destination type to
branch values and require matching branch results. `while`, classic scalar `for`
and `assert` require bool conditions. `break`/`continue` require an enclosing loop,
and nested scopes preserve the loop context until that loop's body finishes.

Classic `for` admits optional initializer, condition and post clauses. Its header
uses the current scalar subset and rejects calls, blocks and other effects under
ADR 0213; body expressions retain ordinary call semantics. `defer` admits direct
internal void calls with scalar arguments under ADR 0217. Both argument types and
arity are checked. `discard` checks its operand. The candidate performs static
analysis only: it never executes a loop, assertion, deferred call or input program.

The function return gate follows the Go reference: direct return, an exact
`assert(false)`, or complete nested `if/else` coverage satisfies the gate. A loop
by itself is not return proof, and a plain standalone block's nested return is not
promoted. All source branches and statements after a return are still checked.
Ownership/initialization Flow and code generation remain later components.
`switch` requires the later composite-type pattern implementation; it stays
explicitly unsupported here.

```sh
lexer-candidate --check-types-file samples/control-names-demo.fy
```

The portable `--types` oracle now adds normal and rejected control cases from
`cmd/fairyc-lexer-file-oracle/control_types.go`; the existing verification command
uses freshly regenerated references. `tests/type_control.fy` reads actual retained
branch and comparison types, contextual u8 branch values, and return/loop results
in both native profiles. Evidence: `docs/ops/evidence/control-types-cli-20261005.json`.

## Multi-file scalar checking (`--check-modules-file`)

Version 0.8.0 connects explicit imports to the existing name/type/control-flow
walkers. Run from this example directory:

```text
lexer-candidate --check-modules-file samples/modules.txt
```

The UTF-8 manifest contains one unquoted, untrimmed relative filename per line.
Paths resolve against the process working directory, including `..`; this is
explicit read-only input selection. LF, CRLF and no final newline are accepted.
Empty lines, BOM, control bytes, absolute entries and colon/ADS spelling are
rejected. Spaces and Korean filenames are accepted. The manifest filename itself
uses the existing CLI UTF-8 path reader. Limits are 8192 manifest bytes, 4095
bytes per path, 32 files and 32768 bytes per source, with the existing parser and
name/type budgets. Duplicate source selection fails as a duplicate module.

`module_registry.fy` builds module identities, import aliases and scalar function
signatures from the actual parser-owned syntax. Token normalization ignores
comments/trivia in qualified paths without rewriting the source. All files are
registered before imports resolve, so manifest order, forward imports and import
cycles do not prevent lookup. Each import alias and ordinary function name shares
the module declaration namespace. Only an explicit `alias::function` reaches a
public item in its target module; imported aliases do not re-export items. Private
functions remain usable within their own module.

`name_resolution::resolve_modules` retains each qualified expression's declaration
link and copied scalar signature. `type_check` applies the real imported parameter
and result types to the existing contextual integer and call checks. Every source
body is checked, including private functions and modules not called by another
file. Local variable names may shadow an alias for unqualified references without
changing the qualified import lookup. `module_check.fy` and `module_cli.fy` publish
`modules ok` and exit 0 only after the whole set succeeds. Failures exit 2 with
empty stdout and an input or source-location diagnostic on stderr. Inputs are
never modified. Single-file modes retain their existing import rejection.

This mode receives an explicit source set, not a resolved workspace/package graph:
it does not discover files, validate filename/module correspondence, parse
`fairy.toml` or locks, select SDK dependencies, or implement cross-package identity.
Reserved `source_location` item/alias bindings fail closed. Constants, composite
values, generic functions, attributes, effects and ownership
remain outside this scalar component. Unsupported constructs fail closed.

Generate Go semantic reference observations and verify both native profiles:

```text
go run ./cmd/fairyc-lexer-file-oracle --modules .cache/modules-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/modules-file-cli/tokens-reference.json --declarations-reference .cache/modules-file-cli/declarations-reference.json --names-reference .cache/modules-file-cli/names-reference.json --types-reference .cache/modules-file-cli/types-reference.json --modules-reference .cache/modules-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The portable Python harness prepares files only; registry/resolution/typing run in
the Fairy executable. Semantic rejection categories are compared to Go diagnostics,
and candidate byte ranges are checked against the selected source. Candidate
messages are private, not the stable Go diagnostic schema. Manifest/IO/syntax
controls check exact status/output and input preservation. Native `tests/modules.fy`
checks retained import signatures and resolution/type facts. Evidence:
`docs/ops/evidence/modules-file-cli-20261005.json`. Windows native execution is
checked locally; Linux native execution remains deferred to the owner's batch.

## Owned scalar HIR (`--hir-file`)

`lexer-candidate --hir-file <path>` reads the same bounded source file, then
parses, resolves names, checks scalar types and consumes the successful `Typed`
value through `hir_lower::lower`. The returned `hir_model::Program` owns source
bytes, canonical integer text and separate function, binding, expression,
statement, block, edge and signature arenas. It retains no parser or type-check
table. `hir_lower::lower(typed, module_index)` also preserves external function
module/declaration identity when its caller supplies an imported typed input;
the file CLI itself accepts one module without imports.

The private TSV format begins with `fairyc.scalar-hir/v1` and a `P` module-index
record. `F`, `B`, `E`, `S`, `K`, `A` and `T` records describe functions, bindings,
expressions, statements, blocks, ordered edges and signature types. Entity IDs
are one-based; zero means absent. Edge/signature indexes and module indexes are
zero-based. Enum labels and this format are private implementation contracts.

Supported semantics are scalar literals, resolved locals and parameters, unary
and binary operations, ordered calls, blocks and tails, `if`, `while`, classic
`for`, nearest-loop `break`/`continue`, assignment, return, assert, discard and
unevaluated void-call `defer`. Grouping and callee syntax become semantic links.
Integer payload is canonical decimal ASCII, including direct signed minima and
AMD64 `usize` maximum, without overflowing a signed intermediate.

The source cap is 32768 bytes, with at most 512 declarations/functions, 1152
bindings, 4096 expressions, 8192 statements/blocks, 32768 edges/signature entries,
86016 payload bytes and recursion depth 64. The existing parser/resolver/type
budgets apply first. Complete output is buffered under a 2 MiB cap before stdout
publication. Malformed, unsupported, allocation or limit failures return 2 with
empty stdout; scanner/name/type errors retain their existing located messages.
There is no MIR lowering, interpreter, backend or executable compilation of the
input in this mode.

From the repository root, regenerate the independent Go HIR oracle with:

```sh
go run ./cmd/fairyc-lexer-file-oracle --hir .cache/hir-file-cli/reference.json
```

Pass that file as `--hir-reference` to `verify.py`, together with the earlier
token/declaration/name/type/module references. `verify_hir.py` reconstructs nested
semantic trees from the native TSV and compares them with Go HIR; it also rejects
cycles and unreachable retained nodes. `tests/hir.fy` exercises the owned API
directly. The Go oracle and Python comparator are test utilities; the candidate
runtime uses Fairy C and public SDK only.

`samples/hir-demo.fy` demonstrates a contextual i64 literal, ordered call
arguments, a conditional result and a classic `for` loop:

```sh
lexer-candidate --hir-file samples/hir-demo.fy
```

Windows development and release pass 1024 CLI executions across the existing
file modes and HIR. Each profile compares 34 positive HIR fixtures and 26
rejections; six direct native API tests pass per profile. The final comparator
checks declaration targets, lexical scope relationships, source spans and error
categories, and rejects 159 deliberate observation mutations per profile.
Check, scoped Go test/vet and Linux/Windows C emission pass; Linux native execution
is deferred to the owner's batch. Compiler/SDK inputs remain unchanged. Evidence:
`docs/ops/evidence/hir-file-cli-20261005.json`.

## Owned scalar MIR (`--mir-file`)

`lexer-candidate --mir-file <path>` reads the same bounded source file, then
parses, resolves names, checks scalar types, lowers the successful `Typed` value
to HIR and consumes that `hir_model::Program` through `mir_lower::lower`. The
returned `mir_model::Program` owns source bytes, canonical literal payload and
separate target, signature, function, local, block, operation and edge arenas; no
parser, type or HIR table is retained. Lowering admits sequential scalar code
(literals, resolved locals and parameters, unary and binary operations, ordered
calls, assignment, checked compound assignment, nested lexical blocks and
assertions) plus value and void conditionals: `if`/`else`, `if` without an `else`,
`else if` chains, arms where both branches return, and short-circuit `&&`/`||` whose
right operand is evaluated only when the left side requires it and may be any
admitted boolean expression, including side effects, calls or a return block. Those
CFG forms use an unconditional jump terminator and a two-way boolean-branch
terminator alongside the earlier checked switch. `while`, classic `for` and
nearest-loop `break`/`continue` now lower to this same CFG. Initializers run once;
an omitted `for` condition is true. `continue` reaches the post clause before the
next condition, while `break` skips post. Post blocks are allocated only when a
body fallthrough or continue reaches them, avoiding orphan blocks for unconditional
breaks or returns. Loop targets use a bounded 64-frame stack of HIR Statement IDs;
the existing recursion budget applies first. Bounded lexical `defer` captures
copy-scalar arguments immediately, once from left to right, and emits ordinary
void internal calls in LIFO order when the registering block exits by fallthrough,
return, break or continue. The return or block-tail value is evaluated before those
calls. Traps skip deferred calls; a trapping deferred call skips remaining calls.
Capture descriptors use a bounded compile-time scope stack, not a runtime queue.
Resource drops, Region resets and Result propagation are outside this scalar MIR
subset; their ordering with defer remains subsequent ownership work.

The private TSV format begins with `fairyc.scalar-mir/v1` and a `P` module-index
record. `T`, `S`, `F`, `L`, `B`, `O` and `A` records describe call
targets, signature types, functions, locals, basic blocks/terminators, operations
and ordered edges. Entity IDs are one-based and zero means absent; edge and
signature indexes are zero-based. Blocks form a mutable-local control-flow graph,
not SSA. A checked operation emits a typed `check` marker, a two-successor
checked-switch terminator and a non-returning trap block whose mask keeps the
type-specific failure categories: integer overflow, division by zero,
signed-division overflow, invalid shift and assertion failure. Nested calls publish
their argument edge range in the shared arena only after every argument completes:
evaluation collects the ordered argument locals in a temporary vector left to right
and appends the contiguous range after the last argument, so an argument that returns
leaves neither orphan argument slots nor a dangling parent call. The branch work adds
no record tag or field: the jump and boolean-branch terminators reuse the existing `B`
layout, and a literal `assert(false)` ends its block with an unreachable,
successor-free success block so no live success path remains. The record labels, trap
masks and this format are private implementation contracts.

Limits are 32768 source bytes, 512 functions/targets, 16384 locals, 16384 blocks,
32768 operations, 65536 edges/signatures and 688128 payload bytes, with recursion
depth 64, at most 256 simultaneously active defer descriptors and 4096 captured
argument references; the existing parser, resolver, type and HIR budgets apply
first. Complete output is buffered under a 2 MiB cap before stdout publication. Malformed, unsupported,
allocation, limit and HIR-lowering failures return 2 with empty stdout and a plain
message; scanner, name and type errors keep their existing located messages.

`mir_model::assemble` checks construction ranges and links. The separate Fairy
`mir_verify::check` then checks the admitted scalar invariants: `V1001` operation,
literal, call, return and table shapes; `V1002` definite initialization on every
reachable predecessor path; and `V1501` exact checked arithmetic, trap edges and
unreachable proofs. Both lowering and rendering require this verification to
succeed. Rendering rejects an invalid graph before appending any bytes, preserving
an existing caller buffer. CLI verification failures exit 2 with empty stdout and
the private `MIR verification V1001`, `V1002` or `V1501` error on stderr.

The verifier requires closed internal function bodies and signatures, unique
parameter/edge/operation ownership, correctly typed operands and destinations,
canonical range-checked integer literals, and no orphan executable table entries.
Checked arithmetic must be the first operation of its success block, match the
guard's frozen operands, and have no unchecked predecessor bypass. Trap blocks
contain no operations or cleanup. An unreachable success block requires a false
assertion on a non-parameter local with exactly one constant-false definition.

Initialization uses heap-backed bitsets and a worklist. Reachability is established
first; predecessor intersections descend from the greatest fixed point, with entry
always seeded only from its declared parameters. Uses are checked after convergence.
Call results become initialized only on the normal successor. Zero-iteration loops
and cyclic predecessor paths cannot manufacture initialized values. Analysis is
bounded to 67108864 charged steps across the program; exhaustion returns the existing
limit code 2, and allocation failure returns 3. Per-function bitset storage is at
most 33554432 bytes under the existing local/block caps, plus bounded graph indexes.

Move/drop discipline, borrow origins, Region placement and unsafe provenance
remain subsequent verifier components because their value forms are not admitted
by this scalar model. There is no candidate interpreter, backend or executable
compilation of the input in this mode.

From the repository root, regenerate the independent Go MIR oracle with:

```sh
go run ./cmd/fairyc-lexer-file-oracle --mir .cache/mir-file-cli/reference.json
```

Pass that file as `--mir-reference` to `verify.py`, together with the earlier
token/declaration/name/type/module/HIR references. `verify_mir.py` is a test-only
reader and scalar executor: it reconstructs the tables from the native TSV, checks
the block and edge graph, and executes admitted programs to compare the observed
return value, trap category and fixture-only deferred-call trace with the verified
Go MIR. The trace hook is enabled only for explicitly marked reference fixtures.
It is an independent
observation utility, never a candidate runtime dependency, and it does not natively
compile or execute the input. Every admitted fixture must now match the executed
Go MIR reference; a reference execution error fails the oracle rather than
substituting a fixed expected result. `tests/mir.fy` exercises the owned API directly.

`samples/mir-demo.fy` demonstrates a checked compound assignment across a forward
call, and `samples/mir-branches-demo.fy` demonstrates a conditional result and a
short-circuit assertion:

```sh
lexer-candidate --mir-file samples/mir-demo.fy
lexer-candidate --mir-file samples/mir-branches-demo.fy
lexer-candidate --mir-file samples/mir-loops-demo.fy
lexer-candidate --mir-file samples/mir-defer-demo.fy
```

The current reference set is 106 fixtures, all admitted: 87 returning programs
and 19 trapping programs covering constants, forward and nested
calls, lexical blocks, checked compound assignment, signed `i32`/`i64` overflow and
negation, division-by-zero, signed division/remainder overflow, invalid-shift and
overflowing-shift traps, and assertion traps, extended by value and void conditionals
with and without an `else`, `else if` chains, returning condition operands, an
argument that returns from the first or last call position, a returning `if` used as
an argument, both-returning arms, branch side effects taken and skipped, conditional
fallthrough, and short-circuit `&&`/`||` with side-effecting or trapping right
operands; the unsigned-limit case exercises `usize`, `u32` and `u8` maximum values as
accepted successes rather than traps. Loop cases additionally cover zero iterations,
`i64` counters, side-effecting/calling conditions, nested and mixed loops, loop-local
shadowing, omitted clauses, initializer assignment, return/break/continue in a while
condition, early body returns, body/post overflow and calls skipped by loop control.
Defer cases cover immediate `i64` capture, nested LIFO scopes, skipped branches,
left-to-right argument evaluation, return during registration, return-value-first
cleanup, zero arguments, local shadowing, short-circuit paths, loop fallthrough,
break/continue and traps during registration, body execution or deferred cleanup.
Twenty-one explicitly marked cases compare exact emitted trace bytes as well as
their return value or trap category.

All 106 admitted fixtures match the Go MIR interpreter, with no contract goldens.
The Go lowerer captures source-local scalar reads before subsequent binary operands
or call arguments can change them, and propagates termination through binary and
short-circuit operands, conditions and enclosing returns. Thus
`var i32 x=5;return x+if(true){x=2;3}else{4};` returns 8, and returning operands in
conditions or calls preserve their inner `return 9` without evaluating later calls.
Twelve dedicated Go verifier/interpreter regressions cover these paths, including
the already-correct conditional argument, its fallthrough alternative and the
subsequent while-condition return fix. The same twelve assertions execute in
generated native C under low and optimized MSVC.

The earlier return-order fix passed 1162 CLI executions (581 per profile),
including its 59 MIR fixtures (56 admitted: 42 returning, 14 trapping; 3 deferred).
After restricting snapshots to source locals rather than already-materialized
temporaries, the final Go reference is regenerated and the 59-case MIR comparison
reruns on each retained image. Scoped Go MIR, interpreter, C backend and fixture
suites and vet pass. At that snapshot, `TestStructNestedStorageMIR/array` failed
with both the unchanged baseline lowerer and the revised lowerer. Subsequent
contract review established that its `i32[2]` field is already admitted by ADR 0468;
the failure was an obsolete rejection expectation, not a verifier defect. The
follow-up test now rejects zero-length, oversize and nested arrays and admits
scalar array fields of length 1, 2 and 4096 through source analysis, MIR codec and
both Windows/Linux C emission. Historical evidence retains the observed failure
and its initial classification; this correction does not change language behavior.

That return-order unit intentionally changed Go MIR source. Of the pinned compiler's 619 input
records, only `internal/mir/lower.go` differs; the pinned compiler executable and SDK
remain unchanged. The candidate is still built with that prebuilt compiler, while
the oracle and native regression use current Go source. Example package version
0.11.0 and its Fairy product sources were unchanged in that unit. Evidence:
`docs/ops/evidence/mir-control-order-fix-20261006.json`. Earlier branch and sequential
evidence remains in `mir-branch-file-cli-20261006.json` and
`mir-sequential-file-cli-20261006.json`. Linux native execution is deferred to the
owner's batch, and self-hosting promotion remains blocked.

The preceding loop unit advanced this example to version 0.12.0. Its comparison passed
1212 CLI executions (606 per profile), with all 83 admitted MIR fixtures matching
the executed Go reference and one deferred case. The 12 direct API tests in
`tests/mir.fy` pass under both native profiles. They include structural checks that
while-break reaches exit, for-continue reaches the checked post before the header,
and unconditional break allocates no unreachable post block. Supplementary file
cases reject break/continue outside a loop, a non-bool for condition, an effectful
post clause and a deferred call inside a loop, with empty stdout and unchanged
input bytes. `samples/mir-loops-demo.fy` emits MIR whose test executor returns 2
after an `i64` loop with continue and a while with break. Both target C emission
profiles and the scoped Go workspace/vet checks pass; Linux native execution remains
deferred. The Go while-condition return defect found by this unit was fixed first
in its own commit, with interpreter and low/optimized MSVC regressions.

No TSV tag/field, SDK API, language revision, runtime symbol or stable CLI command
is added. The pinned compiler executable and SDK remain unchanged; current Go
oracle sources are intentionally distinct from the pinned compiler build inputs.
Evidence: `docs/ops/evidence/mir-loop-file-cli-20261006.json`.

The defer unit advances this example to version 0.13.0. Its full comparison passes
1256 CLI executions (628 per profile), including all 106 MIR fixtures. Fifteen
direct API tests in `tests/mir.fy` pass in both native profiles, with new checks
for scalar capture temporaries, nested LIFO calls and absence of deferred-call
argument edges on a trapped path. The sample's test executor returns 7. Both
Windows and Linux target C emission and the scoped Go workspace/vet checks pass;
Linux native execution remains deferred to the owner's batch. A Go lowering defect
found by the registration-return case was fixed first in a separate commit:
an argument that returns must stop further capture and parent-call registration.
Evidence: `docs/ops/evidence/mir-defer-file-cli-20261006.json`.

Version 0.14.0 adds the scalar verifier. Its comparison passes 1256 CLI executions
in both profiles together, retaining all 106 MIR fixtures with unchanged results
and trace bytes. One native test per profile runs a grouped graph inventory:
four valid graphs and 23 invalid graphs that all pass basic construction, plus two
work-budget rejection cases and one invalid-render buffer-preservation case.
The inventory covers call argument/result typing, missing bodies, duplicate target
identities, uninitialized operands and joins, loop entry versus backedges, forged
unreachable proofs, malformed literals, orphan records and checked-operation bypass.
This grouping avoids recompiling the full candidate for every individual mutation.
Windows and Linux C emission and scoped Go workspace/vet checks pass. Linux native
execution remains deferred to the owner's batch; ownership verification and
self-hosting promotion remain subsequent work. Evidence:
`docs/ops/evidence/mir-verifier-file-cli-20261006.json`.

## Recognition and limits

Current recognition: leading UTF-8 BOM, ASCII identifiers and exact current
keywords, longest-match punctuation, horizontal whitespace/LF/CRLF, line and
documentation comments, nested block comments, arbitrary-width 2/10/16-base
integers, decimal fractions and exponents. `src/numbers.fy` follows the reference
numeric recovery envelope and emits `E0004` for malformed numeric tokens.
`src/strings.fy` recognizes ordinary, byte and C strings, simple/hex/Unicode
escapes, exact hexadecimal cooked bytes and C terminators. Invalid escape,
UTF-8, NUL, newline and EOF recovery preserve the reference diagnostic spans.
Invalid code bytes, non-ASCII code points, lone CR and unterminated block comments
retain lossless tokens with `E0001`. `src/utf8.fy` rejects overlong encodings,
surrogates, out-of-range scalars and truncated sequences without skipping bytes.
Token observations are
`kind trivia startByte endByte cookedLength` numeric fields followed by each cooked
byte as a decimal number, separated by spaces. `true`, `false` and `null` retain
their cooked spelling; numbers retain their canonical decimal bytes. EOF appears exactly once.
These numbers are private test observations tied to `internal/lexer/token.go`.
After tokens, a diagnostic count precedes each diagnostic's length-prefixed code,
repair and token bytes followed by its half-open start/end span. The current
diagnostic subset is `E0001`/`E0004`/`E0005`/`E0006`/`E0010`, including comment and string
`exit` repairs and expected closing tokens. The separate diagnostic store uses
stable source/code/repair/token ordering and preserves duplicate records.

Input is bounded to 32768 bytes, each materialized decimal expansion to 16384 bytes
and cumulative retained cooked data to 65536 bytes. Status 0 means a
complete supported token/diagnostic stream (not syntactically valid code);
2 means unsupported input, 3 allocation failure, and 4 candidate-limit exhaustion.
Scanning finishes before output starts, so scan failure produces no partial stream.
The default diagnostic budget admits 256 ordinary diagnostics. The next attempted
diagnostic stops recognition before allocating that diagnostic and adds one
uncharged `E0010`; the complete lossless exhaustion stream is printed normally.
Output failures still use the existing assertion/trap boundary.

`scanner::report_limits` accepts caller-selected source-byte and token limits
within the candidate's independent storage guards. Exhaustion emits `E0010` once,
retains the denied token and remaining bytes as invalid records, and always emits
an uncharged EOF. Byte limits are checked before recognition; token limits count
trivia and the leading BOM. Native tests compare zero, exact and exceeded limits
with the Go reference, including malformed-token diagnostic ordering.
`scanner::report_budget` also accepts a diagnostic limit. The same counter follows
code, numeric and string recognition, including escapes, invalid UTF-8, raw NUL,
newline and final literal validation. Tests compare both the failure position and
the retained diagnostics, including zero allowance and the default 256/257 boundary.
Independent storage guards still apply if callers request more than 256 ordinary
diagnostics. `src/budget.fy` supplies the work meter as an immutable value returned
by each charge. Native tests compare charge sequences against the Go budget and
check exact-limit acceptance, sticky exhaustion and arithmetic at the maximum `usize` value.
`scanner::report_work` connects the meter to BOM, whitespace, comments,
identifiers, punctuation and non-ASCII code bytes. Native comparisons vary work
limits at every source-byte boundary together with token limits, including the
reference's one-byte progress guard when a multi-byte step cannot start.
Numeric recognition also charges source advancement and complete pre-trimming
decimal expansion before materializing output. Exact expansion boundaries,
cumulative numbers, signed 64-bit exponent endpoints and out-of-range exponents
are compared against Go. Independent expansion/storage caps remain in force.
String recognition carries the same work meter through prefixed openers, raw
UTF-8, escapes, malformed-escape recovery and closing quotes. A multi-byte charge
either advances in full or leaves the cursor unchanged, matching the reference.
Native work-boundary tests also combine token and diagnostic limits and mixed
numeric/string input. All recognition paths now participate in work charging;
independent candidate storage caps and the separate compiler-component promotion
gate remain in force.
Status 2 remains reserved for unsupported extensions;
it must not be interpreted as a successful empty token stream.
The Go lexer stays authoritative. The start evidence and separate promotion gate
are in `docs/architecture/self-hosting-readiness-v0.md`.

`TestWindowsFairyLexerCandidate` compares native low/optimized observations against
the Go reference and separately tests unsupported and limit outcomes. The harness
supplies only raw bytes, not precomputed candidate tokens or recognition tables.

## Complete checked-in source corpus

`TestWindowsFairyLexerCorpus` reads every `.fy` file below `spec/fixtures`, without
filtering by pass/fail status. It compares current and immediately previous lexer
revision modes and deterministic NUL insertion, first-byte deletion and invalid-byte
replacement. Every source/mode/mutation case runs with five budget settings:
defaults, half the source-byte allowance, one token, no diagnostics, and half the
source length in work units. The current corpus has 303 fixture source files plus
the real `std_bytes_quickstart` and `std_io_scalar_quickstart` main sources, both
larger than the former 4096-byte limit. Identifier lengths around 4096 and 16384,
a 6000-byte string, and closed/unterminated 12000-byte comments add large-input
boundaries. All 7670 framed observations match Go under low and optimized Windows MSVC.
Comparison retains exact `E0010` positions, invalid ranges, preceding diagnostics
and EOF for exhausted runs. Newly added oversized fixtures
fail the lane rather than silently disappearing. Embedded Go-only seeds and
arbitrary inputs outside the stated candidate limits are not a full conformance claim.

The maintained `src/main.fy` driver reads private binary stdin frames: one mode
byte and a two-byte little-endian source length. Modes 0/1 use default budgets;
modes 2/3 add four unsigned 32-bit little-endian limits in byte, token, diagnostic,
work order. Odd modes use the previous revision. Exact source bytes follow the
header. EOF is valid only between frames. Invalid mode, truncated framing or more
than 2048 frames exits 8; oversized source exits 4. The harness checks invalid
mode, every truncated extended-header length, missing body and a 32769-byte length
at both profiles.

The harness compiles once per profile through the ordinary workspace compiler,
then reuses that artifact for output-bounded batches of at most 2048 frames. Each batch stays below 60KB
of expected output and retains the existing native driver's 64KB output limit and
trap classification. Fixture coverage remains complete; the two project sources
are a bounded workload sample, not all project files. A single observation that
exceeds the output cap fails explicitly, so 32KB source admission does not promise
that every such source fits this private reporting transport. This is exact record
comparison, not a token hash.

## Own-source comparison without increasing process output limits

`TestWindowsFairyLexerStreamSelfCorpus` reads every `.fy` file in this candidate's
`src` directory, including the scanner and observation driver themselves. Input
admission is 32KB so the scanner with its consumer API fits without slicing source
files into lexically unrelated fragments. Boundary witnesses include 32767/32768
byte identifiers, a long cooked string and an unterminated long comment.

`pages::run` is a separate private one-request driver. Its input is a revision byte
(0 current, 1 previous), source length and observation start as u32 little-endian
words, then exact source bytes and EOF. It rejects truncated headers/bodies,
trailing bytes, source lengths above 32768 and observation starts above 1048576.
Each request collects a full Stream, then prints at most 4096 observation integers.
The harness rejoins every page and compares all token metadata and cooked bytes
with Go, under both revisions and native profiles. It keeps the existing 64KB
process-output guard and compares exact values rather than hashes. Diagnostic
records in this API transport use recognition order.

Paging repeats collection per request; it is bounded comparison transport, not a
streaming lexer or a throughput optimization. The source remains owned in full.
The initial run compared eight source files across 148 pages; the expanded local
corpus now compares nine source files and four boundary witnesses across 196 pages.
This is self-lexing evidence, not full parser conformance or bootstrap promotion.
The original Stream API/page transport and first parser passed hosted Linux and
Windows verification at `ecf428a3` in [Frontend run 34441267986](https://github.com/0disoft/fairyc/actions/runs/34441267986).
That historical receipt does not cover later changes; delivery requires successful
CI for the exact pushed commit.

## Bounded syntax parser

`syntax::parse` consumes a Stream and an item limit no greater than 512. It returns
an owned `Parsed` containing declarations, expressions, binding entries,
statement trees, written types, struct/resource fields and item attributes, retaining the entire original Stream. This
candidate recognizes `module path;`, `import path as alias;`, and functions with
optional `pub` visibility (retained in `public`, `public_start`, `public_end`;
private declarations use false and zero spans). The modifier is not discarded
from unsupported declaration forms. Functions preserve written parameter and
result types using ordinary Fairy spelling (`fn i32 f(i32 x) { return x; }`).
Parameter lists allow a trailing comma. Bodies admit empty blocks,
`let`/`var` bindings with optional type annotations and required initializers,
and `return` with an optional expression. Statement-only nested blocks, expression
statements, ordinary and compound assignments, and `if (...) { ... } else if (...)
{ ... } else { ... }`, `while (...) { ... }`, `break;` and `continue;` are admitted.
Classic `for` loops and other statements
remain outside this subset.
It does not resolve names or imports, validate types, module-to-file mapping or
return obligations, admit the complete Stage 0 grammar, or produce executable MIR.

Ordinary functions, structs and resources accept ordered attributes before optional `pub`.
`attributes.fy` retains qualified names, source spans and ordered path, integer
or string arguments (including empty argument lists and trailing commas).
Declaration `attribute_start`/`attribute_count` selects the global attribute range.
`attribute_at` exposes each attribute's name span and argument range;
`attribute_argument_at` and `attribute_argument_text` expose argument kind
(1 path, 2 integer, 3 string), span, token and original spelling.
The parse shares limits of 256 attributes and 1024 arguments, checked before
allocation; exhaustion returns 4 without a tree. Failed items roll both tables
back, and recovery does not attach a rejected prefix to a later declaration.
Attribute semantics, effect validation and permissions remain outside this parser.

The private `types` module recognizes named/qualified types, `ref`, `ref mut`,
grouping, unsized slices and fixed arrays. It preserves `ref u8[]` versus `(ref u8)[]` without
resolving names. `Declaration.result_type` and `Entry.type_root` are one-based
roots; an inferred binding uses zero. `type_at`, `type_count` and `type_text`
expose postorder nodes and exact source slices. Node kinds are 1 path, 2 reference,
3 mutable reference, 4 slice and 5 group; wrappers point to `left`, `right` is zero.
All declarations share 4096 type nodes and a 32-active-call type-depth limit.
Failed declarations roll back type nodes with the other tables.
Raw pointer types use kind 11 (`ptr`) and 12 (`ptr mut`), with their child in
`left` and zero in `right`. Their prefix binding distinguishes `ptr u8[16]`
from `(ptr u8)[16]`. Foreign function types remain unsupported.

Named types also accept nonempty generic argument lists, trailing commas and
nesting. Kind 6 retains the base path in `left` and a reverse argument-link tail
in `right`; kind 7 links an argument root in `left` to its predecessor in `right`.
`type_argument_count` and `type_argument` expose source order. Generic nodes and
links share the same type-node budget, and argument nesting shares type depth.
The type cursor consumes each half of `>>` at its actual byte boundary without
rewriting source tokens or changing expression shifts. Stray closing halves fail.
Type applications do not perform specialization or type resolution.

Ordinary `struct Name { ... }` declarations use declaration kind 4, with optional
`pub` on the struct and individual fields. `field_start`/`field_count` select a
zero-based source-ordered range in the separate `field_at`/`field_count` table.
Each field preserves its full/name/public spans, visibility and `type_root`.
Structs have no body or result-type root. All structs share a 1024-field limit;
failed declarations roll back fields and types together. Duplicate-field legality,
layout and ownership are not checked. Resources use declaration kind 6
and share the same field/type arenas and 1024-field budget. Optional `cleanup`
bodies reuse the statement parser and are retained by `cleanup_body` in the shared
statement arena. `cleanup_start`/`cleanup_end` span the keyword through closing
brace; all three zero means unwritten. Empty cleanup has a real empty block.
Cleanup entries use the declaration's source-ordered entry range. Resources still
have no function `body` or result type. Body nodes, entries and nesting share the
same budgets as functions, and malformed cleanup rolls back the whole declaration.
Declaration attributes and `pub` spans are preserved as for structs; imports
cannot follow resources. Attributed cleanup remains unsupported. Generic
declarations and record expressions preserve syntax only. No resource ownership or
destruction behavior is implemented by the candidate. Malformed resources roll
back fields, types and attributes before recovering to the next declaration.

Expressions admit integers, `true`/`false`/`null`, strings, byte strings, qualified identifier paths, calls,
nested parentheses, prefix
`+ - ! ~ *`, and the grammar's binary operators from multiplicative through
logical-or. C strings and other literals
remain outside this subset. Precedence follows `spec/grammar/stage0.ebnf`;
binary operators associate left, while unparenthesized comparison chains fail.
The private postorder node kinds are 1 literal, 2 path, 3 group, 4 prefix,
5 binary, 6 call, 7 argument link, 8 field, 9 index, 10 record and 11 record-field link. Field/index receivers are
in `left`; index expressions are in `right`. Field text is the exact member token.
Calls, fields, single indices and range slices form repeatable postfix chains.
Kind 26 retains a slice receiver in `left` and a kind-27 bounds node in `right`.
Bounds retain optional start/end expression IDs in `left`/`right`; zero means a
written omission, not a synthesized numeric value. The bounds token is `..`;
its span excludes brackets, while the slice spans the complete receiver and suffix.
All four forms `a[x..y]`, `a[..y]`, `a[x..]`, `a[..]` share existing node/depth
budgets. No bounds evaluation, range ordering or borrow validation occurs.
One-based child IDs (zero absent), half-open source-byte spans and
source token spelling preserve structure without evaluating or resolving names.
Declaration `expression` is the last top-level return root, or zero if absent/empty.
`expression_count`, `expression_at` and `expression_text` expose read-only views.
The independent `bodies.fy` module owns recursive expression, statement and block parsing;
`parse_tokens.fy` owns exact token spelling, trivia skipping and operator
classification. Existing body helpers delegate to it; cursor mutation and mutual
recursion remain in `bodies.fy`, without an import back from the token module.
`syntax.fy` retains declaration rollback and the existing query facade.
`literal_kind` exposes the source token kind. `literal_cooked` borrows the
scanner's canonical cooked representation: string/byte-string values are lowercase
hexadecimal bytes, not a newly decoded buffer. Raw spelling and byte spans remain
unchanged. Keyword literals `true`, `false` and `null` use expression kind 1,
token kind 4, and preserve their spelling in both raw and cooked views. `null`
support is syntax-only; it does not add pointer evaluation or type checking.
Lexical diagnostics still reject the entire parse with status 5.
Path text is its exact source slice, including intervening trivia. Calls retain
the callee in `left` and a reverse-linked argument tail in `right`; each argument
link points to its expression in `left` and its predecessor in `right`.
`call_argument_count` and `call_argument` expose the original argument order.
Empty calls, trailing argument commas and repeated call suffixes are admitted.
All call/argument nodes share the existing node limit, and argument recursion
shares the expression-depth limit. Generic specialization execution remains unsupported.
Records retain a path root (ordinary), kind-20 type application (generic), or
kind-28 variant name (contextual) in
`left`, and the reversed field-link tail in `right`. Field links retain the value in `left`, previous link in `right`,
member token and full initializer span. `record_field_count` and `record_field`
expose field-link IDs in source order; `expression_text` returns each field name.
Empty records, nesting and trailing commas are admitted. Duplicate fields,
field types and layout are not checked. Record and field-link nodes consume the
same expression budget; no separate record arena or unbounded side table exists.
Failed declarations roll back their nodes before recovery. A shared 4160-expression-node
ceiling and 64-active-expression-call limit return status 4 without a result.

Value-block expressions have kind 12: `left` selects a statement-block ID and
`right` is absent. The block's `expression` field is its optional tail root,
separate from its ordered statement links. A semicolon keeps an expression in
the statement list; an unwritten tail remains zero. Function bodies still require
statement terminators. Nested value blocks share the 32-block-depth and
64-active-expression bounds instead of resetting them at the expression/body edge.
Entry IDs are reserved before parsing an initializer, so nested entries cannot
overtake their source owner or bypass the 1024-entry budget. Completed records
may be stored in postorder; `entry_at` resolves the source-order ID with a bounded
scan. Statement entry references use those same IDs.

Conditional expressions use kind 13 (`left` condition, `right` branch record).
Kind 14 retains the then statement-block ID in `left` and the optional else
expression ID in `right` (kind 12 block or kind 13 nested conditional).
Conditions, branch order, written tails and full byte spans are retained.
An unwritten else stays zero; branch type agreement and value availability are
semantic questions. Else-if recursion remains bounded by the expression limit.

Switch expressions use kind 15 (`left` subject, `right` reversed case-link tail).
Kind 16 is a shallow leading-dot variant pattern with an optional kind-17 binding
in `left`; kind 17 retains an identifier or `_` spelling. Kind 18 is a case
(`left` pattern, `right` statement-block ID), and kind 19 links cases in reverse.
`switch_case_count` and `switch_case` expose case IDs in written order.
Variant names and binding names retain their own tokens; pattern, case and switch
spans remain distinct. Empty and duplicate cases are syntax-only observations:
exhaustiveness, reachability and payload/type validity are not checked.
Named record patterns such as `.Pair { .left = first, .right = second }` use
kind 32, distinct from kind 16 even for explicit empty braces. Their `left` is
zero and `right` is a reverse-linked kind-33 field tail. Each field retains its
name token and full span, a kind-17 binding in `left`, and its predecessor in
`right`. `pattern_field_count` and `pattern_field` expose field links in source
order (sharing the record-field traversal). Name tokens retain exact spelling.
Trailing commas, duplicate field names and `_` bindings are syntax-only; no
binding resolution or exhaustiveness check is performed. Whole-value wildcards
and multiple positional bindings remain unsupported. All pattern/case/link nodes
use the same 4160-expression-node budget.
Case bodies share block and expression depth limits with their enclosing tree.

Contextual variant names such as `.ok`, `.err` and `.none` use kind 28, with no
children. The full span includes the leading dot and intervening trivia; the
token and `expression_text` retain the identifier. Existing call nodes wrap this
callee for `.ok(value)`, preserving ordered arguments and shared budgets. Type
inference and variant validity are not checked. Contextual record construction
`.Variant { .field = value, ... }` reuses kind 10 with a kind-28 left child and
the existing ordered field-link tail. Its full span includes the leading dot;
the kind-28 child's token/text identifies the variant name. Empty initializers,
nesting, trailing commas and subsequent postfix access use existing record rules.
No type inference, variant resolution, layout or execution is introduced.

Explicit type applications are recognized only after a path and when the closing angle
list is followed by a call or record initializer, using read-only lookahead capped at 256 significant
tokens. Trivia does not consume that token allowance. Kind 20 retains the path in
`left` and a type-argument link tail in `right`; the ordinary kind-6 call owns
this specialization node as its callee, or a kind-10 record owns it as its constructor.
`specialization_argument_count` and
`specialization_argument` expose written type roots in source order. Type
arguments share the declaration type arena and its limits. Nested `>>` is split
only by the existing type cursor; ordinary comparisons and shifts are unchanged.

Reference expressions use kind 21 for `ref` and 22 for `ref mut`, with the
operand root in `left` and absent `right`. Full spans retain intervening trivia
and the mutable marker; the source token retains `ref`. Postfix operations bind
inside the reference, while binary operators remain outside. These nodes consume
the existing expression-node and active-depth budget. Borrow validity, operand
mutability and lifetime rules are not checked by this syntax candidate.

Kind 23 is a written `as` cast: `left` is the operand and `right` is the target
type root. Casts associate left, below prefixes and above multiplication; their
operator tokens and full spans remain available. Targets use the shared type
parser and budget. No conversion, range check or target-type validation occurs.

Array lists `[]`, `[value]` and `[a, b,]` use kind 24 with absent `left` and a
kind-25 element-link tail in `right`. Each link retains its value in `left` and
the previous link in `right`. `array_element_count` and `array_element` expose
source-ordered values without flattening nested arrays. Array spans include both
brackets; element spans exclude separators. Both kinds share the expression-node
and depth budgets. Repetition `[value; count]` instead uses kind 29, retaining the
single value expression in `left` and a written integer/path count in `right`.
No element links, expansion, count evaluation or value duplication are performed.
Qualified count paths are preserved; arithmetic, calls, casts and parenthesized
count expressions remain outside this narrow subset. Element type compatibility,
constant validity and array evaluation are not checked by the candidate.

`entry_at` and `entry_count` expose source-ordered parameters and statements,
bounded to 1024 entries across the entire parse. Declaration `entry_start` and
`entry_count` select a zero-based range. Entry kinds are 1 parameter, 2 let,
3 var and 4 return; each preserves complete, name and optional type byte spans
plus its expression root. Zero name/type spans mean unwritten, not synthesized.
Errors roll back entries and expression nodes together. Duplicate names, type
compatibility, unreachable returns and binding resolution are semantic concerns
and are not checked here.

`Declaration.body` is a one-based root in a separate postorder statement tree;
zero is reserved for module/import declarations. `statement_at` and
`statement_count` expose nodes: 1 block, 2 binding, 3 return, 4 assignment,
6 expression statement, 7 list link, 8 while, 9 break, 10 continue and 11 assert.
Assert retains its condition in `expression` and its full keyword-through-semicolon
span. It requires parentheses and a semicolon, including in value blocks; it is
not a call or tail expression. Condition type and runtime assertion behavior are
outside this syntax layer. Its nodes share existing expression/statement budgets.
Kind 12 is a `discard` statement with a required operand in `expression` and a
full keyword-through-semicolon span. It is neither a return entry nor a value
block tail. No must-use, ownership transfer or destruction behavior is inferred.
While preserves its condition in `expression` and its body in `first`; break and
continue have no children. Loop-context legality remains a semantic concern.
Binding/return `entry` is a one-based
reference into the existing entry table. Assignment `target` and `expression`
refer to distinct expression roots; `assignment_operator` returns its raw spelling.
A block's `first` is a reverse-linked
statement-list tail and `second` its direct child count; list links use `first`
for the child and `second` for the previous link. `block_statement_count` and
`block_statement` return direct children in source order, without flattening nested
scopes. Block/if statements use a kind-6 expression-statement wrapper, including
an optional written semicolon. Their expression roots use the same block/if
representation as value positions, preserving branch tails and any following
postfix or binary expression. The former standalone if statement kind 5 is no
longer emitted. Assignment-place validity remains a semantic check.

The statement tree has a shared 4096-node limit (including list links) and a
32-frame block/conditional depth limit. Failures roll back statements, expressions
and entries together. Assignment-target legality, shadowing, condition types and
branch return obligations remain semantic checks, not parser acceptance criteria.

Declarations preserve source order, complete spans and name/alias spans. Problems
retain exact source-byte spans and private category codes (1 module, 2 import,
3 function, 4 struct or unsupported source item, 6 resource). A failed declaration never becomes an
ordinary declaration node. Recovery rescans its bounded token range with brace
depth before resuming at a following import/function; nested functions cannot
escape a malformed enclosing body. Original trivia and rejected source remain
available through the retained Stream rather than being reconstructed from nodes.

Problems additionally retain `expected` and the actual source `token` index.
`problem_actual` borrows its exact raw spelling (empty at EOF); `expected_text`
renders the private expectation code: 0 supported syntax, 1 identifier, 2 type,
3 expression, 4 `(`, 5 `)`, 6 `{`, 7 `}`, 8 `;`, 9 `=`, 10 `as`, 11 `]`,
12 `module`, 13 `>`, 14 `.`, 15 attribute argument, 16 array length,
17 `:`, 18 `case`. Code 0 deliberately remains general for unsupported constructs;
these are diagnostic hints, not inserted tokens, repairs or semantic acceptance.
Expectations reset before each source item so recovery cannot leak an old hint.

Allocation failure returns 3; item or 32-problem exhaustion returns 4; lexical
diagnostics return 5. These failures consume the input and publish no Parsed value.
Callers that need lexical diagnostics must inspect the Stream before passing it
to the parser. A successful Result can contain syntax problems: clients must check
`problem_count`, and must not treat this partial syntax tree as a compiler
acceptance decision. `declaration_at`, `problem_at` and `source_view` are read-only.
The scanner's source/work bounds plus monotone parsing and one bounded recovery
rescan limit work; expression recursion is bounded separately as above.

Native tests compare function/import spans, expression/call structure, ordered
parameters and bindings, nested blocks, conditional branches and assignment
operands with normalized Go ASTs. They also exercise missing headers/delimiters,
declaration recovery, import order, nested malformed bodies and byte, depth, node,
entry, diagnostic and work boundaries. Large repetitive inputs are constructed
at runtime without reducing their tested sizes. Focused Mustflow lanes separate
parser checks, Stream/self-lexing, corpus/budget checks and heap accounting;
hosted Go tests still discover all tests normally.

## Own-source syntax coverage

Fixed-array type nodes preserve an element root and an unevaluated integer or
qualified-path length root. Private kinds 8/9/10 represent array/integer length/path
length; all consume the existing 4096-node type budget. Empty brackets remain
slices. Length arithmetic, evaluation and size validity are not parser decisions.

All sixteen current source files, including the filename adapter, integrity checker and token helpers, parse without
syntax problems in both Windows profiles. The [package syntax report](parser-package-items-2026-09-12.md)
records the preceding fixture milestone; the [SDK syntax report](parser-sdk-syntax-2026-09-12.md)
records the current pointer/generic and SDK boundary. The [literal/item report](parser-literal-items-2026-09-12.md)
and [control syntax report](parser-control-items-2026-09-12.md)
retains the preceding milestone. The [preceding evidence report](parser-evidence-2026-09-11.md)
records the initial declaration comparison, integrity checks and fixture baseline.
The [earlier coverage report](parser-coverage-closure-2026-09-11.md)
records the preceding thirteen-file milestone.
The corpus test explicitly requires complete syntax for these
named files and fails on diagnostics, budget exhaustion or file removal. New
files remain observations until independently admitted to this regression gate.
For admitted files, the native declaration list is also compared with the Go CST:
kind, source order, complete byte span and declared name/alias span must match.
This is a declaration-level comparison, not full expression-tree equivalence.
The private `integrity::valid` checker additionally validates expression/type/
statement kinds, required and optional arena references, backward same-arena
links, token/source bounds, block link counts and declaration ranges. It runs on
native corpus results, including incomplete results. It does not prove semantic
validity, cross-arena graph acyclicity, unique ownership or full source coverage.
Synthetic-node probes exercise missing, out-of-range, self-referential and
wrong-kind links without mutating the published parse result.
Private candidate helpers `valid_field(parsed, field, index)` and
`valid_declaration_fields(parsed, declaration)` check detached copies through the
same paths used by `valid`. Payload probes cover field/type/key categories,
visibility metadata, member/brace/name bounds, backward payload ranges and orphaned
or wrongly owned fields. Field-local checks and declaration-range checks are
complementary; neither alone proves whole-tree validity. Payload fields must have
zero visibility spans. No published parse-result mutation API is added.
The fixture observation lane reads the checked-in parser manifest unchanged,
with at most 32 cases and 32768 source bytes per case. It separates supported
syntax, unsupported valid syntax and matching rejection using the filename-aware
entry point. Unexpected false acceptance or budget exhaustion fails the lane rather
than becoming a supported result. Both profiles must return identical observations.
All 22 current cases are required: 12 successful inputs and 10 rejected inputs.
An independent ID/status inventory rejects removal, outcome changes and duplicate
IDs before native execution. Per-case removal/outcome-mutation controls run before
the Windows platform guard; required native completion is also checked per profile.
The fixture manifest and its expected diagnostics remain unchanged.
The [statement/resource report](parser-coverage-statements-2026-09-11.md) records
the earlier eleven-file result; older baselines are retained as historical evidence.
Full-file exhaustion and separate first-blocker prefix probes are distinguished.
This must not be confused with the lexer's complete source-token comparison.

Postfix propagation uses private expression kind 30 with one operand and the
written question-mark token. Chains, grouping and prefix/infix precedence are
compared with Go AST shapes. This preserves syntax only, not error propagation
execution semantics.

Private declaration kind 7 preserves a written `const` declaration: `result_type`
is its declared type root and `expression` is its initializer. Visibility,
attributes and declaration/name spans retain the existing declaration contract.
Initializers are parsed, not evaluated.

Private statement kind 13 preserves `defer expression;`, including its operand,
source span and position in the enclosing block. Execution timing, call validity
and cleanup order remain semantic/lowering responsibilities.

Private statement kind 14 preserves a classic `for`: optional condition in
`expression`, body in `first` and a kind-15 header in `second`. Header children
are optional initialization (binding/assignment) and post assignment. Zero means
unwritten, not a synthetic true expression. The initialization statement span
includes its terminating semicolon; the header span includes both parentheses.
The body is not desugared into a while loop. Arbitrary expression-only header
clauses remain outside this grammar-bounded slice, even where Go syntax recovery
accepts them for later semantic rejection.

Private statement kind 18 retains `for (let u8 byte in view) { ... }` with the
item name in `target`, iterable expression in `expression`, body in `first`, and
item type in `second`. The parser keeps source spans and checks node references;
the compiler's semantic pass enforces the exact immutable byte-view type. The
expression arena admits at most 4160 nodes so the parser can parse its own
updated `bodies.fy`; the type and statement arenas remain bounded at 4096.

Floating tokens (kind 6) are preserved as literal expressions with their original
token, byte span and lexer-normalized spelling. The parser performs no numeric
conversion or floating-point evaluation.

C-string tokens (kind 9) are also literal expressions. Their cooked view retains
the lexer-produced terminal NUL in its hexadecimal representation; embedded NUL
and invalid escapes remain lexical errors. No pointer or FFI behavior is added.

Private declaration kind 8 preserves module-level `static_assert(expression);`.
Its name spans and result type are zero because no name or type was written;
`expression` is the required condition root. Attributes and public syntax retain
the declaration contract. No constant evaluation occurs, and `static_assert`
remains an ordinary identifier outside the module-item dispatch position.

## Package syntax and file identity

Private declaration kind 9 retains `unsafe extern C fn Type name(...);` as a
foreign declaration, with a result type, ordered parameters and no body or value.
Kind 31 is a distinct unsafe-block expression: its left child is a statement block
and its right child is zero. It retains the `unsafe` keyword in the full byte span,
uses the existing block/depth budget, and participates in postfix/infix syntax.
These nodes do not grant unsafe execution or perform foreign calls.

Private declaration kind 10 is a nameless `state` item. Its ordered fields use the
existing shared 1024-field arena with explicit tags: 0 struct/resource, 1 cell,
2 map, 3 enum member, 4 enum payload field. `type_root` is the field value type for tags
0..2 and 4; enum members have zero type roots. Only maps have a nonzero `key_type`.
Names, field spans and attribute/visibility syntax are retained. Storage validity,
package ownership, duplicate names and persistence remain semantic concerns.

Private declaration kind 11 preserves `enum Name { Member; Some { T value; }; ... }`, including
empty enums, optional declaration attributes/`pub`, duplicate member spelling,
source order and exact full/name byte spans. Members use tag 3 in the existing
`field_start`/`field_count` range, with zero type/key roots and no member visibility.
Payload fields use tag 4 and appear in source order immediately before their owner
member (postorder) in that range. The member's `payload_start`/`payload_count`
select those fields. `payload_open`/`payload_end` preserve the half-open brace
span, distinguishing an explicit empty payload from no payload (all four values
zero). Non-member records have all four payload values zero. The validator checks
payload ownership, ordering, containment and field categories.
They share the 1024-entry field arena with structs, resources and state fields;
failed declarations roll back the whole member range and attribute prefix.
Recovery recognizes enum boundaries without promoting nested declarations.
Generic enum headers and explicit discriminant assignments remain
unsupported. No enum value layout, type checking or execution is introduced.
`fairyc_go_test_parser_enums` verifies this boundary and related field consumers.

`file_syntax::parse(stream, max_items, relative_path)` is a separate private entry
point; `syntax::parse` stays byte-only. It accepts a caller-supplied source-root
relative filename, normalizes slash/backslash and lexical dot/parent segments,
then checks ASCII identifiers and the `.fy` extension like the Go source model.
It reads no files and never normalizes source bytes or ignores source comments.
Path length is bounded at 32768 bytes and at most 1024 live path segments. Errors
3/4/5 retain allocation/budget/lexical meaning; 6 means invalid identity and 7
means module mismatch. Invalid paths fail first. For valid paths, syntax problems
remain owned incomplete results and take precedence over a module mismatch.
These private errors are not stable Go diagnostic IDs or diagnostic-span parity.

The configured `fairyc_go_test_parser_package_items` lane verifies these additions
in both native profiles. The filename-aware fixture driver uses a bounded path
line followed by unchanged source bytes; it has no module-mismatch exception.

## Generic declarations and SDK observations

Functions, structs and resources preserve optional nonempty generic parameter
lists, including source order, trailing commas and duplicate names. Duplicate-name
validity and generic aggregate admission remain semantic concerns. This does not
add parameter constraints, defaults, specialization or name resolution.

`Declaration.generic_parameters` is a type-arena link root; zero means no written
list. `generic_start`/`generic_end` retain its complete angle-bracket span.
`generic_parameter_count` and `generic_parameter_at` expose ordered name nodes.
Private type-arena kind 13 is a parameter-name leaf and 14 links that leaf in
`left` to its preceding link in `right`. Neither kind is a value type. They share
the existing 4096-node budget and declaration rollback with ordinary type syntax;
two nodes are charged per parameter before successful declaration publication.

The SDK corpus reads unchanged sources as bounded input data, not as candidate
modules. `std::vec`, `std::bytes`, `std::text`, `std::alloc`, `std::process`, `std::io` and `std::fs`
are required-complete cases:
both native profiles must return status 0, zero problems and the exact whole Go
declaration list (kind, full span and name span). Exhaustion or partial parsing
fails immediately instead of falling back to a prefix observation. An independent
required inventory rejects omissions, duplicate disk/name identities and renamed
required mappings; each profile must finish all required files. Missing source
files fail at read time.

All seven current cases are required under the same source/node limits.
Logs identify each file's required status.
IO source is parsed as data;
its console functions and effects are not executed by the corpus probe. Likewise,
FS input is only parsed; filesystem SDK operations are not executed. The Go oracle
retains enum declarations as comparison kind 11; payloadless and payload-bearing
member syntax are supported by the candidate.
Future observation-only files may retain partial results with ordered Go-CST
subsequence checks and at most 128 original complete top-level prefix probes on
exhaustion. They do not weaken the required cases. `fairyc_go_test_parser_sdk_gate`
runs the inventory/outcome rejection controls and native corpus without unrelated
syntax suites; `fairyc_go_test_parser_sdk` also runs the corpus. Complete syntax
is not proof of SDK execution or semantic conformance. See the
[SDK syntax report](parser-sdk-syntax-2026-09-12.md).

## Verified scalar C emission (`--c-file`)

Version 0.15.0 adds a Fairy-written C backend over the verified scalar MIR:

```sh
lexer-candidate --c-file samples/mir-defer-demo.fy
```

It accepts one module with exactly one `fn i32 main()` without parameters. The
existing scalar types, checked arithmetic, calls, branches, loops and lexical
defer are emitted as deterministic C11. The transport requires a 64-bit,
8-bit-byte, two's-complement target. Private numeric `fy_` symbols avoid source
identifier interpolation. Link the unit with the private scalar runtime providing
`fy_runtime_trap`; it is not a standalone public C ABI. Defining
`FY_CANDIDATE_NO_MAIN` omits the generated main wrapper.

`src/c_flow.fy` conservatively proves nonreturning closed callees and retains only
reachable blocks. A block-local final bool constant or literal zero divisor may select one successor;
unknown conditions and calls keep both possible paths. The bounded monotone pass
does not assume recursive calls return or trap. Emission omits normal continuations
of proven nonreturning calls and emits only required arithmetic helpers, keeping
strict C warning checks enabled. Arithmetic uses unsigned bit transport for signed
operations and explicit overflow, divisor and shift guards.

The same 32768-byte source bound applies. C output is buffered under a 2 MiB cap
before stdout publication. Invalid MIR/entry rejection preserves the caller's
existing Buffer. A later writer allocation/size failure may leave an unpublished
prefix in that internal Buffer; CLI failures still emit no C bytes. The writer
also rejects caller buffers already above its cap without arithmetic underflow.

Portable verification uses Python only for fixtures and observation, with a Go
test runner invoking the shared C driver; code generation remains Fairy C:

```sh
go build -o .cache/c-file-cli/native-runner ./cmd/fairyc-candidate-c-native
python examples/lexer_candidate/verify_c.py --images /path/to/dev /path/to/release --reference .cache/mir-defer-file-cli/reference.json --native-runner .cache/c-file-cli/native-runner --receipt .cache/c-file-cli/c-receipt.json
```

Both candidate images emit byte-identical C for 106 Go-reference fixtures.
Native comparison observes the full i32 return separately from process status,
checks the private trap receipt and exact stdout, and adds a trace hook only to
test copies of the explicit defer trace function. The original generated main
also runs in a normal-return and a trap case. Windows low/optimized verification
covers 216 native executions, 228 C CLI executions and the existing 1256 CLI
executions. `tests/c_emit.fy` checks entry rejection, Buffer preservation,
determinism and the writer cap in both native profiles.
The Windows test runner can reuse an earlier successful build with the same
source/runtime hashes, compiler path/version and profile after verifying the
saved image hash. It always executes that image again and compares fresh results.
Linux builds are not reused by this runner.

Evidence: `docs/ops/evidence/c-file-cli-20261006.json`. Linux-target C emission is
checked; Linux native execution is deferred to the owner's consolidated batch.
Compiler/SDK inputs are unchanged. Compound values, ownership/cleanup lowering,
complete verifier coverage, package linking and a self-compiling compiler remain
subsequent work. No self-hosting promotion or performance claim is made.

## Same-file nominal struct type checking (historical: 0.16.0)

This section records the 0.16.0 scope and verification. Later HIR/MIR admission
is described in the versioned sections below.

Version 0.16.0 connects plain struct declarations to the existing name and type
checker. Named identity is the defining declaration, so equal field shapes do not
make two structs interchangeable. Scalar fields and acyclic nested plain structs
support record literals, reordered and shorthand initializers, copied bindings,
whole-value assignment, field reads/writes and internal function arguments/returns.
Every declared field must be supplied exactly once. Field writes require a mutable
local root; parameter, let-bound and temporary roots stay immutable. Scalar field
initializers retain contextual integer range/minimum checks.

`src/nominal_types.fy` owns the declaration lookup and bounded acyclic field graph.
Private type IDs are 256 plus the zero-based declaration index; scalar IDs stay
unchanged. The binding dump adds `struct` symbols. The typed result retains
declaration-based symbol/expression type IDs and `type_check::field_target` links to
the parser field arena, preserving written initializer order. No new language
syntax, public SDK type or runtime ABI is introduced.

The existing 128-global, 512-declaration, 1024-field and depth-64 bounds apply.
Resources, borrowed/array fields, generic structs, methods, attributes and imported
nominal definitions require subsequent components. At version 0.16.0 the scalar
HIR/MIR/C modes rejected nominal input with empty stdout. Version 0.17.0 below
admitted nominal HIR while MIR/C still rejected it. The type-checking unit does
not execute input struct operations.

```sh
go run ./cmd/fairyc-lexer-file-oracle --nominal .cache/nominal-types-file-cli/reference.json
python examples/lexer_candidate/verify_nominal.py --images /path/to/dev /path/to/release --reference .cache/nominal-types-file-cli/reference.json --receipt .cache/nominal-types-file-cli/nominal-receipt.json
```

Windows development/release verification compares 23 semantic outcomes per
profile with the Go reference, checks three explicit candidate rejection controls
per profile and six scalar-backend rejection controls, totaling 58 CLI runs.
The reference records Go semantic facts separately from later MIR errors: unused
by-value cycles pass Go semantics but are rejected by this candidate's finite
layout prerequisite; a bare type-name value also passes Go semantics but fails
Go MIR lowering and is rejected here. These three controls are not counted as
semantic-parity cases. Two native API tests per profile check retained field links,
nested mutation, nominal identity, empty structs, shorthand and malformed inputs.
The existing 1256 CLI runs pass with updated independent struct declaration
expectations. Linux native execution stays deferred to the consolidated batch.

During comparison, reordered aggregate initializers exposed a Go MIR defect:
the lowerer retained written field order while the verifier required declaration
order, and mutable source reads could change during later initializers. The
separate compiler regression now checks seven struct/resource cases through
verified MIR, interpreter results and both Windows native profiles. It evaluates
initializers in written order, captures copied source values, then normalizes
only construction operands; an initializer return stops remaining construction.

Evidence: `docs/ops/evidence/nominal-types-file-cli-20261006.json`.

## Owned nominal HIR (historical: 0.17.0)

This section records the 0.17.0 scope and verification, including its then-current
MIR/C rejection controls. Version 0.18.0 MIR admission is described below.

Version 0.17.0 retains same-file plain struct layouts and semantic trees after
the parsed, resolved and typed input owners are destroyed. The candidate owns
declaration-based type IDs, field types/name spans, constructor targets, written
initializer order, field accesses, nested assignment places and nominal function
signatures. It supports scalar and acyclic nested struct fields, empty structs,
forward declarations, shorthand, whole-value assignment and struct-valued
branches/calls/returns. At 0.17.0, input struct code did not execute: MIR lowering
rejected the nominal table before publishing a program; MIR/C errors published no
stdout.

`src/hir_shapes.fy` owns checked layout tables: at most 512 shapes, 1024 fields
and depth 64. Unknown field types, cycles, wrong owners/ranges and reused
identities are rejected before graph traversal. `src/hir_shape_lower.fy` copies
those facts from checked input. HIR assembly checks declaration/field identities,
field operand types, postorder edges, initializer uniqueness and complete
construction. A forged nominal variant cannot alias a scalar tag.

Scalar-only HIR retains byte-identical `fairyc.scalar-hir/v1`. Nominal HIR uses
the separate private `fairyc.nominal-hir/v1` transport: `N` rows retain type,
binding, field start/count and visibility; `D` rows retain field ID, owner/type and name span.
Named types render as `nominal:<declaration-type-id>`. Expression kinds add
`construct`, `field` and ordered `initializer`; their binding slots identify
the nominal declaration or field respectively. Scalar tags and record columns
are unchanged. This is an example-private format, not a stable CLI/SDK/ABI change.

```sh
go run ./cmd/fairyc-lexer-file-oracle --nominal-hir .cache/nominal-hir-file-cli/reference.json
python examples/lexer_candidate/verify_nominal_hir.py --images /path/to/dev /path/to/release --reference .cache/nominal-hir-file-cli/reference.json --receipt .cache/nominal-hir-file-cli/hir-receipt.json
```

Independent Go semantic HIR supplies 15 positive fixtures. Both native candidate
profiles compare complete function trees and field layouts (30 comparisons),
including initializer mutation/order and early return. The nominal checker
corpus retains 58 CLI runs: 46 semantic comparisons, six explicit candidate
rejections, two accepted HIR controls and four MIR/C rejections. Native API tests
check retained owners, deterministic rendering and malformed layout rejection.
The 1256 scalar CLI regression executions remain unchanged.
Sample: `samples/nominal_hir.fy`.

Evidence: `docs/ops/evidence/nominal-hir-file-cli-20261006.json`. At 0.17.0, struct
MIR, verification and C execution remained next; Linux native execution was
deferred to the consolidated batch. No self-hosting promotion is claimed.

## Owned nominal MIR

Example version 0.18.0 extends `--mir-file` to same-file plain, acyclic copy
structs with scalar or nested struct fields. `src/mir_lower.fy` consumes HIR;
`src/mir_model.fy` owns copied nominal layouts, source bytes and MIR arenas after
HIR destruction. Declaration-based identities remain distinct. Nominal function
signatures, ordered calls, struct-valued branches/returns, whole-value assignment
and nested field assignment lower into the existing control-flow graph.
Resources, borrowed/array fields, generic structs and imported nominal definitions
remain outside this subset. The existing 512-shape, 1024-field and depth-64 limits
apply alongside the scalar MIR budgets.

Three operation kinds carry struct values:

- **6, construct:** evaluate initializers in written order and freeze their
  values, then append operands in declaration order. A returning initializer
  stops construction. Call arguments are likewise frozen before later arguments
  can mutate their source.
- **7, field read:** copy a field from an evaluated, frozen owner value.
- **8, functional update:** produce a replacement owner value. `src/mir_nominal.fy`
  rereads the current assignment root and nested owners after RHS evaluation,
  preserving sibling changes made by that RHS. Compound assignment freezes the
  old target field before the RHS, computes the checked result, rebuilds the
  containing structs and copies the replacement into the mutable root.

`src/mir_verify_nominal.fy` checks constructor arity and declaration-order field
types, field ownership, operand/result types and unused payload slots. The
existing verifier checks edge ownership, calls/returns and definite initialization,
including every constructor operand. Lowering and rendering both require a valid
graph; invalid rendering preserves the caller's Buffer.

`src/mir_output.fy` emits private `fairyc.nominal-mir/v1`: `N` rows contain type,
binding, field start/count and visibility; `D` rows contain field ID, owner/type
and name span. Nominal `O` rows append `argument_start` and `argument_count` after
the literal payload column; constructor operands use the existing `A` edge arena.
Scalar-only output keeps `fairyc.scalar-mir/v1` and its columns unchanged.
Version 0.18.0 belongs to this example; it adds no public SDK, stable CLI or ABI
change.

`verify_nominal_mir.py` compares 20 Go oracle seeds across development/release
candidate images: **40 comparisons passed**. The candidate is native compiled
Fairy C. The script checks layouts and signatures, then independently executes
emitted MIR through the test-only Python observer against expected Go MIR
interpreter outcomes. Cases cover initializer order and
early return, copied call arguments, nested/sibling mutation, branches, loops,
arithmetic traps and defer traces. The separate Go call-argument snapshot fix is
commit `58350fb9`.

The 0.18.0 regression runs also passed: **58 nominal CLI runs**, **30 nominal HIR
comparisons** and **1256 scalar CLI runs**. Nominal CLI controls now accept HIR
and MIR and rejected C output. Native API final checks passed:
`tests/mir_nominal.fy` (2), `tests/hir_nominal.fy` (3) and `tests/mir_verify.fy` (1),
six tests per development/release profile, including ten nominal MIR mutations.
These historical comparisons established no empty-struct parity. Bootstrap
commit `8bd0e831` subsequently connected private empty construction, copying,
arguments, returns and nesting through MIR, the interpreter and C execution.

Version 0.18.0 still rejected nominal C output. Its evidence is recorded in
`docs/ops/evidence/nominal-mir-file-cli-20261006.json`.

## Owned nominal C

Version 0.19.0 extends `--c-file` to the same-file plain copy structs admitted by
nominal MIR. `src/c_nominal.fy` emits dependencies before their by-value owners,
constructs fields from the already frozen MIR operands, reads fields and copies
an owner before updating one field. Function signatures, arguments, returns and
locals retain the nominal type. Forward type references, nested layouts and distinct
types with identical fields keep their existing source identities.

C symbols use numeric arena IDs rather than source names. Private empty structs
contain one initialized dummy byte; this byte is not a source field and is not a
public ABI. Scalar-only units retain `fairyc.candidate-scalar-c/v1`; nominal units
use `fairyc.candidate-nominal-c/v1`. The existing 2 MiB output bound and 64-bit C11
target guards remain. Invalid MIR is rejected before appending output.

```text
fairyc build --release --locked
build/lexer-candidate.exe --c-file samples/nominal-c-demo.fy
```

The generated unit still needs the private trap runtime when compiled. The
portable `verify_c.py --nominal` harness invokes the existing native C runner;
scripts provide fixtures and observations, not product parsing or C generation.
It compares Go-interpreter outcomes for constructors, initializer ordering,
copied arguments, nested/sibling updates, control flow, traps and defer traces.
The initial run used a fixed native expectation for an empty-field RHS because
the Go bootstrap rejected that store. The field-store correction now admits it,
and `verify_c.py` takes that expectation from the Go MIR interpreter, alongside
empty-field early return and nested replacement. `@repr(C)`, resources, arrays/views, enums, generics
and multi-file nominal types remained outside version 0.19.0. Version 0.21.0
adds the scalar fixed arrays described below; views remain a separate step.

Windows MSVC development/release evidence, raw invocations, expected outcomes,
source and compiler identities and EXE hashes are recorded in
`docs/ops/evidence/nominal-c-file-cli-20261006.json`. Linux native execution is
deferred to the owner's consolidated batch; no performance measurement is claimed.

The initial focused run passed 48 nominal MIR comparisons, 74 nominal C CLI runs,
56 nominal native executions (24 Go-reference cases plus one fixed control and
three standalone images across both profiles), 58 nominal type/HIR/MIR/C CLI
controls, 20 native scalar regressions and three native API tests per profile.

The field-store follow-up passed 12 MIR comparisons, 36 C CLI runs and 18 Windows
native executions, restricted to three corrected cases and three prior controls.
All six expected outcomes came from the Go interpreter. Raw invocations, source
digests, reused candidate/compiler identities and fresh native EXE hashes are in
`docs/ops/evidence/empty-struct-field-store-20261006.json`; the original evidence
above retains its historical fixed-control classification.

## Owned scalar fixed arrays

Version 0.20.0 admits scalar fixed-array types and literals into name resolution,
type checking and owned HIR. Elements are `i32`, `i64`, `u8`, `u32`, `usize` or
`bool`, with literal lengths 1..64. Ordinary and repeated initializers preserve
written order; the repeated value is represented once. Exact element types and
lengths govern copies, function arguments and returns. Indexes use `usize` and
`.len` returns `usize`. Indexed assignment requires a mutable named array local.
Empty, oversized, nested or nominal-element arrays, array fields, slices and
borrowed arrays remain outside this component. Version 0.20.0 retained these
forms in HIR; version 0.21.0 connects them to verified MIR and native C execution.

`tests/hir_arrays.fy` checks retained types, element order, repeat representation,
index/length nodes and type-ID limits alongside existing nominal HIR controls.

Version 0.21.0 lowers construction, value copies, arguments, returns, length,
indexed reads and indexed assignment through `src/mir_arrays.fy` and
`src/mir_lower.fy`. Initializer operands are frozen in written order. A repeated
initializer evaluates its expression once, including any effects or early return.
An indexed store freezes and checks its index before evaluating its RHS; a
compound store also freezes the old element before the RHS. Replacing the whole
array during the RHS preserves the replacement's other elements.

`src/mir_verify_arrays.fy` checks exact element and length types. Indexed loads
and stores require a single-definition checked-index temporary for the same
array type, with definite initialization proving that the check dominates use.
`tests/mir_arrays.fy` rejects nine malformed MIR variants and confirms that a
rejected render preserves an existing output buffer.

`src/c_arrays.fy` emits each used array type as a private C struct with a fixed
`fy_items` field. Struct assignment preserves by-value array copies. Bounds trap
with category 6 before indexing; arithmetic uses the existing checked scalar
operations. Array units use `fairyc.candidate-array-c/v1`, array HIR/MIR dumps
use `fairyc.array-hir/v1` and `fairyc.array-mir/v1`. The scalar and nominal C
formats remain unchanged for programs without arrays. This transport retains
the existing 64-bit C11 guards and 2 MiB output bound; it adds no public ABI.

```text
fairyc build --release --locked
build/lexer-candidate.exe --c-file samples/array-c-demo.fy
```

The sample copies an array, updates the original and returns 3 from the copy.
Generated C still requires the private trap runtime when compiled. The portable
`verify_c.py --arrays` harness compares output from separate development/release
candidate images and runs emitted C against Go MIR-interpreter expectations.
It covers all six scalar elements, lengths 1 and 64, evaluation order, early
returns, loops, bounds and overflow traps. Scripts supply fixtures and observe
results; Fairy C owns parsing, lowering, verification and C generation.

Windows MSVC results and source/compiler/SDK identities are recorded in
`docs/ops/evidence/scalar-array-c-file-cli-20261006.json`. The run passed 68 array
CLI checks, 17 identical C units and 40 native executions, plus 54 nominal and
20 scalar native regressions. Three native API tests per candidate profile cover
HIR retention, temporary-array dump identity, nine MIR mutations and C output
controls. The Go bootstrap's array-call argument snapshot defect was corrected
in `5945ae28`; the reference now agrees with the candidate's by-value result.
Linux native validation
remains deferred to the owner's consolidated batch. No build-cost or performance
measurement is claimed. Array fields, nested or nominal-element arrays, borrowed
arrays, slices/views and array iteration remain separate candidate work.

## Owned static text and byte literals

Version 0.22.0 retains decoded `"..."` and `b"..."` bytes in owned HIR after the
parser and scanner are released. Their exact types are immutable `ref str` and
`ref u8[]`, with Static origins. Text remains UTF-8; byte literals may contain
any octet, including NUL and invalid UTF-8. Existing scanner escape validation
owns decoding. C strings, mutable views, local array borrows and literal-bearing
struct fields remain closed in this candidate slice.

Locals, arguments and literal-returning functions retain those types. Text and
byte views expose byte `.len` and `.is_empty`; only byte views admit indexed
reads with `usize` indexes. Text indexing, writes through byte views and implicit
text/byte conversion are rejected. HIR stores cooked bytes, not source spelling,
and `fairyc.literal-hir/v1` uses hex payloads so embedded NUL, tabs and newlines
cannot split records. Version 0.23.0 connects these retained values to MIR/C below.

`tests/hir_literals.fy` checks parser-independent ownership, distinct types,
Unicode escapes, empty text and embedded binary bytes. Windows native development
and release checks cover this retained HIR step; Linux runs remain deferred.

## Static literal MIR and native C execution

Version 0.23.0 executes immutable Static text and byte descriptors through
verified MIR and private C11 transport. `--mir-file` uses `fairyc.literal-mir/v1`
with hex literal payloads; `--c-file` emits static byte arrays and distinct
pointer/length structs for text and bytes. Empty values have length zero, and
embedded NUL does not terminate either type. Descriptors may be copied, passed,
returned and reassigned without copying their static storage.

Byte reads freeze the receiver before index effects, check `usize` bounds, then
load the byte. `.len` counts bytes and `.is_empty` tests length zero. The MIR
verifier requires the checked index and frozen descriptor to match, with one
definition each and initialization dominating the load. Invalid MIR is rejected
before C rendering changes the caller's buffer. Public grammar, SDK and runtime
contracts are unchanged.

`samples/literal-c-demo.fy` demonstrates Korean text, binary bytes and a
literal-returning function. For example, `build/lexer-candidate.exe --c-file
samples/literal-c-demo.fy` emits its C unit. `verify_c.py --literals` compares
development/release candidate output and Windows native execution with the Go
MIR interpreter. Exact observations are recorded in
`docs/ops/evidence/literal-c-file-cli-20261006.json`.

The run covers 16 programs, 72 C CLI checks and 38 native executions across
MSVC low/optimized profiles, plus array, nominal and scalar regressions.
Three native API tests per candidate profile cover HIR ownership, seven malformed
literal MIR mutations and C output preservation. The independent comparison
exposed the Go bootstrap's delayed descriptor selection; commit `4d5afa17`
fixes indexed reads and slice bounds, with five interpreter/native regressions.
The pinned compiler used to build the candidate remains unchanged.

C strings, view slicing, local-array borrows, mutable views, literal-bearing
struct fields, text indexing and writes through byte views remain outside this
candidate slice. Linux native validation remains deferred to the consolidated
batch. No performance or self-hosting completion is claimed.

## Shared array view HIR

Version 0.24.0 adds owned HIR for shared `u8[N]` array borrows in an expected
`ref u8[]` context and for half-open slicing of byte views. Bounds may be
omitted. Array places must be named, lengths remain 1..64, and text, mutable,
temporary-array and field-array borrows remain closed. HIR origin checking
rejects local escapes and mutation while a shared alias or call argument is
still used. A loop-held descriptor remains live across its backedge.
Non-Static local descriptors must be `let`; mixed-origin or recursive
view-return summaries are not yet admitted. Parameter-derived returns preserve
the single inferred parameter origin. Version 0.25.0 connects these values to
MIR/C execution below.
The HIR lane passed 17 positive/negative CLI cases and the retained-graph native
API test in development/release profiles.

## Shared array views, slices and native C execution

Version 0.25.0 executes shared byte-array borrows and half-open byte slices
through verified MIR and private C11 transport. `--mir-file` publishes
`fairyc.view-mir/v1`; `--c-file` emits pointer/length descriptors backed by the
original named `u8[N]` array. Slice bounds are evaluated once, left to right,
and checked before pointer arithmetic. Reversed and out-of-range slices trap
with category 6; empty and one-past slices are valid.

HIR tracks aliases, loop-held loans and descriptors held through later call
arguments, index expressions and slice bounds. Writes before the borrow or
after its last use are accepted. Half-open source spans keep adjacent completed
writes separate from the borrow that follows. MIR independently recomputes
Static, local-array and parameter origins; local-backed returns, unknown
consumed descriptors and mixed return origins are rejected before C output.
Invalid operation shapes and uninitialized bounds are rejected by the existing
shape and flow checks. Failed rendering preserves the caller's output buffer.

`samples/shared-view-c-demo.fy` demonstrates a local array, a parameter-derived
return and slicing. Run `build/lexer-candidate.exe --c-file
samples/shared-view-c-demo.fy` to emit its C unit. `verify_c.py --views` compares
both candidate profiles with the Go MIR interpreter and runs the emitted C in
MSVC low/optimized profiles. The source, tool hashes and exact observations
are recorded in `docs/ops/evidence/view-c-file-cli-20261006.json`.
The view lane covers 24 Go-oracle programs, 96 C CLI checks and 54 native
executions, including reversed/out-of-range slices and early returns from bounds.
Three native API tests per candidate profile check retained HIR, five malformed
MIR controls and output preservation. Literal, array, nominal and scalar lanes
provide the scoped regressions; the sample also runs in both C profiles.

Mutable views, field/nested-array borrows, text slicing, C strings,
view-bearing aggregate fields and mixed/recursive view-return origins remain
outside this candidate slice. Public grammar, SDK and runtime are unchanged.
Linux execution remains deferred to the requested consolidated batch;
performance and self-hosting have not been measured here.

## Mutable byte-view HIR

Version 0.26.0 retains `ref mut u8[]` separately from immutable byte descriptors.
An explicit expected mutable-view type permits `ref mut` of a named mutable
`u8[N]` local. Named immutable descriptors support byte reads, plain/compound
indexed writes, metadata and mutable child slices. Descriptors themselves cannot
be reassigned; temporary-array and field-array borrows remain closed.

Exclusive generation checking rejects owner reads/writes while a mutable loan
is live, shared/exclusive conflicts in either order, parent/parent-alias access
while a child remains used, and retained siblings even for disjoint intervals.
Exact mutable call arguments create bounded reborrows held through later
arguments. A mutable return preserves one inferred parameter origin and keeps
the caller's parent frozen until its last use. Child uses extend ancestor holds;
parent and owner access resumes after the appropriate last use. Loop-held loans
and indexed-store receivers stay live across backedges and RHS evaluation.

`tests/hir_mutable.fy` covers retained mutable nodes, parent/sibling/call conflicts
and local-return escape rejection. The initial HIR lane includes 30 positive and
negative CLI cases. Mutable-to-shared call conversion, mutable text, field/nested
arrays, mixed/recursive return origins and deferred view arguments remain closed.
Version 0.27.0 connects these retained nodes to execution below.

## Mutable byte views and native C execution

Version 0.27.0 lowers exclusive byte descriptors, mutable slices and indexed
writes to verified MIR and private C11 transport. `struct fy_bytes_mut` has a
mutable byte pointer and an element count; shared descriptors keep const
pointers. A write changes the original array storage. Child writes therefore
remain visible when the parent and original owner resume after the last use.
Exact mutable parameters are passed as descriptors; a returned child retains
the inferred caller origin and suspension checked by HIR.

For an indexed store, MIR freezes the descriptor, evaluates/checks the index,
then evaluates the RHS and performs the write. Compound assignments load the
previous value before RHS effects, apply checked byte arithmetic and store only
on success. Slice bounds evaluate once, left to right, and trap before pointer
arithmetic unless `start <= end <= len`; an empty end slice is valid.

MIR separately validates mutable backing arrays, store value/descriptor types,
matching frozen descriptors and checked indices, definite initialization and
local/parameter return origins. HIR owns exclusive-generation conflicts;
MIR origin verification does not claim a complete source-independent mutable
loan analysis. Invalid MIR is rejected before C rendering changes the caller's
buffer. Public grammar, SDK and runtime contracts are unchanged.

`samples/mutable-view-c-demo.fy` updates a returned payload slice, resumes the
parent and verifies the original array. Emit it with `build/lexer-candidate.exe
--c-file samples/mutable-view-c-demo.fy`. `verify_c.py --mutable` compares both
candidate profiles with independent Go MIR interpreter outcomes and executes the
C in MSVC low/optimized profiles. The mutable lane covers 20 Go-oracle programs,
102 C CLI checks and 46 native executions; the sample adds two native runs.
Exact source/tool hashes, commands and native
observations are in `docs/ops/evidence/mutable-c-file-cli-20261006.json`.

Mutable-to-shared call conversion, mutable text, field/nested-array borrows,
view-bearing aggregates, retained mutable siblings and mixed/recursive return
origins remain closed. Linux native validation stays deferred to the requested
batch; no performance or self-hosting completion is claimed.

## Cost baseline

`scanner::storage` reports source, separately owned cooked and diagnostic-token
bytes plus record counts. It does not report allocator capacity or peak heap/RSS.
The opt-in `fairyc_go_measure_lexer_stream` lane records five native driver
roundtrips after one warmup for each scenario/profile; ordinary tests skip it.
See [the measured baseline](cost-baseline-2026-09-10.md) for raw samples, exact
build identities, included overhead and the limits on interpretation.

The separate [SDK heap observation](heap-baseline-2026-09-10.md) counts actual
allocator requests, peak simultaneously live requested bytes, retained capacity
and balanced destruction, including three injected failures. Its instrumented
builds are not latency benchmarks or RSS measurements.

The [sparse cooked-storage comparison](sparse-storage-2026-09-10.md) preserves the
old baseline and records a 41.06% reduction in peak requested heap bytes for the
same token-heavy input, with a small allocation-count increase. The private token
layout changed; the Stream consumer API and logical payload metrics did not.

## Array fields and rank-two array HIR

Version 0.28.0 retains rank-one and rank-two scalar arrays in local values,
function signatures and acyclic struct fields. Each dimension stays 1..64.
Postfix type constructors apply left to right: `u8[2][3]` is three rows of two
bytes. Nested literals and repeated rows retain ordered, context-typed children;
repeated values are evaluated once and copied. Array IDs preserve existing
rank-one IDs and use a disjoint canonical encoding for rank two.

Field/index assignment paths must end at a mutable named local; let bindings,
parameters and temporaries stay immutable. Whole-row and whole-field replacement
require exact type equality, and compound operators require scalar integers.
`tests/hir_aggregate_arrays.fy` checks retained layouts and child types.
Rank three, arrays of structs/views and field/index byte-view borrows remain
closed. Version 0.29.0 connects these paths to native C execution below.

## Array fields and rank-two native execution

Version 0.29.0 lowers field/index paths through `mir_places.fy`. Index operands
are evaluated once from outer to inner and each dimension is checked before
later index expressions, reads or RHS effects. Named value paths read their
current root after index effects; temporary receivers retain frozen values.
Compound writes load the previous element before RHS evaluation and perform
checked scalar arithmetic. Stores then rebuild the selected path from the
current root, preserving other rows and sibling fields changed by the RHS.
Whole-row/array-field copies and function arguments remain independent values.

C array wrappers are emitted with inner types before outer types and before
structs, including types referenced only by unused struct declarations.
The writer keeps its 2 MiB output limit and admits at most 1024 distinct
rank-two wrapper types in a unit. MIR validation reuses exact aggregate operand
types, checked-index provenance and definite initialization for each layer.
`tests/mir_aggregate_arrays.fy` mutates row operands, bound provenance, update
values and field-read shapes; failed rendering preserves the existing buffer.

Emit `samples/aggregate-array-c-demo.fy` using `--c-file`. The
`verify_c.py --aggregate-arrays` lane uses explicit fixed expectations and
source assertions from `--aggregate-array-cases` in the test utility: 21
programs, 106 CLI checks and 48 native runs; the sample adds two native runs.
These are
not Go MIR interpreter outcomes: the Go reference does not admit rank-two source
types. The existing rank-one/nominal/mutable lanes retain their Go interpreter
oracles. Exact command, source, SDK/tool and native image hashes are recorded in
`docs/ops/evidence/aggregate-array-c-file-cli-20261006.json`.

Ranks above two, arrays of structs/views, and field/index byte-view borrows
remain closed. Public grammar, SDK/runtime and public ABI are unchanged. Linux
native execution remains deferred; this step makes no performance or completed
self-hosting claim.

## Text access diagnostic

Version 0.29.1 reports `text indexing and slicing are not supported` for a
`ref str` receiver, instead of the unrelated `invalid nominal type` message.
This follows the existing surface-syntax contract: text `.len` and `.is_empty`
remain byte metadata; only byte views support bracket indexing and slicing.
The private type diagnostic is 21 and spans the complete access expression.
Type, HIR, MIR and C CLI modes report the same error before publishing output,
including literals, local aliases, parameters and text-returning calls.

`tests/types.fy::text_access_has_a_distinct_diagnostic` exercises the type checker
directly. Exact CLI diagnostics, unchanged input bytes, both candidate profiles
and representative native byte/text-metadata results are recorded in
`docs/ops/evidence/text-access-diagnostic-20261006.json`. This is a diagnostic
correction, with no new text slice syntax, SDK API or runtime behavior.


## Payloadless enum HIR

Version 0.30.0 retains declaration-specific payloadless enums and exhaustive
shallow switches. Enums have 1..128 distinct members and private type IDs in
131072..131583, separate from integers, structs and fixed arrays. Contextual
`.Member` construction works in typed bindings, assignment, parameters, return
values and conditional/switch joins. Each switch lists every member exactly
once; written arm order and arm-local scopes survive HIR lowering. Positional
`.Member` arguments are distinguished from unsupported labeled call arguments.

`tests/hir_enums.fy` checks owned layouts and retained arm order after the input
arenas are released. `verify_enums.py --phase hir` checks accepted values and
rejected empty enums and unsupported payload field types, duplicate members/cases, missing/unknown cases,
integer coercion, distinct declarations, equality, aggregate enum storage,
immutable assignments and borrowed-view joins. Types, HIR, MIR and C execution
are separate candidate milestones; version 0.30.0 retains HIR only. That milestone excluded payloads;
version 0.32.0 adds scalar payload HIR below. Enum comparison and enum
arrays/struct fields remain closed. The public grammar,
SDK/runtime and public ABI are unchanged.


## Payloadless enum native execution

Version 0.31.0 connects enum values and switches to verified MIR and C11.
Operation 21 constructs a declaration-relative tag and operation 22 tests one
member; exact enum types remain distinct in copies, parameters and return values.
The private C representation is a separate struct per enum containing a
`uint32_t` tag. Source enum comparison stays unsupported. Switch subjects are
frozen once, only the selected arm executes, and live arm results join through
exactly typed locals. Returning arms have no fallthrough edge. Existing lexical
defer/loop cleanup applies to each arm; defer still captures scalar arguments,
and direct enum defer arguments are rejected explicitly.

`tests/mir_enums.fy` checks retained tags/branches and rejects forged tags,
wrong declarations/operands, non-bool tests and uninitialized values. Failed MIR
or C rendering preserves existing output bytes. `verify_enums.py --phase full`
checks all four CLI modes in both candidate profiles and runs emitted C in both
native profiles. Fixed expectations are independently compared with Go MIR
interpreter results produced by the test-only `--enum-mir` oracle. This work also
fixed Go MIR call lowering: earlier by-value enum arguments are now copied
before later arguments can overwrite their source local (including inert payload
enums). No parsing or lowering in the Fairy CLI is delegated to that oracle.

Emit `samples/enum-c-demo.fy` with `--c-file`. The evidence file is
`docs/ops/evidence/enum-native-20261006.json`. Payload enums, enum equality,
enum arrays/struct fields and borrowed-view switch joins remain outside this
candidate slice. Public grammar, SDK/runtime and public ABI are unchanged.
Linux native validation and performance measurement remain deferred. These
checks do not constitute completed compiler self-hosting.


## Integer and bool payload enum HIR

Version 0.32.0 extends the Fairy-written candidate's type checking and owned HIR
with named payloads of `i32`, `i64`, `u8`, `u32`, `usize`, and `bool`. Each enum
has 1..128 members; a payload member has 1..16 fields. The shared parser field
arena remains capped at 1024 entries, including member records and payload fields.
Empty payload braces, nested aggregates, resource/view payloads, positional
patterns, enum equality, and enum storage inside arrays/structs remain closed.

Construction uses contextual `.Value { .amount = n, .valid = true }`; every field
must appear exactly once. Written initializer order is retained, and literal
bounds use the declared field type. An exhaustive switch binds every field with
`case .Value { .amount = amount, .valid = valid }: { ... }`. Bindings are immutable,
scoped to that arm block, and can shadow outer locals. `_` discards one selected
field; it creates no local binding. Payloadless members retain their existing
construction and pattern syntax.

Owned shape records preserve payload ranges and scalar types after syntax/type
arenas are released. Variant edges contain ordered Initializer expressions; Arm
edges contain ordered Capture expressions. A Capture identifies its payload field
and has zero (discard) or one immutable Binding ID in its edge range. The private
payload dump is `fairyc.enum-payload-hir/v1`; its D rows add payload start/count.
Payloadless dumps keep their previous format and output bytes.

Version 0.32.0 initially rejected payload layouts at the MIR boundary. Version
0.33.0 connects these retained layouts to MIR and native execution as described
below. The original HIR-only evidence remains a historical record.
`tests/hir_enum_payloads.fy` checks retained data and malformed ranges;
HIR evidence: `docs/ops/evidence/enum-payload-hir-20261007.json`.
The public language grammar and SDK contracts are unchanged.

## Integer and bool payload enum execution

Version 0.33.0 lowers payload construction to operation 21 with declaration-order
operands, tag tests to 22, and scalar captures to 23. Initializers run in written
order and freeze each value before the next initializer. A switch evaluates its
subject once and copies only the selected arm's named captures. Early return
during construction leaves no unused parent edges. `_` never reads union storage.

The owned MIR dump is `fairyc.enum-payload-mir/v1`; D rows retain payload ranges.
Member tags are dense declaration ordinals excluding payload-field records.
The verifier checks owner, field membership/type, argument count/type, unique
edge ownership and definite initialization before either dump or C publication.
`tests/mir_enum_payloads.fy` includes malformed-operation and read mutations.

The C transport uses one struct per enum declaration, a `uint32_t fy_tag`, and
named payload structs inside a union. Construction, assignment, arguments and
return use value copies. Each capture tests the active tag before accessing the
union and calls `fy_runtime_trap(7)` on mismatch. This dynamic guard covers a
structurally valid MIR extraction whose active member differs at runtime.

Use `--mir-file` or `--c-file` on `samples/enum-payload-demo.fy`. As with the
existing C entry contract, native main returns i32; i64 payload computations
remain available inside functions. The sample asserts its i64 result then
returns 0. Run `verify_enum_payloads.py --images <dev-image> <release-image>
--phase full --native-runner <native-runner> --receipt <path>` for deterministic
HIR/MIR/C, malformed inputs, fixed expected native results, and a test-only
corrupted C active-tag case. The latter must trap 7 in both native profiles and
is explicitly not a valid Fairy source program. Raw CLI/native observations and
digests are in `docs/ops/evidence/enum-payload-native-20261007.json`.

Enum comparisons, nested/resource/view payloads, enum array/struct storage,
borrowed-view switch joins, complete self-hosting, Linux validation and
performance measurement remain outside this slice. Stable compiler, public
language/SDK APIs and runner implementation are unchanged.

## Scalar Option type, HIR and MIR

Builtin `Option<i32|i64|u8|u32|usize|bool>` uses contextual `.some(value)` and
`.none`, with exactly one `.some(binding)` and `.none` switch arm. `_` discards
the capture; named captures are immutable and confined to their arm. The six
canonical private IDs 196609..196614 are disjoint from user enum declarations.
Each used instance owns a synthetic, declaration-free shape (binding 0), with
one payload field and two members. Repeated spellings share the same shape.

HIR retains `Option` as a distinct type category; its readable tag is `option:N`.
The owned payload HIR/MIR wire formats remain `fairyc.enum-payload-hir/v1` and
`fairyc.enum-payload-mir/v1`. MIR operations 21/22/23 preserve construction,
tag tests and typed captures. Synthetic layouts are bounded by the same field
and shape arenas and validated before publication. See `samples/option-demo.fy`,
`tests/hir_options.fy` and `verify_options.py --phase mir`.

Evidence: `docs/ops/evidence/candidate-option-mir-20261007.json`. Native Option
acceptance, structured/resource payloads and Result
remain subsequent units in the owner's numbered plan. Source grammar and SDK
contracts are unchanged.

`tests/mir_options.fy` independently forges 5 invalid builtin layouts and 15
invalid operations/reads. It checks different Option instances, wrong member
tags and field ownership, missing or mistyped operands, uninitialized reads,
and preservation of the output buffer when verification fails. Existing enum
mutation tests run alongside it. Evidence:
`docs/ops/evidence/candidate-option-verifier-20261007.json`.
